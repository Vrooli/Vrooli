package viewer

// DOC: ../../docs/plans/knowledge-observatory-documentation-hub-expansion.md#phase-3-document-viewer-week-4
import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrServiceUnavailable = errors.New("viewer service unavailable")
	ErrPathRequired       = errors.New("path is required")
	ErrPathInvalid        = errors.New("path is invalid")
	ErrDocNotFound        = errors.New("document not found")
	ErrFormatInvalid      = errors.New("format is invalid")
	ErrResetUnsupported   = errors.New("reset is not supported for this document")
)

type ContentFormat string

const (
	FormatRaw         ContentFormat = "raw"
	FormatHighlighted ContentFormat = "highlighted"
	FormatPreview     ContentFormat = "preview"
)

// Service provides document viewing and reset operations.
type Service struct {
	scenariosRoot string
	repoRoot      string
}

// DocContentRequest describes a document content request.
type DocContentRequest struct {
	Path   string
	Format string
}

// DocResetRequest describes a reset operation for a document.
type DocResetRequest struct {
	Path           string
	MaxAgeDays     int
	KeepMinEntries int
	PreviewOnly    bool
}

func (r *DocContentRequest) normalize() error {
	if r == nil {
		return ErrPathRequired
	}
	r.Path = strings.TrimSpace(r.Path)
	if r.Path == "" {
		return ErrPathRequired
	}
	format := strings.TrimSpace(strings.ToLower(r.Format))
	if format == "" {
		format = string(FormatRaw)
	}
	switch ContentFormat(format) {
	case FormatRaw, FormatHighlighted, FormatPreview:
		r.Format = format
	default:
		return ErrFormatInvalid
	}
	return nil
}

func (r *DocResetRequest) normalize() error {
	if r == nil {
		return ErrPathRequired
	}
	r.Path = strings.TrimSpace(r.Path)
	if r.Path == "" {
		return ErrPathRequired
	}
	if r.MaxAgeDays < 0 || r.KeepMinEntries < 0 {
		return ErrPathInvalid
	}
	return nil
}

// NewService initializes a viewer service rooted at the scenario directory.
func NewService(scenariosRoot string) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrPathInvalid
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil || !info.IsDir() {
		return nil, ErrPathInvalid
	}
	repoRoot := filepath.Dir(scenariosRoot)
	if repoRoot == scenariosRoot {
		repoRoot = ""
	}
	return &Service{scenariosRoot: scenariosRoot, repoRoot: repoRoot}, nil
}

func (s *Service) resolveDocPath(raw string) (string, string, os.FileInfo, error) {
	if s == nil {
		return "", "", nil, ErrServiceUnavailable
	}
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "", "", nil, ErrPathRequired
	}
	clean = filepath.Clean(clean)

	var abs string
	if filepath.IsAbs(clean) {
		abs = clean
	} else if s.repoRoot != "" {
		abs = filepath.Join(s.repoRoot, clean)
	} else {
		abs = clean
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", "", nil, ErrDocNotFound
	}
	if info.IsDir() {
		return "", "", nil, ErrPathInvalid
	}
	if !isDocFile(abs) {
		return "", "", nil, ErrPathInvalid
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", "", nil, ErrPathInvalid
	}
	if s.repoRoot != "" {
		rel, err := filepath.Rel(s.repoRoot, resolved)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", "", nil, ErrPathInvalid
		}
		info, err = os.Stat(resolved)
		if err != nil {
			return "", "", nil, ErrDocNotFound
		}
		return resolved, filepath.ToSlash(rel), info, nil
	}
	return resolved, filepath.ToSlash(clean), info, nil
}

func isDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".mdx", ".txt", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
