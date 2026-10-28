package validationdesktop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationprovider"
)

type fakeDesktop struct {
	startRecording bool
	stopRecording  bool
	launchOptions  target.ElectronLaunchOptions
	waitCalls      int
	restartCalls   int
	quitCalls      int
	launchCalls    int
	sessionIDs     []string
	waitError      error
}

func (f *fakeDesktop) StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error) {
	return &livedesktop.Session{ID: "session-1"}, nil
}
func (f *fakeDesktop) ExecuteAction(_ context.Context, _ string, action string, _ json.RawMessage) (*livedesktop.ActionResult, error) {
	if action == "start_recording" {
		f.startRecording = true
		return &livedesktop.ActionResult{Data: map[string]any{"capture_id": "recording-1"}}, nil
	}
	if action == "quit_app" {
		f.quitCalls++
		return &livedesktop.ActionResult{}, nil
	}
	f.stopRecording = true
	return &livedesktop.ActionResult{Data: map[string]any{"capture_id": "recording-1"}}, nil
}
func (f *fakeDesktop) LaunchElectronValidation(_ context.Context, sessionID, _ string, options target.ElectronLaunchOptions, _ target.RendererExpectation) (*domainv1.AppTarget, error) {
	f.launchOptions = options
	f.launchCalls++
	f.sessionIDs = append(f.sessionIDs, sessionID)
	return &domainv1.AppTarget{TargetId: "target-1", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer-1"}, nil
}
func (f *fakeDesktop) WaitElectronValidationExit(_ context.Context, sessionID string) error {
	f.waitCalls++
	f.sessionIDs = append(f.sessionIDs, sessionID)
	return f.waitError
}
func (f *fakeDesktop) RequestElectronValidationExit(_ context.Context, sessionID string) error {
	f.quitCalls++
	f.sessionIDs = append(f.sessionIDs, sessionID)
	return nil
}
func (f *fakeDesktop) RestartElectronValidation(_ context.Context, sessionID string) (*domainv1.AppTarget, error) {
	f.restartCalls++
	f.sessionIDs = append(f.sessionIDs, sessionID)
	return &domainv1.AppTarget{TargetId: "target-2", CdpEndpoint: "http://127.0.0.1:9223", RendererId: "renderer-2"}, nil
}
func (f *fakeDesktop) StopSession(string) error { return nil }

type fakeWorkflow struct{ passed bool }

func (f fakeWorkflow) Execute(context.Context, validationprovider.Request) validationprovider.Result {
	return validationprovider.Result{Passed: f.passed, ProviderRunID: "provider-1", Reason: "provider failed"}
}

type recordingWorkflow struct {
	request validationprovider.Request
}

func (f *recordingWorkflow) Execute(_ context.Context, request validationprovider.Request) validationprovider.Result {
	f.request = request
	return validationprovider.Result{Passed: true, ProviderRunID: "provider-1"}
}

type phasedWorkflow struct {
	requests []validationprovider.Request
	results  []validationprovider.Result
}

func (f *phasedWorkflow) Execute(_ context.Context, request validationprovider.Request) validationprovider.Result {
	f.requests = append(f.requests, request)
	index := len(f.requests) - 1
	if index < len(f.results) {
		return f.results[index]
	}
	return validationprovider.Result{Passed: true, ProviderRunID: "provider-default"}
}

type fakeCaptures struct{ capture *Capture }

func (f fakeCaptures) FindCapture(string, string) (*Capture, error) { return f.capture, nil }

func baseRequest() Request {
	return Request{RunID: "run-1", CellID: "cell-1", ScenarioName: "web-console", ArtifactPath: "/tmp/app", ArtifactDigest: "sha256:app", JourneyID: "journey-1", WorkflowPath: "/tmp/workflow.json", TargetID: "target-1", ProfileID: "normal", TargetAvailable: true}
}

func TestExecuteUnavailableOwner(t *testing.T) {
	result := Execute(context.Background(), Dependencies{Workflow: fakeWorkflow{passed: true}}, baseRequest())
	if result.Disposition != "unavailable" {
		t.Fatalf("disposition = %q", result.Disposition)
	}
}

func TestExecuteMissingWorkflowIsRefused(t *testing.T) {
	request := baseRequest()
	request.WorkflowPath = ""
	result := Execute(context.Background(), Dependencies{Desktop: &fakeDesktop{}, Workflow: fakeWorkflow{passed: true}}, request)
	if result.Disposition != "refused" || result.Reason != "provider journey has no normalized source path" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteRecordingWithoutPersistedCaptureFails(t *testing.T) {
	result := Execute(context.Background(), Dependencies{Desktop: &fakeDesktop{}, Workflow: fakeWorkflow{passed: true}, Captures: fakeCaptures{}}, baseRequest())
	if result.Disposition != "failed" || result.Reason != "desktop evidence recording was not persisted" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteProviderFailure(t *testing.T) {
	result := Execute(context.Background(), Dependencies{Desktop: &fakeDesktop{}, Workflow: fakeWorkflow{}, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, baseRequest())
	if result.Disposition != "failed" || result.Reason != "provider failed" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteProviderSuccess(t *testing.T) {
	desktop := &fakeDesktop{}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: fakeWorkflow{passed: true}, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, baseRequest())
	if result.Disposition != "pass" || desktop.waitCalls != 0 || desktop.restartCalls != 0 {
		t.Fatalf("unexpected legacy result: %+v desktop=%+v", result, desktop)
	}
}

func TestBundledDeploymentModeReachesDesktopAndProvider(t *testing.T) {
	desktop := &fakeDesktop{}
	workflow := &recordingWorkflow{}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, func() Request {
		request := baseRequest()
		request.DeploymentMode = "bundled"
		return request
	}())
	if result.Disposition != "pass" {
		t.Fatalf("result = %+v", result)
	}
	if !desktop.launchOptions.BundledRuntime || !workflow.request.BundledRuntime {
		t.Fatalf("bundled mode did not reach owners: launch=%v provider=%v", desktop.launchOptions.BundledRuntime, workflow.request.BundledRuntime)
	}
}

func TestOfflineProfileEnablesOwnerNetworkIsolationOnlyForOfflineCells(t *testing.T) {
	for _, profile := range []string{"offline", "normal"} {
		t.Run(profile, func(t *testing.T) {
			desktop := &fakeDesktop{}
			request := baseRequest()
			request.ProfileID = profile
			result := Execute(context.Background(), Dependencies{
				Desktop: desktop, Workflow: fakeWorkflow{passed: true},
				Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}},
			}, request)
			if result.Disposition != "pass" {
				t.Fatalf("result = %+v", result)
			}
			if got, want := desktop.launchOptions.OfflineNetwork, profile == "offline"; got != want {
				t.Fatalf("offline network isolation = %t, want %t", got, want)
			}
		})
	}
}

func TestUnknownDeploymentModeIsRefusedBeforeStartingDesktop(t *testing.T) {
	desktop := &fakeDesktop{}
	request := baseRequest()
	request.DeploymentMode = "mystery"
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: &recordingWorkflow{}}, request)
	if result.Disposition != "refused" || desktop.startRecording || desktop.launchOptions.BundledRuntime {
		t.Fatalf("unknown mode result=%+v desktop=%+v", result, desktop)
	}
}

func writeRestartPair(t *testing.T, continuation string) Request {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "device-studio")
	first := filepath.Join(scenario, "bas", "cases", "desktop")
	if err := os.MkdirAll(first, 0o700); err != nil {
		t.Fatal(err)
	}
	continuationPath := filepath.Join(first, "resume.json")
	if err := os.WriteFile(continuationPath, []byte(`{"metadata":{"name":"resume","labels":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(first, "start.json")
	body, _ := json.Marshal(map[string]any{"metadata": map[string]any{"labels": map[string]string{"restart_continuation": continuation}}})
	if err := os.WriteFile(firstPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	request := baseRequest()
	request.ScenarioRoot = root
	request.ScenarioName = "device-studio"
	request.WorkflowPath = firstPath
	return request
}

func TestExecuteRestartContinuationUsesSameSessionAndFreshTarget(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	desktop := &fakeDesktop{}
	workflow := &phasedWorkflow{results: []validationprovider.Result{{Passed: true, ProviderRunID: "provider-before"}, {Passed: true, ProviderRunID: "provider-after"}}}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, request)
	if result.Disposition != "pass" || len(workflow.requests) != 2 || desktop.waitCalls != 1 || desktop.restartCalls != 1 {
		t.Fatalf("result=%+v workflow requests=%d desktop=%+v", result, len(workflow.requests), desktop)
	}
	if workflow.requests[0].Target.GetTargetId() != "target-1" || workflow.requests[1].Target.GetTargetId() != "target-2" || workflow.requests[1].CellID != "cell-1:restart" || workflow.requests[1].ProfileID != request.ProfileID || workflow.requests[1].ContextID != "desktop-validation-cell-1" || workflow.requests[1].ArtifactDigest != request.ArtifactDigest || desktop.launchOptions.IsolationLeaseID != "run-1:cell-1" {
		t.Fatalf("phase bindings = %+v", workflow.requests)
	}
	if !workflow.requests[0].PrepareIsolationRestart || workflow.requests[0].ResumeIsolationRestart || workflow.requests[0].IsolationLeaseID != "run-1:cell-1" || workflow.requests[1].PrepareIsolationRestart || !workflow.requests[1].ResumeIsolationRestart || workflow.requests[1].IsolationLeaseID != "run-1:cell-1" || desktop.quitCalls != 1 {
		t.Fatalf("restart handoff did not prepare, owner-close, and resume the same lease: requests=%+v quit=%d", workflow.requests, desktop.quitCalls)
	}
	if len(desktop.sessionIDs) != 4 || desktop.sessionIDs[0] != desktop.sessionIDs[1] || desktop.sessionIDs[1] != desktop.sessionIDs[2] || desktop.sessionIDs[2] != desktop.sessionIDs[3] {
		t.Fatalf("owner operations used different sessions: %v", desktop.sessionIDs)
	}
}

func TestExecuteRestartContinuationResolvesSelectedRelativeWorkflowFromScenarioRoot(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	request.WorkflowPath = "bas/cases/desktop/start.json"
	desktop := &fakeDesktop{}
	workflow := &phasedWorkflow{results: []validationprovider.Result{{Passed: true}, {Passed: true}}}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, request)
	if result.Disposition != "pass" || len(workflow.requests) != 2 || desktop.quitCalls != 1 || desktop.waitCalls != 1 || desktop.restartCalls != 1 {
		t.Fatalf("relative selected workflow did not run its restart continuation: result=%+v requests=%d desktop=%+v", result, len(workflow.requests), desktop)
	}
	if workflow.requests[0].WorkflowPath != request.WorkflowPath || workflow.requests[1].WorkflowPath != filepath.Join(request.ScenarioRoot, request.ScenarioName, "bas/cases/desktop/resume.json") {
		t.Fatalf("workflow paths = %q then %q", workflow.requests[0].WorkflowPath, workflow.requests[1].WorkflowPath)
	}
}

func TestExecuteRestartContinuationDoesNotRestartAfterFirstPhaseFailure(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	desktop := &fakeDesktop{}
	workflow := &phasedWorkflow{results: []validationprovider.Result{{Passed: false, ProviderRunID: "provider-before", Reason: "phase one failed"}}}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, request)
	if result.Disposition != "failed" || len(workflow.requests) != 1 || desktop.waitCalls != 0 || desktop.restartCalls != 0 {
		t.Fatalf("result=%+v workflow requests=%d desktop=%+v", result, len(workflow.requests), desktop)
	}
}

func TestExecuteRestartContinuationFailsClosedWhenProcessDoesNotExit(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	desktop := &fakeDesktop{waitError: errors.New("deadline exceeded")}
	workflow := &phasedWorkflow{results: []validationprovider.Result{{Passed: true, ProviderRunID: "provider-before"}}}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, request)
	if result.Disposition != "failed" || desktop.waitCalls != 1 || desktop.restartCalls != 0 || len(workflow.requests) != 1 {
		t.Fatalf("result=%+v workflow requests=%d desktop=%+v", result, len(workflow.requests), desktop)
	}
}

func TestRestartContinuationRejectsEscapeAndSymlinkBeforeLaunch(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "parent traversal", path: "../outside.json"},
		{name: "absolute path", path: "/tmp/outside.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := writeRestartPair(t, tt.path)
			desktop := &fakeDesktop{}
			result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: &phasedWorkflow{}}, request)
			if result.Disposition != "refused" || desktop.launchCalls != 0 || desktop.waitCalls != 0 {
				t.Fatalf("result=%+v desktop=%+v", result, desktop)
			}
		})
	}
	t.Run("symlink escape", func(t *testing.T) {
		request := writeRestartPair(t, "bas/cases/desktop/resume.json")
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"metadata":{"labels":{}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		resume := filepath.Join(request.ScenarioRoot, request.ScenarioName, "bas", "cases", "desktop", "resume.json")
		if err := os.Remove(resume); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, resume); err != nil {
			t.Fatal(err)
		}
		desktop := &fakeDesktop{}
		result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: &phasedWorkflow{}}, request)
		if result.Disposition != "refused" || desktop.launchCalls != 0 {
			t.Fatalf("result=%+v desktop=%+v", result, desktop)
		}
	})
}

func TestRestartContinuationRejectsChainingBeforeLaunch(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	resume := filepath.Join(request.ScenarioRoot, request.ScenarioName, "bas", "cases", "desktop", "resume.json")
	if err := os.WriteFile(resume, []byte(`{"metadata":{"labels":{"restart_continuation":"bas/cases/desktop/again.json"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	desktop := &fakeDesktop{}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: &phasedWorkflow{}}, request)
	if result.Disposition != "refused" || !strings.Contains(result.Reason, "cannot chain") || desktop.launchCalls != 0 {
		t.Fatalf("result=%+v desktop=%+v", result, desktop)
	}
}

func TestExecuteRestartContinuationFailsAfterSecondPhaseAndRetainsEvidence(t *testing.T) {
	request := writeRestartPair(t, "bas/cases/desktop/resume.json")
	desktop := &fakeDesktop{}
	workflow := &phasedWorkflow{results: []validationprovider.Result{{Passed: true, ProviderRunID: "provider-before", Evidence: []*domainv1.LayeredEvidence{evidenceFromText(domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, "phase-one", "validation://phase-one", "before restart")}}, {Passed: false, ProviderRunID: "provider-after", Reason: "reopen failed", Evidence: []*domainv1.LayeredEvidence{evidenceFromText(domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, "phase-two", "validation://phase-two", "after restart")}}}}
	result := Execute(context.Background(), Dependencies{Desktop: desktop, Workflow: workflow, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, request)
	if result.Disposition != "failed" || len(workflow.requests) != 2 || !strings.Contains(result.ProviderRunID, "provider-before") || !strings.Contains(result.ProviderRunID, "provider-after") || len(result.Evidence) < 5 {
		t.Fatalf("result=%+v requests=%d", result, len(workflow.requests))
	}
}
