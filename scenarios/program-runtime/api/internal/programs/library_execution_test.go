package programs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"program-runtime/internal/contracts"
)

func declaredLibrary(name, source, inputs string) LibrarySpec {
	return LibrarySpec{
		Name: name, Scenario: "fixture", Contract: true, Current: true, Version: 1,
		Source: source, Digest: strings.Repeat("a", 64),
		Declaration: json.RawMessage(`{"name":"fixture.` + name + `","version":"2","inputs":` + inputs + `}`),
	}
}

func libraryTestRunner(t *testing.T, specs []LibrarySpec, bindings []BindingSpec, bridge string) *SubprocessRunner {
	t.Helper()
	path, err := filepath.Abs("../../../kernel/host/engine.py")
	if err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunnerWithBindings(path, bindings, bridge, bridge)
	runner.SetLibraryProvider(func() []LibrarySpec { return specs })
	t.Cleanup(func() { _ = runner.Close() })
	return runner
}

func runLibraryTest(t *testing.T, runner *SubprocessRunner, source string) (Result, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runner.ExecuteWithMetadata(ctx, "session-fixture", "program-fixture", "test", source, false)
}

// Exercise the production Python host through the same subprocess protocol as
// real submissions. The HTTP fixture stands in for governed owner operations;
// no model, agent, or live scenario is invoked.
func TestNestedLibrariesUseThePublicGovernedSurface(t *testing.T) { // [REQ:PRT-P0-001]
	var mu sync.Mutex
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		request["test_path"] = r.URL.Path
		requests = append(requests, request)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/reachability"):
			fmt.Fprint(w, `{"vrooli":{"reachable":true}}`)
		case strings.Contains(r.URL.Path, "/projection/"):
			fmt.Fprint(w, `{"result":{"rows":[{"value":"ok"}]},"rows_field":"rows"}`)
		case strings.HasSuffix(r.URL.Path, "/start"):
			fmt.Fprint(w, `{"execution_id":"execution-fixture"}`)
		default:
			fmt.Fprint(w, `{"valueJson":"\"ok\"","validated":true}`)
		}
	}))
	defer server.Close()
	for _, tc := range []struct{ name, expression string }{
		{"inference", `ai.classify("fixture", {"type":"string"})`},
		{"delegation", `agent.start(owner="fixture", workflow_key="fixture")`},
		{"recall", `recall("fixture")`},
		{"capture", `capture("fixture")`},
		{"guide", `guide("fixture")`},
		{"validate", `validate("fixture")`},
		{"project", `vrooli.scenario.status()`},
		{"parallel", `gather(lambda: ai.classify("fixture", {"type":"string"}))[0]`},
		{"parallel_projection", `gather(lambda: recall("fixture"))[0]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `print({"status":"ok", "count":` + tc.expression + `.count()})`
			runner := libraryTestRunner(t, []LibrarySpec{declaredLibrary("child", source, `{}`)},
				[]BindingSpec{{ID: "vrooli/scenario/status", Scenario: "vrooli", Group: "scenario", Command: "status", Effect: "read", Reachable: true}}, server.URL+"/execute")
			for _, program := range []string{source, `print(lib.fixture.child().head(1)[0])`} {
				result, err := runLibraryTest(t, runner, program)
				if err != nil || !strings.Contains(result.Stdout, "'count': 1") {
					t.Fatalf("source=%s result=%+v error=%v", program, result, err)
				}
			}
		})
	}
	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 20 {
		t.Fatalf("requests=%d, want one owner call per execution plus two project reachability checks", len(requests))
	}
	for _, request := range requests {
		if request["session_id"] != "session-fixture" {
			t.Fatalf("lost session identity: %v", request)
		}
		_, binding := request["binding_id"]
		projection := strings.Contains(request["test_path"].(string), "/projection/")
		if (binding || projection) && (request["program_id"] != "program-fixture" || request["provenance"] != "test") {
			t.Fatalf("lost program attribution, including parallel calls: %v", request)
		}
	}
}

func TestNestedLibraryInputAdmissionAndIsolation(t *testing.T) { // [REQ:PRT-P0-002]
	inputs := `{"count":{"type":"integer","required":true},"mode":{"type":"string","enum":["read"],"default":"read"},"items":{"type":"array","default":[]}}`
	source := "inputs['items'].append('child')\nprint({'status':'ok', 'inputs':inputs})"
	runner := libraryTestRunner(t, []LibrarySpec{declaredLibrary("child", source, inputs)}, nil, "")
	for _, tc := range []struct{ args, message string }{
		{"", "missing required input 'count'"},
		{"count=True", "must have type integer"},
		{"count=1.5", "must have type integer"},
		{"count=1, mode='write'", "outside its declared enum"},
		{"count=1, other=True", "unexpected keyword"},
		{"count=1, items=[float('nan')]", "finite JSON value"},
		{"count=1, items=[{1:'numeric', '1':'text'}]", "finite JSON value"},
		{"count=1, items=[(1, 2)]", "finite JSON value"},
	} {
		_, err := runLibraryTest(t, runner, "lib.fixture.child("+tc.args+")")
		if err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Fatalf("args=%s error=%v, want %s", tc.args, err, tc.message)
		}
	}
	program := `items = []
first = lib.fixture.child(count=1, items=items)
second = lib.fixture.child(count=2)
third = lib.fixture.child(count=3)
assert items == []
assert first.head(1)[0]['inputs'] == {'count':1, 'mode':'read', 'items':['child']}
assert second.head(1)[0]['inputs']['items'] == ['child']
assert third.head(1)[0]['inputs']['items'] == ['child']
assert first.meta()['version'] == '2'
assert len(first.meta()['digest']) == 64
print('isolated')`
	result, err := runLibraryTest(t, runner, program)
	if err != nil || strings.TrimSpace(result.Stdout) != "isolated" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestNestedLibrariesEnforceOutputAndRecoverAfterFailure(t *testing.T) { // [REQ:PRT-P0-003]
	for _, tc := range []struct{ name, source, message string }{
		{"missing", "result = Handle([])", "exactly one envelope"},
		{"multiple", "print({})\nprint({})", "exactly one envelope"},
		{"oversized", "print({'value':'x'*4097})", "exceeds 4096 bytes"},
		{"nonfinite", "print({'value':float('nan')})", "Out of range float"},
		{"protected", "ai = None\nprint({})", "protected runtime name"},
		{"cycle", "lib.fixture.child()", "recursive library call"},
		{"parallel_cycle", "gather(lambda: lib.fixture.child())", "recursive library call"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := libraryTestRunner(t, []LibrarySpec{declaredLibrary("child", tc.source, `{}`), declaredLibrary("healthy", "print({'status':'ok'})", `{}`)}, nil, "")
			_, err := runLibraryTest(t, runner, "lib.fixture.child()")
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error=%v, want %s", err, tc.message)
			}
			result, err := runLibraryTest(t, runner, "print(lib.fixture.healthy().head(1)[0]['status'])")
			if err != nil || strings.TrimSpace(result.Stdout) != "ok" {
				t.Fatalf("failure poisoned the session: result=%+v error=%v", result, err)
			}
		})
	}
}

func TestNestedLibrariesPreserveRefusals(t *testing.T) { // [REQ:PRT-P0-002]
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"inference requires an explicit grant"}`)
	}))
	defer server.Close()
	runner := libraryTestRunner(t, []LibrarySpec{declaredLibrary("child", `print({"count":ai.classify("fixture", {"type":"string"}).count()})`, `{}`)}, nil, server.URL+"/execute")
	result, err := runLibraryTest(t, runner, "lib.fixture.child()")
	if err == nil || !strings.Contains(err.Error(), "requires an explicit grant") || result.Stdout != "" {
		t.Fatalf("refusal must remain a failure: result=%+v error=%v", result, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("refused call was retried: calls=%d", calls.Load())
	}
}

func TestNestedLibrariesBoundDepthAndSnapshotOutput(t *testing.T) { // [REQ:PRT-P0-003]
	specs := []LibrarySpec{declaredLibrary("leaf", "envelope = {'status':'failed'}\nprint(envelope)\nenvelope['status'] = 'ok'", `{}`)}
	for depth := 1; depth <= 32; depth++ {
		child := "leaf"
		if depth > 1 {
			child = fmt.Sprintf("level_%d", depth-1)
		}
		specs = append(specs, declaredLibrary(fmt.Sprintf("level_%d", depth), "print(lib.fixture."+child+"().head(1)[0])", `{}`))
	}
	runner := libraryTestRunner(t, specs, nil, "")
	result, err := runLibraryTest(t, runner, "print(lib.fixture.level_31().head(1)[0]['status'])")
	if err != nil || strings.TrimSpace(result.Stdout) != "failed" {
		t.Fatalf("32 calls must preserve the emitted child failure: result=%+v error=%v", result, err)
	}
	_, err = runLibraryTest(t, runner, "lib.fixture.level_32()")
	if err == nil || !strings.Contains(err.Error(), "depth exceeds 32") {
		t.Fatalf("33 calls must fail explicitly: error=%v", err)
	}
}

func TestNestedLibrariesHonorParentCancellation(t *testing.T) { // [REQ:PRT-P0-003]
	runner := libraryTestRunner(t, []LibrarySpec{declaredLibrary("child", "while True:\n    pass", `{}`)}, nil, "")
	// Start the host before measuring cancellation, so startup load does not
	// supply a false pass without reaching the nested execution.
	if _, err := runLibraryTest(t, runner, "print('ready')"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := runner.ExecuteWithMetadata(ctx, "session-fixture", "program-fixture", "test", "lib.fixture.child()", false)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("nested work must end with its parent deadline: %v", err)
	}
	runner.mu.Lock()
	_, alive := runner.processes["session-fixture"]
	runner.mu.Unlock()
	if alive {
		t.Fatal("cancelled nested program retained its kernel")
	}
}

func TestPromotedLibraryRetainsItsResultContract(t *testing.T) { // [REQ:PRT-P0-001]
	runner := libraryTestRunner(t, []LibrarySpec{{
		Name: "promoted", Current: true, Version: 2,
		Source: "result = Handle([{'intent':intent, 'text':text, 'inputs':inputs}])",
	}}, nil, "")
	result, err := runLibraryTest(t, runner, "print(lib.promoted().head(1)[0])")
	if err != nil || strings.TrimSpace(result.Stdout) != "{'intent': '', 'text': '', 'inputs': {}}" {
		t.Fatalf("promoted source contract changed: result=%+v error=%v", result, err)
	}
}

// Both admission paths consume the same declaration. Compare observable values
// and refusal decisions so future changes to either language expose drift.
func TestNestedInputAdmissionMatchesDeclaredRunner(t *testing.T) { // [REQ:PRT-P0-002]
	spec := declaredLibrary("child", "print(inputs)", `{
        "count":{"type":"integer","required":true},
        "mode":{"type":"string","default":"read","enum":["read"]},
        "payload":{"type":"object","default":{"enabled":true},"enum":[{"enabled":true}]},
        "enabled":{"type":"boolean","default":true},
        "samples":{"type":"array","default":[]},
        "rate":{"type":"number","default":0.5}
    }`)
	var declaration struct {
		Inputs map[string]contracts.InputSpec `json:"inputs"`
	}
	if err := json.Unmarshal(spec.Declaration, &declaration); err != nil {
		t.Fatal(err)
	}
	contract := contracts.Contract{Inputs: declaration.Inputs}
	runner := libraryTestRunner(t, []LibrarySpec{spec}, nil, "")
	for _, tc := range []struct {
		name, provided string
		accepted       bool
	}{
		{"defaults", `{"count":1}`, true},
		{"explicit", `{"count":2.0,"enabled":false,"samples":[null,2,"x"],"rate":-1.25}`, true},
		{"missing", `{}`, false},
		{"unknown", `{"count":1,"extra":true}`, false},
		{"boolean_is_not_integer", `{"count":true}`, false},
		{"fraction", `{"count":1.5}`, false},
		{"enum", `{"count":1,"mode":"write"}`, false},
		{"nested_enum_boolean_is_not_number", `{"count":1,"payload":{"enabled":1}}`, false},
		{"boolean_is_not_number", `{"count":1,"rate":true}`, false},
		{"wrong_array", `{"count":1,"samples":"value"}`, false},
		{"wrong_object", `{"count":1,"payload":[]}`, false},
		{"null", `{"count":null}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var provided map[string]any
			if err := json.Unmarshal([]byte(tc.provided), &provided); err != nil {
				t.Fatal(err)
			}
			expected, directErr := contract.ResolveInputs(provided)
			if (directErr == nil) != tc.accepted {
				t.Fatalf("direct admission error=%v, accepted=%v", directErr, tc.accepted)
			}
			result, nestedErr := runLibraryTest(t, runner, fmt.Sprintf("import json\nprint(json.dumps(lib.fixture.child(**json.loads(%q)).head(1)[0]))", tc.provided))
			if (nestedErr == nil) != tc.accepted {
				t.Fatalf("nested admission error=%v, accepted=%v", nestedErr, tc.accepted)
			}
			if !tc.accepted {
				return
			}
			var actual map[string]any
			if err := json.Unmarshal([]byte(result.Stdout), &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("nested inputs=%v, direct inputs=%v", actual, expected)
			}
		})
	}
}

func TestSetpointBoardUsesBoundedFailureEvidence(t *testing.T) {
	base := filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "setpoint-read")
	source, err := os.ReadFile(base + ".py")
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	spec := LibrarySpec{Name: "program-runtime.setpoint-read", Scenario: "program-runtime", Contract: true,
		Current: true, Version: 1, Source: string(source), Declaration: declaration, Digest: strings.Repeat("b", 64)}
	var bindings []BindingSpec
	for _, path := range []string{"programs/governance-share", "bindings/act", "bindings/condition", "sessions/delegations", "library/list", "shapes/list", "programs/mine", "programs/list"} {
		parts := strings.Split(path, "/")
		field := "rows"
		var candidates []string
		if path == "bindings/condition" {
			field = "conditions"
			candidates = []string{"conditions"}
		}
		bindings = append(bindings, BindingSpec{ID: "program-runtime/" + path, Scenario: "program-runtime", Group: parts[0], Command: parts[1], Effect: "read", Reachable: true, RowsField: field, RowFieldCandidates: candidates})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/reachability") {
			fmt.Fprint(w, `{"program-runtime":{"reachable":true}}`)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		rows := []any{}
		switch req["binding_id"] {
		case "program-runtime/programs/list":
			rows = append(rows, map[string]any{"status": "PROGRAM_STATUS_FAILED"})
		case "program-runtime/programs/mine":
			rows = append(rows, map[string]any{"shape": "kernel_runtime", "count": "3", "sampleProgramId": strings.Repeat("x", 10000)})
		case "program-runtime/shapes/list":
			rows = append(rows, map[string]any{"shapeKey": strings.Repeat("unrelated-binding/", 1000), "occurrences": "99"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"rows": rows, "conditions": []any{}})
	}))
	defer server.Close()
	runner := libraryTestRunner(t, []LibrarySpec{spec}, bindings, server.URL+"/execute")
	result, err := runLibraryTest(t, runner, `import json
board = lib.program_runtime.setpoint_read().head(1)[0]
assert board["status"] == "ok", board
assert board["signals"]["failure_shapes"] == [{"shape":"kernel_runtime", "count":3}], board
assert board["signals"]["failure_shapes_window"] == "all-time"
assert len(json.dumps(board).encode()) < 4096
print("bounded failure evidence verified")`)
	if err != nil || !strings.Contains(result.Stdout, "bounded failure evidence verified") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestImprovementEvidenceComposesOwnerPrograms(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	var specs []LibrarySpec
	for _, entry := range []struct{ scenario, name string }{{"program-runtime", "improvement-evidence"}, {"visited-tracker", "attention-select"}, {"agent-manager", "investigation-evidence"}} {
		base := filepath.Join(root, "scenarios", entry.scenario, ".vrooli", "program-runtime", entry.name)
		source, err := os.ReadFile(base + ".py")
		if err != nil {
			t.Fatal(err)
		}
		declaration, err := os.ReadFile(base + ".json")
		if err != nil {
			t.Fatal(err)
		}
		specs = append(specs, LibrarySpec{Name: entry.scenario + "." + entry.name, Scenario: entry.scenario, Contract: true, Current: true, Version: 1, Source: string(source), Declaration: declaration, Digest: strings.Repeat("a", 64)})
	}
	for _, mode := range []string{"success", "evidence-unavailable", "attention-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/reachability") {
					fmt.Fprint(w, `{"visited-tracker":{"reachable":true},"agent-manager":{"reachable":true}}`)
					return
				}
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				id, _ := req["binding_id"].(string)
				if mode == "evidence-unavailable" && strings.HasPrefix(id, "agent-manager/") || mode == "attention-unavailable" && strings.HasPrefix(id, "visited-tracker/") {
					http.Error(w, "connection refused", 503)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch id {
				case "visited-tracker/attention/preview":
					fmt.Fprint(w, `{"candidates":[{"fileId":"file","path":"consumer.py","revision":"abc","score":10}],"observedAt":"2026-09-06T12:00:00Z","campaignRevision":"1"}`)
				case "agent-manager/run/report":
					fmt.Fprint(w, `{"status":"failed","exitCode":1,"eventsAvailability":"complete","receiptsAvailability":"complete"}`)
				case "agent-manager/run/episodes":
					fmt.Fprint(w, `{"episodes":[{"episodeId":"episode","pattern":"binding_failure","ownerConfidence":"manifest-derived"}]}`)
				default:
					http.Error(w, "unexpected binding", 400)
				}
			}))
			defer server.Close()
			runner := libraryTestRunner(t, specs, []BindingSpec{
				{ID: "visited-tracker/attention/preview", Scenario: "visited-tracker", Group: "attention", Command: "preview", Effect: "read", Reachable: true, RowsField: "candidates"},
				{ID: "agent-manager/run/report", Scenario: "agent-manager", Group: "run", Command: "report", Effect: "read", Reachable: true},
				{ID: "agent-manager/run/episodes", Scenario: "agent-manager", Group: "run", Command: "episodes", Effect: "read", Reachable: true, RowsField: "episodes"},
			}, server.URL+"/execute")
			expected := "ok"
			if mode != "success" {
				expected = "partial"
			}
			source := fmt.Sprintf(`result=lib.program_runtime.improvement_evidence(campaign_id="campaign",run_ids=["run"]).head(1)[0]
assert result["status"] == %q, result
assert result["signals"]["reserved"] is False
assert result["signals"]["mutation_authorized"] is False
assert len(result["signals"]["children"]) == 2
assert len(result["evidence"]) == 2
assert all(len(item["artifact"]["digest"]) == 64 for item in result["evidence"])
print("composition verified")`, expected)
			result, err := runLibraryTest(t, runner, source)
			if err != nil || !strings.Contains(result.Stdout, "composition verified") {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestFleetFanoutUsesScopedConditionsAndBoundsSchemaSummaries(t *testing.T) {
	base := filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "fleet-fanout")
	source, err := os.ReadFile(base + ".py")
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	spec := LibrarySpec{Name: "program-runtime.fleet-fanout", Scenario: "program-runtime", Contract: true,
		Current: true, Version: 2, Source: string(source), Declaration: declaration, Digest: strings.Repeat("f", 64)}
	bindings := []BindingSpec{
		{ID: "agent-manager/measures/run-volume", Scenario: "agent-manager", Group: "measures", Command: "run-volume", Effect: "read", Reachable: true, RowsField: "rows"},
		{ID: "ai-gateway/measures/total", Scenario: "ai-gateway", Group: "measures", Command: "total", Effect: "read", Reachable: true, RowsField: "rows"},
		{ID: "program-runtime/bindings/condition", Scenario: "program-runtime", Group: "bindings", Command: "condition", Effect: "read", Reachable: true, RowsField: "conditions", RowFieldCandidates: []string{"conditions", "scenarioConditions"}},
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/reachability") {
			fmt.Fprint(w, `{"program-runtime":{"reachable":true},"agent-manager":{"reachable":true},"ai-gateway":{"reachable":true}}`)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		id, _ := req["binding_id"].(string)
		if id != bindings[0].ID && id != bindings[1].ID && id != bindings[2].ID {
			http.Error(w, "unexpected broad catalog request", 403)
			return
		}
		calls.Add(1)
		row := map[string]any{}
		for i := 0; i < 40; i++ {
			row[fmt.Sprintf("field_%02d_%s", i, strings.Repeat("界", 200))] = "value that must not be printed"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"rows": []any{row}, "conditions": []any{row}})
	}))
	defer server.Close()
	runner := libraryTestRunner(t, []LibrarySpec{spec}, bindings, server.URL+"/execute")
	result, err := runLibraryTest(t, runner, `import json
result = lib.program_runtime.fleet_fanout().head(1)[0]
assert result["status"] == "ok", result
assert result["version"] == "2"
assert "program_runtime_conditions" in result["signals"]["surfaces"]
for surface in result["signals"]["surfaces"].values():
    assert surface["count"] == 1
    assert len(surface["first_row_keys"]) == 10
    assert all(len(key) <= 40 and key.isascii() for key in surface["first_row_keys"])
    assert surface["keys_truncated"] is True
assert len(json.dumps(result).encode()) < 4096
assert "value that must not be printed" not in json.dumps(result)
print("bounded scoped fanout verified")`)
	if err != nil || !strings.Contains(result.Stdout, "bounded scoped fanout verified") || calls.Load() != 3 {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls.Load(), err)
	}
}
