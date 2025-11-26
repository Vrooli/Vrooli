package recorder

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

// DBRecorder persists step outcomes and telemetry into the scenario database.
// It intentionally normalizes payloads into existing ExecutionStep/Artifact
// tables so UI/replay consumers can evolve without engine-specific fields.
type DBRecorder struct {
	repo    ExecutionRepository
	storage storage.StorageInterface
	log     *logrus.Logger

	// best-effort recovery cache to avoid re-checking the same execution ID repeatedly
	checkedExecutions sync.Map
}

// NewDBRecorder constructs a Recorder backed by the database + optional storage.
func NewDBRecorder(repo ExecutionRepository, storage storage.StorageInterface, log *logrus.Logger) *DBRecorder {
	return &DBRecorder{repo: repo, storage: storage, log: log}
}

// RecordStepOutcome stores the execution step and key artifacts. For now, this
// focuses on minimal parity: step row + outcome artifact + optional screenshot.
func (r *DBRecorder) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error) {
	if r == nil || r.repo == nil {
		return RecordResult{}, nil
	}

	outcome = sanitizeOutcome(outcome)

	step := &database.ExecutionStep{
		ID:          uuid.New(),
		ExecutionID: plan.ExecutionID,
		StepIndex:   outcome.StepIndex,
		NodeID:      outcome.NodeID,
		StepType:    outcome.StepType,
		Status:      statusFromOutcome(outcome),
		StartedAt:   outcome.StartedAt,
		CompletedAt: outcome.CompletedAt,
		DurationMs:  outcome.DurationMs,
		Metadata: database.JSONMap{
			"attempt":         outcome.Attempt,
			"payload_version": outcome.PayloadVersion,
		},
	}
	if outcome.Failure != nil {
		step.Metadata["failure"] = outcome.Failure
		if step.Error == "" {
			step.Error = outcome.Failure.Message
		}
	}
	if err := r.repo.CreateExecutionStep(ctx, step); err != nil {
		recovered, recErr := r.recoverMissingExecution(ctx, plan, err)
		if recErr != nil {
			return RecordResult{}, recErr
		}
		if recovered {
			// Retry once after recreating the execution row.
			if retryErr := r.repo.CreateExecutionStep(ctx, step); retryErr != nil {
				return RecordResult{}, retryErr
			}
		} else {
			return RecordResult{}, err
		}
	}

	artifactIDs := make([]uuid.UUID, 0, 3)

	// Store core outcome payload as JSON artifact to keep parity with existing exports.
	outcomeArtifact := &database.ExecutionArtifact{
		ID:           uuid.New(),
		ExecutionID:  plan.ExecutionID,
		StepID:       &step.ID,
		StepIndex:    &outcome.StepIndex,
		ArtifactType: "step_outcome",
		Payload:      database.JSONMap{"outcome": outcome},
	}
	if err := r.repo.CreateExecutionArtifact(ctx, outcomeArtifact); err == nil {
		artifactIDs = append(artifactIDs, outcomeArtifact.ID)
	}

	// Console logs
	if len(outcome.ConsoleLogs) > 0 {
		artifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  plan.ExecutionID,
			StepID:       &step.ID,
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "console",
			Label:        deriveStepLabel(outcome),
			Payload: database.JSONMap{
				"entries": outcome.ConsoleLogs,
			},
		}
		if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
			artifactIDs = append(artifactIDs, artifact.ID)
		}
	}

	// Network events
	if len(outcome.Network) > 0 {
		artifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  plan.ExecutionID,
			StepID:       &step.ID,
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "network",
			Label:        deriveStepLabel(outcome),
			Payload: database.JSONMap{
				"events": outcome.Network,
			},
		}
		if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
			artifactIDs = append(artifactIDs, artifact.ID)
		}
	}

	// Assertion result
	if outcome.Assertion != nil {
		artifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  plan.ExecutionID,
			StepID:       &step.ID,
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "assertion",
			Label:        deriveStepLabel(outcome),
			Payload: database.JSONMap{
				"assertion": outcome.Assertion,
			},
		}
		if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
			artifactIDs = append(artifactIDs, artifact.ID)
		}
	}

	// Extracted data
	if outcome.ExtractedData != nil {
		if len(outcome.ExtractedData) > 0 {
			artifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  plan.ExecutionID,
				StepID:       &step.ID,
				StepIndex:    &outcome.StepIndex,
				ArtifactType: "extracted_data",
				Label:        deriveStepLabel(outcome),
				Payload: database.JSONMap{
					"value": outcome.ExtractedData,
				},
			}
			if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
				artifactIDs = append(artifactIDs, artifact.ID)
			}
		}
	}

	// Engine-provided metadata (video/trace/HAR paths for desktop engines).
	if outcome.Notes != nil {
		if path := strings.TrimSpace(outcome.Notes["video_path"]); path != "" {
			if id := r.persistExternalFile(ctx, plan.ExecutionID, &step.ID, outcome.StepIndex, "video_meta", path); id != nil {
				artifactIDs = append(artifactIDs, *id)
			}
		}
		if path := strings.TrimSpace(outcome.Notes["trace_path"]); path != "" {
			if id := r.persistExternalFile(ctx, plan.ExecutionID, &step.ID, outcome.StepIndex, "trace_meta", path); id != nil {
				artifactIDs = append(artifactIDs, *id)
			}
		}
		if path := strings.TrimSpace(outcome.Notes["har_path"]); path != "" {
			if id := r.persistExternalFile(ctx, plan.ExecutionID, &step.ID, outcome.StepIndex, "har_meta", path); id != nil {
				artifactIDs = append(artifactIDs, *id)
			}
		}
	}

	var timelineScreenshot string
	var timelineScreenshotID *uuid.UUID

	// Persist screenshot if available.
	if outcome.Screenshot != nil && len(outcome.Screenshot.Data) > 0 {
		screenshotInfo, err := r.persistScreenshot(ctx, plan.ExecutionID, outcome)
		if err != nil && r.log != nil {
			r.log.WithError(err).Warn("Failed to persist screenshot artifact")
		} else if screenshotInfo != nil {
			artifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  plan.ExecutionID,
				StepID:       &step.ID,
				StepIndex:    &outcome.StepIndex,
				ArtifactType: "screenshot",
				StorageURL:   screenshotInfo.URL,
				ThumbnailURL: screenshotInfo.ThumbnailURL,
				SizeBytes:    &screenshotInfo.SizeBytes,
				Payload: database.JSONMap{
					"width":      screenshotInfo.Width,
					"height":     screenshotInfo.Height,
					"from_cache": outcome.Screenshot.FromCache,
					"hash":       outcome.Screenshot.Hash,
				},
			}
			if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
				artifactIDs = append(artifactIDs, artifact.ID)
				timelineScreenshotID = &artifact.ID
			}
			timelineScreenshot = screenshotInfo.URL
		} else {
			// Fallback: embed base64 if storage is unavailable.
			artifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  plan.ExecutionID,
				StepID:       &step.ID,
				StepIndex:    &outcome.StepIndex,
				ArtifactType: "screenshot_inline",
				ContentType:  outcome.Screenshot.MediaType,
				Payload: database.JSONMap{
					"base64": base64.StdEncoding.EncodeToString(outcome.Screenshot.Data),
					"hash":   outcome.Screenshot.Hash,
				},
			}
			if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
				artifactIDs = append(artifactIDs, artifact.ID)
				timelineScreenshotID = &artifact.ID
			}
			timelineScreenshot = fmt.Sprintf("inline:%s", deriveStepLabel(outcome))
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
		artifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  plan.ExecutionID,
			StepID:       &step.ID,
			StepIndex:    &outcome.StepIndex,
			ArtifactType: "dom_snapshot",
			Label:        deriveStepLabel(outcome),
			Payload: database.JSONMap{
				"html": html,
			},
		}
		if truncated {
			artifact.Payload["truncated"] = true
		}
		if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
			artifactIDs = append(artifactIDs, artifact.ID)
			domSnapshotArtifactID = &artifact.ID
			domSnapshotPreview = truncateRunes(html, 256)
		}
	}

	// Timeline frame for replay parity; include references to persisted artifacts.
	timelinePayload := buildTimelinePayload(outcome, timelineScreenshot, timelineScreenshotID, domSnapshotArtifactID, domSnapshotPreview, artifactIDs)
	artifact := &database.ExecutionArtifact{
		ID:           uuid.New(),
		ExecutionID:  plan.ExecutionID,
		StepID:       &step.ID,
		StepIndex:    &outcome.StepIndex,
		ArtifactType: "timeline_frame",
		Label:        deriveStepLabel(outcome),
		Payload:      timelinePayload,
	}
	if err := r.repo.CreateExecutionArtifact(ctx, artifact); err == nil {
		artifactIDs = append(artifactIDs, artifact.ID)
		timelineID := artifact.ID
		return RecordResult{
			StepID:             step.ID,
			ArtifactIDs:        artifactIDs,
			TimelineArtifactID: &timelineID,
		}, nil
	}

	return RecordResult{
		StepID:      step.ID,
		ArtifactIDs: artifactIDs,
	}, nil
}

const maxEmbeddedExternalBytes = 5 * 1024 * 1024

// persistExternalFile stores metadata (and optionally inline base64) for engine-provided files (trace/video/HAR).
func (r *DBRecorder) persistExternalFile(ctx context.Context, executionID uuid.UUID, stepID *uuid.UUID, stepIndex int, artifactType, filePath string) *uuid.UUID {
	info, err := os.Stat(filePath)
	if err != nil {
		if r.log != nil {
			r.log.WithError(err).WithField("path", filePath).Debug("external file not readable")
		}
		return nil
	}
	payload := database.JSONMap{
		"path":       filePath,
		"size_bytes": info.Size(),
	}
	if info.Size() > 0 && info.Size() <= maxEmbeddedExternalBytes {
		if data, readErr := os.ReadFile(filePath); readErr == nil {
			payload["base64"] = base64.StdEncoding.EncodeToString(data)
			payload["inline"] = true
		}
	}

	artifact := &database.ExecutionArtifact{
		ID:           uuid.New(),
		ExecutionID:  executionID,
		StepID:       stepID,
		StepIndex:    &stepIndex,
		ArtifactType: artifactType,
		Label:        artifactType,
		Payload:      payload,
	}
	if err := r.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
		if r.log != nil {
			r.log.WithError(err).WithField("artifact_type", artifactType).Warn("failed to persist external file artifact")
		}
		return nil
	}
	return &artifact.ID
}

// RecordTelemetry persists telemetry as lightweight artifacts so replay can
// inspect dropped messages during early rollout.
func (r *DBRecorder) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	if r == nil || r.repo == nil {
		return nil
	}
	artifact := &database.ExecutionArtifact{
		ID:           uuid.New(),
		ExecutionID:  plan.ExecutionID,
		StepIndex:    &telemetry.StepIndex,
		ArtifactType: "telemetry",
		Payload: database.JSONMap{
			"telemetry": telemetry,
		},
	}
	return r.repo.CreateExecutionArtifact(ctx, artifact)
}

// MarkCrash annotates executions when the executor/engine terminates abruptly.
func (r *DBRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error {
	if r == nil || r.repo == nil {
		return nil
	}
	now := time.Now()
	step := &database.ExecutionStep{
		ID:          uuid.New(),
		ExecutionID: executionID,
		StepIndex:   -1,
		NodeID:      "crash",
		StepType:    "crash",
		Status:      "failed",
		StartedAt:   now,
		CompletedAt: &now,
		Error:       failure.Message,
		Metadata: database.JSONMap{
			"failure": failure,
			"partial": true,
		},
	}
	return r.repo.CreateExecutionStep(ctx, step)
}

func (r *DBRecorder) persistScreenshot(ctx context.Context, executionID uuid.UUID, outcome contracts.StepOutcome) (*storage.ScreenshotInfo, error) {
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

// executionFetcher optionally exposed by repositories that can surface execution rows.
type executionFetcher interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error)
}

// executionCreator optionally exposed by repositories that can create execution rows.
type executionCreator interface {
	CreateExecution(ctx context.Context, execution *database.Execution) error
}

// recoverMissingExecution attempts to recreate the execution row when step persistence fails
// due to a missing execution FK. This guards against out-of-order writes when the executor
// is invoked without a pre-created execution record (e.g., from adhoc tooling).
func (r *DBRecorder) recoverMissingExecution(ctx context.Context, plan contracts.ExecutionPlan, originalErr error) (bool, error) {
	if !isForeignKeyViolation(originalErr) {
		return false, originalErr
	}

	cacheKey := plan.ExecutionID.String()
	if _, seen := r.checkedExecutions.Load(cacheKey); seen {
		// Already attempted recovery for this execution.
		return false, originalErr
	}
	r.checkedExecutions.Store(cacheKey, struct{}{})

	fetcher, okFetch := r.repo.(executionFetcher)
	creator, okCreate := r.repo.(executionCreator)
	if !okCreate {
		return false, originalErr
	}

	if okFetch {
		existing, err := fetcher.GetExecution(ctx, plan.ExecutionID)
		if err == nil && existing != nil {
			// Execution already exists; bubble original error.
			return false, originalErr
		}
	}

	exec := &database.Execution{
		ID:          plan.ExecutionID,
		WorkflowID:  plan.WorkflowID,
		Status:      "running",
		TriggerType: "recovered",
		StartedAt:   time.Now().UTC(),
		Progress:    0,
		CurrentStep: "Recovered execution context",
	}

	if err := creator.CreateExecution(ctx, exec); err != nil {
		return false, fmt.Errorf("recover execution %s: %w", plan.ExecutionID, err)
	}

	if r.log != nil {
		r.log.WithField("execution_id", plan.ExecutionID).Warn("Recovered missing execution before persisting step outcome")
	}
	return true, nil
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
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
	out.Network = sanitizeNetwork(out.Network, &out)

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
		if idx >= contracts.ConsoleEntryMaxBytes { // prevent runaway artifacts in extreme cases
			break
		}
	}
	return sanitized
}

func sanitizeNetwork(events []contracts.NetworkEvent, out *contracts.StepOutcome) []contracts.NetworkEvent {
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
		if idx >= contracts.NetworkPayloadPreviewMaxBytes { // avoid unbounded slices from noisy pages
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

// truncateRunes trims a string to the specified rune count to avoid cutting
// multibyte characters mid-codepoint.
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
