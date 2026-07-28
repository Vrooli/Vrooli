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
