package assessment

import (
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Target scoping.
//
// A run is about exactly one target. Two kinds of finding must not move that
// run's ladder:
//
//   - A finding whose subject is a *different* target. A provider that
//     validates more than its own target (storage-manager reports on every
//     resource, tool, and safeguard) attaches a subject to say so. Without
//     this scoping, one undeclared safeguard collapsed the ladder of the
//     scenario whose run merely carried the report.
//   - A finding belonging to a capability that is not meaningful for this
//     target's kind. `packages/api-core` has no Makefile and no
//     coverage/testing.json by construction, so a capability scoped to
//     scenarios must not score — or fail — a package run.
//
// Findings excluded here stay in the assessment's finding list. They are
// reported, they are just not scored, and `provider-conformance` reports the
// provider that emitted an out-of-scope finding rather than hiding it.

// TargetKindName maps the proto enum onto the canonical descriptor spelling
// used by `.vrooli/repo-contract.json` and every `.vrooli/test-genie.json`
// `targets.kinds` entry. An unspecified kind returns "" so callers can treat
// it as "unknown, do not discriminate".
func TargetKindName(kind commonv1.ValidationTargetKind) string {
	switch kind {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:
		return "scenario"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE:
		return "resource"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL:
		return "tool"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD:
		return "safeguard"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM:
		return "team"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE:
		return "package"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE:
		return "control-plane"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS:
		return "docs"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT:
		return "project"
	default:
		return ""
	}
}

// ScenarioTarget builds the implicit target for a scenario-shaped run. It is
// the fallback when a caller supplies no explicit target, which keeps every
// pre-target provider correct without a call-site change: their own findings
// carry either no subject or a subject naming the same scenario.
func ScenarioTarget(scenario string) *commonv1.ValidationTarget {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}
	return &commonv1.ValidationTarget{
		Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO,
		Id:   scenario,
	}
}

// findingScoresTarget reports whether a finding may move target's ladder.
//
// A nil subject always means "the run's own target", so it scores. A nil
// target means the caller could not say what the run is about; discriminating
// on a subject would then be guesswork, so every finding scores and behavior
// is exactly what it was before scoping existed.
func findingScoresTarget(subject, target *commonv1.ValidationTarget) bool {
	if subject == nil || target == nil {
		return true
	}
	if subject.GetKind() != target.GetKind() {
		return false
	}
	return strings.TrimSpace(subject.GetId()) == strings.TrimSpace(target.GetId())
}

// capabilityAppliesToKind reports whether a capability is meaningful for a
// target kind. An empty AppliesTo preserves the legacy behavior of applying
// everywhere; an unknown kind never narrows, so a caller that cannot resolve
// its target kind loses no coverage.
func capabilityAppliesToKind(capability CapabilitySpec, kindName string) bool {
	if len(capability.AppliesTo) == 0 || strings.TrimSpace(kindName) == "" {
		return true
	}
	for _, declared := range capability.AppliesTo {
		if strings.EqualFold(strings.TrimSpace(declared), kindName) {
			return true
		}
	}
	return false
}

// ScopedCapabilities returns the capabilities of spec that are meaningful for
// target, in declaration order. Capabilities that do not apply are omitted
// rather than reported at an inapplicable rung, so a package run's standing
// never advertises a scenario-only ladder it can neither satisfy nor fail.
func ScopedCapabilities(spec Spec, target *commonv1.ValidationTarget) []CapabilitySpec {
	all := mustCapabilities(spec)
	kindName := TargetKindName(target.GetKind())
	if kindName == "" {
		return all
	}
	scoped := make([]CapabilitySpec, 0, len(all))
	for _, capability := range all {
		if capabilityAppliesToKind(capability, kindName) {
			scoped = append(scoped, capability)
		}
	}
	// Never scope a target down to nothing: a spec whose every capability
	// excludes this kind is a declaration defect for provider-conformance to
	// report, not a reason to emit an empty, unfalsifiable standing.
	if len(scoped) == 0 {
		return all
	}
	return scoped
}
