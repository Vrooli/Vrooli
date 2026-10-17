package baselinefloor

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/tuning"
)

// captureRecordFile records the identity of the latest capture of an
// engagement and the state of its shared-source snapshot.
const captureRecordFile = "capture.json"

// sharedSourcesDir holds the repo-shaped copy of the shared sources a
// scenario build reaches outside its own directory.
const sharedSourcesDir = "shared-sources"

// Shared-source snapshot states.
const (
	// SharedSourcesFrozen means every shared source the build needs was
	// copied; the serving tree builds against the copy.
	SharedSourcesFrozen = "frozen"
	// SharedSourcesUnfrozen means the snapshot was skipped or could not be
	// completed (capture --allow-unfrozen); the serving tree links the
	// repository's current shared sources.
	SharedSourcesUnfrozen = "unfrozen"
)

// SharedRoot is one repository path (directory or file) a scenario build
// reads outside the scenario directory.
type SharedRoot struct {
	// Path is repo-relative and slash-separated (packages/api-core/storage,
	// go.mod, ...).
	Path string `json:"path"`
	// Reason names the reference that requires it (go:<module dir>,
	// file:<package>, alias:<config file>, requires:<package>).
	Reason string `json:"reason"`
	// Bytes is the size copied into the snapshot.
	Bytes int64 `json:"bytes,omitempty"`
}

// SharedSources records the shared-source snapshot of a capture.
type SharedSources struct {
	Status string       `json:"status"`
	Reason string       `json:"reason,omitempty"`
	Roots  []SharedRoot `json:"roots,omitempty"`
	Bytes  int64        `json:"bytes,omitempty"`
	// ProtoArtifactID identifies the Proto artifact whose generated outputs
	// were frozen, so a compile error can name the frozen capture.
	ProtoArtifactID string `json:"protoArtifactId,omitempty"`
}

// Frozen reports whether the serving tree can build from the snapshot.
func (s SharedSources) Frozen() bool {
	return s.Status == SharedSourcesFrozen
}

// CaptureRecord is the floor-owned identity of an engagement's latest
// capture, written beside engagement.json. CaptureID is the serving-tree
// generation: a recapture (baseline start --replace) gets a new ID, so the
// next serving start materializes a new tree instead of reusing a copy of
// the previous capture.
type CaptureRecord struct {
	CaptureID     string        `json:"captureId"`
	CapturedAt    time.Time     `json:"capturedAt"`
	Source        string        `json:"source"`
	SharedSources SharedSources `json:"sharedSources"`
}

// NewCaptureID returns a sortable, unique capture identity.
func NewCaptureID(now time.Time) string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return now.UTC().Format("20060102T150405.000000000Z") + "-" + strconv.Itoa(os.Getpid())
	}
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(suffix[:])
}

// CaptureRecordPath resolves capture.json inside an engagement.
func (s *Store) CaptureRecordPath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), captureRecordFile)
}

// SharedSourcesPath resolves the repo-shaped shared-source snapshot inside an
// engagement: ~/.cache/vrooli/<scenario>/baseline-<slug>/shared-sources/.
func (s *Store) SharedSourcesPath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), sharedSourcesDir)
}

// WriteCaptureRecord persists the capture record atomically.
func (s *Store) WriteCaptureRecord(scenario, slug string, record CaptureRecord) error {
	if strings.TrimSpace(record.CaptureID) == "" {
		return fmt.Errorf("baselinefloor: capture record missing captureId")
	}
	dir := s.EngagementDir(scenario, slug)
	if err := os.MkdirAll(dir, tuning.PermGroupDir); err != nil {
		return fmt.Errorf("baselinefloor: mkdir engagement %q: %w", dir, err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("baselinefloor: marshal capture record: %w", err)
	}
	if err := config.WriteOwnedFileAtomic(s.CaptureRecordPath(scenario, slug), append(data, '\n'), tuning.PermSecret); err != nil {
		return fmt.Errorf("baselinefloor: write capture record: %w", err)
	}
	return nil
}

// ReadCaptureRecord loads capture.json. ok is false when the engagement was
// captured before capture records existed; callers then fall back to the
// manifest's CreatedAt as the generation and treat shared sources as
// unfrozen.
func (s *Store) ReadCaptureRecord(scenario, slug string) (CaptureRecord, bool, error) {
	data, err := os.ReadFile(s.CaptureRecordPath(scenario, slug))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CaptureRecord{}, false, nil
		}
		return CaptureRecord{}, false, fmt.Errorf("baselinefloor: read capture record: %w", err)
	}
	var record CaptureRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return CaptureRecord{}, false, fmt.Errorf("baselinefloor: parse capture record: %w", err)
	}
	return record, true, nil
}

// sharedSourceExcludes are skipped inside a frozen root. Unlike a restore
// point, a frozen root KEEPS its generated outputs (dist/, gen/, generated/):
// the serving build consumes them and nothing regenerates a frozen root.
// Installed dependencies are never copied: pnpm injects file: dependencies
// into the consumer, and Go reads modules from the module cache.
var sharedSourceExcludes = map[string]struct{}{
	"node_modules":            {},
	".git":                    {},
	"coverage":                {},
	".turbo":                  {},
	".next":                   {},
	"__pycache__":             {},
	".build-fingerprint.json": {},
}

// CaptureSharedSources copies each root from repoRoot into dest, keeping the
// repository layout (dest/<root.Path>). dest is replaced as a whole: the copy
// is made in a staging sibling and swapped in, so a reader never sees a mix
// of two captures and a failed copy leaves the previous snapshot intact.
// Returned roots carry their copied sizes.
func CaptureSharedSources(repoRoot, dest string, roots []SharedRoot, reflink bool) ([]SharedRoot, int64, error) {
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, tuning.PermGroupDir); err != nil {
		return nil, 0, fmt.Errorf("baselinefloor: shared sources mkdir: %w", err)
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(dest)+".capturing-")
	if err != nil {
		return nil, 0, fmt.Errorf("baselinefloor: shared sources staging: %w", err)
	}
	keepStaging := false
	defer func() {
		if !keepStaging {
			_ = removeTree(staging)
		}
	}()
	ordered := append([]SharedRoot(nil), roots...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	var total int64
	opts := CopyOptions{Exclude: sharedSourceExcludes, Ignore: floorIgnorer, Reflink: reflink}
	for i, root := range ordered {
		rel, err := cleanRepoRel(root.Path)
		if err != nil {
			return nil, 0, err
		}
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		dst := filepath.Join(staging, filepath.FromSlash(rel))
		info, err := os.Stat(src)
		if err != nil {
			return nil, 0, fmt.Errorf("baselinefloor: shared source %s (%s): %w", rel, root.Reason, err)
		}
		var bytes int64
		if info.IsDir() {
			stats, err := CopyTree(src, dst, opts)
			if err != nil {
				return nil, 0, fmt.Errorf("baselinefloor: copy shared source %s: %w", rel, err)
			}
			if err := os.Chmod(dst, info.Mode().Perm()|0o700); err != nil {
				return nil, 0, fmt.Errorf("baselinefloor: shared source mode %s: %w", rel, err)
			}
			bytes = stats.BytesCopied
		} else {
			_, n, err := copyRegularFile(src, dst, info.Mode().Perm()|0o600, reflink)
			if err != nil {
				return nil, 0, fmt.Errorf("baselinefloor: copy shared source %s: %w", rel, err)
			}
			bytes = n
		}
		ordered[i].Path = rel
		ordered[i].Bytes = bytes
		total += bytes
	}
	if err := os.Chmod(staging, tuning.PermGroupDir); err != nil {
		return nil, 0, fmt.Errorf("baselinefloor: shared sources mode: %w", err)
	}
	if err := replaceDir(staging, dest); err != nil {
		return nil, 0, err
	}
	keepStaging = true
	return ordered, total, nil
}

// replaceDir moves staging into place at dest, removing any previous dest.
func replaceDir(staging, dest string) error {
	if _, err := os.Lstat(dest); err == nil {
		old := dest + ".superseded-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := os.Rename(dest, old); err != nil {
			return fmt.Errorf("baselinefloor: move previous %q aside: %w", dest, err)
		}
		defer func() { _ = removeTree(old) }()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("baselinefloor: stat %q: %w", dest, err)
	}
	if err := os.Rename(staging, dest); err != nil {
		return fmt.Errorf("baselinefloor: publish %q: %w", dest, err)
	}
	return nil
}

// cleanRepoRel validates a repo-relative root path: relative, clean, inside
// the repository, and not the repository itself.
func cleanRepoRel(raw string) (string, error) {
	rel := strings.TrimSpace(filepath.ToSlash(raw))
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == "" || clean == "." || filepath.IsAbs(filepath.FromSlash(rel)) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("baselinefloor: shared source path %q must be a relative path inside the repository", raw)
	}
	return clean, nil
}

// removeTree removes a directory tree, first making read-only directories
// writable: a copied module or build output can carry 0555 directories,
// which os.RemoveAll cannot empty.
func removeTree(dir string) error {
	err := os.RemoveAll(dir)
	if err == nil {
		return nil
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && d.IsDir() {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	return os.RemoveAll(dir)
}
