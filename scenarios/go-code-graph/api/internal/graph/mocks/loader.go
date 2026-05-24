// Package mocks holds test fakes for the graph domain seams. Lives
// outside _test.go files so sibling packages (handlers/graph_test,
// cli/domains/graph_test) can import the same fake without copying.
//
// Production code never imports this package; the testutil drift gate
// enforces that no_prod_import_test asserts it.
package mocks

import (
	"context"

	"golang.org/x/tools/go/packages"

	intgraph "go-code-graph/internal/graph"
)

// FakeLoader is the canned PackagesLoader for tests. Set LoadFunc to
// the behaviour the test wants; the zero value returns (nil, nil)
// which is a legitimate "empty module" response.
type FakeLoader struct {
	LoadFunc func(ctx context.Context, scenarioPath string, opts intgraph.LoadOptions) ([]*packages.Package, error)
}

// Load satisfies intgraph.PackagesLoader.
func (f *FakeLoader) Load(ctx context.Context, scenarioPath string, opts intgraph.LoadOptions) ([]*packages.Package, error) {
	if f.LoadFunc != nil {
		return f.LoadFunc(ctx, scenarioPath, opts)
	}
	return nil, nil
}

// Compile-time assertion: FakeLoader satisfies the production seam.
var _ intgraph.PackagesLoader = (*FakeLoader)(nil)
