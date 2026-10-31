package sessions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"
)

var ErrInvalidWorkspace = errors.New("invalid sandbox workspace")

// WorkspaceResolver turns the caller's workspace identifier into the
// directory that a session kernel is permitted to use as its cwd.
type WorkspaceResolver interface {
	Resolve(context.Context, string) (string, error)
}

type localWorkspaceResolver struct{}

func (*localWorkspaceResolver) Resolve(ctx context.Context, id string) (string, error) {
	if !filepath.IsAbs(strings.TrimSpace(id)) {
		return "", fmt.Errorf("%w: local fallback requires an absolute path", ErrInvalidWorkspace)
	}
	return validateWorkspacePath(ctx, id)
}

// TypedWorkspaceResolver resolves workspace roots through workspace-sandbox's
// generated Connect client. The path is validated again locally so a typed
// response never weakens the kernel's independent containment check.
type TypedWorkspaceResolver struct {
	resolver *discovery.Resolver
	client   *http.Client
}

func NewTypedWorkspaceResolver(resolver *discovery.Resolver, client *http.Client) *TypedWorkspaceResolver {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &TypedWorkspaceResolver{resolver: resolver, client: client}
}

func (r *TypedWorkspaceResolver) Resolve(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: identifier is empty", ErrInvalidWorkspace)
	}

	// This is the documented degraded path for installations without
	// workspace-sandbox: an explicit absolute directory can still be used, but
	// it is local path validation rather than copy-on-write isolation.
	if filepath.IsAbs(id) {
		return validateWorkspacePath(ctx, id)
	}

	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "workspace-sandbox")
	if err != nil {
		return "", fmt.Errorf("%w %q: workspace-sandbox unavailable: %v", ErrInvalidWorkspace, id, err)
	}
	client := workspaceconnect.NewWorkspaceSandboxServiceClient(r.client, strings.TrimRight(base, "/"))
	response, err := client.ResolveWorkspace(ctx, connect.NewRequest(&workspacev1.ResolveWorkspaceRequest{SandboxId: id}))
	if err != nil {
		return "", fmt.Errorf("%w %q: resolve workspace: %v", ErrInvalidWorkspace, id, err)
	}
	if !response.Msg.GetSuccess() {
		return "", fmt.Errorf("%w %q: workspace-sandbox returned unsuccessful response", ErrInvalidWorkspace, id)
	}
	return validateWorkspacePath(ctx, response.Msg.GetWorkspaceRoot())
}

func validateWorkspacePath(ctx context.Context, path string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%w: workspace path is empty", ErrInvalidWorkspace)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w %q: make path absolute: %v", ErrInvalidWorkspace, path, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("%w %q: resolve path: %v", ErrInvalidWorkspace, path, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("%w %q: stat path: %v", ErrInvalidWorkspace, path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w %q: path is not a directory", ErrInvalidWorkspace, path)
	}
	return resolved, nil
}
