package capabilityprobe

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestProbeReportsReadyMissingAndUnknown(t *testing.T) {
	defs := []Definition{
		{Capability: "ai-cli", ID: "ready", Command: "ready"},
		{Capability: "ai-cli", ID: "missing", Command: "missing"},
		{Capability: "ai-cli", ID: "unknown", Command: "unknown"},
	}
	look := func(command string) (string, error) {
		if command == "missing" {
			return "", errors.New("not found")
		}
		return "/bin/" + command, nil
	}
	version := func(_ context.Context, path string, _ []string) (string, error) {
		if path == "/bin/unknown" {
			return "", errors.New("failed")
		}
		return "v1.2.3\n", nil
	}
	when := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	got := ProbeWith(context.Background(), defs, look, version, func() time.Time { return when })
	if len(got) != 3 || got[0].State != Ready || got[1].State != Missing || got[2].State != Unknown {
		t.Fatalf("observations = %+v", got)
	}
	if !got[0].ProbedAt.Equal(when) || got[0].Version != "v1.2.3" {
		t.Fatalf("observation metadata = %+v", got[0])
	}
}

func TestProbeWithHonorsCallerDeadlineForHangingCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	got := ProbeWith(ctx, []Definition{{Capability: "ai-cli", ID: "hung", Command: "hung"}},
		func(string) (string, error) { return "/bin/hung", nil },
		func(ctx context.Context, _ string, _ []string) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		}, time.Now)
	if len(got) != 1 || got[0].State != Unknown {
		t.Fatalf("hanging command observation = %+v", got)
	}
}

// This module stays free of dependencies for the agents that embed it, so it
// cannot read the descriptors through their package. This keeps the table equal
// to them: a new resource's harness block fails here until the table has it.
func TestAIToolsMatchesHarnessDescriptors(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(file), "..", "..", "resources", "*", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]Definition{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Harness *struct {
				ID          string `json:"id"`
				Binary      string `json:"binary"`
				DisplayName string `json:"display_name"`
			} `json:"harness"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if manifest.Harness != nil {
			declared[manifest.Harness.ID] = Definition{Capability: "ai-cli", ID: manifest.Harness.ID, Label: manifest.Harness.DisplayName, Command: manifest.Harness.Binary}
		}
	}
	if len(declared) != len(AITools) {
		t.Fatalf("descriptors=%v table=%v", declared, AITools)
	}
	for _, definition := range AITools {
		want, ok := declared[definition.ID]
		if !ok || want.Label != definition.Label || want.Command != definition.Command || want.Capability != definition.Capability {
			t.Errorf("table entry %+v disagrees with descriptor %+v", definition, want)
		}
	}
}

func TestAIToolsMatchesToolManifests(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "internal", "tools")
	var manifestIDs []string
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), "tool.json"))
		if err != nil {
			continue
		}
		var manifest struct {
			Name       string   `json:"name"`
			Commands   []string `json:"commands"`
			Capability string   `json:"capability"`
		}
		if json.Unmarshal(data, &manifest) != nil || manifest.Capability != "ai-cli" {
			continue
		}
		if len(manifest.Commands) == 0 {
			t.Fatalf("%s has no command", entry.Name())
		}
		manifestIDs = append(manifestIDs, manifest.Name)
	}
	want := make([]string, 0, len(AITools))
	for _, definition := range AITools {
		want = append(want, definition.ID)
	}
	sort.Strings(manifestIDs)
	sort.Strings(want)
	if len(manifestIDs) != len(want) {
		t.Fatalf("manifest ids=%v generated ids=%v", manifestIDs, want)
	}
	for i := range want {
		if manifestIDs[i] != want[i] {
			t.Fatalf("manifest ids=%v generated ids=%v", manifestIDs, want)
		}
	}
}
