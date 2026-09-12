package baselinefloor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

// Mode is the execution mode an engagement records.
type Mode string

const (
	// ModeShadow runs a second isolated instance from the working tree while live
	// keeps running from the restore-point copy (the default safe path).
	ModeShadow Mode = "shadow"
	// ModeLive edits the live instance directly, protected only by the restore
	// point — chosen by the decision tree for the trusted base, lifecycle/registry
	// changes, non-duplicable singletons, or non-namespaceable stores.
	ModeLive Mode = "live"
)

// Valid reports whether m is a known mode.
func (m Mode) Valid() bool {
	return m == ModeShadow || m == ModeLive
}

// Manifest is the floor-owned, recovery-critical engagement intent persisted as
// engagement.json beside the restore point. It is deliberately NOT stored inside
// git-control-tower so the floor can roll back a broken git-control-tower; and it
// is a plain glob-discoverable file, so there is no central index to corrupt
// (Baseline Modes §P2 "Engagement-state persistence").
//
// It records intent, not running-process truth: ports/owner/heartbeat live in the
// scenarioruntime registry (P1). The two are cross-referenced by ShadowInstanceKey.
type Manifest struct {
	// Scenario is the target scenario slug (e.g. "swarm-manager").
	Scenario string `json:"scenario"`
	// Slug is the engagement slug — the directory is baseline-<slug>.
	Slug string `json:"slug"`
	// Variant is the shadow variant for a shadow engagement ("shadow"), or "live".
	Variant string `json:"variant"`
	// Mode is the recorded execution mode.
	Mode Mode `json:"mode"`
	// RestorePointPath is the absolute path to the captured restore point.
	RestorePointPath string `json:"restorePointPath"`
	// AnchorBaselineName is the git-control-tower baseline record the engagement
	// diffs against (the validation anchor), empty if none.
	AnchorBaselineName string `json:"anchorBaselineName,omitempty"`
	// AmbientVar is the VROOLI_SHADOW_SCENARIOS value set for this engagement so
	// nested CLI calls auto-route to the shadow (P1.5), empty in live mode.
	AmbientVar string `json:"ambientVar,omitempty"`
	// ShadowInstanceKey is the registry instance key for the shadow
	// ("<scenario>@<variant>"), empty in live mode. It cross-references the
	// running-process registry.
	ShadowInstanceKey string `json:"shadowInstanceKey,omitempty"`
	// CreatedAt is when the engagement was opened.
	CreatedAt time.Time `json:"createdAt"`
	// LastTouchedAt is the lease-renewal timestamp: every `baseline check`/`status`
	// touches it so a human-owned shadow with no heartbeater survives between
	// checks. TTL is measured from this, not CreatedAt.
	LastTouchedAt time.Time `json:"lastTouchedAt"`
	// TTL is the idle cap. Zero means no TTL (an orchestrator-owned engagement
	// protected by a real heartbeat instead). Rendered human-friendly ("3h").
	TTL Duration `json:"ttl"`
}

// Validate checks the required fields a manifest must carry to be recoverable.
func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Scenario) == "" {
		return fmt.Errorf("baselinefloor: manifest missing scenario")
	}
	if strings.TrimSpace(m.Slug) == "" {
		return fmt.Errorf("baselinefloor: manifest missing slug")
	}
	if !m.Mode.Valid() {
		return fmt.Errorf("baselinefloor: manifest invalid mode %q", m.Mode)
	}
	if strings.TrimSpace(m.RestorePointPath) == "" {
		return fmt.Errorf("baselinefloor: manifest missing restorePointPath")
	}
	return nil
}

// ExpiresAt returns the absolute expiry (LastTouchedAt + TTL) and whether a TTL
// is set. A zero TTL means the engagement does not expire on idle (false).
func (m Manifest) ExpiresAt() (time.Time, bool) {
	if m.TTL <= 0 {
		return time.Time{}, false
	}
	return m.LastTouchedAt.Add(m.TTL.AsDuration()), true
}

// Expired reports whether the engagement's idle TTL has elapsed as of now. An
// engagement with no TTL is never idle-expired (orchestrators heartbeat instead).
func (m Manifest) Expired(now time.Time) bool {
	expiry, ok := m.ExpiresAt()
	if !ok {
		return false
	}
	return now.After(expiry)
}

// WriteManifest persists m to its engagement.json, creating the engagement
// directory as needed. The write is atomic (temp file + rename) so a crash mid-
// write never leaves a half-written, unrecoverable manifest.
func (s *Store) WriteManifest(m Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}
	dir := s.EngagementDir(m.Scenario, m.Slug)
	if err := os.MkdirAll(dir, tuning.PermGroupDir); err != nil {
		return fmt.Errorf("baselinefloor: mkdir engagement %q: %w", dir, err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("baselinefloor: marshal manifest: %w", err)
	}
	data = append(data, '\n')

	final := s.ManifestPath(m.Scenario, m.Slug)
	if err := config.WriteOwnedFileAtomic(final, data, tuning.PermSecret); err != nil {
		return fmt.Errorf("baselinefloor: commit manifest %q: %w", final, err)
	}
	return nil
}

// ReadManifest loads the engagement.json for (scenario, slug).
func (s *Store) ReadManifest(scenario, slug string) (Manifest, error) {
	return readManifestFile(s.ManifestPath(scenario, slug))
}

func readManifestFile(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("baselinefloor: read manifest %q: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("baselinefloor: parse manifest %q: %w", path, err)
	}
	return m, nil
}

// ListManifests globs every engagement manifest under the cache root —
// ~/.cache/vrooli/*/baseline-*/engagement.json — and returns them sorted by
// CreatedAt (oldest first). This is the source for `baseline status`: there is no
// central index, so a corrupt or partially-written manifest can never hide the
// others. An unreadable manifest is skipped rather than failing the whole list.
func (s *Store) ListManifests() ([]Manifest, error) {
	pattern := filepath.Join(s.root, "*", engagementDirPrefix+"*", manifestFile)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("baselinefloor: glob manifests: %w", err)
	}
	out := make([]Manifest, 0, len(matches))
	for _, path := range matches {
		m, readErr := readManifestFile(path)
		if readErr != nil {
			// A half-written or corrupt sibling must not blind status to the rest.
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// Touch renews the engagement lease by setting LastTouchedAt to now and
// persisting — the touch-on-access path `baseline check`/`status` use so a
// human-owned shadow survives between heartbeats. Returns the updated manifest.
func (s *Store) Touch(scenario, slug string, now time.Time) (Manifest, error) {
	m, err := s.ReadManifest(scenario, slug)
	if err != nil {
		return Manifest{}, err
	}
	m.LastTouchedAt = now
	if err := s.WriteManifest(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// SetTTL adjusts an engagement's idle TTL (the `baseline status --set-ttl` path)
// and persists. A zero or negative ttl clears it (orchestrator-heartbeat mode).
func (s *Store) SetTTL(scenario, slug string, ttl time.Duration) (Manifest, error) {
	m, err := s.ReadManifest(scenario, slug)
	if err != nil {
		return Manifest{}, err
	}
	if ttl < 0 {
		ttl = 0
	}
	m.TTL = Duration(ttl)
	if err := s.WriteManifest(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
