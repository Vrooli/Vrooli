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

var portLookupFunc = defaultPortLookup

func defaultPortLookup(ctx context.Context, scenarioName, portName string) (int, error) {
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
	portInfo, err := ResolvePort(ctx, scenarioName, portNames...)
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

func ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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
