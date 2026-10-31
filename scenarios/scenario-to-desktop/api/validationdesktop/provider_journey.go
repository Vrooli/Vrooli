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
	"os"
	"path/filepath"
	"strings"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationprovider"
)

type DesktopOwner interface {
	StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error)
	ExecuteAction(context.Context, string, string, json.RawMessage) (*livedesktop.ActionResult, error)
	LaunchElectronValidation(context.Context, string, string, target.ElectronLaunchOptions, target.RendererExpectation) (*domainv1.AppTarget, error)
	RequestElectronValidationExit(context.Context, string) error
	WaitElectronValidationExit(context.Context, string) error
	RestartElectronValidation(context.Context, string) (*domainv1.AppTarget, error)
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
	RunID                          string
	CellID                         string
	ScenarioName                   string
	ScenarioRoot                   string
	ArtifactPath                   string
	ArtifactDigest                 string
	JourneyID                      string
	WorkflowPath                   string
	TargetID                       string
	ProfileID                      string
	TargetAvailable                bool
	DeploymentMode                 string
	BrowserAutomationStudioVariant string
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

func bundledRuntimeForMode(mode string) (bool, error) {
	switch mode {
	case "", "external-server", "cloud-api", "proxy":
		return false, nil
	case "bundled":
		return true, nil
	default:
		return false, fmt.Errorf("unsupported validation deployment mode %q", mode)
	}
}

func Execute(ctx context.Context, deps Dependencies, request Request) Result {
	if deps.Desktop == nil || deps.Workflow == nil {
		return Result{Disposition: "unavailable", Reason: "provider-owned Electron target adapter is unavailable"}
	}
	if !request.TargetAvailable {
		return Result{Disposition: "unavailable", Reason: "selected Electron target is unavailable"}
	}
	bundledRuntime, err := bundledRuntimeForMode(request.DeploymentMode)
	if err != nil {
		return Result{Disposition: "refused", Reason: err.Error()}
	}
	if strings.TrimSpace(request.WorkflowPath) == "" {
		return Result{Disposition: "refused", Reason: "provider journey has no normalized source path"}
	}
	continuationPath, err := restartContinuationPath(request)
	if err != nil {
		return Result{Disposition: "refused", Reason: err.Error()}
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
		BundledRuntime:   bundledRuntime,
		OfflineNetwork:   request.ProfileID == "offline",
	}, target.RendererExpectation{URLPrefix: "http://127.0.0.1:"})
	if err != nil {
		return Result{Disposition: "failed", Reason: fmt.Sprintf("launch and attach Electron target: %v", err)}
	}

	providerResults := []validationprovider.Result{executeWorkflow(ctx, deps.Workflow, request, targetInfo, request.WorkflowPath, request.CellID, contextID, bundledRuntime, continuationPath != "", false)}
	targets := []*domainv1.AppTarget{targetInfo}
	restartErr := ""
	if continuationPath != "" && providerResults[0].Passed {
		// The provider has prepared the paired database/file checkpoint while
		// the target API is still alive. Close through the desktop owner only
		// after that durable handoff is complete.
		if err := deps.Desktop.RequestElectronValidationExit(ctx, session.ID); err != nil {
			restartErr = fmt.Sprintf("close Electron application through its owner: %v", err)
		}
		waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		var waitErr error
		if restartErr == "" {
			waitErr = deps.Desktop.WaitElectronValidationExit(waitCtx, session.ID)
		}
		cancel()
		if restartErr != "" {
			// Keep the owner-action error as the primary restart failure.
		} else if waitErr != nil {
			restartErr = fmt.Sprintf("wait for Electron application to exit before restart: %v", waitErr)
		} else if nextTarget, restartErrValue := deps.Desktop.RestartElectronValidation(ctx, session.ID); restartErrValue != nil {
			restartErr = fmt.Sprintf("restart Electron validation target: %v", restartErrValue)
		} else {
			targets = append(targets, nextTarget)
			providerResults = append(providerResults, executeWorkflow(ctx, deps.Workflow, request, nextTarget, continuationPath, request.CellID+":restart", contextID, bundledRuntime, false, true))
		}
	}
	partialEvidence := providerEvidence(providerResults)

	stopped, stopErr := deps.Desktop.ExecuteAction(context.Background(), session.ID, "stop_recording", json.RawMessage("{}"))
	recordingStopped = true
	if stopErr != nil {
		return Result{Disposition: "failed", Reason: fmt.Sprintf("stop desktop evidence recording: %v", stopErr), Evidence: partialEvidence}
	}
	if stoppedID := actionString(stopped, "capture_id"); stoppedID != "" {
		recordingID = stoppedID
	}
	evidence := partialEvidence
	providerRunIDs := make([]string, 0, len(providerResults))
	for _, providerResult := range providerResults {
		if strings.TrimSpace(providerResult.ProviderRunID) != "" {
			providerRunIDs = append(providerRunIDs, providerResult.ProviderRunID)
		}
	}
	for _, info := range targets {
		evidence = append(evidence, targetEvidence(info, request))
	}
	providerRunID := strings.Join(providerRunIDs, ",")
	evidence = append(evidence, machineEvidence(request, providerRunID))
	if deps.Captures != nil {
		capture, captureErr := deps.Captures.FindCapture(request.ScenarioName, recordingID)
		if captureErr == nil && capture != nil {
			mediaType := "video/mp4"
			evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: capture.ID, Uri: "/api/v1/captures/" + request.ScenarioName + "/" + capture.ID + "/file", Sha256: capture.Checksum, MediaType: &mediaType, Redacted: true})
		} else {
			return Result{Disposition: "failed", Reason: "desktop evidence recording was not persisted", ProviderRunID: providerRunID, Evidence: evidence}
		}
	}
	if restartErr != "" {
		return Result{Disposition: "failed", Reason: restartErr, ProviderRunID: providerRunID, Evidence: evidence}
	}
	for _, providerResult := range providerResults {
		if !providerResult.Passed {
			return Result{Disposition: "failed", Reason: firstNonEmpty(providerResult.Reason, "provider-owned workflow failed"), ProviderRunID: providerRunID, Evidence: evidence}
		}
	}
	return Result{Disposition: "pass", Reason: "provider workflow and Electron desktop evidence completed", ProviderRunID: providerRunID, Evidence: evidence}
}

func providerEvidence(results []validationprovider.Result) []*domainv1.LayeredEvidence {
	evidence := make([]*domainv1.LayeredEvidence, 0)
	for _, result := range results {
		evidence = append(evidence, result.Evidence...)
	}
	return evidence
}

func executeWorkflow(ctx context.Context, workflow WorkflowExecutor, request Request, targetInfo *domainv1.AppTarget, workflowPath, cellID, contextID string, bundledRuntime, prepareRestart, resumeRestart bool) validationprovider.Result {
	return workflow.Execute(ctx, validationprovider.Request{
		MatrixRunID: request.RunID, CellID: cellID, ScenarioName: request.ScenarioName,
		ScenarioPath: scenarioPathForRequest(request.ScenarioRoot, request.ScenarioName, request.ArtifactPath),
		WorkflowPath: workflowPath, Target: targetInfo,
		BrowserAutomationStudioVariant: request.BrowserAutomationStudioVariant,
		ProfileID:                      request.ProfileID, ContextID: contextID, ArtifactDigest: request.ArtifactDigest,
		BundledRuntime:          bundledRuntime,
		IsolationLeaseID:        request.RunID + ":" + request.CellID,
		PrepareIsolationRestart: prepareRestart,
		ResumeIsolationRestart:  resumeRestart,
	})
}

type workflowMetadata struct {
	Metadata struct {
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
}

func restartContinuationPath(request Request) (string, error) {
	workflowPath, err := resolveSelectedWorkflowPath(request)
	if err != nil {
		return "", err
	}
	if workflowPath == "" {
		return "", nil
	}
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		// Preserve existing provider-owned validation for workflows that are not
		// available to this local adapter (for example, remote catalog entries).
		return "", nil
	}
	var first workflowMetadata
	if err := json.Unmarshal(data, &first); err != nil {
		return "", nil
	}
	relative := strings.TrimSpace(first.Metadata.Labels["restart_continuation"])
	if relative == "" {
		return "", nil
	}
	relativePath := filepath.FromSlash(relative)
	if filepath.IsAbs(relativePath) || filepath.Clean(relativePath) != relativePath || !filepath.IsLocal(relativePath) || filepath.Ext(relativePath) != ".json" {
		return "", fmt.Errorf("restart continuation must be a clean relative JSON path")
	}
	if strings.TrimSpace(request.ScenarioRoot) == "" || strings.TrimSpace(request.ScenarioName) == "" {
		return "", fmt.Errorf("restart continuation requires a selected scenario root")
	}
	root, err := filepath.EvalSymlinks(filepath.Join(request.ScenarioRoot, request.ScenarioName))
	if err != nil {
		return "", fmt.Errorf("resolve selected scenario root for restart continuation: %w", err)
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, relativePath))
	if err != nil {
		return "", fmt.Errorf("resolve restart continuation within selected scenario: %w", err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("restart continuation resolves outside selected scenario")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("restart continuation must resolve to a regular file")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read restart continuation: %w", err)
	}
	var continuation workflowMetadata
	if err := json.Unmarshal(data, &continuation); err != nil {
		return "", fmt.Errorf("decode restart continuation metadata: %w", err)
	}
	if strings.TrimSpace(continuation.Metadata.Labels["restart_continuation"]) != "" {
		return "", fmt.Errorf("restart continuations cannot chain")
	}
	return path, nil
}

// resolveSelectedWorkflowPath keeps absolute paths used by existing callers,
// and resolves selected relative catalog paths under the scenario root. A
// missing local file remains eligible for remote provider-owned workflows.
func resolveSelectedWorkflowPath(request Request) (string, error) {
	selected := strings.TrimSpace(request.WorkflowPath)
	if selected == "" || filepath.IsAbs(selected) {
		return selected, nil
	}
	if filepath.Clean(selected) != selected || !filepath.IsLocal(selected) {
		return "", fmt.Errorf("selected workflow path must be a clean relative path")
	}
	if strings.TrimSpace(request.ScenarioRoot) == "" || strings.TrimSpace(request.ScenarioName) == "" {
		return "", nil
	}
	root, err := filepath.EvalSymlinks(filepath.Join(request.ScenarioRoot, request.ScenarioName))
	if err != nil {
		return "", nil
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, selected))
	if err != nil {
		return "", nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("selected workflow resolves outside selected scenario")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", nil
	}
	return path, nil
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
