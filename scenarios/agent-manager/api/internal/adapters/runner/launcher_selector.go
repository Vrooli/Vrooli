package runner

import (
	"context"
	"fmt"
	"slices"
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
// Tracking may fall back with a warning. Protected or effect-bearing runs
// require a bound launcher and a verified containment report; they never
// substitute host execution when either is unavailable.
// Without this seam every runner would copy the switch into its Execute
// method, drift over time, and need parallel routing tests.
//
// The selector owns tracking fallback warnings and protected launch refusals.
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
// factory refuses protected-mode requests; tracking requests warn and use host.
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
// Protected mode and effect-bearing runs fail closed on missing launch wiring,
// identity or containment evidence. Tracking preserves its explicit fallback;
// explicit off mode uses the host unless effect containment is required.
func (s *LauncherSelector) PickFor(ctx context.Context, runID uuid.UUID, cfg *domain.RunConfig, sandboxID *uuid.UUID, sink EventSink) Launcher {
	s.mu.RLock()
	host := s.host
	factory := s.sandboxFactory
	s.mu.RUnlock()

	if cfg == nil || cfg.SandboxConfig == nil {
		return newDeniedLauncher("required containment policy has no sandbox configuration")
	}
	mode := cfg.SandboxConfig.Mode.Effective()
	if cfg.SandboxConfig.WritePolicy != nil && mode != domain.SandboxModeProtected {
		return newDeniedLauncher("workspace write policy requires protected mode")
	}
	if !mode.IsValid() {
		return newDeniedLauncher("unknown sandbox mode: " + string(mode))
	}
	requiresContainment := cfg.RequireEffectContainment || mode == domain.SandboxModeProtected
	if mode == domain.SandboxModeOff {
		if requiresContainment {
			return newDeniedLauncher("effect-bearing run requires protected workspace containment")
		}
		return host
	}
	fallback := func(reason string) Launcher {
		if requiresContainment {
			return newDeniedLauncher("required workspace containment unavailable: " + reason)
		}
		emitLauncherFallbackWarn(runID, sink, reason)
		return host
	}
	if factory == nil {
		return fallback("no SandboxLauncherFactory configured for " + string(mode) + " mode")
	}
	if sandboxID == nil || *sandboxID == uuid.Nil {
		return fallback("SandboxID is nil or zero for " + string(mode) + " mode")
	}
	launcher := factory.LauncherFor(*sandboxID)
	if launcher == nil {
		return fallback("factory returned nil launcher")
	}
	var cont *Containment
	if reporter, ok := factory.(SandboxContainmentReporter); ok {
		if reported, available := reporter.ContainmentFor(ctx, *sandboxID); available {
			cont = reported
		}
	}
	if requiresContainment {
		if cont == nil || cont.Level != "required" || cont.Backend == "" || cont.Backend == "none" {
			return newDeniedLauncher("required workspace containment report is missing or uncontained")
		}
		if missing := cont.MissingProtectedEnforcements(); len(missing) > 0 {
			return newDeniedLauncher("required workspace containment missing: " + strings.Join(missing, ", "))
		}
		if cfg.SandboxConfig.WritePolicy != nil && !cont.HasEnforcement(EnforcementWorkspaceWritePolicy) {
			return newDeniedLauncher("required workspace containment missing: " + EnforcementWorkspaceWritePolicy)
		}
		if requested := cfg.SandboxConfig.WritePolicy; requested != nil &&
			(cont.WritePolicy == nil || !slices.Equal(requested.Paths, cont.WritePolicy.Paths)) {
			return newDeniedLauncher("sandbox's persisted workspace write policy does not match the admitted run")
		}
	}
	networkMode := domain.NetworkAccessLocalhost
	if cfg != nil && cfg.SandboxConfig != nil {
		networkMode = cfg.SandboxConfig.NetworkMode.Effective()
	}
	emitContainmentGapWarn(cont, runID, sink, networkMode)
	return launcher
}

type deniedLauncher struct{ reason string }

func newDeniedLauncher(reason string) Launcher { return &deniedLauncher{reason: reason} }

func (l *deniedLauncher) Launch(context.Context, LaunchRequest) (LaunchedProcess, error) {
	return nil, fmt.Errorf("launch refused: %s", l.reason)
}

// emitContainmentGapWarn records the report already read during selection.
// Baseline gaps reach this point only for tracking. Loopback capability remains
// distinct from baseline containment and is not claimed when unavailable.
func emitContainmentGapWarn(cont *Containment, runID uuid.UUID, sink EventSink, networkMode domain.NetworkAccess) {
	if sink == nil || cont == nil {
		return
	}
	if missing := cont.MissingProtectedEnforcements(); len(missing) > 0 {
		msg := strings.Join([]string{
			"selected sandbox does not enforce ",
			strings.Join(missing, ", "),
			" (effective containment: backend=", cont.Backend,
			", level=", cont.Level,
			", enforcements=[", strings.Join(cont.Enforcements, ", "), "])",
		}, "")
		_ = sink.Emit(domain.NewLogEvent(runID, "warn", msg))
	}
	// The sandbox launcher starts every protected agent under the
	// vrooli-aware profile ("localhost" network mode), but no containment
	// backend enforces loopback-only network — "localhost" currently
	// grants unrestricted network (knw-1784006975589682125). Surface that
	// on the run timeline so the run's true network posture is never
	// silent; the warn disappears once a backend claims the enforcement.
	if networkMode == domain.NetworkAccessLocalhost && !cont.HasEnforcement(EnforcementNetworkLoopbackOnly) {
		_ = sink.Emit(domain.NewLogEvent(runID, "warn",
			"protected agent launches under the vrooli-aware profile (localhost network mode) but the containment backend lacks "+
				EnforcementNetworkLoopbackOnly+" — the agent has unrestricted network access"))
	}
}

// emitLauncherFallbackWarn surfaces a warn-level run log when tracking
// could not obtain its sandbox launcher. Reason becomes part of the
// log message so operators can grep for the specific failure.
func emitLauncherFallbackWarn(runID uuid.UUID, sink EventSink, reason string) {
	if sink == nil {
		return
	}
	msg := strings.Join([]string{
		"tracking mode requested but ",
		reason,
		"; falling back to HostLauncher",
	}, "")
	_ = sink.Emit(domain.NewLogEvent(runID, "warn", msg))
}
