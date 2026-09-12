package sketch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
)

// Revision is an immutable snapshot of the authored page, not a second mutable
// page document. Publication receipts exclude prepared but unapplied revisions.
// ParentHash names the content from which this immutable revision was created.
type Revision struct {
	ContentHash string    `json:"contentHash"`
	ParentHash  string    `json:"parentHash,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Page        []byte    `json:"pageBytes"`
	Current     bool      `json:"-"`
}

type applyManifest struct {
	Expected string `json:"expected"`
	Next     string `json:"next"`
}

var validHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func revisionPath(page, hash string) string {
	return filepath.Join("designs", page, "revisions", hash+".json")
}
func manifestPath(page string) string { return filepath.Join("designs", page, "apply.json") }

func (s *Store) checkpoint(step string) error {
	if s.afterPublish != nil {
		return s.afterPublish(step)
	}
	return nil
}

func readRevision(root *os.Root, page, hash string) (Revision, error) {
	if !validHash.MatchString(hash) {
		return Revision{}, errors.New("invalid revision hash")
	}
	raw, err := root.ReadFile(revisionPath(page, hash))
	if err != nil {
		return Revision{}, err
	}
	var revision Revision
	if err := json.Unmarshal(raw, &revision); err != nil {
		return Revision{}, err
	}
	snapshot, err := decodeSnapshot(revision.Page)
	if err != nil {
		return Revision{}, err
	}
	if revision.ContentHash != hash || snapshot.ContentHash != hash {
		return Revision{}, errors.New("immutable revision content does not match its identity")
	}
	return revision, nil
}

func (s *Store) writeRevision(root *os.Root, page, hash, parent string, raw []byte) error {
	if _, err := readRevision(root, page, hash); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	revision := Revision{ContentHash: hash, ParentHash: parent, CreatedAt: time.Now().UTC(), Page: raw}
	encoded, err := json.MarshalIndent(revision, "", "  ")
	if err != nil {
		return err
	}
	// The same per-page OS lock protects both revision creation and publication.
	return storage.WriteFileAtomicInRoot(root, revisionPath(page, hash), encoded, 0600)
}

// recoverPending is called under the page lock. Only the recorded expected
// page can advance. An unrelated manual edit is reported as conflict and the
// manifest remains available for inspection; recovery never overwrites it.
func (s *Store) recoverPending(root *os.Root, page, name string) (bool, error) {
	raw, err := root.ReadFile(manifestPath(page))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var pending applyManifest
	if err := json.Unmarshal(raw, &pending); err != nil {
		return false, err
	}
	if !validHash.MatchString(pending.Expected) || !validHash.MatchString(pending.Next) {
		return false, errors.New("invalid apply manifest revision")
	}
	revision, err := readRevision(root, page, pending.Next)
	if err != nil {
		return false, err
	}
	if _, err := readRevision(root, page, pending.Expected); err != nil {
		return false, err
	}
	currentBytes, err := root.ReadFile(name)
	if err != nil {
		return false, err
	}
	current, err := decodeSnapshot(currentBytes)
	if err != nil {
		return false, err
	}
	if current.ContentHash != pending.Next {
		if current.ContentHash != pending.Expected {
			return false, &ConflictError{Expected: pending.Expected, Current: current.ContentHash}
		}
		info, err := root.Stat(name)
		if err != nil {
			return false, err
		}
		if err := storage.WriteFileAtomicInRoot(root, name, revision.Page, info.Mode().Perm()); err != nil {
			return false, err
		}
	}
	if err := s.checkpoint("page"); err != nil {
		return false, err
	}
	if err := s.markApplied(root, page, pending.Next); err != nil {
		return false, err
	}
	if err := root.Remove(manifestPath(page)); err != nil {
		return false, err
	}
	if err := s.checkpoint("complete"); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Recover(scenario, page string) (Snapshot, error) {
	root, name, err := s.openPages(scenario, page)
	if err != nil {
		return Snapshot{}, err
	}
	defer root.Close()
	lock, err := root.OpenFile(filepath.Join("pages", "."+page+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Snapshot{}, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, false)
	if err != nil {
		return Snapshot{}, err
	}
	defer release()
	changed, err := s.recoverPending(root, page, name)
	if err != nil {
		return Snapshot{}, err
	}
	raw, err := root.ReadFile(name)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot, err := decodeSnapshot(raw)
	snapshot.Changed = changed
	return snapshot, err
}

func (s *Store) markApplied(root *os.Root, page, hash string) error {
	path := filepath.Join("designs", page, "applied", hash+".json")
	if _, err := root.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return storage.WriteFileAtomicInRoot(root, path, []byte(`{"applied":true}`), 0600)
}

// History lists immutable content revisions with completed publication
// evidence. Prepared revisions are excluded; reapplying a previous content
// revision reuses its identity without rewriting its original metadata.
func (s *Store) History(scenario, page string) ([]Revision, error) {
	root, name, err := s.openPages(scenario, page)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	raw, err := root.ReadFile(name)
	if err != nil {
		return nil, err
	}
	current, err := decodeSnapshot(raw)
	if err != nil {
		return nil, err
	}
	folder, err := root.Open(filepath.Join("designs", page, "applied"))
	if errors.Is(err, os.ErrNotExist) {
		return []Revision{{ContentHash: current.ContentHash, Page: raw, Current: true}}, nil
	}
	if err != nil {
		return nil, err
	}
	defer folder.Close()
	entries, err := folder.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	history := make([]Revision, 0, len(entries))
	currentFound := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		hash := strings.TrimSuffix(entry.Name(), ".json")
		revision, err := readRevision(root, page, hash)
		if err != nil {
			return nil, err
		}
		revision.Current = revision.ContentHash == current.ContentHash
		if revision.Current {
			currentFound = true
		}
		history = append(history, revision)
	}
	if !currentFound {
		history = append(history, Revision{ContentHash: current.ContentHash, Page: raw, Current: true})
	}
	sort.Slice(history, func(i, j int) bool {
		if history[i].CreatedAt.Equal(history[j].CreatedAt) {
			return history[i].ContentHash < history[j].ContentHash
		}
		return history[i].CreatedAt.After(history[j].CreatedAt)
	})
	return history, nil
}
