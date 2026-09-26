package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func freshRecoveryKey(sourceID uuid.UUID) string {
	return "resume-from-failed:" + sourceID.String()
}

func freshRecoveryRequestHash(req ResumeFromFailedRunRequest) string {
	// JSON includes the source, exact custom context and ordered attachment IDs.
	// Empty/nil attachments have the same wire representation (omitempty).
	data, _ := json.Marshal(req)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// FreshRecoveryAccepted reads durable acceptance for the supervisor's exact
// attachment-free request. It does not reserve, dispatch, repair or infer
// acceptance from a missing/expired cache entry. nil, nil is missing evidence,
// never proof that retrying the original request is safe.
func (o *Orchestrator) FreshRecoveryAccepted(ctx context.Context, sourceID uuid.UUID, customContext string) (*domain.Run, error) {
	run, _, err := o.freshRecoveryAccepted(ctx, ResumeFromFailedRunRequest{RunID: sourceID, CustomContext: customContext})
	return run, err
}

func (o *Orchestrator) freshRecoveryAccepted(ctx context.Context, req ResumeFromFailedRunRequest) (*domain.Run, bool, error) {
	claims, ok := o.runs.(repository.RunFreshRecoveryClaimer)
	if !ok || req.RunID == uuid.Nil {
		return nil, false, domain.NewStateError("Run", "unknown", "read fresh recovery receipt", "durable source claim owner or source identity is unavailable")
	}
	hash, err := claims.GetFreshRecoveryClaim(ctx, req.RunID)
	if err != nil {
		return nil, false, err
	}
	claimed := hash != ""
	if claimed && hash != freshRecoveryRequestHash(req) {
		return nil, true, domain.RefuseBeforeEffects(domain.NewValidationErrorWithCode("freshRecovery", "source was consumed by a different recovery request", domain.ErrCodeValidationConflict))
	}
	key := freshRecoveryKey(req.RunID)
	creation, err := o.runs.GetCreationReceipt(ctx, key)
	if err != nil {
		return nil, claimed, err
	}
	recovered, err := o.runs.GetByIdempotencyKey(ctx, key)
	if err != nil {
		return nil, claimed, err
	}
	if recovered == nil {
		return nil, claimed, nil
	}
	if !claimed || creation == nil || creation.IdempotencyKey != key || creation.RunID != recovered.ID || creation.TaskID != recovered.TaskID || creation.OwnerSubject != recovered.OwnerSubject || recovered.ID == req.RunID || recovered.IdempotencyKey != key || len(recovered.SourceRunIDs) != 1 || recovered.SourceRunIDs[0] != req.RunID {
		return nil, true, domain.NewStateError("Run", "unknown", "read fresh recovery receipt", "replacement does not have an exact source/request acceptance proof")
	}
	source, err := o.runs.Get(ctx, req.RunID)
	if err != nil {
		return nil, true, err
	}
	if source == nil || source.TaskID != creation.TaskID || source.OwnerSubject != creation.OwnerSubject {
		return nil, true, domain.NewStateError("Run", "unknown", "read fresh recovery receipt", "replacement creation receipt does not preserve the source task/owner identity")
	}
	if (source.DispatchBinding == nil) != (recovered.DispatchBinding == nil) || (source.DispatchBinding != nil && *source.DispatchBinding != *recovered.DispatchBinding) {
		return nil, true, domain.NewStateError("Run", "unknown", "read fresh recovery receipt", "replacement does not preserve the source dispatch binding; owner reconciliation required")
	}
	return recovered, true, nil
}
