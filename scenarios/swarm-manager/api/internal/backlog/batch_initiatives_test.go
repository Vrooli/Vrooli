package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/testutil"
)

func intPtr(v int) *int { return &v }

func TestBatchCreate_MilestonePriorityAndDeps_Preview(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	deps := []string{"foundation"}
	payload := batchCreateRequest{
		Preview: true,
		Items: []batchCreateItem{
			{Name: "f-item", Title: "F Item", Kind: "idea", Milestone: "foundation"},
			{Name: "d-item", Title: "D Item", Kind: "idea", Milestone: "dependent"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "foundation", Title: "Foundation", Priority: intPtr(1)},
			{Name: "dependent", Title: "Dependent", Priority: intPtr(3), DependsOn: &deps},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if !resp.Preview {
		t.Fatal("expected preview=true in response")
	}
	if len(resp.Milestones) != 2 {
		t.Fatalf("expected 2 milestone results, got %d", len(resp.Milestones))
	}

	byName := make(map[string]batchCreateMilestoneResult, len(resp.Milestones))
	for _, r := range resp.Milestones {
		byName[r.Name] = r
	}

	if byName["foundation"].Priority != 1 {
		t.Errorf("foundation priority = %d, want 1", byName["foundation"].Priority)
	}
	if byName["dependent"].Priority != 3 {
		t.Errorf("dependent priority = %d, want 3", byName["dependent"].Priority)
	}
	if len(byName["dependent"].DependsOn) != 1 || byName["dependent"].DependsOn[0] != "foundation" {
		t.Errorf("dependent depends_on = %v, want [foundation]", byName["dependent"].DependsOn)
	}
}

func TestBatchCreate_MilestonePriority_InvalidRange(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Milestone: "bad"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "bad", Title: "Bad", Priority: intPtr(99)},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "priority") {
		t.Errorf("expected priority error, got: %s", w.Body.String())
	}
}

func TestBatchCreate_MilestoneDepends_UnknownRef(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	ghost := []string{"ghost-milestone"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Milestone: "real"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "real", Title: "Real", DependsOn: &ghost},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "ghost-milestone") {
		t.Errorf("expected unknown-dep error naming the ghost, got: %s", w.Body.String())
	}
}

func TestBatchCreate_MilestoneSelfDepends(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	self := []string{"self-ref"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Milestone: "self-ref"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "self-ref", Title: "Self", DependsOn: &self},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "self") {
		t.Errorf("expected self-ref error, got: %s", w.Body.String())
	}
}

func TestBatchCreate_MilestoneDepends_KindNameFormRejected(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	mixed := []string{"idea/some-item"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Milestone: "mixed"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "mixed", Title: "Mixed", DependsOn: &mixed},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "kind/name") {
		t.Errorf("expected kind/name rejection, got: %s", w.Body.String())
	}
}

func TestBatchCreate_MilestoneCrossDeps_OrderIndependent(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)

	deps := []string{"alpha"}
	// Declare "omega" BEFORE "alpha" — alphabetical order would put "alpha"
	// first anyway, so to prove topological ordering matters we flip the
	// relationship: alpha depends on omega.
	alphaDeps := []string{"omega"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "a-item", Title: "A", Kind: "idea", Milestone: "alpha"},
			{Name: "o-item", Title: "O", Kind: "idea", Milestone: "omega"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "alpha", Title: "Alpha (depends on omega)", DependsOn: &alphaDeps},
			{Name: "omega", Title: "Omega"},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusCreated(t, w)

	if len(ia.createOrder) != 2 {
		t.Fatalf("expected 2 Create calls, got %d: %v", len(ia.createOrder), ia.createOrder)
	}
	if ia.createOrder[0] != "omega" || ia.createOrder[1] != "alpha" {
		t.Errorf("expected order [omega, alpha] (dep-before-dependent), got %v", ia.createOrder)
	}

	_ = deps // silence unused in future additions
}

func TestBatchCreate_MilestoneUpdate_OnPriorityChange(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)
	ia.snapshots["existing"] = MilestoneSnapshot{
		Name:     "existing",
		Title:    "Existing",
		Status:   "active",
		Priority: 5,
	}

	payload := batchCreateRequest{
		Preview: true,
		Items: []batchCreateItem{
			{Name: "item", Title: "Item", Kind: "idea", Milestone: "existing"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "existing", Title: "Existing", Priority: intPtr(2)},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if len(resp.Milestones) != 1 || resp.Milestones[0].Action != "update" {
		t.Fatalf("expected action=update for priority change, got %+v", resp.Milestones)
	}
	if resp.Milestones[0].Priority != 2 {
		t.Errorf("preview priority = %d, want 2", resp.Milestones[0].Priority)
	}
}

func TestBatchCreate_MilestoneUpdate_OnDepsChange(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)
	ia.snapshots["base"] = MilestoneSnapshot{Name: "base", Title: "Base", Status: "active"}
	ia.snapshots["follower"] = MilestoneSnapshot{
		Name:      "follower",
		Title:     "Follower",
		Status:    "active",
		DependsOn: []string{},
	}

	deps := []string{"base"}
	payload := batchCreateRequest{
		Preview: true,
		Items: []batchCreateItem{
			{Name: "b-item", Title: "B Item", Kind: "idea", Milestone: "base"},
			{Name: "f-item", Title: "F Item", Kind: "idea", Milestone: "follower"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "base", Title: "Base"},
			{Name: "follower", Title: "Follower", DependsOn: &deps},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	byName := make(map[string]batchCreateMilestoneResult, len(resp.Milestones))
	for _, r := range resp.Milestones {
		byName[r.Name] = r
	}
	if byName["follower"].Action != "update" {
		t.Errorf("expected follower action=update for deps change, got %q", byName["follower"].Action)
	}
	if byName["base"].Action != "reuse" {
		t.Errorf("expected base action=reuse (no change), got %q", byName["base"].Action)
	}
}

func TestBatchCreate_MilestoneReuse_WhenIdentical(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)
	ia.snapshots["same"] = MilestoneSnapshot{
		Name:      "same",
		Title:     "Same",
		Status:    "active",
		Priority:  4,
		DependsOn: []string{"dep1", "dep2"},
	}
	ia.snapshots["dep1"] = MilestoneSnapshot{Name: "dep1", Title: "D1", Status: "active"}
	ia.snapshots["dep2"] = MilestoneSnapshot{Name: "dep2", Title: "D2", Status: "active"}

	// Provide the same set of deps, but in reversed order to test set-equality.
	deps := []string{"dep2", "dep1"}
	payload := batchCreateRequest{
		Preview: true,
		Items: []batchCreateItem{
			{Name: "x-item", Title: "X Item", Kind: "idea", Milestone: "same"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "same", Title: "Same", Status: strPtr("active"), Priority: intPtr(4), DependsOn: &deps},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[batchCreateResponse](t, w)
	if len(resp.Milestones) != 1 || resp.Milestones[0].Action != "reuse" {
		t.Fatalf("expected action=reuse for identical spec, got %+v", resp.Milestones)
	}
}

func TestBatchCreate_MilestoneRollback_PreservesPriorityAndDeps(t *testing.T) {
	h, _, ia := setupBatchTestHandler(t)
	ia.snapshots["preserved"] = MilestoneSnapshot{
		Name:      "preserved",
		Title:     "Preserved",
		Status:    "active",
		Priority:  7,
		DependsOn: []string{"anchor"},
		Items:     []string{},
	}
	ia.snapshots["anchor"] = MilestoneSnapshot{Name: "anchor", Title: "Anchor", Status: "active"}

	// Force AddItems to fail after the milestone has been updated with new
	// priority, so rollback kicks in via Replace.
	ia.addErr = errStub("disk full")

	deps := []string{"anchor"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "rollback-item", Title: "RI", Kind: "idea", Milestone: "preserved"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "preserved", Title: "Preserved", Priority: intPtr(1), DependsOn: &deps},
		},
	}

	w := doBatchCreate(t, h, payload)
	testutil.AssertStatus(t, w, http.StatusInternalServerError)

	restored := ia.snapshots["preserved"]
	if restored.Priority != 7 {
		t.Errorf("rollback lost priority: got %d, want 7", restored.Priority)
	}
	if len(restored.DependsOn) != 1 || restored.DependsOn[0] != "anchor" {
		t.Errorf("rollback lost depends_on: got %v, want [anchor]", restored.DependsOn)
	}
}

func TestBatchCreate_MilestoneUnknownField_StillRejected(t *testing.T) {
	h, _, _ := setupBatchTestHandler(t)

	body := strings.NewReader(`{
		"items": [
			{"name":"item","title":"Item","kind":"idea","milestone":"bad"}
		],
		"milestones": [
			{"name":"bad","title":"Bad","bogus":"field"}
		]
	}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog/batch", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BatchCreate(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "unknown field") {
		t.Errorf("expected unknown-field error, got: %s", w.Body.String())
	}
}

// Sanity-check that the marshaled request round-trips when the caller encodes
// a *[]string depends_on field to JSON — the default encoder should produce
// a plain array, and decoding it back should not trip strict mode.
func TestBatchCreate_MilestoneDepends_JSONRoundTrip(t *testing.T) {
	deps := []string{"a", "b"}
	payload := batchCreateRequest{
		Items: []batchCreateItem{
			{Name: "i", Title: "I", Kind: "idea", Milestone: "x"},
		},
		Milestones: []batchCreateMilestone{
			{Name: "x", Title: "X", DependsOn: &deps},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"depends_on":["a","b"]`)) {
		t.Errorf("expected depends_on array in wire form, got: %s", raw)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
