package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationmatrix"
	"scenario-to-desktop-api/validationprovider"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type validationArtifactFinder interface {
	FindArtifact(string) (string, error)
}

type validationDesktopOwner interface {
	StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error)
	ExecuteAction(context.Context, string, string, json.RawMessage) (*livedesktop.ActionResult, error)
	LaunchElectronValidation(context.Context, string, string, target.ElectronLaunchOptions, target.RendererExpectation) (*domainv1.ElectronTarget, error)
	StopSession(string) error
}

type validationWorkflowExecutor interface {
	Execute(context.Context, validationprovider.Request) validationprovider.Result
}

// validationMatrixLocalExecutor adapts the existing desktop smoke-test owner
// to the provider-neutral matrix contract. It does not discover or execute
// semantic workflows; those remain owned by their validation provider and are
// represented only through the cell's linked evidence.
type validationMatrixLocalExecutor struct {
	smokeService smoketest.Service
	smokeStore   smoketest.Store
	findArtifact validationArtifactFinder
	captures     *captures.Service
	desktop      validationDesktopOwner
	workflow     validationWorkflowExecutor
}

func (e validationMatrixLocalExecutor) ExecuteLocal(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if e.smokeService == nil || e.smokeStore == nil || e.findArtifact == nil || request.Cell == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "local desktop validation adapter is unavailable"}
	}
	artifactPath, err := e.findArtifact.FindArtifact(request.Cell.GetScenarioName())
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("resolve desktop artifact: %v", err)}
	}
	if digest := strings.TrimSpace(request.ArtifactDigest); strings.HasPrefix(digest, "sha256:") {
		if actual, hashErr := fileDigest(artifactPath); hashErr != nil {
			return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("hash desktop artifact: %v", hashErr)}
		} else if actual != digest {
			return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("desktop artifact digest mismatch: want %s got %s", digest, actual)}
		}
	}
	if !strings.EqualFold(strings.TrimSpace(request.Journey.ExecutionMode), "platform") {
		return e.executeProviderJourney(ctx, request, artifactPath)
	}

	smokeID := "validation-" + request.Cell.GetCellId()
	e.smokeStore.Save(&smoketest.Status{
		SmokeTestID:  smokeID,
		ScenarioName: request.Cell.GetScenarioName(),
		Platform:     e.smokeService.CurrentPlatform(),
		Status:       "running",
		ArtifactPath: artifactPath,
		StartedAt:    time.Now().UTC(),
		Logs:         []string{"validation matrix cell queued"},
		CurrentState: smoketest.StateInitializing,
		RecordingConfig: &smoketest.ScreenRecordingConfig{
			Enabled: true, DisplayWidth: 1920, DisplayHeight: 1080, FPS: 15,
		},
	})
	e.smokeService.PerformSmokeTest(ctx, smokeID, request.Cell.GetScenarioName(), artifactPath, e.smokeService.CurrentPlatform())
	status, ok := e.smokeStore.Get(smokeID)
	if !ok {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: "desktop smoke-test status disappeared"}
	}
	if ctx.Err() != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "desktop cell cancelled"}
	}
	if status.Status != "passed" {
		reason := strings.TrimSpace(status.Error)
		if reason == "" {
			reason = "desktop smoke test failed"
		}
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: reason}
	}
	if strings.EqualFold(status.JourneyDisposition, "degraded") || status.PerformanceStatus == "degraded" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: firstNonEmpty(status.JourneyDegradedReason, status.PerformanceReason, "desktop evidence degraded"), Evidence: smokeEvidence(status, e.captures, request)}
	}
	return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "desktop smoke and evidence completed", Evidence: smokeEvidence(status, e.captures, request)}
}

func (e validationMatrixLocalExecutor) executeProviderJourney(ctx context.Context, request validationmatrix.CellRequest, artifactPath string) validationmatrix.CellResult {
	if e.desktop == nil || e.workflow == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "provider-owned Electron target adapter is unavailable"}
	}
	if request.Target == nil || !request.Target.GetAvailable() {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "selected Electron target is unavailable"}
	}
	if strings.TrimSpace(request.Journey.SourcePath) == "" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "provider journey has no normalized source path"}
	}
	session, err := e.desktop.StartSession(ctx, livedesktop.SessionConfig{Width: 1920, Height: 1080, ScenarioName: request.Cell.GetScenarioName(), Platform: "linux"})
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("start Electron validation desktop: %v", err)}
	}
	defer func() { _ = e.desktop.StopSession(session.ID) }()

	recording, err := e.desktop.ExecuteAction(ctx, session.ID, "start_recording", json.RawMessage("{}"))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("start desktop evidence recording: %v", err)}
	}
	recordingID := actionString(recording, "capture_id")
	if recordingID == "" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: "desktop evidence recording returned no capture identity"}
	}
	recordingStopped := false
	defer func() {
		if !recordingStopped {
			_, _ = e.desktop.ExecuteAction(context.Background(), session.ID, "stop_recording", json.RawMessage("{}"))
		}
	}()

	contextID := "desktop-validation-" + request.Cell.GetCellId()
	targetInfo, err := e.desktop.LaunchElectronValidation(ctx, session.ID, artifactPath, target.ElectronLaunchOptions{
		ContextID:        contextID,
		ScenarioName:     request.Cell.GetScenarioName(),
		ArtifactDigest:   request.ArtifactDigest,
		TargetID:         request.Cell.GetTargetId(),
		JourneyID:        request.Cell.GetJourneyId(),
		ProfileID:        profileID(request.Cell.GetEnvironmentProfile()),
		IsolationLeaseID: request.RunID + ":" + request.Cell.GetCellId(),
	}, target.RendererExpectation{
		// Bundled Electron apps may expose a file:// splash page before their
		// runtime-owned UI is ready. Bind to the loopback UI origin so the
		// provider never receives the bootstrap renderer.
		URLPrefix: "http://127.0.0.1:",
	})
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("launch and attach Electron target: %v", err)}
	}

	providerResult := e.workflow.Execute(ctx, validationprovider.Request{
		MatrixRunID:    request.RunID,
		CellID:         request.Cell.GetCellId(),
		ScenarioName:   request.Cell.GetScenarioName(),
		ScenarioPath:   scenarioPathFromArtifact(artifactPath),
		WorkflowPath:   request.Journey.SourcePath,
		WorkflowID:     request.Journey.JourneyID,
		Target:         targetInfo,
		ProfileID:      profileID(request.Cell.GetEnvironmentProfile()),
		ContextID:      contextID,
		ArtifactDigest: request.ArtifactDigest,
	})

	stopped, stopErr := e.desktop.ExecuteAction(context.Background(), session.ID, "stop_recording", json.RawMessage("{}"))
	recordingStopped = true
	if stopErr != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("stop desktop evidence recording: %v", stopErr), Evidence: providerResult.Evidence}
	}
	if stoppedID := actionString(stopped, "capture_id"); stoppedID != "" {
		recordingID = stoppedID
	}
	evidence := append([]*domainv1.LayeredEvidence(nil), providerResult.Evidence...)
	if capture := findCapture(e.captures, request.Cell.GetScenarioName(), recordingID); capture != nil {
		mediaType := "video/mp4"
		evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: "/api/v1/captures/" + request.Cell.GetScenarioName() + "/" + capture.ID + "/file", Sha256: capture.Checksum, MediaType: &mediaType, Redacted: true})
	} else {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: "desktop evidence recording was not persisted", Evidence: evidence}
	}
	evidence = append(evidence, targetEvidence(targetInfo, request), machineEvidence(request, providerResult.ProviderRunID))
	if !providerResult.Passed {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: firstNonEmpty(providerResult.Reason, "provider-owned workflow failed"), Evidence: evidence}
	}
	return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "provider workflow and Electron desktop evidence completed", Evidence: evidence}
}

func actionString(action *livedesktop.ActionResult, key string) string {
	if action == nil || action.Data == nil {
		return ""
	}
	value, _ := action.Data[key].(string)
	return strings.TrimSpace(value)
}

func findCapture(service *captures.Service, scenario, id string) *captures.Capture {
	if service == nil || id == "" {
		return nil
	}
	items, err := service.Store().List(scenario)
	if err != nil {
		return nil
	}
	for _, item := range items {
		if item.ID == id {
			copy := item
			return &copy
		}
	}
	return nil
}

func scenarioPathFromArtifact(artifactPath string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(artifactPath), "..", "..", ".."))
}

func profileID(profile domainv1.ValidationEnvironmentProfile) string {
	name := strings.TrimPrefix(profile.String(), "VALIDATION_ENVIRONMENT_PROFILE_")
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

func targetEvidence(info *domainv1.ElectronTarget, request validationmatrix.CellRequest) *domainv1.LayeredEvidence {
	value := fmt.Sprintf("target=%s endpoint=%s renderer=%s digest=%s", info.GetTargetId(), info.GetCdpEndpoint(), info.GetRendererId(), request.ArtifactDigest)
	return evidenceFromText(domainv1.LayeredEvidence_KIND_TARGET, "target-"+request.Cell.GetTargetId(), "validation://target/"+request.Cell.GetTargetId(), value)
}

func machineEvidence(request validationmatrix.CellRequest, providerRunID string) *domainv1.LayeredEvidence {
	value := fmt.Sprintf("run=%s cell=%s provider_run=%s profile=%s", request.RunID, request.Cell.GetCellId(), providerRunID, profileID(request.Cell.GetEnvironmentProfile()))
	return evidenceFromText(domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, "assertion-"+request.Cell.GetCellId(), "validation://assertion/"+request.Cell.GetCellId(), value)
}

func evidenceFromText(kind domainv1.LayeredEvidence_Kind, id, uri, value string) *domainv1.LayeredEvidence {
	digest := sha256.Sum256([]byte(value))
	return &domainv1.LayeredEvidence{Kind: kind, EvidenceId: id, Uri: uri, Sha256: "sha256:" + hex.EncodeToString(digest[:]), Redacted: true}
}

func smokeEvidence(status *smoketest.Status, captureService *captures.Service, request validationmatrix.CellRequest) []*domainv1.LayeredEvidence {
	var evidence []*domainv1.LayeredEvidence
	if captureService != nil {
		if list, err := captureService.Store().List(status.ScenarioName); err == nil {
			wanted := map[string]struct{}{status.JourneyCaptureID: {}, "": {}}
			if status.ScreenRecording != nil {
				wanted[status.ScreenRecording.CaptureID] = struct{}{}
			}
			for _, capture := range list {
				if _, ok := wanted[capture.ID]; !ok || capture.ID == "" {
					continue
				}
				mediaType := "application/octet-stream"
				switch capture.Type {
				case captures.CaptureRecording:
					mediaType = "video/webm"
				case captures.CaptureScreenshot:
					mediaType = "image/png"
				case captures.CaptureJourney:
					mediaType = "application/json"
				}
				evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: "/api/v1/captures/" + capture.ID, Sha256: capture.Checksum, MediaType: &mediaType, Redacted: true})
			}
		}
	}
	metadata := fmt.Sprintf("scenario=%s artifact=%s target=%s cell=%s", request.Cell.GetScenarioName(), request.ArtifactDigest, request.Cell.GetTargetId(), request.Cell.GetCellId())
	digest := sha256.Sum256([]byte(metadata))
	checksum := "sha256:" + hex.EncodeToString(digest[:])
	evidence = append(evidence,
		&domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "target-" + request.Cell.GetTargetId(), Uri: "validation://target/" + request.Cell.GetTargetId(), Sha256: checksum, Redacted: true},
		&domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "assertion-" + request.Cell.GetCellId(), Uri: "validation://assertion/" + request.Cell.GetCellId(), Sha256: checksum, Redacted: true},
	)
	if status.EvidenceReview != nil && status.EvidenceReview.WorkflowReference != nil {
		for _, artifact := range status.EvidenceReview.WorkflowReference.Artifacts {
			evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: artifact.ID, Uri: artifact.URI, Sha256: artifact.Checksum, Redacted: artifact.Redacted})
		}
	}
	return evidence
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
