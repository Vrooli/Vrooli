package orchestrator

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func (s *Service) registryReadMode() (string, error) {
	return scenarioruntime.ModeFromEnv()
}

func (s *Service) registryReadsEnabled(scenarioName string) (bool, bool, error) {
	mode, err := s.registryReadMode()
	if err != nil {
		return false, false, err
	}
	return scenarioruntime.ReadEnabledForScenario(mode, scenarioName), scenarioruntime.StrictReadsForScenario(mode, scenarioName), nil
}

func (s *Service) registryDetailsByScenario(ctx context.Context, items []scenario.Scenario) (map[string]Detail, error) {
	if len(items) == 0 {
		return map[string]Detail{}, nil
	}
	store, err := s.openRuntimeRegistry(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	type reconciledDetail struct {
		detail   Detail
		instance scenarioruntime.Instance
	}
	latest := make(map[string]reconciledDetail, len(instances))
	for _, instance := range instances {
		if !scenarioruntime.ScenarioAllowedByEnv(instance.Scenario) {
			continue
		}
		item, ok := registryScenarioBySlug(items, instance.Scenario)
		if !ok {
			continue
		}
		detail, authoritative, err := s.detailFromRegistryInstance(ctx, store, item, instance)
		if err != nil {
			return nil, err
		}
		if !authoritative {
			continue
		}
		if current, ok := latest[instance.Scenario]; !ok || isNewerRuntimeInstance(instance, current.instance) {
			latest[instance.Scenario] = reconciledDetail{detail: detail, instance: instance}
		}
	}

	out := make(map[string]Detail, len(latest))
	for _, item := range items {
		entry, ok := latest[item.Slug]
		if !ok {
			continue
		}
		out[item.Slug] = entry.detail
	}
	return out, nil
}

func registryScenarioBySlug(items []scenario.Scenario, slug string) (scenario.Scenario, bool) {
	for _, item := range items {
		if item.Slug == slug {
			return item, true
		}
	}
	return scenario.Scenario{}, false
}

func (s *Service) registryDetail(ctx context.Context, item scenario.Scenario) (Detail, bool, error) {
	store, err := s.openRuntimeRegistry(ctx)
	if err != nil {
		return Detail{}, false, err
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: item.Slug,
		Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return Detail{}, false, err
	}
	if len(instances) == 0 {
		return Detail{}, false, nil
	}
	detail, authoritative, err := s.detailFromRegistryInstance(ctx, store, item, latestRuntimeInstance(instances))
	if err != nil {
		return Detail{}, false, err
	}
	return detail, authoritative, nil
}

func (s *Service) openRuntimeRegistry(ctx context.Context) (runtimeRegistryQueryStore, error) {
	if s.runtimeRegistry != nil {
		return s.runtimeRegistry(ctx, s.Home)
	}
	return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: s.Home})
}

func (s *Service) detailFromRegistryInstance(ctx context.Context, store runtimeRegistryQueryStore, item scenario.Scenario, instance scenarioruntime.Instance) (Detail, bool, error) {
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		InstanceID: instance.InstanceID,
		Statuses: []string{
			scenarioruntime.ClaimStatusReserved,
			scenarioruntime.ClaimStatusBound,
		},
	})
	if err != nil {
		return Detail{}, false, err
	}
	refs, err := store.ListProcessRefs(ctx, instance.InstanceID)
	if err != nil {
		return Detail{}, false, err
	}
	health, err := store.GetHealthSnapshot(ctx, instance.InstanceID)
	if err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		return Detail{}, false, err
	}
	reconciled, err := s.reconcileRegistryRuntime(ctx, instance, claims, refs)
	if err != nil {
		return Detail{}, false, err
	}
	if !reconciled.Authoritative {
		return Detail{}, false, nil
	}

	records := recordsFromProcessRefs(instance.Scenario, refs)
	runtime := process.ScenarioRuntime{
		Name:         item.Slug,
		Records:      records,
		ProcessCount: countRunningProcessRefs(refs),
		Runtime:      "registry",
	}
	if !instance.StartedAt.IsZero() {
		started := instance.StartedAt
		runtime.StartedAt = &started
		runtime.Runtime = humanRegistryRuntime(instance.StartedAt)
	}

	details := registryRuntimeDetails(item.Manifest, instance, authoritativeClaims(reconciled.Claims), records, health)
	return Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  details,
	}, true, nil
}

func (s *Service) reconcileRegistryRuntime(ctx context.Context, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef) (scenarioruntime.ReconcileResult, error) {
	provider := s.hostSession
	if provider == nil {
		provider = hostsession.DefaultProvider{}.Current
	}
	host, err := provider(ctx, s.Home)
	if err != nil {
		return scenarioruntime.ReconcileResult{}, err
	}
	return scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
		Now:           time.Now().UTC(),
		CurrentBootID: host.BootID,
		Instance:      instance,
		Claims:        claims,
		ProcessRefs:   refs,
		Processes:     processEvidence(refs),
		Listeners:     listenerEvidence(claims, refs),
	}), nil
}

func processEvidence(refs []scenarioruntime.ProcessRef) map[string]scenarioruntime.ProcessEvidence {
	out := make(map[string]scenarioruntime.ProcessEvidence)
	for _, ref := range refs {
		if ref.PID == nil {
			continue
		}
		pid := *ref.PID
		out[strconv.Itoa(pid)] = scenarioruntime.ProcessEvidence{Known: true, Running: process.IsPIDRunning(pid)}
	}
	return out
}

func listenerEvidence(claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef) map[int]scenarioruntime.ListenerEvidence {
	if len(refs) == 0 {
		return nil
	}
	out := make(map[int]scenarioruntime.ListenerEvidence)
	for _, claim := range claims {
		if claim.Status != scenarioruntime.ClaimStatusBound || claim.Port <= 0 {
			continue
		}
		inspection, err := network.InspectPortListeners(claim.Port)
		if err != nil || !inspection.Inspection.Available {
			out[claim.Port] = scenarioruntime.ListenerEvidence{Known: false}
			continue
		}
		out[claim.Port] = scenarioruntime.ListenerEvidence{Known: true, Listening: len(inspection.Listeners) > 0}
	}
	return out
}

func authoritativeClaims(claims []scenarioruntime.ReconciledClaim) []scenarioruntime.PortClaim {
	out := make([]scenarioruntime.PortClaim, 0, len(claims))
	for _, claim := range claims {
		if claim.Authoritative {
			out = append(out, claim.Claim)
		}
	}
	return out
}

func registryRuntimeDetails(manifest scenario.ServiceManifest, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim, records []process.Record, health scenarioruntime.HealthSnapshot) scenario.RuntimeDetails {
	bindings := make([]scenario.RuntimePortBinding, 0, len(claims))
	ports := make(map[string]int, len(claims))
	for _, claim := range claims {
		if claim.Port <= 0 {
			continue
		}
		key := claim.EnvVar
		if key == "" {
			key = manifest.PortEnvVar(claim.PortName)
		}
		if key == "" {
			continue
		}
		if _, exists := ports[key]; !exists {
			ports[key] = claim.Port
		}
		bindings = append(bindings, scenario.RuntimePortBinding{
			Key:  key,
			Step: claim.PortName,
			Port: claim.Port,
		})
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Key == bindings[j].Key {
			return bindings[i].Step < bindings[j].Step
		}
		return bindings[i].Key < bindings[j].Key
	})

	status := "stopped"
	switch instance.Status {
	case scenarioruntime.StatusRunning:
		status = "running"
	case scenarioruntime.StatusStarting:
		status = "starting"
	case scenarioruntime.StatusFailed:
		status = "failed"
	case scenarioruntime.StatusExpired:
		status = "expired"
	case scenarioruntime.StatusStopping:
		status = "stopping"
	}

	var startedAt *time.Time
	if !instance.StartedAt.IsZero() {
		started := instance.StartedAt
		startedAt = &started
	}

	return scenario.RuntimeDetails{
		Status:       status,
		Processes:    countRunningRecords(records),
		Runtime:      "registry",
		StartedAt:    startedAt,
		Ports:        ports,
		PortBindings: bindings,
		ProcessInfo:  append([]process.Record(nil), records...),
		Health:       registryHealthStatus(health),
	}
}

func recordsFromProcessRefs(scenarioName string, refs []scenarioruntime.ProcessRef) []process.Record {
	records := make([]process.Record, 0, len(refs))
	for _, ref := range refs {
		record := process.Record{
			ProcessID: ref.ProcessID,
			Scenario:  scenarioName,
			Step:      ref.Step,
			Command:   ref.Command,
			LogFile:   ref.LogFile,
			StartedAt: ref.StartedAt,
			Status:    ref.Status,
		}
		if ref.PID != nil {
			record.PID = *ref.PID
		}
		if ref.PGID != nil {
			record.PGID = *ref.PGID
		}
		records = append(records, record)
	}
	return records
}

func countRunningProcessRefs(refs []scenarioruntime.ProcessRef) int {
	count := 0
	for _, ref := range refs {
		if ref.Status == "" || ref.Status == "running" {
			count++
		}
	}
	return count
}

func countRunningRecords(records []process.Record) int {
	count := 0
	for _, record := range records {
		if record.Status == "" || record.Status == "running" {
			count++
		}
	}
	return count
}

func registryHealthStatus(snapshot scenarioruntime.HealthSnapshot) string {
	switch snapshot.Status {
	case "", scenarioruntime.HealthStatusUnknown:
		return ""
	case scenarioruntime.HealthStatusNotConfigured:
		return "running"
	default:
		return snapshot.Status
	}
}

func humanRegistryRuntime(startedAt time.Time) string {
	if startedAt.IsZero() {
		return "registry"
	}
	d := time.Since(startedAt)
	if d < time.Second {
		return "just now"
	}
	if d < time.Minute {
		return d.Truncate(time.Second).String()
	}
	if d < time.Hour {
		return d.Truncate(time.Minute).String()
	}
	return d.Truncate(time.Hour).String()
}

func latestRuntimeInstance(instances []scenarioruntime.Instance) scenarioruntime.Instance {
	var latest scenarioruntime.Instance
	for _, instance := range instances {
		if latest.InstanceID == "" || isNewerRuntimeInstance(instance, latest) {
			latest = instance
		}
	}
	return latest
}

func isNewerRuntimeInstance(candidate, current scenarioruntime.Instance) bool {
	if candidate.Generation != current.Generation {
		return candidate.Generation > current.Generation
	}
	if !candidate.UpdatedAt.Equal(current.UpdatedAt) {
		return candidate.UpdatedAt.After(current.UpdatedAt)
	}
	return candidate.InstanceID > current.InstanceID
}
