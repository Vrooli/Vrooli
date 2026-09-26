// Package claimtypes defines the evaluator-backed claim vocabulary shared by
// contract parsing and runtime reconciliation.
package claimtypes

import "sort"

var implemented = []string{
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

// ImplementedClaimTypes returns the claim types for which reconciliation owns
// a deterministic evaluator. The sorted copy keeps parser and registry output
// stable without exposing mutable package state.
func ImplementedClaimTypes() []string {
	out := append([]string(nil), implemented...)
	sort.Strings(out)
	return out
}

// IsImplemented reports whether a claim type has a deterministic evaluator.
func IsImplemented(claimType string) bool {
	for _, implementedType := range implemented {
		if implementedType == claimType {
			return true
		}
	}
	return false
}
