package dependencies

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type fakeScenarioStatusFetcher struct {
	statuses map[string]string
	errs     map[string]error
}

func (f fakeScenarioStatusFetcher) ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error) {
	if err := f.errs[name]; err != nil {
		return nil, err
	}
	status := f.statuses[name]
	if status == "" {
		status = "running"
	}
	return &cliv1.ScenarioStatusSingle{
		Success: true,
		Scenario: &cliv1.ScenarioStatusItem{
			Name:   name,
			Status: status,
		},
	}, nil
}

func TestScenarioDependencyCheckerFailsStoppedRequiredDependency(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".vrooli", "service.json"), `{
  "dependencies": {
    "scenarios": {
      "knowledge-observatory": {"required": true, "startup_policy": "must_start"}
    }
  }
}`)
	checker := NewScenarioDependencyChecker(dir, DefaultSettings().Scenarios, fakeScenarioStatusFetcher{
		statuses: map[string]string{"knowledge-observatory": "stopped"},
	})
	result := checker.Check(context.Background())
	if result.Success {
		t.Fatal("expected stopped dependency failure")
	}
	if !strings.Contains(result.Error.Error(), "knowledge-observatory") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestScenarioDependencyCheckerWarnPolicyDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".vrooli", "service.json"), `{
  "dependencies": {
    "scenarios": {
      "cli-health": {"required": true}
    }
  }
}`)
	settings := DefaultSettings().Scenarios
	settings.HealthPolicy = "warn"
	checker := NewScenarioDependencyChecker(dir, settings, fakeScenarioStatusFetcher{
		errs: map[string]error{"cli-health": fmt.Errorf("unavailable")},
	})
	result := checker.Check(context.Background())
	if !result.Success {
		t.Fatalf("expected warn policy to pass, got %v", result.Error)
	}
}
