package conflicts

import (
	"context"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// Detector is one pluggable detector. Each registered detector receives
// the snapshot + derived domain map + optional verdict provider (signals)
// and returns zero or more Conflict envelopes.
type Detector interface {
	Name() string
	Description() string
	EmitsTypes() []string

	// Detect runs over the snapshot + domain map and emits conflicts. The
	// implementation must not mutate inputs. Order of returned conflicts
	// is implementation-defined; the registry handles overall ordering.
	Detect(ctx context.Context, in DetectInput) ([]Conflict, error)
}

// DetectInput bundles every input a detector may need.
type DetectInput struct {
	Scenario string
	Snapshot graph.GraphSnapshot
	// DomainMap is the derived domain map (replaces the architecture
	// manifest). Detectors resolve a path's owning domain via
	// DomainMap.DomainFor.
	DomainMap domains.DerivedDomainMap
	// VerdictProvider returns the aggregator's verdict for a chunk.
	// Detectors that don't consult signals (e.g., cycle detector) may
	// leave this nil; detectors that do (e.g., mislocated_file) call
	// it for the chunks they care about.
	VerdictProvider VerdictProvider
}

// VerdictProvider is the seam between detectors and the signals
// aggregator. Production wires a thin adapter over signals.Service;
// tests pass a fake. The interface is intentionally batch-only:
// the snapshot + domain map + GraphContext are expensive to build,
// and a per-chunk caller (the previous shape) made DetectConflicts
// O(F²×D×S). Detectors call VerdictsFor once with every chunk they
// need scored; the slice of returned verdicts is aligned with the
// input slice.
type VerdictProvider interface {
	VerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]Verdict, error)
}

// Verdict is the local view of signals.Verdict so detector code does
// not depend on the signals package's full interface. Mirrors the
// fields a detector cares about.
type Verdict struct {
	ChunkID         string
	ChunkPath       string
	Tier            string
	TopDomain       string
	TopValue        float64
	RunnerUpDomain  string
	RunnerUpValue   float64
	Tied            bool
	EvidenceSummary string
}
