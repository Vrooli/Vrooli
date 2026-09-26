package nodereach

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
)

// registryRecorder records the order of the calls Forget makes and can be told
// to answer Revoke the way the real registry answers an already-revoked node.
type registryRecorder struct {
	registryconnect.UnimplementedNodeRegistryServiceHandler
	calls          []string
	revokeResponse error
	removeErr      error
}

func (r *registryRecorder) RevokeNode(_ context.Context, _ *connect.Request[registryv1.RevokeNodeRequest]) (*connect.Response[registryv1.RevokeNodeResponse], error) {
	r.calls = append(r.calls, "revoke")
	if r.revokeResponse != nil {
		return nil, r.revokeResponse
	}
	return connect.NewResponse(&registryv1.RevokeNodeResponse{}), nil
}

func (r *registryRecorder) RemoveNode(_ context.Context, req *connect.Request[registryv1.RemoveNodeRequest]) (*connect.Response[registryv1.RemoveNodeResponse], error) {
	r.calls = append(r.calls, "remove")
	if r.removeErr != nil {
		return nil, r.removeErr
	}
	return connect.NewResponse(&registryv1.RemoveNodeResponse{RemovedNodeId: req.Msg.GetId()}), nil
}

func registryServer(t *testing.T, recorder *registryRecorder) *httptest.Server {
	t.Helper()
	path, handler := registryconnect.NewNodeRegistryServiceHandler(recorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, path) {
			handler.ServeHTTP(w, req)
			return
		}
		http.NotFound(w, req)
	}))
	t.Cleanup(server.Close)
	return server
}

// The registry refuses to remove a node that is still authorized. Forgetting is
// one operator intent, so the client performs both halves in the only order
// that can succeed.
func TestForgetRevokesBeforeRemoving(t *testing.T) {
	recorder := &registryRecorder{}
	client := New(Config{BridgeURL: registryServer(t, recorder).URL})

	if err := client.Forget(context.Background(), "node-a", time.Second); err != nil {
		t.Fatalf("Forget() error = %v", err)
	}
	if len(recorder.calls) != 2 || recorder.calls[0] != "revoke" || recorder.calls[1] != "remove" {
		t.Fatalf("calls = %v, want revoke then remove", recorder.calls)
	}
}

// A retry after a partial failure must still succeed, so a node that is already
// revoked is the state this step wants rather than an error.
func TestForgetToleratesAnAlreadyRevokedNode(t *testing.T) {
	recorder := &registryRecorder{
		revokeResponse: connect.NewError(connect.CodeFailedPrecondition, errors.New("node is already revoked")),
	}
	client := New(Config{BridgeURL: registryServer(t, recorder).URL})

	if err := client.Forget(context.Background(), "node-a", time.Second); err != nil {
		t.Fatalf("Forget() error = %v", err)
	}
	if len(recorder.calls) != 2 || recorder.calls[1] != "remove" {
		t.Fatalf("calls = %v, want the removal to proceed", recorder.calls)
	}
}

// Any other revocation failure must stop the removal: deleting the record of a
// node whose credentials still work is the outcome the registry exists to
// prevent.
func TestForgetStopsWhenRevocationFailsForAnyOtherReason(t *testing.T) {
	recorder := &registryRecorder{
		revokeResponse: connect.NewError(connect.CodeInternal, errors.New("registry unavailable")),
	}
	client := New(Config{BridgeURL: registryServer(t, recorder).URL})

	if err := client.Forget(context.Background(), "node-a", time.Second); err == nil {
		t.Fatal("Forget() succeeded despite a failed revocation")
	}
	for _, call := range recorder.calls {
		if call == "remove" {
			t.Fatal("the node was removed while it was still authorized")
		}
	}
}

func TestForgetAndDecideRefuseEmptyInput(t *testing.T) {
	client := New(Config{BridgeURL: "http://127.0.0.1:1"})
	if err := client.Forget(context.Background(), "  ", time.Second); !IsKind(err, ErrInvalidRequest) {
		t.Errorf("Forget(empty) = %v, want an invalid-request error", err)
	}
	if _, err := client.Decide(context.Background(), DecideRequest{RequestID: " "}, time.Second); !IsKind(err, ErrInvalidRequest) {
		t.Errorf("Decide(empty) = %v, want an invalid-request error", err)
	}
	// Approving without the words the operator read off the other machine is
	// refused before any request is sent.
	if _, err := client.Decide(context.Background(), DecideRequest{RequestID: "req-1", Approve: true}, time.Second); !IsKind(err, ErrInvalidRequest) {
		t.Errorf("Decide(no words) = %v, want an invalid-request error", err)
	}
}

func TestSetScopesRefusesAnEmptyNode(t *testing.T) {
	client := New(Config{BridgeURL: "http://127.0.0.1:1"})
	if _, err := client.SetScopes(context.Background(), "", []string{"*:read"}, time.Second); !IsKind(err, ErrInvalidRequest) {
		t.Errorf("SetScopes(empty) = %v, want an invalid-request error", err)
	}
}
