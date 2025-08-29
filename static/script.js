class NumpadKeyboard {
    constructor() {
        this.ws = null;
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 2000;
        this.wasReplaced = false;
        this.connecting = false;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.connectWebSocket();
        this.updateDeviceId();
    }

    setupEventListeners() {
        // Setup numpad button clicks
        const keys = document.querySelectorAll('.key');
        keys.forEach(key => {
            key.addEventListener('click', (e) => this.handleKeyPress(e));
            
            // Add touch support for mobile
            key.addEventListener('touchstart', (e) => {
                e.preventDefault();
                key.classList.add('pressed');
            });
            
            key.addEventListener('touchend', (e) => {
                e.preventDefault();
                key.classList.remove('pressed');
                this.handleKeyPress(e);
            });
            
            // Add visual feedback for mouse
            key.addEventListener('mousedown', (e) => {
                key.classList.add('pressed');
            });
            
            key.addEventListener('mouseup', (e) => {
                key.classList.remove('pressed');
            });
            
            key.addEventListener('mouseleave', (e) => {
                key.classList.remove('pressed');
            });
        });

        // Handle window focus/blur for connection management
        window.addEventListener('focus', () => {
            if (!this.connected && this.reconnectAttempts === 0) {
                this.connectWebSocket();
            }
        });
    }

    handleKeyPress(event) {
        const key = event.target.getAttribute('data-key');
        if (key && this.connected) {
            this.sendKeyMessage(key);
            this.addVisualFeedback(event.target);
        } else if (!this.connected) {
            this.showNotification('Not connected to server', 'error');
        }
    }

    addVisualFeedback(element) {
        element.style.transform = 'scale(0.95)';
        element.style.background = '#007bff';
        element.style.color = 'white';
        
        setTimeout(() => {
            element.style.transform = '';
            element.style.background = '';
            element.style.color = '';
        }, 150);
    }

    connectWebSocket() {
        if (this.connecting) {
            return;
        }
        
        this.connecting = true;
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                this.connected = true;
                this.reconnectAttempts = 0;
                this.connecting = false;
                this.updateConnectionStatus('Connected', 'connected');
                this.showNotification('Connected to server', 'success');
            };
            
            this.ws.onmessage = (event) => {
                this.handleWebSocketMessage(event);
            };
            
            this.ws.onclose = () => {
                this.connected = false;
                this.updateConnectionStatus('Disconnected');
                // Only attempt reconnect if not due to being replaced by another device
                if (this.wasReplaced) {
                    this.wasReplaced = false;
                    return;
                }
                this.attemptReconnect();
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.connecting = false;
                this.showNotification('Connection error', 'error');
            };
            
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            this.connecting = false;
            this.showNotification('Failed to establish connection', 'error');
        }
    }

    handleWebSocketMessage(event) {
        try {
            const message = JSON.parse(event.data);
            
            if (message.status === 'connected') {
                this.updateConnectionStatus('Connected', 'connected');
                this.showNotification('Successfully connected', 'success');
            } else if (message.status === 'disconnected') {
                this.updateConnectionStatus('Disconnected');
                if (message.reason === 'Another device connected') {
                    this.wasReplaced = true;
                }
                this.showNotification(message.reason || 'Connection closed', 'warning');
            } else if (message.error) {
                this.showNotification(message.error, 'error');
            }
            
        } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
        }
    }

    sendKeyMessage(key) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const message = {
                key: key,
                type: 'key',
                timestamp: Date.now()
            };
            
            this.ws.send(JSON.stringify(message));
            console.log(`Sent key: ${key}`);
        }
    }

    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            this.updateConnectionStatus(`Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            
            setTimeout(() => {
                this.connectWebSocket();
            }, this.reconnectDelay);
        } else {
            this.updateConnectionStatus('Connection failed');
            this.showNotification('Failed to reconnect. Please refresh the page.', 'error');
        }
    }

    updateConnectionStatus(status, className = '') {
        const statusElement = document.getElementById('connection-status');
        statusElement.textContent = status;
        statusElement.className = className;
    }

    updateDeviceId() {
        const deviceInfo = document.getElementById('device-info');
        const deviceId = this.getDeviceId();
        deviceInfo.textContent = `Device: ${deviceId}`;
    }

    getDeviceId() {
        const userAgent = navigator.userAgent;
        let deviceType = 'Unknown';
        
        if (userAgent.match(/Mobile|Android|iPhone|iPad|iPod/i)) {
            deviceType = 'Mobile';
        } else if (userAgent.match(/Tablet|iPad/i)) {
            deviceType = 'Tablet';
        } else {
            deviceType = 'Desktop';
        }
        
        const browser = this.getBrowserName();
        return `${deviceType} (${browser})`;
    }

    getBrowserName() {
        const userAgent = navigator.userAgent;
        
        if (userAgent.match(/Chrome/i)) {
            return 'Chrome';
        } else if (userAgent.match(/Firefox/i)) {
            return 'Firefox';
        } else if (userAgent.match(/Safari/i)) {
            return 'Safari';
        } else if (userAgent.match(/Edge/i)) {
            return 'Edge';
        } else {
            return 'Unknown';
        }
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        // Style the notification
        Object.assign(notification.style, {
            position: 'fixed',
            top: '20px',
            right: '20px',
            padding: '12px 20px',
            borderRadius: '8px',
            color: 'white',
            fontWeight: '500',
            zIndex: '1000',
            opacity: '0',
            transform: 'translateX(100%)',
            transition: 'all 0.3s ease'
        });
        
        // Set background color based on type
        switch (type) {
            case 'success':
                notification.style.background = '#28a745';
                break;
            case 'error':
                notification.style.background = '#dc3545';
                break;
            case 'warning':
                notification.style.background = '#ffc107';
                notification.style.color = '#212529';
                break;
            default:
                notification.style.background = '#007bff';
        }
        
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.style.opacity = '1';
            notification.style.transform = 'translateX(0)';
        }, 100);
        
        // Animate out and remove
        setTimeout(() => {
            notification.style.opacity = '0';
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 3000);
    }
}

// Initialize the numpad keyboard when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new NumpadKeyboard();
});

// Handle page visibility changes
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        console.log('Page hidden - maintaining connection');
    } else {
        console.log('Page visible - checking connection');
    }
});