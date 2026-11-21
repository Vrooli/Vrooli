package credentials

import (
	"context"
	"time"

	"scenario-to-cloud/domain"
)

const (
	LifecyclePlanned    = "planned"
	LifecycleActive     = "active"
	LifecycleRenewalDue = "renewal_due"
	LifecycleExpired    = "expired"
	LifecycleRevoked    = "revoked"
)

const DefaultRenewalWindow = 30 * 24 * time.Hour

// BindingView is the metadata-only read surface: bindings, versions, acks and
// a derived lifecycle standing. The standing is deliberately not persisted;
// it is evaluated against the read clock so expiry cannot become stale state.
type BindingView struct {
	Binding         domain.CredentialBinding `json:"binding"`
	Acks            []domain.CredentialAck   `json:"acks"`
	LifecycleState  string                   `json:"lifecycle_state"`
	LifecycleDetail string                   `json:"lifecycle_detail,omitempty"`
	NextAction      string                   `json:"next_action,omitempty"`
}

func lifecycleStanding(binding domain.CredentialBinding, now time.Time) (string, string, string) {
	switch binding.State {
	case domain.CredentialBindingRevoked:
		return LifecycleRevoked, "This credential was revoked and cannot be used for new distribution.", "materialize a replacement credential"
	case domain.CredentialBindingPlanned:
		return LifecyclePlanned, "This credential has not been materialized on the target.", "materialize the credential"
	}
	if binding.Version.Expired(now) {
		return LifecycleExpired, "The active credential version has expired and is not release-ready.", "rotate the credential before retrying release"
	}
	if binding.Version.RenewalDue(now, DefaultRenewalWindow) {
		return LifecycleRenewalDue, "The active credential version is inside its renewal window.", "rotate the credential before it expires"
	}
	return LifecycleActive, "The active credential version is within its declared validity period.", ""
}

// ListBindings returns every binding with its acknowledgements.
func (s *Service) ListBindings(ctx context.Context, deploymentID string) ([]BindingView, error) {
	bindings, err := s.Store.ListBindings(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	out := make([]BindingView, 0, len(bindings))
	now := s.now()
	for _, b := range bindings {
		acks, err := s.Store.ListAcks(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		if acks == nil {
			acks = []domain.CredentialAck{}
		}
		state, detail, next := lifecycleStanding(b, now)
		out = append(out, BindingView{Binding: b, Acks: acks, LifecycleState: state, LifecycleDetail: detail, NextAction: next})
	}
	return out, nil
}

// GetRotation reads one operation.
func (s *Service) GetRotation(ctx context.Context, id string) (*domain.CredentialRotation, error) {
	rotation, err := s.Store.GetRotation(ctx, id)
	if err != nil {
		return nil, err
	}
	if rotation == nil {
		return nil, newError(CodeRotationNotFound, "no such credential operation").WithDetail("rotation_id", id)
	}
	return rotation, nil
}
