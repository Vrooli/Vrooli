package runner

import (
	"context"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
)

// FiniteNativeFactory is explicitly installed alongside the finite authority.
// It is concrete: editable requests cannot substitute a host launcher. Main
// installs nothing by default; a disabled manifest always refuses.
type FiniteNativeFactory struct{ runtime *isolation.Runtime }

func NewFiniteNativeFactory(manifest isolation.Manifest) (*FiniteNativeFactory, error) {
	rt, e := isolation.NewRuntime(manifest)
	if e != nil {
		return nil, e
	}
	return &FiniteNativeFactory{rt}, nil
}

// Terminal is the native unit owner's additional settlement gate. Installation
// uses orchestration.InstallFiniteNativeIsolation to couple launch and settlement.
func (f *FiniteNativeFactory) Enabled() bool {
	return f != nil && f.runtime != nil && f.runtime.Enabled()
}

func (f *FiniteNativeFactory) CheckBinding(id string, binding isolation.Binding) error {
	if !f.Enabled() {
		return isolation.ErrRefused
	}
	return f.runtime.CheckBinding(id, binding)
}

func (f *FiniteNativeFactory) Terminal(ctx context.Context, id string) error {
	if f == nil || f.runtime == nil {
		return isolation.ErrUnknown
	}
	return f.runtime.Terminal(ctx, id)
}
func (f *FiniteNativeFactory) pick(id uuid.UUID, cfg *domain.RunConfig) Launcher {
	if f == nil || f.runtime == nil || cfg == nil || cfg.Admission == nil || cfg.Admission.Effort == nil || cfg.Admission.EffortIntent == nil {
		return newDeniedLauncher("finite native isolation is not installed")
	}
	i := cfg.Admission.EffortIntent
	b := cfg.Admission.Effort
	// Interactive, child and source recovery need separately qualified migration.
	// This contract supports the real codec pipe initial/durable native route.
	if i.Effect != "run.create" || i.ParentRunID != "" || i.SourceRunID != "" {
		return newDeniedLauncher("finite native child/recovery route is unqualified")
	}
	return &finiteNativeLauncher{f.runtime, isolation.Request{RunID: id.String(), PolicyID: b.PolicyID, PolicyDigest: b.PolicyDigest, ProfileDigest: i.ProfileDigest, Repository: i.Repository, Deadline: b.Deadline, Timeout: cfg.Timeout}}
}

type finiteNativeLauncher struct {
	runtime *isolation.Runtime
	request isolation.Request
}

func (l *finiteNativeLauncher) Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error) {
	if req.RunID.String() != l.request.RunID || len(req.PolicyFiles) > 0 {
		return nil, isolation.ErrRefused
	}
	// Existing protected workspace/policy files are a different contract; never
	// silently bypass them. This first finite route narrows off-mode host runs.
	if req.NetworkMode != "" && req.NetworkMode != string(domain.NetworkAccessFull) {
		return nil, isolation.ErrRefused
	}
	r := l.request
	r.Command = req.Command
	r.Args = append([]string{}, req.Args...)
	r.Env = append([]string{}, req.Env...)
	r.WorkingDir = req.WorkingDir
	p, e := l.runtime.Start(ctx, r, req.Stdin)
	if e != nil {
		return nil, e
	}
	if p.PID() <= 0 {
		p.Kill()
		_ = p.Wait()
		return nil, isolation.ErrUnknown
	}
	p.SetIdleTimeout(req.IdleTimeout)
	return p, nil
}

var _ LaunchedProcess = (*isolation.Process)(nil)
