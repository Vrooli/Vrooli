package domains

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CLIGroupExtractor derives domains from the command groups declared in
// the CLI command manifest. It is a lower ladder rung used for convergence.
type CLIGroupExtractor struct {
	surfaces SurfaceProvider
}

// NewCLIGroupExtractor returns the production CLI-group extractor.
func NewCLIGroupExtractor() *CLIGroupExtractor { return NewCLIGroupExtractorWithSurfaceProvider(nil) }

func NewCLIGroupExtractorWithSurfaceProvider(surfaces SurfaceProvider) *CLIGroupExtractor {
	if surfaces == nil {
		surfaces = NewLocalSurfaceProvider()
	}
	return &CLIGroupExtractor{surfaces: surfaces}
}

// Source identifies this rung.
func (*CLIGroupExtractor) Source() Source { return SourceCLIGroups }

// cliManifest is the minimal shape we read from the CLI command manifest; only
// the group names matter for domain derivation.
type cliManifest struct {
	Groups []struct {
		Name string `json:"name"`
	} `json:"groups"`
}

// Extract reads the scenario CLI command manifest and maps each command group to a domain.
// A missing manifest returns an empty extraction; a malformed one errors.
func (e *CLIGroupExtractor) Extract(ctx context.Context, scenarioDir string) (Extraction, error) {
	provider := e.surfaces
	if provider == nil {
		provider = NewLocalSurfaceProvider()
	}
	inv, err := provider.Inspect(ctx, scenarioDir)
	if err != nil {
		return Extraction{}, fmt.Errorf("inspect surfaces: %w", err)
	}
	cli, ok := surfaceByID(inv, "cli")
	if !ok {
		return Extraction{Source: SourceCLIGroups, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}, nil
	}
	manifestRel := filepath.ToSlash(filepath.Join(surfaceRel(scenarioDir, cli.Path), "manifest.json"))
	path := filepath.Join(cli.Path, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Extraction{Source: SourceCLIGroups, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}, nil
		}
		return Extraction{}, fmt.Errorf("read %s: %w", manifestRel, err)
	}
	var m cliManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Extraction{}, fmt.Errorf("%s: parse: %w", manifestRel, err)
	}
	out := Extraction{Source: SourceCLIGroups, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}
	for _, g := range m.Groups {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			continue
		}
		out.Domains = append(out.Domains, ExtractedDomain{
			Name:  name,
			Paths: []string{"cli/domains/" + name + "/"},
		})
	}
	sort.Slice(out.Domains, func(i, j int) bool { return out.Domains[i].Name < out.Domains[j].Name })
	return out, nil
}
