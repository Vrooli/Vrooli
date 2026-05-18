package dbdetect

import "context"

// Priority ranks evidence strength. Higher values win.
type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
)

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

// Observation is what a Collector emits. It is the raw signal that
// profiles match against; collectors do not know about databases.
type Observation struct {
	Collector string
	Value     string
	Count     int
	Locations []string
}

// Evidence is the resolver's per-database take on a matched observation.
type Evidence struct {
	Source    string
	Priority  Priority
	Detail    string
	Locations []string
}

// Conflict is an informational divergence between sources for a DB.
type Conflict struct {
	Kind   string
	Detail string
}

// DBResult is the per-database resolution.
type DBResult struct {
	DB            string
	Required      bool
	Decision      *Evidence
	Corroborating []Evidence
	Conflicts     []Conflict
}

// DetectionReport is the full result returned by Resolver.Resolve.
type DetectionReport struct {
	Results map[string]DBResult
	Order   []string
}

// Required is the boolean projection consumed by phase orchestration.
func (r DetectionReport) Required(db string) bool {
	res, ok := r.Results[db]
	return ok && res.Required
}

// ScenarioInputs carries the data envelope for one Resolve call.
type ScenarioInputs struct {
	ScenarioDir string
	Manifest    Manifest
	Filesystem  Filesystem
}

// seam: Collector — boundary between raw scanning of scenario inputs
// and database-policy decisions made by the resolver.
type Collector interface {
	Name() string
	Collect(ctx context.Context, in ScenarioInputs) ([]Observation, error)
}

// Matcher decides whether a raw observation is evidence for a profile source.
type Matcher func(Observation) bool

// ProfileSource binds a collector to a matcher and a priority.
type ProfileSource struct {
	Collector string
	Match     Matcher
	Priority  Priority
	// Label is shown in DetectionReport.FormatHuman so the evidence
	// chain reads like documentation. Required.
	Label string
	// Tokens, if set, is the literal token list used by a Tokens() matcher.
	// The source collector consumes the union of these across all profiles
	// to bound its scan. Only meaningful when Collector == "source".
	Tokens []string
}

// Profile declares how evidence for a single DB is collected and ranked.
type Profile struct {
	DB      string
	Sources []ProfileSource
}
