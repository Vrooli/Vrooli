package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
)

// OfflineModeAction toggles network isolation for the session's app.
type OfflineModeAction struct{}

func (a *OfflineModeAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Enabled {
		session.SetNetworkMode("offline", 0)
	} else {
		session.SetNetworkMode("normal", 0)
	}

	// Re-launch app if currently running
	session.mu.Lock()
	running := session.AppRunning
	session.mu.Unlock()
	if running {
		svc.killAppProcess(session)
		if err := svc.LaunchApp(session.ID, ""); err != nil {
			svc.logger.Warn("failed to re-launch app after network mode change", "error", err)
		}
	}

	mode := "normal"
	if p.Enabled {
		mode = "offline"
	}
	return &ActionResult{
		Status:  "ok",
		Message: fmt.Sprintf("Network mode set to %s", mode),
		Data:    map[string]any{"network_mode": mode},
	}, nil
}

// SlowConnectionAction enables bandwidth throttling for the session's app.
type SlowConnectionAction struct{}

func (a *SlowConnectionAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Enabled       bool `json:"enabled"`
		BandwidthKbps int  `json:"bandwidth_kbps"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Enabled {
		bw := p.BandwidthKbps
		if bw <= 0 {
			bw = 256
		}
		session.SetNetworkMode("slow", bw)
	} else {
		session.SetNetworkMode("normal", 0)
	}

	// Re-launch app if currently running
	session.mu.Lock()
	running := session.AppRunning
	session.mu.Unlock()
	if running {
		svc.killAppProcess(session)
		if err := svc.LaunchApp(session.ID, ""); err != nil {
			svc.logger.Warn("failed to re-launch app after bandwidth change", "error", err)
		}
	}

	return &ActionResult{
		Status: "ok",
		Data: map[string]any{
			"network_mode":   session.NetworkMode,
			"bandwidth_kbps": session.BandwidthKbps,
		},
	}, nil
}

// InjectEnvAction sets environment variables for the session.
type InjectEnvAction struct{}

func (a *InjectEnvAction) Execute(_ context.Context, session *Session, _ *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Vars  map[string]string `json:"vars"`
		Merge *bool             `json:"merge"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	merge := true
	if p.Merge != nil {
		merge = *p.Merge
	}

	if merge {
		existing := session.GetEnvVars()
		if existing == nil {
			existing = make(map[string]string)
		}
		for k, v := range p.Vars {
			existing[k] = v
		}
		session.SetEnvVars(existing)
	} else {
		session.SetEnvVars(p.Vars)
	}

	return &ActionResult{
		Status: "ok",
		Data:   map[string]any{"env_vars": session.GetEnvVars()},
	}, nil
}

// ResizeDisplayAction changes the session's display resolution.
type ResizeDisplayAction struct{}

func (a *ResizeDisplayAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Width <= 0 || p.Height <= 0 {
		return nil, fmt.Errorf("width and height must be positive")
	}

	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}

	sizeArg := fmt.Sprintf("%dx%d", p.Width, p.Height)
	if _, err := svc.shell(ctx, nil, "xrandr", "--display", session.Display.DisplayID, "-s", sizeArg); err != nil {
		return nil, fmt.Errorf("xrandr resize failed: %w", err)
	}

	session.mu.Lock()
	session.Width = p.Width
	session.Height = p.Height
	session.mu.Unlock()

	return &ActionResult{
		Status:  "ok",
		Message: fmt.Sprintf("Display resized to %dx%d", p.Width, p.Height),
		Data:    map[string]any{"width": p.Width, "height": p.Height},
	}, nil
}
