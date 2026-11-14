package main

import (
	"encoding/json"
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestBacklogQueueForwardsReviewedExecutionModeAndOptionalSliceLimit(t *testing.T) {
	for _, mode := range []string{"sliced", "goal", ""} {
		name := mode
		if name == "" {
			name = "inherit-reviewed-item"
		}
		t.Run(name, func(t *testing.T) {
			var payload map[string]any
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				_, _ = w.Write([]byte(`{"dry_run":true,"queued":false}`))
			}))
			args := []string{"--kind", "execute", "--name", "fixture", "--json"}
			if mode != "" {
				args = append(args, "--execution-mode", mode, "--max-slices", "2")
			}
			app := newAppT(t)
			clitest.CaptureStdout(t, func() error { return app.cmdBacklogQueue(args) })
			if mode != "" {
				if payload["execution_mode"] != mode || payload["max_slices"] != float64(2) {
					t.Fatalf("owner did not receive selected limits: %+v", payload)
				}
			} else {
				if _, ok := payload["execution_mode"]; ok {
					t.Fatal("CLI replaced the reviewed item execution mode")
				}
				if _, ok := payload["max_slices"]; ok {
					t.Fatal("CLI replaced the reviewed item slice limit")
				}
			}
			if payload["confirm"] != false {
				t.Fatal("selection flags silently authorized execution")
			}
		})
	}
}
