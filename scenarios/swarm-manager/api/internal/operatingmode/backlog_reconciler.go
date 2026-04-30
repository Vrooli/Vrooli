package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Service) CompleteItems(ctx context.Context, req CompleteItemsRequest) (BacklogSyncResult, error) {
	if s.backlogMut == nil {
		return BacklogSyncResult{}, errors.New("operatingmode: BacklogMutator is not configured")
	}
	name := strings.TrimSpace(req.InitiativeName)
	if name == "" {
		return BacklogSyncResult{}, fmt.Errorf("initiative name is required")
	}
	mode, err := requireRoundActionMode(Mode(req.Mode))
	if err != nil {
		return BacklogSyncResult{}, err
	}
	round, err := s.store.LoadRound(name, mode, req.Round)
	if err != nil {
		return BacklogSyncResult{}, err
	}
	def, err := DefinitionFor(mode)
	if err != nil {
		return BacklogSyncResult{}, err
	}
	if def.Mode == ModeItemLevel {
		return BacklogSyncResult{}, fmt.Errorf("item-level mode backlog completion is owned by the existing backlog execution flow")
	}
	if !hasBacklogSyncCapability(def.BacklogSync, BacklogSyncMarkComplete) {
		return BacklogSyncResult{}, fmt.Errorf("mode %q does not allow marking backlog items complete", mode)
	}
	runID := strings.TrimSpace(req.RunID)
	if def.BacklogSync.RequiresRunID && runID == "" {
		return BacklogSyncResult{}, fmt.Errorf("run_id is required")
	}
	if def.BacklogSync.RequiresRunID && strings.TrimSpace(round.RunID) != runID {
		return BacklogSyncResult{}, fmt.Errorf("run_id does not match round %03d", round.Round)
	}
	if len(req.ItemRefs) == 0 {
		return BacklogSyncResult{}, fmt.Errorf("item_refs is required")
	}
	memberRefs := map[string]bool{}
	for _, item := range round.Items {
		memberRefs[strings.TrimSpace(item.Ref)] = true
	}
	source := BacklogMutationSource{
		Entrypoint:     "initiative.operating_mode.complete_items",
		InitiativeName: name,
		Mode:           round.Mode,
		Phase:          round.Phase,
		Round:          round.Round,
		RunID:          round.RunID,
		RequestedBy:    defaultString(req.RequestedBy, s.requestedBy),
	}

	completed := make([]BacklogCompletionResult, 0, len(req.ItemRefs))
	completedRefs := make([]string, 0, len(req.ItemRefs))
	for _, ref := range req.ItemRefs {
		kind, itemName, cleanRef, err := parseBacklogRef(ref)
		if err != nil {
			return BacklogSyncResult{}, err
		}
		if def.BacklogSync.RequiresMembership && !memberRefs[cleanRef] {
			return BacklogSyncResult{}, fmt.Errorf("item %q is not a member of initiative %q", cleanRef, name)
		}
		result, err := s.backlogMut.MarkBacklogItemCompleted(ctx, kind, itemName, source)
		if err != nil {
			return BacklogSyncResult{}, err
		}
		completed = append(completed, result)
		completedRefs = append(completedRefs, cleanRef)
	}

	result := BacklogSyncResult{
		InitiativeName: name,
		Mode:           round.Mode,
		Phase:          round.Phase,
		Round:          round.Round,
		RunID:          round.RunID,
		CompletedItems: completed,
		Noop:           len(completed) == 0,
	}
	MutableRoundPayload(&round).SetBacklogSync(result)
	if err := s.store.SaveRound(round); err != nil {
		return BacklogSyncResult{}, err
	}
	s.emitBacklogSyncedWithSource(round, len(completed), 0, 0, source, completedRefs)
	return result, nil
}

func (s *Service) ApplyBacklogSync(ctx context.Context, req ApplyBacklogSyncRequest) (BacklogSyncResult, error) {
	if s.reconciler == nil {
		return BacklogSyncResult{}, errors.New("operatingmode: ProposalReconciler is not configured")
	}
	name := strings.TrimSpace(req.InitiativeName)
	if name == "" {
		return BacklogSyncResult{}, fmt.Errorf("initiative name is required")
	}
	mode, err := requireRoundActionMode(Mode(req.Mode))
	if err != nil {
		return BacklogSyncResult{}, err
	}
	round, err := s.store.LoadRound(name, mode, req.Round)
	if err != nil {
		return BacklogSyncResult{}, err
	}
	def, err := DefinitionFor(mode)
	if err != nil {
		return BacklogSyncResult{}, err
	}
	if def.Mode == ModeItemLevel {
		return BacklogSyncResult{}, fmt.Errorf("item-level mode backlog reconciliation is owned by the existing backlog execution flow")
	}
	if !hasBacklogSyncCapability(def.BacklogSync, BacklogSyncProposeMutations) {
		return BacklogSyncResult{}, fmt.Errorf("mode %q does not allow backlog mutation proposals", mode)
	}
	runID := strings.TrimSpace(req.RunID)
	if def.BacklogSync.RequiresRunID && runID == "" {
		return BacklogSyncResult{}, fmt.Errorf("run_id is required")
	}
	if def.BacklogSync.RequiresRunID && strings.TrimSpace(round.RunID) != runID {
		return BacklogSyncResult{}, fmt.Errorf("run_id does not match round %03d", round.Round)
	}
	plan, err := backlogSyncPlanFromRound(round)
	if err != nil {
		return BacklogSyncResult{}, err
	}
	if len(plan.Proposal) == 0 {
		return BacklogSyncResult{}, fmt.Errorf("round %03d has no backlog_sync proposal", round.Round)
	}
	now := s.clock().UTC().Format(time.RFC3339)
	result, err := s.reconciler.ApplyBacklogSyncProposal(ctx, ProposalReconcileRequest{
		InitiativeName:      name,
		Mode:                round.Mode,
		Round:               round.Round,
		Phase:               round.Phase,
		RunID:               round.RunID,
		Proposal:            append(json.RawMessage(nil), plan.Proposal...),
		AcceptedMutationIDs: append([]string(nil), req.AcceptedMutationIDs...),
		DecidedBy:           defaultString(req.RequestedBy, s.requestedBy),
		DecidedAtRFC3339:    now,
	})
	if err != nil {
		return BacklogSyncResult{}, fmt.Errorf("apply backlog sync proposal: %w", err)
	}
	syncResult := BacklogSyncResult{
		InitiativeName: name,
		Mode:           round.Mode,
		Phase:          round.Phase,
		Round:          round.Round,
		RunID:          round.RunID,
		ProposalResult: result,
		Noop:           result == nil || result.Applied == 0,
	}
	payload := MutableRoundPayload(&round)
	payload.SetBacklogSync(syncResult)
	payload.SetBacklogSyncAppliedAt(now)
	if err := s.store.SaveRound(round); err != nil {
		return BacklogSyncResult{}, err
	}
	created, updated := 0, 0
	itemRefs := []string(nil)
	if result != nil {
		created = result.Created
		updated = result.Updated
		itemRefs = appliedProposalTargets(result.Outcomes)
	}
	s.emitBacklogSyncedWithSource(round, 0, created, updated, BacklogMutationSource{
		Entrypoint:     "initiative.operating_mode.backlog_sync",
		InitiativeName: name,
		Mode:           round.Mode,
		Phase:          round.Phase,
		Round:          round.Round,
		RunID:          round.RunID,
		RequestedBy:    defaultString(req.RequestedBy, s.requestedBy),
	}, itemRefs)
	return syncResult, nil
}

func appliedProposalTargets(outcomes []ProposalOutcome) []string {
	refs := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		if !outcome.Applied {
			continue
		}
		target := strings.TrimSpace(outcome.Target)
		if target == "" {
			continue
		}
		refs = append(refs, target)
	}
	return refs
}

func hasBacklogSyncCapability(policy BacklogSyncPolicy, want BacklogSyncCapability) bool {
	for _, capability := range policy.Capabilities {
		if capability == want {
			return true
		}
	}
	return false
}

func backlogSyncPlanFromRound(round RoundEnvelope) (BacklogSyncPlan, error) {
	return RoundPayload(round.Payload).BacklogSyncPlan(round.Round)
}

func parseBacklogRef(ref string) (kind, name, clean string, err error) {
	clean = strings.TrimSpace(ref)
	parts := strings.SplitN(clean, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", "", fmt.Errorf("item ref %q must be kind/name", ref)
	}
	kind = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])
	clean = kind + "/" + name
	return kind, name, clean, nil
}
