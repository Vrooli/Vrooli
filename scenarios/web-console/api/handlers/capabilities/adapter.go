package capabilities

import (
	"context"
	"log"
	"time"

	"web-console/internal/backend"

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
	svc := caps.LifecycleActionService{
		Defs:    caps.Known,
		Runner:  a.ActionRunner,
		CLIPath: a.CLIPath,
	}
	result, err := svc.Run(ctx, caps.LifecycleActionRequest{
		CapabilityID: req.CapabilityID,
		ActionKind:   caps.ActionKind(req.ActionKind),
	})
	if err != nil {
		return ActionResult{}, err
	}
	if a.Registry != nil {
		a.Registry.Invalidate()
	}
	snap := a.Resolve(ctx)
	return ActionResult{
		Success:      result.Success,
		Status:       result.Status,
		Message:      result.Message,
		CapabilityID: result.CapabilityID,
		ActionKind:   string(result.ActionKind),
		Snapshot:     snap,
	}, nil
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
			ID:              c.ID,
			Name:            c.Name,
			Description:     c.Description,
			DependencyKind:  string(c.DependencyKind),
			DependencySlug:  c.DependencySlug,
			Features:        c.Features,
			Status:          string(c.Status),
			Message:         c.Message,
			CheckedAt:       c.CheckedAt,
			ReasonCode:      c.ReasonCode,
			ActionKind:      string(c.ActionKind),
			ActionLabel:     c.ActionLabel,
			OperatorCommand: c.OperatorCommand,
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
