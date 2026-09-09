package contracts

import (
	"bytes"
	"encoding/json"
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

func TestReadContractLoadsRubricFields(t *testing.T) { // [REQ:PRT-P1-012]
	document := map[string]any{}
	require.NoError(t, json.Unmarshal(validContractData(t), &document))
	document["verbs"] = []any{"gather", "validate"}
	document["memory"] = map[string]any{"scope": "team:demo", "reads_in": "collect", "writes_in": "report", "entry_kinds": []any{"lesson"}, "scope_input": "scope", "writes_when": "status == ok", "read_scopes": []any{"team:shared"}}
	document["fixtures"] = []any{map[string]any{"id": "live", "inputs": map[string]any{}, "expect": map[string]any{"status": []any{"ok"}}, "requires": []any{"program-runtime"}}}
	document["bindings"] = []any{map[string]any{"id": "demo/read/list", "effect": "read", "optional": true, "via": "binding", "note": "optional read"}}
	document["learning"] = map[string]any{"note_kinds": map[string]any{"domain-note": map[string]any{"type": "object"}}}
	document["inputs"].(map[string]any)["query"] = map[string]any{"type": "string", "free_text": true}
	document["assumptions"] = []any{"The fixture data is available."}
	document["invariants"] = []any{"The envelope is printed exactly once."}
	document["outputs"].(map[string]any)["signals"] = map[string]any{"score": "measured"}
	document["budget"].(map[string]any)["async"] = true
	document["budget"].(map[string]any)["async_reason"] = "the read may be long"

	dir := t.TempDir()
	path := filepath.Join(dir, "demo.json")
	data, err := json.Marshal(document)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.py"), []byte("learn.task()\nlearn.note('domain-note', {})\nprint('source-backed')\n"), 0o600))

	got := readContract("demo", path, testSchema(t))
	require.Empty(t, got.ValidationError)
	require.Equal(t, []string{"gather", "validate"}, got.Verbs)
	require.Equal(t, "team:demo", got.Memory.Scope)
	require.Equal(t, []string{"program-runtime"}, got.Fixtures[0].Requires)
	require.Equal(t, "read", got.Bindings[0].Effect)
	require.True(t, got.Bindings[0].Optional)
	require.Equal(t, []string{"The envelope is printed exactly once."}, got.Invariants)
	require.Equal(t, "measured", got.Signals["score"])
	require.True(t, got.Async)
	require.Equal(t, "the read may be long", got.AsyncReason)
	require.Equal(t, []string{"domain-note"}, got.Learning.NoteKinds)
	require.Equal(t, []string{"note", "task"}, got.Learning.Verbs)
	require.True(t, got.Learning.UsesMemory)
	require.True(t, got.Learning.FreeTextInputsWithoutKey)
}

func TestReadContractReportsMissingSiblingSource(t *testing.T) { // [REQ:PRT-P1-012]
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.json")
	require.NoError(t, os.WriteFile(path, validContractData(t), 0o600))

	got := readContract("demo", path, testSchema(t))
	require.True(t, got.SourceMissing)
	require.Empty(t, got.ValidationError)
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
	contract, ok := index.Get("prompt-manager", "skill-set-read")
	require.True(t, ok)
	require.Empty(t, contract.ValidationError)
	for _, entry := range []struct{ scenario, name string }{
		{"program-runtime", "improvement-evidence"},
		{"visited-tracker", "attention-select"},
		{"agent-manager", "investigation-evidence"},
	} {
		contract, ok := index.Get(entry.scenario, entry.name)
		require.True(t, ok, "missing composition contract %s.%s", entry.scenario, entry.name)
		require.Empty(t, contract.ValidationError)
	}
}

func TestLoadIndexesAllContractsWithoutParseErrors(t *testing.T) { // [REQ:PRT-P1-012]
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := NewIndex()
	require.NoError(t, index.Load(root))
	require.NotEmpty(t, index.List())
	for _, contract := range index.List() {
		require.NotEmpty(t, contract.ID)
		require.NotEmpty(t, contract.Name)
	}
}

func TestResolveInputsAppliesDefaultsAndRejectsInvalidValues(t *testing.T) {
	contract := Contract{Inputs: map[string]InputSpec{
		"name":    {Type: "string", Required: true},
		"enabled": {Type: "boolean", Default: json.RawMessage(`false`)},
		"limit":   {Type: "integer", Default: json.RawMessage(`3`)},
	}}
	resolved, err := contract.ResolveInputs(map[string]any{"name": "fixture"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "fixture", "enabled": false, "limit": float64(3)}, resolved)

	_, err = contract.ResolveInputs(map[string]any{"name": "fixture", "unknown": true})
	require.EqualError(t, err, `unknown input "unknown"`)
	_, err = contract.ResolveInputs(map[string]any{})
	require.EqualError(t, err, `missing required input "name"`)
	_, err = contract.ResolveInputs(map[string]any{"name": "fixture", "limit": 1.5})
	require.EqualError(t, err, `input "limit" must have type integer`)
}
