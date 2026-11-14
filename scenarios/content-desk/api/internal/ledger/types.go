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

// Metric state labels for a truthful metrics projection. A measured zero is
// reported as MetricStateMeasured with Value==0; absent measurements are not
// projected as readings at all and are surfaced by HasMeasurements==false, so
// zero is never confused with missing data.
const (
	MetricStateMeasured = "measured"
	MetricStateStale    = "stale"
)

// MetricReading is the latest observed value for one metric of one draft,
// derived only from stored samples. State is measured when the latest
// observation is within the freshness bound and stale otherwise; the value and
// count are always the retained observation, never a substituted default.
type MetricReading struct {
	Metric         string
	Value          float64
	State          string
	SampleCount    int
	LastObservedAt time.Time
}

// DraftMetricProjection is the current-state metrics read for one draft.
// HasMeasurements is false when no samples exist at all, which is distinct from
// a measured zero. Readings is ordered by metric name and holds the latest
// sample per metric across every release attributed to the draft.
type DraftMetricProjection struct {
	DraftID         string
	HasMeasurements bool
	Readings        []MetricReading
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
