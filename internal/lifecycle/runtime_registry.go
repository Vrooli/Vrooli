package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioenv"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	runtimeRegistryInstanceEnv   = "VROOLI_RUNTIME_INSTANCE_ID"
	runtimeRegistryGenerationEnv = "VROOLI_RUNTIME_GENERATION"
)

type scenarioRuntimeStore interface {
	scenarioruntime.LifecycleRepository
	scenarioruntime.PortClaimRepository
	scenarioruntime.CleanupRepository
	scenarioruntime.HealthRepository
	scenarioruntime.ProcessRefRepository
	scenarioruntime.EventRepository
	scenarioruntime.StartOperationRepository
	Close() error
}

type runtimeRegistrySession struct {
	enabled  bool
	store    scenarioRuntimeStore
	instance scenarioruntime.Instance
	claims   map[string]scenarioruntime.PortClaim
}

type runtimeRegistryStopSession struct {
	enabled   bool
	store     scenarioRuntimeStore
	instances []scenarioruntime.Instance
}

func disabledRuntimeRegistrySession() runtimeRegistrySession {
	return runtimeRegistrySession{}
}

func (r *Runner) beginRuntimeRegistryStart(ctx context.Context, item scenario.Scenario) (runtimeRegistrySession, error) {
	deps := r.runtimeDeps()
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		return runtimeRegistrySession{}, err
	}
	host, err := deps.hostSession(ctx, r.Home)
	if err != nil {
		_ = store.Close()
		return runtimeRegistrySession{}, fmt.Errorf("resolve host session: %w", err)
	}
	sourceDir, err := r.effectiveSourceDir(item)
	if err != nil {
		_ = store.Close()
		return runtimeRegistrySession{}, err
	}
	ownerPID := os.Getpid()
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario:      item.Slug,
		Variant:       item.Variant,
		Status:        scenarioruntime.StatusStarting,
		Phase:         "planning",
		ScopePath:     r.Root,
		OwnerKind:     scenarioruntime.OwnerKindLifecycle,
		OwnerPID:      &ownerPID,
		WorkingDir:    sourceDir,
		HostBootID:    host.BootID,
		HostSessionID: host.SessionID,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		_ = store.Close()
		return runtimeRegistrySession{}, fmt.Errorf("create scenario runtime lease: %w", err)
	}
	return runtimeRegistrySession{
		enabled:  true,
		store:    store,
		instance: instance,
		claims:   map[string]scenarioruntime.PortClaim{},
	}, nil
}

func (r *Runner) beginRuntimeRegistryStop(ctx context.Context, scenarioName, variant string) (runtimeRegistryStopSession, error) {
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return runtimeRegistryStopSession{}, err
	}
	// Scope by variant so stopping one instance never marks/releases a sibling's
	// rows. A normalized variant (never empty) restricts the filter to exactly
	// this instance; an empty variant would match all variants (the dangerous
	// reap-sibling behavior this fixes), so callers pass the resolved variant.
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: scenarioName,
		Variant:  scenarioruntime.InstanceKey{Scenario: scenarioName, Variant: variant}.Normalize().Variant,
		Statuses: scenarioruntime.StopCandidateInstanceStatuses(),
	})
	if err != nil {
		_ = store.Close()
		return runtimeRegistryStopSession{}, fmt.Errorf("list scenario runtime leases for stop: %w", err)
	}
	for _, instance := range instances {
		if instance.Status == scenarioruntime.StatusStarting || instance.Status == scenarioruntime.StatusRunning {
			if _, err := store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, scenarioruntime.StatusStopping, instance.Phase); err != nil {
				_ = store.Close()
				return runtimeRegistryStopSession{}, fmt.Errorf("mark scenario runtime stopping: %w", err)
			}
		}
	}
	return runtimeRegistryStopSession{enabled: true, store: store, instances: instances}, nil
}

func (s runtimeRegistrySession) close() error {
	if !s.enabled || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *runtimeRegistrySession) setPhase(ctx context.Context, phase string) error {
	if !s.enabled {
		return nil
	}
	updated, err := s.store.UpdateInstanceStatus(ctx, s.instance.InstanceID, s.instance.Generation, scenarioruntime.StatusStarting, phase)
	if err != nil {
		return fmt.Errorf("update scenario runtime phase: %w", err)
	}
	s.instance = updated
	return nil
}

// leaseRenewalWarning builds the pump's error sink for one phase. Renewal
// failures are surfaced rather than swallowed: they are the early signal that
// something reaped the lease out from under a live start.
func (r *Runner) leaseRenewalWarning(item scenario.Scenario, phase string) func(error) {
	return func(err error) {
		r.logWarn("Failed to renew scenario runtime lease during phase",
			logx.AttrScenario, item.Slug,
			logx.AttrPhase, phase,
			logx.AttrError, err,
		)
	}
}

// leaseHeartbeatInterval renews well inside the TTL so a single slow or failed
// renewal never lets the lease lapse. Overridden in tests.
var leaseHeartbeatInterval = scenarioruntime.DefaultHeartbeatTTL / 3

// keepLeaseAlive runs fn while renewing the runtime lease on a ticker.
//
// Phases are the long pole of a start: a cold setup builds the whole UI and can
// run for minutes, and nothing inside ExecutePhaseDetailed touches the registry.
// Without this pump the lease sits at its creation deadline for the entire
// phase, so it is already past due the moment the phase ends — which both makes
// the lease a lie about liveness and hands any concurrent `--clean-stale` sweep
// a live start to reap.
//
// The returned pump is stopped AND joined before keepLeaseAlive returns, so the
// caller regains exclusive access to the session; that serialization is what
// makes it safe for the pump to write s.instance without a lock.
func (s *runtimeRegistrySession) keepLeaseAlive(ctx context.Context, onError func(error), fn func() error) error {
	if !s.enabled {
		return fn()
	}
	interval := leaseHeartbeatInterval
	if interval <= 0 {
		interval = time.Second
	}
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// A failed renewal is reported but never aborts the phase: the
				// post-phase heartbeat is the authoritative gate, and a
				// transient store error should not discard minutes of build.
				if err := s.heartbeat(ctx); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()
	err := fn()
	close(done)
	<-exited
	return err
}

func (s *runtimeRegistrySession) heartbeat(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	updated, err := s.store.HeartbeatLease(ctx, s.instance.InstanceID, s.instance.Generation, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		return fmt.Errorf("heartbeat scenario runtime lease: %w", err)
	}
	s.instance = updated
	// Reserved port claims expire on their own TTL; renew them with the lease
	// so a slow start (e.g. a long setup build) keeps its ports instead of
	// having them expired-and-stolen by a concurrent allocation.
	if _, err := s.store.RenewReservedPortClaimsForInstance(ctx, s.instance.InstanceID, time.Now().UTC().Add(scenarioruntime.DefaultReservedClaimTTL)); err != nil {
		return fmt.Errorf("renew scenario runtime port reservations: %w", err)
	}
	return nil
}

func (s runtimeRegistrySession) portClaimOptions(ctx context.Context) ports.RuntimeClaimOptions {
	if !s.enabled {
		return ports.RuntimeClaimOptions{}
	}
	return ports.RuntimeClaimOptions{
		Enabled:    true,
		Context:    ctx,
		Store:      s.store,
		InstanceID: s.instance.InstanceID,
	}
}

func (s *runtimeRegistrySession) adoptOrReservePorts(ctx context.Context, item scenario.Scenario, env ports.Environment) error {
	if !s.enabled {
		return nil
	}
	if len(env.RuntimeClaims) > 0 {
		for name, claim := range env.RuntimeClaims {
			s.claims[name] = claim
		}
	}
	summaries := item.Manifest.SortedPorts()
	for _, summary := range summaries {
		if _, ok := s.claims[summary.Name]; ok {
			continue
		}
		port, ok := env.AllocatedPorts[summary.Name]
		if !ok || port <= 0 {
			continue
		}
		claim, err := s.store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			InstanceID: s.instance.InstanceID,
			Scenario:   item.Slug,
			Variant:    item.Variant,
			PortName:   summary.Name,
			EnvVar:     summary.EnvVar,
			Port:       port,
			BindHost:   "127.0.0.1",
			URL:        runtimePortURL(summary.Name, port),
			Status:     scenarioruntime.ClaimStatusReserved,
		})
		if err != nil {
			return fmt.Errorf("reserve scenario runtime port claim %s=%d: %w", summary.Name, port, err)
		}
		s.claims[summary.Name] = claim
	}
	return nil
}

func (s runtimeRegistrySession) injectEnv(env map[string]string) {
	if !s.enabled || env == nil {
		return
	}
	env[runtimeRegistryInstanceEnv] = s.instance.InstanceID
	env[runtimeRegistryGenerationEnv] = strconv.FormatInt(s.instance.Generation, 10)
}

func (s runtimeRegistrySession) recordHealth(ctx context.Context, item scenario.Scenario, env ports.Environment, healthStatus string) error {
	if !s.enabled {
		return nil
	}
	snapshot := scenarioruntime.HealthProbe{}.Probe(ctx, scenarioruntime.HealthProbeInput{
		InstanceID:   s.instance.InstanceID,
		Scenario:     item.Slug,
		HealthConfig: item.Manifest.HealthConfig(),
		Ports:        healthPortsFromEnv(item.Manifest, env.EnvVars),
	})
	lifecycleStatus := runtimeHealthStatus(healthStatus)
	if lifecycleStatus != scenarioruntime.HealthStatusUnknown && shouldPreferLifecycleHealthStatus(snapshot) {
		readiness := healthStatus == WaitVerdictHealthy || healthStatus == scenarioruntime.HealthStatusDegraded || healthStatus == WaitVerdictRunning
		snapshot.Status = lifecycleStatus
		snapshot.Readiness = &readiness
	}
	if _, err := s.store.UpsertHealthSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("write scenario runtime health snapshot: %w", err)
	}
	return nil
}

func (s *runtimeRegistrySession) bindPorts(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	for name, claim := range s.claims {
		bound, err := s.store.BindPortClaim(ctx, claim.ClaimID)
		if err != nil {
			if errors.Is(err, scenarioruntime.ErrClaimNotReservable) {
				alreadyBound, lookupErr := s.boundClaim(ctx, claim.ClaimID)
				if lookupErr != nil {
					return fmt.Errorf("bind scenario runtime port claim %s=%d: %w (inspect claim: %v)", name, claim.Port, err, lookupErr)
				}
				if alreadyBound.Status == scenarioruntime.ClaimStatusBound {
					s.claims[name] = alreadyBound
					continue
				}
			}
			return fmt.Errorf("bind scenario runtime port claim %s=%d: %w", name, claim.Port, err)
		}
		s.claims[name] = bound
	}
	return nil
}

func (s *runtimeRegistrySession) boundClaim(ctx context.Context, claimID string) (scenarioruntime.PortClaim, error) {
	claims, err := s.store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: s.instance.InstanceID})
	if err != nil {
		return scenarioruntime.PortClaim{}, err
	}
	for _, claim := range claims {
		if claim.ClaimID == claimID {
			return claim, nil
		}
	}
	return scenarioruntime.PortClaim{}, scenarioruntime.ErrNotFound
}

func (s *runtimeRegistrySession) markRunning(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	updated, err := s.store.UpdateInstanceStatus(ctx, s.instance.InstanceID, s.instance.Generation, scenarioruntime.StatusRunning, "develop")
	if err != nil {
		return fmt.Errorf("mark scenario runtime running: %w", err)
	}
	s.instance = updated
	return nil
}

func (s runtimeRegistrySession) publishPeerRecord(ctx context.Context, home string) error {
	if !s.enabled {
		return nil
	}
	refs, err := s.store.ListProcessRefs(ctx, s.instance.InstanceID)
	if err != nil {
		return fmt.Errorf("read peer owner process: %w", err)
	}
	ownerPID := 0
	for _, ref := range refs {
		if ref.Status == WaitVerdictRunning && ref.PID != nil && *ref.PID > 0 {
			ownerPID = *ref.PID
			break
		}
	}
	if ownerPID == 0 {
		// A scenario with no durable process has nothing discoverable to publish.
		// This is valid for lifecycle-only helpers and avoids advertising the
		// short-lived control-plane command as the peer owner.
		return nil
	}
	ports := make(map[string]int, len(s.claims))
	for name, claim := range s.claims {
		if claim.Status == scenarioruntime.ClaimStatusBound {
			ports[name] = claim.Port
		}
	}
	return scenarioenv.Write(home, scenarioenv.PeerRecord{
		Scenario:  s.instance.Scenario,
		Instance:  s.instance.Variant,
		Tier:      1,
		OwnerPID:  ownerPID,
		StartedAt: s.instance.StartedAt,
		Ports:     ports,
	})
}

// attachSupervision hands the freshly-started instance to the runtime
// supervisor before this command returns.
//
// Without it the start ends with the instance owned by THIS process, which is
// about to exit: the lease then names a dead owner and nothing renews it, so
// the scenario reads as stopped roughly 30 seconds later and stays that way
// until some future supervisor tick adopts it. Every consumer that asks in
// between is told a healthy, listening scenario is down, and `--auto-start`
// consumers respond by rebuilding it. Closing the handover here removes the
// window rather than narrowing it.
//
// Failing to attach is not a failed start: the scenario is up either way. It
// degrades to the previous behavior, loudly, so the gap is visible in the log
// instead of showing up later as a mystery restart.
func (r *Runner) attachSupervision(ctx context.Context, session *runtimeRegistrySession, item scenario.Scenario) {
	if !session.enabled {
		return
	}
	// With the supervisor disabled by policy there is nothing to hand off to,
	// and waiting to discover that would tax every start.
	if runtimesupervisor.ModeFromEnv() == runtimesupervisor.ModeOff {
		return
	}
	attached := false
	err := Await(r.awaitClock(), supervisionAttachPolicy, func() (bool, error) {
		instance, ok, err := session.store.AttachLiveSupervision(ctx,
			session.instance.InstanceID, session.instance.Generation, scenarioruntime.DefaultSupervisedLeaseTTL)
		if err != nil {
			// A transient store error should not strand ownership; keep
			// trying until the policy's deadline.
			r.logDebug("Supervision attach attempt failed",
				logx.AttrScenario, item.Slug, logx.AttrError, err)
			return false, nil
		}
		if !ok {
			return false, nil
		}
		session.instance = instance
		attached = true
		return true, nil
	})
	if attached {
		r.logDebug("Handed runtime ownership to the supervisor",
			logx.AttrScenario, item.Slug,
			"supervisor_id", session.instance.SupervisorID,
		)
		return
	}
	r.logWarn("No live runtime supervisor to take ownership; this instance keeps lifecycle ownership and its lease will lapse when this command exits",
		logx.AttrScenario, item.Slug,
		logx.AttrOperation, "attach_supervision",
		logx.AttrError, err,
	)
}

func (r *Runner) ensureRuntimeSupervisor(ctx context.Context, session runtimeRegistrySession) error {
	if !session.enabled {
		return nil
	}
	mode := runtimesupervisor.ModeFromEnv()
	if mode == runtimesupervisor.ModeOff {
		return nil
	}
	deps := r.runtimeDeps()
	// Both writers are nil on purpose: the supervisor outlives this CLI, so
	// wiring it to our streams either discards its output or points it at a
	// descriptor that dies with us. EnsureRunning routes it to the supervisor's
	// own log file instead.
	err := deps.ensureRuntimeSupervisor(ctx, r.Home, nil, nil)
	if err == nil {
		return nil
	}
	if mode == runtimesupervisor.ModeAuto {
		r.logWarn("Runtime supervisor auto-start failed; scenario remains registry-managed but awaits adoption",
			logx.AttrScenario, session.instance.Scenario,
			logx.AttrOperation, "ensure_runtime_supervisor",
			"error", err.Error(),
		)
		return nil
	}
	return fmt.Errorf("ensure runtime supervisor: %w", err)
}

func (s *runtimeRegistrySession) fail(ctx context.Context, cause error) error {
	if !s.enabled || s.store == nil {
		return nil
	}
	if _, err := s.store.ReleaseActivePortClaimsForInstance(ctx, s.instance.InstanceID); err != nil {
		return fmt.Errorf("release scenario runtime claims after failure: %w", err)
	}
	if _, err := s.store.UpdateInstanceStatus(ctx, s.instance.InstanceID, s.instance.Generation, scenarioruntime.StatusFailed, s.instance.Phase); err != nil {
		return fmt.Errorf("mark scenario runtime failed: %w", err)
	}
	details, _ := json.Marshal(map[string]string{"error": cause.Error()})
	if _, err := s.store.RecordEvent(ctx, scenarioruntime.Event{
		InstanceID:  s.instance.InstanceID,
		Scenario:    s.instance.Scenario,
		EventType:   "start_failed",
		DetailsJSON: string(details),
	}); err != nil {
		return fmt.Errorf("record scenario runtime failure event: %w", err)
	}
	return nil
}

func (s runtimeRegistryStopSession) close() error {
	if !s.enabled || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s runtimeRegistryStopSession) finish(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	for _, instance := range s.instances {
		if _, err := s.store.ReleaseActivePortClaimsForInstance(ctx, instance.InstanceID); err != nil {
			return fmt.Errorf("release scenario runtime claims during stop: %w", err)
		}
		if _, err := s.store.StopLease(ctx, instance.InstanceID, instance.Generation, "scenario stopped"); err != nil {
			return fmt.Errorf("stop scenario runtime lease: %w", err)
		}
	}
	return nil
}

func recordRuntimeProcessRef(ctx context.Context, deps lifecycleDeps, home string, env map[string]string, record process.Record) error {
	instanceID := strings.TrimSpace(env[runtimeRegistryInstanceEnv])
	if instanceID == "" {
		return nil
	}
	store, err := deps.runtimeRegistry(ctx, home)
	if err != nil {
		return err
	}
	defer store.Close()
	host, err := deps.hostSession(ctx, home)
	if err != nil {
		return fmt.Errorf("resolve host session for process ref: %w", err)
	}
	pid := record.PID
	pgid := record.PGID
	_, err = store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		InstanceID: instanceID,
		PID:        &pid,
		PGID:       &pgid,
		ProcessID:  record.ProcessID,
		Step:       record.Step,
		Command:    record.Command,
		LogFile:    record.LogFile,
		Status:     record.Status,
		StartedAt:  record.StartedAt,
		HostBootID: host.BootID,
	})
	if err != nil {
		return fmt.Errorf("write scenario runtime process ref: %w", err)
	}
	return nil
}

func runtimePortURL(portName string, port int) string {
	return scenarioruntime.LocalPortURL(portName, port)
}

func runtimeHealthStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "healthy":
		return scenarioruntime.HealthStatusHealthy
	case scenarioruntime.HealthStatusDegraded:
		return scenarioruntime.HealthStatusDegraded
	case "unhealthy":
		return scenarioruntime.HealthStatusUnhealthy
	case "running":
		return scenarioruntime.HealthStatusNotConfigured
	default:
		return scenarioruntime.HealthStatusUnknown
	}
}

// registryRuntimeView is the lifecycle-facing view of registry-authoritative
// runtime state. It captures everything the start/dependency reuse decisions
// need: whether there is an authoritative instance, the ports it claims, and
// the recorded health snapshot. PID visibility is intentionally not part of
// this view — registry lease freshness and listener evidence are the
// authority, not process records.
type registryRuntimeView struct {
	Present       bool
	Authoritative bool
	Instance      scenarioruntime.Instance
	Claims        []scenarioruntime.PortClaim
	Ports         map[string]int
	HealthStatus  string
}

// lookupRegistryRuntime returns the authoritative registry runtime for a
// scenario, or a zero view if no authoritative instance exists. The caller
// can use the result to decide whether to reuse, restart, or start fresh —
// without consulting legacy process records.
func (r *Runner) lookupRegistryRuntime(ctx context.Context, item scenario.Scenario) (registryRuntimeView, error) {
	deps := r.runtimeDeps()
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		return registryRuntimeView{}, err
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: item.Slug,
		Variant:  scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize().Variant,
		Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return registryRuntimeView{}, err
	}
	if len(instances) == 0 {
		return registryRuntimeView{}, nil
	}

	var latest scenarioruntime.Instance
	for _, instance := range instances {
		if latest.InstanceID == "" || isNewerInstance(instance, latest) {
			latest = instance
		}
	}

	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		InstanceID: latest.InstanceID,
		Statuses:   scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		return registryRuntimeView{}, err
	}
	refs, err := store.ListProcessRefs(ctx, latest.InstanceID)
	if err != nil {
		return registryRuntimeView{}, err
	}
	health, err := store.GetHealthSnapshot(ctx, latest.InstanceID)
	if err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		return registryRuntimeView{}, err
	}

	host, err := deps.hostSession(ctx, r.Home)
	if err != nil {
		return registryRuntimeView{}, fmt.Errorf("resolve host session: %w", err)
	}
	// Only Known/Listening is read below, so skip attribution and the
	// subprocess it costs.
	snapshot := network.CaptureTCPListenerPorts()
	reconciled := scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
		Now:           time.Now().UTC(),
		CurrentBootID: host.BootID,
		Instance:      latest,
		Claims:        claims,
		ProcessRefs:   refs,
		Processes:     scenarioruntime.ProcessEvidenceFromRefs(refs, process.IsPIDRunning),
		Listeners: scenarioruntime.ListenerEvidenceFromClaims(claims, refs, func(port int) scenarioruntime.ListenerEvidence {
			state := snapshot.Listening(port)
			if !state.Known {
				return scenarioruntime.ListenerEvidence{Known: false}
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
		}),
	})

	view := registryRuntimeView{
		Present:       true,
		Authoritative: reconciled.Authoritative,
		Instance:      latest,
		HealthStatus:  health.Status,
	}
	if !reconciled.Authoritative {
		return view, nil
	}
	authClaims := make([]scenarioruntime.PortClaim, 0, len(reconciled.Claims))
	ports := map[string]int{}
	for _, claim := range reconciled.Claims {
		if !claim.Authoritative {
			continue
		}
		authClaims = append(authClaims, claim.Claim)
		if claim.Claim.Port <= 0 {
			continue
		}
		key := claim.Claim.EnvVar
		if key == "" {
			key = item.Manifest.PortEnvVar(claim.Claim.PortName)
		}
		if key == "" {
			continue
		}
		if _, exists := ports[key]; !exists {
			ports[key] = claim.Claim.Port
		}
	}
	view.Claims = authClaims
	view.Ports = ports
	return view, nil
}

func isNewerInstance(candidate, current scenarioruntime.Instance) bool {
	if candidate.Generation != current.Generation {
		return candidate.Generation > current.Generation
	}
	if !candidate.UpdatedAt.Equal(current.UpdatedAt) {
		return candidate.UpdatedAt.After(current.UpdatedAt)
	}
	return candidate.InstanceID > current.InstanceID
}

func shouldPreferLifecycleHealthStatus(snapshot scenarioruntime.HealthSnapshot) bool {
	if snapshot.Status == scenarioruntime.HealthStatusNotConfigured || snapshot.Status == scenarioruntime.HealthStatusUnknown {
		return true
	}
	return snapshot.SchemaValid != nil && !*snapshot.SchemaValid
}
