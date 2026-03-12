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

	// Discover workflow files
	workflows, err := discoverWorkflows(deps.FS, deps.RepoDir, req.ScenarioSlug)
	if err != nil {
		return nil, fmt.Errorf("discover workflows: %w", err)
	}

	// Filter by execution mode
	filtered := filterByExecutionMode(workflows, allowedModes)

	captureID := fmt.Sprintf("%d", time.Now().UnixNano())
	role := SnapshotRoleCapture
	if mode == CaptureModeBaseline {
		role = SnapshotRoleBaseline
	}

	// Pre-capture cleanup
	if err := prepareForWorkflowCapture(deps.Storage, deps.RepoID, req.ScenarioSlug, mode); err != nil {
		log.Printf("WARNING: workflow capture cleanup failed for %s: %v", req.ScenarioSlug, err)
	}

	// Compute scenario root so BAS can resolve @selector/ references via selectors.manifest.json
	scenarioRoot := filepath.Join(deps.RepoDir, "scenarios", req.ScenarioSlug)

	var results []WorkflowExecutionResult
	var totalSize int64
	videos := map[string][]byte{}

	for _, wf := range filtered {
		wfResult := WorkflowExecutionResult{
			WorkflowName:  wf.Name,
			ExecutionMode: wf.ExecutionMode,
			Status:        "passed",
			VideoStatus:   "none",
		}

		start := time.Now()

		// Execute the workflow via BAS (adhoc endpoint returns immediately)
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
			results = append(results, wfResult)
			continue
		}

		wfResult.ExecutionID = execResp.ExecutionID

		// Poll until the execution completes (adhoc endpoint doesn't wait)
		detail, pollErr := deps.BAS.PollExecutionCompletion(ctx, execResp.ExecutionID, 500*time.Millisecond)
		wfResult.DurationMs = time.Since(start).Milliseconds()

		if pollErr != nil {
			wfResult.Status = "error"
			wfResult.Error = fmt.Sprintf("poll completion: %v", pollErr)
			results = append(results, wfResult)
			continue
		}

		if detail.Status != "EXECUTION_STATUS_COMPLETED" {
			wfResult.Status = "failed"
			wfResult.Error = detail.Error
		}

		// Fetch recorded videos
		videosResp, err := deps.BAS.GetRecordedVideos(ctx, execResp.ExecutionID)
		if err != nil {
			log.Printf("WARNING: failed to get recorded videos for %s: %v", wf.Name, err)
			wfResult.VideoStatus = "failed"
		} else if len(videosResp.Videos) > 0 {
			wfResult.VideoCount = len(videosResp.Videos)
			wfResult.VideoStatus = "captured"

			// Download each video
			for i, vid := range videosResp.Videos {
				data, _, fetchErr := deps.BAS.GetVideoData(ctx, execResp.ExecutionID, vid.ArtifactID)
				if fetchErr != nil {
					log.Printf("WARNING: failed to fetch video %s for %s: %v", vid.ArtifactID, wf.Name, fetchErr)
					continue
				}
				filename := fmt.Sprintf("%s_%d.webm", sanitizeFilename(wf.Name), i)
				videos[filename] = data
				totalSize += int64(len(data))
			}
		}

		results = append(results, wfResult)
	}

	// Mark skipped workflows
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

	status := "complete"
	var captureErr string
	hasFailure := false
	for _, r := range results {
		if r.Status == "error" || r.Status == "failed" {
			hasFailure = true
			break
		}
	}
	if hasFailure && len(filtered) > 0 {
		allFailed := true
		for _, r := range results {
			if r.Status == "passed" {
				allFailed = false
				break
			}
		}
		if allFailed {
			status = "failed"
			captureErr = "all workflow executions failed"
		}
	}

	result := &WorkflowCaptureResult{
		ID:              captureID,
		ScenarioSlug:    req.ScenarioSlug,
		Role:            role,
		WorkflowResults: results,
		CreatedAt:       time.Now().UTC(),
		Status:          status,
		Error:           captureErr,
		SizeBytes:       totalSize,
	}

	// Save to storage
	if err := deps.Storage.SaveWorkflowCaptureResult(deps.RepoID, req.ScenarioSlug, result, videos); err != nil {
		return nil, fmt.Errorf("save workflow capture: %w", err)
	}

	return result, nil
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
