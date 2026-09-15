package viewer

import (
	"context"
	"path/filepath"
	"strings"
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

	doc, _, err := s.resolveContractDoc(scenario, req.DocType)
	if err != nil {
		return nil, err
	}
	relPath := filepath.Join("scenarios", scenario, filepath.FromSlash(doc.ScenarioPath))

	return s.GetContent(ctx, DocContentRequest{
		Path:   relPath,
		Format: req.Format,
	})
}
