package checks

import (
	"context"
	"strings"

	"business-health/internal/extraction"

	intent "intent-go"
)

// linkageCheck runs the OT ↔ requirement linkage checks in both directions:
// intent.prd_ref_unmatched (requirement → OT, intent-go's exact-ID
// semantics) and intent.ot_orphan (OT → requirement, the first caller of
// intent.CheckOrphanOutcome). P0 orphans escalate to error — preserving the
// legacy prd_operational_target_linkage signal that P0 targets without
// requirements are the loudest gap.
type linkageCheck struct{}

func (linkageCheck) Name() string { return "intent-linkage" }

func (linkageCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.PRDPresent || !c.RegistryPresent {
		return nil // presence findings cover this; the join needs both sides
	}
	var out []intent.Finding
	for _, f := range intent.CheckPRDRefResolves(c.Outcomes, c.Requirements) {
		f.Code = suffixClaim(f.Code, f.ClaimID)
		out = append(out, f)
	}
	for _, f := range intent.CheckOrphanOutcome(c.Outcomes, c.Requirements) {
		if strings.Contains(f.ClaimID, "-P0-") {
			f.Severity = "error"
		}
		f.Code = suffixClaim(f.Code, f.ClaimID)
		out = append(out, f)
	}
	return out
}

// refExistsCheck verifies code-typed validation refs resolve on disk
// (intent.ref_missing). Manual/doc refs are never path-checked (the ref
// mini-format rules live in intent-go's NormalizeRef).
type refExistsCheck struct{}

func (refExistsCheck) Name() string { return "validation-refs" }

func (refExistsCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.RegistryPresent {
		return nil
	}
	var out []intent.Finding
	for _, f := range intent.CheckRefExists(c.ScenarioDir, c.Requirements) {
		f.Code = suffixClaim(f.Code, f.ClaimID)
		out = append(out, f)
	}
	return out
}

// suffixClaim appends the claim ID to the code (`code:CLAIM`) so each
// defect gets a distinct stable ID — matching the native business phase's
// afid behavior. The assessment builder strips the suffix for maturity
// mapping lookups.
func suffixClaim(code, claimID string) string {
	if claimID == "" {
		return code
	}
	return code + ":" + claimID
}
