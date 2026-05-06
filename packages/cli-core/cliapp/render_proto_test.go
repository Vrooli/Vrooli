package cliapp

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestPrintProtoJSON_UsesSnakeCaseFieldNames(t *testing.T) {
	// structpb.Struct exposes fields named with underscores in proto
	// (`fields`); confirm we get protojson's snake_case wire shape, not
	// Go-style CamelCase.
	payload, err := structpb.NewStruct(map[string]any{
		"first_key":  "value-one",
		"second_key": "value-two",
	})
	if err != nil {
		t.Fatalf("build struct: %v", err)
	}

	var out bytes.Buffer
	if err := PrintProtoJSON(&out, payload); err != nil {
		t.Fatalf("PrintProtoJSON: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `"first_key"`) {
		t.Errorf("expected snake_case field name in output, got: %s", body)
	}
	if !strings.HasSuffix(body, "\n") {
		t.Errorf("expected trailing newline, got: %q", body)
	}
}

func TestPrintProtoJSON_RoundTripsThroughProtojson(t *testing.T) {
	original := wrapperspb.String("hello world")

	var out bytes.Buffer
	if err := PrintProtoJSON(&out, original); err != nil {
		t.Fatalf("PrintProtoJSON: %v", err)
	}

	var got wrapperspb.StringValue
	if err := protojson.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("protojson.Unmarshal back: %v (body=%q)", err, out.String())
	}
	if got.Value != "hello world" {
		t.Errorf("round-trip lost value: got %q, want %q", got.Value, "hello world")
	}
}

func TestRenderProtoList_JSONEmitsProtoShapeNotReportShape(t *testing.T) {
	payload, err := structpb.NewStruct(map[string]any{
		"id":    "abc",
		"title": "first",
	})
	if err != nil {
		t.Fatalf("build struct: %v", err)
	}

	var out bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		JSON:   true,
		Stdout: &out,
	})
	human := ListReport{
		Summary:        []string{"Found 1 thing."},
		Results:        []string{"abc — first"},
		RetrievalHints: []string{"`cli get abc`"},
	}
	if err := RenderProtoList(ctx, payload, human); err != nil {
		t.Fatalf("RenderProtoList: %v", err)
	}

	body := out.String()
	if !strings.Contains(body, `"id"`) {
		t.Errorf("expected proto field 'id' in JSON output, got: %s", body)
	}
	if strings.Contains(body, "summary") || strings.Contains(body, "retrieval_hints") {
		t.Errorf("expected NO report wrapper fields in JSON output, got: %s", body)
	}
	if strings.Contains(body, "Found 1 thing.") {
		t.Errorf("expected NO human summary in JSON output, got: %s", body)
	}
}

func TestRenderProtoList_HumanFallsThroughToReport(t *testing.T) {
	payload := wrapperspb.String("ignored in human path")

	var out bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		JSON:   false,
		Stdout: &out,
	})
	human := ListReport{
		Summary: []string{"Found 2 widgets."},
		Results: []string{"widget-a", "widget-b"},
	}
	if err := RenderProtoList(ctx, payload, human); err != nil {
		t.Fatalf("RenderProtoList: %v", err)
	}

	body := out.String()
	if !strings.Contains(body, "Found 2 widgets.") {
		t.Errorf("expected human summary in non-JSON output, got: %s", body)
	}
	if !strings.Contains(body, "widget-a") {
		t.Errorf("expected human result in non-JSON output, got: %s", body)
	}
	if strings.Contains(body, "ignored in human path") {
		t.Errorf("proto payload leaked into human output: %s", body)
	}
}

func TestRenderProtoMutation_JSONEmitsProtoShape(t *testing.T) {
	payload := wrapperspb.String("created-id")

	var out bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		JSON:   true,
		Stdout: &out,
	})
	human := MutationReport{
		Result:      []string{"Created widget."},
		NextCommand: []string{"`cli widgets list`"},
	}
	if err := RenderProtoMutation(ctx, payload, human); err != nil {
		t.Fatalf("RenderProtoMutation: %v", err)
	}

	body := out.String()
	if !strings.Contains(body, "created-id") {
		t.Errorf("expected proto value in JSON output, got: %s", body)
	}
	if strings.Contains(body, "next_command") || strings.Contains(body, "Created widget.") {
		t.Errorf("expected NO report wrapper / human text in JSON output, got: %s", body)
	}
}

func TestRenderProtoMutation_HumanFallsThroughToReport(t *testing.T) {
	payload := wrapperspb.String("ignored in human path")

	var out bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		JSON:   false,
		Stdout: &out,
	})
	human := MutationReport{
		Result:  []string{"Created widget XYZ."},
		Changes: []string{"id=XYZ"},
	}
	if err := RenderProtoMutation(ctx, payload, human); err != nil {
		t.Fatalf("RenderProtoMutation: %v", err)
	}

	body := out.String()
	if !strings.Contains(body, "Created widget XYZ.") {
		t.Errorf("expected human result line in output, got: %s", body)
	}
	if strings.Contains(body, "ignored in human path") {
		t.Errorf("proto payload leaked into human output: %s", body)
	}
}
