package telemetry

import "time"

// SummaryResult contains telemetry file summary information.
type SummaryResult struct {
	Exists         bool   `json:"exists"`
	FilePath       string `json:"file_path,omitempty"`
	FileSizeBytes  int64  `json:"file_size_bytes,omitempty"`
	EventCount     int    `json:"event_count,omitempty"`
	LastIngestedAt string `json:"last_ingested_at,omitempty"`
}

// InsightsResult contains analyzed telemetry insights.
type InsightsResult struct {
	Exists        bool              `json:"exists"`
	LastSession   *SessionInsight   `json:"last_session,omitempty"`
	LastSmokeTest *SmokeTestInsight `json:"last_smoke_test,omitempty"`
	LastError     *ErrorInsight     `json:"last_error,omitempty"`
}

// SessionInsight describes the last app session.
type SessionInsight struct {
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	ReadyAt     string `json:"ready_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// SmokeTestInsight describes the last smoke test.
type SmokeTestInsight struct {
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ErrorInsight describes the last error event.
type ErrorInsight struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
}

// TailResult contains the last N telemetry entries.
type TailResult struct {
	Exists     bool        `json:"exists"`
	Limit      int         `json:"limit"`
	TotalLines int         `json:"total_lines,omitempty"`
	Entries    []TailEntry `json:"entries,omitempty"`
}

// TailEntry represents a single telemetry log entry.
type TailEntry struct {
	Raw   string                 `json:"raw"`
	Event map[string]interface{} `json:"event,omitempty"`
	Error string                 `json:"error,omitempty"`
}

// IngestRequest is the request payload for telemetry ingestion.
type IngestRequest struct {
	ScenarioName   string                   `json:"scenario_name"`
	DeploymentMode string                   `json:"deployment_mode"`
	Source         string                   `json:"source"`
	Events         []map[string]interface{} `json:"events"`
}

// IngestResponse is the response for telemetry ingestion.
type IngestResponse struct {
	Status         string `json:"status"`
	EventsIngested int    `json:"events_ingested"`
	OutputPath     string `json:"output_path"`
}

// SummaryResponse is the HTTP response for telemetry summary.
type SummaryResponse struct {
	ScenarioName   string `json:"scenario_name"`
	Exists         bool   `json:"exists"`
	FilePath       string `json:"file_path,omitempty"`
	FileSizeBytes  int64  `json:"file_size_bytes,omitempty"`
	EventCount     int    `json:"event_count,omitempty"`
	LastIngestedAt string `json:"last_ingested_at,omitempty"`
}

// InsightsResponse is the HTTP response for telemetry insights.
type InsightsResponse struct {
	ScenarioName  string            `json:"scenario_name"`
	Exists        bool              `json:"exists"`
	LastSession   *SessionInsight   `json:"last_session,omitempty"`
	LastSmokeTest *SmokeTestInsight `json:"last_smoke_test,omitempty"`
	LastError     *ErrorInsight     `json:"last_error,omitempty"`
}

// TailResponse is the HTTP response for telemetry tail.
type TailResponse struct {
	ScenarioName string      `json:"scenario_name"`
	Exists       bool        `json:"exists"`
	Limit        int         `json:"limit"`
	TotalLines   int         `json:"total_lines,omitempty"`
	Entries      []TailEntry `json:"entries,omitempty"`
}

// parsedInsights holds intermediate parsing state.
type parsedInsights struct {
	lastSession   *SessionInsight
	lastSmokeTest *SmokeTestInsight
	lastError     *ErrorInsight
}

// errorEvents defines which events are considered errors.
var errorEvents = map[string]bool{
	"startup_error":           true,
	"bundled_runtime_failed":  true,
	"runtime_error":           true,
	"dependency_unreachable":  true,
	"smoke_test_failed":       true,
	"migration_failed":        true,
	"asset_missing":           true,
	"asset_checksum_mismatch": true,
	"asset_size_exceeded":     true,
	"secrets_missing":         true,
	"runtime_secrets_missing": true,
	"service_not_ready":       true,
}

// IsErrorEvent checks if an event type is considered an error.
func IsErrorEvent(event string) bool {
	return errorEvents[event]
}

// ParseEventTimestamp attempts to parse a timestamp from an event payload.
func ParseEventTimestamp(payload map[string]interface{}) (time.Time, bool) {
	for _, key := range []string{"timestamp", "ts", "ingested_at"} {
		if raw, ok := payload[key].(string); ok && raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

// FormatTimeIfSet formats a time if it's not zero.
func FormatTimeIfSet(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// StringFromPayload extracts a string from a payload map.
func StringFromPayload(payload map[string]interface{}, key string) string {
	if raw, ok := payload[key].(string); ok {
		return raw
	}
	return ""
}
