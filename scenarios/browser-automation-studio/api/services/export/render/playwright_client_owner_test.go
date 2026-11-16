package render

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

type captureFactory struct {
	mu      sync.Mutex
	engine  *captureEngine
	resolve int
}

func (f *captureFactory) Resolve(context.Context, string) (autoengine.AutomationEngine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolve++
	return f.engine, nil
}

type captureEngine struct {
	mu      sync.Mutex
	specs   []autoengine.SessionSpec
	session *captureSession
}

func (*captureEngine) Name() string { return "playwright" }

func (*captureEngine) Capabilities(context.Context) (autocontracts.EngineCapabilities, error) {
	return autocontracts.EngineCapabilities{
		SchemaVersion: autocontracts.CapabilitiesSchemaVersion,
		Engine:        "playwright", MaxConcurrentSessions: 1,
		MaxViewportWidth: 4096, MaxViewportHeight: 4096,
	}, nil
}

func (e *captureEngine) StartSession(_ context.Context, spec autoengine.SessionSpec) (autoengine.EngineSession, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.specs = append(e.specs, spec)
	return e.session, nil
}

type captureSession struct {
	mu          sync.Mutex
	started     chan struct{}
	block       bool
	closed      int
	closeCtxErr error
	png         []byte
}

func (s *captureSession) Run(ctx context.Context, instruction autocontracts.CompiledInstruction) (autocontracts.StepOutcome, error) {
	s.mu.Lock()
	started := s.started
	block := s.block
	s.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if block {
		<-ctx.Done()
		return autocontracts.StepOutcome{}, ctx.Err()
	}
	outcome := autocontracts.StepOutcome{
		NodeID: instruction.NodeID, Success: true, StartedAt: time.Now().UTC(),
	}
	if instruction.NodeID[:min(len(instruction.NodeID), 10)] == "screenshot" {
		outcome.Screenshot = &autocontracts.Screenshot{Data: s.png, Width: 32, Height: 24}
	}
	return outcome, nil
}

func (*captureSession) Reset(context.Context) error { return nil }
func (s *captureSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed++
	s.closeCtxErr = ctx.Err()
	return nil
}
func (*captureSession) GetStorageState(context.Context) (json.RawMessage, error) { return nil, nil }

func captureTestFixture(t *testing.T, block bool) (*playwrightCaptureClient, *captureFactory, *captureSession) {
	t.Helper()
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "http://driver.test")
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 32, 24))); err != nil {
		t.Fatal(err)
	}
	session := &captureSession{started: make(chan struct{}, 1), block: block, png: imageBytes.Bytes()}
	engine := &captureEngine{session: session}
	factory := &captureFactory{engine: engine}
	client := newPlaywrightCaptureClient("http://export.test/composer", factory).(*playwrightCaptureClient)
	return client, factory, session
}

func captureTestSpec() *ReplayMovieSpec {
	return &ReplayMovieSpec{
		Execution:    &exportsv1.ReplayExecutionMetadata{ExecutionId: uuid.NewString(), WorkflowId: uuid.NewString()},
		Frames:       []*exportsv1.ReplayFrame{{DurationMs: 100}},
		Summary:      &exportsv1.ReplaySummary{TotalDurationMs: 100},
		Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 32, Height: 24}},
		Playback:     &exportsv1.ReplayPlayback{FrameIntervalMs: 100},
	}
}

func TestPlaywrightCapture_UsesInjectedFactoryAndIsolatesExecutionIdentity(t *testing.T) {
	client, factory, session := captureTestFixture(t, false)
	for i := 0; i < 2; i++ {
		result, err := client.Capture(context.Background(), captureTestSpec(), 100)
		if err != nil {
			t.Fatalf("capture %d: %v", i, err)
		}
		if len(result.Frames) == 0 {
			t.Fatalf("capture %d returned no frames", i)
		}
	}
	if factory.resolve != 2 {
		t.Fatalf("injected factory resolve count = %d, want one per capture", factory.resolve)
	}
	engine := factory.engine
	if len(engine.specs) != 2 {
		t.Fatalf("session start count = %d, want 2", len(engine.specs))
	}
	if engine.specs[0].ExecutionID == engine.specs[1].ExecutionID || engine.specs[0].WorkflowID == engine.specs[1].WorkflowID {
		t.Fatal("separate movie captures reused execution or workflow identity")
	}
	if session.closed != 2 {
		t.Fatalf("closed sessions = %d, want 2", session.closed)
	}
}

func TestPlaywrightCapture_CancellationClosesSession(t *testing.T) {
	client, factory, session := captureTestFixture(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := client.Capture(ctx, captureTestSpec(), 100); done <- err }()
	select {
	case <-session.started:
	case <-time.After(2 * time.Second):
		t.Fatal("capture did not reach engine session")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled capture returned nil error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("capture did not return after cancellation")
	}
	if factory.resolve != 1 {
		t.Fatalf("factory resolve count = %d, want 1", factory.resolve)
	}
	if session.closed != 1 {
		t.Fatalf("closed sessions = %d, want 1", session.closed)
	}
	if session.closeCtxErr != nil {
		t.Fatalf("session cleanup inherited cancellation: %v", session.closeCtxErr)
	}
}
