package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathEscape is returned when a component's SourcePath resolves to
// a location outside the configured source root. Treated by the
// service-error mapper as InvalidArgument — the registry row is
// corrupt and the operator should re-index after fixing the layout on
// disk.
type ErrPathEscape struct {
	SourcePath string
	Root       string
}

func (e ErrPathEscape) Error() string {
	return fmt.Sprintf("source_path %q escapes configured root %q", e.SourcePath, e.Root)
}

// ErrContentConflict is returned when the caller supplied an
// expected_sha256 that does not match the file's current digest. Maps
// to Connect's FailedPrecondition.
type ErrContentConflict struct {
	Got      string
	Expected string
}

func (e ErrContentConflict) Error() string {
	return fmt.Sprintf("content sha256 mismatch (got %s, expected %s)", e.Got, e.Expected)
}

// ContentStore reads and writes the on-disk source file backing a
// component. Lives next to the Repository interface because both seams
// are filesystem-rooted; kept separate from Repository because content
// I/O is not SQL.
type ContentStore interface {
	Read(ctx context.Context, c Component) (Content, error)
	Write(ctx context.Context, c Component, in WriteContentInput) (Content, error)
}

// Content is the I/O envelope returned from Read/Write.
type Content struct {
	Body       string
	SourcePath string
	SHA256     string
}

// WriteContentInput carries everything Write needs.
type WriteContentInput struct {
	Body           string
	ExpectedSHA256 string // empty = no optimistic-concurrency check
}

// PathContentStore extends ContentStore for a companion in the current entry
// file's version folder. Callers may only name a single file; version-folder
// traversal is intentionally not a public content API.
type PathContentStore interface {
	ReadPath(ctx context.Context, c Component, path string) (Content, error)
	WritePath(ctx context.Context, c Component, path string, in WriteContentInput) (Content, error)
}

// FSContentStore is the production ContentStore backed by os.* on a
// fixed root. The path-traversal guard is enforced on every call —
// filepath.Clean does the rejoin, and the result must remain under
// root after evaluating any "../" segments.
type FSContentStore struct {
	root string
}

// NewFSContentStore constructs an FSContentStore rooted at root. The
// root must be absolute; callers resolve it via api-core/storage
// before constructing the store.
func NewFSContentStore(root string) *FSContentStore {
	return &FSContentStore{root: root}
}

// Read returns the full file content + SHA-256 digest. Returns
// ErrPathEscape when SourcePath resolves outside root, and the os.*
// error (wrapped) on any other filesystem failure.
func (s *FSContentStore) Read(_ context.Context, c Component) (Content, error) {
	return s.ReadPath(context.Background(), c, "")
}

func (s *FSContentStore) ReadPath(_ context.Context, c Component, path string) (Content, error) {
	if path != "" {
		if filepath.Base(path) != path || !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
			return Content{}, ErrPathEscape{SourcePath: path, Root: s.root}
		}
		c.SourcePath = filepath.ToSlash(filepath.Join(filepath.Dir(c.SourcePath), path))
	}
	abs, err := s.resolve(c.SourcePath)
	if err != nil {
		return Content{}, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return Content{}, fmt.Errorf("read component content %q: %w", c.SourcePath, err)
	}
	return Content{
		Body:       string(raw),
		SourcePath: c.SourcePath,
		SHA256:     digest(raw),
	}, nil
}

// Write overwrites the source file in place. When ExpectedSHA256 is
// non-empty, it must match the current on-disk digest or
// ErrContentConflict is returned without writing.
func (s *FSContentStore) Write(_ context.Context, c Component, in WriteContentInput) (Content, error) {
	return s.WritePath(context.Background(), c, "", in)
}

func (s *FSContentStore) WritePath(_ context.Context, c Component, path string, in WriteContentInput) (Content, error) {
	if path != "" {
		if filepath.Base(path) != path || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
			return Content{}, ErrPathEscape{SourcePath: path, Root: s.root}
		}
		c.SourcePath = filepath.ToSlash(filepath.Join(filepath.Dir(c.SourcePath), path))
	}
	abs, err := s.resolve(c.SourcePath)
	if err != nil {
		return Content{}, err
	}
	if in.ExpectedSHA256 != "" {
		existing, err := os.ReadFile(abs)
		if err != nil {
			return Content{}, fmt.Errorf("read component content for guard %q: %w", c.SourcePath, err)
		}
		got := digest(existing)
		if got != in.ExpectedSHA256 {
			return Content{}, ErrContentConflict{Got: got, Expected: in.ExpectedSHA256}
		}
	}
	if err := os.WriteFile(abs, []byte(in.Body), 0o600); err != nil {
		return Content{}, fmt.Errorf("write component content %q: %w", c.SourcePath, err)
	}
	return Content{
		Body:       in.Body,
		SourcePath: c.SourcePath,
		SHA256:     digest([]byte(in.Body)),
	}, nil
}

// resolve cleans the relative SourcePath against root and rejects
// anything that escapes — either via absolute path, "..", or symlink
// (EvalSymlinks). Returns the absolute on-disk path on success.
func (s *FSContentStore) resolve(sourcePath string) (string, error) {
	if sourcePath == "" {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	if filepath.IsAbs(sourcePath) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	cleaned := filepath.Clean(sourcePath)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	abs := filepath.Join(s.root, cleaned)
	// Final guard: after Clean+Join, the absolute path must still sit
	// under root. Defends against pathological inputs filepath.Clean
	// didn't normalize the way we expected.
	rel, err := filepath.Rel(s.root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	return abs, nil
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
