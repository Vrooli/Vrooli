package credentials

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

type BreakGlassRequest struct {
	Target       Target
	DeploymentID string
	BindingID    string
	Scope        string
	Window       time.Duration
	Confirmation string
	Operator     string
	RequestKey   string
}

// MaxBreakGlassWindow bounds emergency access.
const MaxBreakGlassWindow = 4 * time.Hour

// ExpectedBreakGlassConfirmation is the exact string an operator must send.
func ExpectedBreakGlassConfirmation(bindingID string) string {
	return "BREAK-GLASS " + bindingID
}

// BreakGlass issues an emergency version through the normal preparation and
// distribution stages, records the window, and leaves predecessor revocation
// to SweepBreakGlass once the window closes (SECRET-08).
func (s *Service) BreakGlass(ctx context.Context, req BreakGlassRequest) (*domain.CredentialRotation, error) {
	if err := validateBreakGlassRequest(req); err != nil {
		return nil, err
	}
	binding, err := s.Store.GetBinding(ctx, req.DeploymentID, req.BindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, newError(CodeBindingNotFound, "no such credential binding").WithDetail("binding_id", req.BindingID)
	}
	if err := s.validateBreakGlassBinding(binding); err != nil {
		return nil, err
	}
	if active, err := s.activeOperation(ctx, binding); err != nil {
		return nil, err
	} else if err := rejectActiveBreakGlass(active); err != nil {
		return nil, err
	}
	rotation, err := s.startBreakGlass(ctx, req, binding)
	if err != nil {
		return rotation, err
	}
	// Run the ordinary stages; the emergency version becomes active only once
	// consumers verified it. Predecessor revocation waits for the window.
	rotation, err = s.advance(ctx, rotation, binding, req.Target, "", ResumeInput{})
	if err != nil && !apierrors.Is(err, CodePendingOperatorInput) {
		return rotation, err
	}
	return rotation, nil
}

func (s *Service) startBreakGlass(ctx context.Context, req BreakGlassRequest, binding *domain.CredentialBinding) (*domain.CredentialRotation, error) {
	id := req.RequestKey
	if strings.TrimSpace(id) == "" {
		id = s.newID()
	}
	now := s.now()
	predecessor := binding.Version
	rotation := &domain.CredentialRotation{
		ID: id, DeploymentID: binding.DeploymentID, BindingID: binding.ID, Kind: domain.CredentialOperationBreakGlass, FromVersion: binding.Version.Number, State: domain.RotationPlanned, Consumers: initialConsumers(binding, now), Receipts: []domain.CredentialReceipt{}, CreatedAt: now, UpdatedAt: now,
		BreakGlass: &domain.BreakGlassWindow{Scope: req.Scope, Operator: req.Operator, IssuedAt: now, ExpiresAt: now.Add(req.Window), PredecessorRef: fmt.Sprintf("%s@%d", binding.ID, predecessor.Number), ConfirmationRef: "confirmed:" + ExpectedBreakGlassConfirmation(req.BindingID)},
	}
	if err := s.createOperation(ctx, rotation); err != nil {
		return nil, err
	}
	if err := s.record(ctx, rotation, "break-glass-audit", domain.RotationPlanned, "confirmed", "", map[string]any{"scope": req.Scope, "operator": req.Operator, "expires_at": rotation.BreakGlass.ExpiresAt, "predecessor": predecessor.Number}); err != nil {
		return rotation, err
	}
	return rotation, nil
}

func validateBreakGlassRequest(req BreakGlassRequest) error {
	if req.Confirmation != ExpectedBreakGlassConfirmation(req.BindingID) {
		return newError(CodeConfirmationRequired, "break-glass requires the exact confirmation string").
			WithDetail("expected_form", "BREAK-GLASS <binding-id>")
	}
	if strings.TrimSpace(req.Scope) == "" || strings.TrimSpace(req.Operator) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "break-glass needs a scope and an operator identity")
	}
	if req.Window <= 0 || req.Window > MaxBreakGlassWindow {
		return apierrors.Newf(apierrors.CodeInvalidRequest, "break-glass window must be between 1s and %s", MaxBreakGlassWindow)
	}
	return nil
}

func (s *Service) validateBreakGlassBinding(binding *domain.CredentialBinding) error {
	if binding.Class == domain.CredentialClassMachineEnrollment {
		return newError(CodeRotationRefused, "machine enrollment credentials have no break-glass path outside vrooli-bridge")
	}
	return nil
}

func rejectActiveBreakGlass(active *domain.CredentialRotation) error {
	if active == nil {
		return nil
	}
	return newError(CodeRotationConflict, "another lifecycle operation is still open on this binding").WithDetail("rotation_id", active.ID)
}

// SweepBreakGlass closes expired windows: every emergency version past its
// window is rotated away and revoked, so revealed material stops working
// without operator action. Returns the follow-on operations.
func (s *Service) SweepBreakGlass(ctx context.Context, deploymentID string, target Target) ([]*domain.CredentialRotation, error) {
	rotations, err := s.Store.ListRotations(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var followOns []*domain.CredentialRotation
	for i := range rotations {
		r := rotations[i]
		if !breakGlassWindowExpired(r, now) {
			continue
		}
		follow, err := s.expireBreakGlass(ctx, r, deploymentID, target, now)
		if err != nil {
			if follow != nil {
				followOns = append(followOns, follow)
			}
			return followOns, err
		}
		followOns = append(followOns, follow)
	}
	return followOns, nil
}

func breakGlassWindowExpired(rotation domain.CredentialRotation, now time.Time) bool {
	return rotation.Kind == domain.CredentialOperationBreakGlass && rotation.BreakGlass != nil && rotation.BreakGlass.AutoRevokedAt == nil && !now.Before(rotation.BreakGlass.ExpiresAt)
}

func (s *Service) expireBreakGlass(ctx context.Context, rotation domain.CredentialRotation, deploymentID string, target Target, now time.Time) (*domain.CredentialRotation, error) {
	if !rotation.State.Terminal() {
		if _, err := s.Resume(ctx, target, rotation.ID, ResumeInput{OperatorConfirmed: true}); err != nil && !apierrors.Is(err, CodeRevocationIncomplete) {
			return nil, err
		}
	}
	follow, err := s.Rotate(ctx, RotateRequest{Target: target, DeploymentID: deploymentID, BindingID: rotation.BindingID, RequestKey: "break-glass-expiry-" + rotation.ID})
	if err != nil {
		return nil, err
	}
	stored, err := s.Store.GetRotation(ctx, rotation.ID)
	if err != nil {
		return follow, err
	}
	if stored == nil {
		return follow, nil
	}
	stored.BreakGlass.AutoRevokedAt = &now
	stored.Receipts = append(stored.Receipts, domain.CredentialReceipt{Step: "break-glass-expiry", State: string(stored.State), Outcome: "auto_revoked", At: now, Details: map[string]any{"follow_on_rotation": follow.ID, "follow_on_state": string(follow.State)}})
	stored.UpdatedAt = now
	if err := s.Store.SaveRotation(ctx, stored); err != nil {
		return follow, err
	}
	return follow, nil
}
