package main

import (
	"encoding/json"
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestBacklogQueueForwardsAdaptiveSelectionAndOptionalSliceLimit(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		t.Run(map[bool]string{true: "explicit", false: "inherit-reviewed-item"}[explicit], func(t *testing.T) {
			var payload map[string]any
			clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				_, _ = w.Write([]byte(`{"dry_run":true,"queued":false}`))
			}))
			args := []string{"--kind", "execute", "--name", "fixture", "--json"}
			if explicit {
				args = append(args, "--strategy", "adaptive-improvement", "--max-slices", "2")
			}
			app := newAppT(t)
			clitest.CaptureStdout(t, func() error { return app.cmdBacklogQueue(args) })
			if explicit {
				if payload["strategy"] != "adaptive-improvement" || payload["max_slices"] != float64(2) {
					t.Fatalf("owner did not receive selected limits: %+v", payload)
				}
			} else {
				if _, ok := payload["strategy"]; ok {
					t.Fatal("CLI replaced the reviewed item strategy")
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
