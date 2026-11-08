package runtimesupervisorsafeguard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// ActivationEffects is the production native adapter for a frozen root update.
// It never renders a replacement definition from today's source tree. Its
// caller owns operator authority and admission's separate install-lock span.
type ActivationEffects struct {
	coreactivation.NativeWorker
	Store *coreactivation.Store
	ops   *activationOps
}

type supervisorStopFence interface {
	Stopped(context.Context) error
	Close() error
}
type supervisorLease struct {
	ID                           string
	PID                          int
	Build, Incarnation           string
	Started, Heartbeat, Deadline time.Time
	FleetRenewed                 bool
}
type activationOps struct {
	inspect     func(context.Context) (platform.CoreServiceState, error)
	command     func(context.Context, ...string) error
	validate    func(platform.RenderedArtifact, platform.Scope) platform.Verdict
	lock        func(context.Context, string) (func(), error)
	image       func(context.Context, int) (string, error)
	incarnation func(int) (string, error)
	lease       func(context.Context, string) (supervisorLease, error)
	pin         func(context.Context, platform.CoreServiceState) (supervisorStopFence, error)
	worker      func(context.Context, coreactivation.Record) error
	exclude     func(context.Context, coreactivation.Record) error
	release     func(context.Context, coreactivation.Record) error
}

func (f ActivationEffects) operations() activationOps {
	if f.ops != nil {
		return *f.ops
	}
	return activationOps{inspect: platform.InspectCoreSupervisor, command: activationCommand, validate: platform.ValidateArtifact, lock: buildinfo.AcquireBinaryInstallLockContext, image: coreactivation.RunningImageDigest, incarnation: process.Incarnation, lease: readActivationLease, pin: func(ctx context.Context, expected platform.CoreServiceState) (supervisorStopFence, error) {
		return platform.PinCoreSupervisor(ctx, expected)
	}, worker: f.NativeWorker.VerifyClaimedSelf, exclude: f.NativeWorker.ExcludeWorker, release: f.NativeWorker.ReleaseWorker}
}

func activationCommand(ctx context.Context, args ...string) error {
	if ctx == nil {
		return coreactivation.ErrConflict
	}
	name, wrapped := hostreqkit.InvokingUserCommand("systemctl", append([]string{"--user"}, args...)...)
	cmd := exec.CommandContext(ctx, name, wrapped...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("root supervisor native command did not complete")
	}
	return nil
}

func (f ActivationEffects) assets() coreactivation.Assets {
	return coreactivation.Assets{Store: f.Store}
}

func (f ActivationEffects) LockAdmission(ctx context.Context, r coreactivation.Record) (func(), error) {
	l, err := f.assets().Layout(r)
	if err != nil {
		return nil, err
	}
	return f.operations().lock(ctx, l.Executable)
}

func (f ActivationEffects) guard(ctx context.Context, r coreactivation.Record, restoring bool) error {
	if runtime.GOOS != "linux" || f.Store == nil || ctx == nil {
		return coreactivation.ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, exists, err := f.Store.Current()
	if err != nil {
		return err
	}
	if !exists || !reflect.DeepEqual(current, r) {
		return coreactivation.ErrConflict
	}
	state, err := f.operations().inspect(ctx)
	if err != nil {
		return err
	}
	overrides, err := coreactivation.NativeOverridesDigest(ctx, state.DropInPaths)
	if err != nil || overrides != r.NativeOverridesDigest {
		return coreactivation.ErrConflict
	}
	if restoring {
		if r.Phase != coreactivation.Restoring || r.Fence != coreactivation.Revoked || !r.RestorationRequired {
			return coreactivation.ErrClaimed
		}
		return f.operations().exclude(ctx, r)
	}
	if !time.Now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return coreactivation.ErrClaimed
	}
	return f.operations().worker(ctx, r)
}

func (f ActivationEffects) validateDefinitions(r coreactivation.Record, sides ...bool) error {
	l, err := f.assets().Layout(r)
	if err != nil {
		return err
	}
	if len(sides) == 0 {
		sides = []bool{false, true}
	}
	for _, candidate := range sides {
		data, err := f.assets().Definition(r, candidate)
		if err != nil {
			return err
		}
		artifact := platform.RenderedArtifact{Target: "linux", Files: []platform.RenderedFile{{Name: l.UnitName, Content: string(data)}}}
		if verdict := f.operations().validate(artifact, platform.ScopeUser); verdict.State != platform.VerdictAccepted {
			return fmt.Errorf("frozen root native definition is not accepted by its validator")
		}
	}
	return nil
}

func (f ActivationEffects) QualifyAdmission(ctx context.Context, r coreactivation.Record) error {
	if runtime.GOOS != "linux" || f.Store == nil {
		return coreactivation.ErrConflict
	}
	if r.CandidateSHA == r.PriorSHA {
		return fmt.Errorf("root update candidate is already the serving image")
	}
	if r.Automatic {
		policy, exists, err := f.Store.Enrollment()
		if err != nil {
			return err
		}
		if !exists || !policy.Enabled {
			return fmt.Errorf("automatic root updates are not enrolled")
		}
	}
	if err := f.validateDefinitions(r); err != nil {
		return err
	}
	state, err := f.observe(ctx, r, false, false)
	if err != nil {
		return err
	}
	incarnation, err := f.operations().incarnation(state.PID)
	if err != nil || state.PID != r.PriorPID || state.InvocationID != r.PriorNativeInvocation || incarnation != r.PriorIncarnation {
		return coreactivation.ErrConflict
	}
	return nil
}

func (f ActivationEffects) PublishCandidate(ctx context.Context, r coreactivation.Record) error {
	l, err := f.assets().Layout(r)
	if err != nil {
		return err
	}
	release, err := f.operations().lock(ctx, l.Executable)
	if err != nil {
		return err
	}
	defer release()
	if err := f.guard(ctx, r, false); err != nil {
		return err
	}
	return f.assets().PublishBinary(ctx, r, false)
}

func (f ActivationEffects) ActivateCandidate(ctx context.Context, r coreactivation.Record) error {
	l, err := f.assets().Layout(r)
	if err != nil {
		return err
	}
	release, err := f.operations().lock(ctx, l.Executable)
	if err != nil {
		return err
	}
	defer release()
	if err := f.guard(ctx, r, false); err != nil {
		return err
	}
	if err := f.validateDefinitions(r, true); err != nil {
		return err
	}
	if err := f.assets().PublishDefinition(ctx, r, false); err != nil {
		return err
	}
	for _, args := range [][]string{{"daemon-reload"}, {"reset-failed", l.UnitName}, {"restart", l.UnitName}} {
		if err := f.guard(ctx, r, false); err != nil {
			return err
		}
		if err := f.assets().VerifyInstalled(ctx, r, true); err != nil {
			return err
		}
		if err := f.operations().command(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

func (f ActivationEffects) VerifyCandidate(ctx context.Context, r coreactivation.Record) error {
	return f.waitVerified(ctx, r, true)
}

func (f ActivationEffects) VerifyPrior(ctx context.Context, r coreactivation.Record) error {
	_, err := f.observe(ctx, r, false, false)
	return err
}

func (f ActivationEffects) RestorePrior(ctx context.Context, r coreactivation.Record) error {
	l, err := f.assets().Layout(r)
	if err != nil {
		return err
	}
	release, err := f.operations().lock(ctx, l.Executable)
	if err != nil {
		return err
	}
	defer release()
	if err := f.guard(ctx, r, true); err != nil {
		return err
	}
	if err := f.validateDefinitions(r, false); err != nil {
		return err
	}
	if err := f.assets().CheckPublicationInputs(ctx, r, true); err != nil {
		return err
	}
	state, err := f.operations().inspect(ctx)
	if err != nil {
		return err
	}
	if state.FragmentPath != l.UnitPath || state.KillMode != "control-group" {
		return coreactivation.ErrConflict
	}
	if state.PID > 0 {
		image, err := f.operations().image(ctx, state.PID)
		if err != nil || image != r.PriorSHA && image != r.CandidateSHA {
			return coreactivation.ErrConflict
		}
	}
	fence, err := f.operations().pin(ctx, state)
	if err != nil {
		return err
	}
	defer fence.Close()
	if err := f.operations().command(ctx, "stop", l.UnitName); err != nil {
		return err
	}
	if err := fence.Stopped(ctx); err != nil {
		return err
	}
	if err := f.guard(ctx, r, true); err != nil {
		return err
	}
	if err := f.assets().PublishBinary(ctx, r, true); err != nil {
		return err
	}
	if err := f.assets().PublishDefinition(ctx, r, true); err != nil {
		return err
	}
	for _, args := range [][]string{{"daemon-reload"}, {"reset-failed", l.UnitName}, {"start", l.UnitName}} {
		if err := f.guard(ctx, r, true); err != nil {
			return err
		}
		if err := f.assets().VerifyInstalled(ctx, r, false); err != nil {
			return err
		}
		if err := f.operations().command(ctx, args...); err != nil {
			return err
		}
	}
	return f.waitVerified(ctx, r, false)
}

func (f ActivationEffects) RetireInputs(ctx context.Context, r coreactivation.Record) error {
	if err := f.operations().release(ctx, r); err != nil {
		return err
	}
	return f.assets().Retire(r)
}

func (f ActivationEffects) ExcludeWorker(ctx context.Context, r coreactivation.Record) error {
	if f.Store == nil {
		return coreactivation.ErrConflict
	}
	current, exists, err := f.Store.Current()
	if err != nil {
		return err
	}
	if !exists || !reflect.DeepEqual(current, r) {
		return coreactivation.ErrConflict
	}
	return f.operations().exclude(ctx, r)
}

func (f ActivationEffects) waitVerified(ctx context.Context, r coreactivation.Record, candidate bool) error {
	for {
		if candidate {
			if err := f.guard(ctx, r, false); err != nil {
				return err
			}
		}
		if _, err := f.observe(ctx, r, candidate, true); err == nil {
			return nil
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("root supervisor verification did not complete: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (f ActivationEffects) observe(ctx context.Context, r coreactivation.Record, candidate, requireRenewal bool) (platform.CoreServiceState, error) {
	l, err := f.assets().Layout(r)
	if err != nil {
		return platform.CoreServiceState{}, err
	}
	if err := f.assets().VerifyInstalled(ctx, r, candidate); err != nil {
		return platform.CoreServiceState{}, err
	}
	state, err := f.operations().inspect(ctx)
	if err != nil {
		return state, err
	}
	if state.LoadState != "loaded" || state.ActiveState != "active" || state.SubState != "running" || state.PendingJob || state.PID <= 0 || state.FragmentPath != l.UnitPath || state.NeedDaemonReload != "no" || state.KillMode != "control-group" || len(state.InvocationID) != 32 {
		return state, fmt.Errorf("root native owner is not a verified active frozen definition")
	}
	overrides, err := coreactivation.NativeOverridesDigest(ctx, state.DropInPaths)
	if err != nil || overrides != r.NativeOverridesDigest {
		return state, coreactivation.ErrConflict
	}
	expectedSHA, expectedBuild := r.PriorSHA, r.PriorBuild
	if candidate {
		expectedSHA, expectedBuild = r.CandidateSHA, r.CandidateBuild
		if state.InvocationID == r.PriorNativeInvocation {
			return state, coreactivation.ErrConflict
		}
	}
	incarnation, err := f.operations().incarnation(state.PID)
	if err != nil {
		return state, err
	}
	image, err := f.operations().image(ctx, state.PID)
	if err != nil || image != expectedSHA {
		return state, coreactivation.ErrConflict
	}
	lease, err := f.operations().lease(ctx, r.Home)
	if err != nil {
		return state, err
	}
	now := time.Now()
	if lease.PID != state.PID || lease.Build != expectedBuild || lease.Incarnation != incarnation || !lease.Deadline.After(now) || lease.Heartbeat.After(now.Add(time.Second)) {
		return state, fmt.Errorf("root supervisor live image and fresh registry owner do not agree")
	}
	if requireRenewal && (!lease.FleetRenewed || lease.Heartbeat.Sub(lease.Started) < 250*time.Millisecond) {
		return state, fmt.Errorf("root supervisor has not completed fresh fleet renewal")
	}
	after, err := f.operations().inspect(ctx)
	if err != nil || after.PID != state.PID || after.InvocationID != state.InvocationID || after.ControlGroup != state.ControlGroup || after.PendingJob {
		return state, coreactivation.ErrConflict
	}
	if digest, err := coreactivation.NativeOverridesDigest(ctx, after.DropInPaths); err != nil || digest != r.NativeOverridesDigest {
		return state, coreactivation.ErrConflict
	}
	if current, err := f.operations().incarnation(state.PID); err != nil || current != incarnation {
		return state, coreactivation.ErrConflict
	}
	if err := f.assets().VerifyInstalled(ctx, r, candidate); err != nil {
		return state, err
	}
	return state, nil
}

func readActivationLease(ctx context.Context, home string) (supervisorLease, error) {
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, ReadOnly: true})
	if err != nil {
		return supervisorLease{}, err
	}
	defer store.Close()
	sessions, err := store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
	if err != nil || len(sessions) != 1 || sessions[0].PID == nil {
		return supervisorLease{}, fmt.Errorf("root supervisor registry owner is ambiguous")
	}
	session := sessions[0]
	var metadata struct {
		Build       string `json:"build_identity"`
		Incarnation string `json:"process_incarnation"`
	}
	if err := json.Unmarshal([]byte(session.MetadataJSON), &metadata); err != nil {
		return supervisorLease{}, err
	}
	lease := supervisorLease{ID: session.SupervisorID, PID: *session.PID, Build: metadata.Build, Incarnation: metadata.Incarnation, Started: session.StartedAt, Heartbeat: session.LastHeartbeatAt, Deadline: session.HeartbeatDeadlineAt, FleetRenewed: true}
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: []string{scenarioruntime.StatusRunning}})
	if err != nil {
		return supervisorLease{}, err
	}
	for _, instance := range instances {
		if strings.TrimSpace(instance.SupervisionPolicy) != "managed" {
			continue
		}
		if instance.SupervisorID != session.SupervisorID || instance.LastHeartbeatAt == nil || !instance.LastHeartbeatAt.After(session.StartedAt) || instance.HeartbeatDeadlineAt == nil || !instance.HeartbeatDeadlineAt.After(time.Now()) {
			lease.FleetRenewed = false
		}
	}
	return lease, nil
}
