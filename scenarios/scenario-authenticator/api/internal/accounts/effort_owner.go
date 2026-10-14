package accounts

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"scenario-authenticator/internal/realm"
)

// ApproveEffortOwner is the identity authority's approval-time adapter. The
// body never selects another principal, including for administrators.
func (s *Service) ApproveEffortOwner(ctx context.Context, accessToken string) (effortauthority.OwnerState, error) {
	v, ok, e := s.Validate(ctx, accessToken)
	if e != nil || !ok {
		return effortauthority.OwnerState{}, effortauthority.ErrRefused
	}
	current, e := s.CurrentEffortOwner(ctx, v.UserID)
	if e != nil {
		return current, e
	}
	// Preserve the offered JWT ceiling; current grants never enlarge it.
	current.Ceiling = append([]string{}, v.Scopes...)
	return current, nil
}

// CurrentEffortOwner is a narrow revocation-coordination read for a retained
// finite grant, not an ordinary JWT authentication endpoint. An owner-private
// authority invokes it only after locating its protected grant. It cannot mint,
// refresh, impersonate an account, or target a principal from an external body.
// Explicit overnight grants survive session logout; account lock/removal and
// current coarse-grant loss refuse new admissions. Delegation/client revocation
// is separately held in that authority's immutable policy/epoch ledger.
func (s *Service) CurrentEffortOwner(ctx context.Context, id string) (effortauthority.OwnerState, error) {
	if s == nil || s.repo == nil || s.authorization == nil {
		return effortauthority.OwnerState{}, effortauthority.ErrRefused
	}
	account, e := s.repo.FindByID(ctx, id)
	if e != nil || account.ID != id || account.RealmID != realm.DefaultID || account.Locked(s.clock.Now()) {
		return effortauthority.OwnerState{}, effortauthority.ErrRefused
	}
	scopes, e := s.authorization.List(ctx, account.ID)
	if e != nil {
		return effortauthority.OwnerState{}, e
	}
	return effortauthority.OwnerState{Subject: account.ID, Enabled: true, Scopes: scopes}, nil
}
