// Package recovery is the application service behind the `vrooli recovery`
// command group — the platform recovery floor's CLI surface for Baseline Modes.
//
// It is a thin orchestration over internal/baselinefloor: it resolves a
// scenario's working tree, stamps engagement timestamps, and composes the
// restore-point copy ladder with the floor-owned engagement manifest store. All
// the actual trusted-base primitives live in baselinefloor; this layer exists so
// the CLI handlers stay declarative and the orchestration is unit-testable
// without parsing argv.
//
// Like the floor it wraps, this package imports no scenario code and manages no
// running processes: the running-process truth lives in the scenarioruntime
// registry (P1), which the higher-level `git-control-tower baseline status` verb
// cross-references against the manifests this surface lists.
package recovery

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"
)

// Service composes the recovery-floor primitives for the CLI. Root is the
// repository root used to resolve a scenario's working tree when a caller does
// not pass an explicit source/destination; Store is the floor manifest store
// (production resolves it sudo-aware via baselinefloor.DefaultStore, tests inject
// a temp root). Clock is the time source for engagement timestamps — nil uses
// time.Now, tests pin it for deterministic golden output.
type Service struct {
	Root  string
	Store *baselinefloor.Store
	Clock func() time.Time
	// HomeDir resolves the operator home for the variant-aware data-dir query
	// (`recovery namespace`). Nil uses os.UserHomeDir; tests point it at a temp
	// tree. It is unused by the restore-point/manifest verbs.
	HomeDir func() (string, error)
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

// scenarioDir resolves a scenario's working tree under the repository root —
// <root>/scenarios/<scenario>. This is the default capture source and restore
// destination; callers pass an explicit path to override it.
func (s Service) scenarioDir(scenario string) string {
	return filepath.Join(s.Root, "scenarios", scenario)
}

// view wraps a manifest with the derived expiry fields the renderers and JSON
// consumers want, computed against now (pure — no registry lookup).
func (s Service) view(m baselinefloor.Manifest) EngagementView {
	now := s.now()
	out := EngagementView{Manifest: m, Expired: m.Expired(now)}
	if expiry, ok := m.ExpiresAt(); ok {
		out.ExpiresAt = &expiry
	}
	return out
}

// EngagementView is a manifest plus its derived idle-expiry, the shape every
// engagement-returning verb emits. The embedded Manifest flattens into the JSON
// object so the manifest fields sit alongside expiresAt/expired.
type EngagementView struct {
	baselinefloor.Manifest
	// ExpiresAt is the absolute idle-expiry (lastTouchedAt + ttl), omitted when
	// the engagement has no TTL (orchestrator-heartbeat mode).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// Expired reports whether the idle TTL has elapsed as of the service clock.
	Expired bool `json:"expired"`
}

// CaptureRequest takes a restore point of a scenario's working tree.
type CaptureRequest struct {
	Scenario string
	Slug     string
	// Source overrides the working-tree path; empty uses <root>/scenarios/<scenario>.
	Source string
	// NoReflink forces the portable deep-copy floor even where CoW is available
	// (deterministic copy stats; mainly for tests).
	NoReflink bool
}

// CaptureOutput reports a completed capture.
type CaptureOutput struct {
	Scenario         string                  `json:"scenario"`
	Slug             string                  `json:"slug"`
	Source           string                  `json:"source"`
	RestorePointPath string                  `json:"restorePointPath"`
	Stats            baselinefloor.CopyStats `json:"stats"`
}

// Capture copies the scenario working tree into the engagement's restore point,
// excluding build artifacts. It is idempotent (re-capture overlays).
func (s Service) Capture(req CaptureRequest) (CaptureOutput, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return CaptureOutput{}, err
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = s.scenarioDir(req.Scenario)
	}
	dest := s.Store.RestorePointPath(req.Scenario, req.Slug)
	stats, err := baselinefloor.Capture(source, dest, copyOpts(req.NoReflink))
	if err != nil {
		return CaptureOutput{}, err
	}
	return CaptureOutput{
		Scenario:         req.Scenario,
		Slug:             req.Slug,
		Source:           source,
		RestorePointPath: dest,
		Stats:            stats,
	}, nil
}

// RestoreRequest overlays a captured restore point back onto a scenario.
type RestoreRequest struct {
	Scenario string
	Slug     string
	// Dest overrides where the restore point is overlaid; empty uses the working tree.
	Dest      string
	NoReflink bool
}

// RestoreOutput reports a completed restore.
type RestoreOutput struct {
	Scenario         string                  `json:"scenario"`
	Slug             string                  `json:"slug"`
	RestorePointPath string                  `json:"restorePointPath"`
	Dest             string                  `json:"dest"`
	Stats            baselinefloor.CopyStats `json:"stats"`
}

// Restore overlays the engagement's restore point onto the destination — the
// sanctioned, git-free code-level undo.
func (s Service) Restore(req RestoreRequest) (RestoreOutput, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return RestoreOutput{}, err
	}
	dest := strings.TrimSpace(req.Dest)
	if dest == "" {
		dest = s.scenarioDir(req.Scenario)
	}
	src := s.Store.RestorePointPath(req.Scenario, req.Slug)
	stats, err := baselinefloor.Restore(src, dest, copyOpts(req.NoReflink))
	if err != nil {
		return RestoreOutput{}, err
	}
	return RestoreOutput{
		Scenario:         req.Scenario,
		Slug:             req.Slug,
		RestorePointPath: src,
		Dest:             dest,
		Stats:            stats,
	}, nil
}

// WriteRequest records (or refreshes) an engagement manifest. The restore-point
// path is derived from (scenario, slug) so it can never drift from where capture
// writes; timestamps are stamped by the service clock.
type WriteRequest struct {
	Scenario          string
	Slug              string
	Mode              string
	Variant           string
	TTL               time.Duration
	AmbientVar        string
	ShadowInstanceKey string
	Anchor            string
}

// WriteEngagement validates and persists the engagement manifest. A re-write
// preserves the original CreatedAt (so the engagement age is stable) and renews
// LastTouchedAt to now.
func (s Service) WriteEngagement(req WriteRequest) (EngagementView, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return EngagementView{}, err
	}
	mode := baselinefloor.Mode(strings.ToLower(strings.TrimSpace(req.Mode)))
	if !mode.Valid() {
		return EngagementView{}, fmt.Errorf("recovery: invalid mode %q (want shadow|live)", req.Mode)
	}
	variant := strings.TrimSpace(req.Variant)
	if variant == "" {
		if mode == baselinefloor.ModeShadow {
			variant = "shadow"
		} else {
			variant = "live"
		}
	}
	now := s.now()
	created := now
	if existing, err := s.Store.ReadManifest(req.Scenario, req.Slug); err == nil {
		created = existing.CreatedAt
	}
	shadowKey := strings.TrimSpace(req.ShadowInstanceKey)
	if shadowKey == "" && mode == baselinefloor.ModeShadow {
		shadowKey = req.Scenario + "@" + variant
	}
	m := baselinefloor.Manifest{
		Scenario:           req.Scenario,
		Slug:               req.Slug,
		Variant:            variant,
		Mode:               mode,
		RestorePointPath:   s.Store.RestorePointPath(req.Scenario, req.Slug),
		AnchorBaselineName: strings.TrimSpace(req.Anchor),
		AmbientVar:         strings.TrimSpace(req.AmbientVar),
		ShadowInstanceKey:  shadowKey,
		CreatedAt:          created,
		LastTouchedAt:      now,
		TTL:                baselinefloor.Duration(req.TTL),
	}
	if err := s.Store.WriteManifest(m); err != nil {
		return EngagementView{}, err
	}
	return s.view(m), nil
}

// Ref names a single engagement by (scenario, slug).
type Ref struct {
	Scenario string
	Slug     string
}

// ShowEngagement reads one engagement manifest.
func (s Service) ShowEngagement(ref Ref) (EngagementView, error) {
	if err := requireRef(ref.Scenario, ref.Slug); err != nil {
		return EngagementView{}, err
	}
	m, err := s.Store.ReadManifest(ref.Scenario, ref.Slug)
	if err != nil {
		return EngagementView{}, err
	}
	return s.view(m), nil
}

// ListOutput is the result of globbing every engagement manifest — the source
// for `git-control-tower baseline status`.
type ListOutput struct {
	Engagements []EngagementView `json:"engagements"`
}

// ListEngagements globs every engagement manifest under the cache root, sorted
// oldest-first, skipping any corrupt sibling.
func (s Service) ListEngagements() (ListOutput, error) {
	manifests, err := s.Store.ListManifests()
	if err != nil {
		return ListOutput{}, err
	}
	views := make([]EngagementView, 0, len(manifests))
	for _, m := range manifests {
		views = append(views, s.view(m))
	}
	return ListOutput{Engagements: views}, nil
}

// Touch renews an engagement lease (the touch-on-access path keeping a
// human-owned shadow alive between checks).
func (s Service) Touch(ref Ref) (EngagementView, error) {
	if err := requireRef(ref.Scenario, ref.Slug); err != nil {
		return EngagementView{}, err
	}
	m, err := s.Store.Touch(ref.Scenario, ref.Slug, s.now())
	if err != nil {
		return EngagementView{}, err
	}
	return s.view(m), nil
}

// SetTTLRequest adjusts an engagement's idle TTL.
type SetTTLRequest struct {
	Scenario string
	Slug     string
	TTL      time.Duration
}

// SetTTL adjusts an engagement's idle TTL (a non-positive value clears it,
// switching to orchestrator-heartbeat mode).
func (s Service) SetTTL(req SetTTLRequest) (EngagementView, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return EngagementView{}, err
	}
	m, err := s.Store.SetTTL(req.Scenario, req.Slug, req.TTL)
	if err != nil {
		return EngagementView{}, err
	}
	return s.view(m), nil
}

// CleanOutput reports a torn-down engagement.
type CleanOutput struct {
	Scenario      string `json:"scenario"`
	Slug          string `json:"slug"`
	EngagementDir string `json:"engagementDir"`
}

// Clean removes an entire engagement directory (restore point + manifest). It is
// idempotent: tearing down a missing engagement is not an error.
func (s Service) Clean(ref Ref) (CleanOutput, error) {
	if err := requireRef(ref.Scenario, ref.Slug); err != nil {
		return CleanOutput{}, err
	}
	dir := s.Store.EngagementDir(ref.Scenario, ref.Slug)
	if err := s.Store.Clean(ref.Scenario, ref.Slug); err != nil {
		return CleanOutput{}, err
	}
	return CleanOutput{Scenario: ref.Scenario, Slug: ref.Slug, EngagementDir: dir}, nil
}

// copyOpts builds the restore-point copy options: nil (defaults: gitignore-aligned
// excludes + reflink fast path) unless the caller forced the deep-copy floor.
func copyOpts(noReflink bool) *baselinefloor.CopyOptions {
	if !noReflink {
		return nil
	}
	return &baselinefloor.CopyOptions{Exclude: baselinefloor.DefaultExcludes(), Reflink: false}
}

func requireRef(scenario, slug string) error {
	if strings.TrimSpace(scenario) == "" {
		return fmt.Errorf("recovery: scenario is required")
	}
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("recovery: slug is required")
	}
	return nil
}
