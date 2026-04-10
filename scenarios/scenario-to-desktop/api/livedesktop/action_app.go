package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/screenrecording"
)

// LaunchAppAction launches an application on the session's display.
type LaunchAppAction struct{}

func (a *LaunchAppAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		AppPath string `json:"app_path"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}
	if err := svc.LaunchApp(session.ID, p.AppPath); err != nil {
		return nil, err
	}
	return &ActionResult{
		Status:  "ok",
		Message: "App launched",
	}, nil
}

// QuitAppAction kills the running application.
type QuitAppAction struct{}

func (a *QuitAppAction) Execute(_ context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	session.mu.Lock()
	running := session.AppRunning
	session.mu.Unlock()

	if !running {
		return nil, fmt.Errorf("no app is running")
	}
	svc.killAppProcess(session)
	return &ActionResult{
		Status:  "ok",
		Message: "App stopped",
	}, nil
}

// ScreenshotAction captures a screenshot of the session's display.
type ScreenshotAction struct{}

func (a *ScreenshotAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}

	sessionDir := filepath.Join(svc.dataDir, "sessions", session.ID)
	filename := fmt.Sprintf("screenshot-%d.png", time.Now().UnixMilli())
	outputPath := filepath.Join(sessionDir, filename)

	// Ensure session directory exists
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	// Capture screenshot through platform backend
	if err := svc.backend.CaptureScreenshot(ctx, session.Display, outputPath); err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	// Persist to captures service if available
	if svc.captures != nil {
		cap, err := svc.captures.SaveCapture(session.ScenarioName, captures.CaptureScreenshot, session.ID, outputPath, session.Width, session.Height, 0)
		if err == nil {
			url := fmt.Sprintf("/api/v1/captures/%s/%s/file", session.ScenarioName, cap.ID)
			return &ActionResult{
				Status: "ok",
				Data: map[string]any{
					"url":        url,
					"filename":   cap.Filename,
					"capture_id": cap.ID,
				},
			}, nil
		}
		// Fall through to session-scoped URL on error
	}

	url := fmt.Sprintf("/api/v1/livedesktop/sessions/%s/files/%s", session.ID, filename)
	return &ActionResult{
		Status: "ok",
		Data: map[string]any{
			"url":      url,
			"filename": filename,
		},
	}, nil
}

// StartRecordingAction begins screen recording.
type StartRecordingAction struct{}

func (a *StartRecordingAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	if svc.recorder == nil {
		return nil, fmt.Errorf("screen recording not available")
	}
	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}

	session.mu.Lock()
	recording := session.IsRecording
	session.mu.Unlock()
	if recording {
		return nil, fmt.Errorf("already recording")
	}

	sessionDir := filepath.Join(svc.dataDir, "sessions", session.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	outputPath := filepath.Join(sessionDir, fmt.Sprintf("recording-%d.mp4", time.Now().UnixMilli()))
	captureID, err := svc.recorder.StartCapture(ctx, screenrecording.CaptureConfig{
		Display:    session.Display.DisplayID(),
		Width:      session.Width,
		Height:     session.Height,
		FPS:        15,
		OutputPath: outputPath,
	})
	if err != nil {
		return nil, fmt.Errorf("starting recording: %w", err)
	}

	session.SetRecording(true, captureID)
	return &ActionResult{
		Status:  "ok",
		Message: "Recording started",
		Data:    map[string]any{"capture_id": captureID},
	}, nil
}

// StopRecordingAction stops screen recording.
type StopRecordingAction struct{}

func (a *StopRecordingAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	if svc.recorder == nil {
		return nil, fmt.Errorf("screen recording not available")
	}

	session.mu.Lock()
	recording := session.IsRecording
	captureID := session.CaptureID
	session.mu.Unlock()
	if !recording {
		return nil, fmt.Errorf("not recording")
	}

	result, err := svc.recorder.StopCapture(ctx, captureID)
	if err != nil {
		return nil, fmt.Errorf("stopping recording: %w", err)
	}

	session.SetRecording(false, "")

	// Persist to captures service if available
	if svc.captures != nil {
		cap, err := svc.captures.SaveCapture(session.ScenarioName, captures.CaptureRecording, session.ID, result.VideoPath, session.Width, session.Height, result.DurationMs)
		if err == nil {
			videoURL := fmt.Sprintf("/api/v1/captures/%s/%s/file", session.ScenarioName, cap.ID)
			return &ActionResult{
				Status: "ok",
				Data: map[string]any{
					"video_url":   videoURL,
					"duration_ms": result.DurationMs,
					"size_bytes":  result.FileSizeBytes,
					"capture_id":  cap.ID,
				},
			}, nil
		}
		// Fall through to session-scoped URL on error
	}

	videoFilename := filepath.Base(result.VideoPath)
	videoURL := fmt.Sprintf("/api/v1/livedesktop/sessions/%s/files/%s", session.ID, videoFilename)
	return &ActionResult{
		Status: "ok",
		Data: map[string]any{
			"video_url":   videoURL,
			"duration_ms": result.DurationMs,
			"size_bytes":  result.FileSizeBytes,
		},
	}, nil
}
