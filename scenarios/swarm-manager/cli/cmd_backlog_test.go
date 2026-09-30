package main

import (
	"encoding/json"
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
				"execution_strategy": "adaptive-improvement",
				"execution_limits":   map[string]any{"max_slices": 128, "max_tokens": "12000000", "max_wall_seconds": "604800", "max_turns": 2000, "max_charge_micro_usd": "1500000000", "max_children": 1024, "max_node_attempts": 2048, "max_retries": 128},
				"plan_ref":           map[string]any{"provider": "plan-manager", "plan_id": "plan-1", "slug": "approved-campaign", "role": "execution_spec"},
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
				received, err = io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = w.Write([]byte(`{"item":{"name":"approved-campaign","kind":"execute"}}`))
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
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		_, _ = w.Write([]byte(`{"item":{"name":"campaign","kind":"execute"}}`))
	}))
	app := newAppT(t)
	clitest.CaptureStdout(t, func() error {
		return app.cmdBacklogUpdate([]string{"--kind", "execute", "--name", "campaign", "--data", `{"execution_limits":null}`, "--json"})
	})
	if received != `{"execution_limits":null}` {
		t.Fatalf("explicit limits clear was lost: %q", received)
	}
}
