// Package agent_manager hosts the outbound Connect client to
// agent-manager. The client resolves the agent-manager scenario's URL
// via api-core/discovery and exposes the AgentManagerClient seam the
// validation_run worker consumes.
//
// CURRENT STATE (2026-05-18): the agent-manager API requires a
// pre-created Task (via CreateTask) plus an AgentProfile id before
// CreateRun can be invoked. Wiring DTV to those primitives is
// followup work; for now this adapter returns ErrDependencyUnavailable
// so the validation_run worker visibly degrades rather than silently
// faking success in production. Test coverage of the worker uses the
// in-package mocks/.FakeAgentManager. See PROBLEMS.md for the followup
// tracking note.
package agent_manager

import (
	"context"
	"net/http"
	"time"

	vrun "development-toolchain-validator/internal/validation_run"

	"development-toolchain-validator/internal/httpc"
	"github.com/vrooli/api-core/discovery"
)

// Options configures the adapter.
type Options struct {
	Resolver    *discovery.Resolver
	Doer        httpc.Doer
	MaxAttempts int
}

// Client is the Connect adapter implementing vrun.AgentManagerClient.
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
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	return &Client{opts: opts}
}

var _ vrun.AgentManagerClient = (*Client)(nil)

// StartSandboxedRun is a not-yet-wired stub. agent-manager's
// CreateRun requires task_id + agent_profile_id from prior provisioning
// RPCs; DTV does not yet manage those. Until that wiring lands, the
// adapter surfaces ErrDependencyUnavailable so callers (the
// validation_run worker) classify runs as RUN_FAILURE with an
// actionable cause rather than fabricating success.
func (c *Client) StartSandboxedRun(ctx context.Context, _ vrun.SandboxedRunSpec) (string, error) {
	// Probe discovery so we still distinguish "scenario not running"
	// from "wiring not implemented yet" in operator logs.
	if _, err := c.opts.Resolver.ResolveScenarioURLDefault(ctx, "agent-manager"); err != nil {
		if discovery.IsScenarioNotRunning(err) {
			return "", vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "scenario not running"}
		}
	}
	return "", vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "task/profile wiring not yet implemented; see PROBLEMS.md"}
}

// WaitForTerminal is the unreachable companion to StartSandboxedRun.
// Returning the same not-implemented error keeps the worker's error
// path uniform across the two call sites.
func (c *Client) WaitForTerminal(_ context.Context, runID string, _ time.Duration) (vrun.RunSummary, error) {
	return vrun.RunSummary{AgentManagerRunID: runID}, vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "task/profile wiring not yet implemented; see PROBLEMS.md"}
}
