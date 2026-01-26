package explorer

// DOC: docs/plans/knowledge-observatory-documentation-hub-expansion.md#phase-2-scenario-documentation-explorer-week-3

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/services/dochealth"
)

// Service provides scenario documentation exploration operations.
type Service struct {
	scenariosRoot string
	repoRoot      string
	health        *dochealth.Service
}

// NewService initializes the explorer service.
func NewService(scenariosRoot string, health *dochealth.Service) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, dochealth.ErrScenarioRootInvalid
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil || !info.IsDir() {
		return nil, dochealth.ErrScenarioRootInvalid
	}
	if health == nil {
		health, err = dochealth.NewService(scenariosRoot)
		if err != nil {
			return nil, err
		}
	}
	repoRoot := filepath.Dir(scenariosRoot)
	if repoRoot == scenariosRoot {
		repoRoot = ""
	}
	return &Service{scenariosRoot: scenariosRoot, repoRoot: repoRoot, health: health}, nil
}

// ListScenarios returns documentation summary information for each scenario.
func (s *Service) ListScenarios(ctx context.Context) ([]dochealth.ScenarioSummary, error) {
	if s == nil || s.health == nil {
		return nil, dochealth.ErrScenarioRootInvalid
	}
	return s.health.ListScenarios(ctx)
}

func (s *Service) scenarioPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", dochealth.ErrScenarioNameInvalid
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", dochealth.ErrScenarioNameInvalid
	}
	path := filepath.Join(s.scenariosRoot, name)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", dochealth.ErrScenarioNotFound
	}
	return path, nil
}

func (s *Service) repoRelative(path string) string {
	if s.repoRoot == "" {
		return filepath.ToSlash(filepath.Clean(path))
	}
	rel, err := filepath.Rel(s.repoRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}
