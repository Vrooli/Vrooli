package graph

import (
	"context"
	"fmt"

	"golang.org/x/tools/go/packages"
)

// PackagesLoaderImpl is the production PackagesLoader implementation.
// It wraps golang.org/x/tools/go/packages.Load with the hard-coded
// load mode required by REQ-P0-001:
//
//	NeedFiles | NeedImports | NeedTypes | NeedSyntax |
//	NeedTypesInfo | NeedName | NeedDeps
//
// The mode is intentionally fixed — a caller asking for "less" would
// produce a non-deterministic subset and break byte-stability of the
// extracted graph. No caching: every call yields a fresh
// packages.Config.
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

// Load runs packages.Load with the fixed mode rooted at scenarioPath.
// IncludeVendor is currently advisory — the underlying loader follows
// the module's own vendor/ presence; the flag is reserved for REQ-P1-003
// to gate post-load filtering once that requirement lands.
func (l *PackagesLoaderImpl) Load(ctx context.Context, scenarioPath string, opts LoadOptions) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode:    loadMode,
		Dir:     scenarioPath,
		Tests:   false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load(%q): %w", scenarioPath, err)
	}
	return pkgs, nil
}

// Compile-time assertion: PackagesLoaderImpl satisfies PackagesLoader.
var _ PackagesLoader = (*PackagesLoaderImpl)(nil)
