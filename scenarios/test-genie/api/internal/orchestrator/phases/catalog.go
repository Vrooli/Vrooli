package phases

import (
	"sort"
	"time"
)

// Catalog exposes the orchestrator's built-in phase registry so the API can
// clearly advertise the supported domain flows (structure, dependencies, etc.).
type Catalog struct {
	specs map[Name]Spec
}

func newCatalog() *Catalog {
	return &Catalog{specs: make(map[Name]Spec)}
}

// NewDefaultCatalog seeds the catalog with the Go-native phase runners.
func NewDefaultCatalog(defaultTimeout time.Duration) *Catalog {
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultTimeout
	}
	catalog := newCatalog()
	weight := 0
	const phaseSourceNative = "native"
	register := func(spec Spec) {
		spec.DefaultTimeout = defaultTimeout
		spec.Weight = weight
		if spec.Source == "" {
			spec.Source = phaseSourceNative
		}
		weight += 10
		catalog.Register(spec)
	}

	register(Spec{
		Name:        Structure,
		Runner:      runStructurePhase,
		Description: "Validates scenario layout, manifests, and JSON health before any tests run.",
	})
	register(Spec{
		Name:        Dependencies,
		Runner:      runDependenciesPhase,
		Description: "Confirms required commands, runtimes, and declared resources are available.",
	})
	register(Spec{
		Name:        Unit,
		Runner:      runUnitPhase,
		Description: "Executes Go unit tests and shell syntax validation for local entrypoints.",
	})
	register(Spec{
		Name:        Integration,
		Runner:      runIntegrationPhase,
		Description: "Exercises the CLI/Bats suite plus scenario-local orchestrator listings.",
	})
	register(Spec{
		Name:        Playbooks,
		Runner:      runPlaybooksPhase,
		Description: "Executes Browser Automation Studio workflows declared under test/playbooks/ to validate end-to-end UI flows.",
	})
	register(Spec{
		Name:        Business,
		Runner:      runBusinessPhase,
		Description: "Audits requirements modules to guarantee operational targets stay mapped.",
	})
	register(Spec{
		Name:        Performance,
		Runner:      runPerformancePhase,
		Optional:    true,
		Description: "Builds the Go API and enforces baseline duration budgets to catch regressions.",
	})
	return catalog
}

// Register inserts or replaces a phase specification in the catalog.
func (c *Catalog) Register(spec Spec) {
	if c == nil {
		return
	}
	name, ok := NormalizeName(spec.Name.String())
	if !ok {
		return
	}
	spec.Name = name
	if spec.DefaultTimeout <= 0 {
		spec.DefaultTimeout = DefaultTimeout
	}
	if spec.Weight == 0 && len(c.specs) > 0 {
		spec.Weight = len(c.specs) * 10
	}
	c.specs[name] = spec
}

// All returns a stable slice of registered specs sorted by their weight/name.
func (c *Catalog) All() []Spec {
	if c == nil || len(c.specs) == 0 {
		return nil
	}
	specs := make([]Spec, 0, len(c.specs))
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
func (c *Catalog) Descriptors() []Descriptor {
	specs := c.All()
	if len(specs) == 0 {
		return nil
	}
	descriptors := make([]Descriptor, 0, len(specs))
	for _, spec := range specs {
		timeout := int(spec.DefaultTimeout.Seconds())
		descriptors = append(descriptors, Descriptor{
			Name:                  spec.Name.String(),
			Optional:              spec.Optional,
			Description:           spec.Description,
			Source:                spec.Source,
			DefaultTimeoutSeconds: timeout,
		})
	}
	return descriptors
}

// Lookup resolves the spec for a user-provided name (case-insensitive).
func (c *Catalog) Lookup(raw string) (Spec, bool) {
	if c == nil {
		return Spec{}, false
	}
	name, ok := NormalizeName(raw)
	if !ok {
		return Spec{}, false
	}
	spec, exists := c.specs[name]
	return spec, exists
}

// Weight returns a deterministic ordering weight for the provided phase.
func (c *Catalog) Weight(name Name) (int, bool) {
	if c == nil {
		return 0, false
	}
	spec, ok := c.specs[name]
	if !ok {
		return 0, false
	}
	return spec.Weight, true
}
