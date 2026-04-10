package viewer

// DOC: docs/reference/api-endpoints.md#documentation-viewer
import (
	"context"

	"knowledge-observatory/internal/docschema"
)

// DocResetResult reports the outcome of a reset operation.
type DocResetResult struct {
	Path           string
	DocType        string
	RemovedCount   int
	KeptCount      int
	RemovedEntries []string
	NewContent     string
	PreviewOnly    bool
}

// ResetDocument applies retention rules to a supported document.
func (s *Service) ResetDocument(ctx context.Context, req DocResetRequest) (*DocResetResult, error) {
	if err := req.normalize(); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	abs, rel, _, err := s.resolveDocPath(req.Path)
	if err != nil {
		return nil, err
	}
	docType, ok := docschema.DocTypeForPath(abs)
	if !ok {
		return nil, ErrResetUnsupported
	}
	if docType != docschema.DocTypeProblems && docType != docschema.DocTypeProgress {
		return nil, ErrResetUnsupported
	}
	result, err := docschema.ResetDocument(abs, docschema.ResetConfig{
		DocType:        docType,
		MaxAgeDays:     req.MaxAgeDays,
		KeepMinEntries: req.KeepMinEntries,
		PreviewMode:    req.PreviewOnly,
	})
	if err != nil {
		return nil, err
	}
	return &DocResetResult{
		Path:           rel,
		DocType:        string(docType),
		RemovedCount:   result.RemovedCount,
		KeptCount:      result.KeptCount,
		RemovedEntries: result.RemovedEntries,
		NewContent:     result.NewContent,
		PreviewOnly:    req.PreviewOnly,
	}, nil
}
