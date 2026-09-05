package autofix

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireParses asserts that src is valid, parseable Go.
func requireParses(t *testing.T, src string) {
	t.Helper()
	_, err := parser.ParseFile(token.NewFileSet(), "fixture.go", src, parser.ParseComments)
	require.NoError(t, err, "fixer produced unparseable Go:\n%s", src)
}

const domainSchemaSQL = `CREATE TABLE IF NOT EXISTS widgets (
  id TEXT PRIMARY KEY
);
`

func TestEnsureEmbed_PreviewScaffoldsSchemaGo(t *testing.T) {
	root := t.TempDir()
	writeAPIFile(t, root, "internal/widgets/schema.sql", domainSchemaSQL)

	candidates, err := Preview(root, []string{RuleEnsureSchemasUnwire})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	c := candidates[0]
	require.Equal(t, RuleEnsureSchemasUnwire, c.RuleID)
	require.Equal(t, filepath.Join(root, "api", "internal", "widgets", "schema.go"), c.FilePath)
	require.Empty(t, c.Before, "scaffold creates a new file")
	require.Contains(t, c.After, "package widgets")
	require.Contains(t, c.After, "//go:embed schema.sql")
	require.Contains(t, c.After, "func Schema() string")
	requireParses(t, c.After)
}

func TestEnsureEmbed_ApplyIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeAPIFile(t, root, "internal/widgets/schema.sql", domainSchemaSQL)
	schemaGo := filepath.Join(root, "api", "internal", "widgets", "schema.go")

	applied, err := Apply(root, []string{RuleEnsureSchemasUnwire})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)

	first, err := os.ReadFile(schemaGo)
	require.NoError(t, err)
	requireParses(t, string(first))

	// Second apply: a sibling embed now exists, so nothing more to scaffold.
	second, err := Apply(root, []string{RuleEnsureSchemasUnwire})
	require.NoError(t, err)
	require.Empty(t, second, "second apply must be a no-op")

	after, err := os.ReadFile(schemaGo)
	require.NoError(t, err)
	require.Equal(t, string(first), string(after), "second apply must not change the scaffolded file")
}

func TestEnsureEmbed_SkipsWhenAlreadyEmbedded(t *testing.T) {
	root := t.TempDir()
	writeAPIFile(t, root, "internal/widgets/schema.sql", domainSchemaSQL)
	writeAPIFile(t, root, "internal/widgets/schema.go",
		"package widgets\n\nimport _ \"embed\"\n\n//go:embed schema.sql\nvar schemaSQL string\n")

	candidates, err := Preview(root, []string{RuleEnsureSchemasUnwire})
	require.NoError(t, err)
	require.Empty(t, candidates, "an already-embedded domain schema must not be re-scaffolded")
}

func TestEnsureEmbed_SkipsSystemHomeAndEmptySchema(t *testing.T) {
	root := t.TempDir()
	// System home is cross-cutting, never a per-domain scaffold target.
	writeAPIFile(t, root, "internal/database/system.sql", domainSchemaSQL)
	// A schema with no CREATE TABLE has nothing to apply.
	writeAPIFile(t, root, "internal/empty/schema.sql", "-- no tables here\n")

	candidates, err := Preview(root, []string{RuleEnsureSchemasUnwire})
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestCoveredCodesMatchRegistry(t *testing.T) {
	// Every covered code must be a code a fixer can actually act on; this guards
	// against AutofixAvailable lying about coverage the registry doesn't have.
	require.True(t, CoveredCodes[RuleDBRowsNotClosed])
	require.True(t, CoveredCodes[RuleEnsureSchemasUnwire])
	require.Len(t, CoveredCodes, 2)
}
