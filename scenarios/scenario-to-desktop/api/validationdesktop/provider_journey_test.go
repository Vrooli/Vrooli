package validationdesktop

import (
	"context"
	"encoding/json"
	"testing"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationprovider"
)

type fakeDesktop struct {
	startRecording bool
	stopRecording  bool
}

func (f *fakeDesktop) StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error) {
	return &livedesktop.Session{ID: "session-1"}, nil
}
func (f *fakeDesktop) ExecuteAction(_ context.Context, _ string, action string, _ json.RawMessage) (*livedesktop.ActionResult, error) {
	if action == "start_recording" {
		f.startRecording = true
		return &livedesktop.ActionResult{Data: map[string]any{"capture_id": "recording-1"}}, nil
	}
	f.stopRecording = true
	return &livedesktop.ActionResult{Data: map[string]any{"capture_id": "recording-1"}}, nil
}
func (f *fakeDesktop) LaunchElectronValidation(context.Context, string, string, target.ElectronLaunchOptions, target.RendererExpectation) (*domainv1.AppTarget, error) {
	return &domainv1.AppTarget{TargetId: "target-1", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer-1"}, nil
}
func (f *fakeDesktop) StopSession(string) error { return nil }

type fakeWorkflow struct{ passed bool }

func (f fakeWorkflow) Execute(context.Context, validationprovider.Request) validationprovider.Result {
	return validationprovider.Result{Passed: f.passed, ProviderRunID: "provider-1", Reason: "provider failed"}
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
	result := Execute(context.Background(), Dependencies{Desktop: &fakeDesktop{}, Workflow: fakeWorkflow{passed: true}, Captures: fakeCaptures{capture: &Capture{ID: "recording-1", Checksum: "sha256:recording"}}}, baseRequest())
	if result.Disposition != "pass" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
