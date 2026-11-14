package activity

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity/activity_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

func TestManifestCoversActivityService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), activityv1.File_research_desk_v1_activity_activity_proto, "ActivityService")
}

type service struct {
	activityconnect.UnimplementedActivityServiceHandler
	commission *activityv1.GetCommissionResponse
	states     []activityv1.CommissionState
	changes    *activityv1.GetChangesRequest
	overview   *activityv1.GetOverviewResponse
}

func (s *service) GetOverview(context.Context, *connect.Request[activityv1.GetOverviewRequest]) (*connect.Response[activityv1.GetOverviewResponse], error) {
	return connect.NewResponse(s.overview), nil
}

func (s *service) ListCommissions(_ context.Context, req *connect.Request[activityv1.ListCommissionsRequest]) (*connect.Response[activityv1.ListCommissionsResponse], error) {
	s.states = req.Msg.States
	return connect.NewResponse(&activityv1.ListCommissionsResponse{}), nil
}

func (s *service) GetCommission(context.Context, *connect.Request[activityv1.GetCommissionRequest]) (*connect.Response[activityv1.GetCommissionResponse], error) {
	return connect.NewResponse(s.commission), nil
}

func (s *service) GetChanges(_ context.Context, req *connect.Request[activityv1.GetChangesRequest]) (*connect.Response[activityv1.GetChangesResponse], error) {
	s.changes = req.Msg
	if req.Msg.Since == nil {
		since := timestamppb.New(time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC))
		return connect.NewResponse(&activityv1.GetChangesResponse{Since: since, BaselineKind: activityv1.Baseline_BASELINE_LAST_DAY, Baseline: "no look recorded; the last 24 hours, since 2026-11-11T09:00:00Z", NotShown: []string{"Workstream status changes need a recorded look; mark this seen to start one."}}), nil
	}
	return connect.NewResponse(&activityv1.GetChangesResponse{Since: req.Msg.Since, Baseline: "since 2026-11-11T19:00:00Z as asked"}), nil
}

func command(t *testing.T, svc *service, name string) (cliapp.Command, *cliapp.ScenarioApp) {
	t.Helper()
	path, handler := activityconnect.NewActivityServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	for _, cmd := range group.Commands {
		if cmd.Name == name {
			return cmd, core
		}
	}
	t.Fatalf("no command %q", name)
	return cliapp.Command{}, nil
}

func TestCommissionsMapsStateWords(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "commissions")
	_, err := cliapptest.RunCommand(t, cmd, core, "--state", "live", "--state", "paused")
	require.NoError(t, err)
	require.Equal(t, []activityv1.CommissionState{activityv1.CommissionState_COMMISSION_STATE_LIVE, activityv1.CommissionState_COMMISSION_STATE_PAUSED}, svc.states)

	_, err = cliapptest.RunCommand(t, cmd, core, "--state", "running")
	require.Error(t, err, "a state outside the closed vocabulary is refused")
}

func TestCommissionShowsSourcesAndUnknowns(t *testing.T) {
	at := timestamppb.New(time.Date(2026, 11, 10, 14, 50, 0, 0, time.UTC))
	svc := &service{commission: &activityv1.GetCommissionResponse{
		Commission: &activityv1.CommissionSummary{
			Id:             "programme-three-20261111",
			State:          activityv1.CommissionState_COMMISSION_STATE_PAUSED,
			StateRule:      "supervisor window open; goal paused",
			Title:          &sharedv1.DerivedText{Value: "Programme three", Evidence: &sharedv1.Evidence{Known: true, Source: "/r/cohorts/programme-three-20261111/README.md"}},
			Started:        &sharedv1.DerivedTime{Value: at, Evidence: &sharedv1.Evidence{Known: true, Source: "/codex/b7/goals_1.sqlite"}},
			AllowanceEnd:   &sharedv1.DerivedTime{Evidence: &sharedv1.Evidence{Reason: "KERNEL.txt names no \"<N>h allowance\""}},
			TopicsEvidence: &sharedv1.Evidence{Reason: "no topic evidence"},
		},
		Ceilings: &sharedv1.Evidence{Reason: "limits.json missing"},
		ReadSets: []*sharedv1.ReadSet{{Role: "scheduler-start", Bytes: 42677, EstimatedTokens: 10669, Flags: []string{"charter 36128 bytes is over the 25000-byte gate"}}},
	}}
	cmd, core := command(t, svc, "commission")
	out, err := cliapptest.RunCommand(t, cmd, core, "programme-three-20261111")
	require.NoError(t, err)
	require.Contains(t, out, "Programme three  [/r/cohorts/programme-three-20261111/README.md]")
	require.Contains(t, out, "[/codex/b7/goals_1.sqlite]")
	require.Contains(t, out, `unknown: KERNEL.txt names no "<N>h allowance"`)
	require.Contains(t, out, "unknown: no topic evidence")
	require.Contains(t, out, "unknown: limits.json missing")
	require.Contains(t, out, "flag:         charter 36128 bytes is over the 25000-byte gate")
	require.Contains(t, out, "session:        none names this commission")
}

func TestChangesSubject(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "changes")
	out, err := cliapptest.RunCommand(t, cmd, core, "topic", "niche-book-production")
	require.NoError(t, err)
	require.Equal(t, activityv1.ChangeSubject_CHANGE_SUBJECT_TOPIC, svc.changes.Subject)
	require.Contains(t, out, "no look recorded; the last 24 hours", "with no look and no time the last day is reported and says so")
	require.Contains(t, out, "not shown: Workstream status changes need a recorded look", "what cannot be shown without a look is said")

	out, err = cliapptest.RunCommand(t, cmd, core, "commission", "programme-four-20261110", "--since", "2026-11-11T19:00:00Z")
	require.NoError(t, err)
	require.True(t, svc.changes.Since.AsTime().Equal(time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)))
	require.Contains(t, out, "as asked")

	_, err = cliapptest.RunCommand(t, cmd, core, "commission", "x", "--since", "yesterday")
	require.ErrorContains(t, err, "--since takes a UTC time")

	_, err = cliapptest.RunCommand(t, cmd, core, "cohort", "x")
	require.ErrorContains(t, err, `"commission" or "topic"`)
}

// `now` always says how the desk reaches Web Console and how the
// courier is, available or not.
func TestNowSaysTheWebConsoleAccessAndTheCourier(t *testing.T) {
	svc := &service{overview: &activityv1.GetOverviewResponse{Lenses: []*activityv1.Lens{
		{Name: "codex sessions", Available: true, Reason: "/codex"},
		{Name: "web-console access", Available: true, Reason: "Web Console access: local availability policy (no desk credential)"},
		{Name: "courier", Available: false, Reason: "no pass yet; the desk cannot withdraw an ask once sent"},
		{Name: "notification-hub", Available: false, Reason: "notification-hub is not running: no port", Command: "vrooli scenario start notification-hub"},
	}}}
	cmd, core := command(t, svc, "now")
	out, err := cliapptest.RunCommand(t, cmd, core)
	require.NoError(t, err)
	require.Contains(t, out, "Web Console access: local availability policy (no desk credential)")
	require.Contains(t, out, "Courier: no pass yet; the desk cannot withdraw an ask once sent")
	require.NotContains(t, out, "codex sessions")
	// The reason names no command; the CLI prints the lens's command beside it.
	require.Contains(t, out, "lens notification-hub unavailable: notification-hub is not running: no port (next: vrooli scenario start notification-hub)")
}
