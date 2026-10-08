package bindings

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

// The projection verbs had no tests at all, which is why `validate` returned an
// empty result with a SUCCEEDED status for days while every gate stayed green.
// These tests assert the two properties no existing gate could express:
//
//  1. an absent optional field never becomes an argument (the "<nil>" class), and
//  2. a documented default is actually reachable.

func projectionRequestFrom(fields map[string]any) projectionRequest {
	request := projectionRequest{Extra: fields}
	request.SessionID = request.first("session_id")
	request.ProgramID = request.first("program_id")
	request.Provenance = request.first("provenance")
	return request
}

// TestNoVerbEverEmitsARenderedNil is the class-level guard. It walks every
// declared verb with only its required field present and asserts that no
// argument value is the string rendering of a missing key. The original defect
// was exactly this: fmt.Sprint on an absent map key produced "<nil>", which is
// non-empty, so it defeated every `!= ""` check and was sent to the owning
// scenario as a real filter.
func TestNoVerbEverEmitsARenderedNil(t *testing.T) {
	minimalInputs := map[string]map[string]any{
		"recall":   {"intent": "retention policy"},
		"validate": {"scenario": "program-runtime"},
		"capture":  {"text": "a note"},
		"guide":    {"intent": "author an implementation plan"},
	}
	for verb, spec := range projectionVerbs {
		if spec.build == nil {
			continue // Explicitly unavailable verbs have no builder.
		}
		fields, declared := minimalInputs[verb]
		if !declared {
			t.Fatalf("verb %q has a builder but no minimal input in this test; "+
				"add one so a new verb cannot skip the nil guard", verb)
		}
		t.Run(verb, func(t *testing.T) {
			args, err := spec.build(projectionRequestFrom(fields))
			if err != nil {
				t.Fatalf("minimal input must build: %v", err)
			}
			if len(args) == 0 {
				t.Fatalf("verb %q built no arguments from its required field", verb)
			}
			for name, value := range args {
				rendered := fmt.Sprint(value)
				if strings.Contains(rendered, "<nil>") {
					t.Fatalf("argument %q carries a rendered nil (%q); an absent field became a real argument", name, rendered)
				}
			}
		})
	}
}

// TestValidateOmitsStatusWhenTheCallerDidNot is the specific regression. The
// verb sent status="<nil>", test-genie matched zero runs, and the verb reported
// success with an empty result.
func TestValidateOmitsStatusWhenTheCallerDidNot(t *testing.T) {
	args, err := projectionVerbs["validate"].build(projectionRequestFrom(map[string]any{"scenario": "program-runtime"}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, present := args["status"]; present {
		t.Fatalf("validate invented a status filter the caller never sent: %v", args["status"])
	}
	if args["scenario"] != "program-runtime" {
		t.Fatalf("scenario argument = %v, want program-runtime", args["scenario"])
	}
	if args["limit"] != 5 {
		t.Fatalf("default limit = %v, want 5", args["limit"])
	}
}

func TestValidateHonoursAnExplicitStatus(t *testing.T) {
	args, err := projectionVerbs["validate"].build(projectionRequestFrom(map[string]any{
		"scenario": "program-runtime",
		"status":   "failed",
		"depth":    "deep",
	}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if args["status"] != "failed" {
		t.Fatalf("status = %v, want failed", args["status"])
	}
	if args["limit"] != 20 {
		t.Fatalf("deep limit = %v, want 20", args["limit"])
	}
}

// TestCaptureDefaultKindIsReachable is the second instance of the same class:
// the documented `kind="note"` default could never be applied, so a caller that
// omitted the field was refused by the verb's own validation.
func TestCaptureDefaultKindIsReachable(t *testing.T) {
	args, err := projectionVerbs["capture"].build(projectionRequestFrom(map[string]any{"text": "a note"}))
	if err != nil {
		t.Fatalf("omitting kind must apply the documented default, got error: %v", err)
	}
	if args["kind"] != "note" {
		t.Fatalf("default kind = %v, want note", args["kind"])
	}
	if args["body"] != "a note" {
		t.Fatalf("body = %v, want the supplied text", args["body"])
	}
}

func TestCaptureRejectsAnUnknownKind(t *testing.T) {
	_, err := projectionVerbs["capture"].build(projectionRequestFrom(map[string]any{"text": "a note", "kind": "diary"}))
	if err == nil {
		t.Fatal("an unknown kind must be refused")
	}
	if !strings.Contains(err.Error(), "diary") {
		t.Fatalf("refusal must name the offending value, got %q", err)
	}
}

// TestCaptureOmitsWorkRecordFieldsTheCallerDidNotSend keeps the optional
// learning-loop fields honest: absent means absent, not "<nil>".
func TestCaptureOmitsWorkRecordFieldsTheCallerDidNotSend(t *testing.T) {
	args, err := projectionVerbs["capture"].build(projectionRequestFrom(map[string]any{
		"text":    "a record",
		"kind":    "work-record",
		"trigger": "a real trigger",
	}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if args["trigger"] != "a real trigger" {
		t.Fatalf("trigger = %v, want the supplied value", args["trigger"])
	}
	for _, absent := range []string{"scope", "approach", "evidence", "outcome"} {
		if _, present := args[absent]; present {
			t.Fatalf("field %q was absent from the request but present in the arguments as %v", absent, args[absent])
		}
	}
}

func TestRecallRequiresAnIntent(t *testing.T) {
	if _, err := projectionVerbs["recall"].build(projectionRequestFrom(map[string]any{})); err == nil {
		t.Fatal("recall must refuse an empty intent")
	}
	args, err := projectionVerbs["recall"].build(projectionRequestFrom(map[string]any{"query": "retention"}))
	if err != nil {
		t.Fatalf("recall must accept the `query` alias: %v", err)
	}
	if args["query"] != "retention" {
		t.Fatalf("query = %v, want retention", args["query"])
	}
	if args["limit"] != 10 {
		t.Fatalf("default limit = %v, want 10", args["limit"])
	}
}

func TestGuideMapsIntentToDiscoveryQuery(t *testing.T) {
	if _, err := projectionVerbs["guide"].build(projectionRequestFrom(map[string]any{})); err == nil {
		t.Fatal("guide must refuse an empty intent")
	}
	args, err := projectionVerbs["guide"].build(projectionRequestFrom(map[string]any{
		"task": "author an implementation plan",
	}))
	if err != nil {
		t.Fatalf("guide must accept the task alias: %v", err)
	}
	queries, ok := args["queries"].([]string)
	if !ok || len(queries) != 1 || queries[0] != "author an implementation plan" {
		t.Fatalf("queries = %#v, want one repeated request value containing the supplied task", args["queries"])
	}
	if len(args) != 1 {
		t.Fatalf("guide invented discovery arguments: %v", args)
	}
}

// TestProjectionRequestFirstTreatsAbsenceAsEmpty pins the helper the whole fix
// rests on. Missing keys, JSON nulls, and whitespace are all absent.
func TestProjectionRequestFirstTreatsAbsenceAsEmpty(t *testing.T) {
	request := projectionRequestFrom(map[string]any{
		"present": "value",
		"null":    nil,
		"blank":   "   ",
		"number":  float64(7),
	})
	for _, testCase := range []struct {
		name string
		keys []string
		want string
	}{
		{"missing key", []string{"absent"}, ""},
		{"json null", []string{"null"}, ""},
		{"whitespace only", []string{"blank"}, ""},
		{"present value", []string{"present"}, "value"},
		{"falls through to the first non-empty", []string{"absent", "null", "blank", "present"}, "value"},
		{"non-string values still render", []string{"number"}, "7"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := request.first(testCase.keys...); got != testCase.want {
				t.Fatalf("first(%v) = %q, want %q", testCase.keys, got, testCase.want)
			}
		})
	}
}

// TestDecodeProjectionRequestLeavesControlFieldsEmptyWhenAbsent guards the
// third instance of the class: session_id, program_id, and provenance were
// stringified the same way and written to the durable invocation ledger, which
// is why it holds rows whose provenance is the literal text "<nil>".
func TestDecodeProjectionRequestLeavesControlFieldsEmptyWhenAbsent(t *testing.T) {
	request, err := decodeProjectionRequest(strings.NewReader(`{"scenario":"program-runtime"}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for name, value := range map[string]string{
		"session_id": request.SessionID,
		"program_id": request.ProgramID,
		"provenance": request.Provenance,
	} {
		if value != "" {
			t.Fatalf("absent %s decoded to %q; the invocation ledger would record it verbatim", name, value)
		}
	}
}

func TestDecodeProjectionRequestKeepsControlFields(t *testing.T) {
	request, err := decodeProjectionRequest(strings.NewReader(
		`{"session_id":"sess_1","program_id":"prog_1","provenance":"PROVENANCE_AGENT","scenario":"x"}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if request.SessionID != "sess_1" || request.ProgramID != "prog_1" || request.Provenance != "PROVENANCE_AGENT" {
		t.Fatalf("control fields lost: %+v", request)
	}
	if request.first("scenario") != "x" {
		t.Fatalf("verb fields must survive decoding")
	}
}

// TestEveryDeclaredVerbIsEitherComposedOrExplicitlyUnavailable keeps the verb
// table honest: a verb with no binding must name its owner so the gap is
// visible rather than silently absent.
func TestEveryDeclaredVerbIsEitherComposedOrExplicitlyUnavailable(t *testing.T) {
	names := make([]string, 0, len(projectionVerbs))
	for name := range projectionVerbs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		spec := projectionVerbs[name]
		if spec.owner == "" {
			t.Fatalf("verb %q names no owner; an unavailable verb must say whose gap it is", name)
		}
		if spec.rows == "" {
			t.Fatalf("verb %q declares no default row projection", name)
		}
		if spec.binding == "" && spec.build != nil {
			t.Fatalf("verb %q has a builder but no binding to compose", name)
		}
		if spec.binding != "" && spec.build == nil {
			t.Fatalf("verb %q composes %q but has no argument builder", name, spec.binding)
		}
	}
}

func TestProjectionRowsAreValidatedAgainstTheResponseContract(t *testing.T) {
	binding := &bindingsv1.Binding{RowsField: "runs", RowFieldCandidates: []string{"summaries", "warnings"}}
	for _, field := range []string{"runs", "summaries", "warnings", projectionWholeResponse} {
		if err := validateProjectionRows(binding, field); err != nil {
			t.Fatalf("field %q should be accepted: %v", field, err)
		}
	}
	err := validateProjectionRows(binding, "missing")
	if err == nil {
		t.Fatal("unknown row field must be rejected")
	}
	for _, want := range []string{"missing", "runs", "summaries", "warnings"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
}
