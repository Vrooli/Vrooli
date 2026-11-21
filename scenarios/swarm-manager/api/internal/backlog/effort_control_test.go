package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/identity"

	"github.com/gorilla/mux"
)

type stubEffortPolicySource struct {
	bindings   map[string]identity.PolicyBinding
	candidates map[string]identity.CandidatePolicyBinding
}

func (source stubEffortPolicySource) ResolveEffortPolicy(effortID string) (EffortPolicyResolution, error) {
	binding, ok := source.bindings[effortID]
	if !ok {
		return EffortPolicyResolution{}, fmt.Errorf("no approved policy for %s", effortID)
	}
	candidate, ok := source.candidates[effortID]
	if !ok {
		return EffortPolicyResolution{}, fmt.Errorf("no candidate policy for %s", effortID)
	}
	return EffortPolicyResolution{Binding: binding, Candidate: candidate}, nil
}

func testEffortCandidate(model string) identity.CandidatePolicyBinding {
	return identity.CandidatePolicyBinding{
		EconomicalRunner: "opencode",
		EconomicalModel:  model,
		EconomicalEffort: "runner-native",
		FallbackRunner:   "codex",
		FallbackModel:    "gpt-5.6-luna",
	}.Bind()
}

func testEffortControlRequest(effortID string, revision int64) identity.EffortControl {
	return identity.EffortControl{
		EffortID:         effortID,
		Slug:             effortID,
		Revision:         revision,
		FamilyID:         "family-1",
		DelegatedActions: []string{identity.DelegatedActionAuthorChildPlans, identity.DelegatedActionDispatch, identity.DelegatedActionRepair},
		Scope: identity.EffortScope{
			Allow: []string{"scenarios/swarm-manager/**", "packages/proto/**"},
			Deny:  []string{"scenarios/swarm-manager/api/internal/auth/**"},
		},
		AggregateLimits: identity.AggregateLimits{
			MaxWorkers:            3,
			MaxConcurrency:        3,
			MaxDepth:              2,
			MaxActiveDescendants:  3,
			MaxPremiumDescendants: 1,
		},
		RepairLimits: identity.RepairLimits{
			PerFingerprint:         2,
			PerComponent:           4,
			PerEffort:              12,
			ComponentActiveMinutes: 90,
		},
		WorkReferences: []identity.WorkReference{
			{Owner: "plan-manager", Kind: "family", ID: "family-1", Role: "topology"},
		},
	}
}

func newEffortControlTestHandler(t *testing.T) (*Handler, *stubEffortPolicySource) {
	t.Helper()
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
	h := NewHandler(t.TempDir(), t.TempDir())
	h.SetEffortControlService(NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source))
	return h, source
}

func doEffortJSON(t *testing.T, handler http.HandlerFunc, method, effortID string, body any) *httptest.ResponseRecorder {
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
	req := httptest.NewRequest(method, "/api/v1/efforts/"+effortID, bytes.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"effortID": effortID})
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func doEffortJSONWithContext(t *testing.T, handler http.HandlerFunc, method, effortID string, body any, ctx context.Context) *httptest.ResponseRecorder {
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
	req := httptest.NewRequest(method, "/api/v1/efforts/"+effortID, bytes.NewReader(payload))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	req = mux.SetURLVars(req, map[string]string{"effortID": effortID})
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func decodeEffortResponse(t *testing.T, recorder *httptest.ResponseRecorder) effortControlResponse {
	t.Helper()
	var response effortControlResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %d %s: %v", recorder.Code, recorder.Body.String(), err)
	}
	return response
}

// [REQ:SWM-P0-004] effort admission is version-checked and binds live policy.
func TestEffortControlAdmitBindsLivePolicy(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1))
	if recorder.Code != http.StatusOK {
		t.Fatalf("admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	admitted := decodeEffortResponse(t, recorder).Effort
	if admitted.PolicyBinding.Digest != "sha256:one" {
		t.Fatalf("admission did not bind the live policy: %+v", admitted.PolicyBinding)
	}
	if admitted.CandidatePolicy.Digest != testEffortCandidate("opencode-go/model-one").Digest {
		t.Fatalf("admission did not bind the resolved candidate policy: %+v", admitted.CandidatePolicy)
	}
	if admitted.Completion.EvidenceComplete || admitted.Completion.HumanAccepted {
		t.Fatalf("admission started with completion standing set: %+v", admitted.Completion)
	}
}

// [REQ:SWM-P0-004] admitting over an existing effort is refused.
func TestEffortControlAdmitOverExistingConflicts(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("first admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusConflict {
		t.Fatalf("second admit should conflict, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

// [REQ:SWM-P0-004] a changed grant requires the exact next revision.
func TestEffortControlAmendRequiresExactNextRevision(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	skipped := testEffortControlRequest("aquila-one", 3)
	skipped.DelegatedActions = []string{identity.DelegatedActionRepair}
	if recorder := doEffortJSON(t, h.AmendEffort, http.MethodPut, "aquila-one", skipped); recorder.Code != http.StatusConflict {
		t.Fatalf("skipped revision should conflict, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	next := testEffortControlRequest("aquila-one", 2)
	next.DelegatedActions = []string{identity.DelegatedActionRepair}
	recorder := doEffortJSON(t, h.AmendEffort, http.MethodPut, "aquila-one", next)
	if recorder.Code != http.StatusOK {
		t.Fatalf("amend status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	amended := decodeEffortResponse(t, recorder).Effort
	if amended.Revision != 2 || amended.Authorizes(identity.DelegatedActionDispatch) {
		t.Fatalf("amendment did not apply the narrowed grant: %+v", amended)
	}
}

// [REQ:SWM-P0-004] two efforts retain independent revisions through the API.
func TestEffortControlTwoEffortsIsolatedThroughAPI(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit one status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-two", testEffortControlRequest("aquila-two", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit two status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	oneBefore := decodeEffortResponse(t, doEffortJSON(t, h.GetEffort, http.MethodGet, "aquila-one", nil)).Effort
	two := testEffortControlRequest("aquila-two", 2)
	two.Scope.Allow = []string{"scenarios/content-desk/**"}
	if recorder := doEffortJSON(t, h.AmendEffort, http.MethodPut, "aquila-two", two); recorder.Code != http.StatusOK {
		t.Fatalf("amend two status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	oneAfter := decodeEffortResponse(t, doEffortJSON(t, h.GetEffort, http.MethodGet, "aquila-one", nil)).Effort
	if oneBefore.AuthorityDigest() != oneAfter.AuthorityDigest() {
		t.Fatal("amending a second effort mutated the first effort's authority")
	}
	if oneAfter.PolicyBinding.Digest != "sha256:one" {
		t.Fatalf("first effort lost its own policy binding: %+v", oneAfter.PolicyBinding)
	}
}

// [REQ:SWM-P0-004] evidence completion never appears as human acceptance.
func TestEffortControlCompletionSeparatesEvidenceFromHumanAcceptance(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	evidence := doEffortJSON(t, h.MarkEffortEvidenceComplete, http.MethodPost, "aquila-one", effortCompletionEvidenceRequest{Refs: []string{"evidence/wake.md"}})
	if evidence.Code != http.StatusOK {
		t.Fatalf("evidence status=%d body=%s", evidence.Code, evidence.Body.String())
	}
	completed := decodeEffortResponse(t, evidence).Effort
	if !completed.Completion.EvidenceComplete || completed.Completion.HumanAccepted || completed.Completion.HumanActor != "" {
		t.Fatalf("evidence completion leaked into human acceptance: %+v", completed.Completion)
	}

	// The owner refuses a truly blank actor even though the transport derives
	// the local operator identity from request provenance when none is sent.
	if _, err := h.effortControl.HumanAccept("aquila-one", "   "); !errors.Is(err, identity.ErrHumanAcceptanceRequired) {
		t.Fatalf("blank actor acceptance should be refused by the owner, got %v", err)
	}
	accepted := doEffortJSON(t, h.AcceptEffortCompletion, http.MethodPost, "aquila-one", effortCompletionAcceptRequest{Actor: "operator"})
	if accepted.Code != http.StatusOK {
		t.Fatalf("human acceptance status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	finalControl := decodeEffortResponse(t, accepted).Effort
	if !finalControl.Completion.HumanAccepted || finalControl.Completion.HumanActor != "operator" {
		t.Fatalf("human acceptance was not recorded: %+v", finalControl.Completion)
	}
	if finalControl.AuthorityDigest() != completed.AuthorityDigest() {
		t.Fatal("human acceptance changed the effort authority digest")
	}
}

// Durable persistence survives a new service over the same store.
func TestEffortControlPersistsAcrossServiceReload(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	first := NewEffortControlService(store, source)
	if _, err := first.Admit(testEffortControlRequest("aquila-one", 1)); err != nil {
		t.Fatalf("admit: %v", err)
	}

	reloaded := NewEffortControlService(NewFileEffortControlStore(store.rootDir), source)
	control, err := reloaded.Get("aquila-one")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if control.Revision != 1 || control.PolicyBinding.Digest != "sha256:one" {
		t.Fatalf("reloaded revision lost authority: %+v", control)
	}
}

// The live effort workspace binds the approved policy digest and the resolved
// candidate policy, and a policy edit changes the digest.
func TestFileEffortPolicySourceBindsApprovedRevision(t *testing.T) {
	root := t.TempDir()
	effortDir := filepath.Join(root, "aquila-one")
	if err := os.MkdirAll(effortDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeEffortJSON := func(policy string) {
		document := fmt.Sprintf(`{"slug":"aquila-one","updated_at":"2026-09-13T00:00:00Z","policy":%s}`, policy)
		if err := os.WriteFile(filepath.Join(effortDir, "effort.json"), []byte(document), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeEffortJSON(`{"repair_limits":{"fingerprint":2},"model_selection":{"worker_preference":{"runner":"opencode","model":"opencode-go/model-one","effort":"runner-native","fallback_runner":"codex","fallback_model":"gpt-5.6-luna"}}}`)

	source := FileEffortPolicySource{Root: root}
	resolution, err := source.ResolveEffortPolicy("aquila-one")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	binding, candidate := resolution.Binding, resolution.Candidate
	if binding.Digest == "" || !strings.HasPrefix(binding.Digest, "sha256:") {
		t.Fatalf("policy digest missing: %+v", binding)
	}
	if candidate.EconomicalRunner != "opencode" || candidate.EconomicalModel != "opencode-go/model-one" || candidate.FallbackModel != "gpt-5.6-luna" {
		t.Fatalf("candidate policy did not bind the workspace preference: %+v", candidate)
	}
	if err := candidate.Validate(); err != nil {
		t.Fatalf("bound candidate policy is invalid: %v", err)
	}

	firstDigest := binding.Digest
	writeEffortJSON(`{"repair_limits":{"fingerprint":4},"model_selection":{"worker_preference":{"runner":"opencode","model":"opencode-go/model-one","effort":"runner-native","fallback_runner":"codex","fallback_model":"gpt-5.6-luna"}}}`)
	updated, err := source.ResolveEffortPolicy("aquila-one")
	if err != nil {
		t.Fatalf("resolve after edit: %v", err)
	}
	if updated.Binding.Digest == firstDigest {
		t.Fatal("a changed approved policy retained the previous digest")
	}
}

// The approved effort.json repair limits are projected onto the admission
// payload with their attempt caps, so the cumulative waste bound comes from the
// reviewed workspace policy rather than a caller-supplied number.
func TestFileEffortPolicySourceProjectsRepairLimits(t *testing.T) {
	root := t.TempDir()
	effortDir := filepath.Join(root, "aquila-one")
	if err := os.MkdirAll(effortDir, 0o700); err != nil {
		t.Fatal(err)
	}
	document := `{"slug":"aquila-one","policy":{` +
		`"repair_limits":{"fingerprint":2,"component":4,"effort":12},` +
		`"repair_time_limits":{"component_active_minutes":90,"effort_fraction_of_approved_work_allowance":0.2},` +
		`"max_probe_attempts":2,"max_transport_attempts":2,"max_status_reads":2,` +
		`"model_selection":{"worker_preference":{"runner":"opencode","model":"opencode-go/model-one","effort":"runner-native","fallback_runner":"codex","fallback_model":"gpt-5.6-luna"}}}}`
	if err := os.WriteFile(filepath.Join(effortDir, "effort.json"), []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}

	resolution, err := FileEffortPolicySource{Root: root}.ResolveEffortPolicy("aquila-one")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !resolution.RepairLimitsPresent {
		t.Fatal("authored repair_limits were not projected")
	}
	want := identity.RepairLimits{
		PerFingerprint:         2,
		PerComponent:           4,
		PerEffort:              12,
		ComponentActiveMinutes: 90,
		EffortFraction:         0.2,
		ProbeAttempts:          2,
		TransportAttempts:      2,
		StatusReads:            2,
	}
	if resolution.RepairLimits != want {
		t.Fatalf("projected repair limits = %+v, want %+v", resolution.RepairLimits, want)
	}
}

// Admitting through the live policy source replaces a caller-supplied repair
// bound with the reviewed effort.json bound before validation.
func TestAdmitBindsReviewedRepairLimitsFromApprovedPolicy(t *testing.T) {
	root := t.TempDir()
	effortDir := filepath.Join(root, "aquila-one")
	if err := os.MkdirAll(effortDir, 0o700); err != nil {
		t.Fatal(err)
	}
	document := `{"slug":"aquila-one","policy":{` +
		`"repair_limits":{"fingerprint":3,"component":5,"effort":11},` +
		`"repair_time_limits":{"component_active_minutes":60,"effort_fraction_of_approved_work_allowance":0.5},` +
		`"max_probe_attempts":1,"max_transport_attempts":1,"max_status_reads":1,` +
		`"model_selection":{"worker_preference":{"runner":"opencode","model":"opencode-go/model-one","effort":"runner-native","fallback_runner":"codex","fallback_model":"gpt-5.6-luna"}}}}`
	if err := os.WriteFile(filepath.Join(effortDir, "effort.json"), []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewFileEffortControlStore(t.TempDir())
	service := NewEffortControlService(store, FileEffortPolicySource{Root: root})
	request := testEffortControlRequest("aquila-one", 1)
	request.RepairLimits = identity.RepairLimits{PerFingerprint: 1, PerComponent: 1, PerEffort: 1, ComponentActiveMinutes: 1}
	admitted, err := service.Admit(request)
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	want := identity.RepairLimits{
		PerFingerprint:         3,
		PerComponent:           5,
		PerEffort:              11,
		ComponentActiveMinutes: 60,
		EffortFraction:         0.5,
		ProbeAttempts:          1,
		TransportAttempts:      1,
		StatusReads:            1,
	}
	if admitted.RepairLimits != want {
		t.Fatalf("admitted repair limits = %+v, want reviewed %+v", admitted.RepairLimits, want)
	}
}

// A policy that authors no repair-limit block preserves the caller's reviewed
// bound instead of silently zeroing it to a value that could never validate.
func TestAdmitPreservesCallerRepairLimitsWhenPolicyOmitsThem(t *testing.T) {
	root := t.TempDir()
	effortDir := filepath.Join(root, "aquila-one")
	if err := os.MkdirAll(effortDir, 0o700); err != nil {
		t.Fatal(err)
	}
	document := `{"slug":"aquila-one","policy":{` +
		`"model_selection":{"worker_preference":{"runner":"opencode","model":"opencode-go/model-one","effort":"runner-native","fallback_runner":"codex","fallback_model":"gpt-5.6-luna"}}}}`
	if err := os.WriteFile(filepath.Join(effortDir, "effort.json"), []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewFileEffortControlStore(t.TempDir())
	service := NewEffortControlService(store, FileEffortPolicySource{Root: root})
	request := testEffortControlRequest("aquila-one", 1)
	caller := identity.RepairLimits{PerFingerprint: 2, PerComponent: 4, PerEffort: 12, ComponentActiveMinutes: 90}
	request.RepairLimits = caller
	admitted, err := service.Admit(request)
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if admitted.RepairLimits != caller {
		t.Fatalf("omitted policy repair limits did not preserve the caller bound: %+v", admitted.RepairLimits)
	}
}

// The owner routes are mounted on the registered backlog router.
func TestEffortControlRoutesRegistered(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)
	router := mux.NewRouter()
	h.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/efforts/aquila-one", nil)
	match := &mux.RouteMatch{}
	if !router.Match(req, match) {
		t.Fatal("effort GET route is not registered")
	}
}

// [REQ:SWM-P0-004] an unauthenticated blank-actor accept is refused and never
// mints product acceptance, even though request provenance defaults to the
// fail-open started-by operator attribution.
func TestEffortControlAcceptRefusesUnverifiedActor(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	refused := doEffortJSON(t, h.AcceptEffortCompletion, http.MethodPost, "aquila-one", effortCompletionAcceptRequest{})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("unauthenticated accept should be refused, got %d body=%s", refused.Code, refused.Body.String())
	}
	stored, err := h.effortControl.Get("aquila-one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.Completion.HumanAccepted || stored.Completion.HumanActor != "" {
		t.Fatalf("refused accept still recorded human acceptance: %+v", stored.Completion)
	}
}

// A verified operator provenance is the only implicit actor accepted; an agent
// or an absent verification is not a human act.
func TestEffortControlAcceptAllowsVerifiedOperatorProvenance(t *testing.T) {
	h, _ := newEffortControlTestHandler(t)

	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	verified := identity.NewContext(context.Background(), identity.Provenance{
		Actor:              identity.TypeOperator,
		VerificationStatus: identity.VerificationVerified,
	})
	accepted := doEffortJSONWithContext(t, h.AcceptEffortCompletion, http.MethodPost, "aquila-one", effortCompletionAcceptRequest{}, verified)
	if accepted.Code != http.StatusOK {
		t.Fatalf("verified operator accept status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	finalControl := decodeEffortResponse(t, accepted).Effort
	if !finalControl.Completion.HumanAccepted || finalControl.Completion.HumanActor != identity.TypeOperator {
		t.Fatalf("verified operator acceptance was not recorded: %+v", finalControl.Completion)
	}

	agent := identity.NewContext(context.Background(), identity.Provenance{
		Actor:              identity.TypeAgent,
		VerificationStatus: identity.VerificationVerified,
		RunID:              "run-1",
		ProfileKey:         "worker-1",
	})
	if recorder := doEffortJSONWithContext(t, h.AcceptEffortCompletion, http.MethodPost, "aquila-one", effortCompletionAcceptRequest{}, agent); recorder.Code != http.StatusBadRequest {
		t.Fatalf("verified agent accept should not be a human acceptance, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestChildGrantForPlanProjectsAdmittedAggregate(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	control := testEffortControlRequest("aquila-child", 1)
	control.AggregateLimits.MaxConcurrency = 2
	control.AggregateLimits.MaxActiveDescendants = 2
	control.AggregateLimits.MaxPremiumDescendants = 1
	control.AggregateLimits.MaxDepth = 1
	control.AggregateLimits.MaxWaitSeconds = 900
	control.WorkReferences = []identity.WorkReference{
		{Owner: "plan-manager", Kind: "plan", ID: "plan-abc", Role: "member"},
	}
	control.PolicyBinding = identity.PolicyBinding{Source: "efforts/aquila-child/effort.json", Digest: "sha256:policy"}
	control.CandidatePolicy = testEffortCandidate("opencode-go/model-child")
	if err := control.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(control); err != nil {
		t.Fatal(err)
	}
	service := NewEffortControlService(store, nil)

	grant, ok, err := service.ChildGrantForPlan("plan-abc")
	if err != nil || !ok {
		t.Fatalf("admitted plan did not resolve: %v ok=%v", err, ok)
	}
	if grant.EffortID != "aquila-child" || grant.MaxConcurrency != 2 || grant.MaxRecursion != 1 || grant.MaxWaitSeconds != 900 {
		t.Fatalf("aggregate child grant projection mismatch: %+v", grant)
	}
	if _, ok, err := service.ChildGrantForPlan("plan-missing"); err != nil || ok {
		t.Fatalf("unowned plan must not resolve: %v ok=%v", err, ok)
	}
	if _, ok, err := service.ChildGrantForPlan("  "); err != nil || ok {
		t.Fatalf("blank plan must not resolve: %v ok=%v", err, ok)
	}
}
