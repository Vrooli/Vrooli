package discovery

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"data-backup-manager/internal/sysmounts"
)

// Service is the application surface the discovery handler depends on. It owns
// the scan → filter → rank pipeline for both suggestion kinds and the dismissal
// write. It holds no durable state beyond the DismissalStore.
type Service interface {
	// ListTargetSuggestions returns ranked, de-duplicated, non-dismissed target
	// suggestions (sources worth protecting that are not yet registered).
	ListTargetSuggestions(ctx context.Context) ([]TargetSuggestion, error)

	// ListDestinationSuggestions returns ranked, non-dismissed destination
	// suggestions (mounted volumes worth backing up to).
	ListDestinationSuggestions(ctx context.Context) ([]DestinationSuggestion, error)

	// DismissSuggestion hides a suggestion permanently by its stable id.
	// Idempotent: re-dismissing an already-dismissed id is a no-op success.
	DismissSuggestion(ctx context.Context, id string) (bool, error)
}

// Deps wires the seams the service composes.
type Deps struct {
	Volumes      VolumeScanner
	Sources      TargetSourceScanner
	Targets      TargetCatalog
	Destinations DestinationCatalog
	Protected    ProtectedPaths
	Dismissals   DismissalStore
}

type service struct {
	deps Deps
}

// NewService constructs the production Service.
func NewService(d Deps) Service { return &service{deps: d} }

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) ListTargetSuggestions(ctx context.Context) ([]TargetSuggestion, error) {
	candidates, err := s.deps.Sources.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan target sources: %w", err)
	}
	existing, err := s.deps.Targets.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read target catalog: %w", err)
	}

	registeredKeys := make(map[string]struct{}, len(existing))
	registeredLocators := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		registeredKeys[ownerNameKey(t.Owner, t.Name)] = struct{}{}
		registeredLocators[cleanPath(t.Locator)] = struct{}{}
	}

	out := make([]TargetSuggestion, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := registeredKeys[ownerNameKey(c.Owner, c.Name)]; ok {
			continue
		}
		if _, ok := registeredLocators[cleanPath(c.Locator)]; ok {
			continue
		}
		id := targetSuggestionID(c.Locator)
		dismissed, derr := s.deps.Dismissals.IsDismissed(ctx, id)
		if derr != nil {
			return nil, fmt.Errorf("check dismissal %s: %w", id, derr)
		}
		if dismissed {
			continue
		}
		out = append(out, TargetSuggestion{ID: id, TargetCandidate: c})
	}

	// Platform state ("vrooli") first, then larger first, then locator for a
	// fully deterministic order.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ap, bp := a.Owner == platformOwner, b.Owner == platformOwner
		if ap != bp {
			return ap
		}
		if a.ApproxBytes != b.ApproxBytes {
			return a.ApproxBytes > b.ApproxBytes
		}
		return a.Locator < b.Locator
	})
	return out, nil
}

func (s *service) ListDestinationSuggestions(ctx context.Context) ([]DestinationSuggestion, error) {
	volumes, err := s.deps.Volumes.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan volumes: %w", err)
	}
	protected, err := s.deps.Protected.ProtectedPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve protected paths: %w", err)
	}
	existing, err := s.deps.Destinations.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read destination catalog: %w", err)
	}

	cleanProtected := make([]string, 0, len(protected))
	for _, p := range protected {
		if p = cleanPath(p); p != "" {
			cleanProtected = append(cleanProtected, p)
		}
	}
	usedLocations := make(map[string]struct{}, len(existing))
	for _, d := range existing {
		usedLocations[cleanPath(d.Location)] = struct{}{}
	}

	out := make([]DestinationSuggestion, 0, len(volumes))
	for _, v := range volumes {
		// Read-only volumes can't hold a kopia repo — never a destination.
		if v.ReadOnly {
			continue
		}
		location := cleanPath(v.Mountpoint)
		if location == "" {
			continue
		}
		if _, ok := usedLocations[location]; ok {
			continue
		}
		id := destSuggestionID(location)
		dismissed, derr := s.deps.Dismissals.IsDismissed(ctx, id)
		if derr != nil {
			return nil, fmt.Errorf("check dismissal %s: %w", id, derr)
		}
		if dismissed {
			continue
		}
		separateOK := !overlapsAny(location, cleanProtected)
		out = append(out, DestinationSuggestion{
			ID:             id,
			Label:          destinationLabel(v),
			Location:       v.Mountpoint,
			Class:          v.Class,
			FreeBytes:      v.FreeBytes,
			TotalBytes:     v.TotalBytes,
			Removable:      v.Removable,
			SeparateRootOK: separateOK,
			Rationale:      destinationRationale(v, separateOK),
		})
	}

	rankDestinations(out)
	return out, nil
}

func (s *service) DismissSuggestion(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, ErrInvalidDiscovery{Field: "id", Reason: "required"}
	}
	// The id encodes which kind it is only via its hash input, which we don't
	// reverse; record the dismissal kind-agnostically. The kind column is
	// informational (e.g. for future audit), so we tag it "suggestion".
	if err := s.deps.Dismissals.Dismiss(ctx, id, "suggestion"); err != nil {
		return false, fmt.Errorf("dismiss %s: %w", id, err)
	}
	return true, nil
}

const (
	platformOwner = "vrooli"
)

// rankDestinations orders suggestions: separate-root-safe before flagged, then
// removable drives first, then fixed volumes (largest free first), network
// last. Stable so equal entries keep scan order.
func rankDestinations(ds []DestinationSuggestion) {
	classRank := func(c DriveClass) int {
		switch c {
		case sysmounts.ClassRemovable:
			return 0
		case sysmounts.ClassFixed:
			return 1
		case sysmounts.ClassNetwork:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.SeparateRootOK != b.SeparateRootOK {
			return a.SeparateRootOK
		}
		if ra, rb := classRank(a.Class), classRank(b.Class); ra != rb {
			return ra < rb
		}
		if a.FreeBytes != b.FreeBytes {
			return a.FreeBytes > b.FreeBytes
		}
		return a.Location < b.Location
	})
}

// ownerNameKey is the catalog dedupe key for a target.
func ownerNameKey(owner, name string) string { return owner + "\x00" + name }

// cleanPath normalizes a path for comparison; empty input stays empty.
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

// overlapsAny reports whether loc overlaps any protected path — i.e. loc equals,
// is contained by, or contains a protected path (Contract Decision D4).
func overlapsAny(loc string, protected []string) bool {
	for _, p := range protected {
		if pathsOverlap(loc, p) {
			return true
		}
	}
	return false
}

// pathsOverlap reports whether a == b, a is under b, or b is under a.
func pathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return a == b || within(a, b) || within(b, a)
}

// within reports whether child is the same as, or nested under, parent. The
// root "/" is a special case: every absolute path is within it, but "/"+sep
// would be "//", so we test the bare prefix there.
func within(child, parent string) bool {
	if child == parent {
		return true
	}
	sep := string(filepath.Separator)
	if parent == sep {
		return strings.HasPrefix(child, sep)
	}
	return strings.HasPrefix(child, parent+sep)
}
