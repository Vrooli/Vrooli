package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/transitions"
)

func TestTransitionRegistryLoadsOnlyAtAPIComposition(t *testing.T) {
	apiRoot := filepath.Join(swarmScenarioRoot(t), "api")
	var sites []string
	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Standalone commands under cmd/ are separate binaries with their own
		// composition; they are not the API server and cannot receive its
		// injected registry. The rule this test protects is that the *server*
		// loads the registry exactly once.
		if rel, relErr := filepath.Rel(apiRoot, path); relErr == nil && strings.HasPrefix(filepath.ToSlash(rel), "cmd/") {
			return nil
		}
		if strings.Contains(string(data), "transitions.LoadDir(") {
			sites = append(sites, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 || filepath.Base(sites[0]) != "main.go" {
		t.Fatalf("transition registry must load once at composition in main.go; sites=%v", sites)
	}
}

func TestWorkflowDefinitionsDoNotRepeatTransitionMapping(t *testing.T) {
	dir := filepath.Join(swarmScenarioRoot(t), ".vrooli", "agent-manager")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"transition"`) {
			t.Fatalf("workflow definition %s repeats registry transition mapping", entry.Name())
		}
	}
}

func TestWorkflowLocatorsDoNotEscapeTransitionRunner(t *testing.T) {
	apiRoot := filepath.Join(swarmScenarioRoot(t), "api")
	registry, err := transitions.LoadDir(filepath.Join(swarmScenarioRoot(t), ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	workflowLiterals := make([]string, 0)
	for _, definition := range registry.Definitions() {
		if definition.Workflow != nil {
			workflowLiterals = append(workflowLiterals, fmt.Sprintf("%q", definition.Workflow.Key))
		}
	}
	var violations []string
	err = filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		pathSlash := filepath.ToSlash(path)
		if strings.Contains(pathSlash, "/internal/transitionrunner/") || strings.Contains(pathSlash, "/internal/agentmanager/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, locator := range workflowLiterals {
			if strings.Contains(string(data), locator) {
				violations = append(violations, path+": "+locator)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("workflow locators must remain registry-private; violations=%v", violations)
	}
}
