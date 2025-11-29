package scenarioport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

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

// DefaultScenarioCLI implements ScenarioCLI using the actual Vrooli CLI.
type DefaultScenarioCLI struct{}

// LookupPort shells out to 'vrooli scenario port' to get the port number.
func (c *DefaultScenarioCLI) LookupPort(ctx context.Context, scenarioName, portName string) (int, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, portName)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return 0, fmt.Errorf("empty port response")
	}

	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid port value %q: %w", trimmed, err)
	}

	if port <= 0 {
		return 0, fmt.Errorf("port must be positive, got %d", port)
	}

	return port, nil
}

// ListScenarios shells out to 'vrooli scenario list --json' to get all scenarios.
func (c *DefaultScenarioCLI) ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}

	var payload scenarioListResponse
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse scenario list: %w", err)
	}

	scenarios := make([]ScenarioMetadata, 0, len(payload.Scenarios))
	for _, item := range payload.Scenarios {
		trimmedName := strings.TrimSpace(item.Name)
		if trimmedName == "" {
			continue
		}
		scenarios = append(scenarios, ScenarioMetadata{
			Name:        trimmedName,
			Description: strings.TrimSpace(item.Description),
			Status:      strings.TrimSpace(item.Status),
		})
	}

	return scenarios, nil
}

// GetStatus shells out to 'vrooli scenario status' to get the current status.
func (c *DefaultScenarioCLI) GetStatus(ctx context.Context, scenarioName string) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", scenarioName)
	output, err := cmd.Output()
	if err != nil {
		return "unknown", nil // Status command failing is not an error condition
	}

	statusStr := strings.TrimSpace(string(output))
	if strings.Contains(strings.ToLower(statusStr), "running") {
		return "running", nil
	} else if strings.Contains(strings.ToLower(statusStr), "stopped") {
		return "stopped", nil
	}
	return "unknown", nil
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

// Legacy compatibility - portLookupFunc now delegates to scenarioCLI
var portLookupFunc = func(ctx context.Context, scenarioName, portName string) (int, error) {
	return scenarioCLI.LookupPort(ctx, scenarioName, portName)
}

// SetPortLookupFuncForTests provides backward compatibility for existing tests.
func SetPortLookupFuncForTests(fn func(context.Context, string, string) (int, error)) func() {
	previous := portLookupFunc
	portLookupFunc = fn
	return func() {
		portLookupFunc = previous
	}
}

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

	if entry := lookupRegistryEntry(trimmedScenario); entry != nil {
		if portName, port := entry.portFor(candidateNames); port > 0 {
			return &PortInfo{Name: portName, Port: port}, nil
		}
	}

	for _, name := range candidateNames {
		port, err := portLookupFunc(ctx, trimmedScenario, name)
		if err != nil || port <= 0 {
			continue
		}
		return &PortInfo{Name: name, Port: port}, nil
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

	combined := append([]string{}, portNames...)
	combined = append(combined, "UI_PORT", "API_PORT")
	candidateNames := uniqueNormalizedNames(combined)

	if entry := lookupRegistryEntry(trimmedScenario); entry != nil {
		if resolved, info, ok := entry.resolveURL(path, candidateNames); ok {
			return resolved, info, nil
		}
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

type scenarioListResponse struct {
	Scenarios []ScenarioMetadata `json:"scenarios"`
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
