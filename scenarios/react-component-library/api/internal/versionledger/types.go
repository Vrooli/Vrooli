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

// UnreadableVersion is a bounded retention defect. The version remains
// protected, but its owning asset can be reported without aborting the whole
// reference graph.
type UnreadableVersion struct {
	LibraryID string
	Version   string
	Reason    string
}

// Reachability is the single retention view shared by cleanup, retirement,
// and presence reconciliation. References are reverse edges: the map key is
// the target and each value identifies an owner that imports it.
type Reachability struct {
	References map[string][]VersionReference
	Reachable  map[string]struct{}
	Unreadable []UnreadableVersion
}

func hasUnreadableVersion(items []UnreadableVersion, libraryID, version string) bool {
	for _, item := range items {
		if item.LibraryID == libraryID && item.Version == version {
			return true
		}
	}
	return false
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
