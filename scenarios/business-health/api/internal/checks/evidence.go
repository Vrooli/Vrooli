package checks

import (
	"context"
	"fmt"

	"business-health/internal/evidence"
	"business-health/internal/extraction"
	"business-health/internal/matrix"

	intent "intent-go"
)

// evidenceCheck derives the evidence_traceability findings from the matrix
// join: stale snapshots (advisory), expired manual attestations (advisory),
// unproven claims and unearned statuses (required). Everything it reads is
// read-only composition — the join and the store never write.
type evidenceCheck struct{}

func (evidenceCheck) Name() string { return "evidence-traceability" }

func (evidenceCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.PRDPresent && !c.RegistryPresent {
		return nil
	}
	store := evidence.NewStore(c.ScenarioDir, nil)
	snap, hasSnap, err := store.ReadSnapshot()
	if err != nil {
		return []intent.Finding{{
			Code:       "business_evidence_stale",
			Severity:   "warning",
			Message:    fmt.Sprintf("requirements-sync snapshot exists but cannot be read: %v", err),
			Suggestion: "Re-run the comprehensive suite so test-genie rewrites the snapshot.",
			Locations:  []string{"coverage/requirements-sync/latest.json"},
			Provenance: "business-health",
		}}
	}
	attestations, err := store.LatestAttestations()
	if err != nil {
		attestations = nil // unreadable ledger degrades to no-manual-evidence
	}
	result := matrix.Join(matrix.Inputs{
		Contract:     c,
		Snapshot:     snap,
		HasSnapshot:  hasSnap,
		Staleness:    store.SnapshotStaleness(snap, hasSnap),
		Attestations: attestations,
		Now:          store.Now(),
	})

	var out []intent.Finding
	for _, d := range result.Drift {
		out = append(out, driftFinding(d))
	}
	return out
}

// driftFinding maps one drift entry onto the frozen vocabulary. The kinds
// and codes are 1:1; severity/clean-requirement policy lives in
// .vrooli/maturity.json (stale/expired advisory, dishonest required).
func driftFinding(d matrix.DriftEntry) intent.Finding {
	f := intent.Finding{
		Message:    d.Detail,
		ClaimID:    d.SubjectID,
		Provenance: "business-health",
	}
	switch d.Kind {
	case "stale_snapshot":
		f.Code = "business_evidence_stale"
		f.Severity = "warning"
		f.Suggestion = "Run the comprehensive suite (`vrooli scenario test <scenario>`); sync refreshes the snapshot during execution."
		f.Locations = []string{"coverage/requirements-sync/latest.json"}
	case "expired_manual":
		f.Code = "business_manual_expired"
		f.Severity = "warning"
		f.Suggestion = "Re-perform the manual procedure and record it with `business-health manual-log add`."
		f.Locations = []string{"coverage/manual-validations/log.jsonl"}
	case "status_unearned":
		f.Code = "business_status_unearned"
		f.Severity = "error"
		f.Suggestion = "Revert the status to what evidence supports, then earn the change through a comprehensive suite run."
		f.Locations = []string{"requirements/"}
	default: // unproven_claim
		f.Code = "business_unproven_claim"
		f.Severity = "warning"
		f.Suggestion = "Produce the evidence (suite run or fresh attestation) or downgrade the claim to what is actually proven."
		f.Locations = []string{"PRD.md"}
	}
	if d.SubjectID != "" {
		f.Code += ":" + d.SubjectID
	}
	return f
}
