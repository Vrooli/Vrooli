package domain

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"resource-doc-parse/cli/internal/artifact"
	"resource-doc-parse/cli/internal/discovery"
	"resource-doc-parse/cli/internal/env"
	"resource-doc-parse/cli/internal/parse"
	"resource-doc-parse/cli/internal/platform"
	"resource-doc-parse/cli/internal/version"
)

//go:embed testdata/health.md
var healthFixture []byte

// Service is the default home for resource-specific Go logic in a native-cli
// resource.
type Service struct {
	Config   env.Config
	Runtime  discovery.Runtime
	Manifest version.Manifest
}

// NewService wires the default resource-local implementation surface.
func NewService(cfg env.Config, runtime discovery.Runtime) Service {
	return Service{
		Config:  cfg,
		Runtime: runtime,
		Manifest: version.Manifest{
			InstalledPath: runtime.InstalledManifest,
			SourcePath:    runtime.SourceManifestPath,
		},
	}
}

// PrintInfo prints placeholder runtime metadata.
func (s Service) PrintInfo(name, version, description string) error {
	fmt.Printf("%s %s\n", name, version)
	fmt.Printf("%s\n", description)
	if s.Runtime.InstalledManifest != "" {
		fmt.Printf("manifest: %s\n", s.Runtime.InstalledManifest)
	}
	return nil
}

// PrintStatus prints placeholder status output.
func (s Service) PrintStatus() error {
	return s.Health()
}

// PrintDomainHelp prints placeholder guidance for the resource-specific command
// surface.
func (s Service) PrintDomainHelp() error {
	return s.Capabilities()
}

func (s Service) resolver() artifact.Resolver {
	return artifact.Resolver{
		SourceRoot:       s.Runtime.SourceRoot,
		ExecutablePath:   s.Runtime.ExecutablePath,
		InstalledDataDir: s.Config.DataDir,
	}
}

func (s Service) runner(ctx context.Context) (*parse.Runner, artifact.Artifact, error) {
	if err := platform.CheckCurrent(); err != nil {
		return nil, artifact.Artifact{}, err
	}
	resolved, err := s.resolver().Resolve()
	if err != nil {
		return nil, artifact.Artifact{}, err
	}
	runner, err := parse.NewRunner(ctx, resolved.Path)
	if err != nil {
		return nil, artifact.Artifact{}, err
	}
	return runner, resolved, nil
}

func (s Service) Health() error {
	ctx := context.Background()
	runner, resolved, err := s.runner(ctx)
	if err != nil {
		return fmt.Errorf("unhealthy: %w", err)
	}
	defer runner.Close()
	tmp, err := os.CreateTemp("", "vrooli-doc-parse-health-*.rtf")
	if err != nil {
		return fmt.Errorf("unhealthy: create fixture: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(healthFixture); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("unhealthy: write fixture: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("unhealthy: close fixture: %w", err)
	}
	response, err := runner.Run(path, []string{"content"})
	if err != nil {
		return fmt.Errorf("unhealthy: parser probe: %w", err)
	}
	var result struct {
		TerminalState string `json:"terminal_state"`
	}
	if err := json.Unmarshal(response, &result); err != nil || result.TerminalState != "parsed" {
		return fmt.Errorf("unhealthy: parser probe returned %s", strings.TrimSpace(string(response)))
	}
	fmt.Printf("healthy artifact=%s sha256=%s\n", resolved.Path, resolved.SHA256)
	return nil
}

func (s Service) Capabilities() error {
	return printJSON(map[string]any{
		"resource": "doc-parse",
		"runtime":  "wasi+wazero",
		"capabilities": map[string]any{
			"content":            true,
			"tables":             true,
			"geometry":           true,
			"pdf_classification": true,
		},
	})
}

func (s Service) Version(name, version string) error {
	resolved, err := s.resolver().Resolve()
	result := map[string]any{"name": name, "version": version, "runtime": "wasi+wazero"}
	if err != nil {
		result["artifact_error"] = err.Error()
	} else {
		result["artifact"] = resolved.Path
		result["artifact_sha256"] = resolved.SHA256
	}
	return printJSON(result)
}

func (s Service) Parse(input string, capabilities []string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("parse requires an input path")
	}
	runner, _, err := s.runner(context.Background())
	if err != nil {
		return err
	}
	defer runner.Close()
	response, err := runner.Run(input, capabilities)
	if err != nil {
		return err
	}
	return printRaw(response)
}

func (s Service) Classify(input string) error {
	if !strings.EqualFold(filepath.Ext(input), ".pdf") {
		return fmt.Errorf("classify requires a PDF input")
	}
	runner, _, err := s.runner(context.Background())
	if err != nil {
		return err
	}
	defer runner.Close()
	response, err := runner.Run(input, []string{"content", "geometry"})
	if err != nil {
		return err
	}
	var source map[string]any
	if err := json.Unmarshal(response, &source); err != nil {
		return fmt.Errorf("decode classification: %w", err)
	}
	selected := map[string]any{}
	for _, key := range []string{"path", "format", "terminal_state", "error", "pdf_type", "confidence", "page_count", "pages_needing_ocr", "has_encoding_issues"} {
		if value, ok := source[key]; ok {
			selected[key] = value
		}
	}
	return printJSON(selected)
}

func printRaw(raw json.RawMessage) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	return printJSON(value)
}

func printJSON(value any) error {
	data, err := cliout.MarshalIndent(value)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
