package scenariocli

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestScenarioStartOperationMessageIncludesProviderFenceLink(t *testing.T) {
	started := time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC)
	message := ScenarioStartOperationMessage(&lifecycle.StartOperationView{
		OperationID: "startop-attempt", FenceOperationID: "restart-provider-fence",
		Scenario: "workspace-sandbox", Variant: "live", Operation: "restart",
		Status: "succeeded", StartedAt: started,
		Steps: []scenarioruntime.StartOperationStep{{Name: "runtime_identity", Status: "done", StartedAt: started,
			Evidence: map[string]string{"runtime_release_id": "release-abc", "start_operation_id": "startop-attempt"}}},
	})
	if message.GetFenceOperationId() != "restart-provider-fence" {
		t.Fatalf("reported fence operation ID = %q, want restart-provider-fence", message.GetFenceOperationId())
	}
	if len(message.GetSteps()) != 1 || message.GetSteps()[0].GetEvidence()["runtime_release_id"] != "release-abc" {
		t.Fatalf("runtime provenance omitted from proto steps: %+v", message.GetSteps())
	}
}
