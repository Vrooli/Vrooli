// Package capabilities provides audio-tools' scenario-specific catalogue and
// checker wiring over the shared capability-registry-go state machine.
package capabilities

import (
	"strings"
	"time"

	"audio-tools/internal/providercatalog"

	"github.com/vrooli/api-core/schedule"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// CapabilityServiceability is the scenario-facing projection of provider
// states. It keeps provider evidence while exposing the shared any-available
// rule used by both health surfaces.
type CapabilityServiceability struct {
	Capability           diagv1.Capability
	Providers            []State
	Serviceable          bool
	UnavailableProviders []string
}

// Serviceability groups catalogue states by their declared feature mapping.
func Serviceability(states []State) []CapabilityServiceability {
	groups := capabilityregistry.RollupByCapability(states, func(state State) []string {
		if state.ID == "audio-tools" {
			return nil
		}
		var names []string
		for _, feature := range state.Features {
			if capability, ok := CapabilityForFeature(feature); ok {
				names = append(names, capability.String())
			}
		}
		return names
	})
	out := make([]CapabilityServiceability, 0, len(groups))
	for _, group := range groups {
		capabilityValue, ok := diagv1.Capability_value[group.Name]
		if !ok {
			continue
		}
		capability := diagv1.Capability(capabilityValue)
		out = append(out, CapabilityServiceability{
			Capability:           capability,
			Providers:            group.Providers,
			Serviceable:          group.Serviceable,
			UnavailableProviders: group.UnavailableProviders,
		})
	}
	return out
}

// RequiredCapability reports whether the audio service must have at least
// one provider for this capability.
func RequiredCapability(capability diagv1.Capability) bool {
	return capability == diagv1.Capability_CAPABILITY_STT ||
		capability == diagv1.Capability_CAPABILITY_TTS ||
		capability == diagv1.Capability_CAPABILITY_TRANSCODE
}

// RequiredFailures returns actionable descriptions for required capabilities
// without an available provider.
func RequiredFailures(states []State) []string {
	var failures []string
	for _, group := range Serviceability(states) {
		if RequiredCapability(group.Capability) && !group.Serviceable {
			name := group.Capability.String()
			if len(group.UnavailableProviders) > 0 {
				name += " (unavailable: " + strings.Join(group.UnavailableProviders, ", ") + ")"
			}
			failures = append(failures, name)
		}
	}
	return failures
}

type (
	DependencyKind  = capabilityregistry.DependencyKind
	ActionKind      = capabilityregistry.ActionKind
	Status          = capabilityregistry.Status
	PlatformSupport = capabilityregistry.PlatformSupport
	PlatformVerdict = capabilityregistry.PlatformVerdict
	Criticality     = capabilityregistry.Criticality
	Def             = capabilityregistry.Def
	State           = capabilityregistry.State
	Checker         = capabilityregistry.Checker
	ResultChecker   = capabilityregistry.ResultChecker
	CheckResult     = capabilityregistry.CheckResult
	Registry        = capabilityregistry.Registry
)

const (
	DependencyScenario        = capabilityregistry.DependencyScenario
	DependencyResource        = capabilityregistry.DependencyResource
	StatusAvailable           = capabilityregistry.StatusAvailable
	StatusUnavailable         = capabilityregistry.StatusUnavailable
	StatusUnknown             = capabilityregistry.StatusUnknown
	ActionKindNone            = capabilityregistry.ActionKindNone
	ActionKindOperatorCommand = capabilityregistry.ActionKindOperatorCommand
	ActionKindScenarioStart   = capabilityregistry.ActionKindScenarioStart
	ActionKindScenarioRestart = capabilityregistry.ActionKindScenarioRestart
	ActionKindOwnerGuidance   = capabilityregistry.ActionKindOwnerGuidance
	PlatformSupported         = capabilityregistry.PlatformSupported
	PlatformDegraded          = capabilityregistry.PlatformDegraded
	PlatformUnsupported       = capabilityregistry.PlatformUnsupported
	CriticalityRequired       = capabilityregistry.CriticalityRequired
	CriticalityOptional       = capabilityregistry.CriticalityOptional
)

// Known is the generated provider catalogue. It is rendered from
// api/internal/providercatalog/catalog.seed.json by `make catalog-gen`; do not
// hand-edit catalog.generated.go.
var Known = generatedKnown

// KnownForPlatform returns the catalogue with resource-backed platform
// verdicts applied. Keeping this construction in the catalogue makes the
// platform contract testable without relying on the process host's GOOS and
// ensures unsupported local providers are marked by design before probing.
func KnownForPlatform(goos string) []Def {
	defs := append([]Def(nil), Known...)
	resolver := NewResourcePlatformResolver(ResourcesFS(), goos)
	for i := range defs {
		if defs[i].DependencyKind == DependencyResource {
			defs[i].Platform = resolver.Resolve(defs[i].DependencySlug)
		}
	}
	return defs
}

// LocalProviderIDs returns the active local-resource provider IDs that own a
// lifecycle row, in catalogue order and deduplicated by resource slug. It is a
// thin wrapper over the generated provider catalogue.
func LocalProviderIDs() []string {
	return providercatalog.LocalProviderIDs()
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}

func NewRegistryWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, clk schedule.Clock) *Registry {
	return capabilityregistry.NewWithClock(defs, checkers, cacheTTL, func() time.Time { return clk.Now() })
}
