package executor

import (
	"bytes"
	"context"
	"testing"
)

func TestBoundedRunFailsClosedBeforeProcessStart(t *testing.T) {
	if result := (Bounded{}).Run(context.Background(), Command{WorkspaceID: "w", Hermetic: HermeticPolicy{Network: "future"}}); result.FailureClass != ClassUnsupported {
		t.Fatalf("unsupported policy = %+v", result)
	}
	if result := (Bounded{}).Run(context.Background(), Command{WorkspaceID: "w"}); result.FailureClass != ClassMisconfiguration {
		t.Fatalf("empty executable = %+v", result)
	}
}

func TestBoundedRunCapturesEvidenceAndUsesTemporaryRoot(t *testing.T) {
	result := (Bounded{}).Run(context.Background(), Command{WorkspaceID: "w", Name: "echo-evidence", Executable: "echo", Args: []string{"evidence"}, TimeoutSeconds: 5, CaptureStdout: true, Hermetic: HermeticPolicy{TemporaryRoot: true}, Env: map[string]string{"UNIT_HEALTH_RUN_BRANCH": "yes"}})
	if result.Status != StatusPassed || !bytes.Equal(result.StdoutEvidence, []byte("evidence\n")) || !result.StdoutEvidenceComplete {
		t.Fatalf("evidence result = %+v", result)
	}
}
