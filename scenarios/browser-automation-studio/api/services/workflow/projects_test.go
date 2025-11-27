package workflow

import (
	"context"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
)

// Validate playright health guard when ENGINE=playwright but driver URL is missing.
func TestCheckAutomationHealth_PlaywrightMissingDriver(t *testing.T) {
	t.Setenv("ENGINE", "playwright")
	t.Setenv("ENGINE_OVERRIDE", "")
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "")

	svc := &WorkflowService{
		engineFactory: nil,
	}
	// use empty repo/hub; guard should trip before factory use.
	ok, err := svc.CheckAutomationHealth(context.Background())
	if err == nil || ok {
		t.Fatalf("expected failure when PLAYWRIGHT_DRIVER_URL missing, got ok=%v err=%v", ok, err)
	}
}

// Ensure health passes when not in playwright mode.
func TestCheckAutomationHealth_NonPlaywright(t *testing.T) {
	t.Setenv("ENGINE", "browserless")
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "")

	svc := &WorkflowService{
		engineFactory: &fakeEngineFactory{},
	}
	ok, err := svc.CheckAutomationHealth(context.Background())
	if err != nil || !ok {
		t.Fatalf("expected ok health, got ok=%v err=%v", ok, err)
	}
}

type fakeEngineFactory struct{}

func (f *fakeEngineFactory) Resolve(ctx context.Context, name string) (autoengine.AutomationEngine, error) {
	return &noopEngine{}, nil
}

type noopEngine struct{}

func (n *noopEngine) Name() string { return "noop" }
func (n *noopEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                "noop",
		Version:               "v1",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    false,
		SupportsHAR:           false,
		SupportsVideo:         false,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     true,
		SupportsTracing:       false,
	}, nil
}
func (n *noopEngine) StartSession(ctx context.Context, spec autoengine.SessionSpec) (autoengine.EngineSession, error) {
	return nil, nil
}
