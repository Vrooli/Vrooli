package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/discovery"
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

// DiscoveryWorkspaceResolver uses workspace-sandbox's current REST endpoint.
// The endpoint is intentionally kept behind this one-method seam because the
// scenario does not yet publish a shared typed workspace-resolution API.
type DiscoveryWorkspaceResolver struct {
	resolver *discovery.Resolver
	client   *http.Client
}

func NewDiscoveryWorkspaceResolver(resolver *discovery.Resolver, client *http.Client) *DiscoveryWorkspaceResolver {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &DiscoveryWorkspaceResolver{resolver: resolver, client: client}
}

func (r *DiscoveryWorkspaceResolver) Resolve(ctx context.Context, id string) (string, error) {
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
	endpoint := strings.TrimRight(base, "/") + "/api/v1/sandboxes/" + url.PathEscape(id) + "/workspace"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("%w %q: create workspace request: %v", ErrInvalidWorkspace, id, err)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w %q: resolve workspace: %v", ErrInvalidWorkspace, id, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%w %q: workspace-sandbox returned HTTP %d", ErrInvalidWorkspace, id, response.StatusCode)
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("%w %q: decode workspace response: %v", ErrInvalidWorkspace, id, err)
	}
	return validateWorkspacePath(ctx, payload.Path)
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
