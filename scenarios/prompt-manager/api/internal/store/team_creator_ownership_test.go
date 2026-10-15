package store

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

// Creator-owns-while-alive: the team's own verified creator may update what it
// created without a separate team.manage grant, but any other agent — and any
// caller against a team with no recorded creator — still faces the gate. The
// bypass only matters inside the bounded/granted branch, so every case below
// presents a grant that does NOT cover team.manage on "draft", isolating the
// bypass from the plain-principal path that skips the gate entirely.
func TestTeamUpdateCreatorOwnsWhileAlive(t *testing.T) { // [REQ:PM-TEAM-CREATOR-OWNERSHIP]
	withUnrelatedGrant := func(p identity.Principal) context.Context {
		p.Grant = &identity.OperationGrant{
			ID: "g", Approver: "owner",
			Operations: []string{"team.prepare"},
			Targets:    []string{"prompt-manager:team/other"},
			ExpiresAt:  time.Now().Add(time.Hour),
		}
		return identity.WithPrincipal(context.Background(), p)
	}

	t.Run("verified creator bypasses the operation gate", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true) // team "draft" created WITH creator-run attribution
		ctx := withUnrelatedGrant(preparationActor())
		if err := s.Update(ctx, "draft", &Team{Mission: "updated by its creator"}); err != nil {
			t.Fatalf("creator updating its own team must not need a team.manage grant: %v", err)
		}
		got, err := s.Get(context.Background(), "draft")
		if err != nil || got.Mission != "updated by its creator" {
			t.Fatalf("creator update did not persist: mission=%q err=%v", got.Mission, err)
		}
	})

	t.Run("a different agent is still blocked", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true)
		other := preparationActor()
		other.Subject = "other-run"
		other.Owner = "other-owner"
		if err := s.Update(withUnrelatedGrant(other), "draft", &Team{Mission: "updated by an intruder"}); err == nil {
			t.Fatal("a non-creator agent must still satisfy the team.manage operation gate")
		}
	})

	t.Run("legacy team with no creation proof is still gated", func(t *testing.T) {
		s, _, _ := preparationFixture(t, false) // created WITHOUT attribution -> CreationProof nil
		if err := s.Update(withUnrelatedGrant(preparationActor()), "draft", &Team{Mission: "updated"}); err == nil {
			t.Fatal("a team with no recorded creator must not be bypassable")
		}
	})
}
