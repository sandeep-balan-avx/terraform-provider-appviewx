package logger

import (
	"log"
	"strings"
	"sync"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel = LevelInfo
	levelMutex   sync.RWMutex
	levelStrings = map[string]int{
		"DEBUG": LevelDebug,
		"INFO":  LevelInfo,
		"WARN":  LevelWarn,
		"ERROR": LevelError,
	}
	levelPrefixes = map[int]string{
		LevelDebug: "[DEBUG] ",
		LevelInfo:  "[INFO] ",
		LevelWarn:  "[WARN] ",
		LevelError: "[ERROR] ",
	}
)

// SetLevel sets the current logging level
func SetLevel(level string) {
	levelMutex.Lock()
	defer levelMutex.Unlock()

	level = strings.ToUpper(level)
	if lvl, ok := levelStrings[level]; ok {
		currentLevel = lvl
	}
}

// Debug logs a debug message if level is sufficient
func Debug(format string, args ...interface{}) {
	levelMutex.RLock()
	defer levelMutex.RUnlock()

	if currentLevel <= LevelDebug {
		log.Printf(levelPrefixes[LevelDebug]+format, args...)
	}
}

// Info logs an info message if level is sufficient
func Info(format string, args ...interface{}) {
	levelMutex.RLock()
	defer levelMutex.RUnlock()

	if currentLevel <= LevelInfo {
		log.Printf(levelPrefixes[LevelInfo]+format, args...)
	}
}

// Warn logs a warning message if level is sufficient
func Warn(format string, args ...interface{}) {
	levelMutex.RLock()
	defer levelMutex.RUnlock()

	if currentLevel <= LevelWarn {
		log.Printf(levelPrefixes[LevelWarn]+format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	levelMutex.RLock()
	defer levelMutex.RUnlock()

	// Always log errors
	log.Printf(levelPrefixes[LevelError]+format, args...)
}
