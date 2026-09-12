//go:build e2e_lang_graph

// Package conflicts_test contains the live end-to-end check that
// runs cartographer against real `go-code-graph` / `typescript-code-graph`
// instances. Tagged `e2e_lang_graph` so it stays opt-in until those
// dependency scenarios ship (tracked in docs/internal/PROBLEMS.md).
//
// Run via: go test -tags e2e_lang_graph ./internal/conflicts/...
package conflicts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"architecture-cartographer/internal/graph/gocodegraph"
)

type staticURLResolver struct {
	baseURL string
}

func (r staticURLResolver) ResolveScenarioURLDefault(_ context.Context, _ string) (string, error) {
	if r.baseURL == "" {
		return "", fmt.Errorf("empty go-code-graph base URL")
	}
	return r.baseURL, nil
}

// TestE2E_LiveGoCodeGraph drives cartographer's production Go adapter through
// a live go-code-graph producer and verifies test-only import metadata survives
// the cross-scenario Connect-RPC boundary.
func TestE2E_LiveGoCodeGraph(t *testing.T) {
	fixture, err := filepath.Abs("../../../../go-code-graph/bas/fixtures/go-tests")
	if err != nil {
		t.Fatalf("resolve go-tests fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := gocodegraph.New(gocodegraph.Config{
		URLResolver: staticURLResolver{baseURL: liveGoCodeGraphURL(t, ctx)},
		ProjectPath: func(_ string) (string, bool, error) {
			return fixture, true, nil
		},
	})

	raw, err := client.Extract(ctx, "go-tests")
	if err != nil {
		t.Fatalf("extract through live go-code-graph: %v", err)
	}

	var sawLiveImport bool
	var sawInternalTestImport bool
	var sawExternalTestImport bool
	for _, edge := range raw.Imports {
		switch {
		case edge.From == "package:github.com/vrooli/fixtures/go-tests/lib" &&
			edge.ToPackageID == "package:fmt" && !edge.TestOnly:
			sawLiveImport = true
		case edge.From == "package:github.com/vrooli/fixtures/go-tests/lib" &&
			edge.ToPackageID == "package:github.com/vrooli/fixtures/go-tests/helper" && edge.TestOnly:
			sawInternalTestImport = true
		case edge.From == "package:github.com/vrooli/fixtures/go-tests/lib_test" &&
			edge.ToPackageID == "package:github.com/vrooli/fixtures/go-tests/lib" && edge.TestOnly:
			sawExternalTestImport = true
		}
	}
	if !sawLiveImport || !sawInternalTestImport || !sawExternalTestImport {
		t.Fatalf("missing expected live/test import classifications: live=%v internal_test=%v external_test=%v imports=%+v",
			sawLiveImport, sawInternalTestImport, sawExternalTestImport, raw.Imports)
	}
}

func liveGoCodeGraphURL(t *testing.T, ctx context.Context) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "go-code-graph", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve go-code-graph port; start it with `make start` in scenarios/go-code-graph or `vrooli scenario start go-code-graph`: %v", err)
	}
	var payload struct {
		Ports []struct {
			Key            string `json:"key"`
			Port           int    `json:"port"`
			ListenerStatus string `json:"listener_status"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode go-code-graph port JSON: %v\n%s", err, string(out))
	}
	for _, p := range payload.Ports {
		if p.Key == "API_PORT" && p.Port > 0 && p.ListenerStatus == "listening" {
			return fmt.Sprintf("http://127.0.0.1:%d", p.Port)
		}
	}
	t.Fatalf("go-code-graph API_PORT is not listening: %s", string(out))
	return ""
}
