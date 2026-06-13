package scenarioport

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliClient is the shared typed Vrooli CLI client. It decodes the
// vrooli.cli.v1 contracts instead of hand-parsing CLI JSON, so a CLI output
// change is a compile error here rather than a silently empty or wrong result.
var cliClient = vroolicli.New()

type PortInfo struct {
	Name string
	Port int
}

type ScenarioMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ScenarioCLI abstracts interactions with the Vrooli scenario CLI.
// This seam enables testing scenario operations without shelling out to the CLI.
type ScenarioCLI interface {
	// LookupPort retrieves the port number for a named port of a scenario.
	LookupPort(ctx context.Context, scenarioName, portName string) (int, error)

	// ListScenarios returns metadata for all known scenarios.
	ListScenarios(ctx context.Context) ([]ScenarioMetadata, error)

	// GetStatus retrieves the current status of a scenario (running, stopped, etc.).
	GetStatus(ctx context.Context, scenarioName string) (string, error)
}

type scenarioPortResolver interface {
	ResolveScenarioPort(ctx context.Context, scenarioName, portName string) (int, error)
}

// DefaultScenarioCLI implements ScenarioCLI using api-core discovery.
type DefaultScenarioCLI struct {
	PortResolver scenarioPortResolver
}

// LookupPort uses api-core discovery to get the port number.
func (c *DefaultScenarioCLI) LookupPort(ctx context.Context, scenarioName, portName string) (int, error) {
	resolver := c.PortResolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	return resolver.ResolveScenarioPort(ctx, scenarioName, portName)
}

// ListScenarios returns all scenarios via the typed CLI client, mapping the
// vrooli.cli.v1 scenario-list contract onto ScenarioMetadata.
func (c *DefaultScenarioCLI) ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	resp, err := cliClient.ListScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}

	scenarios := make([]ScenarioMetadata, 0, len(resp.GetScenarios()))
	for _, item := range resp.GetScenarios() {
		trimmedName := strings.TrimSpace(item.GetName())
		if trimmedName == "" {
			continue
		}
		scenarios = append(scenarios, ScenarioMetadata{
			Name:        trimmedName,
			Description: strings.TrimSpace(item.GetDescription()),
			Status:      strings.TrimSpace(item.GetStatus()),
		})
	}

	return scenarios, nil
}

// GetStatus reads a scenario's lifecycle status via the typed CLI client. A
// failed lookup is not an error condition — it degrades to "unknown".
func (c *DefaultScenarioCLI) GetStatus(ctx context.Context, scenarioName string) (string, error) {
	resp, err := cliClient.ScenarioStatus(ctx, scenarioName)
	if err != nil {
		return "unknown", nil
	}

	switch strings.ToLower(strings.TrimSpace(resp.GetScenario().GetStatus())) {
	case "running":
		return "running", nil
	case "stopped":
		return "stopped", nil
	default:
		return "unknown", nil
	}
}

// Compile-time interface enforcement
var _ ScenarioCLI = (*DefaultScenarioCLI)(nil)

// MockScenarioCLI is a test double for ScenarioCLI.
type MockScenarioCLI struct {
	Ports     map[string]map[string]int // scenarioName -> portName -> port
	Scenarios []ScenarioMetadata
	Statuses  map[string]string // scenarioName -> status
	Errors    map[string]error  // method -> error
}

// NewMockScenarioCLI creates a new mock with empty defaults.
func NewMockScenarioCLI() *MockScenarioCLI {
	return &MockScenarioCLI{
		Ports:    make(map[string]map[string]int),
		Statuses: make(map[string]string),
		Errors:   make(map[string]error),
	}
}

// LookupPort returns the configured port or error.
func (m *MockScenarioCLI) LookupPort(_ context.Context, scenarioName, portName string) (int, error) {
	if err, ok := m.Errors["LookupPort"]; ok && err != nil {
		return 0, err
	}
	if ports, ok := m.Ports[scenarioName]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("port %s not found for scenario %s", portName, scenarioName)
}

// ListScenarios returns the configured scenarios or error.
func (m *MockScenarioCLI) ListScenarios(_ context.Context) ([]ScenarioMetadata, error) {
	if err, ok := m.Errors["ListScenarios"]; ok && err != nil {
		return nil, err
	}
	return m.Scenarios, nil
}

// GetStatus returns the configured status or "unknown".
func (m *MockScenarioCLI) GetStatus(_ context.Context, scenarioName string) (string, error) {
	if err, ok := m.Errors["GetStatus"]; ok && err != nil {
		return "", err
	}
	if status, ok := m.Statuses[scenarioName]; ok {
		return status, nil
	}
	return "unknown", nil
}

// Compile-time interface enforcement
var _ ScenarioCLI = (*MockScenarioCLI)(nil)

// Global CLI instance (allows backward-compatible seam injection)
var scenarioCLI ScenarioCLI = &DefaultScenarioCLI{}

// SetScenarioCLIForTests replaces the global CLI instance for testing.
func SetScenarioCLIForTests(cli ScenarioCLI) func() {
	previous := scenarioCLI
	scenarioCLI = cli
	return func() {
		scenarioCLI = previous
	}
}

const (
	portLookupRetryWindow = 6 * time.Second
	portLookupRetryDelay  = 250 * time.Millisecond
)

func ResolvePort(ctx context.Context, scenarioName string, portNames ...string) (*PortInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmedScenario := strings.TrimSpace(scenarioName)
	if trimmedScenario == "" {
		return nil, fmt.Errorf("scenario name is required")
	}

	combined := append([]string{}, portNames...)
	combined = append(combined, "UI_PORT", "API_PORT")
	candidateNames := uniqueNormalizedNames(combined)

	deadline := time.Now().Add(portLookupRetryWindow)
	var lastErr error

	for {
		for _, name := range candidateNames {
			port, err := scenarioCLI.LookupPort(ctx, trimmedScenario, name)
			if err == nil && port > 0 {
				return &PortInfo{Name: name, Port: port}, nil
			}
			if err != nil {
				lastErr = fmt.Errorf("%s lookup failed: %w", name, err)
			} else {
				lastErr = fmt.Errorf("%s lookup returned invalid port %d", name, port)
			}
		}

		if ctx.Err() != nil || time.Now().After(deadline) {
			break
		}

		timer := time.NewTimer(portLookupRetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			break
		case <-timer.C:
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to resolve port for scenario %s: %w", trimmedScenario, lastErr)
	}
	return nil, fmt.Errorf("failed to resolve port for scenario %s", trimmedScenario)
}

func BuildURL(port int, rawPath string) (string, error) {
	if port <= 0 {
		return "", fmt.Errorf("invalid port: %d", port)
	}

	base := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	}

	trimmedPath := strings.TrimSpace(rawPath)
	if trimmedPath != "" && trimmedPath != "/" {
		if !strings.HasPrefix(trimmedPath, "/") {
			trimmedPath = "/" + trimmedPath
		}
		if parsed, err := url.Parse(trimmedPath); err == nil {
			base.Path = parsed.Path
			base.RawQuery = parsed.RawQuery
			base.Fragment = parsed.Fragment
		} else {
			base.Path = trimmedPath
		}
	}

	return base.String(), nil
}

func ResolveURL(ctx context.Context, scenarioName, path string, portNames ...string) (string, *PortInfo, error) {
	trimmedScenario := strings.TrimSpace(scenarioName)
	if trimmedScenario == "" {
		return "", nil, fmt.Errorf("scenario name is required")
	}

	portInfo, err := ResolvePort(ctx, trimmedScenario, portNames...)
	if err != nil {
		return "", nil, err
	}

	resolvedURL, err := BuildURL(portInfo.Port, path)
	if err != nil {
		return "", nil, err
	}

	return resolvedURL, portInfo, nil
}

// ListScenarios returns metadata for all known scenarios.
// This function delegates to the global scenarioCLI instance.
func ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return scenarioCLI.ListScenarios(ctx)
}

// GetScenarioStatus returns the current status of a scenario.
// This function delegates to the global scenarioCLI instance.
func GetScenarioStatus(ctx context.Context, scenarioName string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return scenarioCLI.GetStatus(ctx, scenarioName)
}

func uniqueNormalizedNames(names []string) []string {
	seen := make(map[string]struct{})
	ordered := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.ToUpper(strings.TrimSpace(name))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		ordered = append(ordered, trimmed)
	}
	return ordered
}
