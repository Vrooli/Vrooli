package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
)

type healthFactory struct {
	ctx      context.Context
	name     string
	resolves int
	engine   *healthEngine
}

func (f *healthFactory) Resolve(ctx context.Context, name string) (engine.AutomationEngine, error) {
	f.ctx = ctx
	f.name = name
	f.resolves++
	return f.engine, nil
}

type healthEngine struct {
	capabilityContext context.Context
	startSessionCalls int
}

func (*healthEngine) Name() string { return "playwright" }
func (e *healthEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	e.capabilityContext = ctx
	if err := ctx.Err(); err != nil {
		return contracts.EngineCapabilities{}, err
	}
	return contracts.EngineCapabilities{
		SchemaVersion: contracts.CapabilitiesSchemaVersion,
		Engine:        "playwright", MaxConcurrentSessions: 1,
	}, nil
}

func (e *healthEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	e.startSessionCalls++
	return nil, nil
}

func TestCheckAutomationHealth_UsesInjectedFactoryAndRequestContext(t *testing.T) {
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "http://driver.test")
	t.Setenv("ENGINE", "playwright")
	t.Setenv("ENGINE_OVERRIDE", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eng := &healthEngine{}
	factory := &healthFactory{engine: eng}
	service := &WorkflowService{engineFactory: factory}

	ok, err := service.CheckAutomationHealth(ctx)
	if err != nil || !ok {
		t.Fatalf("health = (%v, %v), want (true, nil)", ok, err)
	}
	if factory.resolves != 1 || factory.name != "playwright" || factory.ctx != ctx {
		t.Fatalf("factory resolution = count %d, name %q, same context %v", factory.resolves, factory.name, factory.ctx == ctx)
	}
	if eng.capabilityContext != ctx {
		t.Fatal("engine health probe did not receive the caller context")
	}
	if eng.startSessionCalls != 0 {
		t.Fatalf("health check created %d browser sessions", eng.startSessionCalls)
	}
}

func TestCheckAutomationHealth_CancellationPropagatesWithoutStartingSession(t *testing.T) {
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "http://driver.test")
	t.Setenv("ENGINE", "playwright")
	t.Setenv("ENGINE_OVERRIDE", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	eng := &healthEngine{}
	factory := &healthFactory{engine: eng}
	service := &WorkflowService{engineFactory: factory}

	ok, err := service.CheckAutomationHealth(ctx)
	if ok || err != context.Canceled {
		t.Fatalf("health = (%v, %v), want (false, context.Canceled)", ok, err)
	}
	if factory.ctx != ctx || eng.capabilityContext != ctx {
		t.Fatal("canceled caller context was not preserved through factory and engine")
	}
	if eng.startSessionCalls != 0 {
		t.Fatalf("canceled health check created %d browser sessions", eng.startSessionCalls)
	}
}

func TestCheckAutomationHealth_RequiresCanonicalFactory(t *testing.T) {
	t.Setenv("PLAYWRIGHT_DRIVER_URL", "http://driver.test")
	t.Setenv("ENGINE", "playwright")
	t.Setenv("ENGINE_OVERRIDE", "")
	service := &WorkflowService{}

	ok, err := service.CheckAutomationHealth(context.Background())
	if ok || err == nil || !strings.Contains(err.Error(), "canonical automation engine factory") {
		t.Fatalf("health = (%v, %v), want missing canonical factory error", ok, err)
	}
}
