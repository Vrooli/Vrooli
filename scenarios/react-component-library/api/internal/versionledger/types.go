package versionledger

import (
	"fmt"
	"time"
)

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
	Presence        string
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

// ErrEvictionMirrorMismatch prevents destructive removal when the durable
// mirror no longer describes the bytes on disk.
type ErrEvictionMirrorMismatch struct {
	LibraryID string
	Version   string
	Path      string
	Expected  string
	Actual    string
}

func (e ErrEvictionMirrorMismatch) Error() string {
	return fmt.Sprintf("cannot evict %s@%s: mirror mismatch at %s (expected %s, got %s)", e.LibraryID, e.Version, e.Path, e.Expected, e.Actual)
}
