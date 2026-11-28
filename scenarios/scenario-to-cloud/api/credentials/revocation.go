package credentials

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

type RevokeBindingRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	RequestKey   string
}

// Revoke purges the binding's current version from the target and denies
// further distribution. An unreachable target leaves the operation in
// revocation_incomplete with the node retained until confirmed (P13-A05).
func (s *Service) Revoke(ctx context.Context, req RevokeBindingRequest) (*domain.CredentialRotation, error) {
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding").WithDetail("binding_id", req.BindingID)
	}
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return nil, newError(CodeRotationRefused, "machine enrollment credentials are revoked through vrooli-bridge").WithDetail("binding_id", binding.ID)
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	if existing, err := s.Store.GetRotation(ctx, id); err != nil {
		return nil, err
	} else if existing != nil {
		return s.revokeAdvance(ctx, existing, binding, req.Target)
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationRevoke, FromVersion: binding.Version.Number, ToVersion: 0, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.createOperation(ctx, rotation); err != nil {
		return nil, err
	}
	return s.revokeAdvance(ctx, rotation, binding, req.Target)
}

func (s *Service) revokeAdvance(ctx context.Context, rotation *domain.CredentialRotation, binding *domain.CredentialBinding, target Target) (*domain.CredentialRotation, error) {
	if rotation.State.Terminal() {
		return rotation, nil
	}
	step := "revoke-" + sanitizeStep(binding.Descriptor.Field) + "-v" + fmt.Sprint(binding.Version.Number)
	receipt, err := s.Distributor.Revoke(ctx, RevokeRequest{Target: target, Binding: *binding, Version: binding.Version.Number, OperationID: rotation.ID, Step: step})
	if err != nil {
		if IsUnreachable(err) {
			rotation.Unreached = []string{target.NodeLabel()}
			if err := s.record(ctx, rotation, "revoke", domain.RotationRevocationIncomplete, "incomplete", "", map[string]any{"version": binding.Version.Number, "unreached": rotation.Unreached}, RevocationLimitation, "the unreached node is retained until its purge receipt is confirmed"); err != nil {
				return rotation, err
			}
			return rotation, newError(CodeRevocationIncomplete, "the target could not be reached; revocation is incomplete and the node is retained until confirmed").
				WithDetail("rotation_id", rotation.ID).WithDetail("unreached", rotation.Unreached).
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "endpoint", Reference: "/api/v1/deployments/{id}/credentials/rotations/" + rotation.ID + "/resume", Label: "Retry once the target is reachable"})
		}
		return s.fail(ctx, rotation, CodeDistributionFailed, "revocation failed on the target: "+err.Error(), map[string]any{"version": binding.Version.Number})
	}
	rotation.Unreached = nil
	now := s.now()
	for i := range rotation.Consumers {
		rotation.Consumers[i].State = domain.ConsumerUpdated
		rotation.Consumers[i].UpdatedAt = now
	}
	binding.State = domain.CredentialBindingRevoked
	binding.UpdatedAt = now
	if err := s.Store.UpsertBinding(ctx, binding); err != nil {
		return rotation, err
	}
	details := map[string]any{"version": binding.Version.Number, "transport": receipt.Transport}
	for k, v := range receipt.Details {
		details[k] = v
	}
	if err := s.record(ctx, rotation, "revoke", domain.RotationOldVersionRevoked, "revoked", receipt.Ref, details, receipt.Limitations...); err != nil {
		return rotation, err
	}
	if err := s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"binding_state": string(binding.State)}); err != nil {
		return rotation, err
	}
	return rotation, nil
}

// RecoverRequest re-provisions a deployment's bindings onto a replacement
// host from an encrypted recovery bundle already present on that host.
