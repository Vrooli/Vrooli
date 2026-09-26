package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationdesktop"
	"scenario-to-desktop-api/validationprovider"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type validationArtifactFinder interface {
	FindArtifact(string) (string, error)
}

type validationArtifactByDigestFinder interface {
	FindArtifactByDigest(string, string) (string, error)
}

type validationDesktopOwner interface {
	StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error)
	ExecuteAction(context.Context, string, string, json.RawMessage) (*livedesktop.ActionResult, error)
	LaunchElectronValidation(context.Context, string, string, target.ElectronLaunchOptions, target.RendererExpectation) (*domainv1.AppTarget, error)
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
	builder      deliveryramp.Builder
	scenarioRoot string
}

func (e validationMatrixLocalExecutor) Execute(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if e.smokeService == nil || e.smokeStore == nil || e.findArtifact == nil || request.Cell == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "local desktop validation adapter is unavailable"}
	}
	artifactPath, err := e.findValidationArtifact(request)
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
	smoketest.RunSmokeTestRequest(ctx, e.smokeService, smoketest.SmokeTestRequest{
		SmokeTestID: smokeID, ScenarioName: request.Cell.GetScenarioName(), ArtifactPath: artifactPath,
		Platform: e.smokeService.CurrentPlatform(), DeploymentMode: "unknown",
	})
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

func (e validationMatrixLocalExecutor) findValidationArtifact(request validationmatrix.CellRequest) (string, error) {
	scenarioName := request.Cell.GetScenarioName()
	if e.builder != nil {
		artifact, err := e.builder.Build(context.Background(), deliveryramp.BuildRequest{SourceRef: scenarioName})
		if err == nil && strings.TrimSpace(artifact.LocalPath) != "" {
			return artifact.LocalPath, nil
		}
		if err != nil {
			return "", err
		}
	}
	if finder, ok := e.findArtifact.(validationArtifactByDigestFinder); ok && strings.TrimSpace(request.ArtifactDigest) != "" {
		return finder.FindArtifactByDigest(scenarioName, request.ArtifactDigest)
	}
	return e.findArtifact.FindArtifact(scenarioName)
}

func (e validationMatrixLocalExecutor) executeProviderJourney(ctx context.Context, request validationmatrix.CellRequest, artifactPath string) validationmatrix.CellResult {
	targetAvailable := request.Target != nil && request.Target.GetAvailable()
	result := validationdesktop.Execute(ctx, validationdesktop.Dependencies{
		Desktop:  e.desktop,
		Workflow: e.workflow,
		Captures: captureReader{service: e.captures},
	}, validationdesktop.Request{
		RunID: request.RunID, CellID: request.Cell.GetCellId(), ScenarioName: request.Cell.GetScenarioName(),
		ScenarioRoot: e.scenarioRoot, ArtifactPath: artifactPath, ArtifactDigest: request.ArtifactDigest,
		JourneyID: request.Cell.GetJourneyId(), WorkflowPath: request.Journey.SourcePath,
		TargetID: request.Cell.GetTargetId(), ProfileID: profileID(request.Cell.GetEnvironmentProfile()),
		TargetAvailable: targetAvailable,
	})
	return validationmatrix.CellResult{Disposition: mapValidationDisposition(result.Disposition), Reason: result.Reason, Evidence: result.Evidence}
}

func mapValidationDisposition(disposition string) domainv1.ValidationDisposition {
	switch disposition {
	case "pass":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS
	case "failed":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED
	case "degraded":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED
	case "unsupported":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED
	case "refused":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED
	case "not_run":
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
	default:
		return domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE
	}
}

type captureReader struct{ service *captures.Service }

func (r captureReader) FindCapture(scenario, id string) (*validationdesktop.Capture, error) {
	if r.service == nil || id == "" {
		return nil, fmt.Errorf("capture service is unavailable")
	}
	items, err := r.service.Store().List(scenario)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == id {
			return &validationdesktop.Capture{ID: item.ID, Checksum: item.Checksum}, nil
		}
	}
	return nil, fmt.Errorf("capture %q not found", id)
}

func profileID(profile domainv1.ValidationEnvironmentProfile) string {
	name := strings.TrimPrefix(profile.String(), "VALIDATION_ENVIRONMENT_PROFILE_")
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
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
					mediaType = "video/mp4"
				case captures.CaptureScreenshot:
					mediaType = "image/png"
				case captures.CaptureJourney:
					mediaType = "application/json"
				}
				uri := "/api/v1/captures/" + status.ScenarioName + "/" + capture.ID + "/file"
				evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: uri, Sha256: capture.Checksum, MediaType: &mediaType, Redacted: true})
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
