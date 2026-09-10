package health

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

const bundleSHA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func fixtureDeployment(status domain.DeploymentStatus) *domain.Deployment {
	now := time.Now().UTC()
	sha := bundleSHA
	return &domain.Deployment{
		ID:             "7e493e04-1111-4222-8333-444455556666",
		Name:           "landing-page @ example.com",
		ScenarioID:     "landing-page",
		Environment:    "production",
		Target:         identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "1.2.3.4"}},
		Status:         status,
		Manifest:       json.RawMessage(`{"version":"1","scenario":{"id":"landing-page"},"edge":{"domain":"example.com"}}`),
		BundleSHA256:   &sha,
		LastDeployedAt: &now,
	}
}

func fixtureManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Scenario:     domain.ManifestScenario{ID: "landing-page"},
		Target:       domain.ManifestTarget{VPS: &domain.ManifestVPS{Host: "1.2.3.4"}},
		Dependencies: domain.ManifestDependencies{Resources: []string{"postgres"}},
		Edge:         domain.ManifestEdge{Domain: "example.com"},
	}
}

func fixtureIdentity() sshidentity.DeploymentSSHIdentity {
	return sshidentity.DeploymentSSHIdentity{
		KeyPath:              "~/.ssh/id_ed25519",
		PublicKeyFingerprint: "SHA256:test",
		AuthMode:             sshidentity.AuthModeExplicitKey,
		VerificationState:    sshidentity.VerificationAuthorized,
	}
}

func fixtureLiveState(observedAt time.Time, scenarioStatus string) *domain.LiveStateResult {
	return &domain.LiveStateResult{
		OK:        true,
		Timestamp: observedAt.UTC().Format(time.RFC3339),
		System: &domain.SystemState{
			SSH:    domain.SSHHealth{Connected: true, LatencyMs: 40, AuthMode: string(sshidentity.AuthModeExplicitKey), VerificationState: string(sshidentity.VerificationAuthorized)},
			CPU:    domain.CPUInfo{Cores: 4, UsagePercent: 10},
			Memory: domain.MemoryInfo{TotalMB: 4096, UsedMB: 1024, UsagePercent: 25},
			Disk:   domain.DiskInfo{TotalGB: 200, UsedGB: 40, UsagePercent: 20},
		},
		Processes: &domain.ProcessState{
			Scenarios: []domain.ScenarioProcess{{ID: "landing-page", Status: scenarioStatus, PID: 1234}},
			Resources: []domain.ResourceProcess{{ID: "postgres", Status: "running", PID: 5678, Port: 5432}},
		},
		Caddy: &domain.CaddyState{Running: true, Domain: "example.com"},
	}
}

func fixtureDNS() *dns.Evaluation {
	return &dns.Evaluation{
		EdgeDomain: "example.com",
		VPS:        domain.DNSLookupResult{Host: "1.2.3.4", IPs: []string{"1.2.3.4"}},
		Statuses: []dns.DomainStatus{
			{Role: "apex", Host: "example.com", PointsToVPS: true, Lookup: domain.DNSLookupResult{IPs: []string{"1.2.3.4"}}},
		},
	}
}

func fixtureTLS() *tlsinfo.Snapshot {
	return &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{Valid: true, Issuer: "Let's Encrypt", DaysRemaining: 60},
		ALPN:  tlsinfo.ALPNCheck{Status: tlsinfo.ALPNPass, Message: "ok"},
	}
}

func build(t *testing.T, dep *domain.Deployment, live *domain.LiveStateResult, dnsEval *dns.Evaluation, tlsSnap *tlsinfo.Snapshot, now time.Time) *healthv1.HealthObservation {
	t.Helper()
	report := vps.ComputeHealth(dep, fixtureManifest(), fixtureIdentity(), live, dnsEval, tlsSnap, nil)
	report.Freshness = &domain.FreshnessStatus{Status: domain.FreshnessCurrent, Summary: "bundle matches"}
	return Build(Input{Deployment: dep, Report: report, LiveState: live, Now: now}, DefaultPolicy())
}

func checkByID(obs *healthv1.HealthObservation, id string) *healthv1.HealthCheck {
	for _, c := range obs.GetChecks() {
		if c.GetId() == id {
			return c
		}
	}
	return nil
}

// TestBuildCarriesIdentityReleaseAndProducerTime [REQ:STC-P0-033] proves the
// observation is bound to the deployment id (not the scenario name), the
// observed release digest, the target key and the inspection time.
func TestBuildCarriesIdentityReleaseAndProducerTime(t *testing.T) {
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	dep := fixtureDeployment(domain.StatusDeployed)
	obs := build(t, dep, fixtureLiveState(inspected, "running"), fixtureDNS(), fixtureTLS(), inspected.Add(30*time.Second))

	if obs.GetDeploymentId() != dep.ID || obs.GetDeploymentId() == dep.ScenarioID {
		t.Fatalf("deployment_id = %q, want the record id, not the scenario name", obs.GetDeploymentId())
	}
	if obs.GetTargetId() != "host:1.2.3.4" {
		t.Fatalf("target_id = %q", obs.GetTargetId())
	}
	if obs.GetObservedReleaseDigest() != "sha256:"+bundleSHA {
		t.Fatalf("observed_release_digest = %q", obs.GetObservedReleaseDigest())
	}
	if !strings.HasPrefix(obs.GetObservedConfigurationDigest(), "sha256:") {
		t.Fatalf("observed_configuration_digest = %q", obs.GetObservedConfigurationDigest())
	}
	if got := obs.GetObservedAt().AsTime(); !got.Equal(inspected) {
		t.Fatalf("observed_at = %s, want the inspection time %s (not the response time)", got, inspected)
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY || obs.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		t.Fatalf("status/freshness = %s/%s", obs.GetStatus(), obs.GetFreshness())
	}
	if obs.GetPartial() || len(obs.GetMissingDependencies()) != 0 {
		t.Fatalf("complete observation flagged partial: %v", obs.GetMissingDependencies())
	}
	if obs.GetProducerRef() != ProducerRef {
		t.Fatalf("producer_ref = %q", obs.GetProducerRef())
	}
	for _, id := range []string{CheckHostPresence, CheckTransportReach, CheckApplicationReadiness, CheckReleaseFreshness, CheckEdgeDNS, CheckEdgeTLS} {
		c := checkByID(obs, id)
		if c == nil {
			t.Fatalf("check %s missing", id)
		}
		if c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_PASSED {
			t.Fatalf("check %s = %s (%s)", id, c.GetStatus(), c.GetDetail())
		}
	}
}

// TestBuildKeepsCurrentUnhealthyDistinctFromStaleHealthy [REQ:STC-P0-033]
// proves P16 step 8: status and freshness are independent axes.
func TestBuildKeepsCurrentUnhealthyDistinctFromStaleHealthy(t *testing.T) {
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	dep := fixtureDeployment(domain.StatusDeployed)

	currentUnhealthy := build(t, dep, fixtureLiveState(inspected, "stopped"), fixtureDNS(), fixtureTLS(), inspected.Add(10*time.Second))
	if currentUnhealthy.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY {
		t.Fatalf("stopped scenario status = %s", currentUnhealthy.GetStatus())
	}
	if currentUnhealthy.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		t.Fatalf("current unhealthy freshness = %s", currentUnhealthy.GetFreshness())
	}
	if c := checkByID(currentUnhealthy, CheckApplicationReadiness); c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_FAILED || c.GetReasonCode() != "process_not_running" {
		t.Fatalf("application_readiness = %s/%s", c.GetStatus(), c.GetReasonCode())
	}
	if c := checkByID(currentUnhealthy, CheckHostPresence); c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_PASSED {
		t.Fatalf("host_presence should stay passed when only the app is down: %s", c.GetStatus())
	}
	if len(currentUnhealthy.GetNextActions()) == 0 || currentUnhealthy.GetNextActions()[0].GetOwner() != "scenario-to-cloud" {
		t.Fatalf("typed next actions missing: %v", currentUnhealthy.GetNextActions())
	}

	staleHealthy := build(t, dep, fixtureLiveState(inspected, "running"), fixtureDNS(), fixtureTLS(), inspected.Add(DefaultMaxObservationAge+time.Second))
	if staleHealthy.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Fatalf("stale healthy status = %s", staleHealthy.GetStatus())
	}
	if staleHealthy.GetFreshness() != healthv1.Freshness_FRESHNESS_STALE {
		t.Fatalf("stale healthy freshness = %s", staleHealthy.GetFreshness())
	}
}

// TestBuildSSHFailureIsUnknownNeverHealthy [REQ:STC-P0-033] proves that a
// failed inspection yields UNKNOWN status and UNKNOWN freshness with the
// live-state dependency listed as missing.
func TestBuildSSHFailureIsUnknownNeverHealthy(t *testing.T) {
	dep := fixtureDeployment(domain.StatusDeployed)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	obs := build(t, dep, &domain.LiveStateResult{OK: false, Error: "dial tcp: connection refused"}, fixtureDNS(), fixtureTLS(), now)

	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN {
		t.Fatalf("status = %s, want UNKNOWN", obs.GetStatus())
	}
	if obs.GetFreshness() != healthv1.Freshness_FRESHNESS_UNKNOWN {
		t.Fatalf("freshness = %s, want UNKNOWN", obs.GetFreshness())
	}
	if c := checkByID(obs, CheckHostPresence); c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_FAILED || c.GetReasonCode() != "ssh_unreachable" {
		t.Fatalf("host_presence = %s/%s", c.GetStatus(), c.GetReasonCode())
	}
	if c := checkByID(obs, CheckTransportReach); c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE {
		t.Fatalf("transport_reach = %s, want UNAVAILABLE", c.GetStatus())
	}
	if !obs.GetPartial() || obs.GetMissingDependencies()[0] != DependencyLiveState {
		t.Fatalf("partial/missing = %v/%v", obs.GetPartial(), obs.GetMissingDependencies())
	}
	// A nil live state (inspection not attempted) is the same verdict.
	obs = build(t, dep, nil, fixtureDNS(), fixtureTLS(), now)
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN || obs.GetFreshness() != healthv1.Freshness_FRESHNESS_UNKNOWN {
		t.Fatalf("nil live state status/freshness = %s/%s", obs.GetStatus(), obs.GetFreshness())
	}
}

// TestBuildFlagsPartialObservation [REQ:STC-P0-033] proves P16 step 10:
// absent DNS/TLS evidence is visible as partial + missing_dependencies and
// as UNAVAILABLE checks, while the verdict on the evidence that exists is
// unchanged.
func TestBuildFlagsPartialObservation(t *testing.T) {
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	dep := fixtureDeployment(domain.StatusDeployed)
	obs := build(t, dep, fixtureLiveState(inspected, "running"), nil, nil, inspected.Add(time.Second))

	if !obs.GetPartial() {
		t.Fatal("observation without DNS/TLS evidence was not flagged partial")
	}
	if got := strings.Join(obs.GetMissingDependencies(), ","); got != DependencyEdgeDNS+","+DependencyEdgeTLS {
		t.Fatalf("missing_dependencies = %q", got)
	}
	for _, id := range []string{CheckEdgeDNS, CheckEdgeTLS} {
		if c := checkByID(obs, id); c.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE {
			t.Fatalf("%s = %s, want UNAVAILABLE", id, c.GetStatus())
		}
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Fatalf("status with partial evidence = %s", obs.GetStatus())
	}
}

// TestBuildRecordStateOverridesLiveState covers failed/stopped records and
// records that were never deployed.
func TestBuildRecordStateOverridesLiveState(t *testing.T) {
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	cases := map[domain.DeploymentStatus]healthv1.HealthStatus{
		domain.StatusFailed:    healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY,
		domain.StatusStopped:   healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY,
		domain.StatusPending:   healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN,
		domain.StatusDeploying: healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN,
	}
	for status, want := range cases {
		obs := build(t, fixtureDeployment(status), fixtureLiveState(inspected, "running"), fixtureDNS(), fixtureTLS(), inspected)
		if obs.GetStatus() != want {
			t.Fatalf("record %s → %s, want %s", status, obs.GetStatus(), want)
		}
	}
}

// TestMarshalJSONUsesProtoNamesAndEnumNames pins the wire shape consumers
// decode with protojson.
func TestMarshalJSONUsesProtoNamesAndEnumNames(t *testing.T) {
	inspected := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	obs := build(t, fixtureDeployment(domain.StatusDeployed), fixtureLiveState(inspected, "running"), fixtureDNS(), fixtureTLS(), inspected)
	raw, err := MarshalResponseJSON(obs)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope["schema_version"]) != `"1"` {
		t.Fatalf("schema_version = %s", envelope["schema_version"])
	}
	var body map[string]any
	if err := json.Unmarshal(envelope["observation"], &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"deployment_id", "observed_release_digest", "observed_at", "status", "freshness", "checks", "producer_ref", "partial", "missing_dependencies"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("wire body missing %q: %s", key, envelope["observation"])
		}
	}
	if body["status"] != "HEALTH_STATUS_HEALTHY" || body["freshness"] != "FRESHNESS_CURRENT" {
		t.Fatalf("enum names = %v/%v", body["status"], body["freshness"])
	}
	var decoded healthv1.GetHealthObservationResponse
	if err := protojson.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if decoded.GetObservation().GetDeploymentId() != obs.GetDeploymentId() {
		t.Fatalf("round trip lost identity")
	}
}

func TestNormalizeDigest(t *testing.T) {
	if got := NormalizeDigest(" ABC "); got != "sha256:abc" {
		t.Fatalf("NormalizeDigest = %q", got)
	}
	if got := NormalizeDigest("sha256:abc"); got != "sha256:abc" {
		t.Fatalf("NormalizeDigest prefixed = %q", got)
	}
	if got := NormalizeDigest(""); got != "" {
		t.Fatalf("NormalizeDigest empty = %q", got)
	}
}
