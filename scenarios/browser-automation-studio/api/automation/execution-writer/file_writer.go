package executionwriter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
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
}

// ExecutionResultData accumulates execution results to be written to disk.
// This is the file format stored at ExecutionIndex.ResultPath.
type ExecutionResultData struct {
	ExecutionID   string                   `json:"execution_id"`
	WorkflowID    string                   `json:"workflow_id"`
	Steps         []StepResultData         `json:"steps"`
	Artifacts     []ArtifactData           `json:"artifacts"`
	Telemetry     []TelemetryData          `json:"telemetry"`
	TimelineFrame []TimelineFrameData      `json:"timeline_frames"`
	Summary       ExecutionSummary         `json:"summary"`
	mu            sync.Mutex               `json:"-"`
}

type executionTimelineData struct {
	mu sync.Mutex
	pb *bastimeline.ExecutionTimeline
}

// StepResultData captures individual step execution results.
type StepResultData struct {
	StepID      string                 `json:"step_id"`
	StepIndex   int                    `json:"step_index"`
	NodeID      string                 `json:"node_id"`
	StepType    string                 `json:"step_type"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	DurationMs  int                    `json:"duration_ms"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
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
	StepIndex int                      `json:"step_index"`
	Data      contracts.StepTelemetry  `json:"data"`
	Timestamp time.Time                `json:"timestamp"`
}

// TimelineFrameData captures timeline frame data for replay.
type TimelineFrameData struct {
	StepIndex            int            `json:"step_index"`
	NodeID               string         `json:"node_id"`
	StepType             string         `json:"step_type"`
	ScreenshotURL        string         `json:"screenshot_url,omitempty"`
	ScreenshotArtifactID string         `json:"screenshot_artifact_id,omitempty"`
	DOMSnapshotArtifactID string        `json:"dom_snapshot_artifact_id,omitempty"`
	Success              bool           `json:"success"`
	Attempt              int            `json:"attempt"`
	DurationMs           int            `json:"duration_ms"`
	Payload              map[string]any `json:"payload,omitempty"`
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
func NewFileWriter(repo ExecutionIndexRepository, storage storage.StorageInterface, log *logrus.Logger, dataDir string) *FileWriter {
	if dataDir == "" {
		dataDir = "/tmp/bas-executions"
	}
	return &FileWriter{
		repo:    repo,
		storage: storage,
		log:     log,
		dataDir: dataDir,
	}
}

const (
	resultFileName        = "result.json"
	protoTimelineFileName = "timeline.proto.json"
)

// getOrCreateResult gets or creates the result data for an execution.
func (r *FileWriter) getOrCreateResult(plan contracts.ExecutionPlan) *ExecutionResultData {
	key := plan.ExecutionID.String()
	if val, ok := r.results.Load(key); ok {
		return val.(*ExecutionResultData)
	}

	result := &ExecutionResultData{
		ExecutionID: plan.ExecutionID.String(),
		WorkflowID:  plan.WorkflowID.String(),
		Steps:       make([]StepResultData, 0),
		Artifacts:   make([]ArtifactData, 0),
		Telemetry:   make([]TelemetryData, 0),
		TimelineFrame: make([]TimelineFrameData, 0),
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

// writeResultFile persists the execution result data to disk.
func (r *FileWriter) writeResultFile(executionID uuid.UUID, result *ExecutionResultData) error {
	result.mu.Lock()
	defer result.mu.Unlock()

	result.Summary.LastUpdated = time.Now().UTC()

	filePath := r.resultFilePath(executionID)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create result directory: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write result file: %w", err)
	}

	return nil
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
func (r *FileWriter) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error) {
	if r == nil {
		return RecordResult{}, nil
	}

	outcome = sanitizeOutcome(outcome)
	result := r.getOrCreateResult(plan)
	timeline := r.getOrCreateTimeline(plan)

	stepID := uuid.New()
	artifactIDs := make([]uuid.UUID, 0, 8)
	protoArtifacts := make([]*bastimeline.TimelineArtifact, 0, 8)

	// Build step result
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

	// Store core outcome payload as artifact
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

	// Console logs
	if len(outcome.ConsoleLogs) > 0 {
		id := uuid.New()
		artifact := ArtifactData{
			ArtifactID:   id.String(),
			StepID:       stepID.String(),
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "console",
			Label:        deriveStepLabel(outcome),
			Payload:      map[string]any{"entries": outcome.ConsoleLogs},
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
		artifactIDs = append(artifactIDs, id)
		protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
	}

	// Network events
	if len(outcome.Network) > 0 {
		id := uuid.New()
		artifact := ArtifactData{
			ArtifactID:   id.String(),
			StepID:       stepID.String(),
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "network",
			Label:        deriveStepLabel(outcome),
			Payload:      map[string]any{"events": outcome.Network},
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
		artifactIDs = append(artifactIDs, id)
		protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
	}

	// Assertion result
	if outcome.Assertion != nil {
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

	// Extracted data
	if outcome.ExtractedData != nil && len(outcome.ExtractedData) > 0 {
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

	// Engine-provided metadata (video/trace/HAR paths)
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
	var timelineScreenshotID *uuid.UUID

	// Persist screenshot if available
	if outcome.Screenshot != nil && len(outcome.Screenshot.Data) > 0 {
		screenshotInfo, err := r.persistScreenshot(ctx, plan.ExecutionID, outcome)
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
			result.mu.Lock()
			result.Artifacts = append(result.Artifacts, artifact)
			result.mu.Unlock()
			artifactIDs = append(artifactIDs, id)
			protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
			timelineScreenshotID = &id
			timelineScreenshotURL = screenshotInfo.URL
		} else {
			// Fallback: embed base64 if storage is unavailable
			id := uuid.New()
			artifact := ArtifactData{
				ArtifactID:   id.String(),
				StepID:       stepID.String(),
				StepIndex:    &outcome.StepIndex,
				ArtifactType: "screenshot_inline",
				ContentType:  outcome.Screenshot.MediaType,
				Payload: map[string]any{
					"base64": base64.StdEncoding.EncodeToString(outcome.Screenshot.Data),
					"hash":   outcome.Screenshot.Hash,
				},
			}
			result.mu.Lock()
			result.Artifacts = append(result.Artifacts, artifact)
			result.mu.Unlock()
			artifactIDs = append(artifactIDs, id)
			protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
			timelineScreenshotID = &id
			timelineScreenshotURL = fmt.Sprintf("inline:%s", deriveStepLabel(outcome))
		}
	}

	var domSnapshotArtifactID *uuid.UUID
	var domSnapshotPreview string

	// DOM snapshot
	if outcome.DOMSnapshot != nil && outcome.DOMSnapshot.HTML != "" {
		html := outcome.DOMSnapshot.HTML
		truncated := false
		if len(html) > contracts.DOMSnapshotMaxBytes {
			html = html[:contracts.DOMSnapshotMaxBytes]
			truncated = true
			outcome.DOMSnapshot.Truncated = true
		}
		id := uuid.New()
		payload := map[string]any{"html": html}
		if truncated {
			payload["truncated"] = true
		}
		artifact := ArtifactData{
			ArtifactID:   id.String(),
			StepID:       stepID.String(),
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "dom_snapshot",
			Label:        deriveStepLabel(outcome),
			Payload:      payload,
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
		artifactIDs = append(artifactIDs, id)
		protoArtifacts = append(protoArtifacts, artifactDataToProto(&artifact))
		domSnapshotArtifactID = &id
		domSnapshotPreview = truncateRunes(html, 256)
	}

	// Build timeline frame
	timelinePayload := buildTimelinePayload(outcome, timelineScreenshotURL, timelineScreenshotID, domSnapshotArtifactID, domSnapshotPreview, artifactIDs)
	timelineFrame := TimelineFrameData{
		StepIndex:     outcome.StepIndex,
		NodeID:        outcome.NodeID,
		StepType:      outcome.StepType,
		ScreenshotURL: timelineScreenshotURL,
		Success:       outcome.Success,
		Attempt:       outcome.Attempt,
		DurationMs:    outcome.DurationMs,
		Payload:       timelinePayload,
	}
	if timelineScreenshotID != nil {
		timelineFrame.ScreenshotArtifactID = timelineScreenshotID.String()
	}
	if domSnapshotArtifactID != nil {
		timelineFrame.DOMSnapshotArtifactID = domSnapshotArtifactID.String()
	}
	result.mu.Lock()
	result.TimelineFrame = append(result.TimelineFrame, timelineFrame)
	result.mu.Unlock()

	// Write result to file
	if err := r.writeResultFile(plan.ExecutionID, result); err != nil {
		if r.log != nil {
			r.log.WithError(err).Warn("Failed to write execution result file")
		}
	}
	if err := r.appendProtoTimelineEntry(plan, outcome, protoArtifacts, timelineScreenshotID, timelineScreenshotURL, domSnapshotArtifactID, domSnapshotPreview, timeline); err != nil {
		if r.log != nil {
			r.log.WithError(err).Warn("Failed to write proto timeline file")
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

func (r *FileWriter) appendProtoTimelineEntry(
	plan contracts.ExecutionPlan,
	outcome contracts.StepOutcome,
	artifacts []*bastimeline.TimelineArtifact,
	screenshotArtifactID *uuid.UUID,
	screenshotURL string,
	domSnapshotArtifactID *uuid.UUID,
	domSnapshotPreview string,
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
	}
	if strings.TrimSpace(domSnapshotPreview) != "" {
		if entry.Telemetry == nil {
			entry.Telemetry = &basdomain.ActionTelemetry{}
		}
		preview := domSnapshotPreview
		entry.Telemetry.DomSnapshotPreview = &preview
	}

	if entry.Aggregates == nil {
		entry.Aggregates = &bastimeline.TimelineEntryAggregates{}
	}
	if outcome.Success {
		entry.Aggregates.Status = basbase.StepStatus_STEP_STATUS_COMPLETED
	} else {
		entry.Aggregates.Status = basbase.StepStatus_STEP_STATUS_FAILED
	}

	for _, a := range artifacts {
		if a == nil {
			continue
		}
		entry.Aggregates.Artifacts = append(entry.Aggregates.Artifacts, a)
	}
	if domSnapshotArtifactID != nil {
		for _, a := range entry.Aggregates.Artifacts {
			if a != nil && a.Id == domSnapshotArtifactID.String() {
				entry.Aggregates.DomSnapshot = a
				break
			}
		}
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
	case "screenshot", "screenshot_inline":
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

// RecordTelemetry persists telemetry data to the result file.
func (r *FileWriter) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	if r == nil {
		return nil
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

	return r.writeResultFile(plan.ExecutionID, result)
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
	entryID := fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt)
	stepIndex := int32(outcome.StepIndex)

	success := outcome.Success
	ctx := &basbase.EventContext{
		Success: &success,
	}
	if outcome.Failure != nil && strings.TrimSpace(outcome.Failure.Message) != "" {
		msg := outcome.Failure.Message
		ctx.Error = &msg
	}

	entry := &bastimeline.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(outcome.StepIndex),
		StepIndex:   &stepIndex,
		Action: &basactions.ActionDefinition{
			Type: stepTypeToActionType(outcome.StepType),
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

	// Minimal telemetry - enriched later when persisted screenshot is available.
	entry.Telemetry = &basdomain.ActionTelemetry{
		Url: outcome.FinalURL,
	}

	return entry
}

func stepTypeToActionType(stepType string) basactions.ActionType {
	switch strings.ToLower(strings.TrimSpace(stepType)) {
	case "navigate":
		return basactions.ActionType_ACTION_TYPE_NAVIGATE
	case "click":
		return basactions.ActionType_ACTION_TYPE_CLICK
	case "input", "type":
		return basactions.ActionType_ACTION_TYPE_INPUT
	case "wait":
		return basactions.ActionType_ACTION_TYPE_WAIT
	case "assert":
		return basactions.ActionType_ACTION_TYPE_ASSERT
	case "scroll":
		return basactions.ActionType_ACTION_TYPE_SCROLL
	case "hover":
		return basactions.ActionType_ACTION_TYPE_HOVER
	case "keyboard":
		return basactions.ActionType_ACTION_TYPE_KEYBOARD
	case "screenshot":
		return basactions.ActionType_ACTION_TYPE_SCREENSHOT
	case "select":
		return basactions.ActionType_ACTION_TYPE_SELECT
	case "evaluate":
		return basactions.ActionType_ACTION_TYPE_EVALUATE
	case "focus":
		return basactions.ActionType_ACTION_TYPE_FOCUS
	case "blur":
		return basactions.ActionType_ACTION_TYPE_BLUR
	default:
		return basactions.ActionType_ACTION_TYPE_UNSPECIFIED
	}
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

	if err := r.writeResultFile(executionID, result); err != nil {
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
		execution, err := r.repo.GetExecution(ctx, executionID)
		if err == nil && execution != nil {
			execution.Status = database.ExecutionStatusFailed
			execution.ErrorMessage = failure.Message
			now := time.Now()
			execution.CompletedAt = &now
			if err := r.repo.UpdateExecution(ctx, execution); err != nil {
				return fmt.Errorf("update execution index: %w", err)
			}
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

	return r.writeResultFile(executionID, result)
}

// updateExecutionIndex updates the database index with the result path.
func (r *FileWriter) updateExecutionIndex(ctx context.Context, executionID uuid.UUID, resultPath string) error {
	execution, err := r.repo.GetExecution(ctx, executionID)
	if err != nil {
		// Execution may not exist yet - that's OK, it will be created by the workflow service
		return nil
	}

	execution.ResultPath = resultPath
	return r.repo.UpdateExecution(ctx, execution)
}

func (r *FileWriter) persistScreenshot(ctx context.Context, executionID uuid.UUID, outcome contracts.StepOutcome) (*storage.ScreenshotInfo, error) {
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
	return r.storage.StoreScreenshot(ctx, executionID, outcome.NodeID, outcome.Screenshot.Data, contentType)
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

func sanitizeOutcome(out contracts.StepOutcome) contracts.StepOutcome {
	if out.Notes == nil {
		out.Notes = map[string]string{}
	}

	// Screenshot shaping: clamp size and ensure defaults.
	if out.Screenshot != nil {
		if len(out.Screenshot.Data) > contracts.ScreenshotMaxBytes {
			out.Screenshot.Data = out.Screenshot.Data[:contracts.ScreenshotMaxBytes]
			out.Notes["screenshot_truncated"] = fmt.Sprintf("%d_bytes", contracts.ScreenshotMaxBytes)
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
	if out.DOMSnapshot != nil && len(out.DOMSnapshot.HTML) > contracts.DOMSnapshotMaxBytes {
		hash := hashString(out.DOMSnapshot.HTML)
		out.DOMSnapshot.HTML = out.DOMSnapshot.HTML[:contracts.DOMSnapshotMaxBytes]
		out.DOMSnapshot.Truncated = true
		out.DOMSnapshot.Hash = hash
		out.Notes["dom_truncated_hash"] = hash
	}

	out.ConsoleLogs = sanitizeConsole(out.ConsoleLogs)
	out.Network = sanitizeNetwork(out.Network)

	return out
}

func sanitizeConsole(entries []contracts.ConsoleLogEntry) []contracts.ConsoleLogEntry {
	if len(entries) == 0 {
		return entries
	}
	sanitized := make([]contracts.ConsoleLogEntry, 0, len(entries))
	for idx, entry := range entries {
		if len(entry.Text) > contracts.ConsoleEntryMaxBytes {
			hash := hashString(entry.Text)
			entry.Text = entry.Text[:contracts.ConsoleEntryMaxBytes] + "[truncated]"
			entry.Location = appendHash(entry.Location, hash)
		}
		entry.Timestamp = entry.Timestamp.UTC()
		sanitized = append(sanitized, entry)
		if idx >= contracts.ConsoleEntryMaxBytes {
			break
		}
	}
	return sanitized
}

func sanitizeNetwork(events []contracts.NetworkEvent) []contracts.NetworkEvent {
	if len(events) == 0 {
		return events
	}
	sanitized := make([]contracts.NetworkEvent, 0, len(events))
	for idx, ev := range events {
		if len(ev.RequestBodyPreview) > contracts.NetworkPayloadPreviewMaxBytes {
			ev.Truncated = true
			ev.RequestBodyPreview = ev.RequestBodyPreview[:contracts.NetworkPayloadPreviewMaxBytes]
		}
		if len(ev.ResponseBodyPreview) > contracts.NetworkPayloadPreviewMaxBytes {
			ev.Truncated = true
			ev.ResponseBodyPreview = ev.ResponseBodyPreview[:contracts.NetworkPayloadPreviewMaxBytes]
		}
		ev.Timestamp = ev.Timestamp.UTC()
		sanitized = append(sanitized, ev)
		if idx >= contracts.NetworkPayloadPreviewMaxBytes {
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

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}

func buildTimelinePayload(outcome contracts.StepOutcome, screenshotURL string, screenshotID *uuid.UUID, domSnapshotID *uuid.UUID, domSnapshotPreview string, artifactIDs []uuid.UUID) map[string]any {
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
	if domSnapshotID != nil {
		payload["domSnapshotArtifactId"] = domSnapshotID.String()
	}
	if domSnapshotPreview != "" {
		payload["domSnapshotPreview"] = domSnapshotPreview
	}
	if len(artifactIDs) > 0 {
		payload["artifactIds"] = toStringIDs(artifactIDs)
	}
	return payload
}

// Compile-time interface enforcement
var _ ExecutionWriter = (*FileWriter)(nil)
