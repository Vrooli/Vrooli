package browserless

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

// Client orchestrates Browserless-backed executions
// and persists resulting artifacts into the scenario stores.
type Client struct {
	log         *logrus.Logger
	repo        database.Repository
	storage     *storage.MinIOClient
	browserless string
	httpClient  *http.Client
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

	return &Client{
		log:         log,
		repo:        repo,
		storage:     storageClient,
		browserless: strings.TrimRight(browserlessURL, "/"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// ExecuteWorkflow executes the supplied workflow via Browserless, capturing artifacts per step.
func (c *Client) ExecuteWorkflow(ctx context.Context, execution *database.Execution, workflow *database.Workflow, emitter *events.Emitter) error {
	instructions, err := c.buildInstructions(workflow)
	if err != nil {
		return err
	}

	if len(instructions) == 0 {
		c.log.WithField("workflow_id", workflow.ID).Info("Workflow has no executable nodes; skipping Browserless execution")
		execution.Progress = 100
		execution.CurrentStep = "No-op"
		return nil
	}

	session := runtime.NewSession(c.browserless, c.httpClient, c.log)
	totalSteps := len(instructions)
	executedSteps := 0
	overallSuccess := true
	var lastErr error

	for idx, instruction := range instructions {
		stepStartProgress := int(math.Round(float64(idx) / float64(totalSteps) * 100))

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

		response, callErr := session.Execute(ctx, []runtime.Instruction{instruction})
		var step runtime.StepResult

		if callErr != nil || response == nil || len(response.Steps) == 0 {
			step = runtime.StepResult{
				Index:   instruction.Index,
				NodeID:  instruction.NodeID,
				Type:    instruction.Type,
				Success: false,
			}
			if callErr != nil {
				step.Error = callErr.Error()
				lastErr = callErr
			} else {
				step.Error = "browserless returned no step result"
				lastErr = fmt.Errorf("browserless returned no step result for node %s", instruction.NodeID)
			}
		} else {
			step = response.Steps[0]
			step.Index = instruction.Index
			if !step.Success {
				if step.Error != "" {
					lastErr = fmt.Errorf(step.Error)
				} else if response.Error != "" {
					lastErr = fmt.Errorf(response.Error)
				}
			} else if !response.Success {
				step.Success = false
				step.Error = response.Error
				if response.Error != "" {
					lastErr = fmt.Errorf(response.Error)
				}
			}
		}

		executedSteps++

		progress := int(math.Round(float64(idx+1) / float64(totalSteps) * 100))
		if progress > 100 {
			progress = 100
		}

		execution.Progress = progress
		execution.CurrentStep = fmt.Sprintf("%s (%s)", step.Type, step.NodeID)

		if err := c.repo.UpdateExecution(ctx, execution); err != nil {
			c.log.WithError(err).Warn("Failed to persist execution progress")
		}

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Timestamp:   time.Now(),
			StepName:    deriveStepName(step),
			Metadata: database.JSONMap{
				"nodeId":     step.NodeID,
				"durationMs": step.DurationMs,
			},
		}

		if step.Success {
			logEntry.Level = "info"
			logEntry.Message = fmt.Sprintf("%s completed in %dms", step.Type, step.DurationMs)
		} else {
			logEntry.Level = "error"
			if step.Error != "" {
				logEntry.Message = fmt.Sprintf("%s failed: %s", step.Type, step.Error)
			} else {
				logEntry.Message = fmt.Sprintf("%s failed", step.Type)
			}
			if step.Stack != "" {
				logEntry.Metadata["stack"] = step.Stack
			}
		}

		if err := c.repo.CreateExecutionLog(ctx, logEntry); err != nil {
			c.log.WithError(err).Warn("Failed to persist execution log entry")
		}

		var screenshotRecord *database.Screenshot
		if step.ScreenshotBase64 != "" {
			record, persistErr := c.persistScreenshot(ctx, execution, step)
			if persistErr != nil {
				c.log.WithError(persistErr).Warn("Failed to persist screenshot artifact")
			} else {
				screenshotRecord = record
			}
		}

		if emitter != nil {
			payload := map[string]any{
				"duration_ms": step.DurationMs,
				"success":     step.Success,
			}
			if step.Error != "" {
				payload["error"] = step.Error
			}
			if step.Stack != "" {
				payload["stack"] = step.Stack
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
				} else {
					screenshotPayload["base64"] = step.ScreenshotBase64
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
		}

		if !step.Success {
			overallSuccess = false
			break
		}
	}

	execution.Result = database.JSONMap{
		"success": overallSuccess,
		"steps":   executedSteps,
	}

	if err := c.repo.UpdateExecution(ctx, execution); err != nil {
		c.log.WithError(err).Warn("Failed to persist final execution state")
	}

	if !overallSuccess {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("browserless execution failed")
	}

	return nil
}

func (c *Client) buildInstructions(workflow *database.Workflow) ([]runtime.Instruction, error) {
	plan, err := compiler.CompileWorkflow(workflow)
	if err != nil {
		return nil, err
	}

	return runtime.InstructionsFromPlan(plan)
}

func (c *Client) persistScreenshot(ctx context.Context, execution *database.Execution, step runtime.StepResult) (*database.Screenshot, error) {
	data, err := base64.StdEncoding.DecodeString(step.ScreenshotBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid screenshot payload: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("screenshot payload is empty")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		c.log.WithError(err).Debug("Failed to read screenshot dimensions; using defaults")
	}

	info, err := c.storeScreenshot(ctx, execution.ID, deriveStepName(step), data)
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

func (c *Client) storeScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte) (*storage.ScreenshotInfo, error) {
	if c.storage != nil {
		info, err := c.storage.StoreScreenshot(ctx, executionID, stepName, data, "image/png")
		if err == nil {
			return info, nil
		}
		c.log.WithError(err).Warn("MinIO upload failed; falling back to local filesystem")
	}

	dir := filepath.Join(os.TempDir(), "browser-automation-studio", executionID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create local screenshot directory: %w", err)
	}

	filename := strings.TrimSpace(stepName)
	if filename == "" {
		filename = fmt.Sprintf("step-%d", time.Now().UnixNano())
	}
	filePath := filepath.Join(dir, filename+".png")

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write fallback screenshot: %w", err)
	}

	return &storage.ScreenshotInfo{
		URL:          filePath,
		ThumbnailURL: filePath,
		SizeBytes:    int64(len(data)),
	}, nil
}

func deriveStepName(step runtime.StepResult) string {
	if strings.TrimSpace(step.StepName) != "" {
		return step.StepName
	}
	return fmt.Sprintf("%s-%d", step.Type, step.Index+1)
}

// CheckBrowserlessHealth verifies the Browserless /pressure endpoint responds.
func (c *Client) CheckBrowserlessHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
