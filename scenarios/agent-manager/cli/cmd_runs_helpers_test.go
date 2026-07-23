package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestParseResultSpecJSONInputsAndFailures(t *testing.T) {
	t.Run("no configuration", func(t *testing.T) {
		spec, err := parseResultSpec("", "", "", false)
		if err != nil || spec != nil {
			t.Fatalf("parseResultSpec() = %#v, %v; want nil, nil", spec, err)
		}
	})

	t.Run("inline schema", func(t *testing.T) {
		spec, err := parseResultSpec(`{"type":"object"}`, "", "", false)
		if err != nil {
			t.Fatal(err)
		}
		if spec.Kind != domainpb.ResultSpecKind_RESULT_SPEC_KIND_JSON_SCHEMA || string(spec.Schema) != `{"type":"object"}` || spec.ExtractionRole != "" {
			t.Fatalf("unexpected spec: %+v", spec)
		}
	})

	t.Run("schema file with extraction", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "schema.json")
		if err := os.WriteFile(path, []byte(`{"type":"string"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		spec, err := parseResultSpec("", path, "", true)
		if err != nil {
			t.Fatal(err)
		}
		if spec.ExtractionMode != domainpb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK || spec.ExtractionRole != "extract.structured" {
			t.Fatalf("unexpected extraction config: %+v", spec)
		}
	})

	for _, test := range []struct {
		name, schema, file, classification string
	}{
		{"multiple fields", `{}`, "schema.json", "yes,no"},
		{"empty classification", "", "", " , "},
		{"invalid schema", `not-json`, "", ""},
		{"missing schema file", "", filepath.Join(t.TempDir(), "missing.json"), ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseResultSpec(test.schema, test.file, test.classification, false); err == nil {
				t.Fatal("expected parse failure")
			}
		})
	}
}

func TestRunEventDataStringSupportsEveryDisplayablePayload(t *testing.T) {
	tests := []struct {
		name  string
		event *domainpb.RunEvent
	}{
		{"log", &domainpb.RunEvent{Data: &domainpb.RunEvent_Log{Log: &domainpb.LogEventData{Message: "message"}}}},
		{"message", &domainpb.RunEvent{Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Content: "message"}}}},
		{"message deleted", &domainpb.RunEvent{Data: &domainpb.RunEvent_MessageDeleted{MessageDeleted: &domainpb.MessageDeletedEventData{}}}},
		{"tool call", &domainpb.RunEvent{Data: &domainpb.RunEvent_ToolCall{ToolCall: &domainpb.ToolCallEventData{}}}},
		{"tool result", &domainpb.RunEvent{Data: &domainpb.RunEvent_ToolResult{ToolResult: &domainpb.ToolResultEventData{}}}},
		{"status", &domainpb.RunEvent{Data: &domainpb.RunEvent_Status{Status: &domainpb.StatusEventData{}}}},
		{"metric", &domainpb.RunEvent{Data: &domainpb.RunEvent_Metric{Metric: &domainpb.MetricEventData{}}}},
		{"artifact", &domainpb.RunEvent{Data: &domainpb.RunEvent_Artifact{Artifact: &domainpb.ArtifactEventData{}}}},
		{"error", &domainpb.RunEvent{Data: &domainpb.RunEvent_Error{Error: &domainpb.ErrorEventData{}}}},
		{"progress", &domainpb.RunEvent{Data: &domainpb.RunEvent_Progress{Progress: &domainpb.ProgressEventData{}}}},
		{"cost", &domainpb.RunEvent{Data: &domainpb.RunEvent_Cost{Cost: &domainpb.CostEventData{}}}},
		{"rate limit", &domainpb.RunEvent{Data: &domainpb.RunEvent_RateLimit{RateLimit: &domainpb.RateLimitEventData{}}}},
		{"compaction", &domainpb.RunEvent{Data: &domainpb.RunEvent_Compaction{Compaction: &domainpb.CompactionEventData{}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := runEventDataString(test.event)
			if got == "" {
				t.Fatal("expected JSON payload")
			}
		})
	}
	if got := runEventDataString(nil); got != "" {
		t.Fatalf("nil event = %q", got)
	}
	if got := runEventDataString(&domainpb.RunEvent{}); got != "" {
		t.Fatalf("event without data = %q", got)
	}
}

func TestRunEventDataStringDoesNotMutateEvent(t *testing.T) {
	event := &domainpb.RunEvent{Data: &domainpb.RunEvent_Log{Log: &domainpb.LogEventData{Message: "stable"}}}
	before := event.GetLog()
	_ = runEventDataString(event)
	if !reflect.DeepEqual(before, event.GetLog()) {
		t.Fatal("rendering must not mutate the event payload")
	}
}
