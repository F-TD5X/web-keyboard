package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"keyboard/input"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	HandshakeTimeout: 30 * time.Second,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	EnableCompression: true,
}

type KeyMessage struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

type Connection struct {
	conn *websocket.Conn
	send chan []byte
}

type WebSocketServer struct {
	connections map[*Connection]bool
	register    chan *Connection
	unregister  chan *Connection
	broadcast   chan []byte
	mutex       sync.Mutex
	input       input.KeySimulator
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		connections: make(map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		broadcast:   make(chan []byte),
	}
}

func (s *WebSocketServer) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/ws", s.handleWebSocket)
}

func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	connection := &Connection{
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.register <- connection

	go s.writePump(connection)
	go s.readPump(connection)
}

func (s *WebSocketServer) readPump(connection *Connection) {
	defer func() {
		s.unregister <- connection
		connection.conn.Close()
	}()

	for {
		connection.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, message, err := connection.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		log.Printf("Received message: %s", string(message))

		var keyMsg KeyMessage
		if err := json.Unmarshal(message, &keyMsg); err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		if s.input != nil && keyMsg.Type == "key" {
			log.Printf("Key pressed: %s", keyMsg.Key)
			if err := s.input.PressKey(keyMsg.Key); err != nil {
				log.Printf("Key press error: %v", err)
				errorMsg, _ := json.Marshal(map[string]string{
					"error": "Failed to press key: " + keyMsg.Key,
				})
				connection.send <- errorMsg
			} else {
				log.Printf("Key successfully pressed: %s", keyMsg.Key)
			}
		}
	}
}

func (s *WebSocketServer) writePump(connection *Connection) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		connection.conn.Close()
	}()

	for {
		select {
		case message, ok := <-connection.send:
			if !ok {
				connection.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := connection.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			connection.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := connection.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *WebSocketServer) Run() {
	for {
		select {
		case connection := <-s.register:
			s.mutex.Lock()
			if len(s.connections) >= 1 {
				existingConn := s.getExistingConnection()
				if existingConn != nil {
					closeMsg, _ := json.Marshal(map[string]string{
						"status": "disconnected",
						"reason": "Another device connected",
					})
					select {
					case existingConn.send <- closeMsg:
						// Wait a bit for the message to be sent
						time.Sleep(100 * time.Millisecond)
					default:
						// Channel full, connection closing
					}
					existingConn.conn.Close()
				}
			}
			s.connections[connection] = true
			welcomeMsg, _ := json.Marshal(map[string]string{
				"status": "connected",
			})
			connection.send <- welcomeMsg
			s.mutex.Unlock()
			log.Printf("New WebSocket connection. Total connections: %d", len(s.connections))

		case connection := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.connections[connection]; ok {
				delete(s.connections, connection)
				close(connection.send)
			}
			s.mutex.Unlock()
			log.Printf("WebSocket disconnected. Total connections: %d", len(s.connections))

		case message := <-s.broadcast:
			s.mutex.Lock()
			for connection := range s.connections {
				select {
				case connection.send <- message:
				default:
					close(connection.send)
					delete(s.connections, connection)
				}
			}
			s.mutex.Unlock()
		}
	}
}

func (s *WebSocketServer) getExistingConnection() *Connection {
	for conn := range s.connections {
		return conn
	}
	return nil
}

func (s *WebSocketServer) SetInputSimulator(input input.KeySimulator) {
	s.input = input
}

func (s *WebSocketServer) Shutdown() {
	s.mutex.Lock()
	for connection := range s.connections {
		close(connection.send)
		connection.conn.Close()
		delete(s.connections, connection)
	}
	s.mutex.Unlock()
}
