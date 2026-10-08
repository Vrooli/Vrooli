package sessions

import (
	"context"
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

type fakeWorkspaceService struct {
	workspaceconnect.UnimplementedWorkspaceSandboxServiceHandler
	root string
}

func (f fakeWorkspaceService) ResolveWorkspace(_ context.Context, req *connect.Request[workspacev1.ResolveWorkspaceRequest]) (*connect.Response[workspacev1.ResolveWorkspaceResponse], error) {
	return connect.NewResponse(&workspacev1.ResolveWorkspaceResponse{Success: true, SandboxId: req.Msg.GetSandboxId(), WorkspaceRoot: f.root, IsolationMode: "copy"}), nil
}
