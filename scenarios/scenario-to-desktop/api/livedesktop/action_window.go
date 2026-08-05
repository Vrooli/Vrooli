package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"

	"scenario-to-desktop-api/procmetrics"
)

type windowParams struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Button int    `json:"button"`
	Key    string `json:"key"`
}

func decodeWindowParams(params json.RawMessage) (windowParams, error) {
	var p windowParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return p, fmt.Errorf("invalid window action params: %w", err)
		}
	}
	return p, nil
}

func windowController(svc *Service) (*procmetrics.XdotoolDetector, error) {
	if svc.windowController == nil {
		return nil, fmt.Errorf("xdotool window control is not configured")
	}
	return svc.windowController, nil
}

func appPID(session *Session) int {
	if session.AppProcess == nil {
		return 0
	}
	return session.AppProcess.PID()
}

func windowDisplay(session *Session) (string, error) {
	if session.Display == nil || !session.Display.IsRunning() {
		return "", fmt.Errorf("session display is not running")
	}
	return session.Display.DisplayID(), nil
}

type WindowActivateAction struct{}

func (a *WindowActivateAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.ActivateWindow(ctx, appPID(session), display); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Window activated"}, nil
}

type WindowMaximizeAction struct{}

func (a *WindowMaximizeAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.MaximizeWindow(ctx, appPID(session), display, session.Width, session.Height); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Window maximized", Data: map[string]any{"width": session.Width, "height": session.Height}}, nil
}

type WindowResizeAction struct{}

func (a *WindowResizeAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	p, err := decodeWindowParams(params)
	if err != nil || p.Width <= 0 || p.Height <= 0 {
		if err == nil {
			err = fmt.Errorf("width and height must be positive")
		}
		return nil, err
	}
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.ResizeWindow(ctx, appPID(session), display, p.Width, p.Height); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Window resized", Data: map[string]any{"width": p.Width, "height": p.Height}}, nil
}

type WindowMoveAction struct{}

func (a *WindowMoveAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	p, err := decodeWindowParams(params)
	if err != nil {
		return nil, err
	}
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.MoveWindow(ctx, appPID(session), display, p.X, p.Y); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Window moved", Data: map[string]any{"x": p.X, "y": p.Y}}, nil
}

type PointerClickAction struct{}

func (a *PointerClickAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	p, err := decodeWindowParams(params)
	if err != nil {
		return nil, err
	}
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.Click(ctx, display, p.X, p.Y, p.Button); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Pointer click sent", Data: map[string]any{"x": p.X, "y": p.Y, "button": p.Button}}, nil
}

type KeyPressAction struct{}

func (a *KeyPressAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	p, err := decodeWindowParams(params)
	if err != nil {
		return nil, err
	}
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	if err := detector.KeyPress(ctx, display, p.Key); err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Key press sent", Data: map[string]any{"key": p.Key}}, nil
}

type WindowGeometryAction struct{}

func (a *WindowGeometryAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	detector, err := windowController(svc)
	if err != nil {
		return nil, err
	}
	display, err := windowDisplay(session)
	if err != nil {
		return nil, err
	}
	geometry, err := detector.WindowGeometry(ctx, appPID(session), display)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Status: "ok", Message: "Window geometry read", Data: map[string]any{"x": geometry.X, "y": geometry.Y, "width": geometry.Width, "height": geometry.Height}}, nil
}
