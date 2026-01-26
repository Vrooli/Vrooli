package dochealth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/docschema"
)

var (
	ErrScenarioNotFound    = errors.New("scenario not found")
	ErrScenarioNameInvalid = errors.New("scenario name is invalid")
	ErrScenarioRootInvalid = errors.New("scenario root is invalid")
)

// Service provides documentation health operations scoped to scenarios.
type Service struct {
	scenariosRoot string
}

// HealthResult bundles validation results with doc counts.
type HealthResult struct {
	Validation *docschema.ValidationResult
	TotalDocs  int
}

// NewService initializes the doc health service.
func NewService(scenariosRoot string) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrScenarioRootInvalid
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil {
		return nil, ErrScenarioRootInvalid
	}
	if !info.IsDir() {
		return nil, ErrScenarioRootInvalid
	}
	return &Service{scenariosRoot: scenariosRoot}, nil
}

// ValidateScenario runs documentation structure validation for a scenario.
func (s *Service) ValidateScenario(ctx context.Context, scenarioName string) (*HealthResult, error) {
	_ = ctx
	path, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	validation, err := docschema.ValidateScenarioDocumentation(path)
	if err != nil {
		return nil, err
	}
	count, err := countDocs(path)
	if err != nil {
		return nil, err
	}
	return &HealthResult{Validation: validation, TotalDocs: count}, nil
}

// ResetScenarioDoc applies reset rules to a known document in a scenario.
func (s *Service) ResetScenarioDoc(ctx context.Context, scenarioName string, config docschema.ResetConfig) (*docschema.ResetResult, error) {
	_ = ctx
	path, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	rel := config.DocType.ExpectedPath()
	if rel == "" {
		return nil, fmt.Errorf("unknown doc type: %s", config.DocType)
	}
	return docschema.ResetDocument(filepath.Join(path, rel), config)
}

func (s *Service) scenarioPath(scenarioName string) (string, error) {
	name := strings.TrimSpace(scenarioName)
	if name == "" || name == "." || name == ".." {
		return "", ErrScenarioNameInvalid
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", ErrScenarioNameInvalid
	}
	path := filepath.Join(s.scenariosRoot, name)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", ErrScenarioNotFound
	}
	return path, nil
}

func countDocs(scenarioPath string) (int, error) {
	count := 0
	for _, rootFile := range []string{"README.md", "PRD.md"} {
		if exists(filepath.Join(scenarioPath, rootFile)) {
			count++
		}
	}
	for _, docsRoot := range []string{filepath.Join(scenarioPath, "docs")} {
		info, err := os.Stat(docsRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		if err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if isDocFile(path) {
				count++
			}
			return nil
		}); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func isDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".json"
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
