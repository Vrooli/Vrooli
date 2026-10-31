package backlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/descendantbudget"
	"swarm-manager/internal/identity"

	"github.com/gorilla/mux"
)

func newEffortControlDescendantService(t *testing.T, effortID string, mutate func(*identity.EffortControl)) *EffortControlService {
	t.Helper()
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{effortID: {Source: "efforts/" + effortID + "/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{effortID: testEffortCandidate("opencode-go/model-one")},
	}
	service := NewEffortControlService(store, source).WithDescendantRoot(t.TempDir())
	control := testEffortControlRequest(effortID, 1)
	if mutate != nil {
		mutate(&control)
	}
	if _, err := service.Admit(control); err != nil {
		t.Fatalf("admit: %v", err)
	}
	return service
}

func TestAdmitDescendantUsesAdmittedAggregateGrant(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", nil)
	for i, id := range []string{"a", "b", "c"} {
		if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: id, Depth: 1}); err != nil {
			t.Fatalf("admit %d: %v", i, err)
		}
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "d", Depth: 1}); !errors.Is(err, descendantbudget.ErrCapacityExhausted) {
		t.Fatalf("expected capacity exhaustion from the aggregate grant, got %v", err)
	}
}

func TestAdmitDescendantEnforcesPremiumCap(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", nil)
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "p1", Depth: 1, Premium: true}); err != nil {
		t.Fatalf("admit premium: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "p2", Depth: 1, Premium: true}); !errors.Is(err, descendantbudget.ErrCapacityExhausted) {
		t.Fatalf("expected premium exhaustion, got %v", err)
	}
}

func TestAdmitDescendantEnforcesApprovedDepth(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", func(control *identity.EffortControl) {
		control.AggregateLimits.MaxDepth = 1
	})
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "child", Depth: 2}); !errors.Is(err, descendantbudget.ErrDepthExceeded) {
		t.Fatalf("expected depth refusal from the aggregate grant, got %v", err)
	}
}

func TestAdmitNestedDescendantRequiresActiveParent(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", nil)
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "planner", Depth: 1}); err != nil {
		t.Fatalf("admit parent: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "worker", ParentAttemptID: "planner", Depth: 2}); err != nil {
		t.Fatalf("admit nested: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "orphan", ParentAttemptID: "missing", Depth: 2}); !errors.Is(err, descendantbudget.ErrParentNotActive) {
		t.Fatalf("expected parent refusal, got %v", err)
	}
}

func TestReleaseDescendantReturnsSlot(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", func(control *identity.EffortControl) {
		control.AggregateLimits.MaxActiveDescendants = 1
		control.AggregateLimits.MaxConcurrency = 1
		control.AggregateLimits.MaxWorkers = 1
	})
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "a", Depth: 1}); err != nil {
		t.Fatalf("admit: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "b", Depth: 1}); !errors.Is(err, descendantbudget.ErrCapacityExhausted) {
		t.Fatalf("expected capacity exhaustion, got %v", err)
	}
	if _, err := service.ReleaseDescendant("aquila-one", "a"); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "b", Depth: 1}); err != nil {
		t.Fatalf("admit after release: %v", err)
	}
}

func TestDescendantRefusesUnknownEffort(t *testing.T) {
	service := newEffortControlDescendantService(t, "aquila-one", nil)
	if _, err := service.AdmitDescendant("aquila-unknown", descendantbudget.Reservation{AttemptID: "a", Depth: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not-found for an unknown effort, got %v", err)
	}
}

func TestDescendantAccountingMustBeConfigured(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	service := NewEffortControlService(store, source)
	if _, err := service.Admit(testEffortControlRequest("aquila-one", 1)); err != nil {
		t.Fatalf("admit: %v", err)
	}
	if _, err := service.AdmitDescendant("aquila-one", descendantbudget.Reservation{AttemptID: "a", Depth: 1}); !errors.Is(err, ErrDescendantAccountingUnavailable) {
		t.Fatalf("expected unavailable descendant accounting, got %v", err)
	}
}

func doEffortDescendantJSON(t *testing.T, handler http.HandlerFunc, method, effortID, attemptID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = encoded
	} else {
		payload = []byte("{}")
	}
	req := httptest.NewRequest(method, "/api/v1/efforts/"+effortID+"/descendants", bytes.NewReader(payload))
	vars := map[string]string{"effortID": effortID}
	if attemptID != "" {
		vars["attemptID"] = attemptID
	}
	req = mux.SetURLVars(req, vars)
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func TestEffortDescendantAdmitListReleaseOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithDescendantRoot(t.TempDir())
	h.SetEffortControlService(service)
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}

	admit := doEffortDescendantJSON(t, h.AdmitEffortDescendant, http.MethodPost, "aquila-one", "", effortDescendantAdmitRequest{AttemptID: "w147-1", Depth: 1})
	if admit.Code != http.StatusOK {
		t.Fatalf("admit descendant: %d %s", admit.Code, admit.Body.String())
	}
	list := doEffortDescendantJSON(t, h.ListEffortDescendants, http.MethodGet, "aquila-one", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	var decoded effortDescendantListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if decoded.Totals.Active != 1 || decoded.Totals.Limit.MaxActiveDescendants != 3 {
		t.Fatalf("unexpected totals: %+v", decoded.Totals)
	}
	release := doEffortDescendantJSON(t, h.ReleaseEffortDescendant, http.MethodPost, "aquila-one", "w147-1", nil)
	if release.Code != http.StatusOK {
		t.Fatalf("release: %d %s", release.Code, release.Body.String())
	}
	missing := doEffortDescendantJSON(t, h.ReleaseEffortDescendant, http.MethodPost, "aquila-one", "does-not-exist", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unreserved descendant, got %d %s", missing.Code, missing.Body.String())
	}
}

func TestEffortDescendantCapacityRefusedOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithDescendantRoot(t.TempDir())
	h.SetEffortControlService(service)
	control := testEffortControlRequest("aquila-one", 1)
	control.AggregateLimits.MaxActiveDescendants = 1
	control.AggregateLimits.MaxConcurrency = 1
	control.AggregateLimits.MaxWorkers = 1
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", control); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}
	if recorder := doEffortDescendantJSON(t, h.AdmitEffortDescendant, http.MethodPost, "aquila-one", "", effortDescendantAdmitRequest{AttemptID: "a", Depth: 1}); recorder.Code != http.StatusOK {
		t.Fatalf("first admit: %d %s", recorder.Code, recorder.Body.String())
	}
	refused := doEffortDescendantJSON(t, h.AdmitEffortDescendant, http.MethodPost, "aquila-one", "", effortDescendantAdmitRequest{AttemptID: "b", Depth: 1})
	if refused.Code != http.StatusConflict {
		t.Fatalf("expected 409 at the descendant cap, got %d %s", refused.Code, refused.Body.String())
	}
}
