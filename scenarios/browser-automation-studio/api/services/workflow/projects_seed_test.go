package workflow

import (
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestSeedWorkflowDefinitionIncludesReplayEvidence(t *testing.T) {
	t.Parallel()

	definition := seedWorkflowDefinition()
	if got, want := len(definition.GetNodes()), 2; got != want {
		t.Fatalf("seed nodes = %d, want %d", got, want)
	}
	if got, want := len(definition.GetEdges()), 1; got != want {
		t.Fatalf("seed edges = %d, want %d", got, want)
	}
	capture := definition.GetNodes()[1].GetAction()
	if capture == nil || capture.GetType() != basactions.ActionType_ACTION_TYPE_SCREENSHOT {
		t.Fatalf("seed terminal action = %v, want screenshot", capture.GetType())
	}
	if params := capture.GetScreenshot(); params == nil || !params.GetFullPage() {
		t.Fatalf("seed screenshot must capture full page: %+v", params)
	}
}
