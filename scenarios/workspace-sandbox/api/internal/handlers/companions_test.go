package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

type companionStore struct {
	sandbox      *types.Sandbox
	failNextSave bool
	version      int64
}

func newCompanionStore() *companionStore {
	return &companionStore{sandbox: &types.Sandbox{ID: uuid.New(), Status: types.StatusActive, Metadata: map[string]interface{}{}}}
}

func (s *companionStore) Get(context.Context, uuid.UUID) (*types.Sandbox, error) {
	if s.sandbox == nil {
		return nil, errors.New("sandbox not found")
	}
	data, _ := json.Marshal(s.sandbox)
	var copy types.Sandbox
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}

func (s *companionStore) UpdateWithVersionCheck(_ context.Context, value *types.Sandbox, expected int64) error {
	if s.failNextSave {
		s.failNextSave = false
		return errors.New("synthetic version conflict")
	}
	if expected != s.sandbox.Version {
		return errors.New("version conflict")
	}
	value.Version = expected + 1
	s.sandbox = value
	return nil
}

func (s *companionStore) LogAuditEvent(context.Context, *types.AuditEvent) error { return nil }

type companionProvider struct {
	starts int
	stops  int
}

func (p *companionProvider) Start(context.Context, uuid.UUID, handlers.CompanionServiceSpec) (handlers.CompanionRuntime, error) {
	p.starts++
	return handlers.CompanionRuntime{ID: "runtime-1", Address: "http://crm.internal"}, nil
}

func (p *companionProvider) Stop(context.Context, uuid.UUID, handlers.CompanionServiceState) error {
	p.stops++
	return nil
}

func withCompanionStore(store *companionStore) liveOpt {
	return func(h *handlers.Handlers) { h.CompanionStore = store }
}

func withCompanionProvider(provider handlers.CompanionServiceAPI) liveOpt {
	return func(h *handlers.Handlers) { h.CompanionServices = provider }
}

func TestCompanionServiceDeclarationAndLifecycle(t *testing.T) {
	store := newCompanionStore()
	provider := &companionProvider{}
	live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store), withCompanionProvider(provider))
	id := store.sandbox.ID

	resp, body := live.DoJSON(t, http.MethodPut, sandboxesPath(id, "/companion-services"),
		`{"services":[{"name":"crm","kind":"http-mock","version":"1.0.0","options":{"fixture":"contacts-v1"}}]}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("declare status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodGet, sandboxesPath(id, "/companion-services"), "")
	if resp.StatusCode != http.StatusOK || !json.Valid(body) {
		t.Fatalf("list status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(id, "/companion-services/crm/start"), "{}")
	if resp.StatusCode != http.StatusOK || provider.starts != 1 {
		t.Fatalf("start status = %d, starts = %d; body=%s", resp.StatusCode, provider.starts, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(id, "/companion-services/crm/stop"), "{}")
	if resp.StatusCode != http.StatusOK || provider.stops != 1 {
		t.Fatalf("stop status = %d, stops = %d; body=%s", resp.StatusCode, provider.stops, body)
	}
}

func TestCompanionServiceDeclarationRejectsUnboundedFields(t *testing.T) {
	for _, input := range []string{
		`{"services":[{"name":"crm","kind":"custom","version":"1.0.0"}]}`,
		`{"services":[{"name":"crm","kind":"http-mock","version":"latest"}]}`,
		`{"services":[{"name":"crm","kind":"http-mock","version":"1.0.0","image":"attacker/image"}]}`,
		`{"services":[{"name":"crm","kind":"http-mock","version":"1.0.0","options":{"fixture":"contacts","token":"secret"}}]}`,
	} {
		store := newCompanionStore()
		live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store))
		resp, _ := live.DoJSON(t, http.MethodPut, sandboxesPath(store.sandbox.ID, "/companion-services"), input)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("input %s status = %d, want 400", input, resp.StatusCode)
		}
	}
}

func TestCompanionServiceStartFailsClosedWithoutProvider(t *testing.T) {
	store := newCompanionStore()
	store.sandbox.Metadata["companionServices"] = []handlers.CompanionServiceState{{
		Spec: handlers.CompanionServiceSpec{Name: "crm", Kind: "http-mock", Version: "1.0.0"}, Status: "declared",
	}}
	live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store))
	resp, body := live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/crm/start"), "{}")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("start status = %d, want 503; body=%s", resp.StatusCode, body)
	}
}

func TestCompanionServiceStartCompensatesOnVersionConflict(t *testing.T) {
	store := newCompanionStore()
	store.sandbox.Metadata["companionServices"] = []handlers.CompanionServiceState{{
		Spec: handlers.CompanionServiceSpec{Name: "crm", Kind: "http-mock", Version: "1.0.0"}, Status: "declared",
	}}
	store.failNextSave = true
	provider := &companionProvider{}
	live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store), withCompanionProvider(provider))
	resp, body := live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/crm/start"), "{}")
	if resp.StatusCode != http.StatusConflict || provider.starts != 1 || provider.stops != 1 {
		t.Fatalf("start status = %d, starts = %d, stops = %d; body=%s", resp.StatusCode, provider.starts, provider.stops, body)
	}
}

var _ sandbox.ServiceAPI = (*sandboxiface.FakeService)(nil)
