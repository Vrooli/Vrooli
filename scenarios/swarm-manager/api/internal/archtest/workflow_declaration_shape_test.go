package archtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestWorkflowDeclarationsUseSingleRunSugar guards the authoring contract
// implemented by Agent Manager's catalog expander. A one-run declaration must
// not hand-write the entry, edge, or synthesized end node.
func TestWorkflowDeclarationsUseSingleRunSugar(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	dir := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", ".vrooli", "agent-manager"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var declaration struct {
			EntryNode string            `json:"entryNode"`
			Edges     []json.RawMessage `json:"edges"`
			Nodes     []struct {
				Kind string `json:"kind"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(data, &declaration); err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		runs := 0
		onlyRunAndEnd := true
		for _, node := range declaration.Nodes {
			if node.Kind == "run" {
				runs++
			} else if node.Kind != "end" {
				onlyRunAndEnd = false
			}
		}
		if runs == 1 && onlyRunAndEnd && (len(declaration.Nodes) == 1 || len(declaration.Nodes) > 1) && (declaration.EntryNode != "" || len(declaration.Edges) != 0 || len(declaration.Nodes) > 1) {
			t.Errorf("%s: one-run declaration includes a hand-written terminal graph", entry.Name())
		}
	}
}

func TestWorkflowRegistryLinksMatchDeclarations(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", ".vrooli"))
	registryData, err := os.ReadFile(filepath.Join(root, "swarm-transitions", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var definitions []struct {
		Kind     string `json:"kind"`
		Workflow *struct {
			Key string `json:"key"`
		} `json:"workflow"`
	}
	if err := json.Unmarshal(registryData, &definitions); err != nil {
		t.Fatal(err)
	}
	for _, definition := range definitions {
		if definition.Kind != "workflow" || definition.Workflow == nil {
			continue
		}
		path := filepath.Join(root, "agent-manager", filepath.Base(definition.Workflow.Key)+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("registry workflow %q: %v", definition.Workflow.Key, err)
		}
		var declaration struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(data, &declaration); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if declaration.Key != definition.Workflow.Key {
			t.Errorf("registry workflow %q points to declaration key %q", definition.Workflow.Key, declaration.Key)
		}
	}
}
