package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
		if filepath.Base(path) != path || !isReadableCompanion(path) {
			return Content{}, ErrPathEscape{SourcePath: path, Root: s.root}
		}
		c.SourcePath = filepath.ToSlash(filepath.Join(filepath.Dir(c.SourcePath), path))
	}
	abs, err := s.resolve(c.SourcePath)
	if err != nil {
		return Content{}, err
	}
	raw, err := readFileInRoot(s.root, abs)
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
		if filepath.Base(path) != path || !isWritableCompanion(path) {
			return Content{}, ErrPathEscape{SourcePath: path, Root: s.root}
		}
		c.SourcePath = filepath.ToSlash(filepath.Join(filepath.Dir(c.SourcePath), path))
	}
	abs, err := s.resolve(c.SourcePath)
	if err != nil {
		return Content{}, err
	}
	if in.ExpectedSHA256 != "" {
		existing, err := readFileInRoot(s.root, abs)
		if err != nil {
			return Content{}, fmt.Errorf("read component content for guard %q: %w", c.SourcePath, err)
		}
		got := digest(existing)
		if got != in.ExpectedSHA256 {
			return Content{}, ErrContentConflict{Got: got, Expected: in.ExpectedSHA256}
		}
	}
	if err := writeFileAtomicInRoot(s.root, abs, []byte(in.Body), 0o600); err != nil {
		if isRootEscapeError(err) || hasSymlinkParent(s.root, abs) {
			return Content{}, ErrPathEscape{SourcePath: c.SourcePath, Root: s.root}
		}
		return Content{}, fmt.Errorf("write component content %q: %w", c.SourcePath, err)
	}
	return Content{
		Body:       in.Body,
		SourcePath: c.SourcePath,
		SHA256:     digest([]byte(in.Body)),
	}, nil
}

// Component versions may carry authored, non-code artifacts alongside their
// entry and companion modules. JSON contracts are writable only through the
// service boundary, which canonicalizes them before storage and preserves the
// same path and optimistic-concurrency guards as source files.
func isReadableCompanion(path string) bool {
	return strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".json")
}

func isWritableCompanion(path string) bool {
	return isReadableCompanion(path)
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
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", fmt.Errorf("resolve component root: %w", err)
	}
	abs := filepath.Join(root, cleaned)
	// Final guard: after Clean+Join, the absolute path must still sit
	// under root. Defends against pathological inputs filepath.Clean
	// didn't normalize the way we expected.
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	return abs, nil
}

// readFileInRoot keeps reads anchored to the configured root just like writes.
// Lexical resolution alone is insufficient because a component path may pass
// through a symlink that is replaced after validation.
func readFileInRoot(rootPath, absolutePath string) ([]byte, error) {
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolve component root: %w", err)
	}
	abs, err := filepath.Abs(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("resolve component path: %w", err)
	}
	relative, err := filepath.Rel(rootPath, abs)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, ErrPathEscape{SourcePath: absolutePath, Root: rootPath}
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open component root: %w", err)
	}
	defer root.Close()
	file, err := root.Open(filepath.ToSlash(relative))
	if err != nil {
		if isRootEscapeError(err) {
			return nil, ErrPathEscape{SourcePath: absolutePath, Root: rootPath}
		}
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func isRootEscapeError(err error) bool {
	return errors.Is(err, syscall.EXDEV) || strings.Contains(err.Error(), "path escapes from parent")
}

func hasSymlinkParent(rootPath, absolutePath string) bool {
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootPath, absolutePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	for parent := filepath.Dir(relative); parent != "."; parent = filepath.Dir(parent) {
		info, err := os.Lstat(filepath.Join(rootPath, parent))
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
