// Package workspace_sandbox is the optional Connect client to
// workspace-sandbox for fetching per-path file content when the
// manifest's content rules need body inspection. The dependency is
// declared optional in .vrooli/service.json; when it's unreachable the
// validation_run evaluator gracefully degrades to path-only verdicts.
//
// CURRENT STATE (2026-05-18): this adapter is a not-yet-wired stub —
// it probes discovery so degradation messages are honest, but the
// content-fetch RPC mapping is followup work documented in PROBLEMS.md.
package workspace_sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	vrun "development-toolchain-validator/internal/validation_run"

	"development-toolchain-validator/internal/httpc"
	"github.com/vrooli/api-core/discovery"
)

// Options configures the adapter.
type Options struct {
	Resolver *discovery.Resolver
	Doer     httpc.Doer
}

// Client implements vrun.WorkspaceSandboxClient.
type Client struct {
	opts Options
}

// New constructs a Client.
func New(opts Options) *Client {
	if opts.Resolver == nil {
		opts.Resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if opts.Doer == nil {
		opts.Doer = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{opts: opts}
}

var _ vrun.WorkspaceSandboxClient = (*Client)(nil)

// FetchPathContent returns the content of a single file in the
// sandboxed workspace. Returns a sentinel "unavailable" error when
// workspace-sandbox isn't running so the worker degrades to a
// path-only manifest evaluation.
func (c *Client) FetchPathContent(ctx context.Context, _ , path string) (string, error) {
	if _, err := c.opts.Resolver.ResolveScenarioURLDefault(ctx, "workspace-sandbox"); err != nil {
		if discovery.IsScenarioNotRunning(err) {
			return "", errors.New("workspace-sandbox not running; content rule evaluation skipped")
		}
		return "", fmt.Errorf("resolve workspace-sandbox: %w", err)
	}
	return "", fmt.Errorf("workspace-sandbox content fetch not yet wired; see PROBLEMS.md (path=%s)", path)
}
