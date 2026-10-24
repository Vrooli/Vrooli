// Package validationdesktop owns execution of provider semantic journeys
// against an isolated Electron target. It deliberately does not depend on the
// validation-matrix package or package main so pipeline and smoketest callers
// can share the same desktop adapter.
package validationdesktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationprovider"
)

type DesktopOwner interface {
	StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error)
	ExecuteAction(context.Context, string, string, json.RawMessage) (*livedesktop.ActionResult, error)
	LaunchElectronValidation(context.Context, string, string, target.ElectronLaunchOptions, target.RendererExpectation) (*domainv1.AppTarget, error)
	StopSession(string) error
}

type WorkflowExecutor interface {
	Execute(context.Context, validationprovider.Request) validationprovider.Result
}

type Capture struct {
	ID       string
	Checksum string
}

type CaptureReader interface {
	FindCapture(scenario, id string) (*Capture, error)
}

type Request struct {
	RunID           string
	CellID          string
	ScenarioName    string
	ScenarioRoot    string
	ArtifactPath    string
	ArtifactDigest  string
	JourneyID       string
	WorkflowPath    string
	TargetID        string
	ProfileID       string
	TargetAvailable bool
}

type Result struct {
	Disposition   string
	Reason        string
	ProviderRunID string
	Evidence      []*domainv1.LayeredEvidence
}

type Dependencies struct {
	Desktop  DesktopOwner
	Workflow WorkflowExecutor
	Captures CaptureReader
}

func Execute(ctx context.Context, deps Dependencies, request Request) Result {
	if deps.Desktop == nil || deps.Workflow == nil {
		return Result{Disposition: "unavailable", Reason: "provider-owned Electron target adapter is unavailable"}
	}
	if !request.TargetAvailable {
		return Result{Disposition: "unavailable", Reason: "selected Electron target is unavailable"}
	}
	if strings.TrimSpace(request.WorkflowPath) == "" {
		return Result{Disposition: "refused", Reason: "provider journey has no normalized source path"}
	}
	session, err := deps.Desktop.StartSession(ctx, livedesktop.SessionConfig{Width: 1920, Height: 1080, ScenarioName: request.ScenarioName, Platform: "linux"})
	if err != nil {
		return Result{Disposition: "unavailable", Reason: fmt.Sprintf("start Electron validation desktop: %v", err)}
	}
	defer func() { _ = deps.Desktop.StopSession(session.ID) }()

	recording, err := deps.Desktop.ExecuteAction(ctx, session.ID, "start_recording", json.RawMessage("{}"))
	if err != nil {
		return Result{Disposition: "unavailable", Reason: fmt.Sprintf("start desktop evidence recording: %v", err)}
	}
	recordingID := actionString(recording, "capture_id")
	if recordingID == "" {
		return Result{Disposition: "failed", Reason: "desktop evidence recording returned no capture identity"}
	}
	recordingStopped := false
	defer func() {
		if !recordingStopped {
			_, _ = deps.Desktop.ExecuteAction(context.Background(), session.ID, "stop_recording", json.RawMessage("{}"))
		}
	}()

	contextID := "desktop-validation-" + request.CellID
	targetInfo, err := deps.Desktop.LaunchElectronValidation(ctx, session.ID, request.ArtifactPath, target.ElectronLaunchOptions{
		ContextID: contextID, ScenarioName: request.ScenarioName, ArtifactDigest: request.ArtifactDigest,
		TargetID: request.TargetID, JourneyID: request.JourneyID, ProfileID: request.ProfileID,
		IsolationLeaseID: request.RunID + ":" + request.CellID,
	}, target.RendererExpectation{URLPrefix: "http://127.0.0.1:"})
	if err != nil {
		return Result{Disposition: "failed", Reason: fmt.Sprintf("launch and attach Electron target: %v", err)}
	}

	providerResult := deps.Workflow.Execute(ctx, validationprovider.Request{
		MatrixRunID: request.RunID, CellID: request.CellID, ScenarioName: request.ScenarioName,
		ScenarioPath: scenarioPathForRequest(request.ScenarioRoot, request.ScenarioName, request.ArtifactPath),
		WorkflowPath: request.WorkflowPath, WorkflowID: request.JourneyID, Target: targetInfo,
		ProfileID: request.ProfileID, ContextID: contextID, ArtifactDigest: request.ArtifactDigest,
	})

	stopped, stopErr := deps.Desktop.ExecuteAction(context.Background(), session.ID, "stop_recording", json.RawMessage("{}"))
	recordingStopped = true
	if stopErr != nil {
		return Result{Disposition: "failed", Reason: fmt.Sprintf("stop desktop evidence recording: %v", stopErr), Evidence: providerResult.Evidence}
	}
	if stoppedID := actionString(stopped, "capture_id"); stoppedID != "" {
		recordingID = stoppedID
	}
	evidence := append([]*domainv1.LayeredEvidence(nil), providerResult.Evidence...)
	if deps.Captures != nil {
		capture, captureErr := deps.Captures.FindCapture(request.ScenarioName, recordingID)
		if captureErr == nil && capture != nil {
			mediaType := "video/mp4"
			evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: "/api/v1/captures/" + request.ScenarioName + "/" + capture.ID + "/file", Sha256: capture.Checksum, MediaType: &mediaType, Redacted: true})
		} else {
			return Result{Disposition: "failed", Reason: "desktop evidence recording was not persisted", Evidence: evidence}
		}
	}
	evidence = append(evidence, targetEvidence(targetInfo, request), machineEvidence(request, providerResult.ProviderRunID))
	if !providerResult.Passed {
		return Result{Disposition: "failed", Reason: firstNonEmpty(providerResult.Reason, "provider-owned workflow failed"), ProviderRunID: providerResult.ProviderRunID, Evidence: evidence}
	}
	return Result{Disposition: "pass", Reason: "provider workflow and Electron desktop evidence completed", ProviderRunID: providerResult.ProviderRunID, Evidence: evidence}
}

func actionString(action *livedesktop.ActionResult, key string) string {
	if action == nil || action.Data == nil {
		return ""
	}
	value, _ := action.Data[key].(string)
	return strings.TrimSpace(value)
}

func scenarioPathForRequest(scenarioRoot, scenarioName, artifactPath string) string {
	if strings.TrimSpace(scenarioRoot) != "" && strings.TrimSpace(scenarioName) != "" {
		return filepath.Join(scenarioRoot, scenarioName)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(artifactPath), "..", "..", ".."))
}

func targetEvidence(info *domainv1.AppTarget, request Request) *domainv1.LayeredEvidence {
	value := fmt.Sprintf("target=%s endpoint=%s renderer=%s digest=%s", info.GetTargetId(), info.GetCdpEndpoint(), info.GetRendererId(), request.ArtifactDigest)
	return evidenceFromText(domainv1.LayeredEvidence_KIND_TARGET, "target-"+request.TargetID, "validation://target/"+request.TargetID, value)
}

func machineEvidence(request Request, providerRunID string) *domainv1.LayeredEvidence {
	value := fmt.Sprintf("run=%s cell=%s provider_run=%s profile=%s", request.RunID, request.CellID, providerRunID, request.ProfileID)
	return evidenceFromText(domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, "assertion-"+request.CellID, "validation://assertion/"+request.CellID, value)
}

func evidenceFromText(kind domainv1.LayeredEvidence_Kind, id, uri, value string) *domainv1.LayeredEvidence {
	digest := sha256.Sum256([]byte(value))
	return &domainv1.LayeredEvidence{Kind: kind, EvidenceId: id, Uri: uri, Sha256: "sha256:" + hex.EncodeToString(digest[:]), Redacted: true}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
