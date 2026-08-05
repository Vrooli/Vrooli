package smoketest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/procmetrics"
)

// JourneyStep is the durable, human-readable record of one desktop action.
type JourneyStep struct {
	Name            string                      `json:"name"`
	Action          string                      `json:"action"`
	Disposition     string                      `json:"disposition"`
	BeforeCaptureID string                      `json:"before_capture_id,omitempty"`
	AfterCaptureID  string                      `json:"after_capture_id,omitempty"`
	Geometry        *procmetrics.WindowGeometry `json:"geometry,omitempty"`
	Error           string                      `json:"error,omitempty"`
	DegradedReason  string                      `json:"degraded_reason,omitempty"`
	StartedAt       time.Time                   `json:"started_at"`
	CompletedAt     time.Time                   `json:"completed_at"`
}

// JourneyResult is stored as a journey capture beside the video and images.
type JourneyResult struct {
	SchemaVersion                int           `json:"schema_version"`
	SmokeTestID                  string        `json:"smoke_test_id"`
	ScenarioName                 string        `json:"scenario_name"`
	Platform                     string        `json:"platform"`
	Display                      string        `json:"display"`
	WindowManager                string        `json:"window_manager,omitempty"`
	Titlebar                     bool          `json:"titlebar"`
	RecordingStartedBeforeLaunch bool          `json:"recording_started_before_launch"`
	Disposition                  string        `json:"disposition"`
	DegradedReason               string        `json:"degraded_reason,omitempty"`
	Steps                        []JourneyStep `json:"steps"`
	CreatedAt                    time.Time     `json:"created_at"`
}

const (
	journeyPass           = "pass"
	journeyDegraded       = "degraded"
	journeyFailed         = "failed"
	journeyStepPass       = "passed"
	journeyStepFail       = "failed"
	journeyStepDegraded   = "degraded"
	journeyStepTimeout    = 10 * time.Second
	journeyCaptureTimeout = 5 * time.Second
)

func (s *DefaultService) runDesktopJourney(ctx context.Context, smokeTestID, scenarioName, platform string, rec recordingState) JourneyResult {
	journey := JourneyResult{
		SchemaVersion:                1,
		SmokeTestID:                  smokeTestID,
		ScenarioName:                 scenarioName,
		Platform:                     platform,
		Display:                      rec.displayID,
		WindowManager:                rec.windowManager,
		Titlebar:                     rec.titlebar,
		RecordingStartedBeforeLaunch: rec.captureID != "",
		Disposition:                  journeyPass,
		CreatedAt:                    time.Now().UTC(),
	}

	degrade := func(reason string) JourneyResult {
		journey.Disposition = journeyDegraded
		journey.DegradedReason = reason
		return journey
	}
	if runtime.GOOS != "linux" {
		return degrade("platform_not_linux")
	}
	if rec.windowManager == "" {
		return degrade("window_manager_not_started")
	}
	if !rec.titlebar {
		return degrade("window_manager_titlebar_unavailable")
	}
	if s.windowDetector == nil || !s.windowDetector.IsAvailable(ctx) {
		return degrade("xdotool_unavailable")
	}

	visibleCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	for {
		visible, _ := s.windowDetector.HasVisibleWindow(visibleCtx, 0, rec.displayID)
		if visible {
			break
		}
		select {
		case <-visibleCtx.Done():
			return degrade("no_visible_window")
		case <-time.After(100 * time.Millisecond):
		}
	}

	addInteraction := func(name, action string, perform func(context.Context) error, geometry bool) {
		step := JourneyStep{Name: name, Action: action, Disposition: journeyStepPass, StartedAt: time.Now().UTC()}
		beforeCtx, beforeCancel := context.WithTimeout(ctx, journeyCaptureTimeout)
		beforeID, err := s.captureJourneyScreenshot(beforeCtx, smokeTestID, scenarioName, rec.displayID, "before-"+name)
		beforeCancel()
		if err != nil {
			step.Disposition = journeyStepFail
			step.Error = "before screenshot: " + err.Error()
			journey.Disposition = journeyFailed
			journey.Steps = append(journey.Steps, step)
			return
		}
		step.BeforeCaptureID = beforeID

		stepCtx, stepCancel := context.WithTimeout(ctx, journeyStepTimeout)
		err = perform(stepCtx)
		if err != nil {
			step.Disposition = journeyStepFail
			step.Error = err.Error()
			journey.Disposition = journeyFailed
			if stepCtx.Err() != nil {
				journey.DegradedReason = "step_timeout_" + name
			}
		} else if geometry {
			step.Geometry, err = s.windowDetector.WindowGeometry(stepCtx, 0, rec.displayID)
			if err != nil {
				step.Disposition = journeyStepFail
				step.Error = "geometry: " + err.Error()
				journey.Disposition = journeyFailed
			}
		}
		stepCancel()

		afterCtx, afterCancel := context.WithTimeout(ctx, journeyCaptureTimeout)
		afterID, captureErr := s.captureJourneyScreenshot(afterCtx, smokeTestID, scenarioName, rec.displayID, "after-"+name)
		afterCancel()
		if captureErr != nil {
			step.Disposition = journeyStepFail
			if step.Error == "" {
				step.Error = "after screenshot: " + captureErr.Error()
			} else {
				step.Error += "; after screenshot: " + captureErr.Error()
			}
			journey.Disposition = journeyFailed
		} else {
			step.AfterCaptureID = afterID
		}
		step.CompletedAt = time.Now().UTC()
		journey.Steps = append(journey.Steps, step)
	}

	detector := s.windowDetector
	addInteraction("activate", "window_activate", func(stepCtx context.Context) error { return detector.ActivateWindow(stepCtx, 0, rec.displayID) }, true)
	addInteraction("maximize", "window_maximize", func(stepCtx context.Context) error {
		return detector.MaximizeWindow(stepCtx, 0, rec.displayID, rec.displayWidth, rec.displayHeight)
	}, true)
	if step := journey.Steps[len(journey.Steps)-1]; step.Geometry == nil || step.Geometry.Width < rec.displayWidth*90/100 || step.Geometry.Height < rec.displayHeight*90/100 {
		journey.Disposition = journeyFailed
		journey.DegradedReason = "maximize_below_ninety_percent"
	}
	addInteraction("pointer_click", "pointer_click", func(stepCtx context.Context) error {
		return detector.Click(stepCtx, rec.displayID, rec.displayWidth/2, rec.displayHeight/2, 1)
	}, false)
	addInteraction("key_press", "key_press", func(stepCtx context.Context) error { return detector.KeyPress(stepCtx, rec.displayID, "Return") }, false)
	addInteraction("resize", "window_resize", func(stepCtx context.Context) error {
		return detector.ResizeWindow(stepCtx, 0, rec.displayID, rec.displayWidth*80/100, rec.displayHeight*80/100)
	}, true)
	addInteraction("move", "window_move", func(stepCtx context.Context) error { return detector.MoveWindow(stepCtx, 0, rec.displayID, 32, 32) }, true)
	addInteraction("quit", "quit_app", func(stepCtx context.Context) error { return detector.KeyPress(stepCtx, rec.displayID, "alt+F4") }, false)
	for _, step := range journey.Steps {
		if step.Disposition != journeyStepPass {
			journey.Disposition = journeyFailed
		}
	}
	if !journeyHasScreenshotPairs(journey.Steps) {
		journey.Disposition = journeyFailed
		journey.DegradedReason = "interaction_screenshot_pair_missing"
	}
	return journey
}

func journeyHasScreenshotPairs(steps []JourneyStep) bool {
	for _, step := range steps {
		if step.Action == "screenshot" || step.Action == "window_geometry" {
			continue
		}
		if step.BeforeCaptureID == "" || step.AfterCaptureID == "" {
			return false
		}
	}
	return len(steps) > 0
}

func (s *DefaultService) captureJourneyScreenshot(ctx context.Context, smokeTestID, scenarioName, display, label string) (string, error) {
	if s.captures == nil {
		return "", fmt.Errorf("capture storage unavailable")
	}
	tmpDir, err := os.MkdirTemp("", "vrooli-journey-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	path := filepath.Join(tmpDir, label+".png")
	command := `xwd -display "$DISPLAY" -root -silent | ffmpeg -y -f xwd_pipe -i - -frames:v 1 -update 1 "$OUTPUT"`
	result, err := runJourneyCommand(ctx, "sh", []string{"-c", command}, []string{"DISPLAY=" + display, "OUTPUT=" + path})
	if err != nil || result != nil && result.ExitCode != 0 {
		if err == nil {
			err = fmt.Errorf("screenshot command exited with code %d", result.ExitCode)
		}
		return "", err
	}
	cap, err := s.captures.SaveCapture(scenarioName, captures.CaptureScreenshot, "smoke-test:"+smokeTestID, path, 0, 0, 0)
	if err != nil {
		return "", err
	}
	return cap.ID, nil
}

func runJourneyCommand(ctx context.Context, command string, args []string, env []string) (*ExecutionResult, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("journey command args are required")
	}
	process := execCommandContext(ctx, command, args...)
	process.Env = append(os.Environ(), env...)
	output, err := process.CombinedOutput()
	result := &ExecutionResult{Combined: string(output), ExitCode: 0}
	if err != nil {
		result.ExitCode = 1
	}
	return result, err
}

// execCommandContext is a variable to keep the journey's host seam replaceable
// in focused tests without making production code depend on a shell mock.
var execCommandContext = func(ctx context.Context, command string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, command, args...)
}

func (s *DefaultService) persistJourney(journey JourneyResult) string {
	if s.captures == nil {
		return ""
	}
	data, err := json.MarshalIndent(journey, "", "  ")
	if err != nil {
		return ""
	}
	tmp, err := os.CreateTemp("", "vrooli-journey-*.json")
	if err != nil {
		return ""
	}
	path := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return ""
	}
	_ = tmp.Close()
	defer os.Remove(path)
	cap, err := s.captures.SaveCapture(journey.ScenarioName, captures.CaptureJourney, "smoke-test:"+journey.SmokeTestID, path, 0, 0, 0)
	if err != nil {
		return ""
	}
	return cap.ID
}
