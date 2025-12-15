package typeconv

import (
	"github.com/vrooli/browser-automation-studio/automation/recorder"
)

// RetryHistoryEntry captures the outcome of a single retry attempt for a step.
type RetryHistoryEntry struct {
	Attempt        int    `json:"attempt"`
	Success        bool   `json:"success"`
	DurationMs     int    `json:"duration_ms,omitempty"`
	CallDurationMs int    `json:"call_duration_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

// TimelineScreenshot describes screenshot metadata associated with a frame.
type TimelineScreenshot struct {
	ArtifactID   string `json:"artifact_id"`
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	SizeBytes    *int64 `json:"size_bytes,omitempty"`
}

// TimelineArtifact exposes related artifacts such as console or network logs.
type TimelineArtifact struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Label        string         `json:"label,omitempty"`
	StorageURL   string         `json:"storage_url,omitempty"`
	ThumbnailURL string         `json:"thumbnail_url,omitempty"`
	ContentType  string         `json:"content_type,omitempty"`
	SizeBytes    *int64         `json:"size_bytes,omitempty"`
	StepIndex    *int           `json:"step_index,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

// ToRetryHistory converts various slice types to []RetryHistoryEntry.
func ToRetryHistory(value any) []RetryHistoryEntry {
	entries := make([]RetryHistoryEntry, 0)
	switch v := value.(type) {
	case []RetryHistoryEntry:
		return v
	case []map[string]any:
		for _, item := range v {
			if entry := ToRetryHistoryEntry(item); entry != nil {
				entries = append(entries, *entry)
			}
		}
	case []any:
		for _, item := range v {
			if entry := ToRetryHistoryEntry(item); entry != nil {
				entries = append(entries, *entry)
			}
		}
	}
	return entries
}

// ToRetryHistoryEntry converts a map to *RetryHistoryEntry.
// Returns nil if conversion fails or entry is empty.
func ToRetryHistoryEntry(value any) *RetryHistoryEntry {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	entry := RetryHistoryEntry{
		Attempt:        ToInt(m["attempt"]),
		Success:        ToBool(m["success"]),
		DurationMs:     ToInt(m["durationMs"]),
		CallDurationMs: ToInt(m["callDurationMs"]),
		Error:          ToString(m["error"]),
	}
	if entry.DurationMs == 0 {
		entry.DurationMs = ToInt(m["duration_ms"])
	}
	if entry.CallDurationMs == 0 {
		entry.CallDurationMs = ToInt(m["call_duration_ms"])
	}
	if entry.Error == "" {
		entry.Error = ToString(m["error_message"])
	}
	if entry.Attempt == 0 && entry.DurationMs == 0 && entry.CallDurationMs == 0 && entry.Error == "" && entry.Success {
		// treat empty entry as invalid unless it represents a successful attempt with no data
		entry.Success = true
	}
	if entry.Attempt == 0 && entry.Error == "" && entry.DurationMs == 0 && entry.CallDurationMs == 0 {
		return nil
	}
	return &entry
}

// ToTimelineScreenshot converts an artifact to *TimelineScreenshot.
// Returns nil if artifact is nil.
func ToTimelineScreenshot(artifact *recorder.ArtifactData) *TimelineScreenshot {
	if artifact == nil {
		return nil
	}
	shot := &TimelineScreenshot{
		ArtifactID:   artifact.ArtifactID,
		URL:          artifact.StorageURL,
		ThumbnailURL: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
		SizeBytes:    artifact.SizeBytes,
	}
	if artifact.Payload != nil {
		shot.Width = ToInt(artifact.Payload["width"])
		shot.Height = ToInt(artifact.Payload["height"])
	}
	return shot
}

// ToTimelineArtifact converts an artifact to TimelineArtifact.
func ToTimelineArtifact(artifact *recorder.ArtifactData) TimelineArtifact {
	payload := map[string]any{}
	if artifact.Payload != nil {
		payload = artifact.Payload
	}
	return TimelineArtifact{
		ID:           artifact.ArtifactID,
		Type:         artifact.ArtifactType,
		Label:        artifact.Label,
		StorageURL:   artifact.StorageURL,
		ThumbnailURL: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
		SizeBytes:    artifact.SizeBytes,
		StepIndex:    artifact.StepIndex,
		Payload:      payload,
	}
}
