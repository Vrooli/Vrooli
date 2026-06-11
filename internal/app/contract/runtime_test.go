package contractapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestRunSchemaValidationPassesForLiveRepo(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	message, ok := RunSchemaValidation(root)
	if !ok {
		t.Fatalf("RunSchemaValidation(%q) failed: %s", root, message)
	}
	if message != "ok" {
		t.Fatalf("RunSchemaValidation(%q) message = %q, want ok", root, message)
	}
}

func TestRunSchemaValidationFailsForInvalidContract(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	copyContractSchemas(t, fixture.Root)

	contractPath := filepath.Join(fixture.Root, ".vrooli", "repo-contract.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	doc["$schema"] = "schemas/not-the-repo-contract.schema.json"
	testkitgo.WriteJSON(t, contractPath, doc)

	message, ok := RunSchemaValidation(fixture.Root)
	if ok {
		t.Fatal("expected invalid contract to fail schema validation")
	}
	if !strings.Contains(message, "repo contract validation failed:") {
		t.Fatalf("validation message = %q", message)
	}
}

func TestRunSchemaValidationRejectsParentTraversalPath(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	copyContractSchemas(t, fixture.Root)

	contractPath := filepath.Join(fixture.Root, ".vrooli", "repo-contract.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	layout := doc["layout"].(map[string]any)
	layout["docs_dir"] = "../docs"
	testkitgo.WriteJSON(t, contractPath, doc)

	message, ok := RunSchemaValidation(fixture.Root)
	if ok {
		t.Fatal("expected parent traversal path to fail schema validation")
	}
	if !strings.Contains(message, "must not contain '..' segments") {
		t.Fatalf("validation message = %q", message)
	}
}

func TestRunSchemaValidationRejectsInvalidRuntimeHomeEntryPaths(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError string
	}{
		{name: "absolute", path: "/tmp/vrooli", wantError: "must be repository-relative"},
		{name: "windows drive", path: "C:/vrooli", wantError: "must not use a Windows drive prefix"},
		{name: "parent traversal", path: "state/../secrets", wantError: "must not contain '..' segments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := testkitgo.NewRepoFixture(t)
			fixture.WriteRepoContract(t)
			copyContractSchemas(t, fixture.Root)

			contractPath := filepath.Join(fixture.Root, ".vrooli", "repo-contract.json")
			data, err := os.ReadFile(contractPath)
			if err != nil {
				t.Fatalf("read contract: %v", err)
			}

			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatalf("unmarshal contract: %v", err)
			}
			entries := doc["runtime_home"].(map[string]any)["entries"].(map[string]any)
			state := entries["state"].(map[string]any)
			state["path"] = tt.path
			testkitgo.WriteJSON(t, contractPath, doc)

			message, ok := RunSchemaValidation(fixture.Root)
			if ok {
				t.Fatal("expected invalid runtime home entry path to fail schema validation")
			}
			if !strings.Contains(message, "runtime_home.entries.state.path") || !strings.Contains(message, tt.wantError) {
				t.Fatalf("validation message = %q", message)
			}
		})
	}
}

func copyContractSchemas(t *testing.T, dstRoot string) {
	t.Helper()

	srcRoot := filepath.Join(testkitgo.ProjectRoot(t), ".vrooli", "schemas")
	for _, name := range []string{"common.schema.json", "repo-contract.schema.json"} {
		data, err := os.ReadFile(filepath.Join(srcRoot, name))
		if err != nil {
			t.Fatalf("read schema %s: %v", name, err)
		}
		testkitgo.WriteRelativeFile(t, dstRoot, filepath.ToSlash(filepath.Join(".vrooli", "schemas", name)), string(data))
	}
}
