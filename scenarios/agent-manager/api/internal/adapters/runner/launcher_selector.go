package runner

import (
	"context"
	"strings"
	"sync"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// LauncherSelector holds the per-runner launcher wiring and produces a
// Launcher per Execute call.
//
// Why this is its own type:
//
// All three coding-agent runners (claude_code, codex, opencode) need the
// same routing decision when picking between host and sandbox execution:
// "tracking mode → host; protected mode → sandbox if a factory is wired
// AND a sandbox ID is present, otherwise warn and fall back to host."
// Without this seam every runner would copy the switch into its Execute
// method, drift over time, and need parallel routing tests.
//
// The selector also owns the warn-event emission so misconfigured
// environments (protected mode requested, factory missing, etc.) surface
// as visible run-level log events rather than silent downgrades.
//
// The concrete type is exported (rather than only its constructor) so
// the generic [core.Runner] can hold it behind an interface while tests
// in the same package can compose it directly.
type LauncherSelector struct {
	mu             sync.RWMutex
	host           Launcher
	sandboxFactory SandboxLauncherFactory
}

// NewLauncherSelector returns a selector wired with the given launchers.
// A nil host launcher is replaced with a fresh [HostLauncher]; a nil
// factory means protected-mode requests will warn-and-fall-back to host.
//
// Exposed for use by the generic [core.Runner], which holds a selector
// behind an interface so the parent package's concrete type stays internal.
func NewLauncherSelector(host Launcher, factory SandboxLauncherFactory) *LauncherSelector {
	if host == nil {
		host = NewHostLauncher()
	}
	return &LauncherSelector{host: host, sandboxFactory: factory}
}

// SetSandboxLauncherFactory swaps in (or removes) the protected-mode
// factory. Used by main.go where the sandbox provider is constructed
// after the runner registry; tests use it to inject mocks.
func (s *LauncherSelector) SetSandboxLauncherFactory(factory SandboxLauncherFactory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandboxFactory = factory
}

// HostLauncher returns the configured host launcher. Used by Stop()
// implementations that need to talk directly to the host runtime when no
// LaunchedProcess is registered.
func (s *LauncherSelector) HostLauncher() Launcher {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.host
}

// SandboxFactory returns the currently-wired factory, or nil. Used in
// tests; production code should call [Pick] instead.
func (s *LauncherSelector) SandboxFactory() SandboxLauncherFactory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sandboxFactory
}

// Pick returns the Launcher that should be used for the given Execute
// request. Thin wrapper over [PickFor]; see that method for the full
// routing-rules contract.
func (s *LauncherSelector) Pick(ctx context.Context, req ExecuteRequest) Launcher {
	return s.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)
}

// PickFor is the primitives-based routing function shared by Execute and
// Continue paths. Both ExecuteRequest and ContinueRequest carry the same
// four pieces of information the selector needs (run id, run config,
// sandbox id, event sink); accepting them directly avoids forcing every
// request shape to conform to a common interface.
//
// Routing rules:
//
//  1. Tracking mode (or unspecified) → host launcher.
//  2. Protected mode without a wired factory → warn-and-fallback to host.
//  3. Protected mode without sandboxID → warn-and-fallback.
//  4. Protected mode with factory + sandboxID, factory returns nil →
//     warn-and-fallback. (Common when the provider doesn't recognise the
//     sandbox ID, e.g. cross-environment misconfiguration.)
//  5. Protected mode + factory + sandboxID + non-nil launcher → sandbox.
//
// Each warn-and-fallback path emits a log event on the supplied EventSink
// (when non-nil) so operators can spot misconfigured environments rather
// than silently watching protected runs downgrade to host execution.
func (s *LauncherSelector) PickFor(ctx context.Context, runID uuid.UUID, cfg *domain.RunConfig, sandboxID *uuid.UUID, sink EventSink) Launcher {
	_ = ctx // reserved for future per-call factory needs (e.g. tracing)
	s.mu.RLock()
	host := s.host
	factory := s.sandboxFactory
	s.mu.RUnlock()

	if cfg == nil || cfg.SandboxConfig == nil || cfg.SandboxConfig.Mode.Effective() != domain.SandboxModeProtected {
		return host
	}
	if factory == nil {
		emitLauncherFallbackWarn(runID, sink, "no SandboxLauncherFactory configured")
		return host
	}
	if sandboxID == nil {
		emitLauncherFallbackWarn(runID, sink, "SandboxID is nil")
		return host
	}
	launcher := factory.LauncherFor(*sandboxID)
	if launcher == nil {
		emitLauncherFallbackWarn(runID, sink, "factory returned nil launcher")
		return host
	}
	return launcher
}

// emitLauncherFallbackWarn surfaces a warn-level run log when protected
// mode was requested but cannot be honored. Reason becomes part of the
// log message so operators can grep for the specific failure.
func emitLauncherFallbackWarn(runID uuid.UUID, sink EventSink, reason string) {
	if sink == nil {
		return
	}
	msg := strings.Join([]string{
		"protected mode requested but ",
		reason,
		"; falling back to HostLauncher",
	}, "")
	_ = sink.Emit(domain.NewLogEvent(runID, "warn", msg))
}
