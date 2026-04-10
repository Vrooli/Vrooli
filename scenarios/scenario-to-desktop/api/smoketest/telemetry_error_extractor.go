package smoketest

import (
	"bufio"
	"encoding/json"
	"strings"
	"time"
)

// errorEventTypes defines telemetry events that indicate errors.
var errorEventTypes = map[string]bool{
	"smoke_test_failed":  true,
	"app_session_failed": true,
}

// DefaultTelemetryErrorExtractor implements TelemetryErrorExtractor.
type DefaultTelemetryErrorExtractor struct {
	fs FileSystem
}

// NewTelemetryErrorExtractor creates a new telemetry error extractor.
func NewTelemetryErrorExtractor(fs FileSystem) *DefaultTelemetryErrorExtractor {
	return &DefaultTelemetryErrorExtractor{fs: fs}
}

// ExtractErrors reads a telemetry file and extracts any error events.
// Returns errors in reverse chronological order (most recent first).
func (e *DefaultTelemetryErrorExtractor) ExtractErrors(telemetryPath string, limit int) ([]TelemetryError, error) {
	if limit <= 0 {
		limit = 10
	}

	file, err := e.fs.Open(telemetryPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allErrors []TelemetryError
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}

		telErr := e.parseErrorEvent(event)
		if telErr != nil {
			allErrors = append(allErrors, *telErr)
		}
	}

	if err := scanner.Err(); err != nil {
		return allErrors, err
	}

	// Reverse to get most recent first
	reversed := make([]TelemetryError, 0, min(len(allErrors), limit))
	for i := len(allErrors) - 1; i >= 0 && len(reversed) < limit; i-- {
		reversed = append(reversed, allErrors[i])
	}

	return reversed, nil
}

// ExtractLatestError returns the most recent error from a telemetry file.
// Returns nil if no errors are found.
func (e *DefaultTelemetryErrorExtractor) ExtractLatestError(telemetryPath string) (*TelemetryError, error) {
	errors, err := e.ExtractErrors(telemetryPath, 1)
	if err != nil {
		return nil, err
	}
	if len(errors) == 0 {
		return nil, nil
	}
	return &errors[0], nil
}

// ExtractLatestErrorForSession returns the most recent error matching the given session ID.
// Falls back to ExtractLatestError if sessionID is empty.
// Returns nil if no matching errors are found.
func (e *DefaultTelemetryErrorExtractor) ExtractLatestErrorForSession(telemetryPath, sessionID string) (*TelemetryError, error) {
	// Fall back to ExtractLatestError if no session ID provided
	if sessionID == "" {
		return e.ExtractLatestError(telemetryPath)
	}

	// Get all errors (reasonable limit to avoid memory issues)
	errors, err := e.ExtractErrors(telemetryPath, 100)
	if err != nil {
		return nil, err
	}

	// Find the first (most recent) error matching the session ID
	for _, telErr := range errors {
		if telErr.SessionID == sessionID {
			return &telErr, nil
		}
	}

	// No matching error found for this session
	return nil, nil
}

// parseErrorEvent extracts a TelemetryError from an event map if it's an error event.
func (e *DefaultTelemetryErrorExtractor) parseErrorEvent(event map[string]interface{}) *TelemetryError {
	eventType, ok := event["event"].(string)
	if !ok || !errorEventTypes[eventType] {
		return nil
	}

	telErr := &TelemetryError{
		Event: eventType,
	}

	// Extract timestamp
	if ts, ok := event["timestamp"].(string); ok {
		telErr.Timestamp = ts
	}

	// Extract session_id
	if sid, ok := event["session_id"].(string); ok {
		telErr.SessionID = sid
	}

	// Extract deployment mode
	if dm, ok := event["deploymentMode"].(string); ok {
		telErr.DeploymentMode = dm
	}

	// Extract error message from details.error
	if details, ok := event["details"].(map[string]interface{}); ok {
		if errMsg, ok := details["error"].(string); ok {
			telErr.Message = errMsg
		}
		// Also try "reason" field for app_session_failed events
		if telErr.Message == "" {
			if reason, ok := details["reason"].(string); ok {
				telErr.Message = reason
			}
		}
	}

	// Skip events without a meaningful message
	if telErr.Message == "" {
		return nil
	}

	return telErr
}

// FormatTelemetryError creates a human-readable error message from a TelemetryError.
func FormatTelemetryError(err *TelemetryError) string {
	if err == nil {
		return ""
	}

	msg := err.Message

	// Clean up common error prefixes
	msg = strings.TrimPrefix(msg, "Error: ")

	return msg
}

// IsErrorStale returns true if the error's timestamp is before the given smoke test start time.
// This helps detect when telemetry errors from previous sessions are being displayed.
func IsErrorStale(err *TelemetryError, smokeTestStartTime time.Time) bool {
	if err == nil || err.Timestamp == "" {
		return false
	}

	// Parse the error timestamp (ISO 8601 format)
	errTime, parseErr := time.Parse(time.RFC3339, err.Timestamp)
	if parseErr != nil {
		// Try alternative format without timezone
		errTime, parseErr = time.Parse("2006-01-02T15:04:05.000Z", err.Timestamp)
		if parseErr != nil {
			return false
		}
	}

	return errTime.Before(smokeTestStartTime)
}
