package runner

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
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

func NewFiniteNativeFactoryWithWitness(manifest isolation.Manifest, witness *isolation.Witness) (*FiniteNativeFactory, error) {
	rt, e := isolation.NewRuntimeWithWitness(manifest, witness)
	if e != nil {
		return nil, e
	}
	return &FiniteNativeFactory{rt}, nil
}

// NewFiniteNativeFactoryWithOwners binds the live witness and exact fixed-unit
// operation owner. A witness alone never confers permission to start a unit.
func NewFiniteNativeFactoryWithOwners(manifest isolation.Manifest, witness *isolation.Witness, owner *isolation.FixedUnitOwner) (*FiniteNativeFactory, error) {
	rt, e := isolation.NewRuntimeWithWitnessAndUnitOwner(manifest, witness, owner)
	if e != nil {
		return nil, e
	}
	return &FiniteNativeFactory{rt}, nil
}

// Terminal is the native unit owner's additional settlement gate. Installation
// uses orchestration.InstallFiniteNativeIsolation to couple launch and settlement.
func (f *FiniteNativeFactory) Enabled() bool {
	return f != nil && f.runtime != nil && f.runtime.Ready()
}

func (f *FiniteNativeFactory) CheckBinding(id string, binding isolation.Binding) error {
	if !f.Enabled() {
		return isolation.ErrRefused
	}
	if err := f.runtime.CheckBinding(id, binding); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return f.runtime.CheckReady(ctx)
}

func (f *FiniteNativeFactory) Terminal(ctx context.Context, id string) error {
	if f == nil || f.runtime == nil {
		return isolation.ErrUnknown
	}
	return f.runtime.Terminal(ctx, id)
}
func (f *FiniteNativeFactory) pick(id uuid.UUID, cfg *domain.RunConfig) Launcher {
	if !f.Enabled() || cfg == nil || cfg.Admission == nil || cfg.Admission.Effort == nil || cfg.Admission.EffortIntent == nil {
		return newDeniedLauncher("finite native isolation is not installed")
	}
	i := cfg.Admission.EffortIntent
	b := cfg.Admission.Effort
	// Recovery remains unqualified. A serial child must already have its exact
	// accepted reservation and retained terminal source under the owner engine.
	var serial *effortauthority.Intent
	if i.Effect == "run.child" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := f.runtime.CheckSerialEpisode(ctx, *b, *i, id.String())
		cancel()
		if err != nil {
			return newDeniedLauncher("finite native serial handoff is not accepted")
		}
		copy := *i
		serial = &copy
	} else if i.Effect != "run.create" || i.ParentRunID != "" || i.SourceRunID != "" {
		return newDeniedLauncher("finite native recovery route is unqualified")
	}
	return &finiteNativeLauncher{runtime: f.runtime, request: isolation.Request{RunID: id.String(), ReservationKey: i.IdempotencyKey, PolicyID: b.PolicyID, PolicyDigest: b.PolicyDigest, ProfileDigest: i.ProfileDigest, Repository: i.Repository, Deadline: b.Deadline, Timeout: cfg.Timeout}, serialIntent: serial, serialBinding: *b}
}

type finiteNativeLauncher struct {
	runtime       *isolation.Runtime
	request       isolation.Request
	serialIntent  *effortauthority.Intent
	serialBinding effortauthority.Binding
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
	if l.serialIntent != nil && l.runtime.CheckSerialEpisode(ctx, l.serialBinding, *l.serialIntent, l.request.RunID) != nil {
		return nil, isolation.ErrRefused
	}
	r := l.request
	r.Command = req.Command
	r.Args = append([]string{}, req.Args...)
	var e error
	r.Env, e = l.runtime.PrepareEnvironment(r.Repository, req.Env)
	if e != nil {
		return nil, e
	}
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
