package viewer

import (
	"context"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/docschema"
)

// DocAppendRequest describes a request to append an entry to a structured document.
type DocAppendRequest struct {
	ScenarioName string
	DocType      string
	Title        string
	Body         string
	Author       string
	Status       string
}

// DocAppendResult reports the outcome of an append operation.
type DocAppendResult struct {
	ScenarioName string
	DocType      string
	EntryAdded   string
}

// AppendEntry appends a format-compatible entry to a structured document.
func (s *Service) AppendEntry(ctx context.Context, req DocAppendRequest) (*DocAppendResult, error) {
	if s == nil {
		return nil, ErrServiceUnavailable
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, ErrScenarioRequired
	}
	if strings.Contains(scenario, "/") || strings.Contains(scenario, "..") {
		return nil, ErrScenarioInvalid
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}

	dt, err := docschema.ParseDocType(req.DocType)
	if err != nil {
		return nil, ErrDocTypeInvalid
	}
	if dt != docschema.DocTypeProblems && dt != docschema.DocTypeProgress {
		return nil, ErrAppendUnsupported
	}

	var abs string
	if s.repoRoot != "" {
		abs = filepath.Join(s.repoRoot, "scenarios", scenario, dt.ExpectedPath())
	} else {
		abs = filepath.Join(s.scenariosRoot, scenario, dt.ExpectedPath())
	}

	result, err := docschema.AppendEntry(abs, docschema.AppendConfig{
		DocType: dt,
		Title:   title,
		Body:    req.Body,
		Author:  req.Author,
		Status:  req.Status,
	})
	if err != nil {
		return nil, err
	}
	return &DocAppendResult{
		ScenarioName: scenario,
		DocType:      string(dt),
		EntryAdded:   result.EntryAdded,
	}, nil
}
