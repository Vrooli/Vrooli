package evidence

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/planclient"
)

// ReconcilePlanManager pulls authoritative facts rather than accepting pushes
// from another scenario. A producer checkpoint is committed only after every
// fact before it has committed to the canonical evidence ledger.
func (s *Service) ReconcilePlanManager(ctx context.Context, reader planclient.PlanAuditReader, runID string) ([]IngestResult, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	if reader == nil || strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("plan audit reader and run id are required")
	}
	facts, err := reader.ListAuditFacts(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("fetch plan-manager audit facts: %w", err)
	}
	results := make([]IngestResult, 0, len(facts))
	cursor := ""
	for _, fact := range facts {
		if strings.TrimSpace(fact.RunID) != strings.TrimSpace(runID) {
			return nil, fmt.Errorf("plan-manager fact %q belongs to run %q, want %q", fact.EventID, fact.RunID, runID)
		}
		result, err := s.Ingest(ctx, Observation{
			SourceSystem:  "plan-manager",
			SourceEventID: fact.EventID,
			RunID:         fact.RunID,
			Subject:       Subject{Kind: "plan", ID: fact.PlanID},
			Action:        fact.Action,
			Confidence:    ConfidenceAuthoritative,
			Verification:  VerificationVerified,
			ContentDigest: fact.ContentDigest,
			Metadata:      map[string]string{"task_id": fact.TaskID},
			ObservedAt:    fact.OccurredAt,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		cursor = fact.EventID
	}
	if err := s.store.SaveCheckpoint(ctx, Checkpoint{ProducerID: "plan-manager", RunID: runID, FactKind: "plan", Cursor: cursor}); err != nil {
		return nil, fmt.Errorf("save plan-manager evidence checkpoint: %w", err)
	}
	// ListAuditFacts is a complete, run-scoped pull. The terminal watermark is
	// deliberately published only after every normalized fact and its checkpoint
	// commit successfully, so a failed ingest cannot manufacture negative
	// evidence for a requirement gate.
	if err := s.store.SaveWatermark(ctx, Watermark{ProducerID: "plan-manager", RunID: runID, FactKind: "plan", Coverage: "complete run-scoped plan audit query"}); err != nil {
		return nil, fmt.Errorf("save plan-manager evidence terminal watermark: %w", err)
	}
	return results, nil
}
