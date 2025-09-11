package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Component   string                 `json:"component,omitempty"`
	Function    string                 `json:"function,omitempty"`
	File        string                 `json:"file,omitempty"`
	Line        int                    `json:"line,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    string                 `json:"duration,omitempty"`
	HTTPStatus  int                    `json:"http_status,omitempty"`
	HTTPMethod  string                 `json:"http_method,omitempty"`
	HTTPPath    string                 `json:"http_path,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	serviceName    string
	serviceVersion string
	minLevel       LogLevel
	prettyPrint    bool
}

// Global logger instance
var appLogger *Logger

// InitLogger initializes the global application logger
func InitLogger(serviceName, version string) {
	minLevel := LogLevelInfo
	prettyPrint := false

	// Check environment variables for configuration
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			minLevel = LogLevelDebug
		case "INFO":
			minLevel = LogLevelInfo
		case "WARN", "WARNING":
			minLevel = LogLevelWarn
		case "ERROR":
			minLevel = LogLevelError
		}
	}

	if os.Getenv("PRETTY_LOGS") == "true" || os.Getenv("ENABLE_DEBUG_LOGGING") == "true" {
		prettyPrint = true
	}

	appLogger = &Logger{
		serviceName:    serviceName,
		serviceVersion: version,
		minLevel:       minLevel,
		prettyPrint:    prettyPrint,
	}

	// Set up standard Go logger to use our structured logger
	log.SetFlags(0)
	log.SetOutput(&logWriter{logger: appLogger})
}

// logWriter implements io.Writer to redirect standard log output
type logWriter struct {
	logger *Logger
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message != "" {
		w.logger.Info("stdlib").Message(message).Log()
	}
	return len(p), nil
}

// LogBuilder provides a fluent interface for building log entries
type LogBuilder struct {
	logger *Logger
	entry  *LogEntry
}

// Debug creates a debug level log builder
func Debug(component string) *LogBuilder {
	return appLogger.Debug(component)
}

// Info creates an info level log builder
func Info(component string) *LogBuilder {
	return appLogger.Info(component)
}

// Warn creates a warning level log builder
func Warn(component string) *LogBuilder {
	return appLogger.Warn(component)
}

// Error creates an error level log builder
func Error(component string) *LogBuilder {
	return appLogger.Error(component)
}

// Fatal creates a fatal level log builder
func Fatal(component string) *LogBuilder {
	return appLogger.Fatal(component)
}

// Debug creates a debug level log builder
func (l *Logger) Debug(component string) *LogBuilder {
	return l.createLogBuilder(LogLevelDebug, component)
}

// Info creates an info level log builder
func (l *Logger) Info(component string) *LogBuilder {
	return l.createLogBuilder(LogLevelInfo, component)
}

// Warn creates a warning level log builder
func (l *Logger) Warn(component string) *LogBuilder {
	return l.createLogBuilder(LogLevelWarn, component)
}

// Error creates an error level log builder
func (l *Logger) Error(component string) *LogBuilder {
	return l.createLogBuilder(LogLevelError, component)
}

// Fatal creates a fatal level log builder
func (l *Logger) Fatal(component string) *LogBuilder {
	return l.createLogBuilder(LogLevelFatal, component)
}

func (l *Logger) createLogBuilder(level LogLevel, component string) *LogBuilder {
	// Get caller information
	_, file, line, _ := runtime.Caller(3)
	function := getFunctionName(3)

	entry := &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Service:   l.serviceName,
		Version:   l.serviceVersion,
		Component: component,
		Function:  function,
		File:      getShortFilename(file),
		Line:      line,
		Fields:    make(map[string]interface{}),
	}

	return &LogBuilder{
		logger: l,
		entry:  entry,
	}
}

// Message sets the log message
func (b *LogBuilder) Message(message string) *LogBuilder {
	b.entry.Message = message
	return b
}

// Messagef sets the log message with formatting
func (b *LogBuilder) Messagef(format string, args ...interface{}) *LogBuilder {
	b.entry.Message = fmt.Sprintf(format, args...)
	return b
}

// WithError adds error information to the log
func (b *LogBuilder) WithError(err error) *LogBuilder {
	if err != nil {
		b.entry.Error = err.Error()
	}
	return b
}

// WithUser adds user context to the log
func (b *LogBuilder) WithUser(userID string) *LogBuilder {
	b.entry.UserID = userID
	return b
}

// WithRequest adds request context to the log
func (b *LogBuilder) WithRequest(requestID string) *LogBuilder {
	b.entry.RequestID = requestID
	return b
}

// WithHTTP adds HTTP context to the log
func (b *LogBuilder) WithHTTP(method, path string, status int) *LogBuilder {
	b.entry.HTTPMethod = method
	b.entry.HTTPPath = path
	b.entry.HTTPStatus = status
	return b
}

// WithDuration adds duration information to the log
func (b *LogBuilder) WithDuration(duration time.Duration) *LogBuilder {
	b.entry.Duration = duration.String()
	return b
}

// WithField adds a custom field to the log
func (b *LogBuilder) WithField(key string, value interface{}) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

// WithFields adds multiple custom fields to the log
func (b *LogBuilder) WithFields(fields map[string]interface{}) *LogBuilder {
	for k, v := range fields {
		b.entry.Fields[k] = v
	}
	return b
}

// Log outputs the log entry
func (b *LogBuilder) Log() {
	if !b.shouldLog() {
		return
	}

	var output string
	
	if b.logger.prettyPrint {
		output = b.formatPretty()
	} else {
		jsonBytes, err := json.Marshal(b.entry)
		if err != nil {
			// Fallback to simple format if JSON marshaling fails
			output = fmt.Sprintf("[%s] %s %s: %s", 
				b.entry.Timestamp.Format("2006-01-02T15:04:05.000Z"),
				b.entry.Level, 
				b.entry.Component, 
				b.entry.Message)
		} else {
			output = string(jsonBytes)
		}
	}

	fmt.Println(output)

	// Exit if fatal
	if b.entry.Level == LogLevelFatal {
		os.Exit(1)
	}
}

func (b *LogBuilder) shouldLog() bool {
	levelPriority := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}
	
	entryPriority, exists := levelPriority[b.entry.Level]
	if !exists {
		return true
	}
	
	minPriority, exists := levelPriority[b.logger.minLevel]
	if !exists {
		return true
	}
	
	return entryPriority >= minPriority
}

func (b *LogBuilder) formatPretty() string {
	var parts []string
	
	// Timestamp and level
	levelColor := getLevelColor(b.entry.Level)
	timestamp := b.entry.Timestamp.Format("15:04:05.000")
	parts = append(parts, fmt.Sprintf("\033[90m%s\033[0m %s%-5s\033[0m", timestamp, levelColor, b.entry.Level))
	
	// Component and function
	if b.entry.Component != "" {
		parts = append(parts, fmt.Sprintf("\033[36m[%s]\033[0m", b.entry.Component))
	}
	
	// Message
	if b.entry.Message != "" {
		parts = append(parts, b.entry.Message)
	}
	
	// Error
	if b.entry.Error != "" {
		parts = append(parts, fmt.Sprintf("\033[91merror=%s\033[0m", b.entry.Error))
	}
	
	// HTTP context
	if b.entry.HTTPMethod != "" {
		httpInfo := fmt.Sprintf("%s %s", b.entry.HTTPMethod, b.entry.HTTPPath)
		if b.entry.HTTPStatus != 0 {
			statusColor := getStatusColor(b.entry.HTTPStatus)
			httpInfo += fmt.Sprintf(" %s%d\033[0m", statusColor, b.entry.HTTPStatus)
		}
		parts = append(parts, fmt.Sprintf("\033[94m[%s]\033[0m", httpInfo))
	}
	
	// Duration
	if b.entry.Duration != "" {
		parts = append(parts, fmt.Sprintf("\033[93mduration=%s\033[0m", b.entry.Duration))
	}
	
	// User context
	if b.entry.UserID != "" {
		parts = append(parts, fmt.Sprintf("\033[95muser=%s\033[0m", b.entry.UserID))
	}
	
	// Custom fields
	if len(b.entry.Fields) > 0 {
		for k, v := range b.entry.Fields {
			parts = append(parts, fmt.Sprintf("\033[90m%s=%v\033[0m", k, v))
		}
	}
	
	// Location (file:line)
	if b.entry.File != "" && b.entry.Line != 0 {
		parts = append(parts, fmt.Sprintf("\033[90m(%s:%d)\033[0m", b.entry.File, b.entry.Line))
	}
	
	return strings.Join(parts, " ")
}

// Helper functions

func getLevelColor(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "\033[37m" // White
	case LogLevelInfo:
		return "\033[32m" // Green
	case LogLevelWarn:
		return "\033[33m" // Yellow
	case LogLevelError:
		return "\033[31m" // Red
	case LogLevelFatal:
		return "\033[35m" // Magenta
	default:
		return "\033[0m"  // Reset
	}
}

func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "\033[32m" // Green for success
	case status >= 300 && status < 400:
		return "\033[36m" // Cyan for redirect
	case status >= 400 && status < 500:
		return "\033[33m" // Yellow for client error
	case status >= 500:
		return "\033[31m" // Red for server error
	default:
		return "\033[0m"  // Reset
	}
}

func getFunctionName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}
	
	fullName := fn.Name()
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return fullName
}

func getShortFilename(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return fullPath
}

// Convenience functions for common logging patterns

// LogAPIRequest logs an HTTP API request with timing
func LogAPIRequest(method, path string, status int, duration time.Duration, userID string) {
	Info("api").
		Message("API request").
		WithHTTP(method, path, status).
		WithDuration(duration).
		WithUser(userID).
		Log()
}

// LogAPIError logs an API error with context
func LogAPIError(method, path string, err error, userID string) {
	Error("api").
		Message("API request failed").
		WithHTTP(method, path, 0).
		WithError(err).
		WithUser(userID).
		Log()
}

// LogDatabaseOperation logs a database operation
func LogDatabaseOperation(operation, table string, duration time.Duration, err error) {
	builder := Info("database").
		Message(fmt.Sprintf("Database %s on %s", operation, table)).
		WithDuration(duration).
		WithField("operation", operation).
		WithField("table", table)
	
	if err != nil {
		builder = Error("database").
			Message(fmt.Sprintf("Database %s failed on %s", operation, table)).
			WithDuration(duration).
			WithField("operation", operation).
			WithField("table", table).
			WithError(err)
	}
	
	builder.Log()
}

// LogSchedulerEvent logs scheduler/processor events
func LogSchedulerEvent(component, action string, count int, duration time.Duration) {
	Info("scheduler").
		Message(fmt.Sprintf("%s completed %s", component, action)).
		WithDuration(duration).
		WithField("component", component).
		WithField("action", action).
		WithField("count", count).
		Log()
}

// LogAuthEvent logs authentication events
func LogAuthEvent(event, userID string, success bool) {
	level := Info
	if !success {
		level = Warn
	}
	
	level("auth").
		Message(fmt.Sprintf("Authentication %s", event)).
		WithUser(userID).
		WithField("event", event).
		WithField("success", success).
		Log()
}