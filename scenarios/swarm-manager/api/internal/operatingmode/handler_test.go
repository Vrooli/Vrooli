package operatingmode

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestRoundActionWithoutModeRejectsItemLevelInitiative(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"item-init": {
				Name:  "item-init",
				Title: "Item Init",
				Mode:  string(ModeItemLevel),
			},
		}},
		backlogMutator: &fakeBacklogMutator{},
	})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/initiatives/item-init/operating-mode/rounds/1/complete-items", bytes.NewBufferString(`{
		"run_id": "run-1",
		"item_refs": ["execute/do-thing"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "item-level mode") {
		t.Fatalf("body = %q, want item-level mode error", rec.Body.String())
	}
}

func TestCatalogEndpointReturnsRegisteredModes(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	for _, mode := range []string{"item-level", "holistic-loop", "phased-plan-drain"} {
		if !strings.Contains(rec.Body.String(), mode) {
			t.Fatalf("catalog response missing %q: %s", mode, rec.Body.String())
		}
	}
	if !strings.Contains(rec.Body.String(), `"supports_phases":true`) {
		t.Fatalf("catalog response missing phase support metadata: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"description"`) {
		t.Fatalf("catalog response missing description field: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"usage_count"`) {
		t.Fatalf("catalog response missing usage_count field: %s", rec.Body.String())
	}
	// init-a is the default fakeInitiatives item bound to holistic-loop, so
	// the holistic-loop entry should report a non-zero usage count.
	if !strings.Contains(rec.Body.String(), `"usage_count":1`) {
		t.Fatalf("expected at least one mode to report usage_count=1: %s", rec.Body.String())
	}
}

func TestGetModeReturnsLinkedInitiatives(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"loop-init": {
				Name:  "loop-init",
				Title: "Loop Initiative",
				Mode:  string(ModeHolisticLoop),
			},
			"drain-init": {
				Name:  "drain-init",
				Title: "Drain Initiative",
				Mode:  string(ModePhasedPlanDrain),
			},
		}},
	})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/holistic-loop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "loop-init") {
		t.Fatalf("response missing linked initiative loop-init: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "drain-init") {
		t.Fatalf("response should not include initiatives from other modes: %s", rec.Body.String())
	}
}

func TestGetModeRejectsUnknown(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPatchModeAppliesOverlay(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":"Renamed Loop","description":"Tighter wording."}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/holistic-loop", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Renamed Loop") {
		t.Fatalf("response missing updated label: %s", rec.Body.String())
	}

	// A subsequent GET should reflect the persisted overlay.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/operating-modes/holistic-loop", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if !strings.Contains(getRec.Body.String(), "Renamed Loop") {
		t.Fatalf("GET after PATCH did not return updated label: %s", getRec.Body.String())
	}
}

func TestPatchModeRejectsBlankLabel(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":""}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/holistic-loop", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestPatchModeRejectsUnknownMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	body := bytes.NewBufferString(`{"label":"x"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/operating-modes/does-not-exist", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoundActionWithoutModeUsesCurrentNonDefaultInitiativeMode(t *testing.T) {
	root := t.TempDir()
	mutator := &fakeBacklogMutator{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{backlogMutator: mutator})
	_, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		InitiativeName: "init-a",
		ScopeID:        "init-a",
		Phase:          "execute",
		Status:         RoundStatusCompleted,
		RunID:          "run-1",
		Items:          []RoundItem{{Ref: "execute/do-thing"}},
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/initiatives/init-a/operating-mode/rounds/1/complete-items", bytes.NewBufferString(`{
		"run_id": "run-1",
		"item_refs": ["execute/do-thing"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(mutator.completed) != 1 || mutator.completed[0] != "execute/do-thing@initiative.operating_mode.complete_items" {
		t.Fatalf("completed = %#v, want operating-mode completion", mutator.completed)
	}
}

func TestServiceRoundActionsRequireNonDefaultMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{backlogMutator: &fakeBacklogMutator{}})

	_, err := svc.CompleteItems(context.Background(), CompleteItemsRequest{
		InitiativeName: "init-a",
		Round:          1,
		RunID:          "run-1",
		ItemRefs:       []string{"execute/do-thing"},
	})
	if err == nil || !strings.Contains(err.Error(), "mode is required") {
		t.Fatalf("blank mode error = %v, want required mode error", err)
	}

	_, err = svc.CancelRound(context.Background(), "init-a", ModeItemLevel, 1)
	if err == nil || !strings.Contains(err.Error(), "item-level mode") {
		t.Fatalf("item-level cancel error = %v, want item-level mode error", err)
	}
}
