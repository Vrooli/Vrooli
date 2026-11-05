package graph

import (
	"context"

	"golang.org/x/tools/go/packages"
)

// LoadOptions carries the knobs the Service needs to vary loader
// behaviour. Kept narrow so the seam stays a one-method interface.
type LoadOptions struct {
	// IncludeVendor enables walking vendor/ directories and the module
	// cache. Default (false) matches REQ-P1-003.
	IncludeVendor bool
	// Profile selects the loader mode. Full is the compatibility default.
	Profile ExtractionProfile
	// PackagePatterns narrows the go/packages query. Empty means ./....
	PackagePatterns []string
	// EnvironmentFingerprint captures loader-affecting process settings. The
	// production composition root supplies it; the graph package remains
	// independent of process-global environment reads.
	EnvironmentFingerprint string
}

// PackagesLoader is the production-vs-test seam wrapping
// golang.org/x/tools/go/packages.Load. The Service calls Load and is
// agnostic to whether the implementation is the real loader
// (PackagesLoaderImpl) or a FakeLoader from mocks/.
//
// seam: production wires loader_packages.NewPackagesLoader; tests wire
// graph/mocks.FakeLoader. Single method; do not extend without lifting
// a corresponding test fake in lockstep.
type PackagesLoader interface {
	Load(ctx context.Context, modulePath string, opts LoadOptions) ([]*packages.Package, error)
}
