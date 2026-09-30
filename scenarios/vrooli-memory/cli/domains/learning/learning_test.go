package learning

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning/learningv1connect"
)

type flags struct {
	cliapp.OperationContext
	values map[string]string
}

func (f flags) Flag(k string) string { return f.values[k] }

type client struct {
	rpc.LearningServiceClient
	recorded *pb.RecordAttemptRequest
	measured *pb.MeasureLearningRequest
}

func (c *client) RecordAttempt(_ context.Context, r *connect.Request[pb.RecordAttemptRequest]) (*connect.Response[pb.RecordAttemptResponse], error) {
	c.recorded = r.Msg
	return connect.NewResponse(&pb.RecordAttemptResponse{EntryId: "entry"}), nil
}
func (c *client) MeasureLearning(_ context.Context, r *connect.Request[pb.MeasureLearningRequest]) (*connect.Response[pb.MeasureLearningResponse], error) {
	c.measured = r.Msg
	return connect.NewResponse(&pb.MeasureLearningResponse{Scope: r.Msg.Scope, Reason: "unreliable:no_eligible_attempts"}), nil
}
func TestRecordRejectsMalformedAndUnknownFieldsBeforeNetwork(t *testing.T) {
	c := &client{}
	h := handlers{c}
	for _, s := range []string{"broken", `{"madeUp":true}`} {
		_, err := h.record(flags{values: map[string]string{"attempt": s}})
		if err == nil || c.recorded != nil {
			t.Fatal("invalid JSON reached server")
		}
	}
}
func TestTypedCaptureAndWindowSelectors(t *testing.T) {
	c := &client{}
	h := handlers{c}
	_, err := h.record(flags{values: map[string]string{"scope": "desktop-usage", "attempt": `{"attemptId":"attempt"}`}})
	if err != nil || c.recorded.Scope != "desktop-usage" || c.recorded.Attempt.AttemptId != "attempt" {
		t.Fatal(c.recorded, err)
	}
	_, err = h.measure(flags{values: map[string]string{"scope": "desktop-usage", "from": "start", "to": "end", "operation": "inspect", "context-key": "linux-v1"}})
	if err != nil || c.measured.ContextKey != "linux-v1" || c.measured.From != "start" || c.measured.To != "end" {
		t.Fatal(c.measured, err)
	}
}
func TestUnknownEffortRemainsUnknownInHumanReport(t *testing.T) {
	h := handlers{}
	r := h.measureReport(nil, &pb.MeasureLearningResponse{Cohorts: []*pb.Cohort{{Operation: "inspect", UnresolvedTasks: 2}}})
	text := strings.Join(r.Results, "\n")
	if !strings.Contains(text, "success median unknown") || !strings.Contains(text, "2 unresolved") {
		t.Fatal(text)
	}
}
