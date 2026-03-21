package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ClipboardReadAction reads the clipboard contents from the session's display.
type ClipboardReadAction struct{}

func (a *ClipboardReadAction) Execute(ctx context.Context, session *Session, svc *Service, _ json.RawMessage) (*ActionResult, error) {
	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}

	if _, err := exec.LookPath("xclip"); err != nil {
		return &ActionResult{
			Status:  "error",
			Message: "xclip is not installed",
			Data: map[string]any{
				"recovery":      "install_dependency",
				"recovery_hint": "Run: sudo apt-get install -y xclip",
			},
		}, nil
	}

	out, err := svc.shell(ctx, nil, "xclip", "-display", session.Display.DisplayID, "-selection", "clipboard", "-o")
	if err != nil {
		return nil, fmt.Errorf("clipboard read failed: %w", err)
	}

	return &ActionResult{
		Status: "ok",
		Data:   map[string]any{"content": string(out)},
	}, nil
}

// ClipboardWriteAction writes content to the session's clipboard.
type ClipboardWriteAction struct{}

func (a *ClipboardWriteAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}

	var p struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if _, err := exec.LookPath("xclip"); err != nil {
		return &ActionResult{
			Status:  "error",
			Message: "xclip is not installed",
			Data: map[string]any{
				"recovery":      "install_dependency",
				"recovery_hint": "Run: sudo apt-get install -y xclip",
			},
		}, nil
	}

	// Pipe content to xclip via sh -c with echo
	cmd := fmt.Sprintf("echo -n %s | xclip -display %s -selection clipboard -i",
		shellQuote(p.Content), session.Display.DisplayID)
	if _, err := svc.shell(ctx, nil, "sh", "-c", cmd); err != nil {
		return nil, fmt.Errorf("clipboard write failed: %w", err)
	}

	return &ActionResult{
		Status:  "ok",
		Message: "Clipboard updated",
	}, nil
}

// shellQuote wraps a string in single quotes, escaping existing single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// DarkModeAction toggles dark mode for the session's app.
type DarkModeAction struct{}

func (a *DarkModeAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	session.SetDarkMode(p.Enabled)

	// Re-launch app if currently running
	session.mu.Lock()
	running := session.AppRunning
	session.mu.Unlock()
	if running {
		svc.killAppProcess(session)
		if err := svc.LaunchApp(session.ID, ""); err != nil {
			svc.logger.Warn("failed to re-launch app after dark mode change", "error", err)
		}
	}

	return &ActionResult{
		Status:  "ok",
		Message: fmt.Sprintf("Dark mode %s", boolToOnOff(p.Enabled)),
		Data:    map[string]any{"dark_mode": p.Enabled},
	}, nil
}

// LocaleAction sets the locale for the session's app.
type LocaleAction struct{}

func (a *LocaleAction) Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error) {
	var p struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Locale == "" {
		return nil, fmt.Errorf("locale must not be empty")
	}

	session.SetLocale(p.Locale)

	// Re-launch app if currently running
	session.mu.Lock()
	running := session.AppRunning
	session.mu.Unlock()
	if running {
		svc.killAppProcess(session)
		if err := svc.LaunchApp(session.ID, ""); err != nil {
			svc.logger.Warn("failed to re-launch app after locale change", "error", err)
		}
	}

	return &ActionResult{
		Status:  "ok",
		Message: fmt.Sprintf("Locale set to %s", p.Locale),
		Data:    map[string]any{"locale": p.Locale},
	}, nil
}

func boolToOnOff(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
