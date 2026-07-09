package planimport

import (
	"context"
	"encoding/json"
	"testing"

	"swarm-manager/internal/identity"
	"swarm-manager/internal/planclient"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func fixturePlan() *sharedv1.Plan {
	return &sharedv1.Plan{
		Id:   "plan-123",
		Slug: "my-plan",
		ChangeBoundary: &sharedv1.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/my-plan/**"},
			AcceptanceDeny:  []string{"scenarios/my-plan/unsafe/**"},
		},
		// Deliberately out of order to prove Translate sorts by Order.
		Phases: []*sharedv1.Phase{
			{Order: 2, Title: "Build", Intent: "do the build"},
			{
				Order:  1,
				Title:  "Design",
				Intent: "design it",
				ChangeBoundary: &sharedv1.ChangeBoundary{
					AcceptanceAllow: []string{"scenarios/my-plan/design/**"},
					AcceptanceDeny:  []string{"scenarios/my-plan/design/tmp/**"},
				},
			},
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
		if it.PlanRef == nil {
			t.Fatalf("item[%d] missing plan_ref", i)
		}
		if it.PlanRef.Provider != "plan-manager" || it.PlanRef.PlanID != "plan-123" || it.PlanRef.Slug != "my-plan" || it.PlanRef.Role != "execution_spec" {
			t.Errorf("item[%d] plan_ref = %#v", i, it.PlanRef)
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
	if got := payload.Items[0].AcceptanceAllow; len(got) != 1 || got[0] != "scenarios/my-plan/design/**" {
		t.Errorf("phase boundary acceptance_allow = %v", got)
	}
	if got := payload.Items[0].AcceptanceDeny; len(got) != 1 || got[0] != "scenarios/my-plan/design/tmp/**" {
		t.Errorf("phase boundary acceptance_deny = %v", got)
	}
	if got := payload.Items[1].AcceptanceAllow; len(got) != 1 || got[0] != "scenarios/my-plan/**" {
		t.Errorf("plan fallback acceptance_allow = %v", got)
	}
	if got := payload.Items[1].AcceptanceDeny; len(got) != 1 || got[0] != "scenarios/my-plan/unsafe/**" {
		t.Errorf("plan fallback acceptance_deny = %v", got)
	}
}

func TestTranslate_RejectsEmpty(t *testing.T) {
	if _, err := Translate(nil); err == nil {
		t.Error("expected error for nil plan")
	}
	if _, err := Translate(&sharedv1.Plan{Slug: "s"}); err == nil {
		t.Error("expected error for plan with no id")
	}
	if _, err := Translate(&sharedv1.Plan{Id: "id", Slug: "s"}); err == nil {
		t.Error("expected error for plan with no phases")
	}
	if _, err := Translate(&sharedv1.Plan{Id: "id", Phases: []*sharedv1.Phase{{Order: 1}}}); err == nil {
		t.Error("expected error for plan with no slug")
	}
}

// stubFetcher / stubLander exercise the Service end to end.
type stubFetcher struct{ plan *sharedv1.Plan }

func (s stubFetcher) ListPlans(context.Context) ([]*sharedv1.Plan, error) {
	return []*sharedv1.Plan{s.plan}, nil
}

func (s stubFetcher) GetPlan(context.Context, string) (*sharedv1.Plan, error) { return s.plan, nil }

func (s stubFetcher) ImportPlan(context.Context, planclient.ImportPlanInput) (*sharedv1.Plan, error) {
	return s.plan, nil
}

type stubLander struct{ gotPayload string }

func (s *stubLander) LandBatch(_ context.Context, payload BatchPayload, _ identity.Provenance) ([]ImportedRef, error) {
	data, _ := json.Marshal(payload)
	s.gotPayload = string(data)
	refs := make([]ImportedRef, len(payload.Items))
	for i, it := range payload.Items {
		action := "linked"
		if i == 0 {
			action = "created"
		} else if i == 1 {
			action = "updated"
		}
		refs[i] = ImportedRef{Kind: it.Kind, Name: it.Name, Title: it.Title, Action: action}
	}
	return refs, nil
}

type stubInitiativeLander struct {
	calls []struct {
		spec InitiativeSpec
		refs []ImportedRef
	}
	firstAction string
}

func (s *stubInitiativeLander) LandInitiative(_ context.Context, spec InitiativeSpec, refs []ImportedRef, _ identity.Provenance) (ImportedInitiative, error) {
	s.calls = append(s.calls, struct {
		spec InitiativeSpec
		refs []ImportedRef
	}{spec: spec, refs: refs})
	action := "linked"
	if len(s.calls) == 1 && s.firstAction != "" {
		action = s.firstAction
	} else if len(refs) > 0 {
		action = "updated"
	}
	return ImportedInitiative{Name: spec.Name, Title: spec.Title, Mode: spec.Mode, Action: action}, nil
}

func TestService_ImportFetchesTranslatesAndLands(t *testing.T) {
	lander := &stubLander{}
	svc := NewService(stubFetcher{plan: fixturePlan()}, lander, nil)

	res, err := svc.Import(context.Background(), Request{PlanID: "my-plan"}, identity.Provenance{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Slug != "my-plan" || res.Count != 3 || len(res.Items) != 3 {
		t.Fatalf("result = %+v, want slug my-plan count 3", res)
	}
	if res.Created != 1 || res.Updated != 1 || res.Linked != 1 {
		t.Fatalf("counts = created %d updated %d linked %d", res.Created, res.Updated, res.Linked)
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

func TestService_ImportAdoptsExternalMarkdown(t *testing.T) {
	lander := &stubLander{}
	fetcher := recordingFetcher{plan: fixturePlan()}
	svc := NewService(&fetcher, lander, nil)

	_, err := svc.Import(context.Background(), Request{
		Markdown: "# My Plan",
		Title:    "Adopted",
		Slug:     "adopted",
	}, identity.Provenance{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if fetcher.importInput.Markdown != "# My Plan" || fetcher.importInput.Slug != "adopted" {
		t.Fatalf("import input = %+v", fetcher.importInput)
	}
	if fetcher.getID != "" {
		t.Fatalf("GetPlan should not be called for markdown adoption, got %q", fetcher.getID)
	}
}

func TestService_ImportInitiativeContainer(t *testing.T) {
	lander := &stubLander{}
	initLander := &stubInitiativeLander{firstAction: "created"}
	svc := NewService(stubFetcher{plan: fixturePlan()}, lander, initLander)

	res, err := svc.Import(context.Background(), Request{
		PlanID: "my-plan",
		Container: ContainerSpec{
			Type:  "initiative",
			Name:  "my-plan-work",
			Title: "My Plan Work",
			Mode:  "phased-plan-drain",
		},
	}, identity.Provenance{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Container != "initiative" || res.Initiative == nil {
		t.Fatalf("result missing initiative: %+v", res)
	}
	if res.Initiative.Action != "created" {
		t.Fatalf("initiative action = %q, want created", res.Initiative.Action)
	}
	if len(initLander.calls) != 2 {
		t.Fatalf("initiative lander calls = %d, want 2", len(initLander.calls))
	}
	if initLander.calls[0].spec.PlanRef.Role != "operating_mode_plan" {
		t.Fatalf("initiative plan_ref = %+v", initLander.calls[0].spec.PlanRef)
	}
	if len(initLander.calls[1].refs) != 3 {
		t.Fatalf("initiative item refs = %d, want 3", len(initLander.calls[1].refs))
	}
	var p BatchPayload
	if err := json.Unmarshal([]byte(lander.gotPayload), &p); err != nil {
		t.Fatalf("landed payload not valid JSON: %v", err)
	}
	for _, item := range p.Items {
		if item.Initiative != "my-plan-work" {
			t.Fatalf("item %s initiative = %q", item.Name, item.Initiative)
		}
	}
}

type recordingFetcher struct {
	plan        *sharedv1.Plan
	getID       string
	importInput planclient.ImportPlanInput
}

func (s *recordingFetcher) GetPlan(_ context.Context, id string) (*sharedv1.Plan, error) {
	s.getID = id
	return s.plan, nil
}

func (s *recordingFetcher) ListPlans(context.Context) ([]*sharedv1.Plan, error) {
	return []*sharedv1.Plan{s.plan}, nil
}

func (s *recordingFetcher) ImportPlan(_ context.Context, input planclient.ImportPlanInput) (*sharedv1.Plan, error) {
	s.importInput = input
	return s.plan, nil
}
