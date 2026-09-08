package channel

import (
	"context"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/config"

	connectrpc "connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"google.golang.org/protobuf/proto"
)

func TestOpenNodeSessionUsesNativePTYAndResizes(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "creack/pty reports unsupported on Windows; the agent uses the documented pipe fallback")
	}

	c := NewClient(config.Config{}, WithLogger(log.Default()))
	c.openNodeSession("session-pty", &sessionv1.Open{Shell: "/bin/sh"})
	t.Cleanup(func() { c.closeNodeSession("session-pty", "test_done") })

	var current *nodeSession
	require.Eventually(t, func() bool {
		c.mu.Lock()
		current = c.sessions["session-pty"]
		c.mu.Unlock()
		return current != nil
	}, time.Second, time.Millisecond)
	require.NotNil(t, current.terminal, "supported Unix targets must allocate a native PTY")

	c.resizeNodeSession("session-pty", &sessionv1.Resize{Columns: 120, Rows: 40})
}

func TestInteractiveShellDefaultsBeforeCleaningEmptyInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "default shell selection is platform-specific on Windows")
	}
	t.Setenv("SHELL", "")
	require.Equal(t, "/bin/sh", interactiveShell(""))
	require.Equal(t, "/bin/sh", interactiveShell("   "))
}

func TestInteractiveShellFallsBackWhenDefaultIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "POSIX fallback shell selection is platform-specific on Windows")
	}
	t.Setenv("SHELL", "/does/not/exist/zsh")
	require.Equal(t, "/bin/sh", interactiveShell(""))
}

func TestInteractiveCommandEnvMakesConfiguredVrooliBinDiscoverable(t *testing.T) {
	env := interactiveCommandEnv("/Users/test/.vrooli/bin/vrooli", []string{"PATH=/usr/bin:/bin", "HOME=/Users/test"})
	require.Contains(t, env, "HOME=/Users/test")
	var pathValue string
	for _, value := range env {
		if strings.HasPrefix(value, "PATH=") {
			pathValue = strings.TrimPrefix(value, "PATH=")
		}
	}
	require.Equal(t, "/Users/test/.vrooli/bin:/usr/bin:/bin", pathValue)
}

type desktopOwnerFixture struct {
	desktopv1connect.UnimplementedDesktopOwnerServiceHandler
	observedSession string
	actedCommand    string
}

type recordingDesktopAdapter struct {
	commands []*sessionv1.DesktopCommand
	stopped  bool
}

func (a *recordingDesktopAdapter) Execute(_ context.Context, _ *sessionv1.Binding, command *sessionv1.DesktopCommand) (*sessionv1.DesktopResult, error) {
	a.commands = append(a.commands, command)
	return &sessionv1.DesktopResult{CommandId: command.GetCommandId()}, nil
}

func (a *recordingDesktopAdapter) Stop(context.Context, *sessionv1.Binding) error {
	a.stopped = true
	return nil
}

func (f *desktopOwnerFixture) Observe(_ context.Context, req *connectrpc.Request[desktopv1.OwnerObserveRequest]) (*connectrpc.Response[desktopv1.ObserveResponse], error) {
	f.observedSession = req.Msg.Session.GetSessionId()
	return connectrpc.NewResponse(&desktopv1.ObserveResponse{DisplayId: "display", GeometryRevision: "geometry", Width: 1, Height: 1}), nil
}

func (f *desktopOwnerFixture) Act(_ context.Context, req *connectrpc.Request[desktopv1.OwnerActRequest]) (*connectrpc.Response[desktopv1.ActResponse], error) {
	f.actedCommand = req.Msg.CommandId
	return connectrpc.NewResponse(&desktopv1.ActResponse{Receipt: &desktopv1.Receipt{CommandId: req.Msg.CommandId, Outcome: "accepted"}}), nil
}

func (f *desktopOwnerFixture) Stop(context.Context, *connectrpc.Request[desktopv1.OwnerStopRequest]) (*connectrpc.Response[desktopv1.StopResponse], error) {
	return connectrpc.NewResponse(&desktopv1.StopResponse{}), nil
}

func TestUnixDesktopAdapterUsesLeaseScopedOwnerRPC(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "desktop-owner.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	fixture := &desktopOwnerFixture{}
	_, handler := desktopv1connect.NewDesktopOwnerServiceHandler(fixture)
	server := &http.Server{Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); <-done })

	adapter := NewUnixDesktopAdapter(socket)
	binding := &sessionv1.Binding{
		Surface:   &commonv1.SurfaceRef{Target: &commonv1.TargetRef{OwnerScenario: "vrooli-bridge", ResourceId: "node-1", HostNodeId: "node-1"}, OwnerScenario: "device-control", SurfaceId: "desktop"},
		Transport: "desktop", LeaseId: "lease-42", LeaseEpoch: 7, OwnerId: "owner-1", PolicyRevision: "policy-3",
	}
	result, err := adapter.Execute(context.Background(), binding, &sessionv1.DesktopCommand{CommandId: "cmd-1", Operation: sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_OBSERVE})
	require.NoError(t, err)
	require.Equal(t, "lease-42", fixture.observedSession)
	require.Equal(t, "cmd-1", result.GetCommandId())
	require.NotNil(t, result.GetObservation())

	result, err = adapter.Execute(context.Background(), binding, &sessionv1.DesktopCommand{CommandId: "cmd-2", Operation: sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_ACT, Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_MOVE, DisplayId: "display", X: 1, Y: 1}}}})
	require.NoError(t, err)
	require.Equal(t, "cmd-2", fixture.actedCommand)
	require.Equal(t, "accepted", result.GetAct().GetReceipt().GetOutcome())

	_, err = adapter.Execute(context.Background(), binding, &sessionv1.DesktopCommand{CommandId: "cmd-3", Operation: sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_ACT})
	require.ErrorContains(t, err, "requires an action")
	_, err = proto.Marshal(result)
	require.NoError(t, err)
}

func TestDesktopBoundSessionUsesTypedAdapterAndStopsOnClose(t *testing.T) {
	adapter := &recordingDesktopAdapter{}
	c := NewClient(config.Config{}, WithDesktopAdapter(adapter), WithLogger(log.Default()))
	binding := &sessionv1.Binding{
		Surface:   &commonv1.SurfaceRef{Target: &commonv1.TargetRef{OwnerScenario: "vrooli-bridge", ResourceId: "node-1", HostNodeId: "node-1"}, OwnerScenario: "device-control", SurfaceId: "desktop"},
		Transport: "desktop", LeaseId: "lease-42", LeaseEpoch: 7, OwnerId: "owner-1", PolicyRevision: "policy-3",
	}
	c.openNodeSession("desktop-session", &sessionv1.Open{Binding: binding, MaxFrameBytes: 1024})
	command := &sessionv1.DesktopCommand{CommandId: "cmd-1", Operation: sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_OBSERVE}
	raw, err := proto.Marshal(command)
	require.NoError(t, err)
	c.writeNodeSession("desktop-session", &sessionv1.Data{CommandId: "cmd-1", Data: raw})
	require.Len(t, adapter.commands, 1)
	require.Equal(t, "cmd-1", adapter.commands[0].GetCommandId())
	c.closeNodeSession("desktop-session", "revoked")
	require.True(t, adapter.stopped)
}
