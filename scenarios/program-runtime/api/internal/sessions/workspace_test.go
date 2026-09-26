package sessions

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"
)

func TestTypedWorkspaceResolverResolvesWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	serverPath, serverHandler := workspaceconnect.NewWorkspaceSandboxServiceHandler(fakeWorkspaceService{root: root})
	if serverPath == "" || serverHandler == nil {
		t.Fatal("generated workspace handler was not constructed")
	}
	server := httptest.NewServer(serverHandler)
	defer server.Close()

	resolver := NewTypedWorkspaceResolver(discovery.NewStaticResolver(server.URL), server.Client())
	got, err := resolver.Resolve(context.Background(), "sandbox-123")
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("resolved=%q want=%q", got, root)
	}
}

func TestTypedWorkspaceResolverLocalFallbackValidatesPath(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	resolver := NewTypedWorkspaceResolver(nil, nil)
	got, err := resolver.Resolve(context.Background(), root)
	if err != nil || got != root {
		t.Fatalf("resolved=%q err=%v want=%q", got, err, root)
	}
}

type fakeWorkspaceService struct{ root string }

func (f fakeWorkspaceService) ResolveWorkspace(_ context.Context, req *connect.Request[workspacev1.ResolveWorkspaceRequest]) (*connect.Response[workspacev1.ResolveWorkspaceResponse], error) {
	return connect.NewResponse(&workspacev1.ResolveWorkspaceResponse{Success: true, SandboxId: req.Msg.GetSandboxId(), WorkspaceRoot: f.root, IsolationMode: "copy"}), nil
}

func (fakeWorkspaceService) CreateSandbox(context.Context, *connect.Request[workspacev1.CreateSandboxRequest]) (*connect.Response[workspacev1.CreateSandboxResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (fakeWorkspaceService) GetSandboxDiff(context.Context, *connect.Request[workspacev1.GetSandboxDiffRequest]) (*connect.Response[workspacev1.GetSandboxDiffResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (fakeWorkspaceService) PromoteSandbox(context.Context, *connect.Request[workspacev1.PromoteSandboxRequest]) (*connect.Response[workspacev1.PromoteSandboxResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}
