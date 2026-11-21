package runannotations

import (
	"context"
	"time"
)

// DefaultRefreshLimit is how many stale provenance rows one refresh recomputes.
const DefaultRefreshLimit = 200

// ProvenanceRefresh reports one provenance recompute.
type ProvenanceRefresh struct {
	// Policy is the provenance policy hash the rows were recomputed under.
	Policy string `json:"policy"`
	// Refreshed rows were recomputed; Failed rows could not be read (a deleted
	// run) or written (a busy database) and stay stale; Remaining are the stale rows still waiting.
	Refreshed int `json:"refreshed"`
	Failed    int `json:"failed"`
	Remaining int `json:"remaining"`
}

// RefreshProvenance recomputes up to limit annotated runs whose stored
// provenance was computed under another policy. It reads each run's facts and
// writes only its authorship row: no label is evaluated and no model is asked,
// so a policy edit reaches coverage, the mood population and the series without
// re-annotating anything. The pass stops early when the pack's per-pass time
// budget is spent; Remaining says whether to call it again.
func (s *Service) RefreshProvenance(ctx context.Context, limit int) (ProvenanceRefresh, error) {
	if limit <= 0 {
		limit = DefaultRefreshLimit
	}
	policy := s.pack.Pack.Runtime.Provenance
	hash := policy.Hash()
	result := ProvenanceRefresh{Policy: hash}
	ids, err := s.store.StaleProvenanceRuns(ctx, hash, limit)
	if err != nil {
		return result, err
	}
	deadline := s.now().Add(s.passBudget(0))
	for _, id := range ids {
		if ctx.Err() != nil || !s.now().Before(deadline) {
			break
		}
		facts, err := s.features.Facts(ctx, ScopeRun, id)
		if err != nil {
			result.Failed++
			continue
		}
		if facts.RunID == "" {
			facts.RunID = id
		}
		policy.Apply(&facts)
		row := ProvenanceRow{
			RunID: id, WorkloadKind: facts.Workload.Kind,
			OperatorMessages: len(facts.HumanMessages) + facts.BriefMessages,
			HumanMessages:    len(facts.HumanMessages), BriefMessages: facts.BriefMessages,
			UpdatedAt: s.now().UTC(), PolicyHash: hash,
		}
		if count := len(facts.HumanMessages); count > 0 {
			row.LastHumanAt = facts.HumanMessages[count-1].At
		}
		if err := s.store.SaveProvenance(ctx, row); err != nil {
			// A busy database must not abandon the pass: the row stays stale and
			// the next pass retries it.
			result.Failed++
			continue
		}
		result.Refreshed++
	}
	summary, err := s.store.ProvenanceSummary(ctx, time.Time{}, time.Time{}, hash)
	if err != nil {
		return result, err
	}
	result.Remaining = summary.RunsStale
	return result, nil
}
