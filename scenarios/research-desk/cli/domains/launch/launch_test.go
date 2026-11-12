package launch

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

func TestManifestCoversLaunchService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), launchv1.File_research_desk_v1_launch_launch_proto, "LaunchService")
}

type service struct {
	launchconnect.UnimplementedLaunchServiceHandler
	preflight *launchv1.PreflightRequest
	build     *launchv1.BuildRequest
	control   *launchv1.ControlRequest
	launch    *launchv1.LaunchRequest
}

func (s *service) Launch(_ context.Context, req *connect.Request[launchv1.LaunchRequest]) (*connect.Response[launchv1.LaunchResponse], error) {
	s.launch = req.Msg
	return connect.NewResponse(&launchv1.LaunchResponse{Launch: &launchv1.LaunchRecord{CommissionId: req.Msg.CommissionId, Supersedes: []string{"programme-three-20261111"}}}), nil
}

func (s *service) Preflight(_ context.Context, req *connect.Request[launchv1.PreflightRequest]) (*connect.Response[launchv1.PreflightResponse], error) {
	s.preflight = req.Msg
	return connect.NewResponse(&launchv1.PreflightResponse{
		Blockers: []string{"niche-book-production: programme-three-20261111 live; the launch is refused unless this commission is in run_alongside"},
		Topics: []*launchv1.TopicVerdict{{
			Topic: "niche-book-production", CommissionId: "programme-three-20261111", State: "live", Blocks: true, Message: "live",
			Choices: []launchv1.Resolution{launchv1.Resolution_RESOLUTION_SUPERSEDE, launchv1.Resolution_RESOLUTION_RUN_ALONGSIDE}, RecommendedReason: "it is live: the operator chooses",
		}},
	}), nil
}

func (s *service) Build(_ context.Context, req *connect.Request[launchv1.BuildRequest]) (*connect.Response[launchv1.BuildResponse], error) {
	s.build = req.Msg
	return connect.NewResponse(&launchv1.BuildResponse{Packet: &launchv1.PacketResult{Ok: true, Written: true, CommissionId: "programme-five-20261112", Folder: "/r/cohorts/programme-five-20261112"}}), nil
}

func (s *service) Control(_ context.Context, req *connect.Request[launchv1.ControlRequest]) (*connect.Response[launchv1.ControlResponse], error) {
	s.control = req.Msg
	return connect.NewResponse(&launchv1.ControlResponse{GoalStatus: "paused", Message: "pause done; goal paused", Launch: &launchv1.LaunchRecord{Path: "/r/LAUNCH.json"}}), nil
}

func command(t *testing.T, svc *service, name string) (cliapp.Command, *cliapp.ScenarioApp) {
	t.Helper()
	path, handler := launchconnect.NewLaunchServiceHandler(svc)
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

func TestPreflightSendsAFileAsASpecAndAWordAsACommission(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "preflight")
	spec := filepath.Join(t.TempDir(), "spec.json")
	require.NoError(t, os.WriteFile(spec, []byte(`{"topics":["niche-book-production"]}`), 0o644))
	out, err := cliapptest.RunCommand(t, cmd, core, spec)
	require.NoError(t, err)
	require.Equal(t, `{"topics":["niche-book-production"]}`, svc.preflight.GetSpecJson())
	require.Contains(t, out, "Not launchable (1 blocker(s))")
	require.Contains(t, out, "BLOCK niche-book-production · programme-three-20261111 (live)")

	require.Contains(t, out, "choose --supersede programme-three-20261111 or --alongside programme-three-20261111; recommended: none: it is live: the operator chooses")

	_, err = cliapptest.RunCommand(t, cmd, core, "local-models-w1-20261111", "--alongside", "programme-three-20261111", "--assert-never-started")
	require.NoError(t, err)
	require.Equal(t, "local-models-w1-20261111", svc.preflight.GetCommissionId())
	require.True(t, svc.preflight.AssertNeverStarted)
	require.Len(t, svc.preflight.Resolutions, 1)
	require.Equal(t, launchv1.Resolution_RESOLUTION_RUN_ALONGSIDE, svc.preflight.Resolutions[0].Choice)

	// --spec-file is the named form of a spec, as program callers send it.
	_, err = cliapptest.RunCommand(t, cmd, core, "--spec-file", spec)
	require.NoError(t, err)
	require.Equal(t, `{"topics":["niche-book-production"]}`, svc.preflight.GetSpecJson())
	_, err = cliapptest.RunCommand(t, cmd, core, "local-models-w1-20261111", "--spec-file", spec)
	require.ErrorContains(t, err, "not both")
	_, err = cliapptest.RunCommand(t, cmd, core)
	require.ErrorContains(t, err, "give a prepared commission")
}

// The launch request carries each blocker's choice and the
// operator's assertion.
func TestLaunchCarriesChoicesAndTheAssertion(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "launch")
	out, err := cliapptest.RunCommand(t, cmd, core, "local-models-w1-20261111", "--supersede", "programme-three-20261111, programme-four-20261110")
	require.NoError(t, err)
	require.Len(t, svc.launch.Resolutions, 2)
	for _, r := range svc.launch.Resolutions {
		require.Equal(t, launchv1.Resolution_RESOLUTION_SUPERSEDE, r.Choice)
	}
	require.False(t, svc.launch.AssertNeverStarted)
	require.Contains(t, out, "It supersedes programme-three-20261111")

	_, err = cliapptest.RunCommand(t, cmd, core, "x", "--supersede", "a", "--alongside", "a")
	require.ErrorContains(t, err, "choose one")
}

func TestRelaunchAndBuildUseTheirSources(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "build")
	out, err := cliapptest.RunCommand(t, cmd, core, "--relaunch-of", "programme-four-20261110")
	require.NoError(t, err)
	require.Equal(t, "programme-four-20261110", svc.build.GetRelaunchOf())
	require.Contains(t, out, "Wrote /r/cohorts/programme-five-20261112.")

	_, err = cliapptest.RunCommand(t, cmd, core, filepath.Join(t.TempDir(), "missing.json"))
	require.ErrorContains(t, err, "FILE-CONTRACT.md §2")
}

// [REQ:RD-P0-012]
func TestControlWordsAndWindDown(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "control")
	out, err := cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "pause")
	require.NoError(t, err)
	require.Equal(t, launchv1.ControlAction_CONTROL_ACTION_PAUSE, svc.control.Action)
	require.Contains(t, out, "pause done; goal paused.")

	_, err = cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "discard", "--reason", " Bad start. ")
	require.NoError(t, err)
	require.Equal(t, "Bad start.", svc.control.Reason)

	_, err = cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "wind-down")
	require.ErrorContains(t, err, "say <commission-id> wind-down")
}

func TestAutomaticLaunchLineFollowsTheAccessKind(t *testing.T) {
	line := func(kind activityv1.WebConsoleAccess) string {
		r := optionsReport(nil, &launchv1.GetLaunchOptionsResponse{AutomaticAvailable: true, WebConsoleAccessKind: kind})
		return r.Summary[0]
	}
	require.Contains(t, line(activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL), "service credential")
	policy := line(activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY)
	require.Contains(t, policy, "having no credential")
	require.NotContains(t, policy, "with a credential")
	off := optionsReport(nil, &launchv1.GetLaunchOptionsResponse{AutomaticReason: "Web Console is not running"})
	require.Equal(t, "Automatic launch unavailable: Web Console is not running.", off.Summary[0])
}

func TestPreflightPrintsOwnerReadingAndThePacketsOwnState(t *testing.T) {
	r := preflightReport(nil, &launchv1.PreflightResponse{
		Launchable:   true,
		OwnerReading: []*sharedv1.ReadSet{{Role: "owner-start:ai-video", Bytes: 92000, EstimatedTokens: 23000, Flags: []string{"the ai-video mission is 71041 bytes; the next architect pass tidies the mission"}}},
		PacketState:  &launchv1.StateExcerpt{Path: "/r/cohorts/x/STATE.md", Text: "# State\nDo not start yet.\n", Truncated: true},
	})
	out := strings.Join(r.Results, "\n")
	for _, want := range []string{"owner-start:ai-video reads 92000 bytes", "flag: the ai-video mission", "/r/cohorts/x/STATE.md (start only), verbatim:", "    | Do not start yet."} {
		require.Contains(t, out, want)
	}
}
