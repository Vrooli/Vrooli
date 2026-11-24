package activity

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity/activity_v1connect"
)

func TestActivitySummaryAndEventsValidation(t *testing.T) {
	var requestedDays int32
	mux := http.NewServeMux()
	mux.Handle(activityconnect.ActivityServiceGetActivitySummaryProcedure, connect.NewUnaryHandlerSimple(
		activityconnect.ActivityServiceGetActivitySummaryProcedure,
		func(_ context.Context, req *activityv1.GetActivitySummaryRequest) (*activityv1.GetActivitySummaryResponse, error) {
			requestedDays = req.GetDays()
			return &activityv1.GetActivitySummaryResponse{
				Total:    3,
				ByDay:    []*activityv1.ActivityDay{{Date: "2026-11-20", Count: 3}, nil},
				ByActor:  []*activityv1.ActivityActor{{Actor: " ", Count: 3}},
				ByAction: []*activityv1.ActivityAction{{Action: "workspace.save", Count: 3}},
			}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	summary := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "summary")
	out, err := cliapptest.RunCommand(t, summary, core, "--days", " 7 ")
	require.NoError(t, err)
	require.Equal(t, int32(7), requestedDays)
	require.Contains(t, out, "Activity summary: 3 event(s).")
	require.Contains(t, out, "2026-11-20: 3")
	require.Contains(t, out, "(unattributed): 3")
	require.Contains(t, out, "workspace.save: 3")

	_, err = cliapptest.RunCommand(t, summary, core, "--days", "-1")
	require.ErrorContains(t, err, "--days must be a non-negative integer")
	events := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "events")
	_, err = cliapptest.RunCommand(t, events, core, "--limit", "2147483648")
	require.ErrorContains(t, err, "--limit must be a non-negative integer")
}

func TestActivityEmptySummaryAndFormattingFallbacks(t *testing.T) {
	result := (&handlers{}).summaryReport(nil, &activityv1.GetActivitySummaryResponse{
		Empty: true, ByActor: []*activityv1.ActivityActor{{}}, ByAction: []*activityv1.ActivityAction{{}},
	})
	require.Contains(t, result.Summary, "Activity summary: the log is empty (measured zero events).")
	require.Contains(t, result.Results, "activity log: empty (measured zero events)")
	require.Contains(t, result.Results, "  (unattributed): 0")
	require.Contains(t, result.Results, "  (unnamed): 0")
	require.Equal(t, "(nil event)", formatEvent(nil))
	require.Equal(t, "(unknown time) (unattributed) [(no action)] subject — (no summary)", formatEvent(&activityv1.ActivityEvent{}))
	require.Equal(t, "t a [x] kind/id — summary trace=trace-1", formatEvent(&activityv1.ActivityEvent{At: "t", Actor: "a", Action: "x", SubjectKind: "kind", SubjectId: "id", Summary: "summary", TraceId: " trace-1 "}))
}

func TestActivityParseNonNegativeInt32Boundaries(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int32
	}{{"", 0}, {" 0 ", 0}, {"2147483647", 2147483647}} {
		got, err := parseNonNegativeInt32(tc.raw, "--limit")
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, raw := range []string{"-1", "1.5", "x", "2147483648"} {
		_, err := parseNonNegativeInt32(raw, "--limit")
		require.Error(t, err, raw)
	}
}
