package metrics

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
)

type fakeMetricsService struct{}

func (fakeMetricsService) Snapshot(context.Context) Snapshot {
	return Snapshot{Sessions: SessionMetrics{Created: 1, Deleted: 2, Active: 3, Resizes: 4}, Connections: ConnectionMetrics{Total: 5, Active: 6}, Messages: MessageMetrics{Sent: 7, Received: 8}, Reattach: ReattachMetrics{Attempts: 9, Successes: 10, Failures: 11}, Recovery: RecoveryMetrics{Recovered: 12, OrphanedMeta: 13, OrphanedTmux: 14, AttachRetries: 15, PreservedForNow: 16}, AIGenerations: 17, AISuggestions: 18, VoiceSkipVerificationTotal: 19, Uptime: "up"}
}
func TestConnectHandlerMetricsProjectsSnapshot(t *testing.T) {
	resp, err := NewConnectHandler(Deps{Service: fakeMetricsService{}}).Get(context.Background(), connect.NewRequest(&metricsv1.GetRequest{}))
	if err != nil || resp.Msg.Sessions.Created != 1 || resp.Msg.Recovery.PreservedForFutureRecovery != 16 || resp.Msg.AiSuggestions != 18 {
		t.Fatalf("metrics: %#v %v", resp, err)
	}
}
