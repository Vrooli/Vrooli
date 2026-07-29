package studio

import (
	"errors"
	"testing"
	"time"
)

func product(id string) Identity {
	return Identity{ID: id, Name: "Vrooli", Kind: Product, Traits: map[string]string{"form": "console", "finish": "slate"}}
}

func TestIdentityRevisionAndReleaseSpine(t *testing.T) { // [REQ:ASSET-P0-001] [REQ:ASSET-P0-002] [REQ:ASSET-P0-010] [REQ:ASSET-P0-011] [REQ:ASSET-P0-012] [REQ:ASSET-P0-013] [REQ:ASSET-P0-017] [REQ:ASSET-P0-018]
	s := New()
	now := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	id := product("product-v1")
	if err := s.Author(id, "op-1", Operator, now); err != nil {
		t.Fatal(err)
	}
	updated := product("product-v1")
	updated.Traits["finish"] = "cyan"
	revised, err := s.Revise(updated, "op-1", Operator, now)
	if err != nil || revised.Version != 1 {
		t.Fatalf("unreferenced revision = %#v, %v", revised, err)
	}
	s.Renders["r"] = &Render{ID: "r", Status: RenderRunning, ActualCost: 0.02, ActualCostRecorded: true, Provenance: &Provenance{SpecID: "s", Backend: "fake", Model: "test", Seed: "1", Parameters: "{}"}, AssetIDs: []string{"a", "discard"}}
	s.Assets["a"] = &Asset{ID: "a", RenderID: "r", AltText: "Vrooli console", Disclosure: "AI-generated", AIgenerated: true, IdentityVersionIDs: []string{"product-v1"}}
	s.Assets["discard"] = &Asset{ID: "discard", RenderID: "r", AltText: "other", Disclosure: "AI-generated", AIgenerated: true}
	if err := s.Select("a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Release("a"); !hasCause(err, CauseUnresolved) {
		t.Fatalf("expected unresolved cause, got %v", err)
	}
	if err := s.Judge(Verdict{AssetID: "a", IdentityVersionID: "product-v1", ActorID: "agent", ActorKind: Agent, Passed: true, Basis: ProseOnly}); err == nil {
		t.Fatal("agent verdict accepted")
	}
	if err := s.Judge(Verdict{AssetID: "a", IdentityVersionID: "product-v1", ActorID: "op-1", ActorKind: Operator, Passed: true, Basis: ProseOnly}); err != nil {
		t.Fatal(err)
	}
	if err := s.Release("a"); err != nil {
		t.Fatal(err)
	}
	if !s.Identities["product-v1"].Referenced {
		t.Fatal("release did not freeze identity")
	}
	updated = product("product-v1")
	revised, err = s.Revise(updated, "op-1", Operator, now)
	if err != nil || revised.Version != 2 {
		t.Fatalf("frozen revision = %#v, %v", revised, err)
	}
}

func TestReleaseCausesAndRenderInvariants(t *testing.T) { // [REQ:ASSET-P0-006] [REQ:ASSET-P0-007] [REQ:ASSET-P0-008]
	s := New()
	s.Assets["a"] = &Asset{ID: "a", Status: InReview}
	if !hasCause(s.Release("a"), CauseAltText) {
		t.Fatal("missing alt text cause")
	}
	s.Assets["a"].AltText = "alt"
	if !hasCause(s.Release("a"), CauseDisclosure) {
		t.Fatal("missing disclosure cause")
	}
	s.Assets["a"].Disclosure = "required"
	s.Assets["a"].CredentialClaims = "doctor"
	if !hasCause(s.Release("a"), CauseCredentialClaims) {
		t.Fatal("credential cause")
	}
	r := &Render{ID: "r", Status: RenderQueued}
	if err := r.Transition(RenderSucceeded); err == nil {
		t.Fatal("illegal skip accepted")
	}
	if err := r.Transition(RenderRunning); err != nil {
		t.Fatal(err)
	}
	if err := r.Transition(RenderSucceeded); err == nil {
		t.Fatal("missing provenance accepted")
	}
	r.Provenance = &Provenance{SpecID: "s"}
	if err := r.Transition(RenderSucceeded); err == nil {
		t.Fatal("missing actual cost accepted")
	}
}

func TestCampaignBudgetRequiresRecordedOperatorConfirmation(t *testing.T) { // [REQ:ASSET-P1-006]
	s := New()
	if err := s.SetCampaignBudget("launch", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.AuthorizeRender("launch", 1.1, false, "", time.Now()); err == nil {
		t.Fatal("over-budget render was accepted without confirmation")
	}
	if err := s.AuthorizeRender("launch", 1.1, true, "operator-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(s.CampaignBudgets["launch"].Confirmations) != 1 {
		t.Fatal("confirmation not retained")
	}
	s.RecordRenderSpend("launch", 0.8)
	if s.CampaignBudgets["launch"].SpentUSD != 0.8 {
		t.Fatalf("spent=%f", s.CampaignBudgets["launch"].SpentUSD)
	}
}

func hasCause(err error, want ReleaseCause) bool {
	var got *ReleaseError
	return errors.As(err, &got) && got.Cause == want
}
