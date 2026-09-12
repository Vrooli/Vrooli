package cliutil

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Instance routing — the cli-core side of Baseline Modes' shadow addressing.
//
// A scenario can run more than one named instance at once: the default "live"
// variant and, during a baseline engagement, a "shadow" variant on alternate
// ports with isolated stateful namespaces (see internal/scenarioruntime's
// InstanceKey in the platform module — the canonical SSOT). cli-core cannot
// import that internal package across the module boundary, so this file mirrors
// the small, frozen slug format ("scenario" for live, "scenario@variant"
// otherwise) and owns the cli-side routing decision.
//
// Two signals select a variant, in precedence order:
//
//  1. An explicit address — a "scenario@variant" suffix on the name, or the
//     global --instance flag (recorded via SetInstanceOverride, scoped to the
//     CLI's own scenario). Highest precedence; an explicit "live" beats ambient.
//  2. The ambient VROOLI_SHADOW_SCENARIOS list — names the scenarios an active
//     engagement has shadowed, so agent/orchestrator-driven CLI calls auto-route
//     to the shadow without anyone passing a flag.
//  3. Default "live".
//
// For an unshadowed scenario with no override, every helper here returns the
// bare name unchanged, so behavior is byte-for-byte identical to before instance
// routing existed.
const (
	// EnvShadowScenarios names the scenarios currently shadowed by an active
	// baseline engagement (comma- or whitespace-separated bare scenario names).
	// It is the ambient routing signal.
	EnvShadowScenarios = "VROOLI_SHADOW_SCENARIOS"

	// DefaultVariant is the canonical live variant. Mirrors
	// scenarioruntime.DefaultVariant (separate module — cannot be imported).
	DefaultVariant = "live"

	// ShadowVariant is the variant the ambient VROOLI_SHADOW_SCENARIOS signal
	// resolves to.
	ShadowVariant = "shadow"
)

var (
	instanceOverridesMu sync.RWMutex
	// instanceOverrides records explicit --instance selections keyed by the bare
	// scenario name they apply to. Scoped per scenario so a `--instance shadow`
	// on one CLI never forces an unrelated target (e.g. agent-manager) to shadow.
	instanceOverrides = map[string]string{}
)

// SplitInstance splits a name on its last "@" into (scenario, variant). When no
// "@" is present the variant is empty. Whitespace is trimmed on both halves.
func SplitInstance(name string) (string, string) {
	name = strings.TrimSpace(name)
	if i := strings.LastIndex(name, "@"); i >= 0 {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+1:])
	}
	return name, ""
}

// AddressParseError identifies a malformed [node/]scenario[@variant]
// address. Codes are stable so callers can classify usage failures without
// matching human-readable error text.
type AddressParseError struct {
	Code    string
	Address string
	Detail  string
}

func (e *AddressParseError) Error() string {
	if e == nil {
		return "invalid scenario address"
	}
	if e.Detail == "" {
		return fmt.Sprintf("scenario address %q: %s", e.Address, e.Code)
	}
	return fmt.Sprintf("scenario address %q: %s", e.Address, e.Detail)
}

// SplitAddress parses the canonical cross-node address grammar:
// [node/]scenario[@variant]. It is deliberately pure: no environment
// variable or ambient routing state can select a node. The node is empty for
// a local address, and the existing SplitInstance semantics remain unchanged
// for the scenario and variant components.
func SplitAddress(address string) (node, scenario, variant string, err error) {
	original := strings.TrimSpace(address)
	remaining := original
	if separator := strings.IndexByte(remaining, '/'); separator >= 0 {
		node = strings.TrimSpace(remaining[:separator])
		remaining = strings.TrimSpace(remaining[separator+1:])
		if node == "" {
			return "", "", "", &AddressParseError{
				Code:    "empty_node",
				Address: original,
				Detail:  "node prefix is empty",
			}
		}
	}
	scenario, variant = SplitInstance(remaining)
	if scenario == "" {
		return "", "", "", &AddressParseError{
			Code:    "empty_scenario",
			Address: original,
			Detail:  "scenario name is empty",
		}
	}
	return node, scenario, variant, nil
}

// BareScenarioName returns the scenario name with any "@variant" suffix removed.
func BareScenarioName(name string) string {
	base, _ := SplitInstance(name)
	return base
}

func normalizeVariant(variant string) string {
	variant = strings.ToLower(strings.TrimSpace(variant))
	if variant == "" {
		return DefaultVariant
	}
	return variant
}

// InstanceSlug renders the canonical record identifier for a (scenario, variant)
// pair: the bare scenario for the live variant, "scenario@variant" otherwise.
// Round-trips with SplitInstance. Mirrors scenarioruntime.InstanceKey.Slug().
func InstanceSlug(scenario, variant string) string {
	scenario = BareScenarioName(scenario)
	variant = normalizeVariant(variant)
	if variant == DefaultVariant {
		return scenario
	}
	return scenario + "@" + variant
}

// SetInstanceOverride records the explicit instance variant selected for a
// scenario via the global --instance flag. An empty variant clears the override
// (no flag given). An explicit "live" is preserved as an override so it can beat
// the ambient VROOLI_SHADOW_SCENARIOS signal. Scenario is matched on its bare
// name.
func SetInstanceOverride(scenario, variant string) {
	scenario = BareScenarioName(scenario)
	if scenario == "" {
		return
	}
	variant = strings.TrimSpace(variant)
	instanceOverridesMu.Lock()
	defer instanceOverridesMu.Unlock()
	if variant == "" {
		delete(instanceOverrides, scenario)
		return
	}
	instanceOverrides[scenario] = normalizeVariant(variant)
}

func instanceOverride(scenario string) (string, bool) {
	instanceOverridesMu.RLock()
	defer instanceOverridesMu.RUnlock()
	v, ok := instanceOverrides[BareScenarioName(scenario)]
	return v, ok
}

// ShadowedScenarios parses VROOLI_SHADOW_SCENARIOS into a set of bare scenario
// names. Accepts comma- or whitespace-separated values and tolerates stray
// "@variant" suffixes (reduced to the bare name).
func ShadowedScenarios() map[string]struct{} {
	out := map[string]struct{}{}
	raw := os.Getenv(EnvShadowScenarios)
	for _, field := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) {
		if base := BareScenarioName(field); base != "" {
			out[base] = struct{}{}
		}
	}
	return out
}

// IsShadowed reports whether a scenario is named in VROOLI_SHADOW_SCENARIOS.
func IsShadowed(name string) bool {
	_, ok := ShadowedScenarios()[BareScenarioName(name)]
	return ok
}

// ResolveVariant returns the effective instance variant for a scenario name,
// applying precedence: explicit "@variant" suffix or --instance override >
// ambient VROOLI_SHADOW_SCENARIOS > default live.
func ResolveVariant(name string) string {
	base, suffix := SplitInstance(name)
	if suffix != "" {
		return normalizeVariant(suffix)
	}
	if v, ok := instanceOverride(base); ok {
		return v
	}
	if IsShadowed(base) {
		return ShadowVariant
	}
	return DefaultVariant
}

// ResolveShadowTarget returns the scenario identifier that CLI port lookups and
// discovery should address after applying instance-routing precedence: the bare
// name for the live variant, or "scenario@variant" otherwise.
func ResolveShadowTarget(name string) string {
	base, _ := SplitInstance(name)
	return InstanceSlug(base, ResolveVariant(name))
}

// IsNonLiveTarget reports whether a resolved target addresses a non-live variant
// (i.e. carries an "@variant" suffix other than live).
func IsNonLiveTarget(target string) bool {
	_, variant := SplitInstance(target)
	return normalizeVariant(variant) != DefaultVariant
}

var shadowFallbackWarned sync.Map

// WarnShadowFallback emits a one-time stderr warning that a requested shadow
// instance could not be found and the live instance is being used instead.
// Deduplicated per scenario per process so repeated port lookups never spam.
func WarnShadowFallback(scenario string) {
	scenario = BareScenarioName(scenario)
	if scenario == "" {
		return
	}
	if _, loaded := shadowFallbackWarned.LoadOrStore(scenario, struct{}{}); loaded {
		return
	}
	fmt.Fprintf(os.Stderr, "⚠ Requested shadow instance for %s not found — it may have been torn down. Using LIVE.\n", scenario)
}

// ResetShadowFallbackWarning clears the one-time warn dedup for a scenario so
// the next WarnShadowFallback fires again. Intended for tests that exercise the
// shadow→live fallback path.
func ResetShadowFallbackWarning(scenario string) {
	shadowFallbackWarned.Delete(BareScenarioName(scenario))
}
