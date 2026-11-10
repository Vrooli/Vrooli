package repohealth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Exception is the operator's recorded decision to tolerate ONE exact
// unreadable leftover path inside one destination's repository. It is not a
// switch: it names the path, the kind of read failure it was approved for, who
// approved it and when. Anything else unreadable, or this path changing shape,
// is not covered.
type Exception struct {
	ID               string     `json:"id"`
	DestinationID    string     `json:"destination_id"`
	RelPath          string     `json:"rel_path"`
	Class            EntryClass `json:"class"`
	BlobKind         BlobKind   `json:"blob_kind"`
	Problem          Problem    `json:"problem"`
	ApprovedBy       string     `json:"approved_by"`
	ApprovedBySource string     `json:"approved_by_source"`
	ApprovedAt       time.Time  `json:"approved_at"`
	AskID            string     `json:"ask_id,omitempty"`
}

// ExceptionID is the stable identity of one (destination, path) exception.
func ExceptionID(destinationID, relPath string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(destinationID) + "\x00" + path.Clean(relPath)))
	return hex.EncodeToString(sum[:12])
}

// Validate refuses anything that is not an exact, recorded, leftover-temp path.
func (e Exception) Validate() error {
	switch {
	case strings.TrimSpace(e.DestinationID) == "":
		return errors.New("exception needs a destination")
	case strings.TrimSpace(e.ApprovedBy) == "" || strings.TrimSpace(e.ApprovedBySource) == "" || e.ApprovedAt.IsZero():
		return errors.New("exception needs the approving identity and time")
	case e.Class != ClassLeftoverTemp:
		return fmt.Errorf("only an unreadable leftover temporary file can be excepted, not %q", e.Class)
	case e.Problem == "":
		return errors.New("exception needs the read failure it was approved for")
	}
	rel := filepath.ToSlash(e.RelPath)
	if rel == "" || rel != path.Clean(rel) || path.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") ||
		strings.ContainsAny(rel, "*?[") {
		return fmt.Errorf("exception path %q must be one exact path inside the repository", e.RelPath)
	}
	if class, _ := ClassifyName(rel); class != ClassLeftoverTemp {
		return fmt.Errorf("path %q does not have the shape of a leftover temporary file", e.RelPath)
	}
	return nil
}

// ExceptionStore keeps recorded exceptions.
type ExceptionStore interface {
	List(ctx context.Context, destinationID string) ([]Exception, error)
	Add(ctx context.Context, e Exception) error
	Revoke(ctx context.Context, destinationID, relPath string) error
}

// FileExceptionStore keeps one JSON file per destination under Root, outside
// every repository.
type FileExceptionStore struct {
	Root string
	mu   sync.Mutex
}

var _ ExceptionStore = (*FileExceptionStore)(nil)

type exceptionFile struct {
	Version    string      `json:"version"`
	Exceptions []Exception `json:"exceptions"`
}

const exceptionFileVersion = "dbm-repository-exceptions-v1"

func (s *FileExceptionStore) path(destinationID string) (string, error) {
	if s == nil || !filepath.IsAbs(s.Root) || strings.TrimSpace(destinationID) == "" || strings.ContainsAny(destinationID, `/\`) {
		return "", errors.New("invalid exception store or destination id")
	}
	return filepath.Join(filepath.Clean(s.Root), destinationID+".json"), nil
}

func (s *FileExceptionStore) load(p string) (exceptionFile, error) {
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return exceptionFile{Version: exceptionFileVersion}, nil
	}
	if err != nil {
		return exceptionFile{}, err
	}
	var f exceptionFile
	if err := json.Unmarshal(data, &f); err != nil {
		return exceptionFile{}, fmt.Errorf("exception record is corrupt: %w", err)
	}
	return f, nil
}

func (s *FileExceptionStore) save(p string, f exceptionFile) error {
	f.Version = exceptionFileVersion
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".exceptions-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, p)
}

func (s *FileExceptionStore) List(ctx context.Context, destinationID string) ([]Exception, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := s.path(destinationID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load(p)
	if err != nil {
		return nil, err
	}
	return f.Exceptions, nil
}

func (s *FileExceptionStore) Add(ctx context.Context, e Exception) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.RelPath = path.Clean(filepath.ToSlash(e.RelPath))
	if e.ID == "" {
		e.ID = ExceptionID(e.DestinationID, e.RelPath)
	}
	if err := e.Validate(); err != nil {
		return err
	}
	p, err := s.path(e.DestinationID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load(p)
	if err != nil {
		return err
	}
	for i := range f.Exceptions {
		if f.Exceptions[i].RelPath == e.RelPath {
			// Re-approving the same path keeps the original record: who approved
			// it first and when is evidence, not state to overwrite.
			return nil
		}
	}
	f.Exceptions = append(f.Exceptions, e)
	return s.save(p, f)
}

func (s *FileExceptionStore) Revoke(ctx context.Context, destinationID, relPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := s.path(destinationID)
	if err != nil {
		return err
	}
	relPath = path.Clean(filepath.ToSlash(relPath))
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load(p)
	if err != nil {
		return err
	}
	kept := f.Exceptions[:0]
	for _, e := range f.Exceptions {
		if e.RelPath != relPath {
			kept = append(kept, e)
		}
	}
	f.Exceptions = kept
	return s.save(p, f)
}

// Verdict separates what a recorded exception covers from what still blocks.
type Verdict struct {
	// Excused are unreadable entries covered by an exception for that exact path
	// and read failure.
	Excused []Entry
	// Blocking are unreadable entries no exception covers, plus any excepted
	// path whose shape changed.
	Blocking []Entry
	// Dormant are exceptions whose path is no longer unreadable.
	Dormant []Exception
}

// Evaluate applies exceptions to a scan. An exception covers an entry only when
// the destination, the exact relative path, the leftover-temp class and the
// recorded kind of read failure all match.
func Evaluate(fsys FS, root string, unreadable []Entry, exceptions []Exception) Verdict {
	if fsys == nil {
		fsys = OSFS{}
	}
	byPath := make(map[string]Exception, len(exceptions))
	for _, e := range exceptions {
		byPath[e.RelPath] = e
	}
	var v Verdict
	seen := map[string]bool{}
	for _, entry := range unreadable {
		seen[entry.RelPath] = true
		ex, ok := byPath[entry.RelPath]
		if ok && entry.Class == ClassLeftoverTemp && ex.Class == entry.Class && ex.Problem == entry.Problem {
			v.Excused = append(v.Excused, entry)
			continue
		}
		v.Blocking = append(v.Blocking, entry)
	}
	for _, ex := range exceptions {
		if seen[ex.RelPath] {
			continue
		}
		info, err := fsys.Lstat(filepath.Join(root, filepath.FromSlash(ex.RelPath)))
		switch {
		case err == nil && !info.Mode().IsRegular():
			// The excepted name now holds something else. The approval was for an
			// unreadable temporary file, not for this.
			v.Blocking = append(v.Blocking, Entry{
				RelPath: ex.RelPath, Class: ClassOther, BlobKind: ex.BlobKind, Problem: ProblemOther,
				Detail: "an approved path now holds a " + shapeName(info.Mode()) + " instead of a temporary file",
			})
		default:
			v.Dormant = append(v.Dormant, ex)
		}
	}
	return v
}

func shapeName(m fs.FileMode) string {
	switch {
	case m.IsDir():
		return "directory"
	case m&fs.ModeSymlink != 0:
		return "link"
	default:
		return "special file"
	}
}
