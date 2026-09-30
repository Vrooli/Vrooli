//go:build linux

package desktophelper

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"

	"device-control/internal/native/atspi"
	"device-control/internal/sessions"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

func TestSignedHelperInsertsUnicodeIntoGTK(t *testing.T) {
	for _, binary := range []string{"dbus-run-session", "xvfb-run", "/usr/bin/python3"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skip("GTK fixture tool unavailable: " + binary)
		}
	}
	probe := exec.Command("/usr/bin/python3", "-c", "import gi;gi.require_version('Gtk','3.0');from gi.repository import Gtk")
	if probe.Run() != nil {
		t.Skip("GTK Python fixture unavailable")
	}
	directory, err := os.MkdirTemp("", "a11y-")
	require.NoError(t, err)
	defer os.RemoveAll(directory)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resultPath := filepath.Join(directory, "result")
	command := exec.Command("dbus-run-session", "--", "xvfb-run", "-a", "/usr/bin/python3", "../native/atspi/testdata/gtk_fixture.py", resultPath)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "XDG_RUNTIME_DIR=") && !strings.HasPrefix(value, "GSETTINGS_BACKEND=") && !strings.HasPrefix(value, "GIO_USE_VFS=") && !strings.HasPrefix(value, "NO_AT_BRIDGE=") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "XDG_RUNTIME_DIR="+directory, "GSETTINGS_BACKEND=memory", "GIO_USE_VFS=local")
	read, write, err := os.Pipe()
	require.NoError(t, err)
	command.Stdout = write
	require.NoError(t, command.Start())
	write.Close()
	stop := context.AfterFunc(ctx, func() { _ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL) })
	defer func() {
		stop()
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_ = command.Wait()
		read.Close()
	}()
	require.NoError(t, read.SetReadDeadline(time.Now().Add(5*time.Second)))
	var ready struct {
		Address string
		PID     uint32
	}
	scanner := bufio.NewScanner(read)
	for scanner.Scan() {
		if json.Unmarshal(scanner.Bytes(), &ready) == nil && ready.Address != "" {
			break
		}
	}
	require.NotEmpty(t, ready.Address)
	address := strings.Split(ready.Address, ",")[0]
	require.True(t, strings.HasPrefix(address, "unix:path="))
	path := strings.TrimPrefix(address, "unix:path=")
	require.True(t, strings.HasPrefix(path, directory+"/"))
	conn, err := atspi.Dial(ctx, path, func(_ context.Context, pid, uid uint32) error {
		if pid == 0 || uid != uint32(os.Getuid()) {
			return atspi.ErrRefused
		}
		return nil
	})
	require.NoError(t, err)
	defer conn.Close()
	var busID string
	require.NoError(t, conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetId", 0).Store(&busID))
	var names []string
	require.NoError(t, conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names))
	owner := ""
	for _, name := range names {
		if !strings.HasPrefix(name, ":") {
			continue
		}
		var pid uint32
		if conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetConnectionUnixProcessID", 0, name).Store(&pid) == nil && pid == ready.PID {
			owner = name
			break
		}
	}
	require.NotEmpty(t, owner)

	access, err := atspi.New(conn, func(_ context.Context, ref atspi.Ref) error {
		if ref.BusID != busID {
			return atspi.ErrRefused
		}
		return nil
	})
	require.NoError(t, err)
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: busID}
	db := databasetest.NewSQLite(t)
	repo, err := sessions.NewSQLiteDesktopRepository(ctx, db, "gtk-fixture")
	require.NoError(t, err)
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "fixture-host", HostNodeID: "fixture-host"}, OwnerScenario: "device-control", SurfaceID: "fixture-desktop"}

	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	authority, err := sessions.NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	controller, err := sessions.NewDesktopController(repo, authority, backend, surface, "fixture-login", "fixture-helper")
	require.NoError(t, err)
	require.NoError(t, controller.ActivateHelper(ctx, "", 0))
	lease := sessions.DesktopLease{Ref: targetmodel.SessionRef{Surface: surface, SessionID: "fixture-lease", DesktopSessionID: "fixture-login"}, Actor: "fixture-operator", HelperID: "fixture-helper", Epoch: 2, Control: true, ExpiresAt: time.Now().Add(time.Minute)}
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	token, err := sessions.SignDesktopGrant(private, sessions.DesktopGrant{ID: "gtk-grant", Principal: principal, Lease: lease, Operations: []string{"open", "observe", "act", "stop"}, IssuedAt: time.Now(), ExpiresAt: lease.ExpiresAt}, time.Now())
	require.NoError(t, err)
	helper, err := sessions.NewDesktopUnixHelper(controller, authority)
	require.NoError(t, err)
	socket := filepath.Join(directory, "helper.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	require.NoError(t, err)
	helperCtx, stopHelper := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- helper.Serve(helperCtx, listener) }()
	defer func() {
		stopHelper()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("helper cleanup timed out")
		}
	}()
	transport := &http.Transport{DialContext: func(call context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(call, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	wire := desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://helper")
	_, err = wire.Observe(ctx, connect.NewRequest(&desktopv1.ObserveRequest{}))
	require.Error(t, err)
	wireLease := &desktopv1.Lease{Ref: lease.Ref.Proto(), Actor: lease.Actor, HelperId: lease.HelperID, Epoch: lease.Epoch, Control: true, ExpiresAt: timestamppb.New(lease.ExpiresAt)}
	_, err = wire.Open(ctx, semanticRequest(&desktopv1.OpenRequest{Lease: wireLease, ExpectedEpoch: 1}, token))
	require.NoError(t, err)
	_, err = wire.Applications(ctx, connect.NewRequest(&desktopv1.ApplicationsRequest{Lease: wireLease}))
	require.Error(t, err, "discovery requires signed observe authority")
	catalog, err := wire.Applications(ctx, semanticRequest(&desktopv1.ApplicationsRequest{Lease: wireLease}, token))
	require.NoError(t, err)
	require.Len(t, catalog.Msg.Applications, 1)
	application := catalog.Msg.Applications[0]
	require.Equal(t, ready.PID, application.ProcessId)
	require.Equal(t, "gtk_fixture.py", application.Name)
	_, err = wire.Observe(ctx, semanticRequest(&desktopv1.ObserveRequest{Lease: wireLease, ProcessId: ready.PID, ApplicationId: application.ApplicationId, ApplicationRevision: catalog.Msg.Revision}, token))
	require.Error(t, err, "mixed process and application selectors must refuse")
	snapshotResponse, err := wire.Observe(ctx, semanticRequest(&desktopv1.ObserveRequest{Lease: wireLease, ApplicationId: application.ApplicationId, ApplicationRevision: catalog.Msg.Revision}, token))
	require.NoError(t, err)
	snapshot := snapshotResponse.Msg
	require.NotNil(t, snapshot.Semantic)
	var element *desktopv1.SemanticElement
	for _, candidate := range snapshot.Semantic.Elements {
		if candidate.Name == "go-unicode-entry" {
			element = candidate
			break
		}
	}
	require.NotNil(t, element)
	require.NotEmpty(t, element.ParentId)
	require.NotEmpty(t, element.WindowId)
	require.NotEmpty(t, element.Fingerprint)
	require.NotZero(t, snapshot.Semantic.RefreshEpoch)
	var window *desktopv1.SemanticElement
	for _, candidate := range snapshot.Semantic.Elements {
		if candidate.ElementId == element.WindowId {
			window = candidate
		}
	}
	require.NotNil(t, window)
	require.Equal(t, "Go AT-SPI Unicode fixture", window.Name)
	require.True(t, element.Editable)
	require.True(t, element.StateKnown)
	if element.BoundsKnown {
		require.Positive(t, element.Width)
		require.Positive(t, element.Height)
	}
	selector := &desktopv1.SemanticSelector{ObservationRevision: snapshot.Semantic.Revision, WindowId: element.WindowId, Name: element.Name, EditableOnly: true, RefreshEpoch: snapshot.Semantic.RefreshEpoch}
	_, err = wire.Resolve(ctx, connect.NewRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}))
	require.Error(t, err)
	resolved, err := wire.Resolve(ctx, semanticRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}, token))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_UNIQUE, resolved.Msg.Disposition)
	require.Equal(t, []string{element.ElementId}, resolved.Msg.ElementIds)
	selector.Name = "GO-UNICODE-ENTRY"
	selector.MatchMode = desktopv1.SemanticMatchMode_SEMANTIC_MATCH_MODE_NORMALIZED
	normalized, err := wire.Resolve(ctx, semanticRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}, token))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_UNIQUE, normalized.Msg.Disposition)
	require.Equal(t, []string{element.ElementId}, normalized.Msg.ElementIds)
	selector.Name = "go-unicode-entr"
	selector.MatchMode = desktopv1.SemanticMatchMode_SEMANTIC_MATCH_MODE_FUZZY
	selector.Role = element.Role
	fuzzy, err := wire.Resolve(ctx, semanticRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}, token))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_UNIQUE, fuzzy.Msg.Disposition)
	require.Equal(t, []string{element.ElementId}, fuzzy.Msg.ElementIds)
	selector.Role++
	wrongRole, err := wire.Resolve(ctx, semanticRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}, token))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_ABSENT, wrongRole.Msg.Disposition)
	selector.Role = element.Role
	selector.Name = "nonexistent fixture field"
	selector.MatchMode = desktopv1.SemanticMatchMode_SEMANTIC_MATCH_MODE_EXACT
	absent, err := wire.Resolve(ctx, semanticRequest(&desktopv1.ResolveRequest{Lease: wireLease, Selector: selector}, token))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_ABSENT, absent.Msg.Disposition)
	require.Empty(t, absent.Msg.ElementIds)
	claimRequest := &desktopv1.ClaimFlowRequest{Lease: wireLease, RunId: "fixture-flow", Digest: strings.Repeat("a", 64), Steps: 1}
	_, err = wire.ClaimFlow(ctx, connect.NewRequest(claimRequest))
	require.Error(t, err)
	claim, err := wire.ClaimFlow(ctx, semanticRequest(claimRequest, token))
	require.NoError(t, err)
	require.True(t, claim.Msg.Fresh)
	duplicateClaim, err := wire.ClaimFlow(ctx, semanticRequest(claimRequest, token))
	require.NoError(t, err)
	require.False(t, duplicateClaim.Msg.Fresh)
	_, err = wire.FinishFlow(ctx, semanticRequest(&desktopv1.FinishFlowRequest{Lease: wireLease, RunId: claimRequest.RunId, Digest: claimRequest.Digest, Disposition: "passed"}, token))
	require.Error(t, err)
	text := "日本語 العربية café e\u0301 🧪"
	action := &desktopv1.ActRequest{Lease: wireLease, CommandId: "fixture-flow:0", GeometryRevision: snapshot.GeometryRevision, Action: &desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: text, ElementId: element.ElementId, ObservationRevision: snapshot.Semantic.Revision, Position: 2}}}}
	receipt, err := wire.Act(ctx, semanticRequest(action, token))
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Msg.Receipt.Outcome)
	terminal, err := wire.FinishFlow(ctx, semanticRequest(&desktopv1.FinishFlowRequest{Lease: wireLease, RunId: claimRequest.RunId, Digest: claimRequest.Digest, Disposition: "passed"}, token))
	require.NoError(t, err)
	require.EqualValues(t, 1, terminal.Msg.Confirmed)
	completed, err := wire.ClaimFlow(ctx, semanticRequest(claimRequest, token))
	require.NoError(t, err)
	require.False(t, completed.Msg.Fresh)
	require.Equal(t, "passed", completed.Msg.Record.Disposition)

	duplicate, err := wire.Act(ctx, semanticRequest(action, token))
	require.NoError(t, err)
	require.Equal(t, receipt.Msg, duplicate.Msg)
	var persisted string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT state FROM device_control_desktop_sessions").Scan(&persisted))
	require.NotContains(t, persisted, text, "native text must not enter the durable command journal")
	actual, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	require.Equal(t, "é|"+text+"tail", string(actual))
	// An explicit outcome check must read exact current text and never insert it.
	checkedSnapshot, err := wire.Observe(ctx, semanticRequest(&desktopv1.ObserveRequest{Lease: wireLease, ApplicationId: application.ApplicationId, ApplicationRevision: catalog.Msg.Revision}, token))
	require.NoError(t, err)
	var checkedElement *desktopv1.SemanticElement
	for _, candidate := range checkedSnapshot.Msg.Semantic.Elements {
		if candidate.Name == "go-unicode-entry" {
			checkedElement = candidate
		}
	}
	require.NotNil(t, checkedElement)
	check := &desktopv1.AssertTextAction{ExpectedText: "incorrect", ElementId: checkedElement.ElementId, ObservationRevision: checkedSnapshot.Msg.Semantic.Revision}
	assertion := &desktopv1.ActRequest{Lease: wireLease, CommandId: "assert-wrong", GeometryRevision: checkedSnapshot.Msg.GeometryRevision, Action: &desktopv1.Action{Action: &desktopv1.Action_AssertText{AssertText: check}}}
	_, err = wire.Act(ctx, semanticRequest(assertion, token))
	require.Error(t, err)
	check.ExpectedText = string(actual)
	assertion.CommandId = "assert-correct"
	verified, err := wire.Act(ctx, semanticRequest(assertion, token))
	require.NoError(t, err)
	require.Equal(t, "applied", verified.Msg.Receipt.Outcome)
	repeatedCheck, err := wire.Act(ctx, semanticRequest(assertion, token))
	require.NoError(t, err)
	require.Equal(t, verified.Msg, repeatedCheck.Msg)
	afterCheck, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	require.Equal(t, actual, afterCheck)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT state FROM device_control_desktop_sessions").Scan(&persisted))
	require.NotContains(t, persisted, check.ExpectedText)
	// Plain observation must not release cached semantic text or IDs.

	plain, err := wire.Observe(ctx, semanticRequest(&desktopv1.ObserveRequest{Lease: wireLease}, token))
	require.NoError(t, err)
	require.Nil(t, plain.Msg.Semantic)
	require.Empty(t, backend.entries)
	_, err = wire.Stop(ctx, semanticRequest(&desktopv1.StopRequest{Lease: wireLease}, token))
	require.NoError(t, err)
	action.CommandId = "after-stop"
	_, err = wire.Act(ctx, semanticRequest(action, token))
	require.Error(t, err)
	var state sessions.DesktopState
	require.NoError(t, repo.Update(ctx, func(current *sessions.DesktopState) error { state = *current; return nil }))
	require.Nil(t, state.Lease)
	require.False(t, state.CleanupPending)
}

func semanticRequest[T any](message *T, token string) *connect.Request[T] {
	request := connect.NewRequest(message)
	request.Header().Set("Authorization", "DesktopGrant "+token)
	return request
}
