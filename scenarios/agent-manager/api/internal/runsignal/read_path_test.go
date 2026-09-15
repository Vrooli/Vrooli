package runsignal

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestReadPathAcceptsClaudeFilePath(t *testing.T) {
	call := &domain.ToolCallEventData{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/example.go"}}
	if got := ReadPath(call); got != "/tmp/example.go" {
		t.Fatalf("ReadPath=%q", got)
	}
}
