// Package systemevents normalizes host-level events into an operator timeline.
package systemevents

import "time"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type SourceStatusState string

const (
	SourceOK          SourceStatusState = "ok"
	SourceUnsupported SourceStatusState = "unsupported"
	SourceDegraded    SourceStatusState = "degraded"
	SourceFailed      SourceStatusState = "failed"
)

type Event struct {
	ID          int64          `json:"id,omitempty"`
	Fingerprint string         `json:"fingerprint"`
	OccurredAt  time.Time      `json:"occurredAt"`
	IngestedAt  time.Time      `json:"ingestedAt,omitempty"`
	Source      string         `json:"source"`
	Platform    string         `json:"platform"`
	Category    string         `json:"category"`
	Severity    Severity       `json:"severity"`
	Title       string         `json:"title"`
	Summary     string         `json:"summary"`
	BootID      string         `json:"bootId,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

type SourceStatus struct {
	Source         string            `json:"source"`
	Platform       string            `json:"platform"`
	Status         SourceStatusState `json:"status"`
	LastIngestedAt time.Time         `json:"lastIngestedAt,omitempty"`
	LastError      string            `json:"lastError,omitempty"`
	Capabilities   map[string]any    `json:"capabilities,omitempty"`
}

type Filters struct {
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Category  []string
	Severity  []Severity
	Source    []string
	Platform  []string
	BootID    string
	Correlate bool
}

type Correlation struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Rationale    string   `json:"rationale"`
	EventIDs     []int64  `json:"eventIds"`
	EventSources []string `json:"eventSources"`
	TimeDelta    string   `json:"timeDelta,omitempty"`
	Confidence   string   `json:"confidence"`
}

type Response struct {
	Events       []Event        `json:"events"`
	Count        int            `json:"count"`
	Sources      []SourceStatus `json:"sources"`
	Filters      FiltersEcho    `json:"filters"`
	Correlations []Correlation  `json:"correlations,omitempty"`
}

type FiltersEcho struct {
	Since     string     `json:"since,omitempty"`
	Until     string     `json:"until,omitempty"`
	Limit     int        `json:"limit"`
	Category  []string   `json:"category,omitempty"`
	Severity  []Severity `json:"severity,omitempty"`
	Source    []string   `json:"source,omitempty"`
	Platform  []string   `json:"platform,omitempty"`
	BootID    string     `json:"bootId,omitempty"`
	Correlate bool       `json:"correlate"`
}

type IngestSummary struct {
	Ingested   int            `json:"ingested"`
	Deduped    int            `json:"deduped"`
	Sources    []SourceStatus `json:"sources"`
	DurationMs int64          `json:"durationMs"`
	// ExecsAvoided is the cumulative count of kernel-grep journalctl
	// invocations skipped since process start via incremental ingestion
	// (already-scanned historical boots + cursor-based current-boot reads).
	ExecsAvoided int64 `json:"execsAvoided"`
}
