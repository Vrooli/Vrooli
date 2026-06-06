package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ownerPID := os.Getpid()
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario:      item.Slug,
		Variant:       item.Variant,
		Status:        scenarioruntime.StatusStarting,
		Phase:         "planning",
		ScopePath:     r.Root,
		OwnerKind:     scenarioruntime.OwnerKindLifecycle,
		OwnerPID:      &ownerPID,
		WorkingDir:    item.Path,
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

func (s *runtimeRegistrySession) heartbeat(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	updated, err := s.store.HeartbeatLease(ctx, s.instance.InstanceID, s.instance.Generation, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		return fmt.Errorf("heartbeat scenario runtime lease: %w", err)
	}
	s.instance = updated
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
		readiness := healthStatus == "healthy" || healthStatus == "degraded" || healthStatus == "running"
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

func (r *Runner) ensureRuntimeSupervisor(ctx context.Context, session runtimeRegistrySession) error {
	if !session.enabled {
		return nil
	}
	mode := runtimesupervisor.ModeFromEnv()
	if mode == runtimesupervisor.ModeOff {
		return nil
	}
	deps := r.runtimeDeps()
	err := deps.ensureRuntimeSupervisor(ctx, r.Home, io.Discard, r.Err)
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
	case "degraded":
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
	reconciled := scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
		Now:           time.Now().UTC(),
		CurrentBootID: host.BootID,
		Instance:      latest,
		Claims:        claims,
		ProcessRefs:   refs,
		Processes:     scenarioruntime.ProcessEvidenceFromRefs(refs, process.IsPIDRunning),
		Listeners: scenarioruntime.ListenerEvidenceFromClaims(claims, refs, func(port int) scenarioruntime.ListenerEvidence {
			inspection, err := network.InspectPortListeners(port)
			if err != nil || !inspection.Inspection.Available {
				return scenarioruntime.ListenerEvidence{Known: false}
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: len(inspection.Listeners) > 0}
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
