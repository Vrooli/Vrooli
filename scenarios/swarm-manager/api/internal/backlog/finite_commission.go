// finite_commission.go provides protected exact commission acceptance under the existing owner.
package backlog

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"strconv"
	"strings"
	"swarm-manager/internal/identity"
	"time"
)

type FiniteCommissionWorkShape string

const (
	FinitePlanBacked       FiniteCommissionWorkShape = "plan-backed"
	FiniteBoundedAuthority FiniteCommissionWorkShape = "bounded-authority"
)

type FiniteCommissionTarget struct {
	WorkShape FiniteCommissionWorkShape
	Kind      BacklogKind
	Name      string
}

// FiniteCommissionOwner is unmounted trusted setup. Its immutable target table
// resolves actual governed work; HTTP cannot supply a storage path or actor.
type FiniteCommissionOwner struct {
	handler *Handler
	targets map[string]FiniteCommissionTarget
	store   *FileEffortControlStore
}

func NewFiniteCommissionOwner(h *Handler, targets map[string]FiniteCommissionTarget) (*FiniteCommissionOwner, error) {
	if h == nil || h.effortControl == nil || h.store == nil || len(targets) == 0 {
		return nil, effortauthority.ErrRefused
	}
	store, ok := h.effortControl.store.(*FileEffortControlStore)
	if !ok || store == nil {
		return nil, effortauthority.ErrRefused
	}
	copy := map[string]FiniteCommissionTarget{}
	for id, t := range targets {
		if id == "" || (t.WorkShape != FinitePlanBacked && t.WorkShape != FiniteBoundedAuthority) || (t.WorkShape == FinitePlanBacked && (t.Name == "" || strings.ContainsAny(t.Name, "/\\") || t.Kind == "" || h.planClient == nil)) {
			return nil, effortauthority.ErrRefused
		}
		copy[id] = t
	}
	return &FiniteCommissionOwner{h, copy, store}, nil
}
func verifiedCommissionActor(ctx context.Context, owner string) error {
	p := identity.FromContext(ctx)
	if ctx.Err() != nil || owner == "" || p.Subject != owner || identity.VerifiedOperatorActor(p) == "" {
		return effortauthority.ErrRefused
	}
	return nil
}
func (o *FiniteCommissionOwner) current(ctx context.Context, s effortauthority.CommissionSubject) (identity.EffortControl, BacklogItem, error) {
	if o == nil || ctx.Err() != nil || s.Validate() != nil {
		return identity.EffortControl{}, BacklogItem{}, effortauthority.ErrRefused
	}
	target, ok := o.targets[s.Effort]
	if !ok {
		return identity.EffortControl{}, BacklogItem{}, effortauthority.ErrRefused
	}
	c, e := o.store.Load(s.Effort)
	if e != nil || strconv.FormatInt(c.Revision, 10) != s.Revision {
		return c, BacklogItem{}, effortauthority.ErrRefused
	}
	if target.WorkShape == FiniteBoundedAuthority {
		if strings.TrimPrefix(c.AuthorityDigest(), "sha256:") != s.ContentDigest {
			return c, BacklogItem{}, effortauthority.ErrRefused
		}
		return c, BacklogItem{}, nil
	}
	item, e := o.handler.store.LoadItem(target.Kind, target.Name)
	if e != nil || item.Effort != s.Effort || item.ArchivedAt != nil || o.handler.ValidateAcceptedPlan(ctx, item) != nil || item.PlanAcceptance == nil {
		return c, item, effortauthority.ErrRefused
	}
	hash := strings.TrimPrefix(item.PlanAcceptance.PlanContentHash, "sha256:")
	if hash != s.ContentDigest {
		return c, item, effortauthority.ErrRefused
	}
	return c, item, nil
}

// AcceptFiniteCommission records only the already reviewed exact governed work.
// No token, body actor, editable label or completion disposition is authority.
func (o *FiniteCommissionOwner) AcceptFiniteCommission(ctx context.Context, s effortauthority.CommissionSubject, expectedGeneration uint64) error {
	if verifiedCommissionActor(ctx, s.Owner) != nil {
		return effortauthority.ErrRefused
	}
	c, item, e := o.current(ctx, s)
	if e != nil {
		return e
	}
	return o.store.updateFinite(s.Effort, c.AuthorityDigest(), func(current identity.EffortControl) (identity.EffortControl, error) {
		return o.acceptFiniteTransform(ctx, s, expectedGeneration, current, item)
	})
}

// acceptFiniteTransform is shared by the existing owner and the atomic public
// decision adapter. Callers must hold the existing effort lock and verify their
// channel identity before entering it. It does not persist or dispatch.
func (o *FiniteCommissionOwner) acceptFiniteTransform(ctx context.Context, s effortauthority.CommissionSubject, expectedGeneration uint64, current identity.EffortControl, item BacklogItem) (identity.EffortControl, error) {
	generation := uint64(0)
	if current.FiniteCommission != nil {
		generation = current.FiniteCommission.Generation
	}
	if generation != expectedGeneration || generation == ^uint64(0) {
		return current, effortauthority.ErrRefused
	}
	if current.FiniteCommission != nil && current.FiniteCommission.Subject.Revision == s.Revision {
		return current, effortauthority.ErrRefused
	}
	// Re-read the governed frontier while holding the authority write lock.
	live, liveItem, err := o.current(ctx, s)
	if err != nil || live.AuthorityDigest() != current.AuthorityDigest() || finitePlanSubjectVersion(liveItem) != finitePlanSubjectVersion(item) {
		return current, effortauthority.ErrRefused
	}
	// Deep copy authored maps/slices so caller mutation cannot amend acceptance.
	subject := s
	subject.Members = append([]string(nil), s.Members...)
	subject.Profiles = map[string]string{}
	for k, v := range s.Profiles {
		subject.Profiles[k] = v
	}
	planContentHash := ""
	if liveItem.PlanAcceptance != nil {
		planContentHash = liveItem.PlanAcceptance.PlanContentHash
	}
	current.FiniteCommission = &identity.FiniteCommissionRecord{WorkShape: string(o.targets[s.Effort].WorkShape), Subject: subject, AuthorityDigest: current.AuthorityDigest(), PlanSubjectVersion: finitePlanSubjectVersion(liveItem), PlanContentHash: planContentHash, Actor: s.Owner, AcceptedAt: time.Now().UTC(), Generation: generation + 1}
	return current, nil
}

// Revoke closes future admission only. It never unaccepts a plan, marks a run
// terminal, cancels execution or changes existing completion obligations.
func (o *FiniteCommissionOwner) RevokeFiniteCommission(ctx context.Context, owner, effort string, expectedGeneration uint64) error {
	if o == nil || verifiedCommissionActor(ctx, owner) != nil {
		return effortauthority.ErrRefused
	}
	if _, installed := o.targets[effort]; !installed {
		return effortauthority.ErrRefused
	}
	c, e := o.store.Load(effort)
	if e != nil || c.FiniteCommission == nil {
		return effortauthority.ErrRefused
	}
	return o.store.updateFinite(effort, c.AuthorityDigest(), func(current identity.EffortControl) (identity.EffortControl, error) {
		r := current.FiniteCommission
		if r == nil || r.Subject.Owner != owner || r.Generation != expectedGeneration || r.Generation == ^uint64(0) || r.Revoked {
			return current, effortauthority.ErrRefused
		}
		r.Revoked = true
		r.Generation++
		r.RevokedAt = time.Now().UTC()
		return current, nil
	})
}
func (o *FiniteCommissionOwner) ReadFiniteAcceptance(ctx context.Context, p effortauthority.Policy) (effortauthority.AcceptanceContract, error) {
	var out effortauthority.AcceptanceContract
	if p.Validate() != nil {
		return out, effortauthority.ErrRefused
	}
	s := effortauthority.CommissionSubjectFor(p)
	c, item, e := o.current(ctx, s)
	if e != nil {
		return out, e
	}
	r := c.FiniteCommission
	planHash := ""
	if item.PlanAcceptance != nil {
		planHash = item.PlanAcceptance.PlanContentHash
	}
	if r == nil || r.WorkShape != string(o.targets[s.Effort].WorkShape) || r.Revoked || r.Actor != s.Owner || r.AcceptedAt.IsZero() || r.AcceptedAt.After(time.Now()) || r.Generation == 0 || r.AuthorityDigest != c.AuthorityDigest() || r.PlanSubjectVersion != finitePlanSubjectVersion(item) || r.PlanContentHash != planHash || effortauthority.Digest(r.Subject) != effortauthority.Digest(s) {
		return out, effortauthority.ErrRefused
	}
	// Fence a concurrent revoke/amend that completed during the live plan read.
	after, e := o.store.Load(s.Effort)
	if e != nil || effortauthority.Digest(after) != effortauthority.Digest(c) {
		return out, effortauthority.ErrRefused
	}
	return effortauthority.AcceptanceContract{Subject: r.Subject, Actor: r.Actor, AcceptedAt: r.AcceptedAt, Generation: r.Generation}, nil
}

var _ effortauthority.AcceptanceContractOwner = (*FiniteCommissionOwner)(nil)

func finitePlanSubjectVersion(item BacklogItem) string {
	if item.PlanAcceptance == nil {
		return ""
	}
	return PlanAcceptanceSubjectVersion(item)
}
