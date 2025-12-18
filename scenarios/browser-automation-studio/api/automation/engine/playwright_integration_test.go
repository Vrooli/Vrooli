package engine

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// Requires a running Playwright driver (PLAYWRIGHT_DRIVER_URL). Skipped otherwise.
func TestPlaywrightEngine_IntegrationSmoke(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		t.Skip("PLAYWRIGHT_DRIVER_URL not set; skipping integration smoke test")
	}

	eng, err := NewPlaywrightEngine(nil)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	sess, err := eng.StartSession(context.Background(), SessionSpec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		ViewportWidth:  1200,
		ViewportHeight: 800,
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	defer sess.Close(context.Background())

	out, err := sess.Run(context.Background(), contracts.CompiledInstruction{
		Index:  0,
		NodeID: "nav-1",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: "https://example.com"},
			},
		},
	})
	if err != nil {
		t.Fatalf("run navigate: %v", err)
	}
	if !out.Success {
		t.Fatalf("navigate failed: %+v", out.Failure)
	}
	if out.Screenshot == nil {
		t.Fatalf("expected screenshot in outcome")
	}
}
