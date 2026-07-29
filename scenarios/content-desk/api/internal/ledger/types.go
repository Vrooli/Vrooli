package ledger

import "time"

type PublishRecord struct {
	ID             string
	DraftID        string
	SeriesID       string
	Channel        string
	Audience       string
	PublishedURL   string
	PlatformPostID string
	SourceKind     string
	PublishedAt    time.Time
}

// ReleaseReceipt is the minimal, credential-free outcome Content Desk accepts
// from Channel Manager. The receipt ID is an idempotency boundary: replaying a
// delivery must return the original ledger record rather than add a post.
type ReleaseReceipt struct {
	ReceiptID      string
	DraftID        string
	Channel        string
	PlatformPostID string
	PublishedURL   string
	Status         string
	PublishedAt    time.Time
}

// MetricSample is the idempotent metric-delivery contract accepted from
// Channel Manager. A sample ID identifies the measurement, not a transport
// attempt, so retries cannot duplicate analytics.
type MetricSample struct {
	SampleID   string
	ReleaseID  string
	DraftID    string
	Metric     string
	Value      float64
	ObservedAt time.Time
}

type Remediation struct {
	ID, PublishRecordID, Kind, Status, Note string
	CreatedAt, ResolvedAt                   time.Time
}

type CoverageCell struct {
	CampaignID, Lane, Channel, SKU string
	PublishCount                   int
	LastPublishedAt                time.Time
	Stale                          bool
}

type SubjectFamiliarity struct {
	Subject       string
	Audience      string
	MentionCount  int
	FirstMention  bool
	LastMentionAt time.Time
}

type NarratedItem struct {
	ID         string
	Subject    string
	Scenario   string
	OccurredAt time.Time
}

type ImportSource struct {
	Name string
	Path string
}

type SourceFailure struct {
	Source string
	Err    error
}

type ImportResult struct {
	RunID    string
	Imported int
	Skipped  int
	Failures []SourceFailure
	Complete bool
}
