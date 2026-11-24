package facts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	maxFlowSourceFiles = 120
	maxFlowSourceBytes = 16 << 20
)

var flowScenarioPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// SearchFeatureFlow reads one bounded source path in one scenario and derives
// at most five short traces from the Go call nodes in Code Facts' graph.
func (s *Service) SearchFeatureFlow(ctx context.Context, query, scenario, relativePath string, limit int) (*FlowSearchResponse, error) {
	query = strings.TrimSpace(query)
	scenario = strings.TrimSpace(scenario)
	relativePath = filepath.Clean(strings.TrimSpace(relativePath))
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if !flowScenarioPattern.MatchString(scenario) {
		return nil, fmt.Errorf("scenario must be one scenario slug")
	}
	if relativePath == "." || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path must be a relative source directory inside the scenario")
	}
	base, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	root := filepath.Join(base.GetRootPath(), relativePath)
	baseReal, err := filepath.EvalSymlinks(base.GetRootPath())
	if err != nil {
		return nil, err
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(baseReal, rootReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path is outside the requested scenario")
	}
	info, err := os.Stat(rootReal)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("source flow path must be a directory")
	}
	if err := flowSourceBudget(rootReal); err != nil {
		return nil, err
	}
	report, err := s.Describe(ctx, &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario, Path: rootReal},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_CALLS}, UseCache: true, IncludeGraphs: true,
	})
	if err != nil {
		return nil, fmt.Errorf("read source call graph: %w", err)
	}
	return DeriveFeatureFlows(report.GetGraphs(), query, scenario, limit), nil
}

func flowSourceBudget(root string) error {
	count, bytes := 0, int64(0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		count++
		bytes += info.Size()
		if count > maxFlowSourceFiles || bytes > maxFlowSourceBytes {
			return fmt.Errorf("source flow scope exceeds bounded budget (%d files, %d bytes)", maxFlowSourceFiles, maxFlowSourceBytes)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("source flow scope contains no Go files")
	}
	return nil
}
