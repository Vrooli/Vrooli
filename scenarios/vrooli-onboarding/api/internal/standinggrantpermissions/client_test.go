package standinggrantpermissions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions"
	permissionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions/permissionsv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type projectionServer struct{ t *testing.T }

func (s projectionServer) ListPermissionProjection(_ context.Context, r *connect.Request[permissionsv1.ListPermissionProjectionRequest]) (*connect.Response[permissionsv1.ListPermissionProjectionResponse], error) {
	if r.Header().Get("Authorization") != "Bearer original-owner-proof" {
		s.t.Errorf("owner proof was not forwarded")
	}
	now := timestamppb.New(time.Now().UTC())
	return connect.NewResponse(&permissionsv1.ListPermissionProjectionResponse{Owner: Owner, SourceRevision: "r1", ScopeCatalog: []*permissionsv1.PermissionScopeDeclaration{{Scenario: Owner, Scope: "agent-manager:read", Effect: "read", Command: "permissions/list"}}, Grants: []*permissionsv1.StandingGrantMetadata{{GrantRef: "opaque-ref", Subject: "worker", Approver: "human-owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, State: "approved", ApprovedAt: now, ExpiresAt: timestamppb.New(time.Now().Add(time.Hour))}}}), nil
}
func (projectionServer) ReadPermissionProjection(context.Context, *connect.Request[permissionsv1.ReadPermissionProjectionRequest]) (*connect.Response[permissionsv1.ReadPermissionProjectionResponse], error) {
	return connect.NewResponse(&permissionsv1.ReadPermissionProjectionResponse{}), nil
}

func TestClientUsesTypedOwnerConnectContractAndForwardsProof(t *testing.T) {
	router := http.NewServeMux()
	path, handler := permissionsconnect.NewPermissionProjectionServiceHandler(projectionServer{t})
	router.Handle(path, handler)
	server := httptest.NewServer(router)
	defer server.Close()
	client := &Client{BaseURL: func(context.Context) (string, error) { return server.URL, nil }, HTTP: server.Client()}
	ctx := WithIncomingOwnerProof(context.Background(), http.Header{"Authorization": []string{"Bearer original-owner-proof"}})
	page, err := client.List(ctx, 64, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Owner != Owner || page.SourceRevision != "r1" || len(page.Scopes) != 1 || page.Scopes[0].Value != "agent-manager:read" || len(page.Grants) != 1 || page.Grants[0].ID != "opaque-ref" {
		t.Fatalf("page conversion: %+v", page)
	}
}

func TestClientFailsClosedWithoutOwnerProof(t *testing.T) {
	client := &Client{BaseURL: func(context.Context) (string, error) { return "http://unused", nil }}
	if _, err := client.List(context.Background(), 64, ""); err == nil {
		t.Fatal("owner call without proof accepted")
	}
}
