package main

import (
	"encoding/json"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"reflect"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestBacklogWritesPreserveReviewedExecutionContract(t *testing.T) {
	for _, command := range []string{"create", "update"} {
		t.Run(command, func(t *testing.T) {
			payload := map[string]any{
				"execution_mode":   "goal",
				"execution_limits": map[string]any{"max_slices": 128, "max_tokens": "12000000", "max_wall_seconds": "604800", "max_turns": 2000, "max_charge_micro_usd": "1500000000", "max_children": 1024, "max_node_attempts": 2048, "max_retries": 128},
				"plan_ref":         map[string]any{"provider": "plan-manager", "plan_id": "plan-1", "slug": "approved-campaign", "role": "execution_spec"},
			}
			if command == "create" {
				payload["kind"], payload["name"], payload["title"] = "execute", "approved-campaign", "Approved campaign"
			}
			input, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			var received []byte
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if command == "update" {
					received = readUpdateRPCJSON(t, r)
				} else {
					received, err = io.ReadAll(r.Body)
				}
				if err != nil {
					t.Fatal(err)
				}
				if command == "update" {
					writeUpdateRPCResponse(t, w, `{"item":{"name":"approved-campaign","kind":"execute"}}`)
				} else {
					_, _ = w.Write([]byte(`{"item":{"name":"approved-campaign","kind":"execute"}}`))
				}
			}))
			app := newAppT(t)
			clitest.CaptureStdout(t, func() error {
				if command == "create" {
					return app.cmdBacklogCreate([]string{"--data", string(input), "--json"})
				}
				return app.cmdBacklogUpdate([]string{"--kind", "execute", "--name", "approved-campaign", "--data", string(input), "--json"})
			})
			var expected, actual any
			if err := json.Unmarshal(input, &expected); err != nil {
				t.Fatal(err)
			}
			if command == "update" {
				var request apipb.UpdateItemRequest
				if err := protojson.Unmarshal(received, &request); err != nil {
					t.Fatal(err)
				}
				if request.Kind != "execute" || request.Name != "approved-campaign" || len(request.Fields) != 3 {
					t.Fatalf("mutation boundary changed: %s", received)
				}
				received, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(request.Patch)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := json.Unmarshal(received, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("reviewed contract changed in CLI transport: got %s want %s", received, input)
			}
		})
	}
}

func TestBacklogUpdatePreservesExplicitLimitsClear(t *testing.T) {
	var received string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = string(readUpdateRPCJSON(t, r))
		writeUpdateRPCResponse(t, w, `{"item":{"name":"campaign","kind":"execute"}}`)
	}))
	app := newAppT(t)
	clitest.CaptureStdout(t, func() error {
		return app.cmdBacklogUpdate([]string{"--kind", "execute", "--name", "campaign", "--data", `{"execution_limits":null}`, "--json"})
	})
	var request apipb.UpdateItemRequest
	if err := protojson.Unmarshal([]byte(received), &request); err != nil {
		t.Fatal(err)
	}
	if request.Kind != "execute" || request.Name != "campaign" || !reflect.DeepEqual(request.Fields, []string{"execution_limits"}) || request.Patch.GetExecutionLimits() != nil {
		t.Fatalf("explicit limits clear was lost: %q", received)
	}
}

// These fixtures exercise the actual Connect protobuf transport and sparse
// field-presence contract, including explicit clears omitted from proto bytes.
func readUpdateRPCJSON(t *testing.T, r *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var req apipb.UpdateItemRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	encoded, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
func writeUpdateRPCResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	var response apipb.BacklogItemResponse
	if err := protojson.Unmarshal([]byte(body), &response); err != nil {
		t.Fatal(err)
	}
	encoded, err := proto.Marshal(&response)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/proto")
	_, _ = w.Write(encoded)
}
