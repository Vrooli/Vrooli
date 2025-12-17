package replay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
)

// captureFrame represents a single captured frame.
type captureFrame struct {
	Index       int     `json:"index"`
	TimestampMs int     `json:"timestampMs"`
	FrameIndex  int     `json:"frameIndex"`
	Progress    float64 `json:"progress"`
	Data        string  `json:"data"`
}

// captureResponse contains the result of a capture operation.
type captureResponse struct {
	Success bool           `json:"success"`
	Error   string         `json:"error"`
	Frames  []captureFrame `json:"frames"`
	FPS     int            `json:"fps"`
	Width   int            `json:"width"`
	Height  int            `json:"height"`
}

// playwrightCaptureClient reuses the automation executor + Playwright engine to
// capture replay frames. It navigates to the export page, injects the movie spec,
// waits for render, and captures screenshots.
type playwrightCaptureClient struct {
	exportPageURL string
}

func newPlaywrightCaptureClient(exportPageURL string) replayCaptureClient {
	return &playwrightCaptureClient{exportPageURL: exportPageURL}
}

func (c *playwrightCaptureClient) Capture(ctx context.Context, spec *ReplayMovieSpec, captureInterval int) (*captureResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("playwright capture client not configured")
	}
	if spec == nil || len(spec.Frames) == 0 {
		return nil, fmt.Errorf("movie spec missing frames")
	}
	if strings.TrimSpace(c.exportPageURL) == "" {
		return nil, fmt.Errorf("export page url not configured")
	}

	// Only attempt when Playwright driver is present.
	if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		return nil, fmt.Errorf("PLAYWRIGHT_DRIVER_URL not set; Playwright driver required for capture")
	}

	factory, err := autoengine.DefaultFactory(nil)
	if err != nil {
		return nil, err
	}
	engine, err := factory.Resolve(ctx, "playwright")
	if err != nil {
		return nil, err
	}
	if engine == nil {
		return nil, errors.New("playwright engine not available")
	}

	rec := &inMemoryCaptureRecorder{}
	exec := autoexecutor.NewSimpleExecutor(nil)
	eventSink := autoevents.NewMemorySink(autocontracts.DefaultEventBufferLimits)

	instructions := buildPlaywrightCaptureInstructions(c.exportPageURL, spec, captureInterval)
	plan := autocontracts.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions:   instructions,
		CreatedAt:      time.Now().UTC(),
	}

	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        "playwright",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         eventSink,
		HeartbeatInterval: 0,
	}

	if err := exec.Execute(ctx, req); err != nil {
		return nil, err
	}

	if len(rec.outcomes) == 0 {
		return nil, fmt.Errorf("playwright capture produced zero outcomes")
	}

	frames := make([]captureFrame, 0, len(rec.outcomes))
	for idx, outcome := range rec.outcomes {
		if outcome.Screenshot == nil || len(outcome.Screenshot.Data) == 0 {
			continue
		}
		ts := 0
		if outcome.CompletedAt != nil && plan.CreatedAt.Before(*outcome.CompletedAt) {
			ts = int(outcome.CompletedAt.Sub(plan.CreatedAt).Milliseconds())
		}
		frames = append(frames, captureFrame{
			Index:       idx,
			Data:        base64.StdEncoding.EncodeToString(outcome.Screenshot.Data),
			TimestampMs: ts,
		})
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("playwright capture returned no frames")
	}
	return &captureResponse{Frames: frames}, nil
}

func buildPlaywrightCaptureInstructions(exportPageURL string, spec *ReplayMovieSpec, captureInterval int) []autocontracts.CompiledInstruction {
	totalMs := spec.Summary.TotalDurationMs
	if totalMs <= 0 && spec.Playback.DurationMs > 0 {
		totalMs = spec.Playback.DurationMs
	}
	if totalMs <= 0 {
		totalMs = captureInterval * len(spec.Frames)
	}
	if totalMs <= 0 {
		totalMs = 5000
	}
	if captureInterval <= 0 {
		captureInterval = 1000
	}

	frameCount := int(math.Ceil(float64(totalMs)/float64(captureInterval))) + 1
	if frameCount < 1 {
		frameCount = 1
	}
	if frameCount > 120 {
		frameCount = 120
	}

	specJSON, _ := json.Marshal(spec)
	script := fmt.Sprintf("window.__BAS_RENDER_SPEC=%s;window.dispatchEvent(new CustomEvent('bas:render',{detail:%d}));", string(specJSON), captureInterval)

	instr := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "navigate",
			Type:   "navigate",
			Params: map[string]any{
				"url":       exportPageURL,
				"timeoutMs": 45000,
			},
		},
		{
			Index:  1,
			NodeID: "inject",
			Type:   "evaluate",
			Params: map[string]any{
				"script": script,
			},
		},
	}

	idx := 2
	for i := 0; i < frameCount; i++ {
		instr = append(instr, autocontracts.CompiledInstruction{
			Index:  idx,
			NodeID: fmt.Sprintf("wait-%d", i),
			Type:   "wait",
			Params: map[string]any{
				"ms": captureInterval,
			},
		})
		idx++
		instr = append(instr, autocontracts.CompiledInstruction{
			Index:  idx,
			NodeID: fmt.Sprintf("screenshot-%d", i),
			Type:   "screenshot",
			Params: map[string]any{
				"fullPage": true,
			},
		})
		idx++
	}

	return instr
}

// inMemoryCaptureRecorder captures outcomes without touching the database.
type inMemoryCaptureRecorder struct {
	outcomes  []autocontracts.StepOutcome
	telemetry []autocontracts.StepTelemetry
}

func (r *inMemoryCaptureRecorder) RecordStepOutcome(_ context.Context, _ autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (executionwriter.RecordResult, error) {
	r.outcomes = append(r.outcomes, outcome)
	return executionwriter.RecordResult{}, nil
}

func (r *inMemoryCaptureRecorder) RecordTelemetry(_ context.Context, _ autocontracts.ExecutionPlan, telemetry autocontracts.StepTelemetry) error {
	r.telemetry = append(r.telemetry, telemetry)
	return nil
}

func (r *inMemoryCaptureRecorder) MarkCrash(_ context.Context, _ uuid.UUID, _ autocontracts.StepFailure) error {
	return nil
}

func (r *inMemoryCaptureRecorder) UpdateCheckpoint(_ context.Context, _ uuid.UUID, _ int, _ int) error {
	return nil // In-memory recorder doesn't persist checkpoints
}
