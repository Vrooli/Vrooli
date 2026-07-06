package planimport

import (
	"context"
	"encoding/json"
	"testing"

	"swarm-manager/internal/identity"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func fixturePlan() *sharedv1.Plan {
	return &sharedv1.Plan{
		Slug: "my-plan",
		// Deliberately out of order to prove Translate sorts by Order.
		Phases: []*sharedv1.Phase{
			{Order: 2, Title: "Build", Intent: "do the build"},
			{Order: 1, Title: "Design", Intent: "design it"},
			{Order: 3, Title: "Verify", Acceptance: "it passes"},
		},
	}
}

func TestTranslate_LinearChainWithProvenance(t *testing.T) {
	payload, err := Translate(fixturePlan())
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if len(payload.Items) != 3 {
		t.Fatalf("items = %d, want 3 (one per phase)", len(payload.Items))
	}

	// Phase order (1,2,3) with a linear depends_on chain.
	wantNames := []string{"my-plan-phase-1", "my-plan-phase-2", "my-plan-phase-3"}
	for i, it := range payload.Items {
		if it.Name != wantNames[i] {
			t.Fatalf("item[%d] name = %q, want %q", i, it.Name, wantNames[i])
		}
		if it.Kind != "execute" {
			t.Errorf("item[%d] kind = %q, want execute", i, it.Kind)
		}
		if it.Effort != "" {
			t.Errorf("item[%d] effort = %q, want unsized", i, it.Effort)
		}
		wantProv := "plan-manager:my-plan/phase-" + string(rune('1'+i))
		if it.SpawnedFrom != wantProv {
			t.Errorf("item[%d] spawned_from = %q, want %q", i, it.SpawnedFrom, wantProv)
		}
		if i == 0 {
			if len(it.DependsOn) != 0 {
				t.Errorf("first item should have no deps, got %v", it.DependsOn)
			}
		} else {
			if len(it.DependsOn) != 1 || it.DependsOn[0] != "execute/"+wantNames[i-1] {
				t.Errorf("item[%d] depends_on = %v, want [execute/%s]", i, it.DependsOn, wantNames[i-1])
			}
		}
	}

	// Description falls back to acceptance when intent is empty (phase 3).
	if payload.Items[2].Description != "it passes" {
		t.Errorf("phase 3 description = %q, want acceptance fallback", payload.Items[2].Description)
	}
	// Title carried through (phase 1 = Design).
	if payload.Items[0].Title != "Design" {
		t.Errorf("phase 1 title = %q, want Design", payload.Items[0].Title)
	}
}

func TestTranslate_RejectsEmpty(t *testing.T) {
	if _, err := Translate(nil); err == nil {
		t.Error("expected error for nil plan")
	}
	if _, err := Translate(&sharedv1.Plan{Slug: "s"}); err == nil {
		t.Error("expected error for plan with no phases")
	}
	if _, err := Translate(&sharedv1.Plan{Phases: []*sharedv1.Phase{{Order: 1}}}); err == nil {
		t.Error("expected error for plan with no slug")
	}
}

// stubFetcher / stubLander exercise the Service end to end.
type stubFetcher struct{ plan *sharedv1.Plan }

func (s stubFetcher) GetPlan(context.Context, string) (*sharedv1.Plan, error) { return s.plan, nil }

type stubLander struct{ gotPayload string }

func (s *stubLander) ImportBatch(_ context.Context, payloadJSON string, _ identity.Provenance) ([]ImportedRef, error) {
	s.gotPayload = payloadJSON
	var p BatchPayload
	_ = json.Unmarshal([]byte(payloadJSON), &p)
	refs := make([]ImportedRef, len(p.Items))
	for i, it := range p.Items {
		refs[i] = ImportedRef{Kind: it.Kind, Name: it.Name, Title: it.Title}
	}
	return refs, nil
}

func TestService_ImportFetchesTranslatesAndLands(t *testing.T) {
	lander := &stubLander{}
	svc := NewService(stubFetcher{plan: fixturePlan()}, lander)

	res, err := svc.Import(context.Background(), "my-plan", identity.Provenance{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Slug != "my-plan" || res.Count != 3 || len(res.Items) != 3 {
		t.Fatalf("result = %+v, want slug my-plan count 3", res)
	}
	// The lander received a payload with the linear chain + provenance.
	var p BatchPayload
	if err := json.Unmarshal([]byte(lander.gotPayload), &p); err != nil {
		t.Fatalf("landed payload not valid JSON: %v", err)
	}
	if p.Items[1].SpawnedFrom != "plan-manager:my-plan/phase-2" {
		t.Errorf("landed spawned_from = %q", p.Items[1].SpawnedFrom)
	}
	if len(p.Items[2].DependsOn) != 1 || p.Items[2].DependsOn[0] != "execute/my-plan-phase-2" {
		t.Errorf("landed chain wrong: %v", p.Items[2].DependsOn)
	}
}
