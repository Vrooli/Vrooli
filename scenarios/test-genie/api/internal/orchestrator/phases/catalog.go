package phases

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phaseregistry"
	"test-genie/internal/orchestrator/providerdescriptor"

	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
)

// Catalog exposes the orchestrator's built-in phase registry so the API can
// clearly advertise the supported domain flows (structure, dependencies, etc.).
type Catalog struct {
	specs map[Name]Spec
	order []Name
}

func newCatalog() *Catalog {
	return &Catalog{specs: make(map[Name]Spec)}
}

// NewCatalogFromSpecs builds a catalog from already-materialized phase specs.
// Descriptor-backed orchestration uses this after phaseregistry has validated
// provider-owned descriptors and bound them to Test Genie runner implementations.
func NewCatalogFromSpecs(defaultTimeout time.Duration, specs ...Spec) *Catalog {
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultTimeout
	}
	catalog := newCatalog()
	for _, spec := range specs {
		if spec.DefaultTimeout <= 0 {
			spec.DefaultTimeout = defaultTimeout
		}
		catalog.Register(spec)
	}
	return catalog
}

// NewDefaultCatalog builds the default provider-backed catalog from
// provider-owned descriptors. Test Genie owns only the descriptor validation and
// runner binding; provider metadata lives in scenarios/*/.vrooli/test-genie.json.
func NewDefaultCatalog(defaultTimeout time.Duration) *Catalog {
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultTimeout
	}
	catalog, err := loadDefaultCatalogFromDescriptors(defaultTimeout)
	if err != nil {
		panic(err)
	}
	return catalog
}

func loadDefaultCatalogFromDescriptors(defaultTimeout time.Duration) (*Catalog, error) {
	repoRoot, err := defaultRepoRoot()
	if err != nil {
		return nil, err
	}
	load := providerdescriptor.Load(providerdescriptor.LoadOptions{RepoRoot: repoRoot})
	if err := load.Err(); err != nil {
		return nil, fmt.Errorf("load provider descriptors: %w", err)
	}
	if len(load.Descriptors) == 0 {
		return nil, fmt.Errorf("load provider descriptors: no %s files found under %s", providerdescriptor.RelPath, repoRoot)
	}
	registry, err := BuildDescriptorRegistry(load.Descriptors)
	if err != nil {
		return nil, err
	}
	return NewCatalogFromSpecs(defaultTimeout, SpecsFromRegistry(registry)...), nil
}

func defaultRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..")), nil
}

func BuildDescriptorRegistry(descriptors []providerdescriptor.Descriptor) (*phaseregistry.Registry, error) {
	result := phaseregistry.Build(descriptors, phaseregistry.Options{Bindings: ValidationProviderRegistryBindings()})
	if len(result.Diagnostics) > 0 {
		return nil, fmt.Errorf("build descriptor phase registry: %s", formatRegistryDiagnostics(result.Diagnostics))
	}
	return result.Registry, nil
}

func SpecsFromRegistry(registry *phaseregistry.Registry) []Spec {
	if registry == nil {
		return nil
	}
	specs := make([]Spec, 0, len(registry.All()))
	for _, entry := range registry.All() {
		spec, ok := SpecFromRegistryEntry(entry)
		if !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func SpecFromRegistryEntry(entry phaseregistry.Entry) (Spec, bool) {
	spec, ok := entry.Spec.(Spec)
	return spec, ok
}

func formatRegistryDiagnostics(diagnostics []phaseregistry.Diagnostic) string {
	parts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		prefix := diagnostic.Code
		if diagnostic.Phase != "" {
			prefix = diagnostic.Phase + ":" + prefix
		}
		if diagnostic.Path != "" {
			prefix = diagnostic.Path + ":" + prefix
		}
		parts = append(parts, prefix+": "+diagnostic.Message)
	}
	return strings.Join(parts, "; ")
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
	if spec.Doc == "" {
		spec.Doc = docPathConvention(name)
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		spec.DisplayName = displayNameFromKey(name.String())
	}
	if spec.SkipEnvVar == "" {
		spec.SkipEnvVar = skipEnvVarForPhase(name)
	}
	if spec.Policy.IsZero() {
		spec.Policy = phasepolicy.FromLegacyCatalog(spec.Optional, spec.Advisory)
	}
	// Keep the capability manifest in lockstep with the catalog identity: the
	// phase name and Optional flag are owned by the Spec, so mirror them into
	// the embedded manifest rather than asking every register() call to repeat
	// them. This guarantees Capabilities.Phase/Optional can never drift.
	spec.Capabilities.Phase = name.String()
	spec.Capabilities.Optional = spec.Optional
	if _, exists := c.specs[name]; !exists {
		c.order = append(c.order, name)
	}
	c.specs[name] = spec
}

// All returns registered specs in catalog registration order.
func (c *Catalog) All() []Spec {
	if c == nil || len(c.specs) == 0 {
		return nil
	}
	specs := make([]Spec, 0, len(c.specs))
	for _, name := range c.order {
		if spec, ok := c.specs[name]; ok {
			specs = append(specs, spec)
		}
	}
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
		provider := ""
		if spec.Delegated != nil {
			provider = spec.Delegated.ProviderScenario
		}
		descriptors = append(descriptors, Descriptor{
			Name:                  spec.Name.String(),
			DisplayName:           spec.DisplayName,
			Optional:              spec.Optional,
			Description:           spec.Description,
			Source:                spec.Source,
			Provider:              provider,
			DefaultTimeoutSeconds: timeout,
			DocPath:               spec.Doc,
			SkipEnvVar:            spec.SkipEnvVar,
			Comparable:            spec.Comparable(),
			Advisory:              spec.Advisory,
			ArtifactBacked:        spec.ArtifactBacked,
			NonComparable:         spec.NonComparable,
			Policy:                spec.Policy,
			Runnability:           spec.Capabilities,
			FindingSource:         findingid.SourceToken(spec.FindingSource),
			ProfileMembership:     append([]string(nil), spec.ProfileMembership...),
			FreshnessRequirement:  spec.FreshnessRequirement,
			PhaseClass:            spec.PhaseClass,
			RuntimeClass:          spec.RuntimeClass,
			Concurrency:           spec.Concurrency,
			Dimensions:            append([]string(nil), spec.Dimensions...),
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

// PhaseSetDigest returns a stable digest for a planned phase shape. It changes
// when the ordered phase set changes, so run reuse can fail closed across
// catalog evolution instead of silently reusing an older comprehensive shape.
func PhaseSetDigest(names []string) string {
	normalized := make([]string, 0, len(names))
	for _, raw := range names {
		name, ok := NormalizeName(raw)
		if !ok {
			continue
		}
		normalized = append(normalized, name.String())
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\n")))
	return "phase-set:" + hex.EncodeToString(sum[:])
}

// Comparable reports whether this phase participates in baseline/run
// comparison. Default-new phases are comparable unless explicitly opted out.
func (s Spec) Comparable() bool {
	return !s.NonComparable
}

func skipEnvVarForPhase(name Name) string {
	key := strings.ToUpper(strings.ReplaceAll(name.Key(), "-", "_"))
	if key == "" {
		return ""
	}
	return "TEST_GENIE_SKIP_" + key
}

// Order returns the zero-based registration position for the provided phase.
func (c *Catalog) Order(name Name) (int, bool) {
	if c == nil {
		return 0, false
	}
	normalized, ok := NormalizeName(name.String())
	if !ok {
		return 0, false
	}
	for index, registered := range c.order {
		if registered == normalized {
			return index, true
		}
	}
	return 0, false
}
