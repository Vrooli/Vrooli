package scenarioport

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordingScenarioCLI struct {
	ports    map[string]map[string]int
	errors   map[string]error
	attempts []lookupAttempt
	paths    []string
}

type lookupAttempt struct {
	scenario string
	port     string
}

func (c *recordingScenarioCLI) LookupPort(_ context.Context, scenarioName, portName string) (int, error) {
	c.attempts = append(c.attempts, lookupAttempt{scenario: scenarioName, port: portName})
	if err, ok := c.errors[portName]; ok && err != nil {
		return 0, err
	}
	if ports, ok := c.ports[scenarioName]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, errors.New("port not found")
}

func (c *recordingScenarioCLI) LookupPortAtPath(ctx context.Context, scenarioName, portName, path string) (int, error) {
	c.paths = append(c.paths, path)
	return c.LookupPort(ctx, scenarioName, portName)
}

func (c *recordingScenarioCLI) ListScenarios(_ context.Context) ([]ScenarioMetadata, error) {
	return nil, nil
}

func (c *recordingScenarioCLI) GetStatus(_ context.Context, _ string) (string, error) {
	return "unknown", nil
}

type retryScenarioCLI struct {
	attempts int
}

func (c *retryScenarioCLI) LookupPort(_ context.Context, scenarioName, portName string) (int, error) {
	c.attempts++
	if c.attempts < 3 {
		return 0, errors.New("transient cli failure")
	}
	if scenarioName != "web-console" || portName != "UI_PORT" {
		return 0, errors.New("unexpected lookup")
	}
	return 36233, nil
}

func (c *retryScenarioCLI) LookupPortAtPath(ctx context.Context, scenarioName, portName, _ string) (int, error) {
	return c.LookupPort(ctx, scenarioName, portName)
}

func (c *retryScenarioCLI) ListScenarios(_ context.Context) ([]ScenarioMetadata, error) {
	return nil, nil
}

func (c *retryScenarioCLI) GetStatus(_ context.Context, _ string) (string, error) {
	return "unknown", nil
}

type recordingPortResolver struct {
	scenario string
	port     string
}

func (r *recordingPortResolver) ResolveScenarioPort(_ context.Context, scenarioName, portName string) (int, error) {
	r.scenario = scenarioName
	r.port = portName
	return 49152, nil
}

func TestResolveURLUsesScenarioCLIEvenWhenRegistryEnvIsSet(t *testing.T) {
	t.Setenv("SCENARIO_REGISTRY", `{"browser-automation-studio":{"url":"http://127.0.0.1:6000","ports":{"UI_PORT":6000}}}`)

	cli := &recordingScenarioCLI{
		ports: map[string]map[string]int{
			"browser-automation-studio": {"UI_PORT": 6123},
		},
	}
	restore := SetScenarioCLIForTests(cli)
	defer restore()

	resolved, info, err := ResolveURL(context.Background(), "browser-automation-studio", "/dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "http://localhost:6123/dashboard" {
		t.Fatalf("unexpected URL: %s", resolved)
	}
	if info == nil || info.Name != "UI_PORT" || info.Port != 6123 {
		t.Fatalf("expected port info from scenario CLI, got %+v", info)
	}
	if got, want := cli.attempts, []lookupAttempt{{scenario: "browser-automation-studio", port: "UI_PORT"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lookup attempts: got %+v want %+v", got, want)
	}
}

func TestResolvePortFallsBackThroughScenarioCLI(t *testing.T) {
	t.Setenv("SCENARIO_REGISTRY", `[{"name":"app-monitor","ports":{"API_PORT":7777}}]`)

	cli := &recordingScenarioCLI{
		ports: map[string]map[string]int{
			"app-monitor": {"API_PORT": 8888},
		},
		errors: map[string]error{
			"UI_PORT": errors.New("ui not available"),
		},
	}
	restore := SetScenarioCLIForTests(cli)
	defer restore()

	info, err := ResolvePort(context.Background(), "app-monitor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil || info.Name != "API_PORT" || info.Port != 8888 {
		t.Fatalf("expected API_PORT from scenario CLI, got %+v", info)
	}
	if got, want := cli.attempts, []lookupAttempt{
		{scenario: "app-monitor", port: "UI_PORT"},
		{scenario: "app-monitor", port: "API_PORT"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lookup attempts: got %+v want %+v", got, want)
	}
}

func TestResolveURLAtPathUsesPhysicalScenarioDirectory(t *testing.T) {
	cli := &recordingScenarioCLI{ports: map[string]map[string]int{"generated": {"API_PORT": 6123}}}
	restore := SetScenarioCLIForTests(cli)
	defer restore()

	resolved, info, err := ResolveURLAtPath(context.Background(), "generated", "/tmp/workspace/scenarios/generated", "/notes")
	if err != nil {
		t.Fatalf("ResolveURLAtPath: %v", err)
	}
	if got, want := resolved, "http://localhost:6123/notes"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	if info == nil || info.Name != "API_PORT" {
		t.Fatalf("port info = %#v", info)
	}
	if got, want := cli.paths, []string{"/tmp/workspace/scenarios/generated", "/tmp/workspace/scenarios/generated"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("path lookups = %#v, want %#v", got, want)
	}
}

func TestResolvePortRetriesTransientLookupFailure(t *testing.T) {
	cli := &retryScenarioCLI{}
	restore := SetScenarioCLIForTests(cli)
	defer restore()

	info, err := ResolvePort(context.Background(), "web-console", "UI_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil || info.Port != 36233 || info.Name != "UI_PORT" {
		t.Fatalf("unexpected port info: %+v", info)
	}
	if cli.attempts < 3 {
		t.Fatalf("expected retries before success, attempts=%d", cli.attempts)
	}
}

func TestDefaultScenarioCLILookupPortDelegatesToAPICoreResolver(t *testing.T) {
	resolver := &recordingPortResolver{}
	cli := &DefaultScenarioCLI{PortResolver: resolver}

	port, err := cli.LookupPort(context.Background(), "workspace-sandbox", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 49152 {
		t.Fatalf("unexpected port: %d", port)
	}
	if resolver.scenario != "workspace-sandbox" || resolver.port != "API_PORT" {
		t.Fatalf("resolver called with scenario=%q port=%q", resolver.scenario, resolver.port)
	}
}
