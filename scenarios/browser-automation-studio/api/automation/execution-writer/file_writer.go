package executionwriter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/storage"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FileWriter persists step outcomes and telemetry to JSON files on disk.
// The database stores only execution index data (status, progress, result_path).
// Detailed execution data lives in JSON files at the result_path location.
//
// Artifact collection is controlled by ArtifactCollectionSettings which can be set
// via SetArtifactConfig(). When not set, defaults to "full" profile (all artifacts).
type FileWriter struct {
	repo    ExecutionIndexRepository
	storage storage.StorageInterface
	log     *logrus.Logger
	dataDir string // Base directory for execution result files

	// Mutex for concurrent file writes
	mu sync.Mutex
	// Cache of execution result data being built up
	results sync.Map // executionID -> *ExecutionResultData
	// Cache of proto timeline data being built up (preferred on-disk format).
	timelines sync.Map // executionID -> *executionTimelineData

	// Artifact collection configuration - controls what artifacts are persisted.
	// Protected by mu for thread-safe access.
	artifactConfig config.ArtifactCollectionSettings
}

// ExecutionResultData accumulates execution results to be written to disk.
// This is the file format stored at ExecutionIndex.ResultPath.
type ExecutionResultData struct {
	ExecutionID     string              `json:"execution_id"`
	WorkflowID      string              `json:"workflow_id"`
	Steps           []StepResultData    `json:"steps"`
	Artifacts       []ArtifactData      `json:"artifacts"`
	Telemetry       []TelemetryData     `json:"telemetry"`
	TimelineFrame   []TimelineFrameData `json:"timeline_frames"`
	Summary         ExecutionSummary    `json:"summary"`
	mu              sync.Mutex          `json:"-"`
}

type executionTimelineData struct {
	mu sync.Mutex
	pb *bastimeline.ExecutionTimeline
}

// StepResultData captures individual step execution results.
type StepResultData struct {
	StepID      string         `json:"step_id"`
	StepIndex   int            `json:"step_index"`
	NodeID      string         `json:"node_id"`
	StepType    string         `json:"step_type"`
	Status      string         `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	DurationMs  int            `json:"duration_ms"`
	Error       string         `json:"error,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ArtifactData captures execution artifacts (screenshots, DOM snapshots, etc).
type ArtifactData struct {
	ArtifactID   string         `json:"artifact_id"`
	StepID       string         `json:"step_id,omitempty"`
	StepIndex    *int           `json:"step_index,omitempty"`
	ArtifactType string         `json:"artifact_type"`
	Label        string         `json:"label,omitempty"`
	StorageURL   string         `json:"storage_url,omitempty"`
	ThumbnailURL string         `json:"thumbnail_url,omitempty"`
	ContentType  string         `json:"content_type,omitempty"`
	SizeBytes    *int64         `json:"size_bytes,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

// TelemetryData captures step telemetry for debugging.
type TelemetryData struct {
	StepIndex int                     `json:"step_index"`
	Data      contracts.StepTelemetry `json:"data"`
	Timestamp time.Time               `json:"timestamp"`
}

// TimelineFrameData captures timeline frame data for replay.
type TimelineFrameData struct {
	StepIndex             int            `json:"step_index"`
	NodeID                string         `json:"node_id"`
	StepType              string         `json:"step_type"`
	ScreenshotURL         string         `json:"screenshot_url,omitempty"`
	ScreenshotPath        string         `json:"screenshot_path,omitempty"`
	ScreenshotArtifactID  string         `json:"screenshot_artifact_id,omitempty"`
	DOMSnapshotArtifactID string         `json:"dom_snapshot_artifact_id,omitempty"`
	Success               bool           `json:"success"`
	Attempt               int            `json:"attempt"`
	DurationMs            int            `json:"duration_ms"`
	Payload               map[string]any `json:"payload,omitempty"`
}

// ExecutionSummary provides aggregate statistics.
type ExecutionSummary struct {
	TotalSteps      int       `json:"total_steps"`
	CompletedSteps  int       `json:"completed_steps"`
	FailedSteps     int       `json:"failed_steps"`
	TotalDurationMs int       `json:"total_duration_ms"`
	LastUpdated     time.Time `json:"last_updated"`
}

// NewFileWriter constructs an ExecutionWriter that writes to files and updates the DB index.
// By default, uses the "full" artifact profile (all artifacts collected).
// Call SetArtifactConfig() to customize artifact collection before starting execution.
func NewFileWriter(repo ExecutionIndexRepository, storage storage.StorageInterface, log *logrus.Logger, dataDir string) *FileWriter {
	if dataDir == "" {
		dataDir = "/tmp/bas-executions"
	}
	return &FileWriter{
		repo:           repo,
		storage:        storage,
		log:            log,
		dataDir:        dataDir,
		artifactConfig: config.DefaultArtifactSettings(),
	}
}

// SetArtifactConfig updates the artifact collection settings for this writer.
// Call this before starting execution to configure what artifacts are collected.
// Pass nil to reset to default "full" profile.
func (r *FileWriter) SetArtifactConfig(cfg *config.ArtifactCollectionSettings) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cfg == nil {
		r.artifactConfig = config.DefaultArtifactSettings()
	} else {
		r.artifactConfig = *cfg
	}
}

// GetArtifactConfig returns the current artifact collection settings.
func (r *FileWriter) GetArtifactConfig() config.ArtifactCollectionSettings {
	if r == nil {
		return config.DefaultArtifactSettings()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.artifactConfig
}

const (
	resultFileName        = "result.json"
	protoTimelineFileName = "timeline.proto.json"
	readmeFileName        = "README.md"
)

// getOrCreateResult gets or creates the result data for an execution.
func (r *FileWriter) getOrCreateResult(plan contracts.ExecutionPlan) *ExecutionResultData {
	key := plan.ExecutionID.String()
	if val, ok := r.results.Load(key); ok {
		return val.(*ExecutionResultData)
	}

	result := &ExecutionResultData{
		ExecutionID:     plan.ExecutionID.String(),
		WorkflowID:      plan.WorkflowID.String(),
		Steps:           make([]StepResultData, 0),
		Artifacts:       make([]ArtifactData, 0),
		Telemetry:       make([]TelemetryData, 0),
		TimelineFrame:   make([]TimelineFrameData, 0),
		Summary: ExecutionSummary{
			LastUpdated: time.Now().UTC(),
		},
	}
	r.results.Store(key, result)
	return result
}

func (r *FileWriter) getOrCreateTimeline(plan contracts.ExecutionPlan) *executionTimelineData {
	key := plan.ExecutionID.String()
	if val, ok := r.timelines.Load(key); ok {
		return val.(*executionTimelineData)
	}

	pb := &bastimeline.ExecutionTimeline{
		ExecutionId: plan.ExecutionID.String(),
		WorkflowId:  plan.WorkflowID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_PENDING,
		Progress:    0,
	}
	entry := &executionTimelineData{pb: pb}
	r.timelines.Store(key, entry)
	return entry
}

// resultFilePath returns the path where execution results should be stored.
func (r *FileWriter) resultFilePath(executionID uuid.UUID) string {
	return filepath.Join(r.dataDir, executionID.String(), resultFileName)
}

func (r *FileWriter) protoTimelineFilePath(executionID uuid.UUID) string {
	return filepath.Join(r.dataDir, executionID.String(), protoTimelineFileName)
}

func (r *FileWriter) readmeFilePath(executionID uuid.UUID) string {
	return filepath.Join(r.dataDir, executionID.String(), readmeFileName)
}

// writeResultFile persists the execution result data to disk.
func (r *FileWriter) writeResultFile(executionID uuid.UUID, result *ExecutionResultData, timeline *executionTimelineData) error {
	if result != nil {
		result.mu.Lock()
		result.Summary.LastUpdated = time.Now().UTC()
		result.mu.Unlock()
	}

	payload, err := r.buildResultPayload(executionID, result, timeline)
	if err != nil {
		return err
	}

	filePath := r.resultFilePath(executionID)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create result directory: %w", err)
	}

	if err := r.writeReadmeFile(executionID); err != nil {
		return err
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write result file: %w", err)
	}

	return nil
}

func (r *FileWriter) writeReadmeFile(executionID uuid.UUID) error {
	path := r.readmeFilePath(executionID)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return nil
	}

	content := []byte(`# Browser Automation Studio Execution Artifacts

This folder contains artifacts captured during a single workflow execution.

## Top-level files
- execution.proto.json: execution metadata snapshot (status, timestamps, workflow ID).
- result.json: normalized execution timeline (proto-aligned JSON).
- timeline.proto.json: raw timeline protobuf JSON.

## artifacts/
- artifacts/screenshots: per-step replay screenshots (ordered by step number).
- artifacts/videos: Playwright recordings (per page).
- artifacts/traces: Playwright traces (.zip).
- artifacts/har: HAR captures (.har).
- artifacts/dom: per-step DOM snapshots (.html) when enabled.

If an artifact type is missing, it was not requested or the engine failed to produce it.
`)

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write readme file: %w", err)
	}
	return nil
}

func (r *FileWriter) buildResultPayload(executionID uuid.UUID, result *ExecutionResultData, timeline *executionTimelineData) (map[string]any, error) {
	payload := map[string]any{}

	if timeline != nil && timeline.pb != nil {
		raw, err := protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		}.Marshal(timeline.pb)
		if err != nil {
			return nil, fmt.Errorf("marshal timeline payload: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("parse timeline payload: %w", err)
		}
	}

	if len(payload) == 0 {
		payload["execution_id"] = executionID.String()
		if result != nil && strings.TrimSpace(result.WorkflowID) != "" {
			payload["workflow_id"] = result.WorkflowID
		}
		return payload, nil
	}

	if result != nil && result.Summary.TotalSteps > 0 {
		status := "EXECUTION_STATUS_RUNNING"
		if result.Summary.FailedSteps > 0 {
			status = "EXECUTION_STATUS_FAILED"
		} else if result.Summary.CompletedSteps >= result.Summary.TotalSteps {
			status = "EXECUTION_STATUS_COMPLETED"
		}
		payload["status"] = status
	}

	entriesRaw, ok := payload["entries"].([]any)
	if !ok || len(entriesRaw) == 0 {
		return payload, nil
	}

	for _, entryRaw := range entriesRaw {
		entry, ok := entryRaw.(map[string]any)
		if !ok {
			continue
		}
		if aggregates, ok := entry["aggregates"].(map[string]any); ok {
			delete(aggregates, "artifacts")
		}
	}

	return payload, nil
}

func (r *FileWriter) writeProtoTimelineFile(executionID uuid.UUID, timeline *executionTimelineData) error {
	if timeline == nil || timeline.pb == nil {
		return nil
	}

	timeline.mu.Lock()
	defer timeline.mu.Unlock()

	filePath := r.protoTimelineFilePath(executionID)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create proto timeline directory: %w", err)
	}

	raw, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(timeline.pb)
	if err != nil {
		return fmt.Errorf("marshal proto timeline: %w", err)
	}

	indented := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err == nil {
		indented = buf.Bytes()
	}

	if err := os.WriteFile(filePath, indented, 0o644); err != nil {
		return fmt.Errorf("write proto timeline file: %w", err)
	}
	return nil
}

// RecordStepOutcome stores the execution step and key artifacts to files.
// Respects the artifact collection settings configured via SetArtifactConfig().
func (r *FileWriter) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error) {
	if r == nil {
		return RecordResult{}, nil
	}

	// Get current artifact config (thread-safe copy)
	cfg := r.GetArtifactConfig()

	// Apply configurable limits during sanitization
	outcome = r.sanitizeOutcomeWithConfig(outcome, cfg)
	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)

	stepID := uuid.New()
	artifactIDs := make([]uuid.UUID, 0, 8)
	protoArtifacts := make([]*bastimeline.TimelineArtifact, 0, 8)

	// Build step result (always recorded - core execution data)
	step := StepResultData{
		StepID:      stepID.String(),
		StepIndex:   outcome.StepIndex,
		NodeID:      outcome.NodeID,
		StepType:    outcome.StepType,
		Status:      statusFromOutcome(outcome),
		StartedAt:   outcome.StartedAt,
		CompletedAt: outcome.CompletedAt,
		DurationMs:  outcome.DurationMs,
		Metadata: map[string]any{
			"attempt":         outcome.Attempt,
			"payload_version": outcome.PayloadVersion,
		},
	}
	if outcome.Failure != nil {
		step.Metadata["failure"] = outcome.Failure
		step.Error = outcome.Failure.Message
	}

	result.mu.Lock()
	result.Steps = append(result.Steps, step)
	result.Summary.TotalSteps = len(result.Steps)
	if step.Status == "completed" {
		result.Summary.CompletedSteps++
	} else {
		result.Summary.FailedSteps++
	}
	result.Summary.TotalDurationMs += outcome.DurationMs
	result.mu.Unlock()

	// Store core outcome payload as artifact (always recorded - essential for debugging)
	outcomeArtifactID := uuid.New()
	outcomeArtifact := ArtifactData{
		ArtifactID:   outcomeArtifactID.String(),
		StepID:       stepID.String(),
		StepIndex:    &outcome.StepIndex,
		ArtifactType: "step_outcome",
		Payload:      map[string]any{"outcome": outcome},
	}
	result.mu.Lock()
	result.Artifacts = append(result.Artifacts, outcomeArtifact)
	result.mu.Unlock()
	artifactIDs = append(artifactIDs, outcomeArtifactID)
	protoArtifacts = append(protoArtifacts, artifactDataToProto(&outcomeArtifact))

	artifactBaseName := buildScreenshotBaseName(plan, outcome)
	var consoleArtifact *telemetryArtifactRef
	var networkArtifact *telemetryArtifactRef

	// Console logs - controlled by CollectConsoleLogs
	if cfg.CollectConsoleLogs && len(outcome.ConsoleLogs) > 0 {
		payload, err := json.Marshal(outcome.ConsoleLogs)
		if err != nil && r.log != nil {
			r.log.WithError(err).Warn("Failed to serialize console logs")
		} else if err == nil {
			ref, err := r.persistTelemetryArtifact(ctx, plan.ExecutionID, "console", artifactBaseName, ".json", "application/json", payload)
			if err != nil && r.log != nil {
				r.log.WithError(err).Warn("Failed to persist console log artifact")
			} else if ref != nil {
				consoleArtifact = ref
			}
		}
	}

	// Network events - controlled by CollectNetworkEvents
	if cfg.CollectNetworkEvents && len(outcome.Network) > 0 {
		payload, err := json.Marshal(outcome.Network)
		if err != nil && r.log != nil {
			r.log.WithError(err).Warn("Failed to serialize network events")
		} else if err == nil {
			ref, err := r.persistTelemetryArtifact(ctx, plan.ExecutionID, "network", artifactBaseName, ".json", "application/json", payload)
			if err != nil && r.log != nil {
				r.log.WithError(err).Warn("Failed to persist network event artifact")
			} else if ref != nil {
				networkArtifact = ref
			}
		}
	}

	// Assertion result - controlled by CollectAssertions
	if cfg.CollectAssertions && outcome.Assertion != nil {
		id := uuid.New()
		artifact := ArtifactData{
			ArtifactID:   id.String(),
			StepID:       stepID.String(),
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "assertion",
			Label:        deriveStepLabel(outcome),
			Payload:      map[string]any{"assertion": outcome.Assertion},
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
		artifactIDs = append(artifactIDs, id)
		protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
	}

	// Extracted data - controlled by CollectExtractedData
	if cfg.CollectExtractedData && outcome.ExtractedData != nil && len(outcome.ExtractedData) > 0 {
		id := uuid.New()
		artifact := ArtifactData{
			ArtifactID:   id.String(),
			StepID:       stepID.String(),
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "extracted_data",
			Label:        deriveStepLabel(outcome),
			Payload:      map[string]any{"value": outcome.ExtractedData},
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
		artifactIDs = append(artifactIDs, id)
		protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
	}

	// Engine-provided metadata (video/trace/HAR paths) - always collected if present
	if outcome.Notes != nil {
		for _, key := range []string{"video_path", "trace_path", "har_path"} {
			if path := strings.TrimSpace(outcome.Notes[key]); path != "" {
				if id := r.persistExternalFile(result, stepID.String(), outcome.StepIndex, strings.TrimSuffix(key, "_path")+"_meta", path); id != nil {
					artifactIDs = append(artifactIDs, *id)
				}
			}
		}
	}

	var timelineScreenshotURL string
	var timelineScreenshotThumbURL string
	var timelineScreenshotID *uuid.UUID
	var timelineScreenshotPath string
	var timelineScreenshotSizeBytes int64

	// Persist screenshot if available - controlled by CollectScreenshots
	if cfg.CollectScreenshots && outcome.Screenshot != nil && len(outcome.Screenshot.Data) > 0 {
		screenshotInfo, err := r.persistScreenshot(ctx, plan, plan.ExecutionID, outcome)
		if err != nil && r.log != nil {
			r.log.WithError(err).Warn("Failed to persist screenshot artifact")
		}
		if screenshotInfo != nil {
			id := uuid.New()
			artifact := ArtifactData{
				ArtifactID:   id.String(),
				StepID:       stepID.String(),
				StepIndex:    &outcome.StepIndex,
				ArtifactType: "screenshot",
				StorageURL:   screenshotInfo.URL,
				ThumbnailURL: screenshotInfo.ThumbnailURL,
				SizeBytes:    &screenshotInfo.SizeBytes,
				Payload: map[string]any{
					"width":      screenshotInfo.Width,
					"height":     screenshotInfo.Height,
					"from_cache": outcome.Screenshot.FromCache,
					"hash":       outcome.Screenshot.Hash,
				},
			}
			if screenshotInfo.Path != "" {
				artifact.Payload["path"] = screenshotInfo.Path
			}
			if screenshotInfo.ObjectName != "" {
				artifact.Payload["storage_object"] = screenshotInfo.ObjectName
			}
			result.mu.Lock()
			result.Artifacts = append(result.Artifacts, artifact)
			result.mu.Unlock()
			artifactIDs = append(artifactIDs, id)
			protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
			timelineScreenshotID = &id
			timelineScreenshotURL = screenshotInfo.URL
			timelineScreenshotThumbURL = screenshotInfo.ThumbnailURL
			timelineScreenshotPath = screenshotInfo.Path
			timelineScreenshotSizeBytes = screenshotInfo.SizeBytes
		} else if r.log != nil {
			r.log.WithFields(logrus.Fields{
				"execution_id": plan.ExecutionID,
				"step_index":   outcome.StepIndex,
				"node_id":      outcome.NodeID,
			}).Warn("Screenshot capture skipped: storage unavailable")
		}
	}

	var domSnapshotArtifact *telemetryArtifactRef

	// DOM snapshot - controlled by CollectDOMSnapshots
	if cfg.CollectDOMSnapshots && outcome.DOMSnapshot != nil && outcome.DOMSnapshot.HTML != "" {
		html := outcome.DOMSnapshot.HTML
		maxBytes := cfg.MaxDOMSnapshotBytes
		if maxBytes <= 0 {
			maxBytes = contracts.DOMSnapshotMaxBytes
		}
		if len(html) > maxBytes {
			html = html[:maxBytes]
			outcome.DOMSnapshot.Truncated = true
		}
		ref, err := r.persistTelemetryArtifact(ctx, plan.ExecutionID, "dom", artifactBaseName, ".html", "text/html", []byte(html))
		if err != nil && r.log != nil {
			r.log.WithError(err).Warn("Failed to persist DOM snapshot artifact")
		} else if ref != nil {
			domSnapshotArtifact = ref
		}
	}

	// Build timeline frame
	timelinePayload := buildTimelinePayload(outcome, timelineScreenshotURL, timelineScreenshotID, artifactIDs)
	timelineFrame := TimelineFrameData{
		StepIndex:      outcome.StepIndex,
		NodeID:         outcome.NodeID,
		StepType:       outcome.StepType,
		ScreenshotURL:  timelineScreenshotURL,
		ScreenshotPath: timelineScreenshotPath,
		Success:        outcome.Success,
		Attempt:        outcome.Attempt,
		DurationMs:     outcome.DurationMs,
		Payload:        timelinePayload,
	}
	if timelineScreenshotID != nil {
		timelineFrame.ScreenshotArtifactID = timelineScreenshotID.String()
	}
	result.mu.Lock()
	result.TimelineFrame = append(result.TimelineFrame, timelineFrame)
	result.mu.Unlock()

	if err := r.appendProtoTimelineEntry(plan, outcome, protoArtifacts, timelineScreenshotID, timelineScreenshotURL, timelineScreenshotThumbURL, timelineScreenshotPath, timelineScreenshotSizeBytes, domSnapshotArtifact, consoleArtifact, networkArtifact, timeline); err != nil {
		if r.log != nil {
			r.log.WithError(err).Warn("Failed to write proto timeline file")
		}
	}
	// Write result to file
	if err := r.writeResultFile(plan.ExecutionID, result, timeline); err != nil {
		if r.log != nil {
			r.log.WithError(err).Warn("Failed to write execution result file")
		}
	}

	// Update database index with result path
	if r.repo != nil {
		resultPath := r.resultFilePath(plan.ExecutionID)
		if err := r.updateExecutionIndex(ctx, plan.ExecutionID, resultPath); err != nil {
			if r.log != nil {
				r.log.WithError(err).Warn("Failed to update execution index")
			}
		}
	}

	timelineID := uuid.New()
	return RecordResult{
		StepID:             stepID,
		ArtifactIDs:        artifactIDs,
		TimelineArtifactID: &timelineID,
	}, nil
}

// RecordExecutionArtifacts persists execution-level artifacts (video/trace/HAR) into result.json.
func (r *FileWriter) RecordExecutionArtifacts(ctx context.Context, plan contracts.ExecutionPlan, artifacts []ExternalArtifact) error {
	if r == nil || len(artifacts) == 0 {
		return nil
	}

	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)

	const maxEmbeddedExternalBytes = 5 * 1024 * 1024

	for _, item := range artifacts {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("path", path).Debug("external artifact not readable")
			}
			continue
		}

		artifactType := strings.TrimSpace(item.ArtifactType)
		if artifactType == "" {
			artifactType = "custom"
		}
		isVideo := isVideoArtifactType(artifactType)
		allowInline := !isNonInlineArtifactType(artifactType)

		payload := map[string]any{
			"path":       path,
			"size_bytes": info.Size(),
		}
		for key, value := range item.Payload {
			payload[key] = value
		}

		if allowInline && info.Size() > 0 && info.Size() <= maxEmbeddedExternalBytes {
			if data, readErr := os.ReadFile(path); readErr == nil {
				payload["base64"] = base64.StdEncoding.EncodeToString(data)
				payload["inline"] = true
			}
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = artifactType
		}

		var storageURL string
		contentType := strings.TrimSpace(item.ContentType)
		var sizeBytes *int64
		if r.storage != nil && isVideo {
			artifactInfo, storeErr := r.storage.StoreArtifactFromFile(ctx, plan.ExecutionID, label, path, item.ContentType)
			if storeErr != nil {
				if r.log != nil {
					r.log.WithError(storeErr).WithField("path", path).Warn("Failed to store video artifact")
				}
			} else if artifactInfo != nil {
				storageURL = artifactInfo.URL
				if strings.TrimSpace(artifactInfo.ContentType) != "" {
					contentType = artifactInfo.ContentType
				}
				size := artifactInfo.SizeBytes
				sizeBytes = &size
				payload["storage_object"] = artifactInfo.ObjectName
			}
		}

		artifact := ArtifactData{
			ArtifactID:   uuid.New().String(),
			ArtifactType: artifactType,
			Label:        label,
			Payload:      payload,
			StorageURL:   storageURL,
			ContentType:  contentType,
			SizeBytes:    sizeBytes,
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
	}

	return r.writeResultFile(plan.ExecutionID, result, timeline)
}

func isVideoArtifactType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "video_meta":
		return true
	default:
		return false
	}
}

func isNonInlineArtifactType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "video_meta", "trace", "trace_meta", "har", "har_meta":
		return true
	default:
		return false
	}
}

func (r *FileWriter) appendProtoTimelineEntry(
	plan contracts.ExecutionPlan,
	outcome contracts.StepOutcome,
	artifacts []*bastimeline.TimelineArtifact,
	screenshotArtifactID *uuid.UUID,
	screenshotURL string,
	screenshotThumbURL string,
	screenshotPath string,
	screenshotSizeBytes int64,
	domSnapshotArtifact *telemetryArtifactRef,
	consoleArtifact *telemetryArtifactRef,
	networkArtifact *telemetryArtifactRef,
	timeline *executionTimelineData,
) error {
	if r == nil || timeline == nil || timeline.pb == nil {
		return nil
	}

	entry := stepOutcomeToTimelineEntry(outcome, plan.ExecutionID)
	if entry == nil {
		return nil
	}

	if screenshotArtifactID != nil && strings.TrimSpace(screenshotURL) != "" {
		if entry.Telemetry == nil {
			entry.Telemetry = &basdomain.ActionTelemetry{}
		}
		if entry.Telemetry.Screenshot == nil {
			entry.Telemetry.Screenshot = &basdomain.TimelineScreenshot{}
		}
		entry.Telemetry.Screenshot.ArtifactId = screenshotArtifactID.String()
		entry.Telemetry.Screenshot.Url = screenshotURL
		if strings.TrimSpace(screenshotThumbURL) != "" {
			entry.Telemetry.Screenshot.ThumbnailUrl = screenshotThumbURL
		}
		if screenshotPath != "" {
			entry.Telemetry.Screenshot.Path = &screenshotPath
		}
		if screenshotSizeBytes > 0 {
			size := screenshotSizeBytes
			entry.Telemetry.Screenshot.SizeBytes = &size
		}
		if outcome.Screenshot != nil {
			if outcome.Screenshot.Width > 0 {
				entry.Telemetry.Screenshot.Width = int32(outcome.Screenshot.Width)
			}
			if outcome.Screenshot.Height > 0 {
				entry.Telemetry.Screenshot.Height = int32(outcome.Screenshot.Height)
			}
			if outcome.Screenshot.MediaType != "" {
				entry.Telemetry.Screenshot.ContentType = outcome.Screenshot.MediaType
			}
		}
	}
	if domSnapshotArtifact != nil {
		if entry.Telemetry == nil {
			entry.Telemetry = &basdomain.ActionTelemetry{}
		}
		entry.Telemetry.DomSnapshot = &basdomain.TelemetryArtifact{
			ArtifactId:  domSnapshotArtifact.ID.String(),
			StorageUrl:  domSnapshotArtifact.URL,
			ContentType: domSnapshotArtifact.ContentType,
		}
		if domSnapshotArtifact.Path != "" {
			entry.Telemetry.DomSnapshot.Path = &domSnapshotArtifact.Path
		}
		if domSnapshotArtifact.SizeBytes > 0 {
			size := domSnapshotArtifact.SizeBytes
			entry.Telemetry.DomSnapshot.SizeBytes = &size
		}
	}
	if consoleArtifact != nil {
		if entry.Telemetry == nil {
			entry.Telemetry = &basdomain.ActionTelemetry{}
		}
		entry.Telemetry.ConsoleLogArtifact = &basdomain.TelemetryArtifact{
			ArtifactId:  consoleArtifact.ID.String(),
			StorageUrl:  consoleArtifact.URL,
			ContentType: consoleArtifact.ContentType,
		}
		if consoleArtifact.Path != "" {
			entry.Telemetry.ConsoleLogArtifact.Path = &consoleArtifact.Path
		}
		if consoleArtifact.SizeBytes > 0 {
			size := consoleArtifact.SizeBytes
			entry.Telemetry.ConsoleLogArtifact.SizeBytes = &size
		}
	}
	if networkArtifact != nil {
		if entry.Telemetry == nil {
			entry.Telemetry = &basdomain.ActionTelemetry{}
		}
		entry.Telemetry.NetworkEventArtifact = &basdomain.TelemetryArtifact{
			ArtifactId:  networkArtifact.ID.String(),
			StorageUrl:  networkArtifact.URL,
			ContentType: networkArtifact.ContentType,
		}
		if networkArtifact.Path != "" {
			entry.Telemetry.NetworkEventArtifact.Path = &networkArtifact.Path
		}
		if networkArtifact.SizeBytes > 0 {
			size := networkArtifact.SizeBytes
			entry.Telemetry.NetworkEventArtifact.SizeBytes = &size
		}
	}
	if entry.Telemetry == nil {
		entry.Telemetry = &basdomain.ActionTelemetry{}
	}
	if outcome.ElementBoundingBox != nil {
		entry.Telemetry.ElementBoundingBox = outcome.ElementBoundingBox
	}
	if outcome.ClickPosition != nil {
		entry.Telemetry.ClickPosition = outcome.ClickPosition
	}
	if len(outcome.CursorTrail) > 0 {
		entry.Telemetry.CursorTrail = extractCursorPoints(outcome.CursorTrail)
	}
	if len(outcome.HighlightRegions) > 0 {
		entry.Telemetry.HighlightRegions = outcome.HighlightRegions
	}
	if len(outcome.MaskRegions) > 0 {
		entry.Telemetry.MaskRegions = outcome.MaskRegions
	}
	if outcome.ZoomFactor != 0 {
		zoom := outcome.ZoomFactor
		entry.Telemetry.ZoomFactor = &zoom
	}
	if entry.Aggregates == nil {
		entry.Aggregates = &bastimeline.TimelineEntryAggregates{}
	}
	if outcome.Success {
		entry.Aggregates.Status = basbase.StepStatus_STEP_STATUS_COMPLETED
	} else {
		entry.Aggregates.Status = basbase.StepStatus_STEP_STATUS_FAILED
	}
	if consoleArtifact != nil {
		entry.Aggregates.ConsoleLogCount = int32(len(outcome.ConsoleLogs))
	}
	if networkArtifact != nil {
		entry.Aggregates.NetworkEventCount = int32(len(outcome.Network))
	}

	for _, a := range artifacts {
		if a == nil {
			continue
		}
		entry.Aggregates.Artifacts = append(entry.Aggregates.Artifacts, a)
	}

	timeline.mu.Lock()
	timeline.pb.Entries = append(timeline.pb.Entries, entry)
	timeline.mu.Unlock()

	return r.writeProtoTimelineFile(plan.ExecutionID, timeline)
}

func artifactDataToProto(a *ArtifactData) *bastimeline.TimelineArtifact {
	if a == nil {
		return nil
	}

	contentType := strings.TrimSpace(a.ContentType)
	if contentType == "" {
		contentType = "application/json"
	}
	storageURL := strings.TrimSpace(a.StorageURL)
	if storageURL == "" {
		storageURL = "inline:" + a.ArtifactID
	}

	pb := &bastimeline.TimelineArtifact{
		Id:          a.ArtifactID,
		Type:        artifactTypeToProto(a.ArtifactType),
		StorageUrl:  storageURL,
		ContentType: contentType,
		Payload:     map[string]*commonv1.JsonValue{},
	}
	if strings.TrimSpace(a.Label) != "" {
		label := a.Label
		pb.Label = &label
	}
	if strings.TrimSpace(a.ThumbnailURL) != "" {
		thumb := a.ThumbnailURL
		pb.ThumbnailUrl = &thumb
	}
	if a.SizeBytes != nil {
		pb.SizeBytes = a.SizeBytes
	}
	if a.StepIndex != nil {
		stepIndex := int32(*a.StepIndex)
		pb.StepIndex = &stepIndex
	}
	if a.Payload != nil {
		for k, v := range a.Payload {
			pb.Payload[k] = anyToJsonValue(v)
		}
	}
	return pb
}

func artifactTypeToProto(kind string) basbase.ArtifactType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "timeline_frame":
		return basbase.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME
	case "console":
		return basbase.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG
	case "network":
		return basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT
	case "screenshot":
		return basbase.ArtifactType_ARTIFACT_TYPE_SCREENSHOT
	case "dom_snapshot":
		return basbase.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT
	case "trace", "video", "har", "trace_meta", "video_meta", "har_meta":
		return basbase.ArtifactType_ARTIFACT_TYPE_TRACE
	default:
		return basbase.ArtifactType_ARTIFACT_TYPE_CUSTOM
	}
}

// persistExternalFile stores metadata for engine-provided files (trace/video/HAR).
func (r *FileWriter) persistExternalFile(result *ExecutionResultData, stepID string, stepIndex int, artifactType, filePath string) *uuid.UUID {
	info, err := os.Stat(filePath)
	if err != nil {
		if r.log != nil {
			r.log.WithError(err).WithField("path", filePath).Debug("external file not readable")
		}
		return nil
	}

	payload := map[string]any{
		"path":       filePath,
		"size_bytes": info.Size(),
	}

	const maxEmbeddedExternalBytes = 5 * 1024 * 1024
	if info.Size() > 0 && info.Size() <= maxEmbeddedExternalBytes {
		if data, readErr := os.ReadFile(filePath); readErr == nil {
			payload["base64"] = base64.StdEncoding.EncodeToString(data)
			payload["inline"] = true
		}
	}

	id := uuid.New()
	artifact := ArtifactData{
		ArtifactID:   id.String(),
		StepID:       stepID,
		StepIndex:    &stepIndex,
		ArtifactType: artifactType,
		Label:        artifactType,
		Payload:      payload,
	}

	result.mu.Lock()
	result.Artifacts = append(result.Artifacts, artifact)
	result.mu.Unlock()

	return &id
}

// RecordTelemetry persists real-time telemetry events (heartbeats, console logs, etc.).
// Respects CollectTelemetry setting in artifact config.
func (r *FileWriter) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	if r == nil {
		return nil
	}

	// Check if telemetry collection is enabled
	cfg := r.GetArtifactConfig()
	if !cfg.CollectTelemetry {
		return nil // Skip telemetry recording when disabled
	}

	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)
	telemetryData := TelemetryData{
		StepIndex: telemetry.StepIndex,
		Data:      telemetry,
		Timestamp: time.Now().UTC(),
	}

	result.mu.Lock()
	result.Telemetry = append(result.Telemetry, telemetryData)
	result.mu.Unlock()

	if timeline != nil {
		if logEntry := telemetryToTimelineLog(telemetryData); logEntry != nil {
			timeline.mu.Lock()
			timeline.pb.Logs = append(timeline.pb.Logs, logEntry)
			timeline.mu.Unlock()
			_ = r.writeProtoTimelineFile(plan.ExecutionID, timeline)
		}
	}

	return r.writeResultFile(plan.ExecutionID, result, timeline)
}

func telemetryToTimelineLog(t TelemetryData) *bastimeline.TimelineLog {
	message := strings.TrimSpace(t.Data.Note)
	level := basbase.LogLevel_LOG_LEVEL_INFO

	switch strings.ToLower(strings.TrimSpace(string(t.Data.Kind))) {
	case "console":
		if len(t.Data.Console) > 0 && message == "" {
			first := t.Data.Console[0]
			message = strings.TrimSpace(first.Text)
			switch strings.ToLower(strings.TrimSpace(first.Type)) {
			case "warn", "warning":
				level = basbase.LogLevel_LOG_LEVEL_WARN
			case "error":
				level = basbase.LogLevel_LOG_LEVEL_ERROR
			case "debug":
				level = basbase.LogLevel_LOG_LEVEL_DEBUG
			default:
				level = basbase.LogLevel_LOG_LEVEL_INFO
			}
		}
	case "network":
		level = basbase.LogLevel_LOG_LEVEL_DEBUG
	case "retry":
		level = basbase.LogLevel_LOG_LEVEL_WARN
	case "heartbeat", "progress":
		level = basbase.LogLevel_LOG_LEVEL_INFO
	}

	if message == "" {
		return nil
	}

	return &bastimeline.TimelineLog{
		Id:        fmt.Sprintf("telemetry-%d-%d", t.StepIndex, t.Timestamp.UnixNano()),
		Level:     level,
		Message:   message,
		Timestamp: timestamppb.New(t.Timestamp),
	}
}

func stepOutcomeToTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *bastimeline.TimelineEntry {
	entryID := timelineEntryID(executionID, outcome.StepIndex, outcome.Attempt)
	stepIndex := int32(outcome.StepIndex)
	success := outcome.Success
	ctx := &basbase.EventContext{
		Success: &success,
	}
	if outcome.Failure != nil && strings.TrimSpace(outcome.Failure.Message) != "" {
		msg := outcome.Failure.Message
		ctx.Error = &msg
		if code := strings.TrimSpace(outcome.Failure.Code); code != "" {
			ctx.ErrorCode = &code
		}
	}

	entry := &bastimeline.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(outcome.StepIndex),
		StepIndex:   &stepIndex,
		Action: &basactions.ActionDefinition{
			Type: enums.StringToActionType(outcome.StepType),
		},
		Context: ctx,
	}

	if strings.TrimSpace(outcome.NodeID) != "" {
		nodeID := outcome.NodeID
		entry.NodeId = &nodeID
	}
	if !outcome.StartedAt.IsZero() {
		entry.Timestamp = timestamppb.New(outcome.StartedAt)
	}
	if outcome.DurationMs > 0 {
		dur := int32(outcome.DurationMs)
		entry.DurationMs = &dur
	}
	if entry.Telemetry == nil {
		entry.Telemetry = &basdomain.ActionTelemetry{}
	}
	entry.Telemetry.Url = outcome.FinalURL

	return entry
}

func anyToJsonValue(v any) *commonv1.JsonValue {
	if v == nil {
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{}}
	}
	switch val := v.(type) {
	case *commonv1.JsonValue:
		if val == nil {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{}}
		}
		return val
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for k, nested := range val {
			obj[k] = anyToJsonValue(nested)
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: &commonv1.JsonObject{Fields: obj}}}
	case map[string]string:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for k, nested := range val {
			obj[k] = anyToJsonValue(nested)
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: &commonv1.JsonObject{Fields: obj}}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, nested := range val {
			items = append(items, anyToJsonValue(nested))
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: &commonv1.JsonList{Values: items}}}
	default:
		// Fall back to string representation.
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

func timelineEntryID(executionID uuid.UUID, stepIndex int, attempt int) string {
	return fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), stepIndex, attempt)
}

func extractCursorPoints(trail []contracts.CursorPosition) []*basbase.Point {
	if len(trail) == 0 {
		return nil
	}
	points := make([]*basbase.Point, 0, len(trail))
	for _, entry := range trail {
		if entry.Point != nil {
			points = append(points, entry.Point)
		}
	}
	return points
}

// MarkCrash records a crash event and updates the execution index.
func (r *FileWriter) MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error {
	if r == nil {
		return nil
	}

	// Create a minimal plan for result lookup
	plan := contracts.ExecutionPlan{ExecutionID: executionID}
	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)

	now := time.Now().UTC()
	crashStep := StepResultData{
		StepID:      uuid.New().String(),
		StepIndex:   -1,
		NodeID:      "crash",
		StepType:    "crash",
		Status:      "failed",
		StartedAt:   now,
		CompletedAt: &now,
		Error:       failure.Message,
		Metadata: map[string]any{
			"failure": failure,
			"partial": true,
		},
	}

	result.mu.Lock()
	result.Steps = append(result.Steps, crashStep)
	result.Summary.FailedSteps++
	result.mu.Unlock()

	if err := r.writeResultFile(executionID, result, timeline); err != nil {
		if r.log != nil {
			r.log.WithError(err).Warn("Failed to write crash to result file")
		}
	}
	if timeline != nil {
		timeline.mu.Lock()
		timeline.pb.Logs = append(timeline.pb.Logs, &bastimeline.TimelineLog{
			Id:        fmt.Sprintf("crash-%d", time.Now().UTC().UnixNano()),
			Level:     basbase.LogLevel_LOG_LEVEL_ERROR,
			Message:   failure.Message,
			Timestamp: timestamppb.New(time.Now().UTC()),
		})
		timeline.mu.Unlock()
		_ = r.writeProtoTimelineFile(executionID, timeline)
	}

	// Update database index to mark as failed
	if r.repo != nil {
		now := time.Now().UTC()
		msg := strings.TrimSpace(failure.Message)
		var errMsg *string
		if msg != "" {
			errMsg = &msg
		}
		if err := r.repo.UpdateExecutionStatus(ctx, executionID, database.ExecutionStatusFailed, errMsg, &now, now); err != nil {
			return fmt.Errorf("update execution index: %w", err)
		}
	}

	return nil
}

// UpdateCheckpoint persists the current execution progress to the database index.
func (r *FileWriter) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	if r == nil || r.repo == nil {
		return nil
	}

	execution, err := r.repo.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("get execution: %w", err)
	}

	// Calculate progress as percentage (0-100)
	progress := 0
	if totalSteps > 0 && stepIndex >= 0 {
		progress = ((stepIndex + 1) * 100) / totalSteps
		if progress > 100 {
			progress = 100
		}
	}

	// The ExecutionIndex doesn't have Progress/CurrentStep fields anymore
	// We store progress in the result file instead
	plan := contracts.ExecutionPlan{ExecutionID: executionID, WorkflowID: execution.WorkflowID}
	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)

	result.mu.Lock()
	result.Summary.CompletedSteps = stepIndex + 1
	result.mu.Unlock()

	if timeline != nil {
		timeline.mu.Lock()
		timeline.pb.Progress = int32(progress)
		timeline.mu.Unlock()
		_ = r.writeProtoTimelineFile(executionID, timeline)
	}

	return r.writeResultFile(executionID, result, timeline)
}

// updateExecutionIndex updates the database index with the result path.
func (r *FileWriter) updateExecutionIndex(ctx context.Context, executionID uuid.UUID, resultPath string) error {
	if r.repo == nil {
		return nil
	}
	updatedAt := time.Now().UTC()
	if err := r.repo.UpdateExecutionResultPath(ctx, executionID, resultPath, updatedAt); err != nil {
		// Execution may not exist yet - that's OK, it will be created by the workflow service
		if errors.Is(err, database.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func (r *FileWriter) persistScreenshot(ctx context.Context, plan contracts.ExecutionPlan, executionID uuid.UUID, outcome contracts.StepOutcome) (*storage.ScreenshotInfo, error) {
	if r.storage == nil {
		return nil, nil
	}
	if outcome.Screenshot == nil || len(outcome.Screenshot.Data) == 0 {
		return nil, nil
	}
	contentType := outcome.Screenshot.MediaType
	if contentType == "" {
		contentType = "image/png"
	}
	stepName := buildScreenshotBaseName(plan, outcome)
	return r.storage.StoreScreenshot(ctx, executionID, stepName, outcome.Screenshot.Data, contentType)
}

type telemetryArtifactRef struct {
	ID          uuid.UUID
	URL         string
	Path        string
	ContentType string
	SizeBytes   int64
	ObjectName  string
}

func (r *FileWriter) persistTelemetryArtifact(ctx context.Context, executionID uuid.UUID, folder string, baseName string, ext string, contentType string, data []byte) (*telemetryArtifactRef, error) {
	if r.storage == nil {
		return nil, nil
	}
	if len(data) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(ext) == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	objectName := fmt.Sprintf("%s/artifacts/%s/%s%s", executionID.String(), folder, baseName, ext)
	info, err := r.storage.StoreArtifact(ctx, objectName, data, contentType)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &telemetryArtifactRef{
		ID:          uuid.New(),
		URL:         info.URL,
		Path:        info.Path,
		ContentType: info.ContentType,
		SizeBytes:   info.SizeBytes,
		ObjectName:  info.ObjectName,
	}, nil
}

func statusFromOutcome(outcome contracts.StepOutcome) string {
	if outcome.Success {
		return "completed"
	}
	if outcome.Failure != nil && outcome.Failure.Kind == contracts.FailureKindCancelled {
		return "failed"
	}
	return "failed"
}

func deriveStepLabel(outcome contracts.StepOutcome) string {
	if strings.TrimSpace(outcome.NodeID) != "" {
		return outcome.NodeID
	}
	if outcome.StepIndex >= 0 {
		return fmt.Sprintf("step-%d", outcome.StepIndex)
	}
	return "step"
}

func buildScreenshotBaseName(plan contracts.ExecutionPlan, outcome contracts.StepOutcome) string {
	stepNumber := outcome.StepIndex + 1
	if stepNumber < 0 {
		stepNumber = 0
	}
	stepToken := fmt.Sprintf("%05d", stepNumber)

	workflowName := workflowNameForArtifacts(plan)
	if workflowName == "" {
		workflowName = plan.WorkflowID.String()
	}
	workflowToken := slugifyFilenamePart(workflowName, "-")
	if workflowToken == "" {
		workflowToken = "workflow"
	}

	nodeToken := slugifyFilenamePart(outcome.NodeID, "-")
	if nodeToken == "" {
		nodeToken = "node"
	}

	actionToken := normalizeActionType(outcome.StepType)

	name := fmt.Sprintf("%s--%s--%s--%s", stepToken, workflowToken, nodeToken, actionToken)
	if outcome.Attempt > 1 {
		name = fmt.Sprintf("%s--attempt-%02d", name, outcome.Attempt)
	}
	return name
}

func workflowNameForArtifacts(plan contracts.ExecutionPlan) string {
	if plan.Metadata == nil {
		return ""
	}
	if value, ok := plan.Metadata["workflowName"]; ok {
		if name, ok := value.(string); ok {
			return strings.TrimSpace(name)
		}
	}
	if value, ok := plan.Metadata["workflow_name"]; ok {
		if name, ok := value.(string); ok {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

func slugifyFilenamePart(input string, delimiter string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	lastWasDelimiter := false

	for _, r := range trimmed {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastWasDelimiter = false
			continue
		}
		if !lastWasDelimiter {
			b.WriteString(delimiter)
			lastWasDelimiter = true
		}
	}

	out := strings.Trim(b.String(), delimiter)
	return out
}

func normalizeActionType(actionType string) string {
	trimmed := strings.TrimSpace(actionType)
	if trimmed == "" {
		return "ACTION_TYPE_UNKNOWN"
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	lastWasUnderscore := false

	for _, r := range trimmed {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastWasUnderscore = false
			continue
		}
		if !lastWasUnderscore {
			b.WriteByte('_')
			lastWasUnderscore = true
		}
	}

	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "UNKNOWN"
	}
	if !strings.HasPrefix(out, "ACTION_TYPE_") {
		out = "ACTION_TYPE_" + out
	}
	return out
}

// sanitizeOutcome applies default size limits to outcome fields.
// For configurable limits, use sanitizeOutcomeWithConfig instead.
func sanitizeOutcome(out contracts.StepOutcome) contracts.StepOutcome {
	return sanitizeOutcomeWithLimits(out, contracts.ScreenshotMaxBytes, contracts.DOMSnapshotMaxBytes, contracts.ConsoleEntryMaxBytes, contracts.NetworkPayloadPreviewMaxBytes)
}

// sanitizeOutcomeWithConfig applies configurable size limits from ArtifactCollectionSettings.
func (r *FileWriter) sanitizeOutcomeWithConfig(out contracts.StepOutcome, cfg config.ArtifactCollectionSettings) contracts.StepOutcome {
	maxScreenshot := cfg.MaxScreenshotBytes
	if maxScreenshot <= 0 {
		maxScreenshot = contracts.ScreenshotMaxBytes
	}
	maxDOM := cfg.MaxDOMSnapshotBytes
	if maxDOM <= 0 {
		maxDOM = contracts.DOMSnapshotMaxBytes
	}
	maxConsole := cfg.MaxConsoleEntryBytes
	if maxConsole <= 0 {
		maxConsole = contracts.ConsoleEntryMaxBytes
	}
	maxNetwork := cfg.MaxNetworkPreviewBytes
	if maxNetwork <= 0 {
		maxNetwork = contracts.NetworkPayloadPreviewMaxBytes
	}
	return sanitizeOutcomeWithLimits(out, maxScreenshot, maxDOM, maxConsole, maxNetwork)
}

// sanitizeOutcomeWithLimits applies specified size limits to outcome fields.
func sanitizeOutcomeWithLimits(out contracts.StepOutcome, maxScreenshot, maxDOM, maxConsole, maxNetwork int) contracts.StepOutcome {
	if out.Notes == nil {
		out.Notes = map[string]string{}
	}

	// Screenshot shaping: clamp size and ensure defaults.
	if out.Screenshot != nil {
		if len(out.Screenshot.Data) > maxScreenshot {
			out.Screenshot.Data = out.Screenshot.Data[:maxScreenshot]
			out.Notes["screenshot_truncated"] = fmt.Sprintf("%d_bytes", maxScreenshot)
		}
		if out.Screenshot.MediaType == "" {
			out.Screenshot.MediaType = "image/png"
		}
		if out.Screenshot.Width == 0 || out.Screenshot.Height == 0 {
			out.Screenshot.Width = contracts.DefaultScreenshotWidth
			out.Screenshot.Height = contracts.DefaultScreenshotHeight
		}
	}

	// DOM truncation
	if out.DOMSnapshot != nil && len(out.DOMSnapshot.HTML) > maxDOM {
		hash := hashString(out.DOMSnapshot.HTML)
		out.DOMSnapshot.HTML = out.DOMSnapshot.HTML[:maxDOM]
		out.DOMSnapshot.Truncated = true
		out.DOMSnapshot.Hash = hash
		out.Notes["dom_truncated_hash"] = hash
	}

	out.ConsoleLogs = sanitizeConsoleWithLimit(out.ConsoleLogs, maxConsole)
	out.Network = sanitizeNetworkWithLimit(out.Network, maxNetwork)

	return out
}

// sanitizeConsole applies default size limits to console log entries.
func sanitizeConsole(entries []contracts.ConsoleLogEntry) []contracts.ConsoleLogEntry {
	return sanitizeConsoleWithLimit(entries, contracts.ConsoleEntryMaxBytes)
}

// sanitizeConsoleWithLimit applies configurable size limits to console log entries.
func sanitizeConsoleWithLimit(entries []contracts.ConsoleLogEntry, maxEntryBytes int) []contracts.ConsoleLogEntry {
	if len(entries) == 0 {
		return entries
	}
	sanitized := make([]contracts.ConsoleLogEntry, 0, len(entries))
	for idx, entry := range entries {
		if len(entry.Text) > maxEntryBytes {
			hash := hashString(entry.Text)
			entry.Text = entry.Text[:maxEntryBytes] + "[truncated]"
			entry.Location = appendHash(entry.Location, hash)
		}
		entry.Timestamp = entry.Timestamp.UTC()
		sanitized = append(sanitized, entry)
		if idx >= maxEntryBytes {
			break
		}
	}
	return sanitized
}

// sanitizeNetwork applies default size limits to network events.
func sanitizeNetwork(events []contracts.NetworkEvent) []contracts.NetworkEvent {
	return sanitizeNetworkWithLimit(events, contracts.NetworkPayloadPreviewMaxBytes)
}

// sanitizeNetworkWithLimit applies configurable size limits to network events.
func sanitizeNetworkWithLimit(events []contracts.NetworkEvent, maxPreviewBytes int) []contracts.NetworkEvent {
	if len(events) == 0 {
		return events
	}
	sanitized := make([]contracts.NetworkEvent, 0, len(events))
	for idx, ev := range events {
		if len(ev.RequestBodyPreview) > maxPreviewBytes {
			ev.Truncated = true
			ev.RequestBodyPreview = ev.RequestBodyPreview[:maxPreviewBytes]
		}
		if len(ev.ResponseBodyPreview) > maxPreviewBytes {
			ev.Truncated = true
			ev.ResponseBodyPreview = ev.ResponseBodyPreview[:maxPreviewBytes]
		}
		ev.Timestamp = ev.Timestamp.UTC()
		sanitized = append(sanitized, ev)
		if idx >= maxPreviewBytes {
			break
		}
	}
	return sanitized
}

func hashString(val string) string {
	sum := sha256.Sum256([]byte(val))
	return hex.EncodeToString(sum[:])
}

func appendHash(location, hash string) string {
	if strings.TrimSpace(hash) == "" {
		return location
	}
	if strings.TrimSpace(location) == "" {
		return "hash:" + hash
	}
	return location + " hash:" + hash
}

func toStringIDs(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func buildTimelinePayload(outcome contracts.StepOutcome, screenshotURL string, screenshotID *uuid.UUID, artifactIDs []uuid.UUID) map[string]any {
	payload := map[string]any{
		"stepIndex":     outcome.StepIndex,
		"nodeId":        outcome.NodeID,
		"stepType":      outcome.StepType,
		"screenshotUrl": screenshotURL,
		"success":       outcome.Success,
		"attempt":       outcome.Attempt,
		"durationMs":    outcome.DurationMs,
	}
	if len(outcome.CursorTrail) > 0 {
		payload["cursorTrail"] = outcome.CursorTrail
	}
	if outcome.Failure != nil || !outcome.Success {
		payload["partial"] = true
	}
	if outcome.StartedAt.Unix() > 0 {
		payload["startedAt"] = outcome.StartedAt
	}
	if outcome.CompletedAt != nil {
		payload["completedAt"] = *outcome.CompletedAt
	}
	if outcome.FinalURL != "" {
		payload["finalUrl"] = outcome.FinalURL
	}
	if outcome.ElementBoundingBox != nil {
		payload["elementBoundingBox"] = outcome.ElementBoundingBox
	}
	if outcome.ClickPosition != nil {
		payload["clickPosition"] = outcome.ClickPosition
	}
	if outcome.FocusedElement != nil {
		payload["focusedElement"] = outcome.FocusedElement
	}
	if len(outcome.HighlightRegions) > 0 {
		payload["highlightRegions"] = outcome.HighlightRegions
	}
	if len(outcome.MaskRegions) > 0 {
		payload["maskRegions"] = outcome.MaskRegions
	}
	if outcome.ZoomFactor != 0 {
		payload["zoomFactor"] = outcome.ZoomFactor
	}
	if outcome.ExtractedData != nil {
		payload["extractedDataPreview"] = outcome.ExtractedData
	}
	if outcome.Assertion != nil {
		payload["assertion"] = outcome.Assertion
	}
	if len(outcome.ConsoleLogs) > 0 {
		payload["consoleLogCount"] = len(outcome.ConsoleLogs)
	}
	if len(outcome.Network) > 0 {
		payload["networkEventCount"] = len(outcome.Network)
	}
	if outcome.Failure != nil && strings.TrimSpace(outcome.Failure.Message) != "" {
		payload["error"] = strings.TrimSpace(outcome.Failure.Message)
	}
	if screenshotID != nil {
		payload["screenshotArtifactId"] = screenshotID.String()
	}
	if len(artifactIDs) > 0 {
		payload["artifactIds"] = toStringIDs(artifactIDs)
	}
	return payload
}

// Compile-time interface enforcement
var _ ExecutionWriter = (*FileWriter)(nil)
