package routing_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestThinRouterHasNoProviderSpecificImports is the OT-P2-002 architectural
// guard: the router core must route purely off registered descriptors (scope +
// the generic web-shaped classifier label), never off knowledge of a specific
// provider scenario. It must therefore never import any provider scenario's
// generated code — in particular web-search. If a future change reaches for a
// web-search-specific import to drive auto-routing, this test fails and the
// thin-router invariant is restored by going back through the descriptor.
func TestThinRouterHasNoProviderSpecificImports(t *testing.T) {
	// Provider scenario module path fragments the router must never depend on.
	// search-hub's OWN generated types (registry/routing) are allowed; a leaf
	// provider's (web-search, cli-health, …) are not.
	forbidden := []string{
		"packages/proto/gen/go/web-search",
		"packages/proto/gen/go/cli-health",
		"packages/proto/gen/go/knowledge-observatory",
	}

	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				require.NotContains(t, path, bad,
					"%s imports provider-specific %q — the thin-router invariant forbids per-provider code; route off the descriptor instead", name, path)
			}
		}
	}
}
