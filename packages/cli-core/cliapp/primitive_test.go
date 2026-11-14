package cliapp

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestOperationContext_HasNoOutputModeSurface is the structural guarantee behind
// verified L4: a primitive operation callback receives an OperationContext, and
// that interface must never expose the output mode or the renderers. If any of
// these methods reappears on OperationContext, an operation could branch on
// --json again and this test fails.
func TestOperationContext_HasNoOutputModeSurface(t *testing.T) {
	typ := reflect.TypeOf((*OperationContext)(nil)).Elem()
	for _, forbidden := range []string{
		"JSON", "RenderList", "RenderMutation", "RenderOperational", "Stdout", "Stderr",
	} {
		if _, ok := typ.MethodByName(forbidden); ok {
			t.Errorf("OperationContext must not expose %q — that is an output-mode leak that lets an operation branch on --json", forbidden)
		}
	}
	// The inputs an operation legitimately needs must remain present.
	for _, required := range []string{"Flag", "BoolFlag", "Positional", "FlagBindings", "Core"} {
		if _, ok := typ.MethodByName(required); !ok {
			t.Errorf("OperationContext lost %q — operation callbacks need it to build requests", required)
		}
	}
	// RunContext must remain a superset (it embeds OperationContext), so the
	// full dispatch surface still resolves the operation methods.
	run := reflect.TypeOf((*RunContext)(nil)).Elem()
	if _, ok := run.MethodByName("JSON"); !ok {
		t.Errorf("RunContext must still expose JSON() for top-level handlers")
	}
	if _, ok := run.MethodByName("Flag"); !ok {
		t.Errorf("RunContext must embed OperationContext (Flag missing)")
	}
}

// runPrimitive drives a primitive-built handler through a RunContext with the
// given json flag, capturing stdout.
func runPrimitive(t *testing.T, handler func(RunContext) error, jsonMode bool) (string, error) {
	t.Helper()
	var buf strings.Builder
	ctx := NewTestRunContext(TestRunContextOptions{
		Schema: ArgSchema{},
		JSON:   jsonMode,
		Stdout: &buf,
	})
	err := handler(ctx)
	return buf.String(), err
}

func TestProtoList_SeparatesRenderingFromOperation(t *testing.T) {
	calls := 0
	call := func(ctx OperationContext) (*wrapperspb.StringValue, error) {
		calls++
		return wrapperspb.String("payload-value"), nil
	}
	report := func(ctx OperationContext, resp *wrapperspb.StringValue) ListReport {
		return ListReport{Summary: []string{"one result"}, Results: []string{resp.GetValue()}}
	}
	ph := ProtoList(call, report)
	if ph.Primitive() != PrimitiveProtoList {
		t.Fatalf("ProtoList evidence = %q, want proto_list", ph.Primitive())
	}
	handler := ph.Run

	human, err := runPrimitive(t, handler, false)
	if err != nil {
		t.Fatalf("human render: %v", err)
	}
	if !strings.Contains(human, "Summary:") || !strings.Contains(human, "payload-value") {
		t.Fatalf("human output missing report shape: %q", human)
	}

	jsonOut, err := runPrimitive(t, handler, true)
	if err != nil {
		t.Fatalf("json render: %v", err)
	}
	// --json yields the proto wire shape, not the human ListReport wrapper.
	if strings.Contains(jsonOut, "Summary:") {
		t.Fatalf("json output leaked human report shape: %q", jsonOut)
	}
	var decoded any
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil {
		t.Fatalf("json output is not valid JSON: %v (%q)", err, jsonOut)
	}

	// The operation ran once per render; the handler never branched on --json.
	if calls != 2 {
		t.Fatalf("expected the call func to run once per invocation, got %d", calls)
	}
}

func TestProtoMutation_RendersMutationHumanAndProtoJSON(t *testing.T) {
	ph := ProtoMutation(
		func(ctx OperationContext) (*wrapperspb.StringValue, error) {
			return wrapperspb.String("created-id"), nil
		},
		func(ctx OperationContext, resp *wrapperspb.StringValue) MutationReport {
			return MutationReport{Result: []string{"created " + resp.GetValue()}}
		},
	)
	if ph.Primitive() != PrimitiveProtoMutation {
		t.Fatalf("ProtoMutation evidence = %q, want proto_mutation", ph.Primitive())
	}
	handler := ph.Run
	human, err := runPrimitive(t, handler, false)
	if err != nil {
		t.Fatalf("human render: %v", err)
	}
	if !strings.Contains(human, "Result:") || !strings.Contains(human, "created-id") {
		t.Fatalf("mutation human output wrong: %q", human)
	}
	jsonOut, err := runPrimitive(t, handler, true)
	if err != nil {
		t.Fatalf("json render: %v", err)
	}
	if !strings.Contains(jsonOut, "created-id") || strings.Contains(jsonOut, "Result:") {
		t.Fatalf("mutation json output wrong: %q", jsonOut)
	}
}

func TestProtoOperational_RendersOperationalHumanAndProtoJSON(t *testing.T) {
	ph := ProtoOperational(
		func(ctx OperationContext) (*wrapperspb.StringValue, error) { return wrapperspb.String("ok"), nil },
		func(ctx OperationContext, resp *wrapperspb.StringValue) OperationalReport {
			return OperationalReport{Status: []string{"health: " + resp.GetValue()}}
		},
	)
	if ph.Primitive() != PrimitiveOperational {
		t.Fatalf("ProtoOperational evidence = %q, want operational", ph.Primitive())
	}
	handler := ph.Run
	human, err := runPrimitive(t, handler, false)
	if err != nil {
		t.Fatalf("human render: %v", err)
	}
	if !strings.Contains(human, "Status:") || !strings.Contains(human, "health: ok") {
		t.Fatalf("operational human output wrong: %q", human)
	}
	jsonOut, err := runPrimitive(t, handler, true)
	if err != nil {
		t.Fatalf("json render: %v", err)
	}
	if strings.Contains(jsonOut, "Status:") {
		t.Fatalf("operational json output leaked human shape: %q", jsonOut)
	}
}

func TestAction_RendersMutationHumanAndJSON(t *testing.T) {
	type actionResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	calls := 0
	ph := Action(
		func(ctx OperationContext) (actionResp, error) {
			calls++
			return actionResp{ID: "req-1", Status: "queued"}, nil
		},
		func(ctx OperationContext, resp actionResp) MutationReport {
			return MutationReport{Result: []string{"queued " + resp.ID}}
		},
	)
	if ph.Primitive() != PrimitiveAction {
		t.Fatalf("Action evidence = %q, want action", ph.Primitive())
	}
	human, err := runPrimitive(t, ph.Run, false)
	if err != nil {
		t.Fatalf("human render: %v", err)
	}
	if !strings.Contains(human, "Result:") || !strings.Contains(human, "queued req-1") {
		t.Fatalf("action human output wrong: %q", human)
	}
	jsonOut, err := runPrimitive(t, ph.Run, true)
	if err != nil {
		t.Fatalf("json render: %v", err)
	}
	if strings.Contains(jsonOut, "Result:") {
		t.Fatalf("action json output leaked human shape: %q", jsonOut)
	}
	var decoded actionResp
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil {
		t.Fatalf("action json output is not valid JSON: %v (%q)", err, jsonOut)
	}
	if decoded.ID != "req-1" || decoded.Status != "queued" {
		t.Fatalf("action json decoded wrong: %+v", decoded)
	}
	if calls != 2 {
		t.Fatalf("expected action call once per invocation, got %d", calls)
	}
}

func TestProtoListOutcome_RendersThenReturnsOutcomeInBothModes(t *testing.T) {
	// The outcome error must fire AFTER rendering and be identical for human and
	// --json — the exit code is a property of the response, not the output format.
	build := func(fail bool) PrimitiveHandler {
		return ProtoListOutcome(
			func(OperationContext) (*wrapperspb.StringValue, error) { return wrapperspb.String("v"), nil },
			func(OperationContext, *wrapperspb.StringValue) ListReport {
				return ListReport{Summary: []string{"one"}, Results: []string{"v"}}
			},
			func(resp *wrapperspb.StringValue) error {
				if fail {
					return errString("assessment failed")
				}
				return nil
			},
		)
	}
	if build(false).Primitive() != PrimitiveProtoList {
		t.Fatalf("ProtoListOutcome evidence must be proto_list")
	}

	// Failing outcome: output still renders, and the error is returned in both modes.
	for _, jsonMode := range []bool{false, true} {
		out, err := runPrimitive(t, build(true).Run, jsonMode)
		if err == nil || !strings.Contains(err.Error(), "assessment failed") {
			t.Fatalf("json=%v: expected outcome error, got %v", jsonMode, err)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatalf("json=%v: output must still render before the outcome error", jsonMode)
		}
	}

	// Passing outcome: no error.
	if _, err := runPrimitive(t, build(false).Run, false); err != nil {
		t.Fatalf("passing outcome should not error, got %v", err)
	}
}

func TestPrimitive_PropagatesCallError(t *testing.T) {
	sentinel := "boom"
	handler := ProtoList(
		func(ctx OperationContext) (*wrapperspb.StringValue, error) { return nil, errString(sentinel) },
		func(ctx OperationContext, resp *wrapperspb.StringValue) ListReport { return ListReport{} },
	).Run
	if _, err := runPrimitive(t, handler, false); err == nil || !strings.Contains(err.Error(), sentinel) {
		t.Fatalf("expected call error to propagate, got %v", err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
