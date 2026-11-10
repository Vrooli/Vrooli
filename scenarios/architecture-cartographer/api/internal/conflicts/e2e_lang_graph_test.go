//go:build e2e_lang_graph

// Package conflicts_test contains the live end-to-end check that
// runs Cartographer through the shared Code Facts graph substrate. Tagged
// `e2e_lang_graph` because it requires the managed dependency scenarios.
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

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/codefacts"
)

type staticURLResolver struct {
	baseURL string
}

func (r staticURLResolver) ResolveScenarioURLDefault(_ context.Context, _ string) (string, error) {
	if r.baseURL == "" {
		return "", fmt.Errorf("empty code-facts base URL")
	}
	return r.baseURL, nil
}

// TestE2E_LiveCodeFactsGraph exercises the production substrate boundary.
func TestE2E_LiveCodeFactsGraph(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := codefacts.New(staticURLResolver{baseURL: liveCodeFactsURL(t, ctx)}, root, graph.LanguageGo, nil)
	raw, err := client.Extract(ctx, "architecture-cartographer")
	if err != nil {
		t.Fatalf("extract through shared Code Facts: %v", err)
	}
	var api, cli bool
	for _, file := range raw.Files {
		api = api || file.Path == "api/main.go"
		cli = cli || file.Path == "cli/main.go"
	}
	if !api || !cli {
		t.Fatalf("all scenario parse units must be discovered: api=%v cli=%v", api, cli)
	}
	if raw.SourceFingerprint == "" || len(raw.ExtractionProfiles) == 0 || len(raw.OmittedInformation) == 0 {
		t.Fatalf("shared source/extraction evidence missing: %+v", raw)
	}
}

func liveCodeFactsURL(t *testing.T, ctx context.Context) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "code-facts", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve Code Facts port; start it with `make start` in scenarios/code-facts or `vrooli scenario start code-facts`: %v", err)
	}
	var payload struct {
		Ports []struct {
			Key            string `json:"key"`
			Port           int    `json:"port"`
			ListenerStatus string `json:"listener_status"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode code-facts port JSON: %v\n%s", err, string(out))
	}
	for _, p := range payload.Ports {
		if p.Key == "API_PORT" && p.Port > 0 && p.ListenerStatus == "listening" {
			return fmt.Sprintf("http://127.0.0.1:%d", p.Port)
		}
	}
	t.Fatalf("code-facts API_PORT is not listening: %s", string(out))
	return ""
}
