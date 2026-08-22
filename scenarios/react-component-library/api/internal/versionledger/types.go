package versionledger

import "time"

type VersionLedger struct {
	LibraryID       string
	Version         string
	CreatedAt       time.Time
	ReleasedAt      time.Time
	RetiredAt       time.Time
	LifecycleState  string
	GatePassCount   int
	GateFailCount   int
	TestRuns        int
	TestPassRate    float64
	AdoptionCurrent int
	AdoptionPeak    int
	FileCount       int
	LinesOfCode     int
	DependencyCount int
}

type CleanupScope struct {
	ComponentID   string
	LibraryID     string
	OlderThanDays int
}

type CleanupItem struct {
	Candidate       Candidate
	Eligible        bool
	Reason          string
	AdoptionCount   int
	DependencyCount int
	AgeDays         int
	References      []VersionReference
}

// VersionReference explains why a released version cannot be retired. It is
// deliberately evidence-shaped: cleanup must show the exact owner and source
// expression, not merely report an opaque dependency count.
type VersionReference struct {
	Kind            string
	OwnerLibraryID  string
	OwnerVersion    string
	OwnerPath       string
	ImportSpecifier string
	Evidence        string
	OwnerScenario   string
	AdoptionID      string
}
