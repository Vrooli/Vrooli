package engine

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/cdp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// BrowserlessEngine is a thin AutomationEngine wrapper around the existing CDP
// session implementation. It intentionally avoids orchestrating retries or
// persistence; those concerns live in the executor/recorder layers.
type BrowserlessEngine struct {
	browserlessURL string
	log            *logrus.Logger
}

// NewBrowserlessEngine creates an engine using environment configuration.
func NewBrowserlessEngine(log *logrus.Logger) (*BrowserlessEngine, error) {
	url := resolveBrowserlessURL(log)
	if url == "" {
		return nil, fmt.Errorf("BROWSERLESS_URL or BROWSERLESS_PORT is required")
	}
	return &BrowserlessEngine{
		browserlessURL: url,
		log:            log,
	}, nil
}

// Name returns the engine identifier.
func (e *BrowserlessEngine) Name() string { return "browserless" }

// Capabilities returns a static descriptor for the Browserless engine.
func (e *BrowserlessEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.EventEnvelopeSchemaVersion,
		Engine:                e.Name(),
		Version:               "v1",
		RequiresDocker:        false,
		RequiresXvfb:          false,
		MaxConcurrentSessions: 4,
		AllowsParallelTabs:    true,
		SupportsHAR:           false,
		SupportsVideo:         false,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     true,
		SupportsTracing:       false,
		MaxViewportWidth:      1920,
		MaxViewportHeight:     1080,
	}, nil
}

// StartSession opens a CDP session with the requested viewport.
func (e *BrowserlessEngine) StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error) {
	width := spec.ViewportWidth
	height := spec.ViewportHeight
	if width == 0 {
		width = 1920
	}
	if height == 0 {
		height = 1080
	}

	sess, err := cdp.NewSession(ctx, e.browserlessURL, width, height, e.log.WithField("execution_id", spec.ExecutionID))
	if err != nil {
		return nil, err
	}
	return &browserlessSession{sess: sess, log: e.log}, nil
}

type browserlessSession struct {
	sess cdpSession
	log  *logrus.Logger
}

type cdpSession interface {
	ExecuteInstruction(ctx context.Context, instruction runtime.Instruction) (*runtime.ExecutionResponse, error)
	CleanBrowserState() error
	Close() error
}

func (s *browserlessSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	start := time.Now().UTC()
	runtimeInstr, err := toRuntimeInstruction(instruction)
	if err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("convert instruction: %w", err)
	}

	resp, err := s.sess.ExecuteInstruction(ctx, runtimeInstr)
	if err != nil {
		return contracts.StepOutcome{}, err
	}
	return buildStepOutcomeFromRuntime(instruction, start, resp)
}

func (s *browserlessSession) Reset(ctx context.Context) error {
	return s.sess.CleanBrowserState()
}

func (s *browserlessSession) Close(ctx context.Context) error {
	return s.sess.Close()
}

// resolveBrowserlessURL mirrors the logic used by the legacy client so engines
// can be constructed independently.
func resolveBrowserlessURL(log *logrus.Logger) string {
	if url := strings.TrimSpace(os.Getenv("BROWSERLESS_URL")); url != "" {
		return strings.TrimRight(url, "/")
	}
	if port := os.Getenv("BROWSERLESS_PORT"); port != "" {
		host := os.Getenv("BROWSERLESS_HOST")
		if host == "" {
			host = "localhost"
		}
		return fmt.Sprintf("http://%s:%s", host, port)
	}
	if log != nil {
		log.Warn("BROWSERLESS_URL not configured")
	}
	return ""
}
