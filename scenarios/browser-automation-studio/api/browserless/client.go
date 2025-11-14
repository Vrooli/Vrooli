package browserless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/cdp"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/storage"
)

// Client orchestrates Browserless-backed executions
// and persists resulting artifacts into the scenario stores.
type Client struct {
	log               *logrus.Logger
	repo              database.Repository
	storage           *storage.MinIOClient
	browserless       string
	httpClient        *http.Client
	heartbeatInterval time.Duration
	recordingsRoot    string
}

type loopDirective struct {
	Continue bool
	Break    bool
}

type cdpSession interface {
	ExecuteInstruction(ctx context.Context, instruction runtime.Instruction) (*runtime.ExecutionResponse, error)
	Close() error
}

var newCDPSession = func(ctx context.Context, browserlessURL string, viewportWidth, viewportHeight int, log *logrus.Entry) (cdpSession, error) {
	return cdp.NewSession(ctx, browserlessURL, viewportWidth, viewportHeight, log)
}

const (
	defaultHeartbeatInterval    = 2 * time.Second
	minHeartbeatInterval        = 50 * time.Millisecond
	maxHeartbeatInterval        = 5 * time.Minute
	heartbeatIntervalEnvVar     = "BROWSERLESS_HEARTBEAT_INTERVAL"
	domSnapshotPreviewRuneLimit = 2000
)

const (
	loopDefaultMaxIterations      = 100
	loopDefaultItemVariable       = "loop.item"
	loopDefaultIndexVariable      = "loop.index"
	loopDefaultIterationTimeoutMs = 45000
	loopDefaultTotalTimeoutMs     = 300000
)

func decodeDataHTML(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "data:text/html") {
		return "", nil
	}
	comma := strings.Index(raw, ",")
	if comma == -1 {
		return "", fmt.Errorf("invalid data URL: missing comma")
	}
	meta := raw[:comma]
	data := raw[comma+1:]
	if strings.Contains(strings.ToLower(meta), ";base64") {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("base64 decode failed: %w", err)
		}
		return string(decoded), nil
	}
	decoded, err := url.QueryUnescape(data)
	if err != nil {
		return data, nil
	}
	return decoded, nil
}

// NewClient creates a browserless client.
func NewClient(log *logrus.Logger, repo database.Repository) *Client {
	browserlessURL := os.Getenv("BROWSERLESS_URL")
	if browserlessURL == "" {
		if port := os.Getenv("BROWSERLESS_PORT"); port != "" {
			host := os.Getenv("BROWSERLESS_HOST")
			if host == "" {
				host = "localhost"
			}
			browserlessURL = fmt.Sprintf("http://%s:%s", host, port)
		} else {
			log.Fatal("❌ BROWSERLESS_URL or BROWSERLESS_PORT environment variable is required")
		}
	}

	storageClient, err := storage.NewMinIOClient(log)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize MinIO client - screenshot storage will fall back to local filesystem")
	}

	interval := defaultHeartbeatInterval
	if custom, ok := heartbeatIntervalFromEnv(log); ok {
		interval = custom
	}

	recordingsRoot := paths.ResolveRecordingsRoot(log)
	if log != nil {
		log.WithField("recordings_root", recordingsRoot).Debug("Browserless client using recordings root")
	}

	return &Client{
		log:         log,
		repo:        repo,
		storage:     storageClient,
		browserless: strings.TrimRight(browserlessURL, "/"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		heartbeatInterval: interval,
		recordingsRoot:    recordingsRoot,
	}
}

func sanitizeFrameSlug(stepIndex int, stepName string) string {
	normalized := strings.ToLower(strings.TrimSpace(stepName))
	if normalized == "" {
		return fmt.Sprintf("frame-%04d", stepIndex+1)
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		default:
			// skip unsupported characters
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return fmt.Sprintf("frame-%04d", stepIndex+1)
	}
	return slug
}

func makeFrameFilename(stepIndex int, stepName string) string {
	slug := sanitizeFrameSlug(stepIndex, stepName)
	suffix := uuid.New().String()
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return fmt.Sprintf("%s-%s.png", slug, suffix)
}

// ExecuteWorkflow executes the supplied workflow via Browserless, capturing artifacts per step.
func (c *Client) ExecuteWorkflow(ctx context.Context, execution *database.Execution, workflow *database.Workflow, emitter events.Sink) error {
	// Apply workflow-level execution timeout (default: 2 minutes)
	// This prevents workflows from hanging indefinitely if steps retry continuously
	workflowTimeout := 2 * time.Minute
	if timeoutEnv := os.Getenv("WORKFLOW_EXECUTION_TIMEOUT_MINUTES"); timeoutEnv != "" {
		if minutes, err := strconv.Atoi(timeoutEnv); err == nil && minutes > 0 {
			workflowTimeout = time.Duration(minutes) * time.Minute
		}
	}

	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()

	plan, instructions, err := c.compileWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	execCtx := runtime.NewExecutionContext()

	if plan == nil || len(plan.Steps) == 0 || len(instructions) == 0 {
		c.log.WithField("workflow_id", workflow.ID).Info("Workflow has no executable nodes; skipping Browserless execution")
		execution.Progress = 100
		execution.CurrentStep = "No-op"
		return nil
	}

	stepsByNodeID, activeNodes := buildPlanState(plan)

	// Create CDP session for persistent browser connection
	sessionLogger := c.log.WithField("execution_id", execution.ID)
	viewportWidth, viewportHeight := extractPlanViewport(plan)
	if viewportWidth == 0 {
		viewportWidth = 1920
	}
	if viewportHeight == 0 {
		viewportHeight = 1080
	}

	session, err := newCDPSession(ctx, c.browserless, viewportWidth, viewportHeight, sessionLogger)
	if err != nil {
		return fmt.Errorf("failed to create CDP session: %w", err)
	}
	defer session.Close()

	totalSteps := len(plan.Steps)
	if totalSteps == 0 {
		totalSteps = len(instructions)
	}
	executedSteps := 0
	overallSuccess := true
	fatalFailure := false
	var lastErr error
	var lastCursorPosition *runtime.Point
	var lastReplayScreenshot []byte

	for _, instruction := range instructions {
		// Check if workflow execution timeout has been exceeded
		if ctx.Err() != nil {
			c.log.WithField("execution_id", execution.ID).WithError(ctx.Err()).Warn("Workflow execution canceled or timed out")
			execution.CurrentStep = "Timeout"
			return fmt.Errorf("workflow execution timeout exceeded: %w", ctx.Err())
		}

		if !activeNodes[instruction.NodeID] {
			continue
		}
		delete(activeNodes, instruction.NodeID)

		resolvedInstruction, missingVars, err := runtime.InterpolateInstruction(instruction, execCtx)
		if err != nil {
			return fmt.Errorf("failed to prepare instruction %s: %w", instruction.NodeID, err)
		}
		instruction = resolvedInstruction
		if len(missingVars) > 0 && c.log != nil {
			c.log.WithFields(logrus.Fields{
				"execution_id": execution.ID,
				"node_id":      instruction.NodeID,
				"variables":    missingVars,
			}).Warn("Workflow referenced undefined variables")
		}

		stepDefinition, hasStepDefinition := stepsByNodeID[instruction.NodeID]
		if !hasStepDefinition {
			stepDefinition = compiler.ExecutionStep{NodeID: instruction.NodeID, Index: instruction.Index}
		}

		stepStartProgress := progressPercent(executedSteps, totalSteps)
		progress := progressPercent(executedSteps+1, totalSteps)
		if progress > 100 {
			progress = 100
		}

		allowFailure := false
		if instruction.Params.ContinueOnFailure != nil {
			allowFailure = *instruction.Params.ContinueOnFailure
		}

		maxRetries := instruction.Params.RetryAttempts
		if maxRetries < 0 {
			maxRetries = 0
		}
		maxAttempts := maxRetries + 1
		retryDelayMs := instruction.Params.RetryDelayMs
		if retryDelayMs < 0 {
			retryDelayMs = 0
		}
		initialRetryDelayMs := retryDelayMs
		retryBackoffFactor := instruction.Params.RetryBackoffFactor
		if retryBackoffFactor <= 0 {
			retryBackoffFactor = 1
		} else if retryBackoffFactor < 1 {
			retryBackoffFactor = 1
		}

		// CDP mode: No need for PreloadHTML - browser state persists naturally

		stepRecord := &database.ExecutionStep{
			ExecutionID: execution.ID,
			StepIndex:   instruction.Index,
			NodeID:      instruction.NodeID,
			StepType:    instruction.Type,
			Status:      "running",
			StartedAt:   time.Now(),
			Input:       instructionParamsToJSONMap(instruction),
			Metadata: database.JSONMap{
				"nodeId":       instruction.NodeID,
				"stepType":     instruction.Type,
				"workflowId":   workflow.ID.String(),
				"workflowName": workflow.Name,
			},
		}
		retryMetadata := database.JSONMap{
			"maxAttempts":       maxAttempts,
			"configuredRetries": maxRetries,
		}
		if initialRetryDelayMs > 0 {
			retryMetadata["initialDelayMs"] = initialRetryDelayMs
		}
		if retryBackoffFactor != 1 {
			retryMetadata["backoffFactor"] = retryBackoffFactor
		}
		stepRecord.Metadata["retry"] = retryMetadata

		if err := c.repo.CreateExecutionStep(ctx, stepRecord); err != nil {
			c.log.WithError(err).Warn("Failed to persist execution step start")
			stepRecord = nil
		}

		if emitter != nil {
			emitter.Emit(events.NewEvent(
				events.EventStepStarted,
				execution.ID,
				workflow.ID,
				events.WithStep(instruction.Index, instruction.NodeID, instruction.Type),
				events.WithStatus("running"),
				events.WithProgress(stepStartProgress),
				events.WithMessage(fmt.Sprintf("%s started", strings.ToLower(instruction.Type))),
			))
		}

		var cancelHeartbeat context.CancelFunc
		if emitter != nil && c.heartbeatInterval > 0 {
			hbCtx, cancel := context.WithCancel(ctx)
			cancelHeartbeat = cancel
			go c.emitStepHeartbeats(
				hbCtx,
				emitter,
				execution.ID,
				workflow.ID,
				instruction,
				stepStartProgress,
			)
		}

		var step runtime.StepResult
		var retryHistory []database.JSONMap
		var attemptsUsed int
		var totalAttemptDurationMs int
		step, retryHistory, attemptsUsed, totalAttemptDurationMs, lastErr = c.runInstruction(ctx, session, instruction, stepDefinition, execCtx)

		if cancelHeartbeat != nil {
			cancelHeartbeat()
		}

		if totalAttemptDurationMs <= 0 {
			totalAttemptDurationMs = step.DurationMs
		}

		timelineRetryHistory := make([]map[string]any, 0, len(retryHistory))
		for _, entry := range retryHistory {
			if entry == nil {
				continue
			}
			timelineRetryHistory = append(timelineRetryHistory, map[string]any(entry))
		}

		artifactIDs := make([]string, 0, 4)
		timelineArtifactID := ""
		cursorTrail := make([]runtime.Point, 0, 2)

		execution.Progress = progress
		execution.CurrentStep = fmt.Sprintf("%s (%s)", step.Type, step.NodeID)
		now := time.Now()
		execution.LastHeartbeat = &now

		if err := c.repo.UpdateExecution(ctx, execution); err != nil {
			c.log.WithError(err).Warn("Failed to persist execution progress")
		}

		durationForLog := step.DurationMs
		if totalAttemptDurationMs > 0 {
			durationForLog = totalAttemptDurationMs
		}
		if durationForLog <= 0 {
			durationForLog = step.DurationMs
		}

		attemptSuffix := ""
		if attemptsUsed > 1 {
			attemptSuffix = fmt.Sprintf(" after %d attempts", attemptsUsed)
		}

		retryMetadata["attempt"] = attemptsUsed
		if attemptsUsed > 0 {
			retryMetadata["attemptedRetries"] = attemptsUsed - 1
		}
		retryMetadata["success"] = step.Success
		if totalAttemptDurationMs > 0 {
			retryMetadata["totalDurationMs"] = totalAttemptDurationMs
		}
		if len(retryHistory) > 0 {
			retryMetadata["history"] = retryHistory
			if step.Success {
				if err := applyVariablePostProcessing(execCtx, instruction, step); err != nil && c.log != nil {
					c.log.WithError(err).Warn("Failed to update execution context with variable result")
				}
			}
		}

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Timestamp:   time.Now(),
			StepName:    deriveStepName(step),
			Metadata: database.JSONMap{
				"nodeId":     step.NodeID,
				"durationMs": durationForLog,
				"retry":      retryMetadata,
			},
		}

		if step.Success {
			logEntry.Level = "info"
			logEntry.Message = fmt.Sprintf("%s completed in %dms%s", step.Type, durationForLog, attemptSuffix)
		} else {
			logEntry.Level = "error"
			if step.Error != "" {
				logEntry.Message = fmt.Sprintf("%s failed: %s%s", step.Type, step.Error, attemptSuffix)
			} else {
				logEntry.Message = fmt.Sprintf("%s failed%s", step.Type, attemptSuffix)
			}
			if step.Stack != "" {
				logEntry.Metadata["stack"] = step.Stack
			}
		}

		if step.Type == "assert" && step.Assertion != nil {
			if logEntry.Metadata == nil {
				logEntry.Metadata = database.JSONMap{}
			}
			logEntry.Metadata["assertion"] = step.Assertion
			label := strings.TrimSpace(step.Assertion.Selector)
			if label == "" {
				label = strings.TrimSpace(step.NodeID)
			}
			if label == "" {
				label = strings.TrimSpace(step.Type)
			}
			if label == "" {
				label = "assertion"
			}
			if step.Success {
				logEntry.Message = fmt.Sprintf("assert %s passed in %dms%s", label, durationForLog, attemptSuffix)
			} else {
				reason := strings.TrimSpace(step.Error)
				if reason == "" {
					if step.Assertion.Message != "" {
						reason = step.Assertion.Message
					} else {
						reason = "assertion failed"
					}
				}
				logEntry.Message = fmt.Sprintf("assert %s failed: %s%s", label, reason, attemptSuffix)
			}
		}

		if err := c.repo.CreateExecutionLog(ctx, logEntry); err != nil {
			c.log.WithError(err).Warn("Failed to persist execution log entry")
		}

		if step.ClickPosition != nil {
			if lastCursorPosition != nil {
				cursorTrail = append(cursorTrail, *lastCursorPosition)
			}
			cursorTrail = append(cursorTrail, *step.ClickPosition)
			positionCopy := *step.ClickPosition
			lastCursorPosition = &positionCopy
		} else if step.Type == "navigate" {
			lastCursorPosition = nil
		}

		if len(step.ConsoleLogs) > 0 {
			for _, consoleEntry := range step.ConsoleLogs {
				metadata := database.JSONMap{
					"nodeId":      step.NodeID,
					"consoleType": strings.ToLower(strings.TrimSpace(consoleEntry.Type)),
					"timestampMs": consoleEntry.Timestamp,
				}

				consoleLog := &database.ExecutionLog{
					ExecutionID: execution.ID,
					Timestamp:   timestampFromOffset(execution.StartedAt, consoleEntry.Timestamp),
					Level:       mapConsoleLevel(consoleEntry.Type),
					StepName:    deriveStepName(step),
					Message:     consoleEntry.Text,
					Metadata:    metadata,
				}

				if err := c.repo.CreateExecutionLog(ctx, consoleLog); err != nil {
					c.log.WithError(err).Warn("Failed to persist console log entry")
				}
			}

			artifact := &database.ExecutionArtifact{
				ExecutionID:  execution.ID,
				ArtifactType: "console",
				Label:        deriveStepName(step),
				Payload: database.JSONMap{
					"entries": step.ConsoleLogs,
				},
			}
			artifact.StepIndex = intPointer(step.Index)
			if stepRecord != nil {
				artifact.StepID = &stepRecord.ID
			}
			if err := c.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist console artifact")
			} else {
				artifactIDs = append(artifactIDs, artifact.ID.String())
			}
		}

		if len(step.NetworkEvents) > 0 {
			artifact := &database.ExecutionArtifact{
				ExecutionID:  execution.ID,
				ArtifactType: "network",
				Label:        deriveStepName(step),
				Payload: database.JSONMap{
					"events": step.NetworkEvents,
				},
			}
			artifact.StepIndex = intPointer(step.Index)
			if stepRecord != nil {
				artifact.StepID = &stepRecord.ID
			}
			if err := c.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist network artifact")
			} else {
				artifactIDs = append(artifactIDs, artifact.ID.String())
			}
		}

		if step.Assertion != nil {
			assertionArtifact := &database.ExecutionArtifact{
				ExecutionID:  execution.ID,
				ArtifactType: "assertion",
				Label:        deriveStepName(step),
				Payload: database.JSONMap{
					"assertion": step.Assertion,
				},
			}
			assertionArtifact.StepIndex = intPointer(step.Index)
			if stepRecord != nil {
				assertionArtifact.StepID = &stepRecord.ID
			}
			if err := c.repo.CreateExecutionArtifact(ctx, assertionArtifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist assertion artifact")
			} else {
				artifactIDs = append(artifactIDs, assertionArtifact.ID.String())
			}
		}

		if step.ExtractedData != nil {
			artifact := &database.ExtractedData{
				ID:          uuid.New(),
				ExecutionID: execution.ID,
				StepName:    deriveStepName(step),
				Timestamp:   time.Now(),
				DataKey:     step.NodeID,
				DataValue: database.JSONMap{
					"value": step.ExtractedData,
				},
				Metadata: database.JSONMap{
					"nodeId": step.NodeID,
				},
			}
			if step.ElementBoundingBox != nil {
				artifact.Metadata["elementBoundingBox"] = step.ElementBoundingBox
			}
			if err := c.repo.CreateExtractedData(ctx, artifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist extracted data artifact")
			}

			dataArtifact := &database.ExecutionArtifact{
				ExecutionID:  execution.ID,
				ArtifactType: "extracted_data",
				Label:        deriveStepName(step),
				Payload: database.JSONMap{
					"value": step.ExtractedData,
				},
			}
			dataArtifact.StepIndex = intPointer(step.Index)
			if stepRecord != nil {
				dataArtifact.StepID = &stepRecord.ID
			}
			if err := c.repo.CreateExecutionArtifact(ctx, dataArtifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist extracted data artifact record")
			} else {
				artifactIDs = append(artifactIDs, dataArtifact.ID.String())
			}
		}

		var screenshotRecord *database.Screenshot
		var screenshotArtifactID string
		var domSnapshotArtifactID string
		var domSnapshotPreview string
		if step.ScreenshotBase64 != "" {
			rawScreenshot, decodeErr := base64.StdEncoding.DecodeString(step.ScreenshotBase64)
			if decodeErr != nil || len(rawScreenshot) == 0 {
				c.log.WithError(decodeErr).Warn("Failed to decode screenshot payload; skipping persistence")
			} else {
				if strings.EqualFold(step.Type, "wait") && len(lastReplayScreenshot) > 0 {
					rawScreenshot = append([]byte(nil), lastReplayScreenshot...)
					step.ScreenshotBase64 = base64.StdEncoding.EncodeToString(rawScreenshot)
				} else if c.isLowInformationScreenshot(rawScreenshot) && len(lastReplayScreenshot) > 0 {
					rawScreenshot = append([]byte(nil), lastReplayScreenshot...)
					step.ScreenshotBase64 = base64.StdEncoding.EncodeToString(rawScreenshot)
				}

				record, persistErr := c.persistScreenshot(ctx, execution, step, rawScreenshot)
				if persistErr != nil {
					c.log.WithError(persistErr).Warn("Failed to persist screenshot artifact")
				} else {
					screenshotRecord = record
					artifact := &database.ExecutionArtifact{
						ExecutionID:  execution.ID,
						ArtifactType: "screenshot",
						Label:        deriveStepName(step),
						StorageURL:   record.StorageURL,
						ThumbnailURL: record.ThumbnailURL,
						ContentType:  "image/png",
						Payload: database.JSONMap{
							"width":     record.Width,
							"height":    record.Height,
							"stepType":  step.Type,
							"publicUrl": record.StorageURL,
						},
					}
					if record.SizeBytes > 0 {
						size := record.SizeBytes
						artifact.SizeBytes = &size
					}
					artifact.StepIndex = intPointer(step.Index)
					if stepRecord != nil {
						artifact.StepID = &stepRecord.ID
					}
					if step.ElementBoundingBox != nil {
						artifact.Payload["elementBoundingBox"] = step.ElementBoundingBox
					}
					if step.ClickPosition != nil {
						artifact.Payload["clickPosition"] = step.ClickPosition
					}
					if len(cursorTrail) > 0 {
						artifact.Payload["cursorTrail"] = cursorTrail
					}
					if step.FocusedElement != nil {
						artifact.Payload["focusedElement"] = step.FocusedElement
					}
					if len(step.HighlightRegions) > 0 {
						artifact.Payload["highlightRegions"] = step.HighlightRegions
					}
					if len(step.MaskRegions) > 0 {
						artifact.Payload["maskRegions"] = step.MaskRegions
					}
					if step.ZoomFactor > 0 {
						artifact.Payload["zoomFactor"] = step.ZoomFactor
					}
					if err := c.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
						c.log.WithError(err).Warn("Failed to persist screenshot artifact record")
					} else {
						screenshotArtifactID = artifact.ID.String()
						artifactIDs = append(artifactIDs, screenshotArtifactID)
					}
					lastReplayScreenshot = append([]byte(nil), rawScreenshot...)
				}
			}
		}

		if trimmed := strings.TrimSpace(step.DOMSnapshot); trimmed != "" {
			artifact := &database.ExecutionArtifact{
				ExecutionID:  execution.ID,
				ArtifactType: "dom_snapshot",
				Label:        deriveStepName(step),
				Payload: database.JSONMap{
					"html": trimmed,
				},
			}
			artifact.StepIndex = intPointer(step.Index)
			if stepRecord != nil {
				artifact.StepID = &stepRecord.ID
			}
			if err := c.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
				c.log.WithError(err).Warn("Failed to persist DOM snapshot artifact")
			} else {
				domSnapshotArtifactID = artifact.ID.String()
				artifactIDs = append(artifactIDs, domSnapshotArtifactID)
				domSnapshotPreview = truncateRunes(trimmed, domSnapshotPreviewRuneLimit)
			}
		}

		if stepRecord != nil {
			statusValue := "completed"
			if !step.Success {
				statusValue = "failed"
			}
			stepRecord.Status = statusValue
			completedAt := time.Now()
			stepRecord.CompletedAt = &completedAt
			stepRecord.DurationMs = durationForLog
			stepRecord.Error = strings.TrimSpace(step.Error)
			output := stepResultOutputMap(step)
			metadataUpdate := stepMetadataFromResult(step, screenshotRecord, artifactIDs)
			retrySummary := database.JSONMap{
				"attempt":           attemptsUsed,
				"maxAttempts":       maxAttempts,
				"configuredRetries": maxRetries,
				"success":           step.Success,
			}
			if initialRetryDelayMs > 0 {
				retrySummary["initialDelayMs"] = initialRetryDelayMs
			}
			if retryBackoffFactor != 1 {
				retrySummary["backoffFactor"] = retryBackoffFactor
			}
			if attemptsUsed > 0 {
				retrySummary["attemptedRetries"] = attemptsUsed - 1
			}
			if totalAttemptDurationMs > 0 {
				retrySummary["totalDurationMs"] = totalAttemptDurationMs
			}
			if len(retryHistory) > 0 {
				retrySummary["history"] = retryHistory
			}
			if totalAttemptDurationMs > 0 {
				output["totalDurationMs"] = totalAttemptDurationMs
			}
			output["retry"] = retrySummary
			stepRecord.Output = output
			attachVariableSnapshot(execCtx, stepRecord, output)
			metadataUpdate = mergeJSONMaps(metadataUpdate, database.JSONMap{"retry": retrySummary})
			stepRecord.Metadata = mergeJSONMaps(stepRecord.Metadata, metadataUpdate)
			if err := c.repo.UpdateExecutionStep(ctx, stepRecord); err != nil {
				c.log.WithError(err).Warn("Failed to update execution step record")
			}
		}

		existingArtifactRefs := append([]string(nil), artifactIDs...)
		timelinePayload := database.JSONMap{
			"stepIndex":  step.Index,
			"nodeId":     step.NodeID,
			"stepType":   step.Type,
			"success":    step.Success,
			"durationMs": step.DurationMs,
			"progress":   progress,
		}
		if totalAttemptDurationMs > 0 {
			timelinePayload["totalDurationMs"] = totalAttemptDurationMs
		}
		if attemptsUsed > 0 {
			timelinePayload["retryAttempt"] = attemptsUsed
		}
		if maxAttempts > 1 || attemptsUsed > 1 {
			timelinePayload["retryMaxAttempts"] = maxAttempts
			timelinePayload["retryConfigured"] = maxRetries
			if initialRetryDelayMs > 0 {
				timelinePayload["retryDelayMs"] = initialRetryDelayMs
			}
			if retryBackoffFactor != 1 {
				timelinePayload["retryBackoffFactor"] = retryBackoffFactor
			}
			if len(timelineRetryHistory) > 0 {
				timelinePayload["retryHistory"] = timelineRetryHistory
			}
		}
		if stepRecord != nil {
			if stepRecord.ID != uuid.Nil {
				timelinePayload["executionStepId"] = stepRecord.ID.String()
			}
			timelinePayload["startedAt"] = stepRecord.StartedAt
			if stepRecord.CompletedAt != nil {
				timelinePayload["completedAt"] = *stepRecord.CompletedAt
			}
		}
		if step.FinalURL != "" {
			timelinePayload["finalUrl"] = step.FinalURL
		}
		if step.ElementBoundingBox != nil {
			timelinePayload["elementBoundingBox"] = step.ElementBoundingBox
		}
		if step.ClickPosition != nil {
			timelinePayload["clickPosition"] = step.ClickPosition
		}
		if step.FocusedElement != nil {
			timelinePayload["focusedElement"] = step.FocusedElement
		}
		if len(step.HighlightRegions) > 0 {
			timelinePayload["highlightRegions"] = step.HighlightRegions
		}
		if len(step.MaskRegions) > 0 {
			timelinePayload["maskRegions"] = step.MaskRegions
		}
		if step.ZoomFactor > 0 {
			timelinePayload["zoomFactor"] = step.ZoomFactor
		}
		if step.ExtractedData != nil {
			timelinePayload["extractedDataPreview"] = step.ExtractedData
		}
		if step.Assertion != nil {
			timelinePayload["assertion"] = step.Assertion
		}
		if len(step.ConsoleLogs) > 0 {
			timelinePayload["consoleLogCount"] = len(step.ConsoleLogs)
		}
		if len(step.NetworkEvents) > 0 {
			timelinePayload["networkEventCount"] = len(step.NetworkEvents)
		}
		if !step.Success && strings.TrimSpace(step.Error) != "" {
			timelinePayload["error"] = strings.TrimSpace(step.Error)
		}
		if screenshotArtifactID != "" {
			timelinePayload["screenshotArtifactId"] = screenshotArtifactID
		}
		if domSnapshotArtifactID != "" {
			timelinePayload["domSnapshotArtifactId"] = domSnapshotArtifactID
		}
		if domSnapshotPreview != "" {
			timelinePayload["domSnapshotPreview"] = domSnapshotPreview
		}
		if len(cursorTrail) > 0 {
			timelinePayload["cursorTrail"] = cursorTrail
		}
		if len(existingArtifactRefs) > 0 {
			timelinePayload["artifactIds"] = existingArtifactRefs
		}

		timelineArtifact := &database.ExecutionArtifact{
			ExecutionID:  execution.ID,
			ArtifactType: "timeline_frame",
			Label:        deriveStepName(step),
			Payload:      timelinePayload,
		}
		timelineArtifact.StepIndex = intPointer(step.Index)
		if stepRecord != nil {
			timelineArtifact.StepID = &stepRecord.ID
		}
		if err := c.repo.CreateExecutionArtifact(ctx, timelineArtifact); err != nil {
			c.log.WithError(err).Warn("Failed to persist timeline artifact record")
		} else {
			timelineArtifactID = timelineArtifact.ID.String()
			artifactIDs = append(artifactIDs, timelineArtifactID)
		}

		if emitter != nil {
			payload := map[string]any{
				"duration_ms": step.DurationMs,
				"success":     step.Success,
			}
			if totalAttemptDurationMs > 0 {
				payload["total_duration_ms"] = totalAttemptDurationMs
			}
			if attemptsUsed > 0 {
				payload["retry_attempt"] = attemptsUsed
			}
			if maxAttempts > 1 || attemptsUsed > 1 {
				payload["retry_max_attempts"] = maxAttempts
				payload["retry_configured"] = maxRetries
				if initialRetryDelayMs > 0 {
					payload["retry_delay_ms"] = initialRetryDelayMs
				}
				if retryBackoffFactor != 1 {
					payload["retry_backoff_factor"] = retryBackoffFactor
				}
				if len(timelineRetryHistory) > 0 {
					payload["retry_history"] = timelineRetryHistory
				}
			}
			if step.Error != "" {
				payload["error"] = step.Error
			}
			if step.Stack != "" {
				payload["stack"] = step.Stack
			}
			if step.FinalURL != "" {
				payload["final_url"] = step.FinalURL
			}
			if step.ElementBoundingBox != nil {
				payload["element_bounding_box"] = step.ElementBoundingBox
			}
			if step.ClickPosition != nil {
				payload["click_position"] = step.ClickPosition
			}
			if len(cursorTrail) > 0 {
				payload["cursor_trail"] = cursorTrail
			}
			if step.FocusedElement != nil {
				payload["focused_element"] = step.FocusedElement
			}
			if len(step.HighlightRegions) > 0 {
				payload["highlight_regions"] = step.HighlightRegions
			}
			if len(step.MaskRegions) > 0 {
				payload["mask_regions"] = step.MaskRegions
			}
			if step.ZoomFactor > 0 {
				payload["zoom_factor"] = step.ZoomFactor
			}
			if step.ExtractedData != nil {
				payload["extracted_data"] = step.ExtractedData
			}
			if domSnapshotPreview != "" {
				payload["dom_snapshot_preview"] = domSnapshotPreview
			}
			if domSnapshotArtifactID != "" {
				payload["dom_snapshot_artifact_id"] = domSnapshotArtifactID
			}
			if len(step.ConsoleLogs) > 0 {
				payload["console_log_count"] = len(step.ConsoleLogs)
			}
			if len(step.NetworkEvents) > 0 {
				payload["network_event_count"] = len(step.NetworkEvents)
			}
			if step.Assertion != nil {
				payload["assertion"] = step.Assertion
			}
			if stepRecord != nil && stepRecord.ID != uuid.Nil {
				payload["execution_step_id"] = stepRecord.ID.String()
			}
			if len(artifactIDs) > 0 {
				payload["artifact_ids"] = artifactIDs
			}
			if timelineArtifactID != "" {
				payload["timeline_artifact_id"] = timelineArtifactID
			}

			eventType := events.EventStepCompleted
			status := "succeeded"
			message := fmt.Sprintf("%s completed", strings.ToLower(step.Type))
			if !step.Success {
				eventType = events.EventStepFailed
				status = "failed"
				if strings.TrimSpace(step.Error) != "" {
					message = fmt.Sprintf("%s failed: %s", strings.ToLower(step.Type), step.Error)
				} else {
					message = fmt.Sprintf("%s failed", strings.ToLower(step.Type))
				}
			}
			if step.Type == "assert" && step.Assertion != nil {
				label := strings.TrimSpace(step.Assertion.Selector)
				if label == "" {
					label = strings.TrimSpace(step.NodeID)
				}
				if label == "" {
					label = "assertion"
				}
				if step.Success {
					message = fmt.Sprintf("assert %s passed", label)
				} else if strings.TrimSpace(step.Error) == "" && step.Assertion.Message != "" {
					message = fmt.Sprintf("assert %s failed: %s", label, step.Assertion.Message)
				}
			}

			emitter.Emit(events.NewEvent(
				eventType,
				execution.ID,
				workflow.ID,
				events.WithStep(step.Index, step.NodeID, step.Type),
				events.WithStatus(status),
				events.WithProgress(progress),
				events.WithMessage(message),
				events.WithPayload(payload),
			))

			if step.ScreenshotBase64 != "" {
				screenshotPayload := map[string]any{}
				if screenshotRecord != nil {
					screenshotPayload["screenshot_id"] = screenshotRecord.ID.String()
					screenshotPayload["url"] = screenshotRecord.StorageURL
					screenshotPayload["thumbnail_url"] = screenshotRecord.ThumbnailURL
					screenshotPayload["width"] = screenshotRecord.Width
					screenshotPayload["height"] = screenshotRecord.Height
					screenshotPayload["size_bytes"] = screenshotRecord.SizeBytes
					if screenshotArtifactID != "" {
						screenshotPayload["artifact_id"] = screenshotArtifactID
					}
				} else {
					screenshotPayload["base64"] = step.ScreenshotBase64
				}

				if step.FocusedElement != nil {
					screenshotPayload["focused_element"] = step.FocusedElement
				}
				if len(step.HighlightRegions) > 0 {
					screenshotPayload["highlight_regions"] = step.HighlightRegions
				}
				if len(step.MaskRegions) > 0 {
					screenshotPayload["mask_regions"] = step.MaskRegions
				}
				if step.ZoomFactor > 0 {
					screenshotPayload["zoom_factor"] = step.ZoomFactor
				}
				if timelineArtifactID != "" {
					screenshotPayload["timeline_artifact_id"] = timelineArtifactID
				}
				if attemptsUsed > 0 {
					screenshotPayload["retry_attempt"] = attemptsUsed
				}
				if maxAttempts > 1 {
					screenshotPayload["retry_max_attempts"] = maxAttempts
				}

				emitter.Emit(events.NewEvent(
					events.EventStepScreenshot,
					execution.ID,
					workflow.ID,
					events.WithStep(step.Index, step.NodeID, step.Type),
					events.WithStatus(status),
					events.WithProgress(progress),
					events.WithPayload(screenshotPayload),
				))
			}

			if len(step.ConsoleLogs) > 0 || len(step.NetworkEvents) > 0 {
				telemetryPayload := map[string]any{}
				if len(step.ConsoleLogs) > 0 {
					telemetryPayload["console_logs"] = step.ConsoleLogs
				}
				if len(step.NetworkEvents) > 0 {
					telemetryPayload["network_events"] = step.NetworkEvents
				}
				if step.Assertion != nil {
					telemetryPayload["assertion"] = step.Assertion
				}
				if attemptsUsed > 0 {
					telemetryPayload["retry_attempt"] = attemptsUsed
				}
				if maxAttempts > 1 {
					telemetryPayload["retry_max_attempts"] = maxAttempts
				}

				emitter.Emit(events.NewEvent(
					events.EventStepTelemetry,
					execution.ID,
					workflow.ID,
					events.WithStep(step.Index, step.NodeID, step.Type),
					events.WithStatus(status),
					events.WithProgress(progress),
					events.WithPayload(telemetryPayload),
				))
			}
		}

		nextTargets, hasFailureEdge, directive := evaluateOutgoingEdges(stepDefinition, step, allowFailure)
		if (directive.Continue || directive.Break) && c.log != nil {
			c.log.WithFields(logrus.Fields{
				"execution_id": execution.ID,
				"node_id":      instruction.NodeID,
			}).Warn("Loop directive emitted outside loop body; ignoring")
		}
		for _, targetNode := range nextTargets {
			if _, exists := stepsByNodeID[targetNode]; !exists {
				continue
			}
			activeNodes[targetNode] = true
		}

		executedSteps++

		if !step.Success {
			overallSuccess = false
			if !allowFailure && !hasFailureEdge {
				fatalFailure = true
			}
		}

		if fatalFailure {
			break
		}
	}

	execution.Result = database.JSONMap{
		"success": overallSuccess,
		"steps":   executedSteps,
	}

	if !fatalFailure {
		execution.Progress = 100
	}

	if err := c.repo.UpdateExecution(ctx, execution); err != nil {
		c.log.WithError(err).Warn("Failed to persist final execution state")
	}

	if fatalFailure {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("browserless execution failed")
	}

	return nil
}

func (c *Client) buildInstructions(workflow *database.Workflow) ([]runtime.Instruction, error) {
	_, instructions, err := c.compileWorkflow(context.Background(), workflow)
	return instructions, err
}

func (c *Client) compileWorkflow(ctx context.Context, workflow *database.Workflow) (*compiler.ExecutionPlan, []runtime.Instruction, error) {
	plan, err := compiler.CompileWorkflow(workflow)
	if err != nil {
		return nil, nil, err
	}

	instructions, err := runtime.InstructionsFromPlan(ctx, plan)
	if err != nil {
		return nil, nil, err
	}

	return plan, instructions, nil
}

func extractPlanViewport(plan *compiler.ExecutionPlan) (int, int) {
	if plan == nil || plan.Metadata == nil {
		return 0, 0
	}
	viewportRaw, ok := plan.Metadata["executionViewport"]
	if !ok {
		return 0, 0
	}
	viewportMap, ok := viewportRaw.(map[string]any)
	if !ok {
		return 0, 0
	}
	width := toPositiveInt(viewportMap["width"])
	height := toPositiveInt(viewportMap["height"])
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}

func toPositiveInt(value any) int {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case float32:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case json.Number:
		if intVal, err := v.Int64(); err == nil && intVal > 0 {
			return int(intVal)
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func progressPercent(completed, total int) int {
	if total <= 0 {
		if completed > 0 {
			return 100
		}
		return 0
	}

	if completed <= 0 {
		return 0
	}

	percent := int(math.Round(float64(completed) / float64(total) * 100))
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return percent
}

func evaluateOutgoingEdges(step compiler.ExecutionStep, result runtime.StepResult, allowFailure bool) ([]string, bool, loopDirective) {
	if len(step.OutgoingEdges) == 0 {
		return nil, false, loopDirective{}
	}

	successTargets := make([]string, 0, len(step.OutgoingEdges))
	failureTargets := make([]string, 0, len(step.OutgoingEdges))
	alwaysTargets := make([]string, 0, len(step.OutgoingEdges))
	elseTargets := make([]string, 0, len(step.OutgoingEdges))
	assertionPassTargets := make([]string, 0)
	assertionFailTargets := make([]string, 0)
	fallbackTargets := make([]string, 0)
	conditionTrueTargets := make([]string, 0)
	conditionFalseTargets := make([]string, 0)
	hasFailureEdge := false
	directive := loopDirective{}

	for _, edge := range step.OutgoingEdges {
		targetNode := strings.TrimSpace(edge.TargetNode)
		if targetNode == compiler.LoopContinueTarget {
			directive.Continue = true
			continue
		}
		if targetNode == compiler.LoopBreakTarget {
			directive.Break = true
			continue
		}
		condition := strings.TrimSpace(strings.ToLower(edge.Condition))
		switch condition {
		case "", "next", "success", "on_success", "pass", "passed", "true", "yes":
			successTargets = append(successTargets, edge.TargetNode)
		case "failure", "on_failure", "fail", "failed", "error", "false", "no":
			failureTargets = append(failureTargets, edge.TargetNode)
			hasFailureEdge = true
		case "always", "any":
			alwaysTargets = append(alwaysTargets, edge.TargetNode)
		case "else", "default", "otherwise":
			elseTargets = append(elseTargets, edge.TargetNode)
		case "assert_pass", "assertion_pass", "assert_success", "assert_true":
			assertionPassTargets = append(assertionPassTargets, edge.TargetNode)
		case "assert_fail", "assertion_fail", "assert_failure", "assert_false":
			assertionFailTargets = append(assertionFailTargets, edge.TargetNode)
			hasFailureEdge = true
		case "if_true", "condition_true", "true_branch", "branch_true":
			conditionTrueTargets = append(conditionTrueTargets, edge.TargetNode)
		case "if_false", "condition_false", "false_branch", "branch_false":
			conditionFalseTargets = append(conditionFalseTargets, edge.TargetNode)
		case "loop_continue", "continue_loop":
			directive.Continue = true
		case "loop_break", "break_loop":
			directive.Break = true
		default:
			fallbackTargets = append(fallbackTargets, edge.TargetNode)
		}
	}

	hasExplicitFailureTargets := len(failureTargets) > 0 || len(assertionFailTargets) > 0
	if !hasFailureEdge && hasExplicitFailureTargets {
		hasFailureEdge = true
	}

	dedup := make(map[string]struct{})
	targets := make([]string, 0, len(step.OutgoingEdges))
	appendTargets := func(nodes []string) {
		for _, node := range nodes {
			trimmed := strings.TrimSpace(node)
			if trimmed == "" {
				continue
			}
			if trimmed == compiler.LoopContinueTarget {
				directive.Continue = true
				continue
			}
			if trimmed == compiler.LoopBreakTarget {
				directive.Break = true
				continue
			}
			if _, exists := dedup[trimmed]; exists {
				continue
			}
			dedup[trimmed] = struct{}{}
			targets = append(targets, trimmed)
		}
	}

	assertion := result.Assertion
	if assertion != nil {
		if assertion.Success {
			appendTargets(assertionPassTargets)
		} else {
			appendTargets(assertionFailTargets)
		}
	} else if !result.Success {
		appendTargets(assertionFailTargets)
	}

	if len(conditionTrueTargets) > 0 || len(conditionFalseTargets) > 0 {
		if cond := result.ConditionResult; cond != nil {
			if cond.Outcome {
				appendTargets(conditionTrueTargets)
			} else {
				appendTargets(conditionFalseTargets)
			}
		}
	}

	if result.Success {
		appendTargets(successTargets)
		appendTargets(fallbackTargets)
	} else {
		appendTargets(failureTargets)
		if allowFailure && !hasExplicitFailureTargets {
			appendTargets(successTargets)
			appendTargets(fallbackTargets)
		}
	}

	appendTargets(alwaysTargets)

	if len(targets) == 0 {
		if result.Success {
			appendTargets(successTargets)
			appendTargets(fallbackTargets)
		} else {
			if allowFailure && !hasExplicitFailureTargets {
				appendTargets(successTargets)
				appendTargets(fallbackTargets)
			}
			appendTargets(assertionFailTargets)
			appendTargets(failureTargets)
		}
	}

	if len(targets) == 0 {
		appendTargets(elseTargets)
	}

	if len(targets) == 0 {
		appendTargets(fallbackTargets)
	}

	return targets, hasFailureEdge, directive
}

func (c *Client) runInstruction(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
	stepDefinition compiler.ExecutionStep,
	execCtx *runtime.ExecutionContext,
) (runtime.StepResult, []database.JSONMap, int, int, error) {
	switch stepDefinition.Type {
	case compiler.StepLoop:
		return c.executeLoopStep(ctx, session, instruction, stepDefinition, execCtx)
	case compiler.StepWorkflowCall:
		return c.executeWorkflowCallStep(ctx, session, instruction, execCtx)
	default:
		return c.executeInstructionWithRetries(ctx, session, instruction)
	}
}

func (c *Client) executeInstructionWithRetries(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
) (runtime.StepResult, []database.JSONMap, int, int, error) {
	maxRetries := instruction.Params.RetryAttempts
	if maxRetries < 0 {
		maxRetries = 0
	}
	maxAttempts := maxRetries + 1
	retryDelayMs := instruction.Params.RetryDelayMs
	if retryDelayMs < 0 {
		retryDelayMs = 0
	}
	currentDelayMs := retryDelayMs
	retryBackoffFactor := instruction.Params.RetryBackoffFactor
	if retryBackoffFactor <= 0 {
		retryBackoffFactor = 1
	}
	initialRetryDelayMs := retryDelayMs
	retryHistory := make([]database.JSONMap, 0, maxAttempts)
	totalAttemptDurationMs := 0
	attemptsUsed := 0
	var lastErr error
	var step runtime.StepResult

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptsUsed = attempt
		attemptStart := time.Now()
		response, callErr := session.ExecuteInstruction(ctx, instruction)
		callDuration := int(time.Since(attemptStart) / time.Millisecond)

		attemptStep := runtime.StepResult{
			Index:  instruction.Index,
			NodeID: instruction.NodeID,
			Type:   instruction.Type,
		}

		if callErr != nil || response == nil || len(response.Steps) == 0 {
			attemptStep.Success = false
			if callErr != nil {
				attemptStep.Error = callErr.Error()
				lastErr = callErr
			} else {
				attemptStep.Error = "browserless returned no step result"
				lastErr = fmt.Errorf("browserless returned no step result for node %s", instruction.NodeID)
			}
		} else {
			attemptStep = response.Steps[0]
			if strings.TrimSpace(attemptStep.NodeID) == "" {
				attemptStep.NodeID = instruction.NodeID
			}
			if strings.TrimSpace(attemptStep.Type) == "" {
				attemptStep.Type = instruction.Type
			}
			attemptStep.Index = instruction.Index
			if instruction.Type == "navigate" {
				if instruction.Params.Scenario != "" {
					attemptStep.Scenario = instruction.Params.Scenario
				}
				if instruction.Params.ScenarioPath != "" {
					attemptStep.ScenarioPath = instruction.Params.ScenarioPath
				}
				if instruction.Params.DestinationType != "" {
					attemptStep.DestinationType = instruction.Params.DestinationType
				}
			}
			if !attemptStep.Success {
				if attemptStep.Error != "" {
					lastErr = errors.New(attemptStep.Error)
				} else if response.Error != "" {
					lastErr = errors.New(response.Error)
				}
			} else if response != nil && !response.Success {
				attemptStep.Success = false
				if response.Error != "" {
					attemptStep.Error = response.Error
					lastErr = errors.New(response.Error)
				}
			}
		}

		if attemptStep.DurationMs <= 0 && callDuration > 0 {
			attemptStep.DurationMs = callDuration
		}
		totalAttemptDurationMs += attemptStep.DurationMs

		historyEntry := database.JSONMap{
			"attempt": attempt,
			"success": attemptStep.Success,
		}
		if callDuration > 0 {
			historyEntry["callDurationMs"] = callDuration
		}
		if attemptStep.DurationMs > 0 {
			historyEntry["durationMs"] = attemptStep.DurationMs
		}
		if trimmed := strings.TrimSpace(attemptStep.Error); trimmed != "" {
			historyEntry["error"] = trimmed
		}
		retryHistory = append(retryHistory, historyEntry)

		step = attemptStep

		if step.Success {
			lastErr = nil
			break
		}

		if ctx.Err() != nil {
			lastErr = ctx.Err()
			if step.Error == "" && lastErr != nil {
				step.Error = lastErr.Error()
			}
			break
		}

		if attempt >= maxAttempts {
			break
		}

		if currentDelayMs > 0 {
			delayDuration := time.Duration(currentDelayMs) * time.Millisecond
			timer := time.NewTimer(delayDuration)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				lastErr = ctx.Err()
				if step.Error == "" && lastErr != nil {
					step.Error = lastErr.Error()
				}
				attemptsUsed = attempt
				attempt = maxAttempts
				continue
			}
		}

		if currentDelayMs > 0 && retryBackoffFactor > 1 {
			currentDelayMs = int(float64(currentDelayMs) * retryBackoffFactor)
		}
		if initialRetryDelayMs == 0 && retryDelayMs > 0 {
			initialRetryDelayMs = retryDelayMs
		}
	}

	return step, retryHistory, attemptsUsed, totalAttemptDurationMs, lastErr
}

type loopBodyOutcome struct {
	Success    bool
	Break      bool
	Continue   bool
	DurationMs int
}

func buildInstructionIndex(instructions []runtime.Instruction) map[string]runtime.Instruction {
	index := make(map[string]runtime.Instruction, len(instructions))
	for _, instr := range instructions {
		index[instr.NodeID] = instr
	}
	return index
}

func buildStepIndex(plan *compiler.ExecutionPlan) map[string]compiler.ExecutionStep {
	index := make(map[string]compiler.ExecutionStep, len(plan.Steps))
	for _, step := range plan.Steps {
		index[step.NodeID] = step
	}
	return index
}

func initialActiveNodes(plan *compiler.ExecutionPlan) map[string]bool {
	active := make(map[string]bool)
	if plan == nil {
		return active
	}
	incoming := make(map[string]int, len(plan.Steps))
	for _, step := range plan.Steps {
		incoming[step.NodeID] = 0
	}
	for _, step := range plan.Steps {
		for _, edge := range step.OutgoingEdges {
			target := strings.TrimSpace(edge.TargetNode)
			if target == "" {
				continue
			}
			if target == compiler.LoopBreakTarget || target == compiler.LoopContinueTarget {
				continue
			}
			if _, ok := incoming[target]; ok {
				incoming[target]++
			}
		}
	}
	for nodeID, count := range incoming {
		if count == 0 {
			active[nodeID] = true
		}
	}
	if len(active) == 0 && len(plan.Steps) > 0 {
		active[plan.Steps[0].NodeID] = true
	}
	return active
}

func resolveLoopItems(execCtx *runtime.ExecutionContext, source string) ([]any, error) {
	if execCtx == nil {
		return nil, fmt.Errorf("loop variable context is not initialized")
	}
	value, ok := execCtx.Get(source)
	if !ok {
		return nil, fmt.Errorf("variable %s is not defined", source)
	}
	if value == nil {
		return nil, fmt.Errorf("variable %s is nil", source)
	}
	rv := reflect.ValueOf(value)
	kind := rv.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return nil, fmt.Errorf("variable %s is not iterable", source)
	}
	length := rv.Len()
	items := make([]any, 0, length)
	for i := 0; i < length; i++ {
		items = append(items, rv.Index(i).Interface())
	}
	return items, nil
}

func applyLoopVariables(execCtx *runtime.ExecutionContext, instruction runtime.Instruction, iteration, total int, item any) {
	if execCtx == nil {
		return
	}
	indexVariable := instruction.Params.LoopIndexVariable
	if indexVariable == "" {
		indexVariable = loopDefaultIndexVariable
	}
	itemVariable := instruction.Params.LoopItemVariable
	if itemVariable == "" {
		itemVariable = loopDefaultItemVariable
	}
	execCtx.Set(indexVariable, iteration)
	execCtx.Set(itemVariable, item)
	execCtx.Set("loop.index", iteration)
	execCtx.Set("loop.iteration", iteration+1)
	execCtx.Set("loop.item", item)
	execCtx.Set("loop.total", total)
	execCtx.Set("loop.isFirst", iteration == 0)
	if total > 0 {
		execCtx.Set("loop.isLast", iteration == total-1)
	} else {
		execCtx.Set("loop.isLast", false)
	}
}

func clearLoopVariables(execCtx *runtime.ExecutionContext, instruction runtime.Instruction) {
	if execCtx == nil {
		return
	}
	execCtx.Delete(instruction.Params.LoopIndexVariable)
	execCtx.Delete(instruction.Params.LoopItemVariable)
	for _, key := range []string{"loop.index", "loop.iteration", "loop.item", "loop.total", "loop.isFirst", "loop.isLast"} {
		execCtx.Delete(key)
	}
}

func evaluateLoopVariableCondition(execCtx *runtime.ExecutionContext, variable, operator string, expected any) (bool, error) {
	if execCtx == nil {
		return false, fmt.Errorf("execution context is nil")
	}
	value, ok := execCtx.Get(variable)
	if !ok {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(operator)) {
	case "", "truthy":
		return isTruthy(value), nil
	case "equals":
		return valuesEqual(value, expected), nil
	case "not_equals", "not-equals":
		return !valuesEqual(value, expected), nil
	case "contains":
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), strings.ToLower(fmt.Sprintf("%v", expected))), nil
	default:
		return false, fmt.Errorf("unsupported loop operator %q", operator)
	}
}

func (c *Client) evaluateLoopExpression(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
	variation string,
	execCtx *runtime.ExecutionContext,
) (bool, error) {
	if strings.TrimSpace(variation) == "" {
		return false, fmt.Errorf("loop expression is empty")
	}
	timeout := instruction.Params.TimeoutMs
	if timeout <= 0 {
		timeout = defaultLoopConditionTimeoutMs
	}
	evalInstruction := runtime.Instruction{
		NodeID: fmt.Sprintf("%s::loop-condition", instruction.NodeID),
		Type:   string(compiler.StepEvaluate),
		Params: runtime.InstructionParam{
			Expression: variation,
			TimeoutMs:  timeout,
		},
	}
	resolved, missing, err := runtime.InterpolateInstruction(evalInstruction, execCtx)
	if err != nil {
		return false, err
	}
	if len(missing) > 0 && c.log != nil {
		c.log.WithFields(logrus.Fields{
			"node_id":   instruction.NodeID,
			"variables": missing,
		}).Warn("Loop expression referenced undefined variables")
	}
	step, _, _, _, execErr := c.executeInstructionWithRetries(ctx, session, resolved)
	if execErr != nil {
		return false, execErr
	}
	if !step.Success {
		if step.Error != "" {
			return false, errors.New(step.Error)
		}
		return false, fmt.Errorf("loop expression evaluation failed")
	}
	return isTruthy(step.ExtractedData), nil
}

func isTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return strings.TrimSpace(v) != "" && strings.TrimSpace(strings.ToLower(v)) != "false"
	case int, int32, int64:
		return fmt.Sprintf("%v", v) != "0"
	case float32, float64:
		return fmt.Sprintf("%v", v) != "0"
	default:
		return true
	}
}

func valuesEqual(actual, expected any) bool {
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

const defaultLoopConditionTimeoutMs = 5000

func (c *Client) evaluateLoopCondition(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
	execCtx *runtime.ExecutionContext,
) (bool, error) {
	condType := strings.ToLower(strings.TrimSpace(instruction.Params.LoopConditionType))
	if condType == "" {
		condType = "variable"
	}
	switch condType {
	case "variable":
		variable := strings.TrimSpace(instruction.Params.LoopConditionVariable)
		if variable == "" {
			return false, fmt.Errorf("while loop requires conditionVariable")
		}
		operator := instruction.Params.LoopConditionOperator
		if operator == "" {
			operator = "truthy"
		}
		return evaluateLoopVariableCondition(execCtx, variable, operator, instruction.Params.LoopConditionValue)
	case "expression":
		expression := strings.TrimSpace(instruction.Params.LoopConditionExpression)
		if expression == "" {
			return false, fmt.Errorf("while loop requires conditionExpression")
		}
		return c.evaluateLoopExpression(ctx, session, instruction, expression, execCtx)
	default:
		return false, fmt.Errorf("unsupported loop condition type %q", condType)
	}
}

func (c *Client) runLoopBody(
	ctx context.Context,
	session cdpSession,
	plan *compiler.ExecutionPlan,
	instructions []runtime.Instruction,
	execCtx *runtime.ExecutionContext,
) (loopBodyOutcome, error) {
	outcome := loopBodyOutcome{Success: true}
	if plan == nil || len(plan.Steps) == 0 {
		return outcome, nil
	}
	stepsIndex := buildStepIndex(plan)
	instructionOrder := instructions
	if len(instructionOrder) == 0 {
		var err error
		instructionOrder, err = runtime.InstructionsFromPlan(ctx, plan)
		if err != nil {
			return outcome, err
		}
	}
	active := initialActiveNodes(plan)
	if len(active) == 0 {
		return outcome, fmt.Errorf("loop body has no entry nodes")
	}
	start := time.Now()
	for _, instr := range instructionOrder {
		if !active[instr.NodeID] {
			continue
		}
		delete(active, instr.NodeID)

		resolved, missing, err := runtime.InterpolateInstruction(instr, execCtx)
		if err != nil {
			return outcome, err
		}
		if len(missing) > 0 && c.log != nil {
			c.log.WithFields(logrus.Fields{
				"node_id":   instr.NodeID,
				"variables": missing,
			}).Warn("Loop body referenced undefined variables")
		}

		stepDef, ok := stepsIndex[instr.NodeID]
		if !ok {
			return outcome, fmt.Errorf("loop body missing step definition for node %s", instr.NodeID)
		}

		stepResult, _, _, durationMs, execErr := c.executeInstructionWithRetries(ctx, session, resolved)
		outcome.DurationMs += durationMs
		allowFailure := false
		if resolved.Params.ContinueOnFailure != nil {
			allowFailure = *resolved.Params.ContinueOnFailure
		}
		if err := applyVariablePostProcessing(execCtx, resolved, stepResult); err != nil && c.log != nil {
			c.log.WithError(err).Warn("Failed to persist loop variable output")
		}
		if execErr != nil && stepResult.Error == "" {
			stepResult.Error = execErr.Error()
		}
		if !stepResult.Success && !allowFailure {
			outcome.Success = false
			if execErr != nil {
				return outcome, execErr
			}
			if stepResult.Error != "" {
				return outcome, errors.New(stepResult.Error)
			}
			return outcome, fmt.Errorf("loop body step %s failed", instr.NodeID)
		}

		nextTargets, _, directive := evaluateOutgoingEdges(stepDef, stepResult, allowFailure)
		if directive.Break {
			outcome.Break = true
			break
		}
		if directive.Continue {
			outcome.Continue = true
			break
		}
		for _, target := range nextTargets {
			active[target] = true
		}
	}

	if outcome.DurationMs <= 0 {
		outcome.DurationMs = int(time.Since(start) / time.Millisecond)
	}

	if len(active) > 0 && !outcome.Break && !outcome.Continue {
		outcome.Success = false
		return outcome, fmt.Errorf("loop body did not execute %d pending nodes", len(active))
	}

	return outcome, nil
}

func (c *Client) executeLoopStep(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
	stepDefinition compiler.ExecutionStep,
	execCtx *runtime.ExecutionContext,
) (runtime.StepResult, []database.JSONMap, int, int, error) {
	result := runtime.StepResult{
		Index:   instruction.Index,
		NodeID:  instruction.NodeID,
		Type:    instruction.Type,
		Success: true,
	}
	retryHistory := []database.JSONMap{}
	if stepDefinition.LoopPlan == nil {
		err := fmt.Errorf("loop node %s is missing loop plan", instruction.NodeID)
		result.Success = false
		result.Error = err.Error()
		return result, retryHistory, 1, 0, err
	}
	bodyInstructions, err := runtime.InstructionsFromPlan(ctx, stepDefinition.LoopPlan)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, retryHistory, 1, 0, err
	}
	loopType := strings.ToLower(strings.TrimSpace(instruction.Params.LoopType))
	if loopType == "" {
		loopType = "foreach"
	}
	maxIterations := instruction.Params.LoopMaxIterations
	if maxIterations <= 0 {
		maxIterations = loopDefaultMaxIterations
	}
	iterationTimeout := instruction.Params.LoopIterationTimeoutMs
	if iterationTimeout <= 0 {
		iterationTimeout = loopDefaultIterationTimeoutMs
	}
	totalTimeout := instruction.Params.LoopTotalTimeoutMs
	if totalTimeout <= 0 {
		totalTimeout = loopDefaultTotalTimeoutMs
	}
	iterations := 0
	totalDuration := 0
	start := time.Now()
	var dataset []any
	switch loopType {
	case "foreach":
		source := strings.TrimSpace(instruction.Params.LoopArraySource)
		if source == "" {
			err := fmt.Errorf("loop node %s requires arraySource", instruction.NodeID)
			result.Success = false
			result.Error = err.Error()
			return result, retryHistory, 1, 0, err
		}
		dataset, err = resolveLoopItems(execCtx, source)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, retryHistory, 1, 0, err
		}
		if len(dataset) > maxIterations {
			err := fmt.Errorf("loop node %s exceeded maxIterations (%d)", instruction.NodeID, maxIterations)
			result.Success = false
			result.Error = err.Error()
			return result, retryHistory, 1, 0, err
		}
		for idx, item := range dataset {
			applyLoopVariables(execCtx, instruction, idx, len(dataset), item)
			iterationStart := time.Now()
			outcome, runErr := c.runLoopBody(ctx, session, stepDefinition.LoopPlan, bodyInstructions, execCtx)
			iterationDuration := outcome.DurationMs
			if iterationDuration <= 0 {
				iterationDuration = int(time.Since(iterationStart) / time.Millisecond)
			}
			iterations++
			totalDuration += iterationDuration
			if runErr != nil {
				result.Success = false
				result.Error = runErr.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, runErr
			}
			if !outcome.Success {
				result.Success = false
				result.Error = "loop body failed"
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, fmt.Errorf("loop body failed")
			}
			if err := enforceLoopTimeouts(instruction.NodeID, iterationDuration, totalDuration, iterationTimeout, totalTimeout); err != nil {
				result.Success = false
				result.Error = err.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, err
			}
			if outcome.Break {
				break
			}
			if outcome.Continue {
				continue
			}
		}
	case "repeat":
		repeatCount := instruction.Params.LoopCount
		if repeatCount <= 0 {
			return result, retryHistory, 1, 0, fmt.Errorf("loop node %s requires count > 0", instruction.NodeID)
		}
		if repeatCount > maxIterations {
			maxIterations = repeatCount
		}
		for idx := 0; idx < repeatCount; idx++ {
			applyLoopVariables(execCtx, instruction, idx, repeatCount, idx)
			iterationStart := time.Now()
			outcome, runErr := c.runLoopBody(ctx, session, stepDefinition.LoopPlan, bodyInstructions, execCtx)
			iterationDuration := outcome.DurationMs
			if iterationDuration <= 0 {
				iterationDuration = int(time.Since(iterationStart) / time.Millisecond)
			}
			iterations++
			totalDuration += iterationDuration
			if runErr != nil {
				result.Success = false
				result.Error = runErr.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, runErr
			}
			if !outcome.Success {
				result.Success = false
				result.Error = "loop body failed"
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, fmt.Errorf("loop body failed")
			}
			if err := enforceLoopTimeouts(instruction.NodeID, iterationDuration, totalDuration, iterationTimeout, totalTimeout); err != nil {
				result.Success = false
				result.Error = err.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, err
			}
			if outcome.Break {
				break
			}
			if outcome.Continue {
				continue
			}
		}
	case "while":
		for iterations < maxIterations {
			conditionMet, condErr := c.evaluateLoopCondition(ctx, session, instruction, execCtx)
			if condErr != nil {
				result.Success = false
				result.Error = condErr.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, condErr
			}
			if !conditionMet {
				break
			}
			applyLoopVariables(execCtx, instruction, iterations, maxIterations, iterations)
			iterationStart := time.Now()
			outcome, runErr := c.runLoopBody(ctx, session, stepDefinition.LoopPlan, bodyInstructions, execCtx)
			iterationDuration := outcome.DurationMs
			if iterationDuration <= 0 {
				iterationDuration = int(time.Since(iterationStart) / time.Millisecond)
			}
			iterations++
			totalDuration += iterationDuration
			if runErr != nil {
				result.Success = false
				result.Error = runErr.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, runErr
			}
			if !outcome.Success {
				result.Success = false
				result.Error = "loop body failed"
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, fmt.Errorf("loop body failed")
			}
			if err := enforceLoopTimeouts(instruction.NodeID, iterationDuration, totalDuration, iterationTimeout, totalTimeout); err != nil {
				result.Success = false
				result.Error = err.Error()
				clearLoopVariables(execCtx, instruction)
				return result, retryHistory, 1, totalDuration, err
			}
			if outcome.Break {
				break
			}
			if outcome.Continue {
				continue
			}
		}
		if iterations >= maxIterations {
			err := fmt.Errorf("loop node %s exceeded maxIterations (%d)", instruction.NodeID, maxIterations)
			result.Success = false
			result.Error = err.Error()
			clearLoopVariables(execCtx, instruction)
			return result, retryHistory, 1, totalDuration, err
		}
	default:
		err := fmt.Errorf("loop node %s has unsupported loopType %q", instruction.NodeID, loopType)
		result.Success = false
		result.Error = err.Error()
		return result, retryHistory, 1, 0, err
	}

	clearLoopVariables(execCtx, instruction)
	if result.Success {
		result.ProbeResult = map[string]any{
			"iterations": iterations,
			"loopType":   loopType,
		}
	}
	if totalDuration <= 0 {
		totalDuration = int(time.Since(start) / time.Millisecond)
	}
	result.DurationMs = totalDuration
	return result, retryHistory, 1, totalDuration, nil
}

func enforceLoopTimeouts(nodeID string, iterationDuration, totalDuration, iterationTimeout, totalTimeout int) error {
	if iterationTimeout > 0 && iterationDuration > iterationTimeout {
		return fmt.Errorf("loop node %s exceeded iteration timeout (%dms > %dms)", nodeID, iterationDuration, iterationTimeout)
	}
	if totalTimeout > 0 && totalDuration > totalTimeout {
		return fmt.Errorf("loop node %s exceeded total timeout (%dms > %dms)", nodeID, totalDuration, totalTimeout)
	}
	return nil
}

type workflowCallParamSnapshot struct {
	existed bool
	value   any
}

func (c *Client) applyWorkflowCallParameters(execCtx *runtime.ExecutionContext, params map[string]any) func() {
	if execCtx == nil || len(params) == 0 {
		return func() {}
	}
	backup := make(map[string]workflowCallParamSnapshot, len(params))
	for key, value := range params {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		previous, existed := execCtx.Get(trimmed)
		backup[trimmed] = workflowCallParamSnapshot{existed: existed, value: previous}
		if err := execCtx.Set(trimmed, value); err != nil && c.log != nil {
			c.log.WithError(err).WithField("variable", trimmed).Warn("Failed to set workflow call parameter")
		}
	}
	return func() {
		for key, snapshot := range backup {
			if snapshot.existed {
				if err := execCtx.Set(key, snapshot.value); err != nil && c.log != nil {
					c.log.WithError(err).WithField("variable", key).Warn("Failed to restore workflow call parameter")
				}
			} else {
				execCtx.Delete(key)
			}
		}
	}
}

func buildPlanState(plan *compiler.ExecutionPlan) (map[string]compiler.ExecutionStep, map[string]bool) {
	stepsByNodeID := make(map[string]compiler.ExecutionStep)
	activeNodes := make(map[string]bool)
	if plan == nil {
		return stepsByNodeID, activeNodes
	}
	incomingCounts := make(map[string]int, len(plan.Steps))
	for _, step := range plan.Steps {
		stepsByNodeID[step.NodeID] = step
		incomingCounts[step.NodeID] = 0
	}
	for _, step := range plan.Steps {
		for _, edge := range step.OutgoingEdges {
			incomingCounts[edge.TargetNode]++
		}
	}
	for nodeID, count := range incomingCounts {
		if _, exists := stepsByNodeID[nodeID]; !exists {
			continue
		}
		if count == 0 {
			activeNodes[nodeID] = true
		}
	}
	if len(activeNodes) == 0 && len(plan.Steps) > 0 {
		activeNodes[plan.Steps[0].NodeID] = true
	}
	return stepsByNodeID, activeNodes
}

func (c *Client) executeWorkflowCallStep(
	ctx context.Context,
	session cdpSession,
	instruction runtime.Instruction,
	execCtx *runtime.ExecutionContext,
) (runtime.StepResult, []database.JSONMap, int, int, error) {
	stepResult := runtime.StepResult{
		Index:   instruction.Index,
		NodeID:  instruction.NodeID,
		Type:    instruction.Type,
		Success: true,
	}
	retryHistory := []database.JSONMap{}
	workflowID := strings.TrimSpace(instruction.Params.WorkflowCallID)
	if workflowID == "" {
		err := fmt.Errorf("workflow call node %s missing workflowId", instruction.NodeID)
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, retryHistory, 1, 0, err
	}
	waitForCompletion := true
	if instruction.Params.WorkflowCallWait != nil {
		waitForCompletion = *instruction.Params.WorkflowCallWait
	}
	if !waitForCompletion {
		err := fmt.Errorf("workflow call node %s must wait for completion (async execution not supported)", instruction.NodeID)
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, retryHistory, 1, 0, err
	}
	parsedID, err := uuid.Parse(workflowID)
	if err != nil {
		err = fmt.Errorf("workflow call node %s has invalid workflowId %q: %w", instruction.NodeID, workflowID, err)
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, retryHistory, 1, 0, err
	}
	workflow, err := c.repo.GetWorkflow(ctx, parsedID)
	if err != nil {
		err = fmt.Errorf("workflow call node %s failed to load workflow: %w", instruction.NodeID, err)
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, retryHistory, 1, 0, err
	}
	plan, instructions, err := c.compileWorkflow(ctx, workflow)
	if err != nil {
		err = fmt.Errorf("workflow call node %s failed to compile workflow: %w", instruction.NodeID, err)
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, retryHistory, 1, 0, err
	}
	cleanup := func() {}
	if execCtx != nil && len(instruction.Params.WorkflowCallParams) > 0 {
		cleanup = c.applyWorkflowCallParameters(execCtx, instruction.Params.WorkflowCallParams)
	}
	defer cleanup()
	start := time.Now()
	inlineErr := c.runInlineExecution(ctx, session, plan, instructions, execCtx)
	durationMs := int(time.Since(start) / time.Millisecond)
	stepResult.DurationMs = durationMs
	if inlineErr != nil {
		stepResult.Success = false
		stepResult.Error = inlineErr.Error()
	}
	workflowName := strings.TrimSpace(instruction.Params.WorkflowCallName)
	if workflowName == "" {
		workflowName = workflow.Name
	}
	outputs := map[string]any{}
	if execCtx != nil && len(instruction.Params.WorkflowCallOutputs) > 0 {
		for source, target := range instruction.Params.WorkflowCallOutputs {
			src := strings.TrimSpace(source)
			dst := strings.TrimSpace(target)
			if src == "" || dst == "" {
				continue
			}
			if value, ok := execCtx.Get(src); ok {
				outputs[dst] = value
				if src != dst {
					if err := execCtx.Set(dst, value); err != nil && c.log != nil {
						c.log.WithError(err).WithField("variable", dst).Warn("Failed to persist workflow call output variable")
					}
				}
			}
		}
	}
	metadata := map[string]any{
		"workflowId": workflow.ID.String(),
	}
	if workflowName != "" {
		metadata["workflowName"] = workflowName
	}
	if len(outputs) > 0 {
		metadata["outputs"] = outputs
	}
	stepResult.ExtractedData = metadata
	if inlineErr != nil {
		return stepResult, retryHistory, 1, durationMs, inlineErr
	}
	return stepResult, retryHistory, 1, durationMs, nil
}

func (c *Client) runInlineExecution(
	ctx context.Context,
	session cdpSession,
	plan *compiler.ExecutionPlan,
	instructions []runtime.Instruction,
	execCtx *runtime.ExecutionContext,
) error {
	if plan == nil || len(plan.Steps) == 0 || len(instructions) == 0 {
		return nil
	}
	stepsByNodeID, activeNodes := buildPlanState(plan)
	if len(stepsByNodeID) == 0 {
		return nil
	}
	for _, instruction := range instructions {
		if !activeNodes[instruction.NodeID] {
			continue
		}
		delete(activeNodes, instruction.NodeID)
		resolved, missing, err := runtime.InterpolateInstruction(instruction, execCtx)
		if err != nil {
			return fmt.Errorf("failed to interpolate workflow call instruction %s: %w", instruction.NodeID, err)
		}
		if len(missing) > 0 && c.log != nil {
			c.log.WithFields(logrus.Fields{
				"node_id":   instruction.NodeID,
				"variables": missing,
			}).Warn("Workflow call referenced undefined variables")
		}
		stepDefinition, hasDefinition := stepsByNodeID[instruction.NodeID]
		if !hasDefinition {
			stepDefinition = compiler.ExecutionStep{NodeID: instruction.NodeID, Index: instruction.Index}
		}
		allowFailure := false
		if resolved.Params.ContinueOnFailure != nil {
			allowFailure = *resolved.Params.ContinueOnFailure
		}
		step, _, _, _, stepErr := c.runInstruction(ctx, session, resolved, stepDefinition, execCtx)
		if err := applyVariablePostProcessing(execCtx, resolved, step); err != nil && c.log != nil {
			c.log.WithError(err).Warn("Failed to persist inline workflow variable output")
		}
		nextTargets, hasFailureEdge, directive := evaluateOutgoingEdges(stepDefinition, step, allowFailure)
		if (directive.Break || directive.Continue) && c.log != nil {
			c.log.WithField("node_id", instruction.NodeID).Warn("Loop directive emitted outside loop body; ignoring")
		}
		for _, target := range nextTargets {
			if _, exists := stepsByNodeID[target]; !exists {
				continue
			}
			activeNodes[target] = true
		}
		if !step.Success && !allowFailure && !hasFailureEdge {
			if stepErr != nil {
				return stepErr
			}
			if strings.TrimSpace(step.Error) != "" {
				return fmt.Errorf("workflow call step %s failed: %s", instruction.NodeID, step.Error)
			}
			return fmt.Errorf("workflow call step %s failed", instruction.NodeID)
		}
	}
	return nil
}

func (c *Client) emitStepHeartbeats(
	ctx context.Context,
	emitter events.Sink,
	executionID, workflowID uuid.UUID,
	instruction runtime.Instruction,
	progress int,
) {
	interval := c.heartbeatInterval
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			emittedProgress := progress
			elapsed := time.Since(start)
			message := fmt.Sprintf("%s in progress (%.1fs elapsed)", strings.ToLower(instruction.Type), elapsed.Seconds())
			payload := map[string]any{
				"elapsed_ms": elapsed.Milliseconds(),
				"step_type":  instruction.Type,
			}
			if instruction.NodeID != "" {
				payload["node_id"] = instruction.NodeID
			}

			emitter.Emit(events.NewEvent(
				events.EventStepHeartbeat,
				executionID,
				workflowID,
				events.WithStep(instruction.Index, instruction.NodeID, instruction.Type),
				events.WithStatus("running"),
				events.WithProgress(emittedProgress),
				events.WithMessage(message),
				events.WithPayload(payload),
			))
		}
	}
}

func heartbeatIntervalFromEnv(log *logrus.Logger) (time.Duration, bool) {
	value := strings.TrimSpace(os.Getenv(heartbeatIntervalEnvVar))
	if value == "" {
		return 0, false
	}

	dur, err := time.ParseDuration(value)
	if err != nil {
		if log != nil {
			log.WithError(err).WithField("heartbeat_interval", value).Warn("Invalid BROWSERLESS_HEARTBEAT_INTERVAL; using default")
		}
		return 0, false
	}

	if dur == 0 {
		if log != nil {
			log.Warn("BROWSERLESS_HEARTBEAT_INTERVAL set to 0; disabling heartbeats")
		}
		return 0, true
	}

	if dur < minHeartbeatInterval {
		if log != nil {
			log.WithField("heartbeat_interval", value).Warnf("BROWSERLESS_HEARTBEAT_INTERVAL below minimum; using %s", minHeartbeatInterval)
		}
		return minHeartbeatInterval, true
	}

	if dur > maxHeartbeatInterval {
		if log != nil {
			log.WithField("heartbeat_interval", value).Warnf("BROWSERLESS_HEARTBEAT_INTERVAL above maximum; using %s", maxHeartbeatInterval)
		}
		return maxHeartbeatInterval, true
	}

	return dur, true
}

func (c *Client) persistScreenshot(ctx context.Context, execution *database.Execution, step runtime.StepResult, data []byte) (*database.Screenshot, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("screenshot payload is empty")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		c.log.WithError(err).Debug("Failed to read screenshot dimensions; using defaults")
	}

	info, err := c.storeScreenshot(ctx, execution.ID, step.Index, deriveStepName(step), data, cfg.Width, cfg.Height)
	if err != nil {
		return nil, err
	}

	if cfg.Width > 0 {
		info.Width = cfg.Width
	}
	if cfg.Height > 0 {
		info.Height = cfg.Height
	}

	record := &database.Screenshot{
		ID:           uuid.New(),
		ExecutionID:  execution.ID,
		StepName:     deriveStepName(step),
		Timestamp:    time.Now(),
		StorageURL:   info.URL,
		ThumbnailURL: info.ThumbnailURL,
		Width:        info.Width,
		Height:       info.Height,
		SizeBytes:    info.SizeBytes,
		Metadata: database.JSONMap{
			"nodeId":    step.NodeID,
			"stepIndex": step.Index,
		},
	}

	if err := c.repo.CreateScreenshot(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) isLowInformationScreenshot(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}

	bounds := img.Bounds()
	if bounds.Empty() {
		return true
	}

	width := bounds.Dx()
	height := bounds.Dy()
	stepX := width / 64
	if stepX < 1 {
		stepX = 1
	}
	stepY := height / 64
	if stepY < 1 {
		stepY = 1
	}

	uniqueColors := make(map[[3]uint16]struct{}, 16)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			key := [3]uint16{uint16(r >> 8), uint16(g >> 8), uint16(b >> 8)}
			uniqueColors[key] = struct{}{}
			if len(uniqueColors) > 1 {
				return false
			}
		}
	}

	return len(uniqueColors) <= 1
}

// IsLowInformationScreenshotForTesting exposes low-information detection for unit tests.
func (c *Client) IsLowInformationScreenshotForTesting(data []byte) bool {
	return c.isLowInformationScreenshot(data)
}

func (c *Client) storeScreenshot(ctx context.Context, executionID uuid.UUID, stepIndex int, stepName string, data []byte, width, height int) (*storage.ScreenshotInfo, error) {
	if c.storage != nil {
		info, err := c.storage.StoreScreenshot(ctx, executionID, stepName, data, "image/png")
		if err == nil {
			if width > 0 && info.Width == 0 {
				info.Width = width
			}
			if height > 0 && info.Height == 0 {
				info.Height = height
			}
			if info.ThumbnailURL == "" {
				info.ThumbnailURL = info.URL
			}
			return info, nil
		}
		c.log.WithError(err).Warn("MinIO upload failed; falling back to recordings filesystem")
	}

	root := strings.TrimSpace(c.recordingsRoot)
	if root == "" {
		return nil, fmt.Errorf("recordings root not configured")
	}

	framesDir := filepath.Join(root, executionID.String(), "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create recordings directory: %w", err)
	}

	filename := makeFrameFilename(stepIndex, stepName)
	filePath := filepath.Join(framesDir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to persist replay frame: %w", err)
	}

	publicURL := fmt.Sprintf("/api/v1/recordings/assets/%s/frames/%s", executionID.String(), filename)

	info := &storage.ScreenshotInfo{
		URL:          publicURL,
		ThumbnailURL: publicURL,
		SizeBytes:    int64(len(data)),
		Width:        width,
		Height:       height,
	}

	if (info.Width == 0 || info.Height == 0) && len(data) > 0 {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
			if info.Width == 0 {
				info.Width = cfg.Width
			}
			if info.Height == 0 {
				info.Height = cfg.Height
			}
		}
	}

	return info, nil
}

func instructionParamsToJSONMap(instruction runtime.Instruction) database.JSONMap {
	params := instruction.Params
	encoded, err := json.Marshal(params)
	if err != nil {
		return database.JSONMap{}
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return database.JSONMap{}
	}
	if len(decoded) == 0 {
		return database.JSONMap{}
	}

	return database.JSONMap(decoded)
}

func stepResultOutputMap(step runtime.StepResult) database.JSONMap {
	output := database.JSONMap{
		"success":    step.Success,
		"durationMs": step.DurationMs,
	}

	if step.FinalURL != "" {
		output["finalUrl"] = step.FinalURL
	}
	if step.Scenario != "" {
		output["scenario"] = step.Scenario
	}
	if step.ScenarioPath != "" {
		output["scenarioPath"] = step.ScenarioPath
	}
	if step.DestinationType != "" {
		output["destinationType"] = step.DestinationType
	}
	if step.ElementBoundingBox != nil {
		output["elementBoundingBox"] = step.ElementBoundingBox
	}
	if step.ClickPosition != nil {
		output["clickPosition"] = step.ClickPosition
	}
	if step.ExtractedData != nil {
		output["extractedData"] = step.ExtractedData
	}
	if step.Error != "" {
		output["error"] = step.Error
	}
	if step.Stack != "" {
		output["stack"] = step.Stack
	}
	if len(step.ConsoleLogs) > 0 {
		output["consoleLogCount"] = len(step.ConsoleLogs)
	}
	if len(step.NetworkEvents) > 0 {
		output["networkEventCount"] = len(step.NetworkEvents)
	}
	if step.FocusedElement != nil {
		output["focusedElement"] = step.FocusedElement
	}
	if len(step.HighlightRegions) > 0 {
		output["highlightRegions"] = step.HighlightRegions
	}
	if len(step.MaskRegions) > 0 {
		output["maskRegions"] = step.MaskRegions
	}
	if step.ZoomFactor > 0 {
		output["zoomFactor"] = step.ZoomFactor
	}
	if strings.TrimSpace(step.DOMSnapshot) != "" {
		output["domSnapshotPreview"] = truncateRunes(step.DOMSnapshot, domSnapshotPreviewRuneLimit)
	}

	return output
}

func stepMetadataFromResult(step runtime.StepResult, screenshot *database.Screenshot, artifactIDs []string) database.JSONMap {
	metadata := database.JSONMap{}

	if step.FinalURL != "" {
		metadata["finalUrl"] = step.FinalURL
	}
	if step.ElementBoundingBox != nil {
		metadata["elementBoundingBox"] = step.ElementBoundingBox
	}
	if step.ClickPosition != nil {
		metadata["clickPosition"] = step.ClickPosition
	}
	if screenshot != nil {
		metadata["screenshotId"] = screenshot.ID.String()
		metadata["screenshotUrl"] = screenshot.StorageURL
		metadata["screenshotWidth"] = screenshot.Width
		metadata["screenshotHeight"] = screenshot.Height
	}
	if len(artifactIDs) > 0 {
		metadata["artifactIds"] = artifactIDs
	}
	if step.FocusedElement != nil {
		metadata["focusedElement"] = step.FocusedElement
	}
	if len(step.HighlightRegions) > 0 {
		metadata["highlightRegions"] = step.HighlightRegions
	}
	if len(step.MaskRegions) > 0 {
		metadata["maskRegions"] = step.MaskRegions
	}
	if step.ZoomFactor > 0 {
		metadata["zoomFactor"] = step.ZoomFactor
	}
	if step.Assertion != nil {
		metadata["assertion"] = step.Assertion
	}
	if strings.TrimSpace(step.DOMSnapshot) != "" {
		metadata["domSnapshotPreview"] = truncateRunes(step.DOMSnapshot, domSnapshotPreviewRuneLimit)
	}

	return metadata
}

func truncateRunes(input string, limit int) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || limit <= 0 {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "..."
}

func mergeJSONMaps(base database.JSONMap, updates ...database.JSONMap) database.JSONMap {
	merged := database.JSONMap{}
	for k, v := range base {
		merged[k] = v
	}
	for _, update := range updates {
		for k, v := range update {
			merged[k] = v
		}
	}
	return merged
}

func intPointer(v int) *int {
	value := v
	return &value
}

func deriveStepName(step runtime.StepResult) string {
	if strings.TrimSpace(step.StepName) != "" {
		return step.StepName
	}
	return fmt.Sprintf("%s-%d", step.Type, step.Index+1)
}

func applyVariablePostProcessing(execCtx *runtime.ExecutionContext, instruction runtime.Instruction, step runtime.StepResult) error {
	if execCtx == nil || !step.Success {
		return nil
	}
	value := step.ExtractedData
	if value == nil && instruction.Type == string(compiler.StepSetVariable) {
		value = instruction.Params.VariableValue
	}
	targets := make([]string, 0, 2)
	switch instruction.Type {
	case string(compiler.StepSetVariable):
		if name := strings.TrimSpace(instruction.Params.VariableName); name != "" {
			targets = append(targets, name)
		}
		if store := strings.TrimSpace(instruction.Params.StoreResult); store != "" && store != instruction.Params.VariableName {
			targets = append(targets, store)
		}
	case string(compiler.StepUseVariable):
		if store := strings.TrimSpace(instruction.Params.StoreResult); store != "" {
			targets = append(targets, store)
		}
	default:
		if store := strings.TrimSpace(instruction.Params.StoreResult); store != "" {
			targets = append(targets, store)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	for _, target := range targets {
		if err := execCtx.Set(target, value); err != nil {
			return err
		}
	}
	return nil
}

func attachVariableSnapshot(execCtx *runtime.ExecutionContext, stepRecord *database.ExecutionStep, output database.JSONMap) {
	if execCtx == nil {
		return
	}
	snapshot := execCtx.Snapshot()
	if len(snapshot) == 0 {
		return
	}
	if output != nil {
		output["variables"] = snapshot
	}
	if stepRecord != nil {
		if stepRecord.Metadata == nil {
			stepRecord.Metadata = database.JSONMap{}
		}
		stepRecord.Metadata["variables"] = snapshot
	}
}

func mapConsoleLevel(consoleType string) string {
	switch strings.ToLower(strings.TrimSpace(consoleType)) {
	case "error":
		return "error"
	case "warning", "warn":
		return "warning"
	case "debug":
		return "debug"
	case "info":
		return "info"
	default:
		return "info"
	}
}

func timestampFromOffset(start time.Time, offsetMs int64) time.Time {
	if start.IsZero() {
		return time.Now()
	}
	return start.Add(time.Duration(offsetMs) * time.Millisecond)
}

// CheckBrowserlessHealth verifies the Browserless /pressure endpoint responds.
func (c *Client) CheckBrowserlessHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.BrowserlessHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.browserless+"/pressure", nil)
	if err != nil {
		return fmt.Errorf("failed to build browserless health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("browserless health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("browserless health check returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
