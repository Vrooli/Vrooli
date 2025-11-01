package browserless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
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

const (
	defaultHeartbeatInterval    = 2 * time.Second
	minHeartbeatInterval        = 50 * time.Millisecond
	maxHeartbeatInterval        = 5 * time.Minute
	heartbeatIntervalEnvVar     = "BROWSERLESS_HEARTBEAT_INTERVAL"
	domSnapshotPreviewRuneLimit = 2000
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

	recordingsRoot := resolveRecordingsRoot(log)
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

func resolveRecordingsRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_RECORDINGS_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_RECORDINGS_ROOT without normalization")
		}
		return value
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for recordings root; using relative default")
		}
		return filepath.Join("scenarios", "browser-automation-studio", "data", "recordings")
	}

	absCwd, absErr := filepath.Abs(cwd)
	if absErr == nil {
		dir := absCwd
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			if filepath.Base(dir) == "browser-automation-studio" && filepath.Base(parent) == "scenarios" {
				recordings := filepath.Join(dir, "data", "recordings")
				if abs, err := filepath.Abs(recordings); err == nil {
					return abs
				}
				return recordings
			}
			dir = parent
		}
	}

	root := filepath.Join(absCwd, "scenarios", "browser-automation-studio", "data", "recordings")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
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
	plan, instructions, err := c.compileWorkflow(ctx, workflow)
	if err != nil {
		return err
	}

	if plan == nil || len(plan.Steps) == 0 || len(instructions) == 0 {
		c.log.WithField("workflow_id", workflow.ID).Info("Workflow has no executable nodes; skipping Browserless execution")
		execution.Progress = 100
		execution.CurrentStep = "No-op"
		return nil
	}

	stepsByNodeID := make(map[string]compiler.ExecutionStep, len(plan.Steps))
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

	activeNodes := make(map[string]bool, len(plan.Steps))
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

	session := runtime.NewSession(c.browserless, c.httpClient, c.log)
	if width, height := extractPlanViewport(plan); width > 0 && height > 0 {
		session.SetViewport(width, height)
	}
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
		if !activeNodes[instruction.NodeID] {
			continue
		}
		delete(activeNodes, instruction.NodeID)

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

		if instruction.Type != "navigate" {
			instruction.PreloadHTML = session.LastHTML()
		} else {
			instruction.PreloadHTML = ""
		}

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

		retryHistory := make([]database.JSONMap, 0, maxAttempts)
		currentDelayMs := retryDelayMs
		totalAttemptDurationMs := 0
		attemptsUsed := 0
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
				// Preserve scenario information from instruction params for navigate steps
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
						lastErr = fmt.Errorf(attemptStep.Error)
					} else if response.Error != "" {
						lastErr = fmt.Errorf(response.Error)
					}
				} else if response != nil && !response.Success {
					attemptStep.Success = false
					if response.Error != "" {
						attemptStep.Error = response.Error
						lastErr = fmt.Errorf(response.Error)
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
				if instruction.Type == "navigate" {
					html, decodeErr := decodeDataHTML(instruction.Params.URL)
					if decodeErr != nil {
						c.log.WithError(decodeErr).Debug("Failed to decode data URL for preload")
						session.SetLastHTML("")
					} else {
						session.SetLastHTML(html)
					}
				}
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
					attempt = maxAttempts // force exit
					continue
				}
			}

			if currentDelayMs > 0 {
				nextDelay := int(float64(currentDelayMs) * retryBackoffFactor)
				if nextDelay <= 0 {
					nextDelay = currentDelayMs
				}
				currentDelayMs = nextDelay
			}
		}

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

		nextTargets, hasFailureEdge := evaluateOutgoingEdges(stepDefinition, step, allowFailure)
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

func evaluateOutgoingEdges(step compiler.ExecutionStep, result runtime.StepResult, allowFailure bool) ([]string, bool) {
	if len(step.OutgoingEdges) == 0 {
		return nil, false
	}

	successTargets := make([]string, 0, len(step.OutgoingEdges))
	failureTargets := make([]string, 0, len(step.OutgoingEdges))
	alwaysTargets := make([]string, 0, len(step.OutgoingEdges))
	elseTargets := make([]string, 0, len(step.OutgoingEdges))
	assertionPassTargets := make([]string, 0)
	assertionFailTargets := make([]string, 0)
	fallbackTargets := make([]string, 0)
	hasFailureEdge := false

	for _, edge := range step.OutgoingEdges {
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

	return targets, hasFailureEdge
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
