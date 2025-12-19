package engine

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// SessionReuseMode controls how engines treat state between instructions.
type SessionReuseMode string

const (
	ReuseModeFresh SessionReuseMode = "fresh" // Always start a new session.
	ReuseModeClean SessionReuseMode = "clean" // Reuse browser process, reset storage.
	ReuseModeReuse SessionReuseMode = "reuse" // Reuse session as-is when possible.
)

// FrameStreamingConfig configures live frame streaming during execution.
// When enabled, the playwright-driver will push frames to the callback URL,
// allowing the UI to display a live preview of the execution.
type FrameStreamingConfig struct {
	CallbackURL string // HTTP POST endpoint to receive frames
	Quality     int    // JPEG quality 1-100 (default: 55)
	FPS         int    // Target frames per second (default: 6)
	Scale       string // "css" or "device" (default: "css")
}

// SessionSpec describes the session needed for an execution.
type SessionSpec struct {
	ExecutionID    uuid.UUID
	WorkflowID     uuid.UUID
	ViewportWidth  int
	ViewportHeight int
	ReuseMode      SessionReuseMode
	BaseURL        string
	Labels         map[string]string
	Capabilities   contracts.CapabilityRequirement // Required capabilities derived from plan.
	FrameStreaming *FrameStreamingConfig           // Optional: enables live frame streaming during execution.
}

// AutomationEngine exposes engine capabilities and produces engine sessions.
type AutomationEngine interface {
	Name() string
	Capabilities(ctx context.Context) (contracts.EngineCapabilities, error)
	StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error)
}

// EngineSession executes compiled instructions in a single browser context.
type EngineSession interface {
	Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error)
	Reset(ctx context.Context) error
	Close(ctx context.Context) error
}

// Factory resolves engine implementations by name/config.
type Factory interface {
	Resolve(ctx context.Context, name string) (AutomationEngine, error)
}
