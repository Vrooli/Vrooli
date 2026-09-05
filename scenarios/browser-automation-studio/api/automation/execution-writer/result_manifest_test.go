package executionwriter

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildResultManifestPayloadUsesStableFallbackWithoutTimeline(t *testing.T) {
	executionID := uuid.New()
	payload, err := buildResultManifestPayload(executionID, &ExecutionResultData{WorkflowID: "workflow-1"}, nil)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if payload["execution_id"] != executionID.String() || payload["workflow_id"] != "workflow-1" {
		t.Fatalf("unexpected fallback manifest: %#v", payload)
	}
}
