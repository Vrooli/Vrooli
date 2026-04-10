package scenarios

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#integration-strategy
// DOC: docs/internal/INTEROP_AUDIT.md

const defaultCLITimeout = 20 * time.Second

// ScenarioSource describes scenario data sourced from the Vrooli CLI.
type ScenarioSource struct {
	Name        string
	Description string
	Path        string
	Status      string
	Tags        []string
	Version     string
}

// Source provides scenario inventory data.
type Source interface {
	List(ctx context.Context) ([]ScenarioSource, error)
}

// CLIProviderOptions configures scenario inventory loading via the Vrooli CLI.
type CLIProviderOptions struct {
	Timeout      time.Duration
	IncludePorts bool
}

// CLIProvider lists scenarios via the Vrooli CLI.
type CLIProvider struct {
	timeout      time.Duration
	includePorts bool
}

// NewCLIProvider creates a CLI-backed scenario source.
func NewCLIProvider(timeout time.Duration) *CLIProvider {
	return NewCLIProviderWithOptions(CLIProviderOptions{
		Timeout:      timeout,
		IncludePorts: true,
	})
}

// NewCLIProviderWithOptions creates a CLI-backed scenario source with
// configurable inventory detail.
func NewCLIProviderWithOptions(options CLIProviderOptions) *CLIProvider {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultCLITimeout
	}

	return &CLIProvider{
		timeout:      timeout,
		includePorts: options.IncludePorts,
	}
}

// List retrieves scenarios using `vrooli scenario list --json`, optionally
// including port metadata when configured.
func (p *CLIProvider) List(ctx context.Context) ([]ScenarioSource, error) {
	args := []string{"scenario", "list", "--json"}
	if p.includePorts {
		args = append(args, "--include-ports")
	}

	output, err := executeVrooliCommand(ctx, p.timeout, args...)
	if err != nil {
		return nil, err
	}

	var resp scenarioListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	scenarios := make([]ScenarioSource, 0, len(resp.Scenarios))
	for _, item := range resp.Scenarios {
		scenarios = append(scenarios, ScenarioSource{
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Path:        strings.TrimSpace(item.Path),
			Status:      strings.TrimSpace(item.Status),
			Tags:        item.Tags,
			Version:     strings.TrimSpace(item.Version),
		})
	}

	return scenarios, nil
}

// scenarioListResponse represents `vrooli scenario list --json` output.
type scenarioListResponse struct {
	Success   bool               `json:"success"`
	Scenarios []scenarioMetadata `json:"scenarios"`
}

// scenarioMetadata captures fields returned by the CLI.
type scenarioMetadata struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Path        string         `json:"path"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	Tags        []string       `json:"tags"`
	Ports       []scenarioPort `json:"ports"`
}

// scenarioPort captures port details from the CLI.
type scenarioPort struct {
	Key  string      `json:"key"`
	Step string      `json:"step"`
	Port interface{} `json:"port"`
}
