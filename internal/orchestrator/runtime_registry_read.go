package orchestrator

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

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
	relevant := make([]scenarioruntime.Instance, 0, len(instances))
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		if _, ok := registryScenarioBySlug(items, instance.Scenario); !ok {
			continue
		}
		relevant = append(relevant, instance)
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	if len(relevant) == 0 {
		return map[string]Detail{}, nil
	}

	// Constant query count regardless of fleet size: one claims query grouped
	// in memory plus the chunked batch reads for refs and health.
	activeClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{Statuses: scenarioruntime.ActivePortClaimStatuses()})
	if err != nil {
		return nil, err
	}
	claimsByInstance := make(map[string][]scenarioruntime.PortClaim, len(relevant))
	for _, claim := range activeClaims {
		claimsByInstance[claim.InstanceID] = append(claimsByInstance[claim.InstanceID], claim)
	}
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	healthByInstance, err := store.GetHealthSnapshots(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}

	// Evidence is captured once, AFTER every store read above, so no claim is
	// judged by listener/process evidence older than the claim itself.
	evidence, err := s.captureRegistryEvidence(ctx)
	if err != nil {
		return nil, err
	}

	type reconciledDetail struct {
		detail   Detail
		instance scenarioruntime.Instance
	}
	// Key the dedup by (scenario, variant) so a live and a shadow instance of
	// the same scenario never collapse into one "latest" entry — without this,
	// a shadow could overwrite the live instance's reported status (or vice
	// versa). Live ⇒ bare slug, so single-instance scenarios are unchanged.
	latest := make(map[string]reconciledDetail, len(relevant))
	for _, instance := range relevant {
		item, ok := registryScenarioBySlug(items, instance.Scenario)
		if !ok {
			continue
		}
		detail, authoritative := detailFromRegistryInstance(item, instance,
			claimsByInstance[instance.InstanceID], refsByInstance[instance.InstanceID],
			healthByInstance[instance.InstanceID], evidence)
		if !authoritative {
			continue
		}
		instKey := scenarioruntime.InstanceKey{Scenario: instance.Scenario, Variant: instance.Variant}.Slug()
		if current, ok := latest[instKey]; !ok || isNewerRuntimeInstance(instance, current.instance) {
			latest[instKey] = reconciledDetail{detail: detail, instance: instance}
		}
	}

	out := make(map[string]Detail, len(latest))
	for _, item := range items {
		entry, ok := latest[scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Slug()]
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
		Variant:  scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize().Variant,
		Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return Detail{}, false, err
	}
	if len(instances) == 0 {
		return Detail{}, false, nil
	}
	instance := latestRuntimeInstance(instances)
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		InstanceID: instance.InstanceID,
		Statuses:   scenarioruntime.ActivePortClaimStatuses(),
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
	// Evidence after the store reads — same ordering rule as the fleet path.
	evidence, err := s.captureRegistryEvidence(ctx)
	if err != nil {
		return Detail{}, false, err
	}
	detail, authoritative := detailFromRegistryInstance(item, instance, claims, refs, health, evidence)
	return detail, authoritative, nil
}

func (s *Service) registryDetailAtPath(ctx context.Context, item scenario.Scenario, path string) (Detail, bool, error) {
	store, err := s.openRuntimeRegistry(ctx)
	if err != nil {
		return Detail{}, false, err
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: item.Slug,
		Variant:  scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize().Variant,
		Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return Detail{}, false, err
	}
	cleanPath := filepath.Clean(path)
	matching := make([]scenarioruntime.Instance, 0, len(instances))
	for _, instance := range instances {
		if filepath.Clean(instance.WorkingDir) == cleanPath {
			matching = append(matching, instance)
		}
	}
	if len(matching) == 0 {
		return Detail{}, false, nil
	}

	instance := latestRuntimeInstance(matching)
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID, Statuses: scenarioruntime.ActivePortClaimStatuses()})
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
	evidence, err := s.captureRegistryEvidence(ctx)
	if err != nil {
		return Detail{}, false, err
	}
	detail, authoritative := detailFromRegistryInstance(item, instance, claims, refs, health, evidence)
	return detail, authoritative, nil
}

func (s *Service) openRuntimeRegistry(ctx context.Context) (runtimeRegistryQueryStore, error) {
	if s.runtimeRegistry != nil {
		return s.runtimeRegistry(ctx, s.Home)
	}
	return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: s.Home})
}

// registryEvidence is the per-operation runtime evidence shared across every
// instance reconciled in one listing: the host boot ID and one listener
// snapshot. Capture it AFTER the store reads whose claims it judges.
type registryEvidence struct {
	bootID   string
	snapshot network.TCPListenerSnapshot
}

func (s *Service) captureRegistryEvidence(ctx context.Context) (registryEvidence, error) {
	provider := s.hostSession
	if provider == nil {
		provider = hostsession.DefaultProvider{}.Current
	}
	host, err := provider(ctx, s.Home)
	if err != nil {
		return registryEvidence{}, err
	}
	return registryEvidence{
		bootID:   host.BootID,
		snapshot: network.CaptureTCPListenerSnapshot(),
	}, nil
}

func detailFromRegistryInstance(item scenario.Scenario, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef, health scenarioruntime.HealthSnapshot, evidence registryEvidence) (Detail, bool) {
	reconciled := reconcileRegistryRuntime(instance, claims, refs, evidence)
	if !reconciled.Authoritative {
		return Detail{}, false
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
	}, true
}

func reconcileRegistryRuntime(instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef, evidence registryEvidence) scenarioruntime.ReconcileResult {
	return scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
		Now:           time.Now().UTC(),
		CurrentBootID: evidence.bootID,
		Instance:      instance,
		Claims:        claims,
		ProcessRefs:   refs,
		Processes:     scenarioruntime.ProcessEvidenceFromRefs(refs, process.IsPIDRunning),
		Listeners: scenarioruntime.ListenerEvidenceFromClaims(claims, refs, func(port int) scenarioruntime.ListenerEvidence {
			state := evidence.snapshot.Listening(port)
			if !state.Known {
				return scenarioruntime.ListenerEvidence{Known: false}
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
		}),
	})
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
			Key:            key,
			Step:           claim.PortName,
			Port:           claim.Port,
			ListenerStatus: claim.ListenerStatus,
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
		HealthError:  health.Error,
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
