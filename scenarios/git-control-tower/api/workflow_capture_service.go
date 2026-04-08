package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkflowCaptureDeps holds dependencies for workflow capture orchestration.
type WorkflowCaptureDeps struct {
	BAS     *BrowserAutomationClient
	Storage *VisualCaptureStorage
	FS      FileIO
	RepoDir string
	RepoID  int64
}

// discoveredWorkflow is a workflow file with parsed metadata.
type discoveredWorkflow struct {
	Path          string
	Name          string
	ExecutionMode string
	Definition    json.RawMessage
}

// CaptureWorkflows orchestrates running BAS workflows for a scenario and
// collecting their recorded video artifacts.
func CaptureWorkflows(ctx context.Context, deps WorkflowCaptureDeps, req WorkflowCaptureRequest) (*WorkflowCaptureResult, error) {
	mode := req.Mode
	if mode == "" {
		mode = CaptureModeCapture
	}

	allowedModes := req.ExecutionModes
	if len(allowedModes) == 0 {
		allowedModes = []string{"observer"}
	}

	workflows, err := discoverWorkflows(deps.FS, deps.RepoDir, req.ScenarioSlug)
	if err != nil {
		return nil, fmt.Errorf("discover workflows: %w", err)
	}

	filtered := filterByExecutionMode(workflows, allowedModes)

	role := SnapshotRoleCapture
	if mode == CaptureModeBaseline {
		role = SnapshotRoleBaseline
	}

	if err := prepareForWorkflowCapture(deps.Storage, deps.RepoID, req.ScenarioSlug, mode); err != nil {
		log.Printf("WARNING: workflow capture cleanup failed for %s: %v", req.ScenarioSlug, err)
	}

	scenarioRoot := filepath.Join(deps.RepoDir, "scenarios", req.ScenarioSlug)

	results, videos, totalSize := executeFilteredWorkflows(ctx, deps, filtered, scenarioRoot)
	results = appendSkippedWorkflows(results, workflows, allowedModes)
	status, captureErr := computeCaptureStatus(results, filtered)

	result := &WorkflowCaptureResult{
		ID:              fmt.Sprintf("%d", time.Now().UnixNano()),
		ScenarioSlug:    req.ScenarioSlug,
		Role:            role,
		WorkflowResults: results,
		CreatedAt:       time.Now().UTC(),
		Status:          status,
		Error:           captureErr,
		SizeBytes:       totalSize,
	}

	if err := deps.Storage.SaveWorkflowCaptureResult(deps.RepoID, req.ScenarioSlug, result, videos); err != nil {
		return nil, fmt.Errorf("save workflow capture: %w", err)
	}

	return result, nil
}

// executeFilteredWorkflows runs each filtered workflow and collects results and video data.
func executeFilteredWorkflows(ctx context.Context, deps WorkflowCaptureDeps, filtered []discoveredWorkflow, scenarioRoot string) ([]WorkflowExecutionResult, map[string][]byte, int64) {
	var results []WorkflowExecutionResult
	var totalSize int64
	videos := map[string][]byte{}

	for _, wf := range filtered {
		wfResult, wfVideos, wfSize := executeSingleWorkflow(ctx, deps, wf, scenarioRoot)
		results = append(results, wfResult)
		for k, v := range wfVideos {
			videos[k] = v
		}
		totalSize += wfSize
	}

	return results, videos, totalSize
}

// executeSingleWorkflow runs one workflow, polls for completion, and fetches video artifacts.
func executeSingleWorkflow(ctx context.Context, deps WorkflowCaptureDeps, wf discoveredWorkflow, scenarioRoot string) (WorkflowExecutionResult, map[string][]byte, int64) {
	wfResult := WorkflowExecutionResult{
		WorkflowName:  wf.Name,
		ExecutionMode: wf.ExecutionMode,
		Status:        "passed",
		VideoStatus:   "none",
	}

	start := time.Now()

	execResp, err := deps.BAS.ExecuteAdhocWorkflow(ctx, BASExecuteAdhocRequest{
		FlowDefinition: wf.Definition,
		Parameters: map[string]interface{}{
			"project_root": scenarioRoot,
		},
	}, true)
	if err != nil {
		wfResult.DurationMs = time.Since(start).Milliseconds()
		wfResult.Status = "error"
		wfResult.Error = err.Error()
		return wfResult, nil, 0
	}

	wfResult.ExecutionID = execResp.ExecutionID

	detail, pollErr := deps.BAS.PollExecutionCompletion(ctx, execResp.ExecutionID, 500*time.Millisecond)
	wfResult.DurationMs = time.Since(start).Milliseconds()

	if pollErr != nil {
		wfResult.Status = "error"
		wfResult.Error = fmt.Sprintf("poll completion: %v", pollErr)
		return wfResult, nil, 0
	}

	if detail.Status != "EXECUTION_STATUS_COMPLETED" {
		wfResult.Status = "failed"
		wfResult.Error = detail.Error
	}

	videos, totalSize := fetchWorkflowVideos(ctx, deps.BAS, execResp.ExecutionID, wf.Name, &wfResult)
	return wfResult, videos, totalSize
}

// fetchWorkflowVideos downloads recorded videos for a workflow execution.
func fetchWorkflowVideos(ctx context.Context, bas *BrowserAutomationClient, executionID, wfName string, wfResult *WorkflowExecutionResult) (map[string][]byte, int64) {
	videosResp, err := bas.GetRecordedVideos(ctx, executionID)
	if err != nil {
		log.Printf("WARNING: failed to get recorded videos for %s: %v", wfName, err)
		wfResult.VideoStatus = "failed"
		return nil, 0
	}
	if len(videosResp.Videos) == 0 {
		return nil, 0
	}

	wfResult.VideoCount = len(videosResp.Videos)
	wfResult.VideoStatus = "captured"

	videos := map[string][]byte{}
	var totalSize int64
	for i, vid := range videosResp.Videos {
		if vid.StorageURL == "" {
			log.Printf("WARNING: no storage URL for video %s in %s", vid.ArtifactID, wfName)
			continue
		}
		data, _, fetchErr := bas.GetVideoData(ctx, vid.StorageURL)
		if fetchErr != nil {
			log.Printf("WARNING: failed to fetch video %s for %s: %v", vid.ArtifactID, wfName, fetchErr)
			continue
		}
		filename := fmt.Sprintf("%s_%d.webm", sanitizeFilename(wfName), i)
		videos[filename] = data
		totalSize += int64(len(data))
	}
	return videos, totalSize
}

// appendSkippedWorkflows adds skipped entries for workflows not in the allowed modes.
func appendSkippedWorkflows(results []WorkflowExecutionResult, workflows []discoveredWorkflow, allowedModes []string) []WorkflowExecutionResult {
	for _, wf := range workflows {
		if !isInAllowedModes(wf.ExecutionMode, allowedModes) {
			results = append(results, WorkflowExecutionResult{
				WorkflowName:  wf.Name,
				ExecutionMode: wf.ExecutionMode,
				Status:        "skipped",
				VideoStatus:   "none",
			})
		}
	}
	return results
}

// computeCaptureStatus determines the overall capture status from workflow results.
func computeCaptureStatus(results []WorkflowExecutionResult, filtered []discoveredWorkflow) (string, string) {
	hasFailure := false
	for _, r := range results {
		if r.Status == "error" || r.Status == "failed" {
			hasFailure = true
			break
		}
	}
	if !hasFailure || len(filtered) == 0 {
		return "complete", ""
	}
	for _, r := range results {
		if r.Status == "passed" {
			return "complete", ""
		}
	}
	return "failed", "all workflow executions failed"
}

// discoverWorkflows walks scenario BAS directories to find workflow JSON files.
func discoverWorkflows(fs FileIO, repoDir, scenarioSlug string) ([]discoveredWorkflow, error) {
	baseDirs := []string{
		filepath.Join(repoDir, "scenarios", scenarioSlug, "bas", "cases"),
		filepath.Join(repoDir, "scenarios", scenarioSlug, "bas", "flows"),
		filepath.Join(repoDir, "scenarios", scenarioSlug, "bas", "actions"),
	}

	var workflows []discoveredWorkflow
	for _, baseDir := range baseDirs {
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".json") {
				return nil
			}

			data, readErr := fs.ReadFile(path)
			if readErr != nil {
				return nil // skip unreadable
			}

			name, execMode, parseErr := parseWorkflowMetadata(data)
			if parseErr != nil {
				return nil // skip unparseable
			}

			workflows = append(workflows, discoveredWorkflow{
				Path:          path,
				Name:          name,
				ExecutionMode: execMode,
				Definition:    json.RawMessage(data),
			})
			return nil
		})
		if err != nil {
			// Directory might not exist, that's fine
			continue
		}
	}

	return workflows, nil
}

// parseWorkflowMetadata extracts name and execution_mode from workflow JSON.
func parseWorkflowMetadata(data []byte) (name, executionMode string, err error) {
	var doc struct {
		Metadata struct {
			Description   string `json:"description"`
			ExecutionMode string `json:"execution_mode"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", "", err
	}

	name = doc.Metadata.Description
	if name == "" {
		name = "unnamed"
	}

	executionMode = doc.Metadata.ExecutionMode
	if executionMode == "" {
		executionMode = "observer" // default to safest
	}

	return name, executionMode, nil
}

// filterByExecutionMode filters workflows to only those with allowed modes.
func filterByExecutionMode(workflows []discoveredWorkflow, allowedModes []string) []discoveredWorkflow {
	var filtered []discoveredWorkflow
	for _, wf := range workflows {
		if isInAllowedModes(wf.ExecutionMode, allowedModes) {
			filtered = append(filtered, wf)
		}
	}
	return filtered
}

func isInAllowedModes(mode string, allowed []string) bool {
	for _, a := range allowed {
		if a == mode {
			return true
		}
	}
	return false
}

// prepareForWorkflowCapture handles baseline clearing for workflow captures.
func prepareForWorkflowCapture(storage *VisualCaptureStorage, repoID int64, scenarioSlug, mode string) error {
	switch mode {
	case CaptureModeBaseline:
		return storage.DeleteWorkflowCapturesByRole(repoID, scenarioSlug, SnapshotRoleBaseline)
	case CaptureModeCapture:
		return storage.DeleteWorkflowCapturesByRole(repoID, scenarioSlug, SnapshotRoleCapture)
	default:
		return nil
	}
}
