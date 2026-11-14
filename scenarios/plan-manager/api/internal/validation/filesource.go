package validation

import (
	"context"
	"os"
	"path/filepath"

	planmodel "plan-manager/internal/planmodel"
)

// StatFunc is the filesystem-probe seam (os.Stat by default; faked in tests).
type StatFunc func(path string) (os.FileInfo, error)

// FileResolver is the dependency-free production ReferenceResolver. It resolves
// CODE and DOC references by file existence relative to a repo root: a present
// file is RESOLVED, an absent one is MISSING (moved/deleted). REQ references are
// passed through unchanged (requirement-id resolution needs a richer index;
// code-facts is the soft-dep upgrade that resolves all kinds structurally).
//
// This is the honest local floor: it does not pretend to resolve what it cannot,
// and it captures the single most important signal for staleness — whether the
// referenced location still exists.
type FileResolver struct {
	Root string
	Stat StatFunc
}

// NewFileResolver constructs a FileResolver rooted at root (the repo root), using
// os.Stat. A nil/empty root resolves relative to the process working directory.
func NewFileResolver(root string) FileResolver {
	return FileResolver{Root: root, Stat: os.Stat}
}

var _ ReferenceResolver = FileResolver{}

func (r FileResolver) exists(target string) bool {
	stat := r.Stat
	if stat == nil {
		stat = os.Stat
	}
	path := target
	if r.Root != "" && !filepath.IsAbs(target) {
		path = filepath.Join(r.Root, target)
	}
	_, err := stat(path)
	return err == nil
}

// Resolve annotates the reference with a filesystem-existence resolution.
func (r FileResolver) Resolve(_ context.Context, ref planmodel.Reference) (planmodel.Reference, error) {
	switch ref.Kind {
	case planmodel.ReferenceCode, planmodel.ReferenceDoc:
		if r.exists(ref.Target) {
			ref.Resolution = planmodel.ResolutionResolved
		} else {
			ref.Resolution = planmodel.ResolutionMissing
			ref.Note = "referenced location not found on disk"
		}
	default:
		// REQ and others: not filesystem-resolvable; leave unspecified rather
		// than claim a resolution we cannot back (honest, not degraded).
	}
	return ref, nil
}

// ExistenceStaleness is the dependency-free production StalenessComputer. A
// reference whose location is gone is DEFINITELY_STALE (the model's
// moved/deleted tier); a present location is FRESH. The service refines present
// references to LIGHTLY_STALE through git-sourced per-reference drift when a
// regression anchor SHA is available. This floor honestly reports FRESH for
// present files rather than fabricating a change magnitude it cannot measure.
type ExistenceStaleness struct {
	resolver FileResolver
}

// NewExistenceStaleness constructs the floor StalenessComputer over a FileResolver.
func NewExistenceStaleness(r FileResolver) ExistenceStaleness {
	return ExistenceStaleness{resolver: r}
}

var _ StalenessComputer = ExistenceStaleness{}

// Compute reports DEFINITELY_STALE when the location is gone, else FRESH.
func (s ExistenceStaleness) Compute(_ context.Context, ref planmodel.Reference) (planmodel.StalenessTier, float64, error) {
	if ref.Kind != planmodel.ReferenceCode && ref.Kind != planmodel.ReferenceDoc {
		return planmodel.StalenessFresh, 0, nil
	}
	if s.resolver.exists(ref.Target) {
		return planmodel.StalenessFresh, 0, nil
	}
	return planmodel.StalenessDefinitelyStale, 1, nil
}
