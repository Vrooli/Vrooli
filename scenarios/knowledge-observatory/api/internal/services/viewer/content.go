package viewer

// DOC: docs/reference/api-endpoints.md#documentation-viewer
import (
	"context"
	"os"
	"path/filepath"
	"time"

	"knowledge-observatory/internal/doccontract"
	"knowledge-observatory/internal/doctemplates"
)

// DocResetConfig suggests default reset settings for a document.
type DocResetConfig struct {
	MaxAgeDays     int
	KeepMinEntries int
}

// DocContentResult is the response for a document content request.
type DocContentResult struct {
	Path        string
	Content     string
	Format      string
	DocType     string
	Size        int64
	ModifiedAt  time.Time
	CanReset    bool
	ResetConfig *DocResetConfig
}

// GetContent reads a document from disk and returns metadata.
func (s *Service) GetContent(ctx context.Context, req DocContentRequest) (*DocContentResult, error) {
	if err := req.normalize(); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	abs, rel, info, err := s.resolveDocPath(req.Path)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil, ErrDocNotFound
	}
	docType := ""
	canReset := false
	if doc, ok := s.docForRepoPath(rel); ok {
		docType = doc.DocType
		canReset = doc.Operations.AppendLog != nil && doc.Operations.AppendLog.Enabled && doc.Operations.AppendLog.Retention.SupportsReset
	}
	var resetConfig *DocResetConfig
	if canReset {
		resetConfig = &DocResetConfig{
			MaxAgeDays:     30,
			KeepMinEntries: 3,
		}
	}
	return &DocContentResult{
		Path:        rel,
		Content:     string(content),
		Format:      req.Format,
		DocType:     docType,
		Size:        info.Size(),
		ModifiedAt:  info.ModTime(),
		CanReset:    canReset,
		ResetConfig: resetConfig,
	}, nil
}

func (s *Service) docForRepoPath(repoRel string) (doccontract.Document, bool) {
	scenarioName, scenarioRel, ok := splitScenarioRepoPath(repoRel)
	if !ok {
		return doccontract.Document{}, false
	}
	scenarioPath := filepath.Join(s.scenariosRoot, scenarioName)
	resolved, err := doctemplates.NewResolverFromScenariosRoot(s.scenariosRoot).ResolveScenario(scenarioPath)
	if err != nil || resolved == nil || resolved.Contract == nil {
		return doccontract.Document{}, false
	}
	return resolved.Contract.ResolvePath(scenarioRel)
}
