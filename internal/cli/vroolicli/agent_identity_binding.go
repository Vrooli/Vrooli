package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// AgentIdentityBindingProof is the common control-plane/Agent Manager wire contract.
type AgentIdentityBindingProof = cliutil.AgentSessionBindingProof

type identityBindingLeaseReader = scenarioruntime.EditorLeaseLookup

type identityBindingHost interface {
	Process(int) (agentsession.ProcessFact, error)
	StartedAt(int) (time.Time, error)
	ScopeMembers(string) ([]int, error)
}

type identityBindingDeps struct {
	lease      identityBindingLeaseReader
	host       identityBindingHost
	observe    func(int) (cliutil.ProcessBinding, error)
	webConsole func(context.Context, string, string) (agentsession.WebConsoleSession, error)
	now        func() time.Time
	scopePath  func(string) (string, error)
}

func (app *App) runAgentIdentityBinding(ctx *AppContext, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent identity-binding")
	fs.SetOutput(ctx.Stderr)
	runPID := fs.Int("run-pid", 0, "expected persisted run PID; enables exact attached-run lease correlation")
	runID := fs.String("run-id", "", "optional exact Agent Manager run context; required when proving a renewal")
	harnessSessionID := fs.String("harness-session-id", "", "exact host launcher session whose editor lease is checked")
	webConsoleSessionID := fs.String("web-console-session-id", "", "optional expected Web Console session; otherwise derived from the verified launch owner")
	harnessKind := fs.String("harness-kind", "", "exact harness kind from the attached run")
	jsonOut := fs.Bool("json", false, "emit the versioned proof JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent identity-binding", "unexpected positional arguments")
	}
	if !(*jsonOut || ctx.Globals.JSON) {
		return clipolicy.UsageErrorf("agent identity-binding", "--json is required")
	}
	if strings.TrimSpace(*harnessSessionID) == "" || strings.TrimSpace(*harnessKind) == "" {
		return clipolicy.UsageErrorf("agent identity-binding", "--harness-session-id and --harness-kind are required")
	}
	home, err := configpkg.HomeDir()
	if err != nil {
		return errors.New("identity binding proof unavailable: runtime_home")
	}
	proofCtx, cancel := context.WithTimeout(ctx.OperationContext(), 2*time.Second)
	defer cancel()
	store, err := scenarioruntime.NewSQLiteStore(proofCtx, scenarioruntime.Config{HomeDir: home, ReadOnly: true})
	if err != nil {
		return errors.New("identity binding proof unavailable: runtime_registry")
	}
	defer store.Close()
	host := agentsession.NewHostFacts()
	probes := agentsession.LiveProbes(home)
	deps := identityBindingDeps{
		lease: store, host: host, observe: cliutil.ObserveProcessBinding,
		webConsole: probes.WebConsoleSession, now: time.Now, scopePath: agentsession.ScopeCgroupPath,
	}
	var proof AgentIdentityBindingProof
	if *runPID != 0 {
		proof, err = proveAgentRunBinding(proofCtx, deps, strings.TrimSpace(*runID), strings.TrimSpace(*harnessSessionID), strings.TrimSpace(*webConsoleSessionID), strings.TrimSpace(*harnessKind), *runPID)
	} else {
		proof, err = proveAgentIdentityBinding(proofCtx, deps, strings.TrimSpace(*runID), strings.TrimSpace(*harnessSessionID), strings.TrimSpace(*webConsoleSessionID), strings.TrimSpace(*harnessKind))
	}
	if err != nil {
		return fmt.Errorf("identity binding proof unavailable: %w", err)
	}
	if err := identityBindingContext(proofCtx); err != nil {
		return fmt.Errorf("identity binding proof unavailable: %w", err)
	}
	return cliout.WriteJSON(ctx.Stdout, proof)
}

func proveAgentIdentityBinding(ctx context.Context, deps identityBindingDeps, runID, harnessSessionID, expectedWebConsoleSessionID, harnessKind string) (AgentIdentityBindingProof, error) {
	if deps.lease == nil || deps.host == nil || deps.observe == nil || deps.webConsole == nil || deps.now == nil || deps.scopePath == nil {
		return AgentIdentityBindingProof{}, errors.New("proof_reader_unavailable")
	}
	if harnessSessionID == "" || harnessKind == "" {
		return AgentIdentityBindingProof{}, errors.New("selector_missing")
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	first, err := deps.lease.GetEditorLease(ctx, harnessSessionID)
	if err != nil {
		return AgentIdentityBindingProof{}, errors.New("lease_unavailable")
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	webConsoleSessionID, ownerKind, err := validateIdentityBindingLease(first, harnessSessionID, expectedWebConsoleSessionID, harnessKind, deps.now())
	if err != nil {
		return AgentIdentityBindingProof{}, err
	}
	firstObservation, err := observeIdentityBinding(ctx, deps, first)
	if err != nil {
		return AgentIdentityBindingProof{}, err
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	if webConsoleSessionID != "" {
		owner, _ := agentsession.OwnerFromLease(first.Owner)
		session, err := deps.webConsole(ctx, owner.Evidence.WebConsoleStateRoot, webConsoleSessionID)
		// A transient owner read under concurrent startup load can fail even
		// while the exact session is alive. Retry one bounded read, but never
		// turn a confirmed absence or a mismatched identity into a proof.
		if err != nil && !errors.Is(err, agentsession.ErrNotFound) && ctx.Err() == nil {
			session, err = deps.webConsole(ctx, owner.Evidence.WebConsoleStateRoot, webConsoleSessionID)
		}
		if err != nil || session.ID != webConsoleSessionID {
			return AgentIdentityBindingProof{}, errors.New("web_console_session_unverified")
		}
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	second, err := deps.lease.GetEditorLease(ctx, harnessSessionID)
	if err != nil || !sameIdentityBindingLease(first, second) {
		return AgentIdentityBindingProof{}, errors.New("lease_changed_during_proof")
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	secondWebConsoleSessionID, secondOwnerKind, err := validateIdentityBindingLease(second, harnessSessionID, expectedWebConsoleSessionID, harnessKind, deps.now())
	if err != nil || secondWebConsoleSessionID != webConsoleSessionID || secondOwnerKind != ownerKind {
		if err != nil {
			return AgentIdentityBindingProof{}, err
		}
		return AgentIdentityBindingProof{}, errors.New("lease_changed_during_proof")
	}
	secondObservation, err := observeIdentityBinding(ctx, deps, second)
	if err != nil || firstObservation != secondObservation {
		return AgentIdentityBindingProof{}, errors.New("process_changed_during_proof")
	}
	if err := identityBindingContext(ctx); err != nil {
		return AgentIdentityBindingProof{}, err
	}
	return AgentIdentityBindingProof{
		ProofVersion: cliutil.AgentSessionBindingProofVersion, RunID: runID, HarnessSessionID: harnessSessionID, LeaseSessionID: harnessSessionID,
		WebConsoleSessionID: webConsoleSessionID, HarnessKind: harnessKind,
		OwnerKind: ownerKind, Binding: secondObservation.Process,
	}, nil
}

func validateIdentityBindingLease(lease scenarioruntime.EditorLease, harnessSessionID, expectedWebConsoleSessionID, harnessKind string, now time.Time) (string, string, error) {
	if lease.SessionID != harnessSessionID || lease.Harness != harnessKind || lease.Status != scenarioruntime.EditorLeaseActive || lease.PID <= 1 || lease.HostBootID == "" || lease.Scope == "" {
		return "", "", errors.New("lease_identity_mismatch")
	}
	owner, ok := agentsession.OwnerFromLease(lease.Owner)
	ownerKind := "unknown"
	if ok {
		ownerKind = owner.Kind
	}
	webConsoleSessionID := ""
	if ownerKind == agentsession.KindWebConsoleSession {
		if !ok || owner.Source != agentsession.SourceLaunch || owner.Evidence.ReadFromPID != lease.PID {
			return "", "", errors.New("launch_owner_unverified")
		}
		webConsoleSessionID = owner.Ref
		if webConsoleSessionID == "" || owner.Evidence.WebConsoleSessionID != webConsoleSessionID || owner.Evidence.WebConsoleStateRoot == "" {
			return "", "", errors.New("launch_owner_unverified")
		}
	}
	if expectedWebConsoleSessionID != "" && expectedWebConsoleSessionID != webConsoleSessionID {
		return "", "", errors.New("lease_identity_mismatch")
	}
	if lease.CreatedAt.IsZero() || lease.CreatedAt.After(now.UTC()) {
		return "", "", errors.New("lease_time_unverified")
	}
	return webConsoleSessionID, ownerKind, nil
}

// identityBindingObservation is what one kernel read of a lease established.
// Process is the lease's own process: the incarnation a proof binds and the one
// a grant is later matched against. ScopeRoot is the process that actually sits
// in the session scope. They are the same process when the launcher placed
// itself in the scope and exec'd the agent. When the launcher stays alive to
// wait for the agent (spawn-and-wait, which every `vrooli agent launch` takes
// because its operation context is cancellable), the launcher keeps its own
// cgroup and the agent is its direct child inside the scope.
type identityBindingObservation struct{ Process, ScopeRoot cliutil.ProcessBinding }

func observeIdentityBinding(ctx context.Context, deps identityBindingDeps, lease scenarioruntime.EditorLease) (identityBindingObservation, error) {
	if err := identityBindingContext(ctx); err != nil {
		return identityBindingObservation{}, err
	}
	unit := (agentsession.Session{Scope: lease.Scope}).ScopeUnit()
	if unit == "" {
		return identityBindingObservation{}, errors.New("session_scope_unverified")
	}
	expectedCgroup, err := deps.scopePath(unit)
	if err != nil || expectedCgroup == "" || !strings.HasPrefix(expectedCgroup, "/") {
		return identityBindingObservation{}, errors.New("session_scope_unverified")
	}
	if err := identityBindingContext(ctx); err != nil {
		return identityBindingObservation{}, err
	}
	members, err := deps.host.ScopeMembers(unit)
	if err != nil {
		return identityBindingObservation{}, errors.New("scope_membership_unverified")
	}
	rootPID, ok := identityBindingScopeRoot(deps.host, members, lease.PID)
	if !ok {
		return identityBindingObservation{}, errors.New("scope_membership_unverified")
	}
	if err := identityBindingContext(ctx); err != nil {
		return identityBindingObservation{}, err
	}
	root, err := observeIdentityBindingProcess(deps, rootPID, lease.HostBootID)
	if err != nil || root.Cgroup != "0::"+expectedCgroup {
		return identityBindingObservation{}, errors.New("kernel_binding_unverified")
	}
	process := root
	if rootPID != lease.PID {
		// The lease process is the launcher outside the scope. It must be the
		// exact incarnation the lease recorded, not merely a PID whose child
		// happens to be in the scope.
		process, err = observeIdentityBindingProcess(deps, lease.PID, lease.HostBootID)
		if err != nil {
			return identityBindingObservation{}, errors.New("kernel_binding_unverified")
		}
	}
	if err := identityBindingContext(ctx); err != nil {
		return identityBindingObservation{}, err
	}
	for _, observed := range []cliutil.ProcessBinding{root, process} {
		fact, err := deps.host.Process(observed.PID)
		if err != nil || fact.PID != observed.PID || fact.StartTime != observed.Start {
			return identityBindingObservation{}, errors.New("process_incarnation_unverified")
		}
	}
	startedAt, err := deps.host.StartedAt(lease.PID)
	if err != nil || startedAt.IsZero() || startedAt.After(lease.CreatedAt) {
		return identityBindingObservation{}, errors.New("process_started_after_lease")
	}
	if err := identityBindingContext(ctx); err != nil {
		return identityBindingObservation{}, err
	}
	return identityBindingObservation{Process: process, ScopeRoot: root}, nil
}

// identityBindingScopeRoot names the scope member the lease process owns: the
// lease process itself when it is a member, otherwise its one direct child that
// is. The kernel parent link is what ties the child to this exact launcher: a
// launcher that exits reparents its child, so the link cannot outlive the
// incarnation the lease recorded. Zero or several candidates fail closed.
func identityBindingScopeRoot(host identityBindingHost, members []int, leasePID int) (int, bool) {
	if containsPID(members, leasePID) {
		return leasePID, true
	}
	root := 0
	for _, member := range members {
		fact, err := host.Process(member)
		if err != nil || fact.PID != member || fact.PPID != leasePID {
			continue
		}
		if root != 0 {
			return 0, false
		}
		root = member
	}
	return root, root != 0
}

// observeIdentityBindingProcess reads one process's kernel binding and requires
// it to be complete and born in the lease's boot.
func observeIdentityBindingProcess(deps identityBindingDeps, pid int, leaseBootID string) (cliutil.ProcessBinding, error) {
	binding, err := deps.observe(pid)
	if err != nil || binding.PID != pid || binding.UID < 0 || binding.Start == "" || binding.Boot == "" || binding.Boot != leaseBootID {
		return cliutil.ProcessBinding{}, errors.New("kernel_binding_unverified")
	}
	return binding, nil
}

func identityBindingContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return errors.New("proof_cancelled")
	}
	return nil
}

func sameIdentityBindingLease(a, b scenarioruntime.EditorLease) bool {
	return scenarioruntime.EditorLeaseBindingEqual(a, b)
}

func containsPID(pids []int, want int) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}

// proveAgentRunBinding correlates the immutable attached anchor and expected
// run process with its host lease. It reads only two exact keys; an unavailable
// registry or a conflicting canonical lease never falls back to another key.
func proveAgentRunBinding(ctx context.Context, deps identityBindingDeps, runID, harnessSessionID, webConsoleSessionID, harnessKind string, runPID int) (AgentIdentityBindingProof, error) {
	if runID == "" || runPID <= 1 || harnessSessionID == "" || deps.lease == nil {
		return AgentIdentityBindingProof{}, errors.New("run_binding_missing")
	}
	lease, err := scenarioruntime.ResolveAttachedEditorLease(ctx, deps.lease, runID, harnessSessionID)
	if err != nil {
		if contextErr := identityBindingContext(ctx); contextErr != nil {
			return AgentIdentityBindingProof{}, contextErr
		}
		if errors.Is(err, scenarioruntime.ErrEditorLeaseIdentityMismatch) {
			return AgentIdentityBindingProof{}, errors.New("lease_identity_mismatch")
		}
		return AgentIdentityBindingProof{}, errors.New("lease_unavailable")
	}
	proof, err := proveAgentIdentityBinding(ctx, deps, runID, lease.SessionID, webConsoleSessionID, harnessKind)
	if err != nil {
		return AgentIdentityBindingProof{}, err
	}
	if proof.Binding.PID != runPID {
		return AgentIdentityBindingProof{}, errors.New("run_process_mismatch")
	}
	// The nested proof rereads its selected key, but a canonical lease can
	// appear while a legacy-key proof runs. Re-resolve both selectors before
	// accepting the proof, and preserve the original selected binding exactly.
	finalLease, err := scenarioruntime.ResolveAttachedEditorLease(ctx, deps.lease, runID, harnessSessionID)
	if contextErr := identityBindingContext(ctx); contextErr != nil {
		return AgentIdentityBindingProof{}, contextErr
	}
	if err != nil || !scenarioruntime.EditorLeaseBindingEqual(lease, finalLease) {
		return AgentIdentityBindingProof{}, errors.New("lease_changed_during_proof")
	}
	proof.HarnessSessionID = harnessSessionID
	return proof, nil
}
