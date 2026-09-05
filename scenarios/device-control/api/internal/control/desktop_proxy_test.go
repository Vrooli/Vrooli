package control

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"image"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	"device-control/internal/desktophelper"
	internalflows "device-control/internal/flows"
	"device-control/internal/sessions"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

type proxyNative struct {
	releases           atomic.Int32
	effects            atomic.Int32
	failAfterEffect    atomic.Bool
	failAtEffect       atomic.Int32
	blockedObservation atomic.Pointer[proxyObservationGate]
}

type proxyObservationGate struct{ entered chan struct{} }

func (*proxyNative) Validate(context.Context, sessions.DesktopCommand) error { return nil }
func (n *proxyNative) Apply(_ context.Context, command sessions.DesktopCommand) error {
	var action desktopv1.Action
	if err := protojson.Unmarshal(command.Payload, &action); err != nil {
		return err
	}
	if assertion := action.GetAssertText(); assertion != nil {
		if assertion.ExpectedText != "日本語" {
			return errors.New("fixture assertion mismatch")
		}
		return nil
	}
	effect := n.effects.Add(1)
	if n.failAfterEffect.Load() || n.failAtEffect.Load() == effect {
		return errors.New("native acknowledgement lost after mutation")
	}
	return nil
}
func (n *proxyNative) ReleaseHeld(context.Context) error { n.releases.Add(1); return nil }
func (*proxyNative) VerifyWindowProcess(_ context.Context, window uint64, pid uint32) error {
	if window != 42 || pid != uint32(os.Getpid()) {
		return sessions.ErrDesktopAdmission
	}
	return nil
}

func (*proxyNative) CaptureActivation(context.Context) (sessions.DesktopActivationContext, error) {
	return sessions.DesktopActivationContext{ActiveWindow: 991, ProcessID: 992, DisplayID: "display", GeometryRevision: "geometry", PointerX: -20, PointerY: 30, CapturedAt: time.Now()}, nil
}

func (*proxyNative) Observe(context.Context) (sessions.DesktopObservation, error) {
	return sessions.DesktopObservation{Image: image.NewRGBA(image.Rect(0, 0, 4, 4)), DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}, nil
}

func TestDesktopOwnerProxyKeepsGrantInternalAndReleasesLease(t *testing.T) {
	svc, db := testService(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o700))
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	config := desktophelper.Config{StateDirectory: dir, GrantStatusFile: filepath.Join(dir, "grants.json")}
	registration := desktophelper.Registration{Surface: surface, SessionID: "login", HelperID: "helper", Epoch: 1}
	owner, err := NewLocalDesktopAdmission(svc, private, "fake", surface, func(context.Context) (desktophelper.Registration, error) { return registration, nil })
	require.NoError(t, err)
	// The helper reads the publication file, so Open would fail if owner Open
	// raced the first periodic snapshot instead of awaiting its acknowledgement.
	authority, err := sessions.NewSignedDesktopAuthority(public, func(_ context.Context, id string) (bool, error) {
		data, err := os.ReadFile(config.GrantStatusFile)
		if err != nil {
			return false, err
		}
		var state desktophelper.GrantStatus
		if err = json.Unmarshal(data, &state); err != nil {
			return false, err
		}
		if !time.Now().Before(state.ExpiresAt) {
			return false, nil
		}
		for _, active := range state.Active {
			if active == id {
				return true, nil
			}
		}
		return false, nil
	})
	require.NoError(t, err)
	repo, err := sessions.NewSQLiteDesktopRepository(context.Background(), db, "login")
	require.NoError(t, err)
	native := &proxyNative{}
	controller, err := sessions.NewDesktopController(repo, authority, native, surface, "login", "helper")
	require.NoError(t, err)
	require.NoError(t, controller.ActivateHelper(context.Background(), "", 0))
	helper, err := sessions.NewDesktopUnixHelper(controller, authority)
	require.NoError(t, err)
	helperListener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "helper.sock"), Net: "unix"})
	require.NoError(t, err)
	ownerListener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "owner.sock"), Net: "unix"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	helperDone, ownerDone := make(chan error, 1), make(chan error, 1)
	go func() { helperDone <- helper.Serve(ctx, helperListener) }()
	go func() { ownerDone <- owner.serveDesktopOwner(ctx, ownerListener, config) }()
	t.Cleanup(func() {
		cancel()
		for _, done := range []chan error{ownerDone, helperDone} {
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Error("shutdown timed out")
			}
		}
	})
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", ownerListener.Addr().String())
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := desktopv1connect.NewDesktopOwnerServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://owner")
	accountClient := desktopv1connect.NewDesktopAccountServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://owner")
	_, err = accountClient.Open(ctx, connect.NewRequest(&desktopv1.OwnerOpenRequest{Surface: surface.Proto(), TtlSeconds: 60}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "account service cannot inherit local Unix authority")
	described, err := client.Describe(ctx, connect.NewRequest(&desktopv1.OwnerDescribeRequest{}))
	require.NoError(t, err)
	descriptor, err := targetmodel.SurfaceDescriptorFromProto(described.Msg.Surface)
	require.NoError(t, err)
	require.Equal(t, surface, descriptor.Ref)
	require.Equal(t, "login", descriptor.DesktopSessionID)
	for _, fact := range descriptor.Capabilities {
		require.Equal(t, targetmodel.CapabilityUnknown, fact.State)
	}
	require.Empty(t, svc.ListLiveSessions(), "catalog inspection must not acquire desktop authority")
	cancelledRequest := &desktopv1.OwnerOpenRequest{Surface: surface.Proto(), TtlSeconds: 60, Control: true, RequestId: "57c93a1c-2d6c-49f9-a4cf-13fdcc5b3102"}
	releases := native.releases.Load()
	cancelled, err := client.ReconcileOpen(ctx, connect.NewRequest(cancelledRequest))
	require.NoError(t, err)
	require.Equal(t, "not_admitted", cancelled.Msg.State)
	require.Nil(t, cancelled.Msg.Session)
	_, err = client.Open(ctx, connect.NewRequest(cancelledRequest))
	require.Error(t, err, "late arrival must not revive the cancelled request")
	require.Empty(t, svc.ListLiveSessions())
	require.Equal(t, releases, native.releases.Load(), "reconciliation must not call the helper")
	openRequest := &desktopv1.OwnerOpenRequest{Surface: surface.Proto(), TtlSeconds: 60, Control: true, RequestId: "91dc6d09-73b3-4645-8bc0-a51edc36e7cf"}
	opened, err := client.Open(ctx, connect.NewRequest(openRequest))
	require.NoError(t, err)
	require.NotEmpty(t, opened.Msg.Session.SessionId)
	releases = native.releases.Load()
	disposition, err := client.ReconcileOpen(ctx, connect.NewRequest(openRequest))
	require.NoError(t, err)
	require.Equal(t, "forwarding", disposition.Msg.State)
	require.True(t, proto.Equal(opened.Msg.Session, disposition.Msg.Session))
	_, err = client.Open(ctx, connect.NewRequest(openRequest))
	require.Error(t, err)
	require.Equal(t, releases, native.releases.Load(), "duplicate/reconciliation cannot replay helper admission")

	_, err = svc.Acquire("fake", "flow", time.Minute)
	require.Error(t, err)
	observed, err := client.Observe(ctx, connect.NewRequest(&desktopv1.OwnerObserveRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	require.NotEmpty(t, observed.Msg.Png)
	captured, err := client.CaptureActivation(ctx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	require.NotEmpty(t, captured.Msg.ContextId)
	reread, err := client.ReadActivation(ctx, connect.NewRequest(&desktopv1.OwnerReadActivationRequest{Session: opened.Msg.Session, ContextId: captured.Msg.ContextId}))
	require.NoError(t, err)
	require.True(t, proto.Equal(captured.Msg, reread.Msg))
	companion, err := client.CaptureCompanionActivation(ctx, connect.NewRequest(&desktopv1.OwnerCompanionActivationRequest{Session: opened.Msg.Session, CompanionWindow: 42}))
	require.NoError(t, err)
	require.NotEmpty(t, companion.Msg.ContextId)
	_, err = client.CaptureCompanionActivation(ctx, connect.NewRequest(&desktopv1.OwnerCompanionActivationRequest{Session: opened.Msg.Session, CompanionWindow: 43}))
	require.Error(t, err)

	wrongContextSession := proto.Clone(opened.Msg.Session).(*commonv1.SessionRef)
	wrongContextSession.DesktopSessionId = "other-login"
	_, err = client.ReadActivation(ctx, connect.NewRequest(&desktopv1.OwnerReadActivationRequest{Session: wrongContextSession, ContextId: captured.Msg.ContextId}))
	require.Error(t, err)

	catalog, err := client.Applications(ctx, connect.NewRequest(&desktopv1.OwnerApplicationsRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	require.Len(t, catalog.Msg.Applications, 1)
	selected, err := client.Observe(ctx, connect.NewRequest(&desktopv1.OwnerObserveRequest{Session: opened.Msg.Session, ApplicationId: catalog.Msg.Applications[0].ApplicationId, ApplicationRevision: catalog.Msg.Revision}))
	require.NoError(t, err)
	require.Equal(t, uint32(42), selected.Msg.Semantic.ProcessId)
	resolved, err := client.Resolve(ctx, connect.NewRequest(&desktopv1.OwnerResolveRequest{Session: opened.Msg.Session, Selector: &desktopv1.SemanticSelector{ObservationRevision: "fields", WindowId: "window", Name: "Entry", EditableOnly: true}}))
	require.NoError(t, err)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_UNIQUE, resolved.Msg.Disposition)
	require.Equal(t, []string{"field"}, resolved.Msg.ElementIds)

	request := &desktopv1.OwnerActRequest{Session: opened.Msg.Session, CommandId: "once", GeometryRevision: observed.Msg.GeometryRevision, Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, Button: desktopv1.PointerAction_BUTTON_PRIMARY, DisplayId: "display", X: 1, Y: 1}}}}
	receipt, err := client.Act(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Msg.Receipt.Outcome)
	_, err = client.Act(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	require.EqualValues(t, 1, native.effects.Load())
	args, err := structpb.NewStruct(map[string]any{"window": "Editor", "text": "日本語"})
	require.NoError(t, err)
	flowRequest := &desktopv1.OwnerRunFlowRequest{Session: opened.Msg.Session, RunId: "flow-run", ApplicationId: "app", ApplicationRevision: "catalog", Flow: &flowsv1.Flow{Transport: "desktop", Steps: []*flowsv1.Step{{Id: "insert", Kind: "desktop-text", Target: "Entry", Arguments: args}}}}
	flowResult, err := client.RunFlow(ctx, connect.NewRequest(flowRequest))
	require.NoError(t, err)
	require.Equal(t, "passed", flowResult.Msg.Disposition)
	require.EqualValues(t, 1, flowResult.Msg.Confirmed)
	require.EqualValues(t, 2, native.effects.Load())
	repeated, err := client.RunFlow(ctx, connect.NewRequest(flowRequest))
	require.NoError(t, err)
	require.Equal(t, flowResult.Msg.Disposition, repeated.Msg.Disposition)
	require.EqualValues(t, 2, native.effects.Load())
	flowRequest.Flow.Name = "changed intent"
	_, err = client.RunFlow(ctx, connect.NewRequest(flowRequest))
	require.Error(t, err)
	require.EqualValues(t, 2, native.effects.Load())

	// The first mutation can take effect even when its acknowledgement is lost.
	// Neither the next step nor a repeated request may repeat that uncertain work.
	native.failAfterEffect.Store(true)
	unknownRequest := &desktopv1.OwnerRunFlowRequest{Session: opened.Msg.Session, RunId: "unknown-run", ApplicationId: "app", ApplicationRevision: "catalog", Flow: &flowsv1.Flow{Transport: "desktop", Steps: []*flowsv1.Step{
		{Id: "uncertain", Kind: "desktop-text", Target: "Entry", Arguments: args},
		{Id: "must-not-run", Kind: "desktop-text", Target: "Entry", Arguments: args},
	}}}
	unknown, err := client.RunFlow(ctx, connect.NewRequest(unknownRequest))
	require.NoError(t, err)
	require.Equal(t, "incomplete", unknown.Msg.Disposition)
	require.Zero(t, unknown.Msg.Confirmed)
	require.EqualValues(t, 2, unknown.Msg.Steps)
	require.EqualValues(t, 3, native.effects.Load())
	native.failAfterEffect.Store(false)
	repeatedUnknown, err := client.RunFlow(ctx, connect.NewRequest(unknownRequest))
	require.NoError(t, err)
	require.Equal(t, unknown.Msg.Disposition, repeatedUnknown.Msg.Disposition)
	require.Zero(t, repeatedUnknown.Msg.Confirmed)
	require.Equal(t, unknown.Msg.Digest, repeatedUnknown.Msg.Digest)
	require.EqualValues(t, 3, native.effects.Load(), "a recovered helper must not replay an uncertain flow")

	native.failAtEffect.Store(5)
	prefixRequest := &desktopv1.OwnerRunFlowRequest{Session: opened.Msg.Session, RunId: "prefix-run", ApplicationId: "app", ApplicationRevision: "catalog", Flow: &flowsv1.Flow{Transport: "desktop", Steps: []*flowsv1.Step{
		{Id: "confirmed", Kind: "desktop-text", Target: "Entry", Arguments: args},
		{Id: "uncertain", Kind: "desktop-text", Target: "Entry", Arguments: args},
		{Id: "must-not-run", Kind: "desktop-text", Target: "Entry", Arguments: args},
	}}}
	prefix, err := client.RunFlow(ctx, connect.NewRequest(prefixRequest))
	require.NoError(t, err)
	require.Equal(t, "incomplete", prefix.Msg.Disposition)
	require.EqualValues(t, 1, prefix.Msg.Confirmed)
	require.EqualValues(t, 3, prefix.Msg.Steps)
	require.EqualValues(t, 5, native.effects.Load())
	native.failAtEffect.Store(0)
	repeatedPrefix, err := client.RunFlow(ctx, connect.NewRequest(prefixRequest))
	require.NoError(t, err)
	require.Equal(t, prefix.Msg.Disposition, repeatedPrefix.Msg.Disposition)
	require.Equal(t, prefix.Msg.Confirmed, repeatedPrefix.Msg.Confirmed)
	require.Equal(t, prefix.Msg.Digest, repeatedPrefix.Msg.Digest)
	require.EqualValues(t, 5, native.effects.Load(), "duplicate must not replay the confirmed or uncertain step")
	// Terminal provenance survives repository reconstruction and is inaccessible
	// through a different actor or surface, even with the exact run identifier.
	reopenedRuns, err := internalflows.NewSQLiteDesktopRuns(ctx, db)
	require.NoError(t, err)
	scope := internalflows.DesktopRunScope{Actor: owner.principal.String(), DeviceID: "fake", Surface: surface, DesktopSessionID: "login", LeaseID: opened.Msg.Session.SessionId, RunID: "prefix-run"}
	storedRun, err := reopenedRuns.Get(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, "incomplete", storedRun.Disposition)
	require.EqualValues(t, 1, storedRun.Confirmed)
	require.Len(t, storedRun.Flow.Steps, 3)
	require.Equal(t, prefix.Msg.Digest, storedRun.Digest)
	require.NoError(t, reopenedRuns.Put(ctx, storedRun), "identical persistence is idempotent")
	otherScope := scope
	otherScope.Actor = "other-actor"
	_, err = reopenedRuns.Get(ctx, otherScope)
	require.Error(t, err)
	otherScope = scope
	otherScope.Surface.SurfaceID = "other-desktop"
	_, err = reopenedRuns.Get(ctx, otherScope)
	require.Error(t, err)
	storedRun.Disposition = "passed"
	storedRun.Confirmed = 3
	require.Error(t, reopenedRuns.Put(ctx, storedRun), "terminal uncertainty cannot be rewritten as success")
	scope.RunID = "flow-run"
	storedPassed, err := reopenedRuns.Get(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, "passed", storedPassed.Disposition)
	require.EqualValues(t, 1, storedPassed.Confirmed)
	verifiedRequest := &desktopv1.OwnerRunFlowRequest{Session: opened.Msg.Session, RunId: "verified-run", ApplicationId: "app", ApplicationRevision: "catalog", Flow: &flowsv1.Flow{Transport: "desktop", Steps: []*flowsv1.Step{{Id: "verify", Kind: "desktop-text-assert", Target: "Entry", Arguments: args}}}}
	_, err = client.RunFlow(ctx, connect.NewRequest(verifiedRequest))
	require.NoError(t, err)
	promoteRequest := &desktopv1.OwnerPromoteFlowRequest{Session: opened.Msg.Session, SourceSession: opened.Msg.Session, SourceRunId: "verified-run", ContextKey: "editor:v1"}
	promoted, err := client.PromoteFlow(ctx, connect.NewRequest(promoteRequest))
	require.NoError(t, err)
	promotedAgain, err := client.PromoteFlow(ctx, connect.NewRequest(promoteRequest))
	require.NoError(t, err)
	require.Equal(t, promoted.Msg, promotedAgain.Msg)
	getSaved := &desktopv1.OwnerGetSavedFlowRequest{Session: opened.Msg.Session, Id: promoted.Msg.Id, Version: 1, ContextKey: "editor:v1"}
	_, err = accountClient.GetSavedFlow(ctx, connect.NewRequest(getSaved))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	fetched, err := client.GetSavedFlow(ctx, connect.NewRequest(getSaved))
	require.NoError(t, err)
	require.Equal(t, promoted.Msg, fetched.Msg)
	getSaved.ContextKey = "other-context"
	_, err = client.GetSavedFlow(ctx, connect.NewRequest(getSaved))
	require.Error(t, err)
	getSaved.ContextKey = "editor:v1"
	savedRun := &desktopv1.OwnerRunSavedFlowRequest{Session: opened.Msg.Session, Id: promoted.Msg.Id, Version: 1, ContextKey: "editor:v1", RunId: "saved-replay", ApplicationId: "app", ApplicationRevision: "catalog"}
	replayed, err := client.RunSavedFlow(ctx, connect.NewRequest(savedRun))
	require.NoError(t, err)
	require.Equal(t, "passed", replayed.Msg.Disposition)
	replayDuplicate, err := client.RunSavedFlow(ctx, connect.NewRequest(savedRun))
	require.NoError(t, err)
	require.Equal(t, replayed.Msg, replayDuplicate.Msg)
	verifiedRequest.RunId = savedRun.RunId
	_, err = client.RunFlow(ctx, connect.NewRequest(verifiedRequest))
	require.Error(t, err, "inline request cannot reuse the identity of a saved revision run")
	scope.RunID = savedRun.RunId
	savedEvidence, err := reopenedRuns.Get(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, &internalflows.DesktopFlowRevision{ID: promoted.Msg.Id, Version: 1, ContextKey: "editor:v1"}, savedEvidence.SavedRevision)
	require.EqualValues(t, 5, native.effects.Load(), "assertions must not inject native input")

	gate := &proxyObservationGate{entered: make(chan struct{})}
	native.blockedObservation.Store(gate)
	stoppedFlow := make(chan error, 1)
	stopRequest := &desktopv1.OwnerRunFlowRequest{Session: opened.Msg.Session, RunId: "stop-run", ApplicationId: "app", ApplicationRevision: "catalog", Flow: unknownRequest.Flow}
	go func() {
		_, runErr := client.RunFlow(ctx, connect.NewRequest(stopRequest))
		stoppedFlow <- runErr
	}()
	select {
	case <-gate.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("flow did not enter native observation")
	}
	// Native observation has no release other than context cancellation. Stop
	// must interrupt it, not wait for the normal five-second step deadline.
	stopCtx, stopCancel := context.WithTimeout(ctx, time.Second)
	defer stopCancel()
	_, err = client.Stop(stopCtx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	select {
	case runErr := <-stoppedFlow:
		require.Error(t, runErr, "a revoked run cannot report a successful finish")
	case <-time.After(time.Second):
		t.Fatal("flow did not exit after Stop")
	}
	require.EqualValues(t, 5, native.effects.Load())
	_, err = client.RunFlow(ctx, connect.NewRequest(stopRequest))
	require.Error(t, err, "duplicate under a stopped lease must be refused")
	require.EqualValues(t, 5, native.effects.Load())
	require.Empty(t, svc.ListLiveSessions())
	_, err = client.GetSavedFlow(ctx, connect.NewRequest(getSaved))
	require.Error(t, err, "stopped admission cannot read candidate text")
	_, err = client.Observe(ctx, connect.NewRequest(&desktopv1.OwnerObserveRequest{Session: opened.Msg.Session}))
	require.Error(t, err)

	history, err := client.ReadCleanup(ctx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	require.True(t, history.Msg.Released)
	require.Equal(t, opened.Msg.Session.SessionId, history.Msg.Session.SessionId)
	discovered, err := client.ListAdmissions(ctx, connect.NewRequest(&desktopv1.OwnerListAdmissionsRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, discovered.Msg.Admissions, 1)
	require.True(t, proto.Equal(opened.Msg.Session, discovered.Msg.Admissions[0].Session))
	require.Empty(t, discovered.Msg.NextPageToken)
	// Reconstruct owner and proxy without either active in-memory map.
	recovered, err := NewLocalDesktopAdmission(svc, private, "fake", surface, func(context.Context) (desktophelper.Registration, error) { return registration, nil })
	require.NoError(t, err)
	helperTransport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", filepath.Join(dir, "helper.sock"))
	}}
	defer helperTransport.CloseIdleConnections()
	proxy := &desktopOwnerRPC{owner: recovered, helper: desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: helperTransport, Timeout: 5 * time.Second}, "http://helper")}
	discoveredAgain, err := proxy.ListAdmissions(ctx, connect.NewRequest(&desktopv1.OwnerListAdmissionsRequest{}))
	require.NoError(t, err)
	require.True(t, proto.Equal(discovered.Msg, discoveredAgain.Msg))
	afterRestart, err := proxy.ReadCleanup(ctx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.NoError(t, err)
	require.True(t, afterRestart.Msg.Released)
	require.Empty(t, recovered.grants)
	require.Empty(t, proxy.sessions)
	require.Empty(t, svc.ListLiveSessions())
	recovered.webSubject = "different-account"
	wrongActor := context.WithValue(ctx, desktopWebIdentityKey{}, owneridentity.Identity{Subject: "different-account", Scopes: []string{"device-control:read"}, ExpiresAt: time.Now().Add(time.Minute)})
	_, err = proxy.ReadCleanup(wrongActor, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.Error(t, err)
	otherHistory, err := proxy.ListAdmissions(wrongActor, connect.NewRequest(&desktopv1.OwnerListAdmissionsRequest{}))
	require.NoError(t, err)
	require.Empty(t, otherHistory.Msg.Admissions)
	require.EqualValues(t, 5, native.effects.Load())

	ref, err := targetmodel.SessionRefFromProto(opened.Msg.Session)
	require.NoError(t, err)
	stored, err := recovered.admissions.Get(ctx, ref, recovered.principal.String())
	require.NoError(t, err)
	wrongLease := helperLease(stored)
	wrongLease.Epoch++
	proxy.helper = cleanupReplyFixture{reply: &desktopv1.CleanupResponse{Lease: wrongLease, Released: true, ObservedAt: timestamppb.Now()}}
	_, err = proxy.ReadCleanup(ctx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: opened.Msg.Session}))
	require.Error(t, err, "a released receipt for another epoch cannot confirm this lease")
	// A stale registration cannot open another helper epoch. Failed helper Open
	// must roll back the ordinary device exclusion acquired by the owner.
	_, err = client.Open(ctx, connect.NewRequest(&desktopv1.OwnerOpenRequest{Surface: surface.Proto(), TtlSeconds: 60, Control: true}))
	require.Error(t, err)
	require.Empty(t, svc.ListLiveSessions())
	unknownAdmission, err := client.ListAdmissions(ctx, connect.NewRequest(&desktopv1.OwnerListAdmissionsRequest{}))
	require.NoError(t, err)
	require.Len(t, unknownAdmission.Msg.Admissions, 2, "lost or rejected helper admission must remain discoverable for reconciliation")
	for _, admission := range unknownAdmission.Msg.Admissions {
		if admission.Session.SessionId == opened.Msg.Session.SessionId {
			continue
		}
		absent, err := client.ReadCleanup(ctx, connect.NewRequest(&desktopv1.OwnerStopRequest{Session: admission.Session}))
		require.NoError(t, err)
		require.True(t, absent.Msg.Released, "revoked never-admitted lease should reconcile without native cleanup")
		require.True(t, proto.Equal(admission.Session, absent.Msg.Session))
	}

	require.EqualValues(t, 5, native.effects.Load(), "discovery must not invoke native input")
}

func (n *proxyNative) Applications(_ context.Context, leaseID string) (sessions.DesktopApplications, error) {
	if leaseID == "" {
		return sessions.DesktopApplications{}, sessions.ErrDesktopAdmission
	}
	return sessions.DesktopApplications{Revision: "catalog", ExpiresAt: time.Now().Add(20 * time.Second), Applications: []sessions.DesktopApplication{{ID: "app", Name: "Editor", ProcessID: 42}}}, nil
}

func (n *proxyNative) ObserveApplication(ctx context.Context, id, revision, leaseID string) (sessions.DesktopObservation, error) {
	if id != "app" || revision != "catalog" || leaseID == "" {
		return sessions.DesktopObservation{}, sessions.ErrDesktopAdmission
	}
	if gate := n.blockedObservation.Load(); gate != nil {
		close(gate.entered)
		<-ctx.Done()
		return sessions.DesktopObservation{}, ctx.Err()
	}
	out, err := n.Observe(ctx)
	out.Semantic = &sessions.DesktopSemanticObservation{Revision: "fields", ExpiresAt: time.Now().Add(time.Second), ProcessID: 42, Elements: []sessions.DesktopSemanticElement{{ID: "window", WindowID: "window", Name: "Editor"}, {ID: "field", WindowID: "window", ParentID: "window", Name: "Entry", Editable: true}}}
	return out, err
}

func (n *proxyNative) Resolve(_ context.Context, selector sessions.DesktopSelector, lease string) (sessions.DesktopResolution, error) {
	if lease == "" || selector.Revision != "fields" || selector.WindowID != "window" || selector.Name != "Entry" || !selector.EditableOnly {
		return sessions.DesktopResolution{}, sessions.ErrDesktopAdmission
	}
	return sessions.DesktopResolution{Disposition: "unique", Revision: "fields", GeometryRevision: "geometry", ExpiresAt: time.Now().Add(500 * time.Millisecond), ElementIDs: []string{"field"}}, nil
}

type cleanupReplyFixture struct {
	desktopv1connect.DesktopHelperServiceClient
	reply *desktopv1.CleanupResponse
}

func (f cleanupReplyFixture) ReadCleanup(context.Context, *connect.Request[desktopv1.StopRequest]) (*connect.Response[desktopv1.CleanupResponse], error) {
	return connect.NewResponse(f.reply), nil
}

func TestActivationReplyValidation(t *testing.T) {
	now := time.Now()
	valid := &desktopv1.ActivationReference{ContextId: "daebc72f-7922-46ee-bb97-f3d51db11b94", DisplayId: "display", GeometryRevision: "geometry", CapturedAt: timestamppb.New(now.Add(-time.Second)), ExpiresAt: timestamppb.New(now.Add(20 * time.Second))}
	require.True(t, validActivationReference(valid, now.Add(time.Minute), now))
	for _, change := range []func(*desktopv1.ActivationReference){
		func(v *desktopv1.ActivationReference) { v.ContextId = "00000000-0000-0000-0000-000000000000" },
		func(v *desktopv1.ActivationReference) { v.CapturedAt = timestamppb.New(now.Add(time.Second)) },
		func(v *desktopv1.ActivationReference) { v.ExpiresAt = timestamppb.New(now) },
		func(v *desktopv1.ActivationReference) { v.ExpiresAt = timestamppb.New(now.Add(time.Minute)) },
		func(v *desktopv1.ActivationReference) { v.DisplayId = "" },
	} {
		bad := proto.Clone(valid).(*desktopv1.ActivationReference)
		change(bad)
		require.False(t, validActivationReference(bad, now.Add(time.Minute), now))
	}
	require.False(t, validActivationReference(valid, now.Add(time.Second), now))
	require.False(t, validActivationReference(nil, now, now))
}
