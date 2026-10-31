package surfaces

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
)

type accountFixture struct {
	desktopv1connect.UnimplementedDesktopAccountServiceHandler
	calls atomic.Int32
}

func (f *accountFixture) Open(_ context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenResponse], error) {
	f.calls.Add(1)
	if r.Header().Get("Authorization") != "Bearer accepted" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("private-owner-detail"))
	}
	if r.Header().Get("Cookie") != "" || r.Header().Get("X-Actor") != "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected metadata"))
	}
	return connect.NewResponse(&desktopv1.OwnerOpenResponse{}), nil
}

func (f *accountFixture) ReadCleanup(_ context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.OwnerCleanupResponse], error) {
	if r.Header().Get("Authorization") != "Bearer accepted" || r.Header().Get("Cookie") != "" || r.Header().Get("X-Actor") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("rejected metadata"))
	}
	return connect.NewResponse(&desktopv1.OwnerCleanupResponse{Session: r.Msg.Session, Released: false}), nil
}

func (f *accountFixture) ListAdmissions(_ context.Context, r *connect.Request[desktopv1.OwnerListAdmissionsRequest]) (*connect.Response[desktopv1.OwnerListAdmissionsResponse], error) {
	if r.Header().Get("Authorization") != "Bearer accepted" || r.Header().Get("Cookie") != "" || r.Header().Get("X-Actor") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("rejected metadata"))
	}
	return connect.NewResponse(&desktopv1.OwnerListAdmissionsResponse{NextPageToken: r.Msg.PageToken}), nil
}

func TestPortalDesktopRequiresBearerAndUsesAccountNamespace(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "account", true: "legacy-only"}[legacy], func(t *testing.T) {
			fixture := &accountFixture{}
			path, handler := desktopv1connect.NewDesktopAccountServiceHandler(fixture)
			if legacy {
				path = "/vrooli.device_control.v1.desktop.DesktopOwnerService/"
			}
			ownerMux := http.NewServeMux()
			ownerMux.Handle(path, handler)
			socket := filepath.Join(t.TempDir(), "owner.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			ownerServer := &http.Server{Handler: ownerMux}
			done := make(chan error, 1)
			go func() { done <- ownerServer.Serve(listener) }()
			t.Cleanup(func() { ownerServer.Close(); <-done })
			router := mux.NewRouter()
			DesktopModule(socket).Mount(router)
			portal := httptest.NewServer(router)
			t.Cleanup(portal.Close)
			client := surfacesv1connect.NewDesktopSessionServiceClient(portal.Client(), portal.URL)
			_, err = client.Open(context.Background(), connect.NewRequest(&desktopv1.OwnerOpenRequest{}))
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			require.Zero(t, fixture.calls.Load())
			request := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
			request.Header().Set("Authorization", "Bearer accepted")
			request.Header().Set("Cookie", "untrusted-cookie")
			request.Header().Set("X-Actor", "forged")
			result, err := client.Open(context.Background(), request)
			if legacy {
				require.Error(t, err)
				require.Zero(t, fixture.calls.Load(), "legacy local service must never receive account requests")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "no-store", result.Header().Get("Cache-Control"))
			require.EqualValues(t, 1, fixture.calls.Load())

			cleanupRequest := connect.NewRequest(&desktopv1.OwnerStopRequest{})
			_, err = client.ReadCleanup(context.Background(), cleanupRequest)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			cleanupRequest.Header().Set("Authorization", "Bearer accepted")
			cleanupRequest.Header().Set("Cookie", "untrusted")
			cleanupRequest.Header().Set("X-Actor", "forged")
			cleanupReply, err := client.ReadCleanup(context.Background(), cleanupRequest)
			require.NoError(t, err)
			require.False(t, cleanupReply.Msg.Released)
			require.Equal(t, "no-store", cleanupReply.Header().Get("Cache-Control"))
			listRequest := connect.NewRequest(&desktopv1.OwnerListAdmissionsRequest{PageToken: "12", PageSize: 2})
			_, err = client.ListAdmissions(context.Background(), listRequest)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			listRequest.Header().Set("Authorization", "Bearer accepted")
			listRequest.Header().Set("Cookie", "untrusted")
			listRequest.Header().Set("X-Actor", "forged")
			listReply, err := client.ListAdmissions(context.Background(), listRequest)
			require.NoError(t, err)
			require.Equal(t, "12", listReply.Msg.NextPageToken)
			require.Equal(t, "no-store", listReply.Header().Get("Cache-Control"))
			reconcileRequest := connect.NewRequest(&desktopv1.OwnerOpenRequest{RequestId: "exact-request"})
			_, err = client.ReconcileOpen(context.Background(), reconcileRequest)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			reconcileRequest.Header().Set("Authorization", "Bearer accepted")
			reconcileRequest.Header().Set("Cookie", "untrusted")
			reconcileRequest.Header().Set("X-Actor", "forged")
			reconciled, err := client.ReconcileOpen(context.Background(), reconcileRequest)
			require.NoError(t, err)
			require.Equal(t, "exact-request", reconciled.Msg.RequestId)
			require.Equal(t, "not_admitted", reconciled.Msg.State)
			require.Equal(t, "no-store", reconciled.Header().Get("Cache-Control"))

			appRequest := connect.NewRequest(&desktopv1.OwnerApplicationsRequest{})
			_, err = client.Applications(context.Background(), appRequest)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			appRequest.Header().Set("Authorization", "Bearer accepted")
			appRequest.Header().Set("Cookie", "untrusted")
			catalog, err := client.Applications(context.Background(), appRequest)
			require.NoError(t, err)
			require.Equal(t, "no-store", catalog.Header().Get("Cache-Control"))
			require.Equal(t, "app", catalog.Msg.Applications[0].ApplicationId)
			flowRequest := connect.NewRequest(&desktopv1.OwnerRunFlowRequest{RunId: "exact-flow"})
			_, err = client.RunFlow(context.Background(), flowRequest)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			flowRequest.Header().Set("Authorization", "Bearer accepted")
			flowRequest.Header().Set("Cookie", "untrusted")
			run, err := client.RunFlow(context.Background(), flowRequest)
			require.NoError(t, err)
			require.Equal(t, "exact-flow", run.Msg.RunId)
			require.Equal(t, "claimed", run.Msg.Disposition)
			require.Equal(t, "no-store", run.Header().Get("Cache-Control"))

			{
				savedRequest := connect.NewRequest(&desktopv1.OwnerPromoteFlowRequest{SourceRunId: "source", ContextKey: "editor:v1", Id: "saved", ExpectedVersion: 1})
				_, err := client.PromoteFlow(context.Background(), savedRequest)
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				savedRequest.Header().Set("Authorization", "Bearer accepted")
				savedRequest.Header().Set("Cookie", "untrusted")
				savedResult, err := client.PromoteFlow(context.Background(), savedRequest)
				require.NoError(t, err)
				require.Equal(t, "no-store", savedResult.Header().Get("Cache-Control"))
				require.EqualValues(t, 2, savedResult.Msg.Version)
			}

			{
				savedRequest := connect.NewRequest(&desktopv1.OwnerGetSavedFlowRequest{Id: "saved", Version: 2, ContextKey: "editor:v1"})
				_, err := client.GetSavedFlow(context.Background(), savedRequest)
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				savedRequest.Header().Set("Authorization", "Bearer accepted")
				savedRequest.Header().Set("Cookie", "untrusted")
				savedResult, err := client.GetSavedFlow(context.Background(), savedRequest)
				require.NoError(t, err)
				require.Equal(t, "no-store", savedResult.Header().Get("Cache-Control"))
				require.EqualValues(t, 2, savedResult.Msg.Version)
			}

			{
				savedRequest := connect.NewRequest(&desktopv1.OwnerRunSavedFlowRequest{Id: "saved", Version: 2, ContextKey: "editor:v1", RunId: "saved-run"})
				_, err := client.RunSavedFlow(context.Background(), savedRequest)
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				savedRequest.Header().Set("Authorization", "Bearer accepted")
				savedRequest.Header().Set("Cookie", "untrusted")
				savedResult, err := client.RunSavedFlow(context.Background(), savedRequest)
				require.NoError(t, err)
				require.Equal(t, "no-store", savedResult.Header().Get("Cache-Control"))
				require.Equal(t, "saved-run", savedResult.Msg.RunId)
			}

			request.Header().Set("Authorization", "Bearer denied")
			_, err = client.Open(context.Background(), request)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.False(t, strings.Contains(err.Error(), "private-owner-detail"))
		})
	}
}

func (f *accountFixture) Applications(ctx context.Context, r *connect.Request[desktopv1.OwnerApplicationsRequest]) (*connect.Response[desktopv1.ApplicationsResponse], error) {
	request := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
	for key, values := range r.Header() {
		request.Header()[key] = values
	}
	if _, err := f.Open(ctx, request); err != nil {
		return nil, err
	}
	return connect.NewResponse(&desktopv1.ApplicationsResponse{Revision: "catalog", Applications: []*desktopv1.Application{{ApplicationId: "app", Name: "Editor", ProcessId: 42}}}), nil
}

func (f *accountFixture) RunFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	check := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
	for k, v := range r.Header() {
		check.Header()[k] = v
	}
	if _, err := f.Open(ctx, check); err != nil {
		return nil, err
	}
	return connect.NewResponse(&desktopv1.FlowRecord{RunId: r.Msg.RunId, Disposition: "claimed"}), nil
}

func TestFlowForwardingHasBoundedLongerDeadlineAndNoRetry(t *testing.T) {
	r := connect.NewRequest(&desktopv1.OwnerRunFlowRequest{RunId: "exact-run"})
	r.Header().Set("Authorization", "Bearer accepted")
	r.Header().Set("Cookie", "private")
	calls := 0
	_, err := desktopForwardWithin(context.Background(), r, func(ctx context.Context, forward *connect.Request[desktopv1.OwnerRunFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
		calls++
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.Greater(t, time.Until(deadline), 30*time.Second)
		require.LessOrEqual(t, time.Until(deadline), 35*time.Second)
		require.Equal(t, "exact-run", forward.Msg.RunId)
		require.Equal(t, "Bearer accepted", forward.Header().Get("Authorization"))
		require.Empty(t, forward.Header().Get("Cookie"))
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("private backend detail"))
	}, 35*time.Second)
	require.Error(t, err)
	require.Equal(t, 1, calls)
	require.NotContains(t, err.Error(), "private backend detail")
}

func (f *accountFixture) PromoteFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerPromoteFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	check := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
	for k, v := range r.Header() {
		check.Header()[k] = v
	}
	if _, err := f.Open(ctx, check); err != nil {
		return nil, err
	}
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.SourceRunId != "source" || r.Msg.ExpectedVersion != 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&desktopv1.SavedDesktopFlow{Id: r.Msg.Id, Version: 2, ContextKey: r.Msg.ContextKey}), nil
}

func (f *accountFixture) GetSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	check := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
	for k, v := range r.Header() {
		check.Header()[k] = v
	}
	if _, err := f.Open(ctx, check); err != nil {
		return nil, err
	}
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.Version != 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&desktopv1.SavedDesktopFlow{Id: r.Msg.Id, Version: 2, ContextKey: r.Msg.ContextKey}), nil
}

func (f *accountFixture) RunSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunSavedFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	check := connect.NewRequest(&desktopv1.OwnerOpenRequest{})
	for k, v := range r.Header() {
		check.Header()[k] = v
	}
	if _, err := f.Open(ctx, check); err != nil {
		return nil, err
	}
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.Version != 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&desktopv1.FlowRecord{RunId: r.Msg.RunId, Disposition: "claimed"}), nil
}

func (f *accountFixture) ReconcileOpen(_ context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenDisposition], error) {
	if r.Header().Get("Authorization") != "Bearer accepted" || r.Header().Get("Cookie") != "" || r.Header().Get("X-Actor") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("rejected metadata"))
	}
	return connect.NewResponse(&desktopv1.OwnerOpenDisposition{RequestId: r.Msg.RequestId, State: "not_admitted"}), nil
}
