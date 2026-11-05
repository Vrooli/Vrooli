package workspace

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace"
	v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/server"
	internal "nutrition-planner/internal/workspace"
)

func TestWorkspaceAPIUsesLocalOwnerWithoutAuthentication(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(internal.Schema()); err != nil {
		t.Fatal(err)
	}

	workspaces := internal.NewService(internal.NewSQLiteRepository(db, schedule.System()))
	app := server.New(server.Deps{Clock: schedule.System()}, ModuleWithService(workspaces, nil))
	httpServer := httptest.NewServer(app.Handler())
	t.Cleanup(httpServer.Close)
	client := v1connect.NewWorkspaceServiceClient(httpServer.Client(), httpServer.URL)

	created, err := client.CreateWorkspace(context.Background(), connect.NewRequest(&v1.CreateWorkspaceRequest{Name: "Local kitchen"}))
	if err != nil {
		t.Fatalf("anonymous create failed: %v", err)
	}
	if got := created.Msg.GetWorkspace().GetOwnerSubject(); got != internal.LocalOwnerSubject {
		t.Fatalf("owner subject = %q, want fixed local owner %q", got, internal.LocalOwnerSubject)
	}

	listed, err := client.ListWorkspaces(context.Background(), connect.NewRequest(&v1.ListWorkspacesRequest{}))
	if err != nil {
		t.Fatalf("anonymous list failed: %v", err)
	}
	if len(listed.Msg.GetWorkspaces()) != 1 || listed.Msg.GetWorkspaces()[0].GetId() != created.Msg.GetWorkspace().GetId() {
		t.Fatalf("anonymous list did not return created local workspace: %#v", listed.Msg.GetWorkspaces())
	}
}
