package dochealth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/doclogs"
	"knowledge-observatory/internal/docschema"
	"knowledge-observatory/internal/doctemplates"
	"knowledge-observatory/internal/docvalidation"
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
	Validation *docvalidation.Result
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
	validation, err := docvalidation.ValidateScenarioDocumentation(path)
	if err != nil {
		return nil, err
	}
	count := docvalidation.CountDocs(path)
	return &HealthResult{Validation: validation, TotalDocs: count}, nil
}

// ResetScenarioDoc applies reset rules to a known document in a scenario.
func (s *Service) ResetScenarioDoc(ctx context.Context, scenarioName string, docID string, config doclogs.ResetConfig) (*doclogs.ResetResult, string, error) {
	_ = ctx
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, "", err
	}
	resolved, err := doctemplates.NewResolverFromScenariosRoot(s.scenariosRoot).ResolveScenario(scenarioPath)
	if err != nil {
		return nil, "", err
	}
	doc, ok := resolved.Contract.ResolveIdentifier(docID)
	if !ok || doc.Operations.AppendLog == nil || !doc.Operations.AppendLog.Enabled || !doc.Operations.AppendLog.Retention.SupportsReset {
		return nil, "", fmt.Errorf("reset is not supported for document %q", docID)
	}
	result, err := doclogs.Reset(filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath)), *doc.Operations.AppendLog, config)
	return result, doc.DocType, err
}

// AuditScenario runs a comprehensive documentation audit for a scenario.
func (s *Service) AuditScenario(ctx context.Context, scenarioName string) (*docschema.AuditResult, error) {
	_ = ctx
	path, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	return docschema.AuditScenarioDocumentation(path)
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
