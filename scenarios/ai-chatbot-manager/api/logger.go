package main

import (
	"log"
	"os"
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[CHATBOT-API] ", log.LstdFlags|log.Lshortfile),
	}
}