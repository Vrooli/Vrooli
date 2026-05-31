package queue

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestTokensFromRun(t *testing.T) {
	tests := []struct {
		name string
		run  *domainpb.Run
		want int
	}{
		{
			name: "present",
			run:  &domainpb.Run{Summary: &domainpb.RunSummary{TokensUsed: 12345}},
			want: 12345,
		},
		{
			name: "absent summary is unknown (0)",
			run:  &domainpb.Run{},
			want: 0,
		},
		{
			name: "zero tokens reported",
			run:  &domainpb.Run{Summary: &domainpb.RunSummary{TokensUsed: 0}},
			want: 0,
		},
		{
			name: "nil run is unknown (0)",
			run:  nil,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokensFromRun(tt.run); got != tt.want {
				t.Fatalf("tokensFromRun = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMapAgentManagerResult_CapturesTokens(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status:  domainpb.RunStatus_RUN_STATUS_COMPLETE,
		Summary: &domainpb.RunSummary{Description: "done", TokensUsed: 4096},
	}
	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "tok-task"}, "agent-tag", "output", nil)
	if resp.TokensUsed != 4096 {
		t.Fatalf("expected TokensUsed=4096, got %d", resp.TokensUsed)
	}
}
