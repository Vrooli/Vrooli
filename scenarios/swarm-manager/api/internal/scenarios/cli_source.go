package scenarios

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
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
	client       *vroolicli.Client
	includePorts bool
}

// DirectoryProvider lists scenarios directly from the local scenarios
// directory. It intentionally avoids lifecycle/status checks; graph topology
// projections use it when they only need stable scenario identity for target
// context.
type DirectoryProvider struct {
	scenariosDir string
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
		client:       vroolicli.New(vroolicli.WithTimeout(timeout)),
		includePorts: options.IncludePorts,
	}
}

// NewDirectoryProvider creates a filesystem-backed scenario source.
func NewDirectoryProvider(scenariosDir string) *DirectoryProvider {
	return &DirectoryProvider{scenariosDir: strings.TrimSpace(scenariosDir)}
}

// List retrieves scenarios using `vrooli scenario list --json`, optionally
// including port metadata when configured. It decodes the typed
// vrooli.cli.v1 contract, so a CLI output change is a compile error here rather
// than a silently empty or mis-shaped result.
func (p *CLIProvider) List(ctx context.Context) ([]ScenarioSource, error) {
	var opts []vroolicli.ListScenariosOption
	if p.includePorts {
		opts = append(opts, vroolicli.WithPorts())
	}

	resp, err := p.client.ListScenarios(ctx, opts...)
	if err != nil {
		return nil, err
	}

	scenarios := make([]ScenarioSource, 0, len(resp.GetScenarios()))
	for _, item := range resp.GetScenarios() {
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			continue
		}
		scenarios = append(scenarios, ScenarioSource{
			Name:        name,
			Description: strings.TrimSpace(item.GetDescription()),
			Path:        strings.TrimSpace(item.GetPath()),
			Status:      strings.TrimSpace(item.GetStatus()),
			Tags:        item.GetTags(),
			Version:     strings.TrimSpace(item.GetVersion()),
		})
	}

	return scenarios, nil
}

// List retrieves scenario identities from directories that contain a Vrooli
// scenario service contract.
func (p *DirectoryProvider) List(_ context.Context) ([]ScenarioSource, error) {
	root := p.scenariosDir
	if root == "" {
		root = "scenarios"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	scenarios := make([]ScenarioSource, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		path := filepath.Join(root, name)
		if _, err := os.Stat(filepath.Join(path, ".vrooli", "service.json")); err != nil {
			continue
		}
		scenarios = append(scenarios, ScenarioSource{
			Name:   name,
			Path:   path,
			Status: "available",
		})
	}
	return scenarios, nil
}
