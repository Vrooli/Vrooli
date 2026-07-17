package agentopsdiag

import (
	"context"

	"connectrpc.com/connect"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// GetWorkflowProjection returns the canonical workflow projection: the durable
// instance (state, decisions, timers, legal actions) plus each operation record
// enriched from its immutable execution snapshot. It reads only the canonical
// workflow.json + execution snapshots — never a legacy store.
func (s *Service) GetWorkflowProjection(_ context.Context, req *connect.Request[apipb.AgentOpsGetWorkflowProjectionRequest]) (*connect.Response[apipb.AgentOpsGetWorkflowProjectionResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	w, found, err := s.repo.Load(target.Kind, target.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &apipb.AgentOpsGetWorkflowProjectionResponse{Found: found}
	if !found {
		return connect.NewResponse(resp), nil
	}
	resp.Workflow = workflowToProto(w)
	if lp, ok := s.catalog.PolicyForDomain(w.Domain.Kind); ok {
		resp.PolicyId = lp.Policy.ID
		resp.PolicyRevision = lp.Revision
	}

	// attempt/retry linkage: a retry is always a NEW record for the same
	// operation, so the ordinal among same-operation records in workflow order
	// is the attempt number, and the previous record is the prior attempt.
	attempts := map[agentops.OperationID]int{}
	priorExec := map[agentops.OperationID]string{}
	for _, op := range w.Operations {
		attempts[op.Operation]++
		proj := &apipb.AgentOpsOperationProjection{
			Operation:        string(op.Operation),
			ExecutionId:      op.ExecutionID,
			RunId:            op.RunID,
			State:            op.State,
			Outcome:          op.Outcome,
			IdempotencyKey:   op.IdempotencyKey,
			ProvenanceDigest: op.ProvenanceDigest,
			Attempt:          int32(attempts[op.Operation]),
			PriorExecutionId: priorExec[op.Operation],
		}
		priorExec[op.Operation] = op.ExecutionID

		// Enrich from the immutable snapshot; its absence degrades the row, not
		// the projection (snapshot_found=false).
		if snap, ok, snapErr := s.executions.Load(target.Kind, target.ID, op.ExecutionID); snapErr == nil && ok {
			proj.SnapshotFound = true
			if snap.LegacyImport != nil {
				// A Phase-8 legacy execution import: no compiled-mode/binding
				// provenance ever existed, so those fields stay honestly empty;
				// the row is labeled instead of fabricated.
				proj.LegacyImport = true
				proj.RecordedAt = snap.RecordedAt
			} else {
				proj.OperationVersion = snap.Provenance.OperationVersion
				proj.Mode = snap.Provenance.Mode
				proj.ModeRevision = snap.Provenance.ModeRevision
				proj.BindingLayer = bindingLayerToProto(snap.Provenance.Binding.Layer)
				proj.BindingOwnerKind = snap.Provenance.Binding.OwnerKind
				proj.BindingOwnerId = snap.Provenance.Binding.OwnerID
				proj.RecordedAt = snap.RecordedAt
			}
		}
		resp.Operations = append(resp.Operations, proj)
	}
	return connect.NewResponse(resp), nil
}

// ListExecutionHistory returns execution provenance summaries for a target,
// newest first, optionally capped by limit.
func (s *Service) ListExecutionHistory(_ context.Context, req *connect.Request[apipb.AgentOpsListExecutionHistoryRequest]) (*connect.Response[apipb.AgentOpsListExecutionHistoryResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	stored, err := s.executions.List(target.Kind, target.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if limit := int(req.Msg.GetLimit()); limit > 0 && len(stored) > limit {
		stored = stored[:limit]
	}
	resp := &apipb.AgentOpsListExecutionHistoryResponse{}
	for _, se := range stored {
		p := se.Snapshot.Provenance
		if se.Snapshot.LegacyImport != nil {
			// Phase-8 legacy execution import: render only the fields the import
			// really carries (operation, the legacy entry's own terminal status,
			// the deterministic imported_at). No provenance digests ever existed,
			// so the row is labeled legacy_import and never claims reproducibility.
			resp.Executions = append(resp.Executions, &apipb.AgentOpsExecutionSummary{
				ExecutionId:  se.ID,
				Operation:    string(p.Operation),
				Outcome:      se.Snapshot.Outcome,
				Reproducible: false,
				RecordedAt:   se.Snapshot.RecordedAt,
				LegacyImport: true,
			})
			continue
		}
		_, reproErr := s.executions.Reproduce(target.Kind, target.ID, se.ID)
		resp.Executions = append(resp.Executions, &apipb.AgentOpsExecutionSummary{
			ExecutionId:         se.ID,
			Operation:           string(p.Operation),
			OperationVersion:    p.OperationVersion,
			Mode:                p.Mode,
			ModeRevision:        p.ModeRevision,
			BindingLayer:        bindingLayerToProto(p.Binding.Layer),
			CompiledModeDigest:  p.CompiledModeDigest,
			PromptCatalogDigest: p.PromptCatalogDigest,
			CallerInputDigest:   p.CallerInputDigest,
			Outcome:             se.Snapshot.Outcome,
			Reproducible:        reproErr == nil,
			RecordedAt:          se.Snapshot.RecordedAt,
		})
	}
	return connect.NewResponse(resp), nil
}

// GetMigrationStatus reads the Phase-8 migration-status document. An absent
// document (or an unconfigured path) is the not-started state, never an error.
func (s *Service) GetMigrationStatus(_ context.Context, _ *connect.Request[apipb.AgentOpsGetMigrationStatusRequest]) (*connect.Response[apipb.AgentOpsGetMigrationStatusResponse], error) {
	resp := &apipb.AgentOpsGetMigrationStatusResponse{State: opsrunner.MigrationNotStarted}
	if s.migrationStatusPath == "" {
		return connect.NewResponse(resp), nil
	}
	st, found, err := opsrunner.LoadMigrationStatus(s.migrationStatusPath)
	if err != nil {
		// A present-but-invalid status document must never masquerade as
		// not-started: fail closed so the operator investigates.
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp.DocumentFound = found
	resp.State = st.State
	resp.Epoch = int32(st.Epoch)
	resp.StagedCount = int32(st.StagedCount)
	resp.PromotedCount = int32(st.PromotedCount)
	resp.QuarantinedCount = int32(st.QuarantinedCount)
	resp.StartedAt = st.StartedAt
	resp.UpdatedAt = st.UpdatedAt
	resp.ReportPath = st.ReportPath
	return connect.NewResponse(resp), nil
}
