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

	repocontract "github.com/vrooli/repo-contract-go"
)

// CLIGroupExtractor derives domains from the command groups declared in
// the CLI command manifest. It is a lower ladder rung used for convergence.
type CLIGroupExtractor struct{}

// NewCLIGroupExtractor returns the production CLI-group extractor.
func NewCLIGroupExtractor() *CLIGroupExtractor { return &CLIGroupExtractor{} }

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
func (e *CLIGroupExtractor) Extract(_ context.Context, scenarioDir string) (Extraction, error) {
	manifestRel := cliManifestRel(scenarioDir)
	path := filepath.Join(scenarioDir, filepath.FromSlash(manifestRel))
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Extraction{Source: SourceCLIGroups}, nil
		}
		return Extraction{}, fmt.Errorf("read %s: %w", manifestRel, err)
	}
	var m cliManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Extraction{}, fmt.Errorf("%s: parse: %w", manifestRel, err)
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

func cliManifestRel(scenarioDir string) string {
	repoRoot, err := repocontract.FindRepoRootFromPath(scenarioDir)
	if err != nil {
		repoRoot = ""
	}
	rel, _ := repocontract.ScenarioCLIManifestRel(repoRoot)
	return rel
}
