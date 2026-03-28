package toolexecution

import (
	"context"
	"testing"
)

func TestServerExecutorUnknownToolReturnsStructuredError(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{})

	result, err := executor.Execute(context.Background(), "does_not_exist", nil)
	if err != nil {
		t.Fatalf("expected structured error result, got %v", err)
	}
	if result.Success {
		t.Fatal("expected unknown tool to fail")
	}
	if result.Code != CodeUnknownTool {
		t.Fatalf("expected unknown_tool code, got %q", result.Code)
	}
}

func TestExecutionResultHelpers(t *testing.T) {
	success := SuccessResult(map[string]string{"status": "ok"})
	if !success.Success || success.IsAsync {
		t.Fatalf("expected synchronous success result, got %+v", success)
	}

	async := AsyncResult("payload", "run-123")
	if !async.Success || !async.IsAsync || async.RunID != "run-123" || async.Status != StatusPending {
		t.Fatalf("expected async result to be pending with run id, got %+v", async)
	}

	failure := ErrorResult("bad input", CodeInvalidArgs)
	if failure.Success || failure.Code != CodeInvalidArgs || failure.Error != "bad input" {
		t.Fatalf("expected structured error result, got %+v", failure)
	}
}
