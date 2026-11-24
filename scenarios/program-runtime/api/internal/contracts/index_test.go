package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestRuntimeDeclarationCarriesCompiledOutputSchemaWithoutChangingArtifact(t *testing.T) {
	original := []byte(`{"name":"demo.program","output_schema":"program.output.schema.json"}`)
	schema := []byte(`{"type":"object","required":["status"]}`)
	contract := Contract{Declaration: original, OutputSchemaSource: schema}
	contract.Digest = ContentDigest(original, nil, schema)

	runtimeDeclaration, err := contract.RuntimeDeclaration()
	require.NoError(t, err)
	require.Equal(t, original, contract.Declaration)
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(runtimeDeclaration, &document))
	require.JSONEq(t, string(schema), string(document["_runtime_output_schema"]))
	require.NotContains(t, string(original), "_runtime_output_schema")
	contract.OutputSchemaSource = []byte(`{"type":"array"}`)
	_, err = contract.RuntimeDeclaration()
	require.ErrorContains(t, err, "content digest does not match")
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

func TestConcurrentLoadsCannotPublishOlderSnapshotAfterNewer(t *testing.T) {
	index := NewIndex()
	olderReadStarted := make(chan struct{})
	releaseOlderRead := make(chan struct{})
	var releaseOlderOnce sync.Once
	releaseOlder := func() { releaseOlderOnce.Do(func() { close(releaseOlderRead) }) }
	t.Cleanup(releaseOlder)
	olderDone := make(chan error, 1)
	go func() {
		olderDone <- index.loadSnapshot(func() ([]Contract, map[string]int64, error) {
			close(olderReadStarted)
			<-releaseOlderRead
			return []Contract{{ID: "demo.program", Digest: "older"}}, map[string]int64{"demo": 1}, nil
		})
	}()
	<-olderReadStarted

	newerReadStarted := make(chan struct{})
	newerAttempted := make(chan struct{})
	newerDone := make(chan error, 1)
	go func() {
		close(newerAttempted)
		newerDone <- index.loadSnapshot(func() ([]Contract, map[string]int64, error) {
			close(newerReadStarted)
			return []Contract{{ID: "demo.program", Digest: "newer"}}, map[string]int64{"demo": 2}, nil
		})
	}()
	<-newerAttempted

	// On an unsynchronized index, the newer read completes while the older read
	// is blocked. Releasing the older read then deterministically regresses the
	// digest. A serialized load keeps the newer read queued until publication.
	newerFinishedEarly := false
	select {
	case <-newerReadStarted:
		require.NoError(t, <-newerDone)
		newerFinishedEarly = true
	case <-time.After(100 * time.Millisecond):
	}
	releaseOlder()
	require.NoError(t, <-olderDone)
	if !newerFinishedEarly {
		require.NoError(t, <-newerDone)
	}

	got := index.List()
	require.Len(t, got, 1)
	require.Equal(t, "newer", got[0].Digest)
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

func TestResolveInputsMatchesSharedVectors(t *testing.T) {
	type vector struct {
		Name          string                     `json:"name"`
		Inputs        map[string]json.RawMessage `json:"inputs"`
		Provided      map[string]any             `json:"provided"`
		Expected      map[string]any             `json:"expected"`
		ErrorContains []string                   `json:"error_contains"`
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "input-resolution-vectors.json"))
	require.NoError(t, err)
	var vectors []vector
	require.NoError(t, json.Unmarshal(data, &vectors))
	for _, test := range vectors {
		t.Run(test.Name, func(t *testing.T) {
			inputs := make(map[string]InputSpec, len(test.Inputs))
			for name, raw := range test.Inputs {
				var spec struct {
					Type     string          `json:"type"`
					Required bool            `json:"required"`
					Default  json.RawMessage `json:"default"`
					Enum     []any           `json:"enum"`
				}
				require.NoError(t, json.Unmarshal(raw, &spec))
				inputs[name] = InputSpec{Type: spec.Type, Required: spec.Required, Default: spec.Default, Enum: spec.Enum}
			}
			resolved, err := (Contract{Inputs: inputs}).ResolveInputs(test.Provided)
			if len(test.ErrorContains) > 0 {
				require.Error(t, err)
				matched := false
				for _, want := range test.ErrorContains {
					matched = matched || strings.Contains(err.Error(), want)
				}
				require.True(t, matched, "error %q should contain one of %q", err, test.ErrorContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.Expected, resolved)
		})
	}
}

func TestLearningCapabilityPolicyRequiresExplicitOwnerAndEffects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy map[string]any
		valid  bool
	}{
		{"scoped read", map[string]any{"scenario": "browser-automation-studio", "effects": []string{"read"}}, true},
		{"missing owner", map[string]any{"effects": []string{"read"}}, false},
		{"unknown effect", map[string]any{"scenario": "browser-automation-studio", "effects": []string{"everything"}}, false},
		{"no effects", map[string]any{"scenario": "browser-automation-studio", "effects": []string{}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var document map[string]any
			require.NoError(t, json.Unmarshal(validContractData(t), &document))
			document["learning"] = map[string]any{"capabilities": []any{tc.policy}}
			data, err := json.Marshal(document)
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "scoped.json")
			require.NoError(t, os.WriteFile(path, data, 0o600))
			got := readContract("demo", path, testSchema(t))
			if tc.valid {
				require.Empty(t, got.ValidationError)
				require.Equal(t, []LearningCapability{{Scenario: "browser-automation-studio", Effects: []string{"read"}}}, got.Learning.Capabilities)
			} else {
				require.NotEmpty(t, got.ValidationError)
			}
		})
	}
}

func researchContract(t *testing.T, name string) []byte {
	t.Helper()
	var contract map[string]any
	require.NoError(t, json.Unmarshal(validContractData(t), &contract))
	contract["name"] = name
	delete(contract, "$schema")
	data, err := json.Marshal(contract)
	require.NoError(t, err)
	return data
}

func TestResearchProgramsResolveByIDAndStayOutOfTheCallableSet(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	home := t.TempDir()
	topicPrograms := filepath.Join(home, "topics", "ux", "programs")
	require.NoError(t, os.MkdirAll(topicPrograms, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "programs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(topicPrograms, "review.json"), researchContract(t, "research/ux/review"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(topicPrograms, "review.py"), []byte("print('v1')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(topicPrograms, "copied.json"), researchContract(t, "research/other/copied"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(home, "programs", "sweep.json"), researchContract(t, "research/sweep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(home, "programs", "sweep.py"), []byte("print('sweep')\n"), 0o644))

	index := NewIndex()
	index.SetResearchHome(home)
	require.NoError(t, index.Load(root))

	review, ok := index.Lookup("research/ux/review")
	require.True(t, ok)
	require.Empty(t, review.ValidationError)
	require.Equal(t, "print('v1')\n", review.Source)
	sweep, ok := index.Lookup("research/sweep")
	require.True(t, ok)
	require.Empty(t, sweep.ValidationError)
	copied, ok := index.Lookup("research/ux/copied")
	require.True(t, ok)
	require.Contains(t, copied.ValidationError, `must be "research/ux/copied"`)
	_, ok = index.Lookup("research/ux/absent")
	require.False(t, ok)
	_, ok = index.Lookup("prompt-manager.skill-set-read")
	require.True(t, ok, "scenario names still resolve through Lookup")

	require.Len(t, index.Research(), 3)
	for _, contract := range index.List() {
		require.False(t, contract.Research, "research programs are not lib-callable: %s", contract.ID)
	}

	later := time.Now().Add(2 * time.Second)
	require.NoError(t, os.WriteFile(filepath.Join(topicPrograms, "review.py"), []byte("print('v2')\n"), 0o644))
	require.NoError(t, os.Chtimes(filepath.Join(topicPrograms, "review.py"), later, later))
	changed, err := index.Refresh(root)
	require.NoError(t, err)
	require.True(t, changed, "a research program edit must reload")
	review, _ = index.Lookup("research/ux/review")
	require.Equal(t, "print('v2')\n", review.Source)

	plain := NewIndex()
	require.NoError(t, plain.Load(root))
	require.Empty(t, plain.Research(), "an index without a research home never reads one")
}

func TestResearchProgramsThatLinkOutOfTheHomeAreRefused(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	home, outside := t.TempDir(), t.TempDir()
	programs := filepath.Join(home, "programs")
	require.NoError(t, os.MkdirAll(programs, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(programs, "leak.json"), researchContract(t, "research/leak"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "leak.py"), []byte("print('outside')\n"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(outside, "leak.py"), filepath.Join(programs, "leak.py")))

	index := NewIndex()
	index.SetResearchHome(home)
	require.NoError(t, index.Load(root))
	leak, ok := index.Lookup("research/leak")
	require.True(t, ok)
	require.Contains(t, leak.ValidationError, "outside the research home")
	require.Empty(t, leak.Source, "a source outside the home is never loaded")
}
