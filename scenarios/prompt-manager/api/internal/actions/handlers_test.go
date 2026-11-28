package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func TestValidateHandler(t *testing.T) {
	actionStore := newFakeActionStore(validAction(nil))
	service := NewService(actionStore, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})
	handler := NewHandlers(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/team.swarm.work.list/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.list"})
	rr := httptest.NewRecorder()

	handler.Validate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result ValidationResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Valid || !result.Runnable {
		t.Fatalf("unexpected validation result: %#v", result)
	}
}

func TestValidateHandlerReturnsUnprocessableForInvalidAction(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Command.Argv = []string{"git", "status"}
	}))
	handler := NewHandlers(NewService(actionStore, nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/team.swarm.work.list/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.list"})
	rr := httptest.NewRecorder()

	handler.Validate(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestRunHandler(t *testing.T) {
	actionStore := newFakeActionStore(validAction(nil))
	service := NewService(actionStore, runnableResolver())
	service.runner = &stubRunner{result: CommandRunResult{ExitCode: 0, Stdout: "ok"}}
	handler := NewHandlers(service)

	body := bytes.NewBufferString(`{"input":{"identifier":"implementation-plan-authoring"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/team.swarm.work.list/run", body)
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.list"})
	rr := httptest.NewRecorder()

	handler.Run(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result RunResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusCompleted || result.Stdout != "ok" {
		t.Fatalf("unexpected run result: %#v", result)
	}
}

func TestRunHandlerReturnsUnprocessableForRejectedRun(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Status = store.StatusDraft
	}))
	handler := NewHandlers(NewService(actionStore, runnableResolver()))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/team.swarm.work.list/run", bytes.NewBufferString(`{}`))
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.list"})
	rr := httptest.NewRecorder()

	handler.Run(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestActionCRUDHandlers(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.ID = "team.swarm.work.list"
		action.Status = store.StatusDraft
		action.Pack = "drafts"
		action.Tags = []string{"team"}
	}))
	handler := NewHandlers(NewService(actionStore, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/actions?pack=drafts&status=draft&tag=team", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var listed []store.Action
	if err := json.NewDecoder(rr.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != "team.swarm.work.list" {
		t.Fatalf("unexpected list response: %#v", listed)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/actions/team.swarm.work.list", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.list"})
	rr = httptest.NewRecorder()
	handler.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", rr.Code, rr.Body.String())
	}

	createBody := mustJSON(t, CreateRequest{
		Pack: "drafts",
		Action: *validAction(func(action *store.Action) {
			action.ID = "team.swarm.work.create"
			action.Status = store.StatusDraft
		}),
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/actions", bytes.NewReader(createBody))
	rr = httptest.NewRecorder()
	handler.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rr.Code, rr.Body.String())
	}

	updateBody := mustJSON(t, validAction(func(action *store.Action) {
		action.ID = "team.swarm.work.create"
		action.Name = "Updated Decision Action"
		action.Status = store.StatusDraft
	}))
	req = httptest.NewRequest(http.MethodPut, "/api/v1/actions/team.swarm.work.create", bytes.NewReader(updateBody))
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.create"})
	rr = httptest.NewRecorder()
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/actions/team.swarm.work.create", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team.swarm.work.create"})
	rr = httptest.NewRecorder()
	handler.Delete(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete/archive status = %d, body = %s", rr.Code, rr.Body.String())
	}
	archived, _ := actionStore.Get(context.Background(), "team.swarm.work.create")
	if archived.Status != store.StatusArchived {
		t.Fatalf("DELETE without hard=true should archive, got %s", archived.Status)
	}
}

func TestCreateHandlerReturnsUnprocessableForValidationFailure(t *testing.T) {
	actionStore := newFakeActionStore()
	handler := NewHandlers(NewService(actionStore, NewManifestCommandResolver("")))
	body := mustJSON(t, CreateRequest{
		Pack: "drafts",
		Action: *validAction(func(action *store.Action) {
			action.ID = "team.swarm.work.bad"
			action.Status = store.StatusActive
			action.Command.Argv = []string{"prompt-manager", "not-a-real-command"}
		}),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(actionStore.actions) != 0 {
		t.Fatalf("invalid action should not be persisted")
	}
}

func TestActionHandlersReturnNotFoundAndInvalidJSON(t *testing.T) {
	handler := NewHandlers(NewService(newFakeActionStore(), stubResolver{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/actions/missing.action", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing.action"})
	rr := httptest.NewRecorder()
	handler.Get(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, body = %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/actions", bytes.NewBufferString("{"))
	rr = httptest.NewRecorder()
	handler.Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

type fakeActionStore struct {
	actions    map[string]*store.Action
	runHistory []store.ActionRunHistoryEntry
	err        error
}

func newFakeActionStore(actions ...*store.Action) *fakeActionStore {
	s := &fakeActionStore{actions: map[string]*store.Action{}}
	for _, action := range actions {
		copy := *action
		s.actions[action.ID] = &copy
	}
	return s
}

func (s *fakeActionStore) List(ctx context.Context) ([]store.Action, error) {
	if s.err != nil {
		return nil, s.err
	}
	var actions []store.Action
	for _, action := range s.actions {
		actions = append(actions, *action)
	}
	return actions, nil
}

func (s fakeActionStore) Get(ctx context.Context, id string) (*store.Action, error) {
	if s.err != nil {
		return nil, s.err
	}
	action, ok := s.actions[id]
	if !ok {
		return nil, fmt.Errorf("action not found: %s", id)
	}
	copy := *action
	return &copy, nil
}

func (s *fakeActionStore) Create(ctx context.Context, pack string, action *store.Action) error {
	if s.err != nil {
		return s.err
	}
	if _, exists := s.actions[action.ID]; exists {
		return fmt.Errorf("action already exists: %s", action.ID)
	}
	copy := *action
	copy.Pack = pack
	s.actions[action.ID] = &copy
	return nil
}

func (s *fakeActionStore) Update(ctx context.Context, id string, action *store.Action) error {
	if s.err != nil {
		return s.err
	}
	current, ok := s.actions[id]
	if !ok {
		return fmt.Errorf("action not found: %s", id)
	}
	copy := *action
	if copy.Pack == "" {
		copy.Pack = current.Pack
	}
	s.actions[id] = &copy
	return nil
}

func (s *fakeActionStore) Archive(ctx context.Context, id string) error {
	action, ok := s.actions[id]
	if !ok {
		return fmt.Errorf("action not found: %s", id)
	}
	action.Status = store.StatusArchived
	return nil
}

func (s *fakeActionStore) Delete(ctx context.Context, id string) error {
	if _, ok := s.actions[id]; !ok {
		return fmt.Errorf("action not found: %s", id)
	}
	delete(s.actions, id)
	return nil
}

func (s *fakeActionStore) AppendRunHistory(ctx context.Context, id string, entry store.ActionRunHistoryEntry) error {
	if _, ok := s.actions[id]; !ok {
		return fmt.Errorf("action not found: %s", id)
	}
	s.runHistory = append(s.runHistory, entry)
	return nil
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
