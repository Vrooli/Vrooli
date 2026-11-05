package backlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/identity"
	"swarm-manager/internal/providerpool"

	"github.com/gorilla/mux"
)

func newEffortControlProviderService(t *testing.T, effortID string, mutate func(*identity.EffortControl)) *EffortControlService {
	t.Helper()
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{effortID: {Source: "efforts/" + effortID + "/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{effortID: testEffortCandidate("opencode-go/model-one")},
	}
	service := NewEffortControlService(store, source).WithProviderPoolRoot(t.TempDir())
	control := testEffortControlRequest(effortID, 1)
	if mutate != nil {
		mutate(&control)
	}
	if _, err := service.Admit(control); err != nil {
		t.Fatalf("admit: %v", err)
	}
	return service
}

func TestReserveProviderSlotUsesAdmittedAggregateGrant(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	for i := 0; i < 3; i++ {
		if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{
			AttemptID: string(rune('a' + i)),
			Pool:      "opencode-go",
			Units:     1,
			Known:     true,
		}); err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{
		AttemptID: "d",
		Pool:      "opencode-go",
		Units:     1,
		Known:     true,
	}); !errors.Is(err, providerpool.ErrPoolSaturated) {
		t.Fatalf("expected saturation from the aggregate concurrency grant, got %v", err)
	}
}

func TestReserveProviderSlotEnforcesAggregateUnits(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", func(control *identity.EffortControl) {
		control.AggregateLimits.MaxTokens = 10
	})
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "a", Pool: "opencode-go", Units: 6, Known: true}); err != nil {
		t.Fatalf("reserve a: %v", err)
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "b", Pool: "opencode-go", Units: 6, Known: true}); !errors.Is(err, providerpool.ErrAllowanceExhausted) {
		t.Fatalf("expected shared-allowance exhaustion, got %v", err)
	}
}

func TestProviderPoolRecoveryGatesReservation(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	state, err := service.ObserveProviderLimit("aquila-one", providerpool.Observation{
		Pool:   "codex",
		Class:  providerpool.ClassSubscriptionWeekly,
		Source: "account-status",
	})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if state.Eligible || !state.RequiresObservation {
		t.Fatalf("unknown weekly reset must pause the pool pending observation: %+v", state)
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "a1", Pool: "codex", Known: true}); !errors.Is(err, providerpool.ErrPoolBlocked) {
		t.Fatalf("expected blocked admission, got %v", err)
	}
	state, err = service.ObserveProviderLimit("aquila-one", providerpool.Observation{
		Pool:   "codex",
		Class:  providerpool.ClassRecovered,
		Source: "owner-wake",
	})
	if err != nil || !state.Eligible {
		t.Fatalf("explicit recovery must re-enable the pool: %+v err=%v", state, err)
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "a1", Pool: "codex", Known: true}); err != nil {
		t.Fatalf("reserve after recovery: %v", err)
	}
}

func TestProviderReservationRefusesUnknownEffort(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	if _, err := service.ReserveProviderSlot("aquila-unknown", providerpool.Reservation{AttemptID: "a1", Pool: "opencode-go", Known: true}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not-found for an unknown effort, got %v", err)
	}
}

func TestProviderAccountingMustBeConfigured(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	service := NewEffortControlService(store, source)
	if _, err := service.Admit(testEffortControlRequest("aquila-one", 1)); err != nil {
		t.Fatalf("admit: %v", err)
	}
	if _, err := service.ProviderPoolState("aquila-one", "opencode-go"); !errors.Is(err, ErrProviderAccountingUnavailable) {
		t.Fatalf("expected unavailable provider accounting, got %v", err)
	}
	if _, err := service.RefreshProviderPoolLimits("aquila-one"); !errors.Is(err, ErrProviderAccountingUnavailable) {
		t.Fatalf("expected unavailable provider-pool refresh, got %v", err)
	}
}

func TestObserveProviderFailureTranslatesAndBlocksPool(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	state, err := service.ObserveProviderFailure("aquila-one", providerpool.ProviderFailure{
		Pool: "openrouter", Class: providerpool.GatewayFailureRateLimited,
		HTTPStatus: 429, RetryAfter: "12", Source: "http:429",
	})
	if err != nil {
		t.Fatalf("observe provider failure: %v", err)
	}
	if state.Eligible || state.Class != providerpool.ClassRateLimit {
		t.Fatalf("rate_limited must pause the pool as %s: %+v", providerpool.ClassRateLimit, state)
	}
	if state.Source != "http:429" {
		t.Fatalf("observed source not carried through: %+v", state)
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "f1", Pool: "openrouter", Known: true}); !errors.Is(err, providerpool.ErrPoolBlocked) {
		t.Fatalf("expected blocked admission after a rate limit, got %v", err)
	}
}

func TestObserveProviderFailureIgnoresNonPoolCondition(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	state, err := service.ObserveProviderFailure("aquila-one", providerpool.ProviderFailure{
		Pool: "openrouter", Class: "timeout",
	})
	if err != nil {
		t.Fatalf("observe provider failure: %v", err)
	}
	if !state.Eligible {
		t.Fatalf("a per-attempt execution fault must not pause the shared pool: %+v", state)
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "f1", Pool: "openrouter", Known: true}); err != nil {
		t.Fatalf("reserve after a non-pool failure: %v", err)
	}
}

func TestObserveProviderFailureRefusesUnknownEffort(t *testing.T) {
	service := newEffortControlProviderService(t, "aquila-one", nil)
	if _, err := service.ObserveProviderFailure("aquila-unknown", providerpool.ProviderFailure{
		Pool: "openrouter", Class: providerpool.GatewayFailureInsufficientCredits,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not-found for an unknown effort, got %v", err)
	}
}

func TestRefreshProviderPoolLimitsRefreshesOnlyAttributedPools(t *testing.T) {
	providerRoot := t.TempDir()
	source := &stubEffortPolicySource{
		bindings: map[string]identity.PolicyBinding{
			"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"},
			"aquila-two": {Source: "efforts/aquila-two/effort.json", Digest: "sha256:two"},
		},
		candidates: map[string]identity.CandidatePolicyBinding{
			"aquila-one": testEffortCandidate("opencode-go/model-one"),
			"aquila-two": testEffortCandidate("opencode-go/model-two"),
		},
	}
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).WithProviderPoolRoot(providerRoot)
	for _, effortID := range []string{"aquila-one", "aquila-two"} {
		if _, err := service.Admit(testEffortControlRequest(effortID, 1)); err != nil {
			t.Fatalf("admit %s: %v", effortID, err)
		}
	}
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "one-1", Pool: "one-pool", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve one: %v", err)
	}
	if _, err := service.ReserveProviderSlot("aquila-two", providerpool.Reservation{AttemptID: "two-1", Pool: "two-pool", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve two: %v", err)
	}
	// A pool shared by both efforts is not attributable, so it is not refreshed.
	if _, err := service.ReserveProviderSlot("aquila-one", providerpool.Reservation{AttemptID: "one-2", Pool: "shared-pool", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve shared one: %v", err)
	}
	if _, err := service.ReserveProviderSlot("aquila-two", providerpool.Reservation{AttemptID: "two-2", Pool: "shared-pool", Units: 1, Known: true}); err != nil {
		t.Fatalf("reserve shared two: %v", err)
	}

	amended := testEffortControlRequest("aquila-one", 2)
	amended.AggregateLimits.MaxTokens = 7
	if _, err := service.Amend(amended); err != nil {
		t.Fatalf("amend: %v", err)
	}
	reports, err := service.RefreshProviderPoolLimits("aquila-one")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if len(reports) != 1 || reports[0].Pool != "one-pool" {
		t.Fatalf("only the pool owned by the amended effort may refresh: %+v", reports)
	}
	if reports[0].Limit.MaxUnits != 7 || reports[0].State.Limit.MaxUnits != 7 {
		t.Fatalf("amended unit bound not applied: %+v", reports[0])
	}
	shared, err := service.ProviderPoolState("aquila-one", "shared-pool")
	if err != nil {
		t.Fatalf("shared state: %v", err)
	}
	if shared.Limit.MaxUnits == 7 {
		t.Fatalf("a pool shared with another effort must not be rewritten: %+v", shared.Limit)
	}
	// The other effort's dedicated pool keeps its original bound.
	other, err := service.ProviderPoolState("aquila-two", "two-pool")
	if err != nil {
		t.Fatalf("other state: %v", err)
	}
	if other.Limit.MaxUnits != 0 {
		t.Fatalf("an unrelated effort's pool must not be rewritten: %+v", other.Limit)
	}
}

func doEffortProviderJSON(t *testing.T, handler http.HandlerFunc, effortID, pool, attemptID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/efforts/"+effortID+"/provider-pools/"+pool, bytes.NewReader(encoded))
	vars := map[string]string{"effortID": effortID, "pool": pool}
	if attemptID != "" {
		vars["attemptID"] = attemptID
	}
	req = mux.SetURLVars(req, vars)
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func TestEffortProviderReserveBlockSettleOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithProviderPoolRoot(t.TempDir())
	h.SetEffortControlService(service)
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}

	reserve := doEffortProviderJSON(t, h.ReserveEffortProviderSlot, "aquila-one", "opencode-go", "", effortProviderReserveRequest{AttemptID: "w146-1", Units: 2, Known: true})
	if reserve.Code != http.StatusOK {
		t.Fatalf("reserve: %d %s", reserve.Code, reserve.Body.String())
	}
	list := doEffortProviderJSON(t, h.ListEffortProviderReservations, "aquila-one", "opencode-go", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	settle := doEffortProviderJSON(t, h.SettleEffortProviderSlot, "aquila-one", "opencode-go", "w146-1", effortProviderSettleRequest{UsedUnits: 2, Known: true})
	if settle.Code != http.StatusOK {
		t.Fatalf("settle: %d %s", settle.Code, settle.Body.String())
	}

	blocked := doEffortProviderJSON(t, h.ObserveEffortProviderLimit, "aquila-one", "codex", "", providerpool.Observation{Class: providerpool.ClassAPICreditExhausted})
	if blocked.Code != http.StatusOK {
		t.Fatalf("observe: %d %s", blocked.Code, blocked.Body.String())
	}
	refused := doEffortProviderJSON(t, h.ReserveEffortProviderSlot, "aquila-one", "codex", "", effortProviderReserveRequest{AttemptID: "w146-2", Known: true})
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for a paused pool, got %d %s", refused.Code, refused.Body.String())
	}
}

func TestObserveProviderFailureOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithProviderPoolRoot(t.TempDir())
	h.SetEffortControlService(service)
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}

	// A per-attempt execution fault is accepted but does not pause the pool.
	ignored := doEffortProviderJSON(t, h.ObserveEffortProviderFailure, "aquila-one", "openrouter", "", providerpool.ProviderFailure{Class: "execution_error"})
	if ignored.Code != http.StatusOK {
		t.Fatalf("non-pool failure: %d %s", ignored.Code, ignored.Body.String())
	}
	var ignoredState effortProviderStateResponse
	if err := json.Unmarshal(ignored.Body.Bytes(), &ignoredState); err != nil {
		t.Fatalf("decode non-pool state: %v", err)
	}
	if !ignoredState.State.Eligible {
		t.Fatalf("non-pool failure must not pause the pool: %+v", ignoredState.State)
	}

	// An observed credit exhaustion pauses the pool and refuses a reservation.
	blocked := doEffortProviderJSON(t, h.ObserveEffortProviderFailure, "aquila-one", "openrouter", "", providerpool.ProviderFailure{
		Class: providerpool.GatewayFailureInsufficientCredits, HTTPStatus: 402, Source: "http:402",
	})
	if blocked.Code != http.StatusOK {
		t.Fatalf("observe failure: %d %s", blocked.Code, blocked.Body.String())
	}
	var blockedState effortProviderStateResponse
	if err := json.Unmarshal(blocked.Body.Bytes(), &blockedState); err != nil {
		t.Fatalf("decode blocked state: %v", err)
	}
	if blockedState.State.Eligible || blockedState.State.Class != providerpool.ClassAPICreditExhausted {
		t.Fatalf("insufficient credits must pause the pool: %+v", blockedState.State)
	}
	refused := doEffortProviderJSON(t, h.ReserveEffortProviderSlot, "aquila-one", "openrouter", "", effortProviderReserveRequest{AttemptID: "w157-1", Known: true})
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for a paused pool, got %d %s", refused.Code, refused.Body.String())
	}
}

func TestAmendEffortRefreshesProviderPoolBoundOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithProviderPoolRoot(t.TempDir())
	h.SetEffortControlService(service)
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}
	// Attribute the pool to this effort so the amendment refresh can find it.
	if reserve := doEffortProviderJSON(t, h.ReserveEffortProviderSlot, "aquila-one", "opencode-go", "", effortProviderReserveRequest{AttemptID: "w152-1", Units: 1, Known: true}); reserve.Code != http.StatusOK {
		t.Fatalf("reserve: %d %s", reserve.Code, reserve.Body.String())
	}

	amended := testEffortControlRequest("aquila-one", 2)
	amended.AggregateLimits.MaxTokens = 9
	recorder := doEffortJSON(t, h.AmendEffort, http.MethodPut, "aquila-one", amended)
	if recorder.Code != http.StatusOK {
		t.Fatalf("amend: %d %s", recorder.Code, recorder.Body.String())
	}
	var response effortControlResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode amend response: %v", err)
	}
	if len(response.ProviderPools) != 1 || response.ProviderPools[0].Pool != "opencode-go" {
		t.Fatalf("amend must report the refreshed pool: %+v", response.ProviderPools)
	}
	if response.ProviderPools[0].Limit.MaxUnits != 9 || response.ProviderPools[0].State.Limit.MaxUnits != 9 {
		t.Fatalf("refreshed bound not derived from the amended grant: %+v", response.ProviderPools[0])
	}
	if state, err := service.ProviderPoolState("aquila-one", "opencode-go"); err != nil || state.Limit.MaxUnits != 9 {
		t.Fatalf("persisted bound not refreshed: %+v err=%v", state.Limit, err)
	}

	// The explicit refresh endpoint is idempotent for an interrupted response.
	again := doEffortProviderJSON(t, h.RefreshEffortProviderPools, "aquila-one", "", "", nil)
	if again.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", again.Code, again.Body.String())
	}
}
