package system

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestPlacementObservationAvoidsArtifactAuditsAndPreservesDrift(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli resource status ollama --health-only --no-fast --json"] = testutil.MockResponse{Output: []byte(`{"resource":{"running":true,"declared_mode":"cuda","observed_mode":"cpu","mode_drift":true,"artifact_verification":"not_evaluated"}}`)}
	reporter := CLIPlacementReporter{Executor: executor, Resources: []string{"ollama"}}
	drifted, err := reporter.DriftedResources(context.Background())
	if err != nil || len(drifted) != 1 || drifted[0].Name != "ollama" || drifted[0].Declared != "cuda" || drifted[0].Observed != "cpu" {
		t.Fatalf("runtime-only placement observation lost drift: %+v %v", drifted, err)
	}
}
