package captures

import "time"

// CaptureType identifies whether a capture is a screenshot or recording.
type CaptureType string

const (
	CaptureScreenshot CaptureType = "screenshot"
	CaptureRecording  CaptureType = "recording"
	CaptureJourney    CaptureType = "journey"
)

// Capture holds metadata for a single persisted capture file.
type Capture struct {
	ID            string      `json:"id"`
	ScenarioName  string      `json:"scenario_name"`
	Type          CaptureType `json:"type"`
	Filename      string      `json:"filename"`
	FileSizeBytes int64       `json:"file_size_bytes"`
	Width         int         `json:"width,omitempty"`
	Height        int         `json:"height,omitempty"`
	DurationMs    int64       `json:"duration_ms,omitempty"`
	Checksum      string      `json:"checksum"`
	SourceSession string      `json:"source_session"`
	CreatedAt     time.Time   `json:"created_at"`
}

// CapturesSummary provides aggregate statistics for a scenario's captures.
type CapturesSummary struct {
	Count      int   `json:"count"`
	TotalBytes int64 `json:"total_bytes"`
}
