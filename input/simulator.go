package input

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/go-vgo/robotgo"
)

type KeySimulator interface {
	PressKey(key string) error
}

type keySimulator struct{}

func NewKeySimulator() KeySimulator {
	return &keySimulator{}
}

func (k *keySimulator) PressKey(key string) error {
	switch runtime.GOOS {
	case "linux":
		return k.pressKeyLinux(key)
	case "windows":
		return k.pressKeyWindows(key)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func (k *keySimulator) pressKeyLinux(key string) error {
	return k.pressKeyWithRobotGo(key)
}

func (k *keySimulator) pressKeyWindows(key string) error {
	return k.pressKeyWithRobotGo(key)
}

func (k *keySimulator) pressKeyWithRobotGo(key string) error {
	keyMap := map[string]string{
		"0": "kp_0", "1": "kp_1", "2": "kp_2", "3": "kp_3", "4": "kp_4",
		"5": "kp_5", "6": "kp_6", "7": "kp_7", "8": "kp_8", "9": "kp_9",
		"*": "kp_multiply", "+": "kp_add", "-": "kp_subtract", ".": "kp_decimal", "/": "kp_divide",
		"enter": "kp_enter", "backspace": "backspace", "escape": "escape",
	}

	mappedKey, exists := keyMap[strings.ToLower(key)]
	if !exists {
		mappedKey = key
	}

	robotgo.KeyTap(mappedKey)
	return nil
}