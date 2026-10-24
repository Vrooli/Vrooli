// Package descriptorimage provides a shared, restart-free view of the
// generated protobuf descriptor image.
package descriptorimage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var ErrNoSnapshot = errors.New("descriptorimage: no known-good snapshot")

// Config defines the descriptor and the generated inputs whose changes should
// trigger a reload. Manifest paths are watched even though they are not part
// of the descriptor bytes: binding registries derive their surface from both.
type Config struct {
	DescriptorPath string
	ManifestPaths  []string
}

// Snapshot is immutable after construction. Consumers should retain one
// snapshot for the duration of a request and obtain another for the next
// request. Files is never mutated by Source after publication.
type Snapshot struct {
	Files         *protoregistry.Files
	Descriptor    *descriptorpb.FileDescriptorSet
	Digest        string
	Generation    uint64
	LoadedAt      time.Time
	ArtifactMTime time.Time
	stamps        []fileStamp
	raw           []byte
}

// Source loads and refreshes one immutable descriptor snapshot. Snapshot calls
// only take the reload mutex when a watched-file stamp changed; unchanged reads
// use an atomic pointer after portable os.Stat calls.
type Source struct {
	descriptorPath string
	watchedPaths   []string
	current        atomic.Pointer[Snapshot]
	reloadMu       sync.Mutex
	lastErrMu      sync.RWMutex
	lastErr        error
	lastFailureAt  time.Time
}

type fileStamp struct {
	path    string
	info    os.FileInfo
	missing bool
}

// New constructs a source. It does not read the filesystem; the first
// Snapshot call performs the initial load.
func New(config Config) (*Source, error) {
	descriptorPath := filepath.Clean(strings.TrimSpace(config.DescriptorPath))
	if descriptorPath == "." || descriptorPath == "" {
		return nil, fmt.Errorf("descriptorimage: descriptor path is required")
	}
	paths := make([]string, 0, 1+len(config.ManifestPaths))
	paths = append(paths, descriptorPath)
	paths = append(paths, config.ManifestPaths...)
	paths = normalizePaths(paths)
	return &Source{descriptorPath: descriptorPath, watchedPaths: paths}, nil
}

// NewForRepo watches the global descriptor and all scenario CLI manifests.
func NewForRepo(repoRoot string) (*Source, error) {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	if repoRoot == "." || repoRoot == "" {
		return nil, fmt.Errorf("descriptorimage: repository root is required")
	}
	manifestPaths, err := scenarioManifestPaths(repoRoot)
	if err != nil {
		return nil, err
	}
	return New(Config{
		DescriptorPath: filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"),
		ManifestPaths:  manifestPaths,
	})
}

func scenarioManifestPaths(repoRoot string) ([]string, error) {
	root := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("descriptorimage: read scenarios: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(root, entry.Name(), "cli", "manifest.json"))
	}
	return paths, nil
}

func normalizePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "." || path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// Snapshot returns the current known-good snapshot. A failed reload returns
// the previous snapshot together with the parse/stat error. The first failed
// load returns a typed error because there is no safe surface to serve.
func (s *Source) Snapshot() (*Snapshot, error) {
	if s == nil {
		return nil, fmt.Errorf("descriptorimage: nil source")
	}
	current := s.current.Load()
	stamps := s.statWatchedPaths()
	if current != nil && sameStamps(current.stamps, stamps) {
		return current, nil
	}

	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	current = s.current.Load()
	stamps = s.statWatchedPaths()
	if current != nil && sameStamps(current.stamps, stamps) {
		return current, nil
	}

	next, err := s.load(stamps, current)
	if err != nil {
		s.recordFailure(err)
		if current != nil {
			return current, err
		}
		return nil, fmt.Errorf("%w: %v", ErrNoSnapshot, err)
	}
	s.clearFailure()
	s.current.Store(next)
	return next, nil
}

func (s *Source) load(stamps []fileStamp, current *Snapshot) (*Snapshot, error) {
	raw, err := os.ReadFile(s.descriptorPath)
	if err != nil {
		return nil, fmt.Errorf("read descriptor image %q: %w", s.descriptorPath, err)
	}
	set := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(raw, set); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor image %q: %w", s.descriptorPath, err)
	}
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, fmt.Errorf("build descriptor registry: %w", err)
	}
	digest := sha256.Sum256(raw)
	generation := uint64(1)
	if current != nil {
		generation = current.Generation + 1
	}
	artifactMTime := time.Time{}
	for _, stamp := range stamps {
		if !stamp.missing && stamp.info.ModTime().After(artifactMTime) {
			artifactMTime = stamp.info.ModTime()
		}
	}
	return &Snapshot{
		Files:         files,
		Descriptor:    set,
		Digest:        "sha256:" + hex.EncodeToString(digest[:]),
		Generation:    generation,
		LoadedAt:      time.Now().UTC(),
		ArtifactMTime: artifactMTime,
		stamps:        cloneStamps(stamps),
		raw:           append([]byte(nil), raw...),
	}, nil
}

func (s *Source) statWatchedPaths() []fileStamp {
	stamps := make([]fileStamp, 0, len(s.watchedPaths))
	for _, path := range s.watchedPaths {
		info, err := os.Stat(path)
		stamps = append(stamps, fileStamp{path: path, info: info, missing: err != nil})
	}
	return stamps
}

func sameStamps(a, b []fileStamp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].path != b[i].path || a[i].missing != b[i].missing {
			return false
		}
		if a[i].missing {
			continue
		}
		if !os.SameFile(a[i].info, b[i].info) || a[i].info.Size() != b[i].info.Size() || !a[i].info.ModTime().Equal(b[i].info.ModTime()) {
			return false
		}
	}
	return true
}

func cloneStamps(in []fileStamp) []fileStamp {
	return append([]fileStamp(nil), in...)
}

func (s *Source) recordFailure(err error) {
	s.lastErrMu.Lock()
	s.lastErr = err
	s.lastFailureAt = time.Now().UTC()
	s.lastErrMu.Unlock()
}

func (s *Source) clearFailure() {
	s.lastErrMu.Lock()
	s.lastErr = nil
	s.lastFailureAt = time.Time{}
	s.lastErrMu.Unlock()
}

// LastReloadError returns the most recent failed reload, if any.
func (s *Source) LastReloadError() error {
	s.lastErrMu.RLock()
	defer s.lastErrMu.RUnlock()
	return s.lastErr
}

// LastReloadFailureAt returns the UTC time of the most recent failed reload.
func (s *Source) LastReloadFailureAt() time.Time {
	s.lastErrMu.RLock()
	defer s.lastErrMu.RUnlock()
	return s.lastFailureAt
}

// WatchedPaths returns a copy of the inputs whose stamps trigger reloads.
func (s *Source) WatchedPaths() []string { return append([]string(nil), s.watchedPaths...) }

// DescriptorPath returns the source's canonical descriptor input.
func (s *Source) DescriptorPath() string { return s.descriptorPath }

// LoadWithRetry performs bounded initial loading for process startup.
func (s *Source) LoadWithRetry(attempts int, delay time.Duration) (*Snapshot, error) {
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		var snapshot *Snapshot
		snapshot, err = s.Snapshot()
		if err == nil {
			return snapshot, nil
		}
		if attempt+1 < attempts && delay > 0 {
			time.Sleep(delay)
		}
	}
	return nil, err
}

// RangeFiles visits the immutable descriptor files in a snapshot.
func (s *Snapshot) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	if s == nil || s.Files == nil || f == nil {
		return
	}
	s.Files.RangeFiles(f)
}

// DescriptorBytes returns a copy of the serialized descriptor for adapters
// that need a bytes-based API. The published snapshot retains its own copy.
func (s *Snapshot) DescriptorBytes() []byte {
	if s == nil {
		return nil
	}
	return append([]byte(nil), s.raw...)
}
