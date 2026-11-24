package credentials

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

type RotateRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	// Value is the operator-supplied replacement for external classes; it is
	// kept in memory and never persisted. Empty means mint (generated classes).
	Value      string
	RequestKey string
}

// ResumeInput continues an operation that stopped on an operator step,
// an overlap window or missing acknowledgements.
type ResumeInput struct {
	OperatorConfirmed bool
}

// Rotate runs the state machine as far as it can in one call and returns the
// durable operation. A non-terminal return is truthful: the state names what
// is still outstanding.
func (s *Service) Rotate(ctx context.Context, req RotateRequest) (*domain.CredentialRotation, error) {
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding on this deployment").WithDetail("binding_id", req.BindingID)
	}
	if err := validateRotationBinding(binding); err != nil {
		return nil, err
	}
	if _, err := s.provider(binding.Class); err != nil {
		return nil, err
	}
	if active, err := s.activeOperation(ctx, binding); err != nil {
		return nil, err
	} else if active, err := reuseOrRejectRotation(active, req.RequestKey); err != nil || active != nil {
		return active, err
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationRotate, FromVersion: binding.Version.Number, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.createOperation(ctx, rotation); err != nil {
		return nil, err
	}
	return s.advance(ctx, rotation, binding, req.Target, req.Value, ResumeInput{})
}

func (s *Service) createOperation(ctx context.Context, rotation *domain.CredentialRotation) error {
	if err := s.Store.CreateRotation(ctx, rotation); err != nil {
		if errors.Is(err, domain.ErrCredentialOperationConflict) {
			return newError(CodeRotationConflict, "another lifecycle operation is still open on this binding")
		}
		return err
	}
	return nil
}

func validateRotationBinding(binding *domain.CredentialBinding) error {
	if binding.State == domain.CredentialBindingRevoked {
		return newError(CodeRotationRefused, "the binding is revoked; materialise a new version through deploy instead").WithDetail("binding_id", binding.ID)
	}
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return newError(CodeRotationRefused, "machine enrollment credentials are owned by vrooli-bridge; rotate or revoke the node identity there").
			WithDetail("binding_id", binding.ID).
			WithNextAction(apierrors.NextAction{Owner: "vrooli-bridge", Kind: "command", Reference: "vrooli-bridge nodes", Label: "Rotate the node enrollment through Bridge"})
	}
	return nil
}

func reuseOrRejectRotation(active *domain.CredentialRotation, requestKey string) (*domain.CredentialRotation, error) {
	if active == nil {
		return nil, nil
	}
	if requestKey != "" && active.ID == requestKey {
		return active, nil
	}
	return nil, newError(CodeRotationConflict, "another lifecycle operation is still open on this binding").WithDetail("rotation_id", active.ID).WithDetail("state", string(active.State))
}

// Resume continues a non-terminal operation.
func (s *Service) Resume(ctx context.Context, target Target, rotationID string, input ResumeInput) (*domain.CredentialRotation, error) {
	rotation, err := s.Store.GetRotation(ctx, rotationID)
	if err != nil {
		return nil, err
	}
	if rotation == nil {
		return nil, newError(CodeRotationNotFound, "no such credential operation").WithDetail("rotation_id", rotationID)
	}
	if rotation.State.Terminal() {
		return rotation, nil
	}
	binding, err := s.Store.GetBinding(ctx, rotation.DeploymentID, rotation.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "the operation's binding no longer exists").WithDetail("binding_id", rotation.BindingID)
	}
	switch rotation.Kind {
	case domain.CredentialOperationRevoke:
		return s.revokeAdvance(ctx, rotation, binding, target)
	case domain.CredentialOperationRecover:
		return rotation, nil
	}
	return s.advance(ctx, rotation, binding, target, "", input)
}

func (s *Service) activeOperation(ctx context.Context, binding *domain.CredentialBinding) (*domain.CredentialRotation, error) {
	rotations, err := s.Store.ListRotations(ctx, binding.DeploymentID)
	if err != nil {
		return nil, err
	}
	for i := range rotations {
		r := rotations[i]
		if r.BindingID == binding.ID && !r.State.Terminal() {
			return &r, nil
		}
	}
	return nil, nil
}

func initialConsumers(binding *domain.CredentialBinding, now time.Time) []domain.CredentialConsumerProgress {
	out := make([]domain.CredentialConsumerProgress, 0, len(binding.ConsumerRefs))
	for _, consumer := range binding.ConsumerRefs {
		out = append(out, domain.CredentialConsumerProgress{Consumer: consumer, State: domain.ConsumerPending, UpdatedAt: now})
	}
	return out
}

func (s *Service) record(ctx context.Context, rotation *domain.CredentialRotation, step string, state domain.CredentialRotationState, outcome string, ref string, details map[string]any, limitations ...string) error {
	now := s.now()
	if err := rotation.TransitionTo(state); err != nil {
		return err
	}
	rotation.UpdatedAt = now
	rotation.Receipts = append(rotation.Receipts, domain.CredentialReceipt{Step: step, State: string(state), Outcome: outcome, At: now, TargetReceiptRef: ref, Details: details, Limitations: limitations})
	if state.Terminal() {
		rotation.CompletedAt = &now
	}
	s.logf("credential operation=%s binding=%s step=%s state=%s outcome=%s", rotation.ID, rotation.BindingID, step, state, outcome)
	return s.Store.SaveRotation(ctx, rotation)
}

func (s *Service) fail(ctx context.Context, rotation *domain.CredentialRotation, code, message string, details map[string]any) (*domain.CredentialRotation, error) {
	rotation.Error = &domain.CredentialOperationError{Code: code, Message: message}
	if err := s.record(ctx, rotation, "fail", domain.RotationFailed, "failed", "", details); err != nil {
		return rotation, err
	}
	return rotation, newError(code, message).WithDetail("rotation_id", rotation.ID)
}

// advance drives the rotation state machine. Each transition is persisted
// before the next side effect so a crash leaves an accurate ledger.
func (s *Service) advance(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, value string, input ResumeInput) (*domain.CredentialRotation, error) {
	provider, err := s.provider(binding.Class)
	if err != nil {
		return rotation, err
	}
	version := versionForRotation(rotation, binding)
	for {
		done, err := s.advanceStep(ctx, rotation, binding, target, &value, &version, input, provider)
		if done || err != nil {
			return rotation, err
		}
	}
}

func versionForRotation(rotation *domain.CredentialRotation, binding *domain.CredentialBinding) domain.CredentialVersion {
	if rotation.ToVersion == 0 {
		return domain.CredentialVersion{}
	}
	version := domain.CredentialVersion{Number: rotation.ToVersion}
	if binding.Version.Number == rotation.ToVersion {
		return binding.Version
	}
	if detail := versionFromReceipts(rotation); detail != nil {
		return *detail
	}
	return version
}

func (s *Service) advanceStep(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, value *string, version *domain.CredentialVersion, input ResumeInput, provider Provider) (bool, error) {
	switch rotation.State {
	case domain.RotationPlanned:
		return false, s.advancePlanned(ctx, rotation, binding, value, version)
	case domain.RotationNewVersionCreated:
		return s.advanceNewVersionCreated(ctx, rotation, binding, target, *value, version, provider)
	case domain.RotationProviderPrepared:
		return s.advanceProviderPrepared(ctx, rotation, binding, target, *value, *version)
	case domain.RotationConsumersUpdated:
		return s.advanceConsumersUpdated(ctx, rotation, binding, target, *version, provider)
	case domain.RotationVerified, domain.RotationPendingOperatorInput:
		return s.advanceVerified(ctx, rotation, binding, target, input, provider)
	case domain.RotationOldVersionRevoked:
		return true, s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"active_version": binding.Version.Number})
	case domain.RotationRecovering:
		_, err := s.recover(ctx, rotation, binding, target, *version, "resumed while recovering")
		return true, err
	default:
		return true, nil
	}
}

func (s *Service) advancePlanned(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, value *string, version *domain.CredentialVersion) error {
	if *value == "" {
		if binding.SourceClass != domain.SecretClassPerInstallGenerated && binding.Class == domain.CredentialClassExternalAPICredential {
			_, err := s.fail(ctx, rotation, CodeRotationRefused, "an external credential rotation needs the replacement value supplied in memory; none was given", nil)
			return err
		}
		generated, err := s.generate(*binding)
		if err != nil {
			_, failErr := s.fail(ctx, rotation, CodeRotationRefused, "mint replacement value: "+err.Error(), nil)
			return failErr
		}
		*value = generated
	}
	next, err := s.nextVersion(ctx, *binding)
	if err != nil {
		return err
	}
	*version = domain.CredentialVersion{Number: next, ContentRef: newContentRef(), CreatedAt: s.now()}
	rotation.ToVersion = version.Number
	return s.record(ctx, rotation, "new-version", domain.RotationNewVersionCreated, "created", "", map[string]any{"version": version.Number, "content_ref": version.ContentRef, "created_at": version.CreatedAt})
}

func (s *Service) advanceNewVersionCreated(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, value string, version *domain.CredentialVersion, provider Provider) (bool, error) {
	if value == "" {
		_, err := s.fail(ctx, rotation, CodeRotationRefused, "the replacement value is no longer in memory; start a new rotation", map[string]any{"version": rotation.ToVersion})
		return true, err
	}
	prepared, err := provider.Prepare(ctx, PrepareRequest{Target: target, Binding: *binding, Version: *version, Value: value, OperationID: rotation.ID})
	if err != nil {
		return true, s.failPrepare(ctx, rotation, err)
	}
	details := prepared.Details
	if details == nil {
		details = map[string]any{}
	}
	details["dual_accept"] = prepared.DualAccept
	if prepared.ExpiresAt != nil {
		expires := prepared.ExpiresAt.UTC()
		version.ExpiresAt = &expires
		details["expires_at"] = expires
	}
	return false, s.record(ctx, rotation, "provider-prepare", domain.RotationProviderPrepared, "prepared", "", details)
}

func (s *Service) failPrepare(ctx context.Context, rotation *domain.CredentialRotation, err error) error {
	var refusal *RotationRefusal
	if errors.As(err, &refusal) {
		_, failErr := s.fail(ctx, rotation, CodeRotationRefused, refusal.Reason, map[string]any{"owner": refusal.Owner})
		return failErr
	}
	// Provider rejection leaves the predecessor untouched (SECRET-03).
	_, failErr := s.fail(ctx, rotation, CodeDistributionFailed, "provider rejected the new version: "+err.Error(), map[string]any{"predecessor_retained": true, "version": rotation.FromVersion})
	return failErr
}

func (s *Service) advanceProviderPrepared(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, value string, version domain.CredentialVersion) (bool, error) {
	if value == "" {
		_, err := s.recover(ctx, rotation, binding, target, version, "the replacement value is no longer in memory after provider preparation")
		return true, err
	}
	step := "rotate-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(version.Number)
	receipt, err := s.Distributor.Deliver(ctx, DeliverRequest{Target: target, Binding: *binding, Version: version, Value: value, OperationID: rotation.ID, Step: step})
	if err != nil {
		if apierrors.Is(err, apierrors.CodeForbiddenRevoked) {
			rotation.Error = &domain.CredentialOperationError{Code: apierrors.CodeForbiddenRevoked, Message: "distribution denied: the grant is revoked"}
			_ = s.record(ctx, rotation, "consumer-update", domain.RotationFailed, "denied", "", map[string]any{"predecessor_retained": true})
			return true, err
		}
		_, recoverErr := s.recover(ctx, rotation, binding, target, version, "distribution failed: "+err.Error())
		return true, recoverErr
	}
	binding.GrantRef = grantRefFrom(receipt, binding.GrantRef)
	markConsumersUpdated(rotation, version.Number, s.now())
	restart, err := s.restartConsumers(ctx, target, binding)
	if err != nil {
		_, recoverErr := s.recover(ctx, rotation, binding, target, version, "consumer restart failed: "+err.Error())
		return true, recoverErr
	}
	details := map[string]any{"version": version.Number, "transport": receipt.Transport, "consumers": binding.ConsumerRefs, "restart": restart}
	return false, s.record(ctx, rotation, "consumer-update", domain.RotationConsumersUpdated, "updated", receipt.Ref, details)
}

func markConsumersUpdated(rotation *domain.CredentialRotation, version int64, now time.Time) {
	for i := range rotation.Consumers {
		rotation.Consumers[i].State = domain.ConsumerUpdated
		rotation.Consumers[i].Version = version
		rotation.Consumers[i].UpdatedAt = now
	}
}

func (s *Service) restartConsumers(ctx context.Context, target Target, binding *domain.CredentialBinding) (string, error) {
	if s.Restarter == nil || len(binding.ConsumerRefs) == 0 {
		return "not_requested", nil
	}
	if err := s.Restarter.Restart(ctx, target, binding.ConsumerRefs); err != nil {
		return "", err
	}
	return "restarted", nil
}

func (s *Service) advanceConsumersUpdated(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, version domain.CredentialVersion, provider Provider) (bool, error) {
	if err := s.collectAcks(ctx, rotation, binding, target, version.Number); err != nil {
		return true, err
	}
	acks, err := s.Store.ListAcks(ctx, binding.ID)
	if err != nil {
		return true, err
	}
	verified, err := provider.Verify(ctx, VerifyRequest{Target: target, Binding: *binding, Version: version, Consumers: binding.ConsumerRefs, Acks: acks})
	if err != nil {
		_, recoverErr := s.recover(ctx, rotation, binding, target, version, "verification failed: "+err.Error())
		return true, recoverErr
	}
	markConsumersVerification(rotation, verified, s.now())
	if len(verified.Unverified) > 0 {
		rotation.Unreached = sortedKeys(verified.Unverified)
		return true, s.record(ctx, rotation, "verify", domain.RotationConsumersUpdated, "incomplete", "", map[string]any{"version": version.Number, "unreached": rotation.Unreached, "verified": verified.Verified}, "rotation is incomplete until every listed consumer acknowledges version "+fmt.Sprint(version.Number))
	}
	return false, s.promoteVerifiedVersion(ctx, rotation, binding, version, verified)
}

func markConsumersVerification(rotation *domain.CredentialRotation, verified VerifyResult, now time.Time) {
	for i := range rotation.Consumers {
		consumer := &rotation.Consumers[i]
		if reason, missing := verified.Unverified[consumer.Consumer]; missing {
			consumer.State = domain.ConsumerUnreachable
			consumer.Reason = reason
		} else {
			consumer.State = domain.ConsumerAcknowledged
			consumer.Reason = ""
		}
		consumer.UpdatedAt = now
	}
}

func (s *Service) promoteVerifiedVersion(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, version domain.CredentialVersion, verified VerifyResult) error {
	rotation.Unreached = nil
	previous := binding.Version
	binding.PreviousVersion = &previous
	binding.Version = version
	binding.State = domain.CredentialBindingMaterialized
	binding.UpdatedAt = s.now()
	if err := s.Store.UpsertBinding(ctx, binding); err != nil {
		return err
	}
	details := verified.Details
	if details == nil {
		details = map[string]any{}
	}
	details["version"] = version.Number
	details["verified"] = verified.Verified
	return s.record(ctx, rotation, "verify", domain.RotationVerified, "verified", "", details)
}

func (s *Service) advanceVerified(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, input ResumeInput, provider Provider) (bool, error) {
	if done, err := s.waitForVerifiedResume(rotation, input); done || err != nil {
		return true, err
	}
	result, predecessor, err := s.revokeProviderPredecessor(ctx, rotation, binding, target, input, provider)
	if err != nil || providerRevokePending(result) {
		return true, err
	}
	return false, s.finishVerifiedRotation(ctx, rotation, binding, target, predecessor, result)
}

func (s *Service) waitForVerifiedResume(rotation *domain.CredentialRotation, input ResumeInput) (bool, error) {
	if rotation.State == domain.RotationPendingOperatorInput && !input.OperatorConfirmed {
		return true, newError(CodePendingOperatorInput, "the rotation is waiting for the operator to revoke the predecessor at the provider").WithDetail("rotation_id", rotation.ID).WithDetail("handoff", rotation.PendingOperatorInput)
	}
	if rotation.ResumeAfter != nil && s.now().Before(*rotation.ResumeAfter) && !input.OperatorConfirmed {
		return true, nil
	}
	return false, nil
}

func providerRevokePending(result RevokePredecessorResult) bool {
	return result.Handoff != nil || result.ResumeAfter != nil && !result.Done
}

func (s *Service) finishVerifiedRotation(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, predecessor domain.CredentialVersion, result RevokePredecessorResult) error {
	receipt, err := s.revokeTargetPredecessor(ctx, rotation, binding, target, predecessor)
	if err != nil {
		return err
	}
	binding.PreviousVersion = nil
	binding.UpdatedAt = s.now()
	if err := s.Store.UpsertBinding(ctx, binding); err != nil {
		return err
	}
	details := result.Details
	if details == nil {
		details = map[string]any{}
	}
	details["predecessor"] = predecessor.Number
	return s.record(ctx, rotation, "revoke-predecessor", domain.RotationOldVersionRevoked, "revoked", receipt.Ref, details, RevocationLimitation)
}

func (s *Service) revokeProviderPredecessor(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, input ResumeInput, provider Provider) (RevokePredecessorResult, domain.CredentialVersion, error) {
	predecessor := domain.CredentialVersion{Number: rotation.FromVersion}
	if binding.PreviousVersion != nil {
		predecessor = *binding.PreviousVersion
	}
	if rotation.State == domain.RotationPendingOperatorInput {
		result := RevokePredecessorResult{Done: true, Details: map[string]any{"operator_confirmed": true, "handoff": rotation.PendingOperatorInput.Reference}}
		result, _, err := s.recordPredecessorHandoff(ctx, rotation, result, predecessor)
		return result, predecessor, err
	}
	result, err := provider.RevokePredecessor(ctx, RevokePredecessorRequest{Target: target, Binding: *binding, Predecessor: predecessor, Successor: binding.Version, OperationID: rotation.ID})
	if err != nil {
		_, failErr := s.fail(ctx, rotation, CodeDistributionFailed, "predecessor revocation failed at the provider: "+err.Error(), map[string]any{"new_version_active": binding.Version.Number})
		return RevokePredecessorResult{}, predecessor, failErr
	}
	result, _, err = s.recordPredecessorHandoff(ctx, rotation, result, predecessor)
	return result, predecessor, err
}

func (s *Service) recordPredecessorHandoff(ctx context.Context, rotation *domain.CredentialRotation, result RevokePredecessorResult, predecessor domain.CredentialVersion) (RevokePredecessorResult, domain.CredentialVersion, error) {
	if result.Handoff != nil {
		rotation.PendingOperatorInput = result.Handoff
		rotation.PendingOperatorInput.RequestedAt = s.now()
		err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationPendingOperatorInput, "pending_operator_input", "", map[string]any{"handoff": result.Handoff.Reference, "provider": result.Handoff.Provider})
		return result, predecessor, err
	}
	if result.ResumeAfter != nil && !result.Done {
		rotation.ResumeAfter = result.ResumeAfter
		err := s.record(ctx, rotation, "revoke-predecessor", domain.RotationVerified, "deferred", "", map[string]any{"resume_after": result.ResumeAfter, "details": result.Details})
		return result, predecessor, err
	}
	rotation.PendingOperatorInput = nil
	rotation.ResumeAfter = nil
	return result, predecessor, nil
}

func (s *Service) revokeTargetPredecessor(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, predecessor domain.CredentialVersion) (Receipt, error) {
	if predecessor.Number == 0 {
		return Receipt{}, nil
	}
	step := "revoke-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(predecessor.Number)
	receipt, err := s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: predecessor.Number, OperationID: rotation.ID, Step: step})
	if err == nil {
		return receipt, nil
	}
	if IsUnreachable(err) {
		rotation.Unreached = []string{target.NodeLabel()}
		recordErr := s.record(ctx, rotation, "revoke-predecessor", domain.RotationVerified, "incomplete", "", map[string]any{"predecessor": predecessor.Number, "unreached": rotation.Unreached}, RevocationLimitation)
		if recordErr != nil {
			return Receipt{}, recordErr
		}
		return Receipt{}, newError(CodeRevocationIncomplete, "the new version is active but the predecessor could not be purged from an unreachable target").WithDetail("rotation_id", rotation.ID).WithDetail("unreached", rotation.Unreached)
	}
	_, failErr := s.fail(ctx, rotation, CodeDistributionFailed, "predecessor purge failed on the target: "+err.Error(), map[string]any{"new_version_active": binding.Version.Number})
	return Receipt{}, failErr
}

// recover retires whatever the failed attempt distributed, keeps the
// predecessor as the active version and terminates in Failed.
func (s *Service) recover(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target, version domain.CredentialVersion, reason string) (*domain.CredentialRotation, error) {
	if err := s.record(ctx, rotation, "recover", domain.RotationRecovering, "recovering", "", map[string]any{"reason": reason, "predecessor_retained": true, "active_version": binding.Version.Number}); err != nil {
		return rotation, err
	}
	details := map[string]any{"predecessor_retained": true, "active_version": binding.Version.Number}
	if version.Number > 0 {
		step := "recover-purge-v" + fmt.Sprint(version.Number)
		if receipt, err := s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: version.Number, OperationID: rotation.ID, Step: step}); err != nil {
			details["purge_new_version"] = "unconfirmed: " + err.Error()
			if IsUnreachable(err) {
				rotation.Unreached = []string{target.NodeLabel()}
			}
		} else {
			details["purge_new_version"] = "purged"
			details["purge_receipt"] = receipt.Ref
		}
	}
	rotation.Error = &domain.CredentialOperationError{Code: CodeDistributionFailed, Message: reason}
	if err := s.record(ctx, rotation, "fail", domain.RotationFailed, "failed", "", details); err != nil {
		return rotation, err
	}
	return rotation, newError(CodeDistributionFailed, reason).WithDetail("rotation_id", rotation.ID).WithDetail("predecessor_retained", true)
}

// RevokeBindingRequest revokes the current version of a binding.
