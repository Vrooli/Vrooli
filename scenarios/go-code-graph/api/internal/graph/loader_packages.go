package graph

import (
	"context"
	"fmt"

	"golang.org/x/tools/go/packages"
)

// PackagesLoaderImpl is the production PackagesLoader implementation.
// It wraps golang.org/x/tools/go/packages.Load with the full compatibility
// mode required by REQ-P0-001 and supports explicit lighter profiles.
//
//	NeedFiles | NeedImports | NeedTypes | NeedSyntax |
//	NeedTypesInfo | NeedName | NeedDeps
//
// Full mode remains fixed and is the default. Lighter modes are intentionally
// opt-in because they omit semantic facts by contract.
type PackagesLoaderImpl struct{}

// NewPackagesLoader returns the production PackagesLoader.
func NewPackagesLoader() PackagesLoader { return &PackagesLoaderImpl{} }

// loadMode is the only mode this loader uses. Exported as a var (not a
// const) only because it's a bitwise OR; treat it as immutable.
var loadMode = packages.NeedFiles |
	packages.NeedImports |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedName |
	packages.NeedDeps

var structuralLoadMode = packages.NeedFiles |
	packages.NeedImports |
	packages.NeedSyntax |
	packages.NeedName

// Load runs packages.Load with the fixed mode rooted at modulePath.
// IncludeVendor is honored by Service.Extract via a post-load directory
// filter (filterVendorPackages); the packages loader itself runs with
// its default vendor behavior so the wire shape of returned packages is
// uniform across both branches.
func (l *PackagesLoaderImpl) Load(ctx context.Context, modulePath string, opts LoadOptions) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode:    modeForProfile(opts.Profile),
		Dir:     modulePath,
		Tests:   opts.Profile.normalized() == ExtractionProfileFull,
	}
	patterns := opts.PackagePatterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load(%q): %w", modulePath, err)
	}
	return pkgs, nil
}

func modeForProfile(profile ExtractionProfile) packages.LoadMode {
	if profile.normalized() == ExtractionProfileStructural {
		return structuralLoadMode
	}
	return loadMode
}

// Compile-time assertion: PackagesLoaderImpl satisfies PackagesLoader.
var _ PackagesLoader = (*PackagesLoaderImpl)(nil)
