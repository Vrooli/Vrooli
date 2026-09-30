package lifecycle

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestPrepareProviderFenceAllowsRestartWhenScenarioIsStopped(t *testing.T) {
	home := t.TempDir()
	runner := &Runner{Home: home, deps: lifecycleDeps{
		runtimeRegistry: func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		},
	}}

	fence, err := runner.prepareProviderFence(t.Context(), scenario.Scenario{Slug: "stopped-scenario"}, false, "", "operation-1")
	if err != nil {
		t.Fatalf("prepareProviderFence() error = %v, want nil", err)
	}
	if fence != nil {
		t.Fatalf("prepareProviderFence() fence = %#v, want nil for stopped scenario", fence)
	}
}
