package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const initialRecoveryEpoch uint64 = 1

// currentRecoveryEpoch is the single read path for capability validation. A
// restored metadata database starts at epoch one; activation advances its
// durable row before any recovered authority is made usable.
func (p *PasswordManager) currentRecoveryEpoch(ctx context.Context, workspace string) (uint64, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return 0, fmt.Errorf("workspace is required for recovery epoch")
	}
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.recoveryEpochs[workspace] == 0 {
			p.recoveryEpochs[workspace] = initialRecoveryEpoch
		}
		return p.recoveryEpochs[workspace], nil
	}
	if p.db == nil {
		return 0, fmt.Errorf("database is required for recovery epoch")
	}
	if _, err := p.db.ExecContext(ctx, `INSERT INTO pm_recovery_state(workspace_id,epoch) VALUES($1,$2) ON CONFLICT (workspace_id) DO NOTHING`, workspace, initialRecoveryEpoch); err != nil {
		return 0, fmt.Errorf("initialize recovery epoch: %w", err)
	}
	var epoch int64
	if err := p.db.QueryRowContext(ctx, `SELECT epoch FROM pm_recovery_state WHERE workspace_id=$1`, workspace).Scan(&epoch); err != nil {
		return 0, fmt.Errorf("read recovery epoch: %w", err)
	}
	if epoch < int64(initialRecoveryEpoch) {
		return 0, fmt.Errorf("stored recovery epoch is invalid")
	}
	return uint64(epoch), nil
}

// activateRecovery invalidates every capability issued before the recovered
// metadata was made current. Restored grants are placed into an explicit
// review state; a restored database must never silently resurrect a revoked
// grant. A new grant can be created after current membership and provider state
// have been checked by the operator.
func (p *PasswordManager) activateRecovery(ctx context.Context, workspace, actor string, expectedEpoch uint64) (uint64, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" || strings.TrimSpace(actor) == "" {
		return 0, fmt.Errorf("workspace and actor are required for recovery activation")
	}
	if p.memory {
		p.mu.Lock()
		current := p.recoveryEpochs[workspace]
		if current == 0 {
			current = initialRecoveryEpoch
		}
		if expectedEpoch != 0 && expectedEpoch != current {
			p.mu.Unlock()
			return 0, errRecoveryEpochConflict
		}
		if current == ^uint64(0) {
			p.mu.Unlock()
			return 0, fmt.Errorf("recovery epoch exhausted")
		}
		current++
		p.recoveryEpochs[workspace] = current
		for id, session := range p.brokers {
			if session.Workspace == workspace && session.Status == "active" {
				session.Status = "recovery_invalidated"
				p.brokers[id] = session
			}
		}
		for id, session := range p.browsers {
			if session.Workspace == workspace && session.Status == "active" {
				session.Status = "recovery_invalidated"
				session.Filled = nil
				p.browsers[id] = session
			}
		}
		for id, session := range p.sshSigners {
			if session.Workspace == workspace && session.Status == "active" {
				session.Status = "recovery_invalidated"
				p.sshSigners[id] = session
			}
		}
		for id, grant := range p.grants {
			if grant.WorkspaceID == workspace && grant.Status == "active" {
				grant.Status = "recovery_review"
				p.grants[id] = grant
			}
		}
		for key, assurance := range p.assurance {
			if assurance.Workspace == workspace {
				delete(p.assurance, key)
			}
		}
		for id, export := range p.exports {
			if export.Workspace == workspace {
				delete(p.exports, id)
			}
		}
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "recovery.activate", Outcome: "success", Decision: "review_required", Detail: fmt.Sprintf("recovery epoch advanced to %d; restored grants require reauthorization", current)})
		return current, nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin recovery activation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `INSERT INTO pm_recovery_state(workspace_id,epoch) VALUES($1,$2) ON CONFLICT (workspace_id) DO NOTHING`, workspace, initialRecoveryEpoch); err != nil {
		return 0, fmt.Errorf("initialize recovery epoch: %w", err)
	}
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT epoch FROM pm_recovery_state WHERE workspace_id=$1`, workspace).Scan(&current); err != nil {
		return 0, fmt.Errorf("read recovery epoch: %w", err)
	}
	if current < int64(initialRecoveryEpoch) || current == int64(^uint64(0)>>1) {
		return 0, fmt.Errorf("stored recovery epoch is invalid or exhausted")
	}
	if expectedEpoch != 0 && uint64(current) != expectedEpoch {
		return 0, errRecoveryEpochConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE pm_recovery_state SET epoch=epoch+1,updated_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND epoch=$2`, workspace, current)
	if err != nil {
		return 0, fmt.Errorf("advance recovery epoch: %w", err)
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return 0, errRecoveryEpochConflict
	}
	next := uint64(current + 1)
	if _, err := tx.ExecContext(ctx, `UPDATE pm_broker_sessions SET status='recovery_invalidated' WHERE workspace_id=$1 AND status='active'`, workspace); err != nil {
		return 0, fmt.Errorf("invalidate broker sessions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE pm_grants SET status='recovery_review' WHERE workspace_id=$1 AND status='active'`, workspace); err != nil {
		return 0, fmt.Errorf("quarantine restored grants: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE pm_assurance_tokens SET consumed_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND consumed_at IS NULL`, workspace); err != nil {
		return 0, fmt.Errorf("invalidate action assurance: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE pm_export_handles SET used_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND used_at IS NULL`, workspace); err != nil {
		return 0, fmt.Errorf("invalidate export capabilities: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit recovery activation: %w", err)
	}
	committed = true
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "recovery.activate", Outcome: "success", Decision: "review_required", Detail: fmt.Sprintf("recovery epoch advanced to %d; restored grants require reauthorization", next)})
	return next, nil
}

type recoveryActivationInput struct {
	ExpectedEpoch  uint64 `json:"expected_epoch"`
	AssuranceToken string `json:"assurance_token"`
}

func (h *passwordManagerHandlers) activateRecovery(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityVaultManage); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireApprover(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input recoveryActivationInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if err := h.manager.consumeAssurance(r.Context(), workspace, actor, "recovery:activate", strconv.FormatUint(input.ExpectedEpoch, 10), input.AssuranceToken); err != nil {
		writeVaultError(w, err)
		return
	}
	epoch, err := h.manager.activateRecovery(r.Context(), workspace, actor, input.ExpectedEpoch)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recovery_epoch": epoch,
		"status":         "active",
		"grants":         "review_required",
		"sessions":       "invalidated",
		"remediation":    "reconcile current membership and provider state, then issue new grants",
	})
}
