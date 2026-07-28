package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:SWM-P0-014] workflow transitions use the shared declared dispatcher.
func TestWorkflowTransportIsOwnedByTransitionRunner(t *testing.T) {
	apiRoot := filepath.Join(swarmScenarioRoot(t), "api")
	var violations []string
	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "/internal/transitionrunner/") || strings.Contains(filepath.ToSlash(path), "/internal/agentmanager/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, call := range []string{"StartWorkflow(", "CollectWorkflow("} {
			if strings.Contains(string(data), call) {
				violations = append(violations, path+": "+call)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("workflow transport escaped transitionrunner: %v", violations)
	}
}
