package logutil

import (
	"context"

	"github.com/sirupsen/logrus"
)

// LogContext represents different components that can generate logs
type LogContext string

const (
	ContextServer    LogContext = "server"
	ContextSession   LogContext = "session"
	ContextBrowser   LogContext = "browser"
	ContextRecording LogContext = "recording"
	ContextWebSocket LogContext = "websocket"
	ContextDriver    LogContext = "driver"
	ContextAPI       LogContext = "api"
)

// StructuredLogger provides context-aware logging with correlation ID support
type StructuredLogger struct {
	logger        *logrus.Logger
	correlationID string
	context       LogContext
	fields        logrus.Fields
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(logger *logrus.Logger) *StructuredLogger {
	return &StructuredLogger{
		logger: logger,
		fields: make(logrus.Fields),
	}
}

// WithCorrelationID returns a new logger with the correlation ID set
func (l *StructuredLogger) WithCorrelationID(correlationID string) *StructuredLogger {
	newLogger := l.clone()
	newLogger.correlationID = correlationID
	return newLogger
}

// WithContext returns a new logger with the log context set
func (l *StructuredLogger) WithContext(ctx LogContext) *StructuredLogger {
	newLogger := l.clone()
	newLogger.context = ctx
	return newLogger
}

// WithField returns a new logger with a field added
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := l.clone()
	newLogger.fields[key] = value
	return newLogger
}

// WithFields returns a new logger with fields added
func (l *StructuredLogger) WithFields(fields logrus.Fields) *StructuredLogger {
	newLogger := l.clone()
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithSessionID is a convenience method for adding session_id field
func (l *StructuredLogger) WithSessionID(sessionID string) *StructuredLogger {
	return l.WithField("session_id", sessionID)
}

// WithURL is a convenience method for adding url field
func (l *StructuredLogger) WithURL(url string) *StructuredLogger {
	return l.WithField("url", url)
}

// WithError is a convenience method for adding error field
func (l *StructuredLogger) WithError(err error) *StructuredLogger {
	return l.WithField("error", err.Error())
}

// entry creates a logrus entry with all configured fields
func (l *StructuredLogger) entry() *logrus.Entry {
	entry := l.logger.WithFields(l.fields)
	if l.correlationID != "" {
		entry = entry.WithField("correlation_id", l.correlationID)
	}
	if l.context != "" {
		entry = entry.WithField("log_context", string(l.context))
	}
	return entry
}

// clone creates a shallow copy of the logger
func (l *StructuredLogger) clone() *StructuredLogger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	return &StructuredLogger{
		logger:        l.logger,
		correlationID: l.correlationID,
		context:       l.context,
		fields:        newFields,
	}
}

// Logging methods

func (l *StructuredLogger) Debug(msg string) {
	l.entry().Debug(msg)
}

func (l *StructuredLogger) Info(msg string) {
	l.entry().Info(msg)
}

func (l *StructuredLogger) Warn(msg string) {
	l.entry().Warn(msg)
}

func (l *StructuredLogger) Error(msg string) {
	l.entry().Error(msg)
}

// Debugf logs a formatted debug message
func (l *StructuredLogger) Debugf(format string, args ...interface{}) {
	l.entry().Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *StructuredLogger) Infof(format string, args ...interface{}) {
	l.entry().Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *StructuredLogger) Warnf(format string, args ...interface{}) {
	l.entry().Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *StructuredLogger) Errorf(format string, args ...interface{}) {
	l.entry().Errorf(format, args...)
}

// Context key for storing structured logger in context
type loggerKey struct{}

// ContextWithLogger returns a new context with the structured logger attached
func ContextWithLogger(ctx context.Context, logger *StructuredLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext extracts the structured logger from the context
func LoggerFromContext(ctx context.Context) *StructuredLogger {
	if logger, ok := ctx.Value(loggerKey{}).(*StructuredLogger); ok {
		return logger
	}
	return nil
}

// CorrelationIDKey is the context key for storing correlation ID
type correlationIDKey struct{}

// ContextWithCorrelationID returns a new context with the correlation ID attached
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from the context
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}
