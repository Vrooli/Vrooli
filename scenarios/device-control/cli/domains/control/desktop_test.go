package control

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	connectrpc "connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

func TestDesktopCLIAccountAuthorityNeverFallsBack(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "owner.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	var calls atomic.Int32
	_, account := desktopv1connect.NewDesktopAccountServiceHandler(desktopCLIAccount{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != desktopv1connect.DesktopAccountServiceGetSavedFlowProcedure {
			t.Error("account operation used local authority namespace")
			http.Error(w, "wrong namespace", http.StatusForbidden)
			return
		}
		account.ServeHTTP(w, r)
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Close(); <-done })
	request := filepath.Join(dir, "request.json")
	require.NoError(t, os.WriteFile(request, []byte(`{"id":"saved","version":2}`), 0o600))
	token := filepath.Join(dir, "access-token")
	require.NoError(t, os.WriteFile(token, []byte("fixture-token\n"), 0o600))
	result, err := desktopCallAuthorized(context.Background(), socket, "get-saved-flow", request, "", token)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.(*desktopv1.SavedDesktopFlow).Version)
	require.NoError(t, os.WriteFile(token, []byte("invalid-token"), 0o600))
	_, err = desktopCallAuthorized(context.Background(), socket, "get-saved-flow", request, "", token)
	require.Equal(t, connectrpc.CodePermissionDenied, connectrpc.CodeOf(err))
	require.NotContains(t, err.Error(), "private-fixture-token")
	require.EqualValues(t, 2, calls.Load(), "denial must not retry through local authority")
	require.NoError(t, os.Chmod(token, 0o644))
	_, err = desktopCallAuthorized(context.Background(), socket, "get-saved-flow", request, "", token)
	require.ErrorContains(t, err, "private regular file")
	require.NoError(t, os.Chmod(token, 0o600))
	link := filepath.Join(dir, "linked-token")
	require.NoError(t, os.Symlink(token, link))
	_, err = desktopCallAuthorized(context.Background(), socket, "get-saved-flow", request, "", link)
	require.Error(t, err)
	require.EqualValues(t, 2, calls.Load(), "invalid token files must fail before transport")
}

type desktopCLIObserver struct {
	desktopv1connect.UnimplementedDesktopOwnerServiceHandler
}

func (desktopCLIObserver) Observe(context.Context, *connectrpc.Request[desktopv1.OwnerObserveRequest]) (*connectrpc.Response[desktopv1.ObserveResponse], error) {
	var data bytes.Buffer
	_ = png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 4, 3)))
	return connectrpc.NewResponse(&desktopv1.ObserveResponse{Png: data.Bytes(), Width: 4, Height: 3, DisplayId: "display", GeometryRevision: "geometry"}), nil
}

func TestDesktopCLIUsesUnixOwnerAndPrivateExplicitImageOutput(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "owner.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	_, handler := desktopv1connect.NewDesktopOwnerServiceHandler(desktopCLIObserver{})
	server := &http.Server{Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Close(); <-done })
	request := filepath.Join(dir, "request.json")
	require.NoError(t, os.WriteFile(request, []byte(`{}`), 0o600))
	output := filepath.Join(dir, "capture.png")
	result, err := desktopCall(context.Background(), socket, "observe", request, output)
	require.NoError(t, err)
	observation := result.(*desktopv1.ObserveResponse)
	require.Empty(t, observation.Png, "pixels must not be emitted into CLI JSON/logs")
	require.EqualValues(t, 4, observation.Width)
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	before, err := os.ReadFile(output)
	require.NoError(t, err)
	_, err = desktopCall(context.Background(), socket, "observe", request, output)
	require.Error(t, err)
	after, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, before, after, "existing output must not be overwritten")
	require.NoError(t, os.WriteFile(request, []byte(`{"helper_token":"forged"}`), 0o600))
	_, err = desktopCall(context.Background(), socket, "open", request, "")
	require.ErrorContains(t, err, "unknown field")
	group := DesktopGroup()
	require.False(t, group.NeedsAPI)
	for _, command := range group.Subcommands {
		require.False(t, command.NeedsAPI, "local owner must not require public API discovery")
	}
}

func (desktopCLIObserver) Applications(_ context.Context, r *connectrpc.Request[desktopv1.OwnerApplicationsRequest]) (*connectrpc.Response[desktopv1.ApplicationsResponse], error) {
	if r.Msg.Session.GetSessionId() != "lease" {
		return nil, fmt.Errorf("wrong session")
	}
	return connectrpc.NewResponse(&desktopv1.ApplicationsResponse{Revision: "catalog", Applications: []*desktopv1.Application{{ApplicationId: "app", Name: "Editor", ProcessId: 42}}}), nil
}

func (desktopCLIObserver) Resolve(_ context.Context, r *connectrpc.Request[desktopv1.OwnerResolveRequest]) (*connectrpc.Response[desktopv1.ResolveResponse], error) {
	selector := r.Msg.Selector
	if r.Msg.Session.GetSessionId() != "lease" || selector.GetObservationRevision() != "observed" || selector.GetWindowId() != "window" || selector.GetName() != "Entry" || !selector.GetEditableOnly() {
		return nil, fmt.Errorf("changed selector")
	}
	return connectrpc.NewResponse(&desktopv1.ResolveResponse{Disposition: desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS, ObservationRevision: "observed", ElementIds: []string{"first", "second"}}), nil
}

func TestDesktopCLIDiscoveryAndResolutionPreserveReferencesAndAmbiguity(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "owner.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	_, handler := desktopv1connect.NewDesktopOwnerServiceHandler(desktopCLIObserver{})
	server := &http.Server{Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Close(); <-done })
	request := filepath.Join(dir, "request.json")
	require.NoError(t, os.WriteFile(request, []byte(`{"session":{"session_id":"lease"}}`), 0o600))
	catalog, err := desktopCall(context.Background(), socket, "applications", request, "")
	require.NoError(t, err)
	require.Equal(t, "app", catalog.(*desktopv1.ApplicationsResponse).Applications[0].ApplicationId)
	require.NoError(t, os.WriteFile(request, []byte(`{"session":{"session_id":"lease"},"selector":{"observation_revision":"observed","window_id":"window","name":"Entry","editable_only":true}}`), 0o600))
	result, err := desktopCall(context.Background(), socket, "resolve", request, "")
	require.NoError(t, err)
	resolved := result.(*desktopv1.ResolveResponse)
	require.Equal(t, desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS, resolved.Disposition)
	require.Equal(t, []string{"first", "second"}, resolved.ElementIds)
}

func (desktopCLIObserver) PromoteFlow(_ context.Context, r *connectrpc.Request[desktopv1.OwnerPromoteFlowRequest]) (*connectrpc.Response[desktopv1.SavedDesktopFlow], error) {
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.Session.GetSessionId() != "lease" || r.Msg.SourceRunId != "source" || r.Msg.ExpectedVersion != 1 {
		return nil, fmt.Errorf("saved revision binding changed")
	}
	return connectrpc.NewResponse(&desktopv1.SavedDesktopFlow{Id: r.Msg.Id, Version: 2, ContextKey: r.Msg.ContextKey}), nil
}

func (desktopCLIObserver) GetSavedFlow(_ context.Context, r *connectrpc.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connectrpc.Response[desktopv1.SavedDesktopFlow], error) {
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.Session.GetSessionId() != "lease" || r.Msg.Version != 2 {
		return nil, fmt.Errorf("saved revision binding changed")
	}
	return connectrpc.NewResponse(&desktopv1.SavedDesktopFlow{Id: r.Msg.Id, Version: 2, ContextKey: r.Msg.ContextKey}), nil
}

func (desktopCLIObserver) RunSavedFlow(_ context.Context, r *connectrpc.Request[desktopv1.OwnerRunSavedFlowRequest]) (*connectrpc.Response[desktopv1.FlowRecord], error) {
	if r.Msg.Id != "saved" || r.Msg.ContextKey != "editor:v1" || r.Msg.Session.GetSessionId() != "lease" || r.Msg.Version != 2 {
		return nil, fmt.Errorf("saved revision binding changed")
	}
	return connectrpc.NewResponse(&desktopv1.FlowRecord{RunId: r.Msg.RunId, Disposition: "claimed"}), nil
}

func TestDesktopCLISavedRevisionRequests(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "owner.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	_, handler := desktopv1connect.NewDesktopOwnerServiceHandler(desktopCLIObserver{})
	server := &http.Server{Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Close(); <-done })
	for _, operation := range []string{"promote-flow", "get-saved-flow", "run-saved-flow"} {
		t.Run(operation, func(t *testing.T) {
			body := `{"session":{"session_id":"lease"},"id":"saved","context_key":"editor:v1",`
			switch operation {
			case "promote-flow":
				body += `"source_run_id":"source","expected_version":1}`
			case "get-saved-flow":
				body += `"version":2}`
			case "run-saved-flow":
				body += `"version":2,"run_id":"exact-run"}`
			}
			request := filepath.Join(dir, "request.json")
			require.NoError(t, os.WriteFile(request, []byte(body), 0o600))
			result, err := desktopCall(context.Background(), socket, operation, request, "")
			require.NoError(t, err)
			if run, ok := result.(*desktopv1.FlowRecord); ok {
				require.Equal(t, "exact-run", run.RunId)
				require.Equal(t, "claimed", run.Disposition)
			} else {
				require.EqualValues(t, 2, result.(*desktopv1.SavedDesktopFlow).Version)
			}
		})
	}
}

type desktopCLIAccount struct {
	desktopv1connect.UnimplementedDesktopAccountServiceHandler
}

func (desktopCLIAccount) GetSavedFlow(_ context.Context, r *connectrpc.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connectrpc.Response[desktopv1.SavedDesktopFlow], error) {
	if r.Header().Get("Authorization") != "Bearer fixture-token" {
		return nil, connectrpc.NewError(connectrpc.CodePermissionDenied, fmt.Errorf("private-fixture-token"))
	}
	return connectrpc.NewResponse(&desktopv1.SavedDesktopFlow{Id: r.Msg.Id, Version: r.Msg.Version}), nil
}

type cleanupCLIHandler struct {
	desktopv1connect.UnimplementedDesktopOwnerServiceHandler
}

func (cleanupCLIHandler) ReadCleanup(_ context.Context, r *connectrpc.Request[desktopv1.OwnerStopRequest]) (*connectrpc.Response[desktopv1.OwnerCleanupResponse], error) {
	if r.Msg.Session.SessionId == "missing" {
		return nil, connectrpc.NewError(connectrpc.CodeNotFound, fmt.Errorf("no cleanup evidence"))
	}
	return connectrpc.NewResponse(&desktopv1.OwnerCleanupResponse{Session: r.Msg.Session, Released: false}), nil
}

func TestDesktopCLIReadCleanupPreservesUnknownAndSelectedAuthority(t *testing.T) {
	for _, account := range []bool{false, true} {
		t.Run(fmt.Sprint(account), func(t *testing.T) {
			dir := t.TempDir()
			socket := filepath.Join(dir, "owner.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			path, handler := desktopv1connect.NewDesktopOwnerServiceHandler(cleanupCLIHandler{})
			if account {
				path, handler = desktopv1connect.NewDesktopAccountServiceHandler(cleanupCLIHandler{})
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, path+"ReadCleanup", r.URL.Path)
				if account {
					require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				} else {
					require.Empty(t, r.Header.Get("Authorization"))
				}
				handler.ServeHTTP(w, r)
			})}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() { server.Close(); <-done })
			request := filepath.Join(dir, "request.json")
			require.NoError(t, os.WriteFile(request, []byte(`{"session":{"sessionId":"exact-session"}}`), 0o600))
			token := ""
			if account {
				token = filepath.Join(dir, "token")
				require.NoError(t, os.WriteFile(token, []byte("fixture-token"), 0o600))
			}
			result, err := desktopCallAuthorized(context.Background(), socket, "read-cleanup", request, "", token)
			require.NoError(t, err)
			receipt := result.(*desktopv1.OwnerCleanupResponse)
			require.False(t, receipt.Released)
			require.Equal(t, "exact-session", receipt.Session.SessionId)
			require.NoError(t, os.WriteFile(request, []byte(`{"session":{"sessionId":"missing"}}`), 0o600))
			_, err = desktopCallAuthorized(context.Background(), socket, "read-cleanup", request, "", token)
			require.Equal(t, connectrpc.CodeNotFound, connectrpc.CodeOf(err))
			require.EqualValues(t, 2, calls.Load())
		})
	}
}

func (cleanupCLIHandler) ListAdmissions(_ context.Context, r *connectrpc.Request[desktopv1.OwnerListAdmissionsRequest]) (*connectrpc.Response[desktopv1.OwnerListAdmissionsResponse], error) {
	if r.Msg.PageToken == "missing" {
		return nil, connectrpc.NewError(connectrpc.CodeInvalidArgument, fmt.Errorf("invalid cursor"))
	}
	return connectrpc.NewResponse(&desktopv1.OwnerListAdmissionsResponse{NextPageToken: r.Msg.PageToken}), nil
}

func TestDesktopCLIListAdmissionsPreservesCursorAndSelectedAuthority(t *testing.T) {
	for _, account := range []bool{false, true} {
		t.Run(fmt.Sprint(account), func(t *testing.T) {
			dir := t.TempDir()
			socket := filepath.Join(dir, "owner.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			path, handler := desktopv1connect.NewDesktopOwnerServiceHandler(cleanupCLIHandler{})
			if account {
				path, handler = desktopv1connect.NewDesktopAccountServiceHandler(cleanupCLIHandler{})
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, path+"ListAdmissions", r.URL.Path)
				if account {
					require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				} else {
					require.Empty(t, r.Header.Get("Authorization"))
				}
				handler.ServeHTTP(w, r)
			})}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() { server.Close(); <-done })
			request := filepath.Join(dir, "request.json")
			require.NoError(t, os.WriteFile(request, []byte(`{"page_token":"12","page_size":1}`), 0o600))
			token := ""
			if account {
				token = filepath.Join(dir, "token")
				require.NoError(t, os.WriteFile(token, []byte("fixture-token"), 0o600))
			}
			result, err := desktopCallAuthorized(context.Background(), socket, "list-admissions", request, "", token)
			require.NoError(t, err)
			page := result.(*desktopv1.OwnerListAdmissionsResponse)
			require.Equal(t, "12", page.NextPageToken)
			require.NoError(t, os.WriteFile(request, []byte(`{"page_token":"missing"}`), 0o600))
			_, err = desktopCallAuthorized(context.Background(), socket, "list-admissions", request, "", token)
			require.Equal(t, connectrpc.CodeInvalidArgument, connectrpc.CodeOf(err))
			require.EqualValues(t, 2, calls.Load())
		})
	}
}

func (cleanupCLIHandler) ReconcileOpen(_ context.Context, r *connectrpc.Request[desktopv1.OwnerOpenRequest]) (*connectrpc.Response[desktopv1.OwnerOpenDisposition], error) {
	if r.Msg.RequestId == "missing" {
		return nil, connectrpc.NewError(connectrpc.CodeInvalidArgument, fmt.Errorf("invalid request identity"))
	}
	return connectrpc.NewResponse(&desktopv1.OwnerOpenDisposition{RequestId: r.Msg.RequestId, State: "not_admitted"}), nil
}
func TestDesktopCLIReconcileOpenPreservesRequestAndSelectedAuthority(t *testing.T) {
	for _, account := range []bool{false, true} {
		t.Run(fmt.Sprint(account), func(t *testing.T) {
			dir := t.TempDir()
			socket := filepath.Join(dir, "owner.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			path, handler := desktopv1connect.NewDesktopOwnerServiceHandler(cleanupCLIHandler{})
			if account {
				path, handler = desktopv1connect.NewDesktopAccountServiceHandler(cleanupCLIHandler{})
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, path+"ReconcileOpen", r.URL.Path)
				if account {
					require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				} else {
					require.Empty(t, r.Header.Get("Authorization"))
				}
				handler.ServeHTTP(w, r)
			})}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() { server.Close(); <-done })
			request := filepath.Join(dir, "request.json")
			require.NoError(t, os.WriteFile(request, []byte(`{"request_id":"exact-request"}`), 0o600))
			token := ""
			if account {
				token = filepath.Join(dir, "token")
				require.NoError(t, os.WriteFile(token, []byte("fixture-token"), 0o600))
			}
			result, err := desktopCallAuthorized(context.Background(), socket, "reconcile-open", request, "", token)
			require.NoError(t, err)
			page := result.(*desktopv1.OwnerOpenDisposition)
			require.Equal(t, "exact-request", page.RequestId)
			require.Equal(t, "not_admitted", page.State)
			require.NoError(t, os.WriteFile(request, []byte(`{"request_id":"missing"}`), 0o600))
			_, err = desktopCallAuthorized(context.Background(), socket, "reconcile-open", request, "", token)
			require.Equal(t, connectrpc.CodeInvalidArgument, connectrpc.CodeOf(err))
			require.EqualValues(t, 2, calls.Load())
		})
	}
}

func (cleanupCLIHandler) CaptureActivation(_ context.Context, r *connectrpc.Request[desktopv1.OwnerStopRequest]) (*connectrpc.Response[desktopv1.ActivationReference], error) {
	if r.Msg.Session.GetSessionId() != "exact-session" {
		return nil, connectrpc.NewError(connectrpc.CodePermissionDenied, fmt.Errorf("wrong session"))
	}
	return connectrpc.NewResponse(&desktopv1.ActivationReference{ContextId: "fixture-context", PointerX: -20}), nil
}
func (cleanupCLIHandler) ReadActivation(_ context.Context, r *connectrpc.Request[desktopv1.OwnerReadActivationRequest]) (*connectrpc.Response[desktopv1.ActivationReference], error) {
	if r.Msg.ContextId != "fixture-context" {
		return nil, connectrpc.NewError(connectrpc.CodeNotFound, fmt.Errorf("missing context"))
	}
	return connectrpc.NewResponse(&desktopv1.ActivationReference{ContextId: r.Msg.ContextId, PointerX: -20}), nil
}
func TestDesktopCLIActivationUsesExplicitAuthorityWithoutReplay(t *testing.T) {
	for _, account := range []bool{false, true} {
		t.Run(fmt.Sprint(account), func(t *testing.T) {
			dir := t.TempDir()
			socket := filepath.Join(dir, "o.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			path, handler := desktopv1connect.NewDesktopOwnerServiceHandler(cleanupCLIHandler{})
			if account {
				path, handler = desktopv1connect.NewDesktopAccountServiceHandler(cleanupCLIHandler{})
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.True(t, r.URL.Path == path+"CaptureActivation" || r.URL.Path == path+"ReadActivation")
				if account {
					require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				} else {
					require.Empty(t, r.Header.Get("Authorization"))
				}
				handler.ServeHTTP(w, r)
			})}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() { server.Close(); <-done })
			token := ""
			if account {
				token = filepath.Join(dir, "token")
				require.NoError(t, os.WriteFile(token, []byte("fixture-token"), 0600))
			}
			request := filepath.Join(dir, "request.json")
			require.NoError(t, os.WriteFile(request, []byte(`{"session":{"sessionId":"exact-session"}}`), 0600))
			captured, err := desktopCallAuthorized(context.Background(), socket, "capture-activation", request, "", token)
			require.NoError(t, err)
			require.EqualValues(t, -20, captured.(*desktopv1.ActivationReference).PointerX)
			require.NoError(t, os.WriteFile(request, []byte(`{"session":{"sessionId":"exact-session"},"contextId":"fixture-context"}`), 0600))
			read, err := desktopCallAuthorized(context.Background(), socket, "read-activation", request, "", token)
			require.NoError(t, err)
			require.Equal(t, "fixture-context", read.(*desktopv1.ActivationReference).ContextId)
			require.NoError(t, os.WriteFile(request, []byte(`{"contextId":"missing"}`), 0600))
			_, err = desktopCallAuthorized(context.Background(), socket, "read-activation", request, "", token)
			require.Equal(t, connectrpc.CodeNotFound, connectrpc.CodeOf(err))
			require.EqualValues(t, 3, calls.Load())
		})
	}
}
