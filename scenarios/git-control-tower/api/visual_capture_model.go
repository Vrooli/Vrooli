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

type BASScreenshotRequest struct {
	URL      string      `json:"url"`
	Viewport BASViewport `json:"viewport"`
}

type BASScreenshotResponse struct {
	Screenshot     string `json:"screenshot"`
	URL            string `json:"url"`
	DurationMS     int    `json:"duration_ms"`
	ViewportWidth  int    `json:"viewportWidth"`
	ViewportHeight int    `json:"viewportHeight"`
}

type BASExecuteAdhocRequest struct {
	FlowDefinition    json.RawMessage        `json:"flow_definition"`
	WaitForCompletion bool                   `json:"wait_for_completion"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

type BASExecuteResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type BASExecutionScreenshot struct {
	Screenshot struct {
		Data     string `json:"data"`
		MimeType string `json:"mime_type"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
	} `json:"screenshot"`
	StepIndex int    `json:"step_index"`
	StepLabel string `json:"step_label"`
	Timestamp string `json:"timestamp"`
}

type BASScreenshotsResponse struct {
	Screenshots []BASExecutionScreenshot `json:"screenshots"`
	Total       int                      `json:"total"`
}

// Visual Capture API Types

type VisualCaptureRequest struct {
	ScenarioSlug string      `json:"scenarioSlug"`
	Viewport     BASViewport `json:"viewport,omitempty"`
	Pages        []string    `json:"pages,omitempty"`
}

type SnapshotSetMeta struct {
	ID                  string    `json:"id"`
	ScenarioSlug        string    `json:"scenarioSlug"`
	CommitHash          string    `json:"commitHash,omitempty"`
	TriggerType         string    `json:"triggerType"`
	Pages               []string  `json:"pages"`
	ScreenshotCount     int       `json:"screenshotCount"`
	VideoCount          int       `json:"videoCount"`
	CreatedAt           time.Time `json:"createdAt"`
	SizeBytes           int64     `json:"sizeBytes"`
	Status              string    `json:"status"`
	Error               string    `json:"error,omitempty"`
	PageDiscoveryMethod string    `json:"pageDiscoveryMethod,omitempty"` // "lighthouse" | "fallback" | "explicit"
}

type SnapshotSetDetail struct {
	SnapshotSetMeta
	Screenshots []SnapshotFile `json:"screenshots"`
	Videos      []SnapshotFile `json:"videos"`
}

type SnapshotFile struct {
	Filename  string `json:"filename"`
	PagePath  string `json:"pagePath,omitempty"`
	PageLabel string `json:"pageLabel,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int64  `json:"sizeBytes"`
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
