// Package storage is music-tools' persistence seam for audio bytes. It keeps
// audio bytes OUT of SQLite and out of the repo: blobs live under the api-core
// storage substrate (namespace-aware, outside-repo by default), addressed by
// opaque keys. Outputs are user-owned — a request may redirect a result to an
// arbitrary host path instead of the managed blob store.
//
// Owned generated outputs stream through a bounded reader. Domain callers validate
// the expected media format; arbitrary upload/decoded-duration inspection is separate.
package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/blobstore"
	corestorage "github.com/vrooli/api-core/storage"
)

// Store persists music blobs under api-core storage and routes outputs to
// either the managed blob store or a caller-supplied host path.
type Store struct {
	blobs       blobstore.BlobStore
	root        string
	mu          sync.Mutex
	entries     map[string]Entry
	budgetBytes uint64
}

type Entry struct {
	Key         string
	SHA256      string
	Owner       string
	Size        uint64
	LastAccess  time.Time
	Regenerable bool
	Protected   bool
	// EvictionPriority orders eligible entries before age. Lower values are
	// reclaimed first; domain owners use this for explicit lifecycle policy.
	EvictionPriority int
}

// New resolves the blob root for the given storage namespace (use
// corestorage.ScenarioNamespace("music-tools"), which is shadow/variant aware)
// and returns a filesystem-backed Store rooted there. The root lives under the
// platform data class, outside the repo.
func New(namespace string) (*Store, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: corestorage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: resolver: %w", err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return nil, fmt.Errorf("storage: resolve namespace %q: %w", namespace, err)
	}
	root := filepath.Join(paths.DataDir, "blobs")
	if err := corestorage.EnsureDirectory(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create blob root: %w", err)
	}
	store := NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.restoreEntries(); err != nil {
		return nil, err
	}
	return store, nil
}

// NewWithBlobStore wraps a BlobStore backed by the supplied filesystem root.
// Tests may wrap a temporary filesystem store to inject substrate failures.
func NewWithBlobStore(bs blobstore.BlobStore, root string) *Store {
	return &Store{blobs: bs, root: root, entries: map[string]Entry{}}
}

func (s *Store) SetBudgetBytes(max uint64) { s.mu.Lock(); s.budgetBytes = max; s.mu.Unlock() }

func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastAccess.Before(out[j].LastAccess) })
	return out
}

const MaxOwnedBytes int64 = 128 << 20

// ownedReader bounds bytes before BlobStore's atomic commit, including streams
// without Content-Length. It checks cancellation for every read.
type ownedReader struct {
	ctx  context.Context
	r    io.Reader
	size int64
}

func (r *ownedReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	p = p[:min(len(p), int(MaxOwnedBytes-r.size+1))]
	n, err := r.r.Read(p)
	r.size += int64(n)
	if r.size > MaxOwnedBytes {
		return 0, fmt.Errorf("generated audio exceeds 128 MiB")
	}
	return n, err
}

func ownershipKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "ownership/" + hex.EncodeToString(sum[:]) + ".json"
}

func (s *Store) PutOwned(ctx context.Context, key, owner string, r io.Reader, mime string, regenerable bool) error {
	digest := sha256.New()
	stream := &ownedReader{ctx: ctx, r: io.TeeReader(r, digest)}
	if err := s.Put(ctx, key, stream, mime); err != nil {
		// Shared BlobStore can commit bytes then fail its MIME metadata write.
		// Account such retained bytes conservatively without declaring a take ready.
		if path, pathErr := s.PathForKey(key); pathErr == nil {
			if info, statErr := os.Lstat(path); statErr == nil && info.Mode().IsRegular() {
				s.mu.Lock()
				s.entries[key] = Entry{Key: key, Owner: owner, Size: uint64(info.Size()), Protected: true, LastAccess: info.ModTime()}
				s.mu.Unlock()
			}
		}
		return err
	}
	entry := Entry{Key: key, Owner: owner, SHA256: hex.EncodeToString(digest.Sum(nil)), Size: uint64(stream.size), LastAccess: time.Now().UTC(), Regenerable: regenerable, Protected: true}
	s.mu.Lock()
	s.entries[key] = entry
	s.mu.Unlock()
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err = s.blobs.Put(ctx, ownershipKey(key), bytes.NewReader(raw), "application/json"); err != nil {
		return fmt.Errorf("retain audio ownership metadata: %w", err)
	}
	return nil
}

// restoreEntries replays immutable ownership receipts. All loaded blobs begin
// protected until the authoritative pool reapplies its persisted take state.
func (s *Store) restoreEntries() error {
	dir := filepath.Join(s.root, "ownership")
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		files = nil
		err = nil
	}
	if err != nil {
		return err
	}
	if len(files) > 100000 {
		return fmt.Errorf("ownership inventory exceeds supported 100000 entries")
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 16384 {
			return fmt.Errorf("invalid ownership receipt %s", file.Name())
		}
		raw, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}
		var entry Entry
		if err = json.Unmarshal(raw, &entry); err != nil {
			return err
		}
		if err = validateKey(entry.Key); err != nil {
			return err
		}
		if ownershipKey(entry.Key) != "ownership/"+file.Name() {
			return fmt.Errorf("ownership key mismatch")
		}
		stat, err := os.Lstat(filepath.Join(s.root, filepath.FromSlash(entry.Key)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() || uint64(stat.Size()) != entry.Size {
			return fmt.Errorf("owned blob size/type mismatch: %s", entry.Key)
		}
		entry.Protected = true
		s.entries[entry.Key] = entry
	}
	// Historical generated outputs without receipts remain accounted and protected.
	// Only explicit authoritative pool reconciliation may make them reclaimable.
	count := 0
	err = filepath.WalkDir(filepath.Join(s.root, "out"), func(path string, d os.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".meta") {
			return nil
		}
		count++
		if count > 100000 {
			return fmt.Errorf("legacy inventory exceeds 100000 entries")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular legacy generated blob")
		}
		key, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		key = filepath.ToSlash(key)
		if _, ok := s.entries[key]; !ok {
			s.entries[key] = Entry{Key: key, Size: uint64(info.Size()), Protected: true, LastAccess: info.ModTime()}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) Touch(key string) {
	s.mu.Lock()
	if entry, ok := s.entries[key]; ok {
		entry.LastAccess = time.Now().UTC()
		s.entries[key] = entry
	}
	s.mu.Unlock()
}

func (s *Store) SetProtected(key string, protected bool) {
	s.mu.Lock()
	if entry, ok := s.entries[key]; ok {
		entry.Protected = protected
		s.entries[key] = entry
	}
	s.mu.Unlock()
}

// ProtectAndCommit serializes reservation commit against eligible deletion. If
// deletion won first, reservation fails instead of granting missing audio. Only
// confirmed physical absence returns false without error; access/I/O/type failures
// retain their cause and must not retire the authoritative pool take.
func (s *Store) ProtectAndCommit(key string, commit func() error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok {
		return false, nil
	}
	path, err := s.PathForKey(key)
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		delete(s.entries, key)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verify reservation blob %q: %w", key, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("reservation blob %q is not a regular file", key)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rc, _, err := s.blobs.Get(ctx, key)
	if errors.Is(err, os.ErrNotExist) {
		delete(s.entries, key)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open reservation blob %q: %w", key, err)
	}
	if err := rc.Close(); err != nil {
		return false, fmt.Errorf("close reservation blob %q: %w", key, err)
	}
	protected := entry.Protected
	entry.Protected = true
	s.entries[key] = entry
	if err := commit(); err != nil {
		entry.Protected = protected
		s.entries[key] = entry
		return false, err
	}
	return true, nil
}

func sameEntry(a, b Entry) bool {
	return a.Key == b.Key && a.Owner == b.Owner && a.SHA256 == b.SHA256 && a.Size == b.Size && a.LastAccess.Equal(b.LastAccess) && a.EvictionPriority == b.EvictionPriority
}

// DeleteEligible is the only preview/eviction deletion primitive. Observation,
// identity/protection verification and deletion share one lock with reservation.
func (s *Store) DeleteEligible(ctx context.Context, expected Entry) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.entries[expected.Key]
	if !ok || current.Protected || !current.Regenerable || !sameEntry(current, expected) {
		return false, nil
	}
	if err := s.deleteLocked(ctx, current.Key); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SetEvictionPriority(key string, priority int) {
	s.mu.Lock()
	if entry, ok := s.entries[key]; ok {
		entry.EvictionPriority = priority
		s.entries[key] = entry
	}
	s.mu.Unlock()
}

// Evict reclaims the oldest eligible entries until total storage is within the
// configured budget. Protected entries (including reserved takes) are never
// removed. Callers order candidates by domain policy before invoking this
// method; the storage seam does not infer taste or pool state.
func (s *Store) Evict(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	budget := s.budgetBytes
	entries := make([]Entry, 0, len(s.entries))
	var total uint64
	for _, entry := range s.entries {
		total += entry.Size
		if entry.Regenerable && !entry.Protected {
			entries = append(entries, entry)
		}
	}
	s.mu.Unlock()
	if budget == 0 || total <= budget {
		return nil, nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].EvictionPriority != entries[j].EvictionPriority {
			return entries[i].EvictionPriority < entries[j].EvictionPriority
		}
		return entries[i].LastAccess.Before(entries[j].LastAccess)
	})
	var removed []string
	for _, entry := range entries {
		if total <= budget {
			break
		}
		removedCurrent, err := s.DeleteEligible(ctx, entry)
		if err != nil {
			return removed, err
		}
		if !removedCurrent {
			continue
		}
		total -= entry.Size
		removed = append(removed, entry.Key)
	}
	return removed, nil
}

// Root returns the absolute blob root (diagnostics only).
func (s *Store) Root() string { return s.root }

// PathForKey returns the filesystem path for a managed blob key. It is a
// read-only diagnostics seam for the owner cleanup provider; deletion still
// goes through Store.Delete so the blob-store abstraction remains authoritative.
func (s *Store) PathForKey(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

// Put stores bytes under key with the given MIME type.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.blobs.Put(ctx, key, r, mime); err != nil {
		return fmt.Errorf("storage: put %q: %w", key, err)
	}
	s.mu.Lock()
	if entry, ok := s.entries[key]; ok {
		entry.LastAccess = time.Now().UTC()
		s.entries[key] = entry
	}
	s.mu.Unlock()
	return nil
}

// Get opens the blob stored under key, returning its reader and MIME type.
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := validateKey(key); err != nil {
		return nil, "", err
	}
	rc, mime, err := s.blobs.Get(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("storage: get %q: %w", key, err)
	}
	s.Touch(key)
	return rc, mime, nil
}

// Delete removes the blob under key (no error if already absent).
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteLocked(ctx, key)
}

func (s *Store) deleteLocked(ctx context.Context, key string) error {
	if err := s.blobs.Delete(ctx, key); err != nil {
		// A substrate can remove audio and then fail removing its sidecar.
		// Preserve the error while keeping availability truthful if audio is gone.
		if path, pathErr := s.PathForKey(key); pathErr == nil {
			if _, statErr := os.Lstat(path); errors.Is(statErr, os.ErrNotExist) {
				delete(s.entries, key)
			}
		}
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	// Audio is removed even when subsequent ownership-receipt deletion fails.
	// Keep the failed receipt for diagnosis, but never advertise absent audio.
	delete(s.entries, key)
	if err := s.blobs.Delete(ctx, ownershipKey(key)); err != nil {
		return fmt.Errorf("delete ownership receipt: %w", err)
	}
	return nil
}

// OutputTarget selects where an operation result is written. Exactly one field
// must be set. BlobKey stores the result in the managed blob store (the
// default, returns the key as the ref). LocalPath writes the result directly to
// a host path the user owns (the per-request override, returns the path).
type OutputTarget struct {
	BlobKey   string
	LocalPath string
}

// Validate ensures exactly one target is set.
func (t OutputTarget) Validate() error {
	switch {
	case t.BlobKey != "" && t.LocalPath != "":
		return fmt.Errorf("storage: output target has both blob key and local path")
	case t.BlobKey == "" && t.LocalPath == "":
		return fmt.Errorf("storage: output target is empty")
	case t.BlobKey != "":
		return validateKey(t.BlobKey)
	default:
		if !filepath.IsAbs(t.LocalPath) {
			return fmt.Errorf("storage: local output path must be absolute: %q", t.LocalPath)
		}
		return nil
	}
}

// Write routes a result to its target and returns a reference (the blob key, or
// the absolute local path). Local writes are atomic (temp file + rename) and
// create parent directories, so a partially written file is never observable.
func (s *Store) Write(ctx context.Context, target OutputTarget, r io.Reader, mime string) (ref string, err error) {
	if err := target.Validate(); err != nil {
		return "", err
	}
	if target.BlobKey != "" {
		if err := s.Put(ctx, target.BlobKey, r, mime); err != nil {
			return "", err
		}
		return target.BlobKey, nil
	}
	if err := writeFileAtomic(target.LocalPath, r); err != nil {
		return "", fmt.Errorf("storage: write local output: %w", err)
	}
	return target.LocalPath, nil
}

// validateKey rejects keys that escape the blob root or are empty. The
// filesystem store re-checks, but validating here gives a uniform error and
// guards in-memory stores too.
func validateKey(key string) error {
	k := strings.TrimSpace(filepath.ToSlash(key))
	if k == "" {
		return fmt.Errorf("storage: key is required")
	}
	if strings.HasPrefix(k, "/") || k == ".." || strings.HasPrefix(k, "../") || strings.Contains(k, "/../") {
		return fmt.Errorf("storage: invalid key %q", key)
	}
	return nil
}

func writeFileAtomic(path string, r io.Reader) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".imgtmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := io.Copy(tmp, r); err != nil {
		cleanup()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
