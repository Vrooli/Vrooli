package viewer

// DOC: docs/reference/api-endpoints.md#documentation-viewer
import (
	"context"
	"path/filepath"

	"knowledge-observatory/internal/doccontract"
	"knowledge-observatory/internal/doclogs"
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

	var abs, rel string
	var docType string
	var op *doccontract.AppendLogOperation
	if req.Path != "" {
		var err error
		abs, rel, _, err = s.resolveDocPath(req.Path)
		if err != nil {
			return nil, err
		}
		doc, ok := s.docForRepoPath(rel)
		if !ok {
			return nil, ErrResetUnsupported
		}
		docType = doc.DocType
		op = doc.Operations.AppendLog
	} else {
		doc, scenarioPath, err := s.resolveContractDoc(req.ScenarioName, req.DocID)
		if err != nil {
			return nil, err
		}
		abs = filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath))
		rel = filepath.ToSlash(filepath.Join("scenarios", req.ScenarioName, filepath.FromSlash(doc.ScenarioPath)))
		docType = doc.DocType
		op = doc.Operations.AppendLog
	}
	if op == nil || !op.Enabled || !op.Retention.SupportsReset {
		return nil, ErrResetUnsupported
	}
	result, err := doclogs.Reset(abs, *op, doclogs.ResetConfig{
		MaxAgeDays:     req.MaxAgeDays,
		KeepMinEntries: req.KeepMinEntries,
		PreviewMode:    req.PreviewOnly,
	})
	if err != nil {
		return nil, err
	}
	return &DocResetResult{
		Path:           rel,
		DocType:        docType,
		RemovedCount:   result.RemovedCount,
		KeptCount:      result.KeptCount,
		RemovedEntries: result.RemovedEntries,
		NewContent:     result.NewContent,
		PreviewOnly:    req.PreviewOnly,
	}, nil
}
