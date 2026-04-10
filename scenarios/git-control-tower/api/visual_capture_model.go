package main

import (
	"encoding/json"
	"time"
)

// BAS Client Types (mirror browser-automation-studio API contracts)

type BASViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type BASExecuteAdhocRequest struct {
	FlowDefinition json.RawMessage        `json:"flow_definition"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
}

type BASExecuteResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// BASExecutionDetail is the subset of GET /api/v1/executions/{id} we need for polling.
// BAS responds with protobuf JSON (lowerCamelCase field names).
type BASExecutionDetail struct {
	ExecutionID string `json:"executionId"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type BASExecutionScreenshot struct {
	Screenshot struct {
		ArtifactID   string `json:"artifact_id"`
		Url          string `json:"url"`
		ThumbnailUrl string `json:"thumbnail_url"`
		ContentType  string `json:"content_type"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
	} `json:"screenshot"`
	StepIndex int    `json:"step_index"`
	StepLabel string `json:"step_label"`
	Timestamp string `json:"timestamp"`
}

type BASScreenshotsResponse struct {
	Screenshots []BASExecutionScreenshot `json:"screenshots"`
	Total       int                      `json:"total"`
}

// CapturePreset defines a named capture configuration with viewport dimensions
// and theme. Extensible for future properties (e.g. deviceScaleFactor, locale).
type CapturePreset struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Theme  string `json:"theme"` // "light" | "dark"
}

// Visual Capture API Types

// Snapshot roles distinguish the purpose of each capture.
const (
	SnapshotRoleBaseline = "baseline" // Reference point (the "Before")
	SnapshotRoleCapture  = "capture"  // Comparison capture (the "After")
)

// Capture modes control how a new capture interacts with existing snapshots.
const (
	CaptureModeBaseline = "baseline" // Set new baseline, clear existing captures
	CaptureModeCapture  = "capture"  // Capture current state for comparison against baseline
)

type VisualCaptureRequest struct {
	ScenarioSlug string          `json:"scenarioSlug"`
	Mode         string          `json:"mode,omitempty"`        // "baseline" | "capture" (default: "capture")
	TriggerType  string          `json:"triggerType,omitempty"` // "manual" | "periodic" (default: "manual")
	Presets      []CapturePreset `json:"presets,omitempty"`
	Pages        []string        `json:"pages,omitempty"`
}

type SnapshotSetMeta struct {
	ID                  string          `json:"id"`
	ScenarioSlug        string          `json:"scenarioSlug"`
	Role                string          `json:"role"` // "baseline" | "capture"
	CommitHash          string          `json:"commitHash,omitempty"`
	TriggerType         string          `json:"triggerType"`
	Pages               []string        `json:"pages"`
	ScreenshotCount     int             `json:"screenshotCount"`
	VideoCount          int             `json:"videoCount"`
	VideoStatus         string          `json:"videoStatus,omitempty"` // "not_implemented" | "disabled" | "captured" | "failed"
	CreatedAt           time.Time       `json:"createdAt"`
	SizeBytes           int64           `json:"sizeBytes"`
	Status              string          `json:"status"`
	Error               string          `json:"error,omitempty"`
	Presets             []CapturePreset `json:"presets"`
	PageDiscoveryMethod string          `json:"pageDiscoveryMethod,omitempty"` // "lighthouse" | "fallback" | "explicit"
}

// SnapshotStalenessInfo describes whether the most recent capture is outdated
// relative to the scenario's source files.
type SnapshotStalenessInfo struct {
	IsStale          bool       `json:"isStale"`
	LastFileChange   *time.Time `json:"lastFileChange,omitempty"`
	CaptureCreatedAt *time.Time `json:"captureCreatedAt,omitempty"`
}

// EffectiveRole returns the snapshot's role, defaulting to "capture" for
// legacy snapshots that predate the role field.
func (m SnapshotSetMeta) EffectiveRole() string {
	if m.Role == "" {
		return SnapshotRoleCapture
	}
	return m.Role
}

type SnapshotSetDetail struct {
	SnapshotSetMeta
	Screenshots []SnapshotFile `json:"screenshots"`
	Videos      []SnapshotFile `json:"videos"`
}

type SnapshotFile struct {
	Filename       string `json:"filename"`
	PagePath       string `json:"pagePath,omitempty"`
	PageLabel      string `json:"pageLabel,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	ViewportWidth  int    `json:"viewportWidth,omitempty"`
	ViewportHeight int    `json:"viewportHeight,omitempty"`
	Theme          string `json:"theme,omitempty"`
	SizeBytes      int64  `json:"sizeBytes"`
}

type VisualCaptureStorageStats struct {
	TotalSizeBytes int64                      `json:"totalSizeBytes"`
	SnapshotCount  int                        `json:"snapshotCount"`
	PerScenario    []ScenarioStorageBreakdown `json:"perScenario"`
}

type ScenarioStorageBreakdown struct {
	ScenarioSlug  string `json:"scenarioSlug"`
	SnapshotCount int    `json:"snapshotCount"`
	SizeBytes     int64  `json:"sizeBytes"`
}

type LighthousePage struct {
	ID              string `json:"id"`
	Path            string `json:"path"`
	Label           string `json:"label"`
	WaitForSelector string `json:"waitForSelector,omitempty"`
}

type LighthouseConfig struct {
	Enabled bool             `json:"enabled"`
	Pages   []LighthousePage `json:"pages"`
}

// BAS Recorded Videos Types

type BASVideoArtifact struct {
	ArtifactID  string `json:"artifact_id"`
	StorageURL  string `json:"storage_url"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type BASRecordedVideosResponse struct {
	ExecutionID string             `json:"execution_id"`
	Videos      []BASVideoArtifact `json:"videos"`
}

// Workflow Capture Types

type WorkflowCaptureRequest struct {
	ScenarioSlug   string          `json:"scenarioSlug"`
	Mode           string          `json:"mode,omitempty"`        // "baseline" | "capture" (default: "capture")
	TriggerType    string          `json:"triggerType,omitempty"` // "manual" (default)
	Presets        []CapturePreset `json:"presets,omitempty"`
	ExecutionModes []string        `json:"executionModes,omitempty"` // filter: ["observer"], ["observer","mutating"], etc.
}

type WorkflowExecutionResult struct {
	WorkflowName  string `json:"workflowName"`
	ExecutionMode string `json:"executionMode"`
	ExecutionID   string `json:"executionId,omitempty"`
	Status        string `json:"status"` // "passed" | "failed" | "skipped" | "error"
	Error         string `json:"error,omitempty"`
	DurationMs    int64  `json:"durationMs"`
	VideoCount    int    `json:"videoCount"`
	VideoStatus   string `json:"videoStatus,omitempty"` // "captured" | "failed" | "none"
}

type WorkflowCaptureResult struct {
	ID              string                    `json:"id"`
	ScenarioSlug    string                    `json:"scenarioSlug"`
	Role            string                    `json:"role"` // "baseline" | "capture"
	WorkflowResults []WorkflowExecutionResult `json:"workflowResults"`
	CreatedAt       time.Time                 `json:"createdAt"`
	Status          string                    `json:"status"` // "complete" | "failed"
	Error           string                    `json:"error,omitempty"`
	SizeBytes       int64                     `json:"sizeBytes"`
}
