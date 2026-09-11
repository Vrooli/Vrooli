package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	"google.golang.org/protobuf/encoding/protojson"
)

type recordingApplyExecutor struct {
	mu     sync.Mutex
	calls  []string
	failOn string
}

func (e *recordingApplyExecutor) call(kind, name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, kind+":"+name)
	if e.failOn == kind+":"+name {
		return context.Canceled
	}
	return nil
}

func (e *recordingApplyExecutor) snapshotCalls() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.calls...)
}

func waitApplyTerminal(t *testing.T, id string) applyRun {
	t.Helper()
	// An apply run now computes the readiness verdict before it decides whether
	// the configuration marker may be written, and that verdict includes a
	// credential-authority diagnosis. The budget is generous because the point
	// of the wait is the terminal state, not the duration.
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if run, ok := applyRunSnapshot(id); ok && run.Status != "pending" && run.Status != "applying" {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("apply run %s did not reach a terminal state", id)
	return applyRun{}
}

func startApplyRequest(t *testing.T, server *Server) (*httptest.ResponseRecorder, applyRun) {
	t.Helper()
	startBody := prepareApplyRequest(t, server, "test-"+time.Now().Format("20060102150405.000000000"))
	w := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply", startBody)
	var response applyv1.StartApplyResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode start apply response: %v; body=%s", err, w.Body.String())
	}
	if response.GetRun() == nil {
		t.Fatalf("start apply response had no run: %s", w.Body.String())
	}
	return w, applyRunFromProto(response.GetRun())
}

func prepareApplyRequest(t *testing.T, server *Server, idempotencyKey string) string {
	t.Helper()
	planResponse := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan", `{"target":"local"}`)
	if planResponse.Code != http.StatusOK {
		t.Fatalf("get apply plan status = %d: %s", planResponse.Code, planResponse.Body.String())
	}
	var plan applyv1.GetApplyPlanResponse
	if err := protojson.Unmarshal(planResponse.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode apply plan: %v; body=%s", err, planResponse.Body.String())
	}
	reviewBody := `{"target":"local","planId":"` + plan.GetPlanId() + `","planDigest":"` + plan.GetPlanDigest() + `","expectedRevision":"` + plan.GetRevision() + `"}`
	reviewResponse := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply", reviewBody)
	if reviewResponse.Code != http.StatusOK {
		t.Fatalf("review apply status = %d: %s", reviewResponse.Code, reviewResponse.Body.String())
	}
	var review applyv1.ReviewApplyResponse
	if err := protojson.Unmarshal(reviewResponse.Body.Bytes(), &review); err != nil {
		t.Fatalf("decode apply review: %v; body=%s", err, reviewResponse.Body.String())
	}
	return `{"target":"local","planId":"` + review.GetPlanId() + `","planDigest":"` + review.GetPlanDigest() + `","expectedRevision":"` + review.GetRevision() + `","consentReceiptId":"` + review.GetConsentReceiptId() + `","idempotencyKey":"` + idempotencyKey + `"}`
}

func getApplyStatusRequest(t *testing.T, server *Server, id string) (*httptest.ResponseRecorder, applyRun) {
	t.Helper()
	w := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyRun", `{"runId":"`+id+`"}`)
	var response applyv1.GetApplyRunResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode apply status response: %v; body=%s", err, w.Body.String())
	}
	return w, applyRunFromProto(&response)
}

func applyRunFromProto(response *applyv1.GetApplyRunResponse) applyRun {
	result := applyRun{ID: response.GetRunId(), Status: response.GetLegacyStatus(), SelectionDigest: response.GetSelectionDigest(), Error: response.GetError(), DegradedDigest: response.GetDegradedDigest()}
	if result.Status == "" {
		result.Status = response.GetStatus().String()
	}
	for _, step := range response.GetSteps() {
		result.Items = append(result.Items, applyItemResult{applyItem: applyItem{ID: step.GetId(), Kind: step.GetKind(), Name: step.GetName(), Dependencies: step.GetDependencies(), Required: step.GetRequired(), Privileged: step.GetPrivileged(), State: step.GetObservedState()}, Outcome: step.GetLegacyOutcome(), Disposition: step.GetDisposition(), Error: step.GetError(), Remediation: step.GetRemediation(), BlockedBy: step.GetBlockedBy(), ErrorCode: step.GetErrorCode()})
	}
	for _, blocker := range response.GetBlockers() {
		result.Blockers = append(result.Blockers, completionBlocker{Kind: blocker.GetKind(), Name: blocker.GetName(), Reason: blocker.GetReason(), Remediation: blocker.GetRemediation()})
	}
	for _, blocker := range response.GetDegraded() {
		result.Degraded = append(result.Degraded, completionBlocker{Kind: blocker.GetKind(), Name: blocker.GetName(), Reason: blocker.GetReason(), Remediation: blocker.GetRemediation()})
	}
	return result
}

func (e *recordingApplyExecutor) InstallTool(_ context.Context, name string) error {
	return e.call("tool", name)
}

func (e *recordingApplyExecutor) ApplySafeguard(_ context.Context, name string) error {
	return e.call("safeguard", name)
}

func (e *recordingApplyExecutor) EnableResource(_ context.Context, name string) error {
	return e.call("resource", name)
}

func (e *recordingApplyExecutor) StartScenario(_ context.Context, name string) error {
	return e.call("scenario", name)
}

// useInProcessApplyRunner replaces the detached runner with an in-process one
// for behaviour tests.
//
// Production hands each accepted run to a separate process so the run survives
// this API being restarted by its own apply. A test does not want a real fork:
// it wants the executor's behaviour, deterministically, against its fixtures.
// The detachment itself is covered separately -- see TestApplyRunnerMode and
// TestObservedApplyRunReportsAnAbandonedRun -- so stubbing here does not leave
// the handoff untested.
func useInProcessApplyRunner(t *testing.T) {
	t.Helper()
	previous := spawnApplyRunner
	spawnApplyRunner = func(ctx context.Context, run applyRun) error {
		run.RunnerPID = os.Getpid()
		run.Heartbeat = operatorStateNow().UTC().Format(time.RFC3339)
		updateApplyRun(run)
		go executeApplyRun(ctx, run)
		return nil
	}
	t.Cleanup(func() { spawnApplyRunner = previous })
}

func TestV2ApplyOrdersDependenciesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	// Own the home and the external probes. Without this the readiness pass
	// inside apply shells out to the operator's real credential doctor, so the
	// verdict depends on whichever kopia repositories and paired devices happen
	// to exist on the developer's machine -- which is both non-deterministic
	// and order-dependent, because a neighbouring test stubs the same globals.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	stubExternalReadinessProbes(t)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"postgres":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres","hostTools":[],"hostSafeguards":[]}`)
	writeBootstrapMarker(t, root)
	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })
	useInProcessApplyRunner(t)

	w, first := startApplyRequest(t, NewServer())
	if w.Code != http.StatusOK {
		t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
	}
	firstTerminal := waitApplyTerminal(t, first.ID)
	if firstTerminal.Status != "applied" {
		t.Fatalf("first apply = %#v", firstTerminal)
	}
	calls := fake.snapshotCalls()
	if len(calls) != 2 || calls[0] != "resource:postgres" || calls[1] != "scenario:alpha" {
		t.Fatalf("calls = %#v", calls)
	}

	w, second := startApplyRequest(t, NewServer())
	if w.Code != http.StatusOK {
		t.Fatalf("second apply status = %d: %s", w.Code, w.Body.String())
	}
	if len(fake.snapshotCalls()) != 2 {
		t.Fatalf("second apply issued mutating calls: %#v", fake.snapshotCalls())
	}
	if second.Status != "already_satisfied" || !containsJSONString(w.Body.Bytes(), "already_satisfied") {
		t.Fatalf("second apply = %s", w.Body.String())
	}
	status, statusRun := getApplyStatusRequest(t, NewServer(), second.ID)
	if status.Code != http.StatusOK || statusRun.Status != "already_satisfied" {
		t.Fatalf("apply status = %d: %s", status.Code, status.Body.String())
	}
	missing := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyRun", `{"runId":"missing"}`)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing apply status = %d", missing.Code)
	}
}

func TestV2ApplyReplaysConsumedConsentWithSameIdempotencyKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	stubExternalReadinessProbes(t)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true}}`)
	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })
	useInProcessApplyRunner(t)

	server := NewServer()
	body := prepareApplyRequest(t, server, "replay-consent")
	firstResponse := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply", body)
	var first applyv1.StartApplyResponse
	if err := protojson.Unmarshal(firstResponse.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first apply response: %v; body=%s", err, firstResponse.Body.String())
	}
	if first.GetRun() == nil {
		t.Fatalf("first apply response had no run: %s", firstResponse.Body.String())
	}
	waitApplyTerminal(t, first.GetRun().GetRunId())

	secondResponse := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply", body)
	var second applyv1.StartApplyResponse
	if err := protojson.Unmarshal(secondResponse.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode replay response: %v; body=%s", err, secondResponse.Body.String())
	}
	if secondResponse.Code != http.StatusOK || second.GetRun() == nil || second.GetRun().GetRunId() != first.GetRun().GetRunId() {
		t.Fatalf("replay response = %d %s", secondResponse.Code, secondResponse.Body.String())
	}
	if calls := fake.snapshotCalls(); len(calls) > 1 {
		t.Fatalf("replay issued mutating calls: %#v", calls)
	}
}

func TestV2ApplySkipsDependentAfterFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	// Own the home and the external probes. Without this the readiness pass
	// inside apply shells out to the operator's real credential doctor, so the
	// verdict depends on whichever kopia repositories and paired devices happen
	// to exist on the developer's machine -- which is both non-deterministic
	// and order-dependent, because a neighbouring test stubs the same globals.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	stubExternalReadinessProbes(t)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"postgres":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres"}`)
	fake := &recordingApplyExecutor{failOn: "resource:postgres"}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })
	useInProcessApplyRunner(t)

	w, run := startApplyRequest(t, NewServer())
	if w.Code != http.StatusOK {
		t.Fatalf("apply = %d: %s", w.Code, w.Body.String())
	}
	terminal := waitApplyTerminal(t, run.ID)
	if terminal.Status != "partially_applied" || !containsJSONString(mustJSON(t, terminal), "blocked") {
		t.Fatalf("apply = %#v", terminal)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 {
		t.Fatalf("dependent scenario was executed: %#v", calls)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestApplyNeverRestartsItself pins the guard that keeps an apply run alive.
//
// vrooli-onboarding is a scenario like any other, so it appears in its own
// selection closure and therefore in its own apply plan. Executing that item
// means `vrooli scenario start vrooli-onboarding`, which stops the process
// running the apply: the run never reaches a terminal state, every item after
// it never executes, and the operator's wizard loses the API it is polling
// immediately after answering the consent prompt.
func TestApplyNeverRestartsItself(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	// Own the home and the external probes. Without this the readiness pass
	// inside apply shells out to the operator's real credential doctor, so the
	// verdict depends on whichever kopia repositories and paired devices happen
	// to exist on the developer's machine -- which is both non-deterministic
	// and order-dependent, because a neighbouring test stubs the same globals.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	stubExternalReadinessProbes(t)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"),
		`{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"`+onboardingScenarioName+`":{"enabled":true},"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", onboardingScenarioName, ".vrooli", "service.json"),
		`{"service":{"name":"`+onboardingScenarioName+`","system_required":true}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"),
		`{"service":{"name":"alpha","system_required":true}}`)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })
	useInProcessApplyRunner(t)

	w, started := startApplyRequest(t, NewServer())
	if w.Code != http.StatusOK {
		t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
	}
	run := waitApplyTerminal(t, started.ID)

	for _, call := range fake.snapshotCalls() {
		if call == "scenario:"+onboardingScenarioName {
			t.Fatalf("apply restarted the scenario serving the run; calls = %#v", fake.snapshotCalls())
		}
	}

	// Skipping silently would be its own defect: the operator must be able to
	// see that the item was not executed, and why.
	var found bool
	for _, item := range run.Items {
		if item.Kind != "scenario" || item.Name != onboardingScenarioName {
			continue
		}
		found = true
		if item.Outcome != "skipped_self" {
			t.Fatalf("self item outcome = %q, want skipped_self", item.Outcome)
		}
		if !strings.Contains(item.Remediation, "already running") {
			t.Fatalf("a skipped item must explain itself, got %q", item.Remediation)
		}
	}
	if !found {
		t.Fatal("the plan did not contain the onboarding scenario; this test no longer covers the self-restart path")
	}

	// The skip must not be laundered into a failure. The run may still end
	// short of "applied" on its readiness verdict -- that is a separate,
	// legitimate outcome -- but it must not be reported as partially applied,
	// which is the status reserved for an item that actually failed.
	if run.Status == "partially_applied" {
		t.Fatalf("a skipped self-item was counted as a failure; status = %q", run.Status)
	}
	for _, item := range run.Items {
		if item.Outcome == "blocked" && item.BlockedBy == "scenario:"+onboardingScenarioName {
			t.Fatalf("item %s was blocked by the skipped self-item", item.Name)
		}
	}
}
