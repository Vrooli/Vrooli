package perfbudget_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-cloud/backup"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/certification"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/health"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/perfbudget"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// TestEveryRetryClassHasAFiniteCeiling [REQ:STC-P0-043] proves P23-A03 for
// retries: every execplan retry contract maps to a budget whose attempt
// count and total backoff are finite, whose per-attempt delay never exceeds
// the cap, and whose attempt beyond the ceiling is refused rather than
// delayed.
func TestEveryRetryClassHasAFiniteCeiling(t *testing.T) {
	b := loadBudgetsExternal(t)
	// The execplan vocabulary is the source of truth; a new retry class
	// without a budget must fail here.
	vocabulary := []string{execplan.RetrySafeReplay, execplan.RetryObserveThenReplay, execplan.RetryRecover}
	for _, class := range vocabulary {
		if _, err := b.RetryFor(class); err != nil {
			t.Fatalf("execplan retry class %q: %v", class, err)
		}
	}
	for _, class := range perfbudget.RequiredRetryClasses() {
		rb, err := b.RetryFor(class)
		if err != nil {
			t.Fatal(err)
		}
		cap := time.Duration(rb.BackoffCapSeconds * float64(time.Second))
		var total time.Duration
		for attempt := 1; attempt <= rb.MaxAttempts; attempt++ {
			d, ok := rb.Backoff(attempt)
			if !ok {
				t.Fatalf("%s attempt %d refused inside the ceiling", class, attempt)
			}
			if attempt == 1 && d != 0 {
				t.Fatalf("%s first attempt delayed by %s", class, d)
			}
			if d > cap {
				t.Fatalf("%s attempt %d backoff %s exceeds cap %s", class, attempt, d, cap)
			}
			total += d
		}
		if _, ok := rb.Backoff(rb.MaxAttempts + 1); ok {
			t.Fatalf("%s admitted attempt %d beyond the ceiling", class, rb.MaxAttempts+1)
		}
		if _, ok := rb.Backoff(1_000_000); ok {
			t.Fatalf("%s admitted an unbounded attempt", class)
		}
		if got := rb.TotalBackoff(); got != total || got > time.Duration(rb.MaxAttempts)*cap {
			t.Fatalf("%s total backoff %s (sum %s, bound %s)", class, got, total, time.Duration(rb.MaxAttempts)*cap)
		}
	}
	// recover is a single attempt by construction: a recovery is never
	// replayed blind (P07 design 4).
	recover, _ := b.RetryFor(execplan.RetryRecover)
	if recover.MaxAttempts != 1 {
		t.Fatalf("recover max_attempts = %d, want 1", recover.MaxAttempts)
	}
}

// TestOperationTimeoutsAreBoundedAndMatchTheFrozenBudget [REQ:STC-P0-043]
// proves the production operations config carries a positive bound for
// every timeout and that the frozen budget records the same numbers, so a
// silent change in either is caught.
func TestOperationTimeoutsAreBoundedAndMatchTheFrozenBudget(t *testing.T) {
	b := loadBudgetsExternal(t)
	cfg := operations.DefaultConfig()
	got := map[string]time.Duration{
		"queue":     cfg.QueueTimeout,
		"execution": cfg.ExecutionTimeout,
		"transport": cfg.TransportTimeout,
		"observer":  cfg.ObserverTimeout,
		"lease_ttl": cfg.LeaseTTL,
		"heartbeat": cfg.HeartbeatInterval,
		"reconcile": cfg.ReconcileInterval,
	}
	for name, d := range got {
		if d <= 0 {
			t.Fatalf("operations.DefaultConfig %s is unbounded", name)
		}
		raw, ok := b.Phase23.OperationTimeouts[name]
		if !ok {
			t.Fatalf("budget has no operation timeout %q", name)
		}
		var seconds float64
		if err := json.Unmarshal(raw, &seconds); err != nil {
			t.Fatalf("timeout %s: %v", name, err)
		}
		if time.Duration(seconds*float64(time.Second)) != d {
			t.Fatalf("timeout %s: code %s, budget %.0fs", name, d, seconds)
		}
	}
	if cfg.Workers != b.Phase23.DeploymentQueue.WorkerPoolSize {
		t.Fatalf("worker pool %d != budget %d", cfg.Workers, b.Phase23.DeploymentQueue.WorkerPoolSize)
	}
	if cfg.HeartbeatInterval*2 > cfg.LeaseTTL {
		t.Fatalf("heartbeat %s cannot keep a %s lease alive", cfg.HeartbeatInterval, cfg.LeaseTTL)
	}
}

// TestPerDeploymentSerialisationIsReceiptedNotDuplicated [REQ:STC-P0-043]
// references RUN-08 instead of re-proving it: the package-lane receipt
// exists, passed, and names the owning test; the budget's one-writer limit
// points at it.
func TestPerDeploymentSerialisationIsReceiptedNotDuplicated(t *testing.T) {
	m, err := certification.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := certification.LoadEvidenceDir(filepath.Join("..", "..", "certification", "evidence"), m)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range receipts {
		if r.CaseID != "RUN-08" || r.Lane != certification.LanePackage {
			continue
		}
		found = true
		if r.Verdict != certification.VerdictPassed {
			t.Fatalf("RUN-08 package receipt verdict %s", r.Verdict)
		}
		named := false
		for _, ref := range r.ArtifactRefs {
			if ref == "api/operations/service_test.go#TestCompetingOperationsSerialiseThroughTheOwner" {
				named = true
			}
		}
		if !named {
			t.Fatalf("RUN-08 receipt does not name the serialisation test: %v", r.ArtifactRefs)
		}
	}
	if !found {
		t.Fatalf("RUN-08 package-lane receipt missing; serialisation is unproven")
	}
	b := loadBudgetsExternal(t)
	if b.Phase23.DeploymentQueue.EffectfulOperationsPerDeploymentMax != 1 {
		t.Fatalf("budget allows %d writers per deployment", b.Phase23.DeploymentQueue.EffectfulOperationsPerDeploymentMax)
	}
}

// TestRetentionNeverDeletesProtectedOrActiveArtifacts [REQ:STC-P0-043]
// proves P23-A06 on the existing planners: under the frozen retention
// budget (and under a policy far stricter than it), backup.Plan keeps every
// point held by an active release, a retained predecessor, a non-terminal
// operation or a pin, and bundle.PlanVPSBundleGC keeps the active and
// predecessor bundles plus every protected digest.
func TestRetentionNeverDeletesProtectedOrActiveArtifacts(t *testing.T) {
	b := loadBudgetsExternal(t)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	rp := func(id, release, op string, age time.Duration) domain.RecoveryPoint {
		return domain.RecoveryPoint{ID: id, DeploymentID: "dep-1", ReleaseDigest: release, OperationID: op, CapturedAt: now.Add(-age)}
	}
	points := []domain.RecoveryPoint{
		rp("rp-active-old", "sha256:active", "", 400*24*time.Hour),
		rp("rp-predecessor-old", "sha256:predecessor", "", 400*24*time.Hour),
		rp("rp-inflight", "sha256:retired", "op-running", 400*24*time.Hour),
		rp("rp-pinned", "sha256:retired", "", 400*24*time.Hour),
		rp("rp-stale-1", "sha256:retired", "", 400*24*time.Hour),
		rp("rp-stale-2", "sha256:retired", "", 200*24*time.Hour),
		rp("rp-fresh-1", "sha256:retired", "", time.Hour),
	}
	refs := backup.References{
		ActiveReleases:        []string{"sha256:active"},
		RetainedReleases:      []string{"sha256:predecessor"},
		NonTerminalOperations: []string{"op-running"},
		PinnedRecoveryPoints:  []string{"rp-pinned"},
	}
	protected := map[string]bool{"rp-active-old": true, "rp-predecessor-old": true, "rp-inflight": true, "rp-pinned": true}

	frozen := backup.Policy{KeepLast: b.Phase23.RetentionBudgets.RecoveryPoints.KeepLast, MaxAge: time.Duration(b.Phase23.RetentionBudgets.RecoveryPoints.Days) * 24 * time.Hour}
	for name, policy := range map[string]backup.Policy{"frozen": frozen, "aggressive": {KeepLast: 1, MaxAge: time.Minute}} {
		plan := backup.Plan(points, refs, policy, now)
		for _, id := range plan.Delete {
			if protected[id] {
				t.Fatalf("%s policy deleted protected point %s: %+v", name, id, plan)
			}
		}
		for id := range protected {
			if _, ok := plan.Protected[id]; !ok {
				t.Fatalf("%s policy did not record protection for %s: %+v", name, id, plan)
			}
		}
		if len(plan.Delete) == 0 {
			t.Fatalf("%s policy retired nothing; stale unprotected points must be reclaimable: %+v", name, plan)
		}
	}
	// Under the frozen budget the fresh unprotected point stays.
	plan := backup.Plan(points, refs, frozen, now)
	for _, id := range plan.Delete {
		if id == "rp-fresh-1" {
			t.Fatalf("frozen policy deleted a fresh point: %+v", plan)
		}
	}

	bundles := []domain.VPSBundleInfo{
		{Filename: "app-active.tar.gz", ScenarioID: "app", Sha256: "active", SizeBytes: 10, ModTime: "2026-09-09T11:00:00Z"},
		{Filename: "app-predecessor.tar.gz", ScenarioID: "app", Sha256: "predecessor", SizeBytes: 10, ModTime: "2026-09-08T11:00:00Z"},
		{Filename: "app-old-protected.tar.gz", ScenarioID: "app", Sha256: "held-by-operation", SizeBytes: 10, ModTime: "2026-08-01T11:00:00Z"},
		{Filename: "app-old-1.tar.gz", ScenarioID: "app", Sha256: "old1", SizeBytes: 10, ModTime: "2026-08-02T11:00:00Z"},
		{Filename: "app-old-2.tar.gz", ScenarioID: "app", Sha256: "old2", SizeBytes: 10, ModTime: "2026-07-02T11:00:00Z"},
		{Filename: "neighbor.tar.gz", ScenarioID: "neighbor", Sha256: "n1", SizeBytes: 10, ModTime: "2026-01-01T11:00:00Z"},
	}
	kept, deleted, _ := bundle.PlanVPSBundleGC(bundles, "app", b.Phase23.RetentionBudgets.ReleaseArtifacts.KeepLatestPerScenario, []string{"held-by-operation"})
	keptSHA := map[string]bool{}
	for _, k := range kept {
		keptSHA[k.Sha256] = true
	}
	for _, want := range []string{"active", "predecessor", "held-by-operation"} {
		if !keptSHA[want] {
			t.Fatalf("bundle GC did not keep %s: kept %v deleted %v", want, kept, deleted)
		}
	}
	for _, d := range deleted {
		if d.ScenarioID != "app" {
			t.Fatalf("bundle GC scoped to app touched neighbour %s", d.Filename)
		}
		if d.Sha256 == "active" || d.Sha256 == "predecessor" || d.Sha256 == "held-by-operation" {
			t.Fatalf("bundle GC deleted %s", d.Filename)
		}
	}
	if len(deleted) != 2 {
		t.Fatalf("expected the two unprotected old bundles retired, got %v", deleted)
	}
}

// TestStaleObservationCannotReportHealthy [REQ:STC-P0-043] proves P23 step
// 13 through the health builder and the alert evaluator: an observation
// whose inspection is older than the freshness bound is STALE even when
// every check passed, the evaluator raises health_observation_stale, and
// perfbudget.ServiceHealthy is false; the same input inside the bound is healthy.
func TestStaleObservationCannotReportHealthy(t *testing.T) {
	b := loadBudgetsExternal(t)
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	dep := healthyDeployment()
	staleAfter := time.Duration(b.Phase23.Detection.StaleObservationAfterSeconds) * time.Second
	if staleAfter != health.DefaultMaxObservationAge {
		t.Fatalf("detection.stale_observation_after_seconds %s != health.DefaultMaxObservationAge %s", staleAfter, health.DefaultMaxObservationAge)
	}

	current := buildObservation(dep, inspected, inspected.Add(10*time.Second))
	if current.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY || current.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		t.Fatalf("current observation = %s/%s", current.GetStatus(), current.GetFreshness())
	}
	alerts := perfbudget.Evaluate(perfbudget.AlertInput{Observation: current, Now: inspected.Add(10 * time.Second)}, b.Phase23.Detection)
	if len(alerts) != 0 || !perfbudget.ServiceHealthy(alerts) {
		t.Fatalf("current healthy observation raised %+v", alerts)
	}

	stale := buildObservation(dep, inspected, inspected.Add(staleAfter+time.Second))
	if stale.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Fatalf("stale status = %s (checks unchanged, only age differs)", stale.GetStatus())
	}
	if stale.GetFreshness() != healthv1.Freshness_FRESHNESS_STALE {
		t.Fatalf("stale freshness = %s", stale.GetFreshness())
	}
	alerts = perfbudget.Evaluate(perfbudget.AlertInput{Observation: stale, Now: inspected.Add(staleAfter + time.Second)}, b.Phase23.Detection)
	if perfbudget.ServiceHealthy(alerts) {
		t.Fatalf("stale observation reported healthy: %+v", alerts)
	}
	if len(alerts) != 1 || alerts[0].CheckID != perfbudget.CheckHealthObservationStale || alerts[0].Severity != perfbudget.SeverityCritical {
		t.Fatalf("alerts = %+v", alerts)
	}
	// A CURRENT observation whose observed_at is nonetheless older than the
	// detection bound on the consumer clock is also refused: the consumer
	// does not trust the producer's freshness alone.
	relabelled := buildObservation(dep, inspected, inspected.Add(10*time.Second))
	alerts = perfbudget.Evaluate(perfbudget.AlertInput{Observation: relabelled, Now: inspected.Add(staleAfter + time.Minute)}, b.Phase23.Detection)
	if perfbudget.ServiceHealthy(alerts) {
		t.Fatalf("old CURRENT observation reported healthy: %+v", alerts)
	}
	// Absent observation: never healthy.
	if perfbudget.ServiceHealthy(perfbudget.Evaluate(perfbudget.AlertInput{Now: inspected}, b.Phase23.Detection)) {
		t.Fatalf("missing observation reported healthy")
	}
}

// TestAlertConditionsAndRecoveryTransitions [REQ:STC-P0-043] proves the
// alert contract of docs/reference/service-objectives.md: a controlled
// outage, a certificate inside the warning window, a recovery point older
// than the RPO and a pending rotation each raise exactly one typed alert
// with a stable incident_id, the event facts carry notification-hub's
// required keys, and recovery emits a resolved event for the same incident
// (P23-A04, P23-A05).
func TestAlertConditionsAndRecoveryTransitions(t *testing.T) {
	b := loadBudgetsExternal(t)
	d := b.Phase23.Detection
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	dep := healthyDeployment()

	outage := buildObservationWithStatus(dep, now, now, "stopped")
	if outage.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY {
		t.Fatalf("outage status = %s", outage.GetStatus())
	}
	open := perfbudget.Evaluate(perfbudget.AlertInput{
		Observation:         outage,
		CertificateNotAfter: now.Add(time.Duration(d.CertificateExpiryWarningDays) * 24 * time.Hour),
		HasProtectedWrites:  true,
		LastRecoveryPointAt: now.Add(-time.Duration(d.BackupRPOSecondsMax+1) * time.Second),
		RotationPending:     []string{"postgres/app"},
		Now:                 now,
	}, d)
	byCheck := map[string]perfbudget.Alert{}
	for _, a := range open {
		byCheck[a.CheckID] = a
	}
	for _, check := range []string{perfbudget.CheckServiceUnavailable, perfbudget.CheckCertificateExpiring, perfbudget.CheckBackupFreshness, perfbudget.CheckCredentialRotationPending} {
		a, ok := byCheck[check]
		if !ok {
			t.Fatalf("condition %s not raised: %+v", check, open)
		}
		if a.Status != perfbudget.AlertOpen || a.IncidentID != dep.ID+":"+check || a.DeploymentID != dep.ID || a.NextAction.Reference == "" {
			t.Fatalf("alert %s malformed: %+v", check, a)
		}
		facts := a.Facts()
		for _, key := range []string{"check_id", "severity", "status", "message", "reason", "incident_id", "deployment_id", "target_id", "release_digest", "observed_at", "detected_at", "next_action"} {
			if _, ok := facts[key]; !ok {
				t.Fatalf("alert %s facts missing %s", check, key)
			}
		}
		if _, err := json.Marshal(facts); err != nil {
			t.Fatalf("facts not encodable: %v", err)
		}
	}
	if len(open) != 4 || perfbudget.ServiceHealthy(open) {
		t.Fatalf("open = %d alerts, healthy=%v", len(open), perfbudget.ServiceHealthy(open))
	}
	if byCheck[perfbudget.CheckServiceUnavailable].Severity != perfbudget.SeverityCritical || byCheck[perfbudget.CheckCertificateExpiring].Severity != perfbudget.SeverityWarning {
		t.Fatalf("severities: %+v", byCheck)
	}
	// A certificate one day outside the window raises nothing.
	quiet := perfbudget.Evaluate(perfbudget.AlertInput{Observation: buildObservation(dep, now, now), CertificateNotAfter: now.Add(time.Duration(d.CertificateExpiryWarningDays+1) * 24 * time.Hour), Now: now}, d)
	if len(quiet) != 0 {
		t.Fatalf("certificate outside the window raised %+v", quiet)
	}

	// Recovery: service back, fresh recovery point, rotation done; the
	// certificate is still expiring.
	later := now.Add(90 * time.Second)
	recovered := perfbudget.Evaluate(perfbudget.AlertInput{
		Observation:         buildObservation(dep, later, later),
		CertificateNotAfter: now.Add(time.Duration(d.CertificateExpiryWarningDays) * 24 * time.Hour),
		HasProtectedWrites:  true,
		LastRecoveryPointAt: later,
		Now:                 later,
	}, d)
	events := perfbudget.Transitions(open, recovered, later)
	var resolved []string
	for _, e := range events {
		if e.Status == perfbudget.AlertResolved {
			resolved = append(resolved, e.CheckID)
			if e.DetectedAt != later || e.Severity != perfbudget.SeverityInformational {
				t.Fatalf("resolved event malformed: %+v", e)
			}
		} else {
			t.Fatalf("unexpected newly opened alert on recovery: %+v", e)
		}
	}
	if len(resolved) != 3 || !perfbudget.ServiceHealthy(recovered) {
		t.Fatalf("resolved = %v, healthy=%v (certificate alert must stay open)", resolved, perfbudget.ServiceHealthy(recovered))
	}
	// Steady state publishes nothing.
	if again := perfbudget.Transitions(recovered, recovered, later.Add(time.Minute)); len(again) != 0 {
		t.Fatalf("steady state re-published %+v", again)
	}
}

// ---- observation fixtures (mirrors api/health test fixtures through the
// exported vps.ComputeHealth + health.Build seams) ----------------------

func healthyDeployment() *domain.Deployment {
	now := time.Date(2026, 9, 9, 11, 0, 0, 0, time.UTC)
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	return &domain.Deployment{
		ID: "3c1c9a1e-1111-4222-8333-444455556666", Name: "app @ example.com", ScenarioID: "app", Environment: "production",
		Target:       identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "1.2.3.4"}},
		Status:       domain.StatusDeployed,
		Manifest:     json.RawMessage(`{"version":"1","scenario":{"id":"app"},"edge":{"domain":"example.com"}}`),
		BundleSHA256: &sha, LastDeployedAt: &now,
	}
}

func buildObservation(dep *domain.Deployment, inspected, now time.Time) *healthv1.HealthObservation {
	return buildObservationWithStatus(dep, inspected, now, "running")
}

func buildObservationWithStatus(dep *domain.Deployment, inspected, now time.Time, scenarioStatus string) *healthv1.HealthObservation {
	live := &domain.LiveStateResult{
		OK:        true,
		Timestamp: inspected.UTC().Format(time.RFC3339),
		System: &domain.SystemState{
			SSH:    domain.SSHHealth{Connected: true, LatencyMs: 40, AuthMode: string(sshidentity.AuthModeExplicitKey), VerificationState: string(sshidentity.VerificationAuthorized)},
			CPU:    domain.CPUInfo{Cores: 2, UsagePercent: 10},
			Memory: domain.MemoryInfo{TotalMB: 4096, UsedMB: 1024, UsagePercent: 25},
			Disk:   domain.DiskInfo{TotalGB: 40, UsedGB: 8, UsagePercent: 20},
		},
		Processes: &domain.ProcessState{
			Scenarios: []domain.ScenarioProcess{{ID: "app", Status: scenarioStatus, PID: 1234}},
			Resources: []domain.ResourceProcess{{ID: "postgres", Status: "running", PID: 5678, Port: 5432}},
		},
		Caddy: &domain.CaddyState{Running: true, Domain: "example.com"},
	}
	manifest := domain.CloudManifest{
		Scenario:     domain.ManifestScenario{ID: "app"},
		Target:       domain.ManifestTarget{VPS: &domain.ManifestVPS{Host: "1.2.3.4"}},
		Dependencies: domain.ManifestDependencies{Resources: []string{"postgres"}},
		Edge:         domain.ManifestEdge{Domain: "example.com"},
	}
	ident := sshidentity.DeploymentSSHIdentity{KeyPath: "~/.ssh/id_ed25519", PublicKeyFingerprint: "SHA256:test", AuthMode: sshidentity.AuthModeExplicitKey, VerificationState: sshidentity.VerificationAuthorized}
	dnsEval := &dns.Evaluation{
		EdgeDomain: "example.com",
		VPS:        domain.DNSLookupResult{Host: "1.2.3.4", IPs: []string{"1.2.3.4"}},
		Statuses:   []dns.DomainStatus{{Role: "apex", Host: "example.com", PointsToVPS: true, Lookup: domain.DNSLookupResult{IPs: []string{"1.2.3.4"}}}},
	}
	tlsSnap := &tlsinfo.Snapshot{Probe: tlsinfo.ProbeResult{Valid: true, Issuer: "Let's Encrypt", DaysRemaining: 60}, ALPN: tlsinfo.ALPNCheck{Status: tlsinfo.ALPNPass, Message: "ok"}}
	report := vps.ComputeHealth(dep, manifest, ident, live, dnsEval, tlsSnap, nil)
	report.Freshness = &domain.FreshnessStatus{Status: domain.FreshnessCurrent, Summary: "bundle matches"}
	return health.Build(health.Input{Deployment: dep, Report: report, LiveState: live, Now: now}, health.DefaultPolicy())
}

func loadBudgetsExternal(t *testing.T) *perfbudget.Budgets {
	t.Helper()
	b, err := perfbudget.Load(filepath.Join("..", "..", "certification", "budgets.json"))
	if err != nil {
		t.Fatalf("load budgets: %v", err)
	}
	return b
}
