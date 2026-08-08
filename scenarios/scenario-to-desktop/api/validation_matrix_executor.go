package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/validationmatrix"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type validationArtifactFinder interface {
	FindArtifact(string) (string, error)
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
				evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: "/api/v1/captures/" + capture.ID, Sha256: capture.Checksum, Redacted: true})
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
