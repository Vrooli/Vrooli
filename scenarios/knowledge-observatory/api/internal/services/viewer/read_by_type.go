package viewer

import (
	"context"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/docschema"
)

// DocReadByTypeRequest describes a document read request using a friendly type name.
type DocReadByTypeRequest struct {
	ScenarioName string
	DocType      string
	Format       string
}

// GetContentByType resolves a doc type to its canonical path and reads it.
func (s *Service) GetContentByType(ctx context.Context, req DocReadByTypeRequest) (*DocContentResult, error) {
	if s == nil {
		return nil, ErrServiceUnavailable
	}
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, ErrScenarioRequired
	}
	if strings.Contains(scenario, "/") || strings.Contains(scenario, "..") {
		return nil, ErrScenarioInvalid
	}

	dt, err := docschema.ParseDocType(req.DocType)
	if err != nil {
		return nil, ErrDocTypeInvalid
	}
	relPath := filepath.Join("scenarios", scenario, dt.ExpectedPath())

	return s.GetContent(ctx, DocContentRequest{
		Path:   relPath,
		Format: req.Format,
	})
}
