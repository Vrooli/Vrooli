// Package util provides shared utilities for the landing-manager API.
package util

import (
	"encoding/json"
	"log"
	"time"
)

// LogStructured writes a structured log entry at info level
func LogStructured(msg string, fields map[string]interface{}) {
	LogWithLevel("info", msg, fields)
}

// LogStructuredError writes a structured log entry at error level
func LogStructuredError(msg string, fields map[string]interface{}) {
	LogWithLevel("error", msg, fields)
}

// LogStructuredWarn writes a structured log entry at warn level
func LogStructuredWarn(msg string, fields map[string]interface{}) {
	LogWithLevel("warn", msg, fields)
}

// LogWithLevel writes a structured log entry at the specified level
func LogWithLevel(level, msg string, fields map[string]interface{}) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if len(fields) == 0 {
		log.Printf(`{"level":"%s","message":"%s","timestamp":"%s"}`, level, msg, timestamp)
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"%s","message":"%s","fields":%s,"timestamp":"%s"}`, level, msg, fieldsJSON, timestamp)
}

// LogFailure is a convenience wrapper for logging failure events with consistent structure.
// It ensures failures are observable with context about what failed and recovery options.
func LogFailure(operation string, err error, context map[string]interface{}) {
	fields := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	}
	for k, v := range context {
		fields[k] = v
	}
	LogStructuredError("operation_failed", fields)
}

// LogDegradation logs when a system degrades gracefully instead of failing completely.
// This helps operators understand when fallback behavior is active.
func LogDegradation(component, reason string, context map[string]interface{}) {
	fields := map[string]interface{}{
		"component": component,
		"reason":    reason,
	}
	for k, v := range context {
		fields[k] = v
	}
	LogStructuredWarn("graceful_degradation", fields)
}
