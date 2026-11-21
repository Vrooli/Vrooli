// Package sessionlease is the runtime-registry implementation of cli-core's
// EditorLeaseRecorder: the coding-agent launcher binaries register it so
// every session is visible in `vrooli agent list` with its tree, scope and
// claims for as long as the process lives. cli-core cannot import the
// registry (244 modules replace cli-core), so the seam lives there and the
// store here.
package sessionlease

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Recorder adapts the runtime registry to cli-core's seam.
type Recorder struct {
	// Open returns the registry store; tests inject a temporary one.
	Open func(ctx context.Context) (*scenarioruntime.SQLiteStore, error)
	// Owner is what the session about to start belongs to. Nil reads it
	// from this (the launcher's) process: its environment, its terminal and,
	// for a bundled desktop app, the app runtime among its ancestors.
	Owner func() scenarioruntime.EditorLeaseOwner
	// Observe reads exact native process identity; tests inject isolated facts.
	Observe func(int) (cliutil.ProcessBinding, error)
}

// Register installs the recorder as cli-core's default.
func Register() { cliutil.RegisterEditorLeaseRecorder(Recorder{Open: openRegistry}) }

func openRegistry(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
	home, err := config.HomeDir()
	if err != nil {
		return nil, err
	}
	return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
}

func (r Recorder) open(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
	if r.Open == nil {
		return nil, fmt.Errorf("session lease: no registry opener")
	}
	return r.Open(ctx)
}

// launchOwner reads the launcher's own process: the recorder runs inside the
// launcher before it execs or spawns the agent, so its environment is the
// session's environment.
func launchOwner() scenarioruntime.EditorLeaseOwner {
	return agentsession.OwnerAtLaunch(agentsession.EnvironMap(os.Environ()), os.Getpid(), agentsession.NewHostFacts()).LeaseOwner()
}

// Create writes the session's lease with its owner; the boot id makes a
// reboot provable.
func (r Recorder) Create(ctx context.Context, record cliutil.EditorLeaseRecord) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	lease := scenarioruntime.EditorLease{
		SessionID: record.SessionID, Harness: record.Harness, Agent: record.Agent, PID: record.PID,
		WorkingDir: record.WorkingDir, Scope: record.Scope, ContainmentMethod: record.ContainmentMethod, Claims: record.Claims,
	}
	owner := r.Owner
	if owner == nil {
		owner = launchOwner
	}
	lease.Owner = owner()
	if host, err := (hostsession.DefaultProvider{}).Current(ctx, ""); err == nil {
		lease.HostBootID = host.BootID
	}
	_, err = store.CreateEditorLease(ctx, lease, cliutil.EditorLeaseTTL)
	return err
}

// Heartbeat renews the lease deadline.
func (r Recorder) Heartbeat(ctx context.Context, sessionID string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.HeartbeatEditorLease(ctx, sessionID, cliutil.EditorLeaseTTL)
	return err
}

// Stop ends the lease with a reason.
func (r Recorder) Stop(ctx context.Context, sessionID, reason string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.StopEditorLease(ctx, sessionID, reason)
	return err
}

// Overlaps names live sessions whose claims overlap the requested paths.
func (r Recorder) Overlaps(ctx context.Context, paths []string) ([]cliutil.ClaimHolder, error) {
	store, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		return nil, err
	}
	// Only a holder that is provably alive is named; a dead session's claim
	// is not a claim.
	guard := scenarioruntime.StartingLeaseGuard{PIDRunning: platform.IsPIDRunning}
	if host, err := (hostsession.DefaultProvider{}).Current(ctx, ""); err == nil {
		guard.CurrentBootID = host.BootID
	}
	leases = scenarioruntime.LiveEditorLeases(leases, guard)
	now := time.Now().UTC()
	var out []cliutil.ClaimHolder
	for _, overlap := range scenarioruntime.ClaimOverlaps(leases, paths) {
		out = append(out, cliutil.ClaimHolder{
			SessionID: overlap.Holder.SessionID, Agent: overlap.Holder.Agent, PID: overlap.Holder.PID, WorkingDir: overlap.Holder.WorkingDir,
			HeldPath: overlap.HeldPath, Overlap: string(overlap.Overlap), Age: now.Sub(overlap.Holder.CreatedAt),
		})
	}
	return out, nil
}

// BeginBinding records only this new launch attempt. Existing rows never gain
// evidence through reads, migration, owner backfill, or a session refresh.
func (r Recorder) BeginBinding(ctx context.Context, record cliutil.EditorLeaseRecord) (string, error) {
	process, err := r.observe(record.PID)
	if err != nil || process.PID != record.PID || process.PID <= 1 || process.Boot == "" || process.Start == "" {
		return "", fmt.Errorf("%w: %v", cliutil.ErrEditorLeaseBindingUnavailable, err)
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw[:])
	store, err := r.open(ctx)
	if err != nil {
		return "", err
	}
	defer store.Close()
	owner := r.Owner
	if owner == nil {
		owner = launchOwner
	}
	lease := scenarioruntime.EditorLease{
		SessionID: record.SessionID, Harness: record.Harness, Agent: record.Agent,
		PID: record.PID, HostBootID: process.Boot, WorkingDir: record.WorkingDir,
		Scope: record.Scope, Claims: record.Claims, Owner: owner(),
	}
	binding, err := store.CreateEditorLeaseBinding(ctx, lease, scenarioruntime.EditorLeaseProvenance{
		BindingID: id, SessionID: record.SessionID, PID: process.PID, UID: process.UID,
		HostBootID: process.Boot, ProcessStart: process.Start, IntendedScope: record.Scope,
	}, cliutil.EditorLeaseTTL)
	if err != nil {
		return "", err
	}
	return binding.BindingID, nil
}

func (r Recorder) observe(pid int) (cliutil.ProcessBinding, error) {
	if r.Observe != nil {
		return r.Observe(pid)
	}
	return cliutil.ObserveProcessBinding(pid)
}

// RecordContainment accepts only a current exact launcher binding and a native
// successful before-exec observation. Planned names and return-time scopes are
// not proof. Failed evidence writes remain fail-open at the launcher seam.
func (r Recorder) RecordContainment(ctx context.Context, sessionID, bindingID string, evidence cliutil.EditorLeaseContainment) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	binding, err := store.GetEditorLeaseProvenance(ctx, bindingID)
	if err != nil {
		return err
	}
	process, err := r.observe(binding.PID)
	if err != nil || process.PID != binding.PID || process.UID != binding.UID || process.Boot != binding.HostBootID || process.Start != binding.ProcessStart {
		return fmt.Errorf("session lease: launcher incarnation changed or unavailable")
	}
	update := scenarioruntime.EditorLeaseProvenance{}
	switch {
	case evidence.Failure != "" || evidence.Unavailable:
		update.ContainmentState = scenarioruntime.EditorContainmentFailed
		if evidence.Unavailable {
			update.ContainmentState = scenarioruntime.EditorContainmentUnavailable
		}
		update.Failure = evidence.Failure
		if update.Failure == "" {
			update.Failure = "native containment evidence unavailable"
		}
	case !evidence.Session.BeforeExec || !validNativeScopeEvidence(binding, evidence.Session):
		update.ContainmentState = scenarioruntime.EditorContainmentUnavailable
		update.Failure = evidence.Session.ProofFailure
		if update.Failure == "" {
			update.Failure = "native before-exec containment evidence unavailable or does not match launch intent"
		}
	default:
		native := evidence.Session.Process
		again, observeErr := r.observe(native.PID)
		if observeErr != nil || again != native || native.PID <= 1 || native.Start == "" || native.Boot != binding.HostBootID || native.UID != binding.UID {
			update.ContainmentState = scenarioruntime.EditorContainmentUnavailable
			update.Failure = "contained process incarnation changed or unavailable"
		} else {
			update.ContainmentState = scenarioruntime.EditorContainmentEstablished
			update.Scope, update.Method = evidence.Session.Scope, evidence.Session.Method
			update.EvidencePID, update.EvidenceUID = native.PID, native.UID
			update.EvidenceBootID, update.EvidenceProcessStart = native.Boot, native.Start
		}
	}
	_, err = store.RecordEditorLeaseContainment(ctx, sessionID, bindingID, update)
	return err
}

func (r Recorder) HeartbeatBinding(ctx context.Context, sessionID, bindingID string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.HeartbeatEditorLeaseBinding(ctx, sessionID, bindingID, cliutil.EditorLeaseTTL)
	return err
}

func (r Recorder) StopBinding(ctx context.Context, sessionID, bindingID, reason string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.StopEditorLeaseBinding(ctx, sessionID, bindingID, reason)
	return err
}

// A registered container's boolean is not enough: the actual cgroup must match
// the intended native unit, the observed process, and a supported birth-safe
// mechanism. Self placement is safe only for this launcher's own incarnation.
func validNativeScopeEvidence(binding scenarioruntime.EditorLeaseProvenance, session cliutil.ContainedSession) bool {
	ref, err := platform.ParseScopeRef(session.Scope)
	if err != nil || ref.Kind != platform.ScopeKindCgroup || path.Clean(ref.Path) != ref.Path || ref.Path == "/" {
		return false
	}
	intended := strings.TrimSuffix(binding.IntendedScope, ".scope") + ".scope"
	if binding.IntendedScope == "" || path.Base(ref.Path) != intended {
		return false
	}
	member := false
	for _, line := range strings.Split(session.Process.Cgroup, "\n") {
		if line == "0::"+ref.Path {
			member = true
		}
	}
	if !member {
		return false
	}
	switch session.Method {
	case platform.MethodSystemdRun:
		return true
	case platform.MethodTransientUnit, platform.MethodCgroupWrite:
		return session.Process.PID == binding.PID && session.Process.Start == binding.ProcessStart && session.Process.Boot == binding.HostBootID
	default:
		return false
	}
}
