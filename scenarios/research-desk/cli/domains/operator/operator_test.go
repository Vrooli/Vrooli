package operator

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

func TestManifestCoversOperatorService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), operatorv1.File_research_desk_v1_operator_operator_proto, "OperatorService")
}

type service struct {
	operatorconnect.UnimplementedOperatorServiceHandler
	entry   *operatorv1.AddEntryRequest
	courier *operatorv1.SetCourierRequest
	seen    *operatorv1.MarkSeenRequest
}

func (s *service) MarkSeen(_ context.Context, req *connect.Request[operatorv1.MarkSeenRequest]) (*connect.Response[operatorv1.MarkSeenResponse], error) {
	s.seen = req.Msg
	if req.Msg.All {
		return connect.NewResponse(&operatorv1.MarkSeenResponse{Commissions: 24, Topics: 27}), nil
	}
	return connect.NewResponse(&operatorv1.MarkSeenResponse{Path: "/s/looks/topic-x.json", Topics: 1}), nil
}

func (s *service) AddEntry(_ context.Context, req *connect.Request[operatorv1.AddEntryRequest]) (*connect.Response[operatorv1.AddEntryResponse], error) {
	s.entry = req.Msg
	text := "\n### 2026-11-12T10:14:03Z · direction · via desk\n" + req.Msg.Text + "\n"
	return connect.NewResponse(&operatorv1.AddEntryResponse{Path: "/r/OPERATOR.md", Entry: &sharedv1.OperatorEntry{Kind: "direction", Via: "desk", Body: req.Msg.Text}, Warning: "steering is off", Text: text, Written: !req.Msg.DryRun}), nil
}

func (s *service) SetCourier(_ context.Context, req *connect.Request[operatorv1.SetCourierRequest]) (*connect.Response[operatorv1.SetCourierResponse], error) {
	s.courier = req.Msg
	return connect.NewResponse(&operatorv1.SetCourierResponse{On: req.Msg.On, Reason: "global ATTENTION.md items"}), nil
}

func (s *service) ListNeedsYou(context.Context, *connect.Request[operatorv1.ListNeedsYouRequest]) (*connect.Response[operatorv1.ListNeedsYouResponse], error) {
	return connect.NewResponse(&operatorv1.ListNeedsYouResponse{
		Courier:        &operatorv1.CourierStatus{HubRead: true, Writer: true},
		Unreadable:     []*sharedv1.Evidence{{Source: "/r/cohorts/c/OPERATOR.md", Reason: "cannot read /r/cohorts/c/OPERATOR.md: permission denied"}},
		ItemCount:      2,
		OlderListCount: 1,
	}), nil
}

func command(t *testing.T, svc *service, name string) (cliapp.Command, *cliapp.ScenarioApp) {
	t.Helper()
	path, handler := operatorconnect.NewOperatorServiceHandler(svc)
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

func TestSayMapsKindsAndShowsTheWarning(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "say")
	out, err := cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "direction", "Put AV6 first.")
	require.NoError(t, err)
	require.Equal(t, operatorv1.EntryKind_ENTRY_KIND_DIRECTION, svc.entry.Kind)
	require.Equal(t, "Put AV6 first.", svc.entry.Text)
	require.Contains(t, out, "Warning: steering is off")

	_, err = cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "enable-steering")
	require.ErrorContains(t, err, "enable-steering <commission-id>")

	out, err = cliapptest.RunCommand(t, cmd, core, "programme-four-20261110", "direction", "Models first.", "--dry-run")
	require.NoError(t, err)
	require.True(t, svc.entry.DryRun)
	require.Contains(t, out, "Dry run: nothing written. This text would be appended to /r/OPERATOR.md:")
	require.Contains(t, out, "### 2026-11-12T10:14:03Z · direction · via desk")
}

// One command records a look at everything; it is mark-seen --all.
func TestMarkSeenAllIsCaughtUp(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "mark-seen")
	out, err := cliapptest.RunCommand(t, cmd, core, "--all")
	require.NoError(t, err)
	require.True(t, svc.seen.All)
	require.Contains(t, out, "a look recorded at 24 commission(s) and 27 topic(s)")

	_, err = cliapptest.RunCommand(t, cmd, core, "topic", "x", "--all")
	require.ErrorContains(t, err, "not both")
	_, err = cliapptest.RunCommand(t, cmd, core)
	require.ErrorContains(t, err, "or --all")
}

func TestCourierSwitchWords(t *testing.T) {
	svc := &service{}
	cmd, core := command(t, svc, "courier")
	_, err := cliapptest.RunCommand(t, cmd, core, "on", "--global")
	require.NoError(t, err)
	require.True(t, svc.courier.On)
	require.True(t, svc.courier.Global)

	_, err = cliapptest.RunCommand(t, cmd, core, "maybe", "x")
	require.ErrorContains(t, err, `"on" or "off"`)
}

// An unreadable file is said, never shown as no items.
func TestNeedsYouSaysAnUnreadableFile(t *testing.T) {
	cmd, core := command(t, &service{}, "needs-you")
	out, err := cliapptest.RunCommand(t, cmd, core)
	require.NoError(t, err)
	require.Contains(t, out, "1 file(s) cannot be read")
	require.Contains(t, out, "unreadable, so unknown: cannot read /r/cohorts/c/OPERATOR.md: permission denied")
	// The count is what still waits; an answered item is listed, not counted.
	require.Contains(t, out, "2 item(s) wait for an answer (answered items are listed, not counted) and 1 older list(s)")
}
