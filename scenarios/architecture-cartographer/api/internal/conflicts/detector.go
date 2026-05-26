package conflicts

import (
	"context"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
)

// Detector is one pluggable detector. Each registered detector receives
// the snapshot + manifest + optional verdict provider (signals) and
// returns zero or more Conflict envelopes.
type Detector interface {
	Name() string
	Description() string
	EmitsTypes() []string

	// Detect runs over the snapshot + manifest and emits conflicts. The
	// implementation must not mutate inputs. Order of returned conflicts
	// is implementation-defined; the registry handles overall ordering.
	Detect(ctx context.Context, in DetectInput) ([]Conflict, error)
}

// DetectInput bundles every input a detector may need.
type DetectInput struct {
	Scenario string
	Snapshot graph.GraphSnapshot
	Manifest manifest.ManifestDefinition
	// VerdictProvider returns the aggregator's verdict for a chunk.
	// Detectors that don't consult signals (e.g., cycle detector) may
	// leave this nil; detectors that do (e.g., mislocated_file) call
	// it for the chunks they care about.
	VerdictProvider VerdictProvider
}

// VerdictProvider is the seam between detectors and the signals
// aggregator. Production wires a thin adapter over signals.Service;
// tests pass a fake.
type VerdictProvider interface {
	VerdictFor(ctx context.Context, scenario string, chunk graph.Chunk) (Verdict, error)
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
