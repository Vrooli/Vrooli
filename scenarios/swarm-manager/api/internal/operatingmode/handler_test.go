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
