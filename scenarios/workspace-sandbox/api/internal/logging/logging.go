// Package logging provides structured logging for the workspace-sandbox service.
//
// # Signal Surface Design
//
// This package implements structured logging to surface runtime signals that help
// operators and developers understand system behavior:
//
//   - Event-based: Each log entry is tied to a specific event type
//   - Contextual: Includes relevant metadata (sandbox ID, user, operation)
//   - Queryable: Outputs JSON for easy parsing by log aggregators
//   - Leveled: Supports debug, info, warn, error for filtering
//
// # Event Types
//
// Events are categorized by subsystem:
//   - sandbox.*: Sandbox lifecycle events (create, mount, stop, approve, reject, delete)
//   - driver.*: Driver operations (mount, unmount, cleanup, size_check)
//   - api.*: API request/response events
//   - policy.*: Policy validation events
package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents a log severity level.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Entry represents a structured log entry.
type Entry struct {
	Time      string                 `json:"time"`
	Level     Level                  `json:"level"`
	Event     string                 `json:"event"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	SandboxID string                 `json:"sandboxId,omitempty"`
	Actor     string                 `json:"actor,omitempty"`
	Duration  float64                `json:"durationMs,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger provides structured logging with consistent formatting.
type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	service string
	level   Level
}

// Option configures a Logger.
type Option func(*Logger)

// WithOutput sets the output writer.
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.out = w
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// New creates a new structured logger.
func New(service string, opts ...Option) *Logger {
	l := &Logger{
		out:     os.Stdout,
		service: service,
		level:   LevelInfo,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// log writes a structured log entry.
func (l *Logger) log(level Level, event, message string, fields map[string]interface{}) {
	// Skip if below configured level
	if !l.shouldLog(level) {
		return
	}

	entry := Entry{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		Event:   event,
		Message: message,
		Service: l.service,
		Fields:  fields,
	}

	// Extract special fields
	if sandboxID, ok := fields["sandboxId"].(string); ok {
		entry.SandboxID = sandboxID
		delete(fields, "sandboxId")
	}
	if actor, ok := fields["actor"].(string); ok {
		entry.Actor = actor
		delete(fields, "actor")
	}
	if duration, ok := fields["durationMs"].(float64); ok {
		entry.Duration = duration
		delete(fields, "durationMs")
	}
	if errMsg, ok := fields["error"].(string); ok {
		entry.Error = errMsg
		delete(fields, "error")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	l.out.Write(data)
	l.out.Write([]byte("\n"))
}

// shouldLog returns true if the given level should be logged.
func (l *Logger) shouldLog(level Level) bool {
	levels := map[Level]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

// Debug logs a debug-level event.
func (l *Logger) Debug(event, message string, fields map[string]interface{}) {
	l.log(LevelDebug, event, message, fields)
}

// Info logs an info-level event.
func (l *Logger) Info(event, message string, fields map[string]interface{}) {
	l.log(LevelInfo, event, message, fields)
}

// Warn logs a warning-level event.
func (l *Logger) Warn(event, message string, fields map[string]interface{}) {
	l.log(LevelWarn, event, message, fields)
}

// Error logs an error-level event.
func (l *Logger) Error(event, message string, fields map[string]interface{}) {
	l.log(LevelError, event, message, fields)
}

// --- Convenience Methods for Common Events ---

// SandboxCreated logs a sandbox creation event.
func (l *Logger) SandboxCreated(sandboxID, scopePath, owner string, durationMs float64) {
	l.Info("sandbox.created", "Sandbox created successfully", map[string]interface{}{
		"sandboxId":  sandboxID,
		"scopePath":  scopePath,
		"actor":      owner,
		"durationMs": durationMs,
	})
}

// SandboxMounted logs a sandbox mount event.
func (l *Logger) SandboxMounted(sandboxID, mergedDir string, durationMs float64) {
	l.Info("sandbox.mounted", "Overlay mounted", map[string]interface{}{
		"sandboxId":  sandboxID,
		"mergedDir":  mergedDir,
		"durationMs": durationMs,
	})
}

// SandboxStopped logs a sandbox stop event.
func (l *Logger) SandboxStopped(sandboxID string, durationMs float64) {
	l.Info("sandbox.stopped", "Sandbox stopped", map[string]interface{}{
		"sandboxId":  sandboxID,
		"durationMs": durationMs,
	})
}

// SandboxApproved logs a sandbox approval event.
func (l *Logger) SandboxApproved(sandboxID, actor string, filesApplied int, commitHash string) {
	l.Info("sandbox.approved", "Changes approved and applied", map[string]interface{}{
		"sandboxId":    sandboxID,
		"actor":        actor,
		"filesApplied": filesApplied,
		"commitHash":   commitHash,
	})
}

// SandboxRejected logs a sandbox rejection event.
func (l *Logger) SandboxRejected(sandboxID, actor string) {
	l.Info("sandbox.rejected", "Changes rejected", map[string]interface{}{
		"sandboxId": sandboxID,
		"actor":     actor,
	})
}

// SandboxDeleted logs a sandbox deletion event.
func (l *Logger) SandboxDeleted(sandboxID string, durationMs float64) {
	l.Info("sandbox.deleted", "Sandbox deleted", map[string]interface{}{
		"sandboxId":  sandboxID,
		"durationMs": durationMs,
	})
}

// SandboxError logs a sandbox error event.
func (l *Logger) SandboxError(sandboxID, operation string, err error) {
	l.Error("sandbox.error", "Sandbox operation failed", map[string]interface{}{
		"sandboxId": sandboxID,
		"operation": operation,
		"error":     err.Error(),
	})
}

// DriverError logs a driver error event.
func (l *Logger) DriverError(operation string, err error, details map[string]interface{}) {
	fields := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	}
	for k, v := range details {
		fields[k] = v
	}
	l.Error("driver.error", "Driver operation failed", fields)
}

// APIRequest logs an API request event.
func (l *Logger) APIRequest(method, path string, statusCode int, durationMs float64) {
	l.Info("api.request", "API request completed", map[string]interface{}{
		"method":     method,
		"path":       path,
		"statusCode": statusCode,
		"durationMs": durationMs,
	})
}

// PolicyValidation logs a policy validation event.
func (l *Logger) PolicyValidation(policyType, sandboxID string, passed bool, reason string) {
	event := "policy.passed"
	msg := "Policy validation passed"
	if !passed {
		event = "policy.failed"
		msg = "Policy validation failed"
	}
	l.Info(event, msg, map[string]interface{}{
		"policyType": policyType,
		"sandboxId":  sandboxID,
		"passed":     passed,
		"reason":     reason,
	})
}

// --- Context-aware logging ---

type ctxKey struct{}

// WithLogger returns a context with the logger attached.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves the logger from context, or returns a default logger.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return New("workspace-sandbox")
}
