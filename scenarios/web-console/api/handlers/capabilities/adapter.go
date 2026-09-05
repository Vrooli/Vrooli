package capabilities

import (
	"context"
	"fmt"
	"log"
	"time"

	"web-console/internal/backend"

	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	caps "web-console/internal/capabilities"
)

// Adapter is the production Service implementation: it bridges the
// in-process capability registry and backend registry to the transport
// shape the Connect handler consumes. Constructed in api/main.go and
// passed to Module.
//
// The defaultBackend hook is a closure rather than a direct dependency
// on session.Manager so this package stays free of session/Server
// imports; the call site supplies whatever logic resolves the active
// default backend.
type Adapter struct {
	Registry        *caps.Registry
	BackendRegistry *backend.Registry
	DefaultBackend  func() string
	Logger          *log.Logger
	ActionRunner    caps.CommandRunner
	CLIPath         string
	RemoteInstall   func(context.Context, string, string) (caps.LifecycleActionResult, error)
	// ConfirmInstall asks the target itself whether a capability is now
	// present, after an install has run. It is the only evidence this package
	// accepts that an install worked: an exit code reports that a command ran,
	// which is a different claim. Nil leaves every install unconfirmed rather
	// than silently promoting it to a success.
	ConfirmInstall func(ctx context.Context, targetID, capabilityID string) (state, version string)
}

// stampInstallOutcome replaces an installer's own verdict with the target's.
//
// It runs for operator-command actions only, because those are the ones that
// claim to add something to a machine. A failed installer keeps its failure
// and its message; a not-applicable answer is already final and is left alone.
func (a *Adapter) stampInstallOutcome(ctx context.Context, targetID string, result caps.LifecycleActionResult) caps.LifecycleActionResult {
	if result.Status == caps.InstallStatusNotApplicable {
		return result
	}
	if !result.Success {
		result.Status = caps.InstallStatusFailed
		if result.Message == "" {
			result.Message = "the installer did not complete"
		}
		return result
	}
	if a.ConfirmInstall == nil {
		result.Success = false
		result.Status = caps.InstallStatusUnconfirmed
		result.Message = "The installer finished, but this machine cannot report whether the agent is now available."
		return result
	}
	state, version := a.ConfirmInstall(ctx, targetID, result.CapabilityID)
	if state == caps.InstallConfirmedState {
		result.Status = caps.InstallStatusInstalled
		result.Message = "Installed" + versionSuffix(version) + "."
		return result
	}
	// The installer succeeded and the machine still does not report the agent.
	// That is genuinely unknown rather than a failure — a machine that reports
	// its inventory on a heartbeat may simply not have reported yet — so say
	// so, and never render it as a completed install.
	result.Success = false
	result.Status = caps.InstallStatusUnconfirmed
	result.Message = "The installer finished, but this machine has not reported the agent as available yet."
	return result
}

func versionSuffix(version string) string {
	if version == "" {
		return ""
	}
	return " (" + version + ")"
}

// Describe exposes the registry contract used by the settings surface and
// dependency conformance. It intentionally bypasses the richer session
// backend projection returned by Resolve.
func (a *Adapter) Describe(ctx context.Context) ([]byte, error) {
	if a == nil || a.Registry == nil {
		return nil, fmt.Errorf("capabilities registry is not configured")
	}
	return a.Registry.Describe(ctx)
}

// Resolve runs the full capability check (with cache) and includes
// session backend options + the active default backend.
func (a *Adapter) Resolve(ctx context.Context) Snapshot {
	start := time.Now()
	states := a.Registry.Resolve(ctx)
	a.log("capabilities: full resolve took %dms", time.Since(start).Milliseconds())

	snap := Snapshot{
		Capabilities: capsToTransport(states),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	if a.BackendRegistry != nil {
		snap.BackendOptions = backendsToTransport(a.BackendRegistry.Available())
	}
	if a.DefaultBackend != nil {
		snap.DefaultBackend = a.DefaultBackend()
	}
	return snap
}

// Liveness returns capability states using fast liveness-only checks.
// BackendOptions and DefaultBackend are intentionally omitted — clients
// that need them call Resolve.
func (a *Adapter) Liveness(ctx context.Context) Snapshot {
	start := time.Now()
	states := a.Registry.ResolveLiveness(ctx)
	a.log("capabilities: liveness resolve took %dms", time.Since(start).Milliseconds())
	return Snapshot{
		Capabilities: capsToTransport(states),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
}

func (a *Adapter) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	if req.TargetID != "" && req.TargetID != "local" && req.ActionKind == string(caps.ActionKindOperatorCommand) {
		if a.RemoteInstall == nil {
			return ActionResult{}, fmt.Errorf("remote capability installation is not configured")
		}
		result, err := a.RemoteInstall(ctx, req.TargetID, req.CapabilityID)
		if err != nil {
			return ActionResult{}, err
		}
		result.CapabilityID = req.CapabilityID
		result = a.stampInstallOutcome(ctx, req.TargetID, result)
		return ActionResult{Success: result.Success, Status: result.Status, Message: result.Message, OperationID: result.OperationID, CapabilityID: result.CapabilityID, ActionKind: string(result.ActionKind), Snapshot: a.Resolve(ctx)}, nil
	}
	if req.ActionKind == string(caps.ActionKindScenarioStart) || req.ActionKind == string(caps.ActionKindScenarioRestart) {
		if a.Registry == nil {
			return ActionResult{}, fmt.Errorf("capabilities registry is not configured")
		}
		var current *caps.State
		for _, state := range a.Registry.Resolve(ctx) {
			if state.ID == req.CapabilityID {
				candidate := state
				current = &candidate
				break
			}
		}
		if current == nil {
			return ActionResult{}, fmt.Errorf("capability %q is not declared", req.CapabilityID)
		}
		if current.ActionKind != caps.ActionKind(req.ActionKind) || current.Status != caps.StatusUnavailable {
			return ActionResult{}, fmt.Errorf("lifecycle action %q is not eligible while capability %q is %s", req.ActionKind, req.CapabilityID, current.Status)
		}
	}
	svc := capreg.LifecycleActionService{
		Defs:    caps.Known,
		Runner:  sharedCommandRunner{runner: a.ActionRunner},
		CLIPath: a.CLIPath,
	}
	sharedResult, err := svc.Run(ctx, capreg.LifecycleActionRequest{
		IntegrationID: req.CapabilityID,
		ActionKind:    capreg.ActionKind(req.ActionKind),
	})
	if err != nil {
		return ActionResult{}, err
	}
	result := caps.LifecycleActionResult{
		Success:      sharedResult.Success,
		Status:       sharedResult.Status,
		Message:      sharedResult.Message,
		CapabilityID: sharedResult.IntegrationID,
		ActionKind:   caps.ActionKind(sharedResult.ActionKind),
	}
	if a.Registry != nil {
		a.Registry.Invalidate()
	}
	if req.ActionKind == string(caps.ActionKindOperatorCommand) {
		result.CapabilityID = req.CapabilityID
		result = a.stampInstallOutcome(ctx, req.TargetID, result)
	}
	snap := a.Resolve(ctx)
	return ActionResult{
		Success:      result.Success,
		Status:       result.Status,
		Message:      result.Message,
		OperationID:  result.OperationID,
		CapabilityID: result.CapabilityID,
		ActionKind:   string(result.ActionKind),
		Snapshot:     snap,
	}, nil
}

type sharedCommandRunner struct{ runner caps.CommandRunner }

func (r sharedCommandRunner) Run(ctx context.Context, name string, args ...string) (capreg.CommandResult, error) {
	if r.runner == nil {
		return capreg.ExecCommandRunner{}.Run(ctx, name, args...)
	}
	result, err := r.runner.Run(ctx, name, args...)
	return capreg.CommandResult{Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode}, err
}

func (a *Adapter) log(format string, args ...any) {
	if a.Logger != nil {
		a.Logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}

func capsToTransport(in []caps.State) []CapabilityState {
	out := make([]CapabilityState, len(in))
	for i, c := range in {
		out[i] = CapabilityState{
			ID:                     c.ID,
			Name:                   c.Name,
			Description:            c.Description,
			DependencyKind:         string(c.DependencyKind),
			DependencySlug:         c.DependencySlug,
			Features:               c.Features,
			Status:                 string(c.Status),
			Message:                c.Message,
			CheckedAt:              c.CheckedAt,
			ReasonCode:             c.ReasonCode,
			ActionKind:             string(c.ActionKind),
			ActionLabel:            c.ActionLabel,
			OperatorCommand:        c.OperatorCommand,
			FeatureStatus:          c.FeatureStatus,
			FeatureReason:          c.FeatureReason,
			FeatureOperatorCommand: c.FeatureOperatorCommand,
			ProviderStatus:         c.ProviderStatus,
			ProviderFeatures:       c.ProviderFeatures,
		}
	}
	return out
}

func backendsToTransport(in []backend.Descriptor) []BackendOption {
	out := make([]BackendOption, len(in))
	for i, b := range in {
		out[i] = BackendOption{
			ID:              string(b.ID),
			DisplayName:     b.DisplayName,
			Description:     b.Description,
			SurvivesRestart: b.SurvivesRestart,
			Available:       b.Available,
			Reason:          b.Reason,
		}
	}
	return out
}
