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

// CLIManifestPath is the scenario-relative location of the CLI command
// manifest the CLIGroupExtractor reads. Note: this is the CLI *command*
// manifest (the generative source of the command tree), unrelated to the
// deleted architecture manifest.
const CLIManifestPath = "cli/manifest.json"

// CLIGroupExtractor derives domains from the command groups declared in
// cli/manifest.json. It is a lower ladder rung used for convergence.
type CLIGroupExtractor struct{}

// NewCLIGroupExtractor returns the production CLI-group extractor.
func NewCLIGroupExtractor() *CLIGroupExtractor { return &CLIGroupExtractor{} }

// Source identifies this rung.
func (*CLIGroupExtractor) Source() Source { return SourceCLIGroups }

// cliManifest is the minimal shape we read from cli/manifest.json — only
// the group names matter for domain derivation.
type cliManifest struct {
	Groups []struct {
		Name string `json:"name"`
	} `json:"groups"`
}

// Extract reads cli/manifest.json and maps each command group to a domain.
// A missing manifest returns an empty extraction; a malformed one errors.
func (e *CLIGroupExtractor) Extract(_ context.Context, scenarioDir string) (Extraction, error) {
	path := filepath.Join(scenarioDir, CLIManifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Extraction{Source: SourceCLIGroups}, nil
		}
		return Extraction{}, fmt.Errorf("read %s: %w", CLIManifestPath, err)
	}
	var m cliManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Extraction{}, fmt.Errorf("%s: parse: %w", CLIManifestPath, err)
	}
	out := Extraction{Source: SourceCLIGroups}
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
