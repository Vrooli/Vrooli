package credentials

import (
	"context"
	"sort"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

type RecoverRequest struct {
	DeploymentID string
	NewTarget    Target
	BundleRef    string
	// Passphrase stays in memory and travels only on standard input.
	Passphrase string
	RequestKey string
}

// Recover is the lost-host path. It verifies the bundle covers every
// materialised binding, restores it into the replacement host's authority
// and confirms each descriptor is configured there. A locked store on the
// replacement host fails closed with credential_store_locked.
func (s *Service) Recover(ctx context.Context, req RecoverRequest) (*domain.CredentialRotation, error) {
	if s.Recovery == nil {
		return nil, newError(CodeRecoveryFailed, "no recovery client is configured for the replacement host")
	}
	if strings.TrimSpace(req.Passphrase) == "" {
		return nil, apierrors.New(apierrors.CodeNeedsInput, "the recovery passphrase is required; supply it in the request body, never on a command line")
	}
	bindings, err := s.Store.ListBindings(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	rotation := &domain.CredentialRotation{ID: id, DeploymentID: req.DeploymentID, Kind: domain.CredentialOperationRecover, State: domain.RotationPlanned, Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now}
	if err := s.createOperation(ctx, rotation); err != nil {
		return nil, err
	}
	covered, err := s.verifyRecoveryBundle(ctx, req, rotation)
	if err != nil {
		return rotation, err
	}
	if rotation, err := s.validateRecoveryCoverage(ctx, req, rotation, bindings, covered); err != nil {
		return rotation, err
	}
	if err := s.record(ctx, rotation, "verify-bundle", domain.RotationProviderPrepared, "verified", "", map[string]any{"bundle_ref": req.BundleRef, "entries": len(covered)}); err != nil {
		return rotation, err
	}
	if err := s.restoreRecoveryBundle(ctx, req, rotation); err != nil {
		return rotation, err
	}
	return s.confirmRecoveryBindings(ctx, req, rotation, bindings)
}

func (s *Service) verifyRecoveryBundle(ctx context.Context, req RecoverRequest, rotation *domain.CredentialRotation) ([]domain.CredentialDescriptor, error) {
	covered, err := s.Recovery.Verify(ctx, req.BundleRef, req.Passphrase)
	if err == nil {
		return covered, nil
	}
	if apierrors.Is(err, CodeStoreLocked) {
		rotation.Error = &domain.CredentialOperationError{Code: CodeStoreLocked, Message: "the replacement host's credential store is locked"}
		_ = s.record(ctx, rotation, "verify-bundle", domain.RotationFailed, "store_locked", "", map[string]any{"target": req.NewTarget.NodeLabel()})
		return nil, StoreLocked(req.NewTarget.NodeLabel())
	}
	_, failErr := s.fail(ctx, rotation, CodeRecoveryFailed, "the recovery bundle could not be verified: "+err.Error(), map[string]any{"bundle_ref": req.BundleRef})
	return nil, failErr
}

func (s *Service) validateRecoveryCoverage(ctx context.Context, req RecoverRequest, rotation *domain.CredentialRotation, bindings []domain.CredentialBinding, covered []domain.CredentialDescriptor) (*domain.CredentialRotation, error) {
	coveredSet := make(map[string]bool, len(covered))
	for _, descriptor := range covered {
		coveredSet[descriptor.Address()] = true
	}
	var missing []string
	for _, binding := range bindings {
		if binding.State == domain.CredentialBindingMaterialized && !coveredSet[binding.Descriptor.Address()] {
			missing = append(missing, binding.Descriptor.Address())
		}
	}
	if len(missing) == 0 {
		return rotation, nil
	}
	sort.Strings(missing)
	return s.fail(ctx, rotation, CodeRecoveryFailed, "the recovery bundle does not cover every materialised binding", map[string]any{"missing": missing, "bundle_ref": req.BundleRef})
}

func (s *Service) restoreRecoveryBundle(ctx context.Context, req RecoverRequest, rotation *domain.CredentialRotation) error {
	if err := s.Recovery.Restore(ctx, req.BundleRef, req.Passphrase); err != nil {
		if apierrors.Is(err, CodeStoreLocked) {
			rotation.Error = &domain.CredentialOperationError{Code: CodeStoreLocked, Message: "the replacement host's credential store is locked"}
			_ = s.record(ctx, rotation, "restore", domain.RotationFailed, "store_locked", "", map[string]any{"target": req.NewTarget.NodeLabel()})
			return StoreLocked(req.NewTarget.NodeLabel())
		}
		_, failErr := s.fail(ctx, rotation, CodeRecoveryFailed, "restore into the replacement host failed: "+err.Error(), map[string]any{"target": req.NewTarget.NodeLabel()})
		return failErr
	}
	return nil
}

func (s *Service) confirmRecoveryBindings(ctx context.Context, req RecoverRequest, rotation *domain.CredentialRotation, bindings []domain.CredentialBinding) (*domain.CredentialRotation, error) {
	if err := s.record(ctx, rotation, "restore", domain.RotationConsumersUpdated, "restored", "", map[string]any{"target": req.NewTarget.NodeLabel()}); err != nil {
		return rotation, err
	}
	var unconfirmed []string
	for i := range bindings {
		b := &bindings[i]
		missing, err := s.confirmRecoveryBinding(ctx, req.NewTarget, rotation, b)
		if err != nil {
			return rotation, err
		}
		if missing {
			unconfirmed = append(unconfirmed, b.Descriptor.Address())
		}
	}
	if len(unconfirmed) > 0 {
		sort.Strings(unconfirmed)
		rotation.Unreached = unconfirmed
		if err := s.record(ctx, rotation, "confirm", domain.RotationConsumersUpdated, "incomplete", "", map[string]any{"unconfirmed": unconfirmed}); err != nil {
			return rotation, err
		}
		return rotation, newError(CodeRecoveryFailed, "restore completed but some bindings are not confirmed on the replacement host").WithDetail("unconfirmed", unconfirmed).WithDetail("rotation_id", rotation.ID)
	}
	if err := s.record(ctx, rotation, "confirm", domain.RotationVerified, "verified", "", map[string]any{"target": req.NewTarget.NodeLabel()}); err != nil {
		return rotation, err
	}
	if err := s.record(ctx, rotation, "complete", domain.RotationComplete, "complete", "", map[string]any{"target": req.NewTarget.NodeLabel(), "bindings": len(rotation.Consumers)}); err != nil {
		return rotation, err
	}
	return rotation, nil
}

func (s *Service) confirmRecoveryBinding(ctx context.Context, target Target, rotation *domain.CredentialRotation, binding *domain.CredentialBinding) (bool, error) {
	if binding.State != domain.CredentialBindingMaterialized {
		return false, nil
	}
	probe, err := s.Distributor.Probe(ctx, target, *binding)
	if err != nil {
		if apierrors.Is(err, CodeStoreLocked) {
			return false, err
		}
		return true, nil
	}
	if !probe.Configured {
		return true, nil
	}
	binding.GrantRef = ""
	binding.UpdatedAt = s.now()
	if err := s.Store.UpsertBinding(ctx, binding); err != nil {
		return false, err
	}
	rotation.Consumers = append(rotation.Consumers, domain.CredentialConsumerProgress{Consumer: binding.ID, State: domain.ConsumerAcknowledged, Version: binding.Version.Number, UpdatedAt: s.now()})
	return false, nil
}
