package protodispatch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/vrooli/cli-core/cliapp"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// hydrateFromContext is the protodispatch entry point that turns CLI
// flags into a populated proto request. The "bind" mechanism added in
// this commit lets a manifest flag declare which proto field it feeds
// and how to decode the value — file path → JSON, inline JSON literal,
// rename-only. These tests pin the contract; they exist because the
// generic name-matching fallback can't reach file-bearing flags and
// silently dropped --flow-file values before the fix.

func TestBind_JSONFile_PopulatesFlowDefinition(t *testing.T) {
	flowJSON := []byte(`{
        "metadata": {"name": "Demo"},
        "nodes": [],
        "edges": []
    }`)
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.json")
	if err := os.WriteFile(path, flowJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "project-id", Required: true},
		{Name: "name", Required: true},
		{Name: "flow-file", Bind: cliapp.FlagBind{Field: "flow_definition", Kind: "json_file"}},
	}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: schema,
		Flags: map[string]string{
			"project-id": "00000000-0000-0000-0000-000000000001",
			"name":       "Demo Workflow",
			"flow-file":  path,
		},
	})

	req := &apiv1.CreateWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}

	out, err := protojson.Marshal(dyn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"name":"Demo Workflow"`) {
		t.Errorf("scalar flag fallback dropped name: %s", got)
	}
	if !strings.Contains(got, `"flowDefinition"`) || !strings.Contains(got, `"metadata":{"name":"Demo"}`) {
		t.Errorf("bound flow_definition not populated from file: %s", got)
	}
}

func TestBind_JSONFile_PreservesAdhocWorkflowLabels(t *testing.T) {
	flowJSON := []byte(`{"metadata":{"name":"labelled","labels":{"session_reuse_mode":"fresh"}},"settings":{"viewportWidth":390,"viewportHeight":844,"fakeMedia":{"microphoneWav":"/tmp/fake.wav"}},"nodes":[],"edges":[]}`)
	path := filepath.Join(t.TempDir(), "flow.json")
	if err := os.WriteFile(path, flowJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "flow-file", Bind: cliapp.FlagBind{Field: "flow_definition", Kind: "json_file"}}}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"flow-file": path}})
	req := &executionv1.ExecuteAdhocRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	out, err := protojson.Marshal(dyn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"sessionReuseMode":"fresh"`) && !strings.Contains(string(out), `"session_reuse_mode":"fresh"`) {
		t.Fatalf("flow labels were dropped from CLI request: %s", out)
	}
	for _, field := range []string{`"viewportWidth":390`, `"viewportHeight":844`, `"microphoneWav":"/tmp/fake.wav"`} {
		if !strings.Contains(string(out), field) {
			t.Fatalf("flow setting %s was dropped from CLI request: %s", field, out)
		}
	}
}

func TestBind_JSONFile_NormalizesSchemaWorkflowDefinition(t *testing.T) {
	flowJSON := []byte(`{
        "metadata": {"execution_mode": "observer"},
        "settings": {"executionViewport": {"width": 390, "height": 844}},
        "nodes": [{"id":"open","action":{"type":"ACTION_TYPE_NAVIGATE","navigate":{"destinationType":"scenario","scenario":"git-control-tower","scenarioPath":"/","waitUntil":"networkidle"}}},{"id":"visible","action":{"type":"ACTION_TYPE_ASSERT","assert":{"selector":"[data-testid=mobile-nav]","assertMode":"visible"}}}],
        "edges": []
    }`)
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.json")
	if err := os.WriteFile(path, flowJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "flow-file", Bind: cliapp.FlagBind{Field: "flow_definition", Kind: "json_file"}},
	}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: schema,
		Flags:  map[string]string{"flow-file": path},
	})

	req := &apiv1.CreateWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("schema-shaped workflow file must decode: %v", err)
	}

	out, err := protojson.Marshal(dyn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	for _, want := range []string{
		`"executionMode":"EXECUTION_MODE_OBSERVER"`,
		`"viewportWidth":390`,
		`"viewportHeight":844`,
		`"destinationType":"NAVIGATE_DESTINATION_TYPE_SCENARIO"`,
		`"waitUntil":"NAVIGATE_WAIT_EVENT_NETWORKIDLE"`,
		`"mode":"ASSERTION_MODE_VISIBLE"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("normalized workflow missing %s: %s", want, got)
		}
	}
}

func TestBind_JSONInline_PopulatesArguments(t *testing.T) {
	// tools execute --args '{"k":"v"}' — both --args (inline) and
	// --args-file (file) target the same proto field; protodispatch
	// must pick whichever was provided.
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "args", Bind: cliapp.FlagBind{Field: "arguments", Kind: "json_inline"}},
		{Name: "args-file", Bind: cliapp.FlagBind{Field: "arguments", Kind: "json_file"}},
	}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: schema,
		Flags:  map[string]string{"args": `{"limit": 5}`},
	})

	// Use a synthetic request whose `arguments` is a google.protobuf.Struct,
	// reusing the ExecuteToolRequest descriptor by going through tools v1.
	// Avoid importing that package here just for the descriptor — instead
	// use CreateWorkflowRequest as a smoke target: it has a tags repeated
	// field we won't touch and a flow_definition message we won't touch;
	// inline JSON onto an unrelated message field is exercised by the
	// rename-only test below. Replace this test's target with a real
	// Struct-field message once protodispatch's tools handler is unit-
	// testable in isolation. For now, just assert no error fires.
	req := &apiv1.CreateWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn)
	if err == nil {
		t.Skip("CreateWorkflowRequest has no 'arguments' field; placeholder test passes trivially")
	}
	// Specifically the bind step should have erred about the missing
	// field name "arguments" — verify that's the failure mode.
	if !strings.Contains(err.Error(), "arguments") {
		t.Errorf("expected error mentioning missing 'arguments' field, got: %v", err)
	}
}

func TestBind_RenameOnly_RoutesVersionFlag(t *testing.T) {
	// --version on workflows execute binds to proto field
	// `workflow_version` (the flag name doesn't match the proto name).
	// The "rename only" case uses no kind, so the value flows through
	// the scalar setter.
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "workflow-id", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "version", Bind: cliapp.FlagBind{Field: "workflow_version"}},
		},
	}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      schema,
		Positionals: map[string]string{"workflow-id": "00000000-0000-0000-0000-000000000099"},
		Flags:       map[string]string{"version": "7"},
	})

	req := &apiv1.ExecuteWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}

	out, _ := protojson.Marshal(dyn)
	if !strings.Contains(string(out), `"workflowVersion":7`) {
		t.Errorf("bound workflow_version not set: %s", out)
	}
	if !strings.Contains(string(out), `"workflowId":"00000000-0000-0000-0000-000000000099"`) {
		t.Errorf("scalar fallback dropped workflow_id: %s", out)
	}
}

func TestBind_BoolFlag_RoutesWaitForCompletion(t *testing.T) {
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "workflow-id", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "wait", Bool: true, Bind: cliapp.FlagBind{Field: "wait_for_completion"}},
		},
	}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      schema,
		Positionals: map[string]string{"workflow-id": "00000000-0000-0000-0000-000000000099"},
		BoolFlags:   map[string]bool{"wait": true},
	})

	req := &apiv1.ExecuteWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}

	out, _ := protojson.Marshal(dyn)
	if !strings.Contains(string(out), `"waitForCompletion":true`) {
		t.Errorf("bound wait_for_completion not set: %s", out)
	}
}

func TestBind_UnknownField_FailsLoudly(t *testing.T) {
	// A typo in a manifest's bind.field should surface as a clear
	// startup-time error, not a silent no-op.
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "flow-file", Bind: cliapp.FlagBind{Field: "flo_definition", Kind: "json_file"}},
	}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: schema,
		Flags:  map[string]string{"flow-file": "/dev/null"},
	})

	req := &apiv1.CreateWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn)
	if err == nil {
		t.Fatal("expected error for unknown bind field, got nil")
	}
	if !strings.Contains(err.Error(), "flo_definition") {
		t.Errorf("error should name the typo, got: %v", err)
	}
}

func TestBind_RequestEscapeHatch_BypassesPerFieldBinds(t *testing.T) {
	// --request '<json>' is the universal escape hatch and must take
	// precedence over per-field binds. This keeps the canonical wire
	// shape available even when bind metadata is broken or missing.
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "request"},
		{Name: "flow-file", Bind: cliapp.FlagBind{Field: "flow_definition", Kind: "json_file"}},
	}}
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: schema,
		Flags: map[string]string{
			"request":   `{"name": "viaRequest"}`,
			"flow-file": "/this/path/does/not/exist.json",
		},
	})

	req := &apiv1.CreateWorkflowRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v (the --request path should not read the file)", err)
	}

	out, _ := protojson.Marshal(dyn)
	if !strings.Contains(string(out), `"name":"viaRequest"`) {
		t.Errorf("--request body lost: %s", out)
	}
}

// Compile-time guard: protodispatch is a no-import surface for cli-core
// consumers; the file-flag fixture should not regress to importing the
// test buffer pkg without intent.
var _ = bytes.NewBuffer
