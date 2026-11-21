// Package claimtypes defines recognized claim types shared by contract parsing
// and runtime reconciliation. Some geometry types remain recognized for
// authored-contract compatibility while ui-health owns their evaluation.
package claimtypes

import "sort"

var recognized = []string{
	"accessible-name",
	"affordance-present",
	"announced",
	"chrome-color-agreement",
	"chrome-pinned",
	"content-not-clipped",
	"dark-parity",
	"differential",
	"element-absent",
	"element-present",
	"error-association",
	"font-size",
	"focus-contained",
	"focus-containment",
	"focus-order",
	"focus-restored",
	"heading-hierarchy",
	"keyboard-reachable",
	"layered-dismissal",
	"motion-duration",
	"no-document-horizontal-overflow",
	"no-layout-animation",
	"portal-boundary",
	"reading-order",
	"responsive-transformation",
	"safe-area-tap-targets",
	"single-dominant-action",
	"single-line-chrome",
	"size-parity",
	"spacing",
	"state-contrast",
	"state-covered",
	"state-distinct",
	"tap-target-size",
	"token-resolution",
	"transition-class-conformance",
	"viewport-fill",
	"visible-without-scroll",
}

var delegatedGeometry = map[string]bool{
	"chrome-pinned":                   true,
	"no-document-horizontal-overflow": true,
	"safe-area-tap-targets":           true,
	"single-line-chrome":              true,
	"tap-target-size":                 true,
	"viewport-fill":                   true,
}

// ImplementedClaimTypes returns claim types with a deterministic Experience
// Manager evaluator. Geometry delegated to ui-health is intentionally absent.
func ImplementedClaimTypes() []string {
	out := make([]string, 0, len(recognized)-len(delegatedGeometry))
	for _, claimType := range recognized {
		if !delegatedGeometry[claimType] {
			out = append(out, claimType)
		}
	}
	sort.Strings(out)
	return out
}

// IsImplemented reports whether a claim type has a deterministic evaluator.
func IsImplemented(claimType string) bool {
	for _, knownType := range recognized {
		if knownType == claimType && !delegatedGeometry[knownType] {
			return true
		}
	}
	return false
}

// IsKnown reports whether a parser recognizes a claim type, including rules
// delegated to another canonical evaluator.
func IsKnown(claimType string) bool {
	for _, knownType := range recognized {
		if knownType == claimType {
			return true
		}
	}
	return false
}
