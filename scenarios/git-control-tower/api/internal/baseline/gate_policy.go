package baseline

import runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"

// GateDecision is GCT's one policy projection over Test Genie's independent
// comparison dimensions. Test Genie owns classification; GCT only decides
// whether a required baseline gate may pass.
type GateDecision struct {
	LegacyVerdict Verdict
	Blocking      bool
	Reason        string
}

func GateDecisionForComparison(cmp *runspb.CompareRunsResponse) GateDecision {
	if cmp == nil || cmp.GetSchemaVersion() < 2 {
		return GateDecision{VerdictNotComparable, true, "legacy comparison evidence"}
	}
	if cmp.GetBehavior() == "regression" || cmp.GetBehavior() == "new-failure" {
		return GateDecision{VerdictRegression, true, cmp.GetBehavior()}
	}
	if cmp.GetCoverage() != "measured" {
		return GateDecision{VerdictNotComparable, true, "required measurement coverage is incomplete"}
	}
	if cmp.GetCompatibility() != "compatible" {
		return GateDecision{VerdictNotComparable, true, "validation contract requires review"}
	}
	// "verified" is the V2 spelling for what is now called strict. Keep it
	// readable in historical comparison envelopes.
	if cmp.GetProvenance() != "strict" && cmp.GetProvenance() != "shared-scoped" && cmp.GetProvenance() != "verified" && cmp.GetProvenance() != "legacy-index" {
		return GateDecision{VerdictNotComparable, true, "source provenance is unavailable or changed during the test attempt"}
	}
	if cmp.GetBehavior() == "preexisting" {
		return GateDecision{LegacyVerdict: VerdictPreexisting}
	}
	return GateDecision{LegacyVerdict: VerdictClean}
}

// GateDecisionForLegacyVerdict is the historical-cache adapter. New results
// always use GateDecisionForComparison; this explicit boundary avoids silently
// reinterpreting cached pre-envelope data.
func GateDecisionForLegacyVerdict(verdict Verdict) GateDecision {
	switch verdict {
	case VerdictRegression, VerdictNewFailure:
		return GateDecision{LegacyVerdict: VerdictRegression, Blocking: true, Reason: string(verdict)}
	case VerdictNotComparable:
		return GateDecision{LegacyVerdict: VerdictNotComparable, Blocking: true, Reason: "legacy comparison is incomplete"}
	case VerdictPreexisting:
		return GateDecision{LegacyVerdict: VerdictPreexisting}
	default:
		return GateDecision{LegacyVerdict: VerdictClean}
	}
}
