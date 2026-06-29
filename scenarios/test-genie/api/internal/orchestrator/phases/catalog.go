package phases

import (
	"strings"
	"time"

	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/runnability"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
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

// NewDefaultCatalog seeds the catalog with the Go-native phase runners.
func NewDefaultCatalog(defaultTimeout time.Duration) *Catalog {
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultTimeout
	}
	catalog := newCatalog()
	const phaseSourceNative = "native"
	register := func(spec Spec) {
		// Only set default timeout if not explicitly specified
		if spec.DefaultTimeout <= 0 {
			spec.DefaultTimeout = defaultTimeout
		}
		if spec.Source == "" {
			spec.Source = phaseSourceNative
		}
		catalog.Register(spec)
	}

	register(delegatedSpec(Delegated{
		Name:             Structure,
		ProviderScenario: "structure-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		Emoji:            "🏗️",
		DetailCommand:    "structure-health validate scenario {{scenario}}",
		Timeout:          60 * time.Second,
		Description:      "Delegates scenario skeleton + lifecycle-wiring validation to structure-health, which reconciles code-facts ground truth against declared service.json intent (profile-aware) and maps findings into the FINDING_SOURCE_STRUCTURE channel before any tests run.",
	}))
	register(delegatedSpec(Delegated{
		Name:             Contracts,
		ProviderScenario: "cli-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_CLI,
		Emoji:            "📑",
		Timeout:          90 * time.Second,
		Description:      "Validates cli/manifest.json bindings against proto descriptors via cli-health, and (with execution requested) reconciles the binary's runtime CLI surface against the manifest.",
		// Opt into execution so cli-health runs its runtime CLI probe (resolve +
		// `--help`-tree walk) on top of the static manifest↔proto cross-check.
		// Degrades to a warning when the scenario's binary is absent in the run
		// context, so this never false-fails an uninstalled CLI.
		IncludeExecution: true,
	}))
	// ui-health is the single UI-validation authority. Its static groups
	// (manifest/slot/overlay, UI-interop, net-new UI standards) always run and
	// gate; with execution requested it additionally drives the BAS render +
	// iframe-bridge handshake runtime group (folded from the retired smoke
	// phase). NeedsUI + RequiredResources:[BAS] let the runnability gate SKIP
	// (never fail) the phase when no UI surface or BAS is present, mirroring the
	// performance phase's execution-mode contract; ui-health additionally
	// degrades its runtime group internally so a reachable-but-UI-less run still
	// returns the static report.
	uiHealthSpec := delegatedSpec(Delegated{
		Name:             UIHealth,
		ProviderScenario: "ui-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_UI,
		Emoji:            "🎨",
		Timeout:          5 * time.Minute,
		IncludeExecution: true,
		Description:      "Delegates all UI validation to ui-health: static ui/manifest.json + slot/overlay rules, static UI-interop, and net-new UI standards (always run and gate), plus a BAS-driven runtime render + iframe-bridge handshake group when execution is requested. Runtime checks degrade to skipped (never failed) when BAS or the UI surface is unavailable.",
	})
	uiHealthSpec.Capabilities = runnability.PhaseCapabilities{
		NeedsUI:           true,
		RequiredResources: []string{runnability.ResourceBAS},
	}
	register(uiHealthSpec)
	register(delegatedSpec(Delegated{
		Name:             Standards,
		ProviderScenario: "scenario-auditor",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Emoji:            "📏",
		DetailCommand:    "scenario-auditor standards scan {{scenario}} --wait",
		Timeout:          60 * time.Second,
		Description:      "Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config).",
		Client:           standardsDelegatedClient,
	}))
	register(delegatedSpec(Delegated{
		Name:             Architecture,
		ProviderScenario: "architecture-cartographer",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		Emoji:            "🏛️",
		DetailCommand:    "architecture-cartographer audit run {{scenario}}",
		Optional:         true,
		Timeout:          120 * time.Second,
		Description:      "Delegates structural-cohesion validation to architecture-cartographer through ScenarioValidationService; blocker findings gate only when the architecture authority is high-confidence unless TEST_GENIE_ARCHITECTURE_GATE overrides rollout mode.",
		GateEnvVar:       "TEST_GENIE_ARCHITECTURE_GATE",
		DefaultGateMode:  validationprovider.GateModeHighConfidence,
	}))
	register(delegatedSpec(Delegated{
		Name:             Dependencies,
		ProviderScenario: "scenario-dependency-analyzer",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
		Emoji:            "📦",
		DetailCommand:    "scenario-dependency-analyzer health {{scenario}}",
		Timeout:          defaultTimeout,
		Description:      "Delegates dependency readiness, runtime dependency status, governance, release-age policy, security index availability, and graph drift to scenario-dependency-analyzer through ScenarioValidationService.",
	}))
	register(delegatedSpec(Delegated{
		Name:             Quality,
		ProviderScenario: "quality-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Emoji:            "🧭",
		DetailCommand:    "quality-health audit run {{scenario}}",
		Timeout:          120 * time.Second,
		Description:      "Delegates static quality contracts, lint/type policy, and strict config validation to quality-health.",
	}))
	register(delegatedSpec(Delegated{
		Name:             Docs,
		ProviderScenario: "knowledge-observatory",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_DOCS,
		Emoji:            "📄",
		DetailCommand:    "knowledge-observatory docs health {{scenario}}",
		Timeout:          60 * time.Second,
		Description:      "Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService.",
	}))
	performanceSpec := delegatedSpec(Delegated{
		Name:             Performance,
		ProviderScenario: "performance-health",
		// Performance produces no machine-readable findings into a campaign
		// channel; it stays a non-producing phase (guarded by
		// TestFindingSourceCoversEveryProducingPhase).
		FindingSource: architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED,
		Emoji:         "⚡",
		DetailCommand: "performance-health audit {{scenario}}",
		Optional:      true,
		Timeout:       5 * time.Minute,
		// Execution mode: performance-health actually benchmarks the Go + UI build
		// and runs Lighthouse-if-UI, persists a perf sample, then gates on budgets
		// + native build-time thresholds (restoring the native phase's enforcement,
		// which readiness-only delegation had dropped). Without this the delegated
		// phase could only PASS/SKIP.
		IncludeExecution: true,
		Description:      "Delegates Go API and UI build benchmarking plus Lighthouse audits (performance, accessibility, SEO) to the performance-health scenario through ScenarioValidationService, running the measurements and gating on the result.",
	})
	// Preserve the native phase's runnability contract: Performance needs a UI
	// surface (Lighthouse + UI build), so the runnability gate skips it when no
	// UI is present rather than failing it. Pinned by
	// TestCapabilityManifestCoversEveryPhase.
	performanceSpec.Capabilities = runnability.PhaseCapabilities{NeedsUI: true}
	register(performanceSpec)
	register(delegatedSpec(Delegated{
		Name:             Unit,
		ProviderScenario: "unit-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		Emoji:            "🧪",
		Timeout:          15 * time.Minute,
		Description:      "Delegates test execution, coverage, test architecture, test quality, and flake/runtime diagnostics to the unit-health scenario, mapping coverage findings into the FINDING_SOURCE_COVERAGE channel that feeds the ecosystem-manager `coverage` dimension.",
		IncludeExecution: true,
	}))
	// Storage runs immediately before playbooks: it delegates test-isolation +
	// storage-conventions validation to storage-health and maps findings into the
	// FINDING_SOURCE_STORAGE channel. Its L2 isolation rung is the fail-closed
	// precondition the playbooks phase keys its routed-or-refuse decision off of —
	// a scenario whose routed-DB seams are unwired (or whose API isolation cannot
	// be statically verified) fails storage and has its destructive playbooks
	// refused before any real mutation can reach a non-isolated database.
	register(delegatedSpec(Delegated{
		Name:             Storage,
		ProviderScenario: "storage-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STORAGE,
		Emoji:            "🗄️",
		DetailCommand:    "storage-health validate scenario {{scenario}}",
		Timeout:          120 * time.Second,
		Description:      "Delegates storage judgment — schema layout, migration hygiene, persistence-seam adoption, and (the safety throughline) test-isolation seam-wiring — to storage-health, mapping findings into the FINDING_SOURCE_STORAGE channel. Its L2 isolation rung statically gates whether the playbooks phase may run destructive end-to-end flows against an isolated test database.",
	}))
	register(Spec{
		Name:        Playbooks,
		Runner:      runPlaybooksPhase,
		Description: "Executes Vrooli Ascension workflows declared under bas/ to validate end-to-end UI flows.",
		Capabilities: runnability.PhaseCapabilities{
			NeedsUI:                   true,
			MutatesLifecycle:          true,
			DBIsolation:               runnability.DBIsolationRouted,
			LifecycleDecisionDeferred: true,
		},
	})
	register(Spec{
		Name:          Business,
		Runner:        runBusinessPhase,
		Description:   "Audits requirements modules to guarantee operational targets stay mapped.",
		FindingSource: architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
	})
	register(delegatedSpec(Delegated{
		Name:             Tidiness,
		ProviderScenario: "tidiness-manager",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		Emoji:            "🧹",
		DetailCommand:    "tidiness-manager scan {{scenario}} --type tidiness",
		Optional:         true,
		Timeout:          120 * time.Second,
		Description:      "Delegates file/function quality checks to tidiness-manager through ScenarioValidationService and maps assessment findings into the FINDING_SOURCE_TIDINESS channel.",
	}))
	register(delegatedSpec(Delegated{
		Name:             Security,
		ProviderScenario: "security-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
		Emoji:            "🔐",
		Optional:         true,
		Timeout:          180 * time.Second,
		Description:      "Delegates security posture validation to security-health (secrets, Go SAST, Go vuln-DB, JS deps) and maps findings into the FINDING_SOURCE_SECURITY channel that gates the ecosystem-manager R1 ladder rung.",
	}))
	register(delegatedSpec(Delegated{
		Name:             Measures,
		ProviderScenario: "measures-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
		Emoji:            "📐",
		Optional:         true,
		Timeout:          180 * time.Second,
		Description:      "Delegates measures-coverage validation to measures-health (stateful-domain coverage + per-measure tier) and maps findings into the FINDING_SOURCE_MEASURES channel that feeds the ecosystem-manager soft `measures` ladder dimension.",
		IncludeExecution: true,
	}))
	register(delegatedSpec(Delegated{
		Name:             Proto,
		ProviderScenario: "proto-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		Emoji:            "🧬",
		Optional:         true,
		Timeout:          120 * time.Second,
		Description:      "Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension.",
	}))
	// Branding delegates brand-identity validation to the brand-manager scenario
	// (the single scenario that both authors and validates branding) through
	// ScenarioValidationService, mapping per-rule findings (display-name,
	// color-system, typography, logo, favicon, WCAG-AA contrast, applied brand
	// markers) into the FINDING_SOURCE_BRANDING channel and climbing a branding
	// maturity ladder. Deterministic rules expose PreviewFix/ApplyFix auto-fixes.
	// Optional so an absent brand-manager skips (never fails) the phase.
	register(delegatedSpec(Delegated{
		Name:             Branding,
		ProviderScenario: "brand-manager",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_BRANDING,
		Emoji:            "🎨",
		DetailCommand:    "test-genie execute {{scenario}} branding --wait",
		Optional:         true,
		Timeout:          120 * time.Second,
		Description:      "Delegates brand-identity validation to brand-manager through ScenarioValidationService (display-name, canonical design-token contract, typography, logo, favicon, WCAG-AA contrast, applied brand markers) and maps findings into the FINDING_SOURCE_BRANDING channel that feeds the ecosystem-manager soft `branding` ladder dimension.",
	}))
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
	if spec.Doc == "" {
		spec.Doc = docPathConvention(name)
	}
	if spec.SkipEnvVar == "" {
		spec.SkipEnvVar = skipEnvVarForPhase(name)
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
		descriptors = append(descriptors, Descriptor{
			Name:                  spec.Name.String(),
			Optional:              spec.Optional,
			Description:           spec.Description,
			Source:                spec.Source,
			DefaultTimeoutSeconds: timeout,
			DocPath:               spec.Doc,
			SkipEnvVar:            spec.SkipEnvVar,
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
