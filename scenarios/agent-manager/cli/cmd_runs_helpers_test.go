package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestRunCreateModelOverrideIsOptional(t *testing.T) {
	for _, tc := range []struct {
		name, model string
		wantSet     bool
	}{
		{"present", "chosen-model", true},
		{"absent", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var received apipb.CreateRunRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(readAll(t, r), &received); err != nil {
					t.Error(err)
				}
				body, err := protojson.Marshal(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}})
				if err != nil {
					t.Error(err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(body)
			}))
			defer server.Close()
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
			args := []string{"create", "--task-id=task-1", "--profile-id=profile-1", "--json"}
			if tc.model != "" {
				args = append(args, "--model="+tc.model)
			}
			if err := (&App{services: NewServices(api)}).cmdRun(args); err != nil {
				t.Fatal(err)
			}
			if tc.wantSet {
				if received.InlineConfig == nil || received.InlineConfig.Model == nil || received.InlineConfig.GetModel() != tc.model {
					t.Fatalf("model override = %+v", received.InlineConfig)
				}
			} else if received.InlineConfig != nil && received.InlineConfig.Model != nil {
				t.Fatalf("unexpected model override = %q", received.InlineConfig.GetModel())
			}
		})
	}
}

func TestRunAttachAndDetachUseTypedIdentityEndpoints(t *testing.T) {
	var attachRequest apipb.AttachRunRequest
	var detachRequest apipb.DetachRunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/runs/attach":
			if r.Method != http.MethodPost {
				t.Errorf("attach method = %s", r.Method)
			}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(readAll(t, r), &attachRequest); err != nil {
				t.Error(err)
			}
			body, err := protojson.Marshal(&apipb.AttachRunResponse{
				Run:           &domainpb.Run{Id: "attached-run"},
				IdentityToken: "token-once",
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = w.Write(body)
		case "/api/v1/runs/attached-run/detach":
			if r.Method != http.MethodPost {
				t.Errorf("detach method = %s", r.Method)
			}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(readAll(t, r), &detachRequest); err != nil {
				t.Error(err)
			}
			body, err := protojson.Marshal(&apipb.DetachRunResponse{Run: &domainpb.Run{
				Id:     "attached-run",
				Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
			}})
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	app := &App{services: NewServices(api)}

	if err := app.cmdRun([]string{"attach", "--harness-kind=codex", "--harness-session-id=session-1", "--process-id=42", "--json"}); err != nil {
		t.Fatal(err)
	}
	if attachRequest.GetHarnessKind() != "codex" || attachRequest.GetHarnessSessionId() != "session-1" || attachRequest.GetProcessId() != 42 {
		t.Fatalf("attach request = %+v", attachRequest)
	}

	if err := app.cmdRun([]string{"detach", "attached-run", "--reason", "finished", "--json"}); err != nil {
		t.Fatal(err)
	}
	if detachRequest.GetRunId() != "attached-run" || detachRequest.GetReason() != "finished" {
		t.Fatalf("detach request = %+v", detachRequest)
	}
}

func TestRejectRunIdentityLifecycleCommand(t *testing.T) {
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "run-token")

	for _, subcommand := range []string{
		"apply-investigation", "approve", "continue", "create", "delete",
		"investigate", "quiesce", "recover", "reject", "sandbox-sync",
		"stop", "stop-all", "stop-by-tag", "wake",
	} {
		if err := rejectRunIdentityLifecycleCommand(subcommand); err == nil {
			t.Errorf("%s was not rejected for a run identity", subcommand)
		}
	}

	for _, subcommand := range []string{"get", "report", "stats", "events", "diff", "park"} {
		if err := rejectRunIdentityLifecycleCommand(subcommand); err != nil {
			t.Errorf("%s was unexpectedly rejected: %v", subcommand, err)
		}
	}

	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "")
	if err := rejectRunIdentityLifecycleCommand("create"); err != nil {
		t.Fatalf("operator create was unexpectedly rejected: %v", err)
	}
}

func TestRegisteredRunCommandsApplyRunIdentityPreflight(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "run-token")

	for _, command := range app.runCommands() {
		if command.Name != "create" {
			continue
		}
		if err := command.Run(nil); err == nil {
			t.Fatal("registered create command bypassed the run-identity preflight")
		}
		return
	}
	t.Fatal("registered create command not found")
}

func readAll(t *testing.T, r *http.Request) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r.Body); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

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
