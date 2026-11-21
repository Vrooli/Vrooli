package engine

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// AndroidWebViewEngine reuses the Playwright/CDP session implementation while
// enforcing the Android ramp's attach invariants at the engine boundary.
// Device discovery, port forwarding, recording, and app lifecycle remain
// owned by scenario-to-android/device-control.
type AndroidWebViewEngine struct {
	playwright *PlaywrightEngine
}

func NewAndroidWebViewEngine(playwright *PlaywrightEngine) (*AndroidWebViewEngine, error) {
	if playwright == nil {
		return nil, errors.New("playwright engine is required")
	}
	return &AndroidWebViewEngine{playwright: playwright}, nil
}

func (e *AndroidWebViewEngine) Name() string { return "android-webview" }

func (e *AndroidWebViewEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	if e == nil || e.playwright == nil {
		return contracts.EngineCapabilities{}, errors.New("Android WebView engine is not configured")
	}
	capabilities, err := e.playwright.Capabilities(ctx)
	if err != nil {
		return contracts.EngineCapabilities{}, err
	}
	capabilities.Engine = e.Name()
	capabilities.SupportsVideo = false
	capabilities.SupportsHAR = false
	capabilities.SupportsTracing = false
	capabilities.SupportsPerfTrace = false
	capabilities.Notes = "CDP attach over a device-control port forward; device recording is owned by the Android ramp"
	return capabilities, nil
}

func (e *AndroidWebViewEngine) StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error) {
	if e == nil || e.playwright == nil {
		return nil, errors.New("Android WebView engine is not configured")
	}
	if spec.AppTarget == nil || !strings.EqualFold(spec.AppTarget.TargetKind, driver.TargetKindAndroidWebView) {
		return nil, errors.New("Android WebView engine requires an android-webview AppTarget")
	}
	if spec.ValidationContext == nil || strings.TrimSpace(spec.ValidationContext.IsolationLeaseID) == "" {
		return nil, errors.New("Android WebView engine requires a validation context with an isolation lease")
	}
	session, err := e.playwright.StartSession(ctx, spec)
	if err != nil {
		return nil, err
	}
	start := time.Now().UTC()
	if spec.RecordingStartAt != nil {
		start = spec.RecordingStartAt.UTC()
	}
	return &androidWebViewSession{EngineSession: session, recordingStart: start, recordingID: spec.RecordingID}, nil
}

type androidWebViewSession struct {
	EngineSession
	recordingStart time.Time
	recordingID    string
}

func (s *androidWebViewSession) CreatePage(ctx context.Context, url string) (*driver.CreatePageResponse, error) {
	pageSession, ok := s.EngineSession.(interface {
		CreatePage(context.Context, string) (*driver.CreatePageResponse, error)
	})
	if !ok {
		return nil, errors.New("Android WebView session does not support page restoration")
	}
	return pageSession.CreatePage(ctx, url)
}

func (s *androidWebViewSession) CreatePopupPage(ctx context.Context, url, openerPageID string) (*driver.CreatePageResponse, error) {
	pageSession, ok := s.EngineSession.(interface {
		CreatePopupPage(context.Context, string, string) (*driver.CreatePageResponse, error)
	})
	if !ok {
		return nil, errors.New("Android WebView session does not support popup restoration")
	}
	return pageSession.CreatePopupPage(ctx, url, openerPageID)
}

func (s *androidWebViewSession) SetActivePage(ctx context.Context, pageID string) error {
	pageSession, ok := s.EngineSession.(interface {
		SetActivePage(context.Context, string) error
	})
	if !ok {
		return errors.New("Android WebView session does not support page selection")
	}
	return pageSession.SetActivePage(ctx, pageID)
}

func (s *androidWebViewSession) InitialDriverPageID() string {
	pageSession, ok := s.EngineSession.(interface{ InitialDriverPageID() string })
	if !ok {
		return ""
	}
	return pageSession.InitialDriverPageID()
}

func (s *androidWebViewSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	outcome, err := s.EngineSession.Run(ctx, instruction)
	offset := time.Since(s.recordingStart)
	if offset < 0 {
		offset = 0
	}
	value := offset.Milliseconds()
	outcome.RecordingOffsetMs = &value
	outcome.RecordingID = s.recordingID
	return outcome, err
}

var _ AutomationEngine = (*AndroidWebViewEngine)(nil)
