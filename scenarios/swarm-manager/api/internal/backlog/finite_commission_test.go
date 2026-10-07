package backlog

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"os"
	"strings"
	"swarm-manager/internal/identity"
	"sync"
	"testing"
	"time"
)

func TestFiniteCommissionStoreRejectsForgedAndStaleDisposition(t *testing.T) {
	s := NewFileEffortControlStore(t.TempDir())
	c := testEffortControlRequest("finite-store-fixture", 1)
	forged := c
	forged.FiniteCommission = &identity.FiniteCommissionRecord{Actor: "unverified", Generation: 1}
	if s.Save(forged) == nil {
		t.Fatal("body-created finite acceptance accepted")
	}
	if _, e := s.Load(c.EffortID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("forged save created owner state: %v", e)
	}
	if e := s.Save(c); e != nil {
		t.Fatal(e)
	}
	if e := s.updateFinite(c.EffortID, c.AuthorityDigest(), func(x identity.EffortControl) (identity.EffortControl, error) {
		x.FiniteCommission = &identity.FiniteCommissionRecord{Actor: "fixture-owner", Generation: 1}
		return x, nil
	}); e != nil {
		t.Fatal(e)
	}
	stale, e := s.Load(c.EffortID)
	if e != nil {
		t.Fatal(e)
	}
	// A second owner-store instance shares the durable per-effort lock.
	second := NewFileEffortControlStore(s.rootDir)
	if e := second.updateFinite(c.EffortID, c.AuthorityDigest(), func(x identity.EffortControl) (identity.EffortControl, error) {
		x.FiniteCommission.Revoked = true
		x.FiniteCommission.Generation++
		return x, nil
	}); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.Save(stale) == nil {
				t.Error("stale save resurrected revoked record")
			}
		}()
	}
	wg.Wait()
	current, e := s.Load(c.EffortID)
	if e != nil || current.FiniteCommission == nil || !current.FiniteCommission.Revoked || current.FiniteCommission.Generation != 2 {
		t.Fatalf("revocation not durable: %+v %v", current, e)
	}
	amended := current
	amended.Revision++
	if e := s.Save(amended); e != nil {
		t.Fatal(e)
	}
	if s.Save(current) == nil {
		t.Fatal("stale authority revision restored")
	}
}
func TestFiniteCommissionVerifiedContextRejectsBeforeOwnerEffects(t *testing.T) {
	for _, p := range []identity.Provenance{{}, {Actor: identity.TypeOperator, VerificationStatus: "absent", Subject: "owner"}, {Actor: identity.TypeAgent, VerificationStatus: identity.VerificationVerified, Subject: "owner"}, {Actor: identity.TypeOperator, VerificationStatus: identity.VerificationVerified, Subject: "other"}} {
		ctx := identity.NewContext(context.Background(), p)
		var o *FiniteCommissionOwner
		if o.AcceptFiniteCommission(ctx, effortauthority.CommissionSubject{Owner: "owner"}, 0) == nil {
			t.Fatal("invalid context accepted")
		}
		if o.RevokeFiniteCommission(ctx, "owner", "fixture", 1) == nil {
			t.Fatal("invalid context revoked")
		}
	}
	if verifiedCommissionActor(identity.NewContext(context.Background(), identity.Provenance{Actor: identity.TypeOperator, VerificationStatus: identity.VerificationVerified, Subject: "owner"}), "owner") != nil {
		t.Fatal("verified exact operator refused")
	}
}

func TestFiniteCommissionOwnerPositiveDriftAndRevocation(t *testing.T) {
	h, _ := setupTestHandler(t)
	subject := effortauthority.CommissionSubject{Owner: "verified-owner", Effort: "GOVERNED-FIXTURE", Revision: "1", ContentDigest: effortauthority.Digest("canonical content"), Team: "fixture-team", Members: []string{"leader"}, Repository: t.TempDir(), TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Profiles: map[string]string{"native": effortauthority.Digest("profile")}}
	item := BacklogItem{Kind: KindExecute, Name: "finite-fixture", Status: StatusBacklog, Title: "Bounded work", Effort: subject.Effort, PlanRef: &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "fixture-plan", Slug: "fixture-plan", Role: PlanRefRoleExecutionSpec}}
	item.PlanAcceptance = &PlanAcceptance{Actor: "prior-plan-actor", PlanContentHash: "sha256:" + subject.ContentDigest, SubjectVersion: PlanAcceptanceSubjectVersion(item)}
	if e := os.MkdirAll(h.store.ItemDir(item.Kind, item.Name), 0700); e != nil {
		t.Fatal(e)
	}
	if e := h.store.SaveItem(item); e != nil {
		t.Fatal(e)
	}
	// Bind acceptance to the persisted canonical authored projection, including
	// defaults normalized by the existing file owner. Do not weaken the predicate.
	stored, err := h.store.LoadItem(item.Kind, item.Name)
	if err != nil {
		t.Fatal(err)
	}
	stored.PlanAcceptance.SubjectVersion = PlanAcceptanceSubjectVersion(stored)
	if err := h.store.SaveItem(stored); err != nil {
		t.Fatal(err)
	}
	item = stored
	h.planClient = acceptancePlanClient{plan: &sharedv1.Plan{Id: "fixture-plan", Slug: "fixture-plan", ContentHash: item.PlanAcceptance.PlanContentHash, Status: sharedv1.PlanStatus_PLAN_STATUS_DRAFT}}
	store := NewFileEffortControlStore(t.TempDir())
	control := testEffortControlRequest(subject.Effort, 1)
	if e := store.Save(control); e != nil {
		t.Fatal(e)
	}
	h.SetEffortControlService(NewEffortControlService(store, nil))
	owner, e := NewFiniteCommissionOwner(h, map[string]FiniteCommissionTarget{subject.Effort: {WorkShape: FinitePlanBacked, Kind: KindExecute, Name: item.Name}})
	if e != nil {
		t.Fatal(e)
	}
	ctx := identity.NewContext(context.Background(), identity.Provenance{Actor: identity.TypeOperator, VerificationStatus: identity.VerificationVerified, Subject: subject.Owner})
	if e := owner.AcceptFiniteCommission(ctx, subject, 0); e != nil {
		t.Fatal(e)
	}
	original, e := store.Load(subject.Effort)
	if e != nil {
		t.Fatal(e)
	}
	// The read consumer's policy contains only authored subject plus valid limits;
	// no human token or grant is manufactured by this acceptance fixture.
	policy := effortauthority.Policy{ID: "fixture-policy", Owner: subject.Owner, Client: "fixture-client", ClientKey: make([]byte, 32), Epoch: 1, Repository: subject.Repository, Effort: subject.Effort, Revision: subject.Revision, ContentDigest: subject.ContentDigest, Team: subject.Team, TeamDigest: subject.TeamDigest, BindingDigest: subject.BindingDigest, Members: subject.Members, Profiles: subject.Profiles, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: time.Now().UTC().Add(-time.Second), Deadline: time.Now().UTC().Add(time.Hour), MaxStarts: 1, MaxConcurrent: 1, MaxTurns: 1, MaxToolCalls: 1, MaxRunSeconds: 10, TotalTurns: 1, TotalToolCalls: 1, TotalRunSeconds: 10, SurviveLogout: true}
	if _, e := owner.ReadFiniteAcceptance(context.Background(), policy); e != nil {
		t.Fatalf("exact accepted subject read: %v", e)
	}
	changed := policy
	changed.Team = "foreign"
	if _, e := owner.ReadFiniteAcceptance(context.Background(), changed); e == nil {
		t.Fatal("full subject team drift accepted")
	}
	item.Title = "changed authored contract"
	if e := h.store.SaveItem(item); e != nil {
		t.Fatal(e)
	}
	if _, e := owner.ReadFiniteAcceptance(context.Background(), policy); e == nil {
		t.Fatal("canonical authored contract drift accepted")
	}
	item.Title = "Bounded work"
	if e := h.store.SaveItem(item); e != nil {
		t.Fatal(e)
	}
	if e := owner.RevokeFiniteCommission(ctx, subject.Owner, subject.Effort, original.FiniteCommission.Generation); e != nil {
		t.Fatal(e)
	}
	if _, e := owner.ReadFiniteAcceptance(context.Background(), policy); e == nil {
		t.Fatal("revoked work accepted")
	}
	after, e := store.Load(subject.Effort)
	if e != nil || after.AuthorityDigest() != original.AuthorityDigest() || effortauthority.Digest(after.Completion) != effortauthority.Digest(original.Completion) {
		t.Fatalf("revocation changed work/completion authority: %v", e)
	}
}

func TestFiniteBoundedAuthorityCommissionRequiresNoPlanAcceptance(t *testing.T) {
	h, _ := setupTestHandler(t)
	store := NewFileEffortControlStore(t.TempDir())
	control := testEffortControlRequest("bounded-commission-fixture", 1)
	if e := store.Save(control); e != nil {
		t.Fatal(e)
	}
	h.SetEffortControlService(NewEffortControlService(store, nil))
	// Explicit trusted startup work shape, not an editable accepted label or
	// an exemption inferred from absent PlanAcceptance.
	owner, e := NewFiniteCommissionOwner(h, map[string]FiniteCommissionTarget{control.EffortID: {WorkShape: FiniteBoundedAuthority}})
	if e != nil {
		t.Fatal(e)
	}
	subject := effortauthority.CommissionSubject{Owner: "verified-owner", Effort: control.EffortID, Revision: "1", ContentDigest: strings.TrimPrefix(control.AuthorityDigest(), "sha256:"), Team: "bounded-team", Members: []string{"leader"}, Repository: t.TempDir(), TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Profiles: map[string]string{"native": effortauthority.Digest("profile")}}
	ctx := identity.NewContext(context.Background(), identity.Provenance{Actor: identity.TypeOperator, VerificationStatus: identity.VerificationVerified, Subject: subject.Owner})
	if e := owner.AcceptFiniteCommission(ctx, subject, 0); e != nil {
		t.Fatal(e)
	}
	c, e := store.Load(control.EffortID)
	if e != nil || c.FiniteCommission == nil || c.FiniteCommission.WorkShape != string(FiniteBoundedAuthority) || c.FiniteCommission.PlanContentHash != "" {
		t.Fatalf("bounded acceptance: %+v %v", c, e)
	}
	if _, _, e := owner.current(context.Background(), subject); e != nil {
		t.Fatal(e)
	}
	changed := subject
	changed.ContentDigest = effortauthority.Digest("unapproved prose")
	if _, _, e := owner.current(context.Background(), changed); e == nil {
		t.Fatal("foreign authored content accepted")
	}
	amended := c
	amended.Revision++
	if e := store.Save(amended); e != nil {
		t.Fatal(e)
	}
	if _, _, e := owner.current(context.Background(), subject); e == nil {
		t.Fatal("superseded revision accepted")
	}
	if e := owner.RevokeFiniteCommission(ctx, subject.Owner, subject.Effort, c.FiniteCommission.Generation); e != nil {
		t.Fatal(e)
	}
}
