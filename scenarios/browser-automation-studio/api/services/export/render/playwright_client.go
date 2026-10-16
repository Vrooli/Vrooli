package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

// viewportStabilizationScript is injected before capture begins to prevent
// layout shifts that cause bottom-of-frame flickering in assembled videos.
//
// The root cause: when page content changes during playback, a vertical
// scrollbar can appear or disappear, changing the viewport width by ~17px.
// Each screenshot captures the page at its current width, so frames alternate
// between two widths. The FFmpeg filter chain normalizes DIMENSIONS via scale,
// but the page CONTENT was already rendered at different widths — elements
// shift horizontally, and the bottom ~50px (where content reflow accumulates)
// flickers visibly. No post-processing filter can fix content that was
// captured in different positions.
//
// The fix: force a permanent scrollbar so the viewport width never changes.
const viewportStabilizationScript = `(function(){` +
	`var s=document.createElement('style');` +
	`s.id='bas-viewport-stabilize';` +
	`s.textContent='html{overflow-y:scroll!important}body{overflow-x:hidden!important}';` +
	`document.head.appendChild(s);` +
	`})();`

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
	if spec == nil || len(spec.GetFrames()) == 0 {
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

	instructions, err := buildPlaywrightCaptureInstructions(c.exportPageURL, spec, captureInterval)
	if err != nil {
		return nil, fmt.Errorf("build capture instructions: %w", err)
	}
	canvas := spec.GetPresentation().GetCanvas()
	viewportWidth, viewportHeight := int(canvas.GetWidth()), int(canvas.GetHeight())
	if viewportWidth <= 0 {
		viewportWidth = 1920
	}
	if viewportHeight <= 0 {
		viewportHeight = 1080
	}
	plan := autocontracts.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions:   instructions,
		CreatedAt:      time.Now().UTC(),
		Metadata:       map[string]any{"executionViewport": map[string]any{"width": viewportWidth, "height": viewportHeight}},
	}

	// Replay screenshots are explicit actions. Incidental evaluate/navigation
	// screenshots double capture work and cannot contribute to the movie.
	artifactConfig := config.ArtifactCollectionSettings{
		ScreenshotPolicy: basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER,
	}
	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        "playwright",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         eventSink,
		HeartbeatInterval: 0,
		ArtifactConfig:    &artifactConfig,
		// The canvas is already in output pixels. Driver-default 2x DPR
		// quadruples pixels and cost without adding source detail.
		BrowserProfile: &sessionprofile.BrowserProfile{Fingerprint: &sessionprofile.FingerprintSettings{DeviceScaleFactor: 1}},
	}

	if err := exec.Execute(ctx, req); err != nil {
		return nil, err
	}

	if len(rec.outcomes) == 0 {
		return nil, fmt.Errorf("playwright capture produced zero outcomes")
	}

	frames := make([]captureFrame, 0, len(rec.outcomes))
	rasterWidth, rasterHeight := 0, 0
	for idx, outcome := range rec.outcomes {
		if !strings.HasPrefix(outcome.NodeID, "screenshot-") || outcome.Screenshot == nil || len(outcome.Screenshot.Data) == 0 {
			continue
		}
		dimensions, _, err := image.DecodeConfig(bytes.NewReader(outcome.Screenshot.Data))
		if err != nil {
			return nil, fmt.Errorf("invalid replay screenshot: %w", err)
		}
		if dimensions.Width != viewportWidth || dimensions.Height != viewportHeight {
			return nil, fmt.Errorf("replay raster %dx%d differs from requested canvas %dx%d", dimensions.Width, dimensions.Height, viewportWidth, viewportHeight)
		}
		rasterWidth, rasterHeight = dimensions.Width, dimensions.Height
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
	fps := int(math.Round(1000.0 / float64(captureInterval)))
	if fps < 1 {
		fps = 1
	}
	return &captureResponse{Frames: frames, FPS: fps, Width: rasterWidth, Height: rasterHeight}, nil
}

func buildPlaywrightCaptureInstructions(exportPageURL string, spec *ReplayMovieSpec, captureInterval int) ([]autocontracts.CompiledInstruction, error) {
	totalMs := int(spec.GetSummary().GetTotalDurationMs())
	if totalMs <= 0 && int(spec.GetPlayback().GetDurationMs()) > 0 {
		totalMs = int(spec.GetPlayback().GetDurationMs())
	}
	if totalMs <= 0 {
		totalMs = captureInterval * len(spec.GetFrames())
	}
	if totalMs <= 0 {
		totalMs = 5000
	}
	if captureInterval <= 0 {
		captureInterval = 1000
	}

	frameCount := int(math.Ceil(float64(totalMs)/float64(captureInterval))) + 1
	if frameCount > maxCaptureFrames {
		return nil, fmt.Errorf("replay requires %d frames; budget is %d: choose a lower FPS or export shorter sections; timeline will not be truncated", frameCount, maxCaptureFrames)
	}

	specJSON, err := protojson.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("serialize replay spec: %w", err)
	}
	// Use the export page's public message/controller contract. The retired
	// bas:render event has no listener and used to capture the error screen.
	script := fmt.Sprintf(`(async()=>{
      const deadline=Date.now()+15000;
      while(!window.basExport || window.basExport.error==='initializing'){if(Date.now()>deadline)throw Error('Replay composer unavailable');await new Promise(r=>setTimeout(r,25));}
      window.postMessage({type:'bas:spec:set',spec:%s},window.location.origin);
      while(!window.basExport.ready){
        const error=window.basExport.error;
        if(error && error!=='initializing' && error!=='Missing replay payload')throw Error(error);
        if(Date.now()>deadline)throw Error('Replay composer did not become ready');
        await new Promise(r=>setTimeout(r,25));
      }
      window.basExport.pause();await window.basExport.seekTo(0);await document.fonts.ready;return {ready:true};
    })()`, string(specJSON))

	// Helper to build action with error handling
	buildAction := func(stepType string, params map[string]any) (*basactions.ActionDefinition, error) {
		action, err := autocompiler.BuildActionDefinition(stepType, params)
		if err != nil {
			return nil, fmt.Errorf("build action %q: %w", stepType, err)
		}
		return action, nil
	}

	// Standalone preview intentionally does not expose the capture controller.
	// Select capture mode while preserving other owner-supplied URL parameters.
	composerURL, err := url.Parse(exportPageURL)
	if err != nil {
		return nil, fmt.Errorf("parse composer URL: %w", err)
	}
	query := composerURL.Query()
	query.Set("mode", "capture")
	composerURL.RawQuery = query.Encode()
	// Build initial navigation and script injection
	navigateAction, err := buildAction("navigate", map[string]any{
		"url":       composerURL.String(),
		"waitUntil": "networkidle",
		"timeoutMs": 45000,
	})
	if err != nil {
		return nil, err
	}

	// Stabilize the viewport layout before any screenshots. Force a permanent
	// scrollbar and hide horizontal overflow so that scrollbar appearance/
	// disappearance during playback cannot change the viewport width. Without
	// this, frames alternate between two widths (~17px scrollbar difference),
	// and even though the assembly filter chain normalizes dimensions, the
	// page CONTENT itself shifts horizontally — causing visible flickering
	// that no post-processing filter can repair.
	stabilizeAction, err := buildAction("evaluate", map[string]any{
		"expression": viewportStabilizationScript,
	})
	if err != nil {
		return nil, err
	}

	evaluateAction, err := buildAction("evaluate", map[string]any{
		"expression": script,
	})
	if err != nil {
		return nil, err
	}

	instr := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "navigate",
			Action: navigateAction,
		},
		{
			Index:  1,
			NodeID: "stabilize-viewport",
			Action: stabilizeAction,
		},
		{
			Index:  2,
			NodeID: "inject",
			Action: evaluateAction,
		},
	}

	// Build wait/screenshot pairs for each frame
	idx := 3
	for i := 0; i < frameCount; i++ {
		targetMs := i * captureInterval
		if targetMs > totalMs {
			targetMs = totalMs
		}
		waitAction, err := buildAction("evaluate", map[string]any{
			"expression": fmt.Sprintf(`(async()=>{if(!window.basExport.ready)throw Error(window.basExport.error||'Replay composer unavailable');await window.basExport.seekTo(%d);await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));
            for(const video of document.querySelectorAll('video[data-bas-source-video]')){
                const deadline=Date.now()+10000;
                while(video.readyState<2||video.seeking){if(video.error)throw Error('Recorded video decode failed');if(Date.now()>deadline)throw Error('Recorded video frame did not settle');await new Promise(r=>setTimeout(r,10));}
            }
            await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));return {ready:true};})()`, targetMs),
		})
		if err != nil {
			return nil, err
		}

		screenshotAction, err := buildAction("screenshot", map[string]any{
			"fullPage": false,
		})
		if err != nil {
			return nil, err
		}

		instr = append(instr, autocontracts.CompiledInstruction{
			Index:  idx,
			NodeID: fmt.Sprintf("wait-%d", i),
			Action: waitAction,
		})
		idx++
		instr = append(instr, autocontracts.CompiledInstruction{
			Index:  idx,
			NodeID: fmt.Sprintf("screenshot-%d", i),
			Action: screenshotAction,
		})
		idx++
	}

	return instr, nil
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

func (r *inMemoryCaptureRecorder) RecordCheckpoint(context.Context, executionwriter.Checkpoint) error {
	return nil // In-memory recorder doesn't persist checkpoints
}

func (r *inMemoryCaptureRecorder) RecordExecutionArtifacts(_ context.Context, _ autocontracts.ExecutionPlan, _ []executionwriter.ExternalArtifact) error {
	return nil // In-memory recorder does not persist artifacts
}

func (r *inMemoryCaptureRecorder) SetArtifactConfig(_ *config.ArtifactCollectionSettings) {
	// In-memory recorder ignores artifact config - collects everything
}

func (r *inMemoryCaptureRecorder) GetArtifactConfig() config.ArtifactCollectionSettings {
	return config.DefaultArtifactSettings() // In-memory recorder uses default (collect all)
}

func (r *inMemoryCaptureRecorder) SetArtifactConfigForExecution(_ uuid.UUID, _ *config.ArtifactCollectionSettings) {
	// In-memory recorder ignores per-execution artifact config - collects everything.
}

func (r *inMemoryCaptureRecorder) ForgetExecution(_ uuid.UUID) {}
