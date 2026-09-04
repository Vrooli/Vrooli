package contracts

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func testSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "schemas", "program-contract.schema.json"))
	require.NoError(t, err)
	compiler := jsonschema.NewCompiler()
	path := "program-contract.schema.json"
	require.NoError(t, compiler.AddResource(path, bytes.NewReader(data)))
	schema, err := compiler.Compile(path)
	require.NoError(t, err)
	return schema
}

func validContractData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "scenarios", "program-runtime", ".vrooli", "program-runtime", "setpoint-read.json"))
	require.NoError(t, err)
	return data
}

func TestIndexListIsStableAndSorted(t *testing.T) {
	index := NewIndex()
	index.contracts = []Contract{{Scenario: "a", Name: "a"}, {Scenario: "z", Name: "z"}}
	got := index.List()
	// Load sorts filesystem results; List returns a copy and does not mutate it.
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].Scenario)
}

func TestReadContractKeepsSchemaInvalidEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"broken"}`), 0o600))
	got := readContract("demo", path, testSchema(t))
	require.Equal(t, "demo", got.Scenario)
	require.NotEmpty(t, got.ValidationError)
}

func TestReadContractLoadsSiblingSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.json")
	require.NoError(t, os.WriteFile(path, validContractData(t), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.py"), []byte("print('source-backed')\n"), 0o600))

	got := readContract("demo", path, testSchema(t))
	require.Empty(t, got.ValidationError)
	require.Equal(t, "print('source-backed')\n", got.Source)
}

func TestReadContractReportsMissingSiblingSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.json")
	require.NoError(t, os.WriteFile(path, validContractData(t), 0o600))

	got := readContract("demo", path, testSchema(t))
	require.Contains(t, got.ValidationError, "source file")
	require.Contains(t, got.ValidationError, "demo.py")
}

func TestIndexRefreshWithoutLoadedFilesIsUnchanged(t *testing.T) {
	index := NewIndex()
	changed, err := index.Refresh(t.TempDir())
	require.NoError(t, err)
	require.False(t, changed)
}

func TestRefreshTracksSiblingSourceMtime(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := NewIndex()
	require.NoError(t, index.Load(root))
	sourcePath := filepath.Join(root, "scenarios", "program-runtime", ".vrooli", "program-runtime", "setpoint-read.py")
	original, ok := index.mtimes[sourcePath]
	require.True(t, ok, "loaded index must track sibling source mtimes")
	index.mtimes[sourcePath] = original - 1
	changed, err := index.Refresh(root)
	require.NoError(t, err)
	require.True(t, changed, "a source-only mtime change must trigger refresh")
	require.Equal(t, original, index.mtimes[sourcePath])
}

func TestIndexCoverageUsesTightestValidSubset(t *testing.T) {
	index := NewIndex()
	index.contracts = []Contract{
		{ID: "demo.wide", BindingIDs: []string{"demo/ops/read", "demo/ops/list"}},
		{ID: "demo.tight", BindingIDs: []string{"demo/ops/read"}},
		{ID: "demo.broken", BindingIDs: []string{"demo/ops/read"}, ValidationError: "invalid"},
	}
	got, ok := index.CoverageFor([]string{"demo/ops/read"})
	require.True(t, ok)
	require.Equal(t, "demo.tight", got.ID)
	require.Equal(t, "demo.tight", index.CoveredBy([]string{"demo/ops/read"}))
	_, ok = index.CoverageFor([]string{"demo/ops/missing"})
	require.False(t, ok)
	_, ok = index.Get("missing", "program")
	require.False(t, ok)
}

func TestLoadIndexesRepositoryContracts(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := NewIndex()
	require.NoError(t, index.Load(root))
	require.NotEmpty(t, index.List())
	require.Equal(t, 35, len(index.List()))
	contract, ok := index.Get("prompt-manager", "skill-set-read")
	require.True(t, ok)
	require.Empty(t, contract.ValidationError)
}
