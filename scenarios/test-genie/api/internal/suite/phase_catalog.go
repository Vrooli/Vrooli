package suite

import (
	"sort"
	"strings"
	"time"
)

// PhaseName identifies a single orchestrator phase.
type PhaseName string

// Canonical phase names implemented by the Go orchestrator.
const (
	PhaseStructure    PhaseName = "structure"
	PhaseDependencies PhaseName = "dependencies"
	PhaseUnit         PhaseName = "unit"
	PhaseIntegration  PhaseName = "integration"
	PhaseBusiness     PhaseName = "business"
	PhasePerformance  PhaseName = "performance"
)

// phaseSpec captures metadata for a runnable phase in the catalog.
type phaseSpec struct {
	Name           PhaseName
	Runner         phaseRunnerFunc
	Optional       bool
	DefaultTimeout time.Duration
	Weight         int
	Description    string
	Source         string
}

// PhaseDescriptor surfaces metadata about registered phases so the UI/CLI can
// describe the orchestration flow without scraping bash scripts.
type PhaseDescriptor struct {
	Name        string `json:"name"`
	Optional    bool   `json:"optional"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

// PhaseCatalog exposes the orchestrator's built-in phase registry so the API can
// clearly advertise the supported domain flows (structure, dependencies, etc.).
type PhaseCatalog struct {
	specs map[PhaseName]phaseSpec
}

func newPhaseCatalog() *PhaseCatalog {
	return &PhaseCatalog{specs: make(map[PhaseName]phaseSpec)}
}

// NewDefaultPhaseCatalog seeds the catalog with the Go-native phase runners.
func NewDefaultPhaseCatalog(defaultTimeout time.Duration) *PhaseCatalog {
	catalog := newPhaseCatalog()
	weight := 0
	const phaseSourceNative = "native"
	register := func(spec phaseSpec) {
		spec.DefaultTimeout = defaultTimeout
		spec.Weight = weight
		if spec.Source == "" {
			spec.Source = phaseSourceNative
		}
		weight += 10
		catalog.Register(spec)
	}

	register(phaseSpec{
		Name:        PhaseStructure,
		Runner:      runStructurePhase,
		Description: "Validates scenario layout, manifests, and JSON health before any tests run.",
	})
	register(phaseSpec{
		Name:        PhaseDependencies,
		Runner:      runDependenciesPhase,
		Description: "Confirms required commands, runtimes, and declared resources are available.",
	})
	register(phaseSpec{
		Name:        PhaseUnit,
		Runner:      runUnitPhase,
		Description: "Executes Go unit tests and shell syntax validation for local entrypoints.",
	})
	register(phaseSpec{
		Name:        PhaseIntegration,
		Runner:      runIntegrationPhase,
		Description: "Exercises the CLI/Bats suite plus scenario-local orchestrator listings.",
	})
	register(phaseSpec{
		Name:        PhaseBusiness,
		Runner:      runBusinessPhase,
		Description: "Audits requirements modules to guarantee operational targets stay mapped.",
	})
	register(phaseSpec{
		Name:        PhasePerformance,
		Runner:      runPerformancePhase,
		Optional:    true,
		Description: "Builds the Go API and enforces baseline duration budgets to catch regressions.",
	})
	return catalog
}

// Register inserts or replaces a phase specification in the catalog.
func (c *PhaseCatalog) Register(spec phaseSpec) {
	if c == nil {
		return
	}
	name, ok := normalizePhaseName(spec.Name.String())
	if !ok {
		return
	}
	spec.Name = name
	if spec.DefaultTimeout <= 0 {
		spec.DefaultTimeout = defaultPhaseTimeout
	}
	if spec.Weight == 0 && len(c.specs) > 0 {
		spec.Weight = len(c.specs) * 10
	}
	c.specs[name] = spec
}

// All returns a stable slice of registered specs sorted by their weight/name.
func (c *PhaseCatalog) All() []phaseSpec {
	if c == nil || len(c.specs) == 0 {
		return nil
	}
	specs := make([]phaseSpec, 0, len(c.specs))
	for _, spec := range c.specs {
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Weight == specs[j].Weight {
			return specs[i].Name.String() < specs[j].Name.String()
		}
		return specs[i].Weight < specs[j].Weight
	})
	return specs
}

// Descriptors returns serialized metadata for registered phases.
func (c *PhaseCatalog) Descriptors() []PhaseDescriptor {
	specs := c.All()
	if len(specs) == 0 {
		return nil
	}
	descriptors := make([]PhaseDescriptor, 0, len(specs))
	for _, spec := range specs {
		descriptors = append(descriptors, PhaseDescriptor{
			Name:        spec.Name.String(),
			Optional:    spec.Optional,
			Description: spec.Description,
			Source:      spec.Source,
		})
	}
	return descriptors
}

// Lookup resolves the spec for a user-provided name (case-insensitive).
func (c *PhaseCatalog) Lookup(raw string) (phaseSpec, bool) {
	if c == nil {
		return phaseSpec{}, false
	}
	name, ok := normalizePhaseName(raw)
	if !ok {
		return phaseSpec{}, false
	}
	spec, exists := c.specs[name]
	return spec, exists
}

// Weight returns a deterministic ordering weight for the provided phase.
func (c *PhaseCatalog) Weight(name PhaseName) (int, bool) {
	if c == nil {
		return 0, false
	}
	spec, ok := c.specs[name]
	if !ok {
		return 0, false
	}
	return spec.Weight, true
}

// normalizePhaseName standardizes arbitrary input into a canonical PhaseName.
func normalizePhaseName(raw string) (PhaseName, bool) {
	normalized := PhaseName(strings.ToLower(strings.TrimSpace(raw)))
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

// String returns the canonical lowercase phase name.
func (n PhaseName) String() string {
	return string(n)
}

// Key returns a safe map key for the phase.
func (n PhaseName) Key() string {
	return strings.ToLower(strings.TrimSpace(n.String()))
}

// IsZero reports whether the name is empty.
func (n PhaseName) IsZero() bool {
	return n.Key() == ""
}
