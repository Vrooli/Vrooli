package vps

import (
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/tlsinfo"
	"strings"
	"testing"
	"time"
)

func newTestDeployment(status domain.DeploymentStatus) *domain.Deployment {
	now := time.Now()
	return &domain.Deployment{
		ID:             "dep-123",
		Name:           "my-app",
		ScenarioID:     "landing-page",
		Status:         status,
		LastDeployedAt: &now,
	}
}

func newTestManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "landing-page"},
		Target: domain.ManifestTarget{
			VPS: &domain.ManifestVPS{Host: "1.2.3.4"},
		},
		Dependencies: domain.ManifestDependencies{
			Resources: []string{"postgres", "redis"},
		},
		Edge: domain.ManifestEdge{
			Domain: "example.com",
		},
	}
}

func newHealthyLiveState() *domain.LiveStateResult {
	return &domain.LiveStateResult{
		OK: true,
		System: &domain.SystemState{
			SSH: domain.SSHHealth{
				Connected:         true,
				LatencyMs:         45,
				AuthMode:          string(sshidentity.AuthModeExplicitKey),
				VerificationState: string(sshidentity.VerificationAuthorized),
			},
			CPU: domain.CPUInfo{
				Cores:        4,
				UsagePercent: 12.3,
			},
			Memory: domain.MemoryInfo{
				TotalMB:      4096,
				UsedMB:       1024,
				UsagePercent: 25,
			},
			Disk: domain.DiskInfo{
				TotalGB:      200,
				UsedGB:       42,
				UsagePercent: 21,
			},
		},
		Processes: &domain.ProcessState{
			Scenarios: []domain.ScenarioProcess{
				{
					ID:     "landing-page",
					Status: "running",
					PID:    1234,
					Resources: domain.ProcessResources{
						CPUPercent: 2.1,
						MemoryMB:   128,
					},
				},
			},
			Resources: []domain.ResourceProcess{
				{ID: "postgres", Status: "running", PID: 5678, Port: 5432},
				{ID: "redis", Status: "running", PID: 9012, Port: 6379},
			},
		},
		Caddy: &domain.CaddyState{
			Running: true,
			Domain:  "example.com",
		},
	}
}

func newExplicitIdentity(state sshidentity.VerificationState) sshidentity.DeploymentSSHIdentity {
	return sshidentity.DeploymentSSHIdentity{
		KeyPath:              "~/.ssh/id_ed25519",
		PublicKeyFingerprint: "SHA256:test",
		AuthMode:             sshidentity.AuthModeExplicitKey,
		VerificationState:    state,
	}
}

func newHealthyDNSEval() *dns.Evaluation {
	return &dns.Evaluation{
		EdgeDomain: "example.com",
		VPS:        domain.DNSLookupResult{Host: "1.2.3.4", IPs: []string{"1.2.3.4"}},
		Statuses: []dns.DomainStatus{
			{Role: "apex", Host: "example.com", PointsToVPS: true, Lookup: domain.DNSLookupResult{IPs: []string{"1.2.3.4"}}},
			{Role: "origin", Host: "do-origin.example.com", PointsToVPS: true, Lookup: domain.DNSLookupResult{IPs: []string{"1.2.3.4"}}},
		},
	}
}

func newHealthyTLSSnapshot() *tlsinfo.Snapshot {
	return &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{
			Valid:         true,
			Issuer:        "Let's Encrypt",
			DaysRemaining: 60,
			NotAfter:      "Mar 15 12:00:00 2026 UTC",
		},
		ALPN: tlsinfo.ALPNCheck{
			Status:  tlsinfo.ALPNPass,
			Message: "TLS-ALPN acme-tls/1 negotiated successfully",
		},
	}
}

func findCheck(resp domain.HealthResponse, category, id string) *domain.HealthCheck {
	for _, sec := range resp.Sections {
		if sec.Category != category {
			continue
		}
		for i := range sec.Checks {
			if sec.Checks[i].ID == id {
				return &sec.Checks[i]
			}
		}
	}
	return nil
}

func TestComputeHealth_Healthy(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthHealthy {
		t.Errorf("expected health=healthy, got %s", resp.Health)
	}
	if resp.DeploymentID != "dep-123" {
		t.Errorf("expected deployment_id=dep-123, got %s", resp.DeploymentID)
	}
	if resp.Domain != "example.com" {
		t.Errorf("expected domain=example.com, got %s", resp.Domain)
	}
	if resp.Host != "1.2.3.4" {
		t.Errorf("expected host=1.2.3.4, got %s", resp.Host)
	}
	if len(resp.Sections) != 6 {
		t.Errorf("expected 6 sections, got %d", len(resp.Sections))
	}
	if len(resp.Recommendations) != 0 {
		t.Errorf("expected 0 recommendations, got %d", len(resp.Recommendations))
	}
}

func TestComputeHealth_SSHAgentIsWarn(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	liveState.System.SSH.AuthMode = string(sshidentity.AuthModeAgent)
	liveState.System.SSH.VerificationState = string(sshidentity.VerificationUnknown)
	identity := sshidentity.DeploymentSSHIdentity{
		AuthMode:          sshidentity.AuthModeAgent,
		VerificationState: sshidentity.VerificationUnknown,
	}

	resp := ComputeHealth(dep, manifest, identity, liveState, newHealthyDNSEval(), newHealthyTLSSnapshot(), nil)

	var keyAuthCheck *domain.HealthCheck
	for _, sec := range resp.Sections {
		if sec.Category != "ssh" {
			continue
		}
		for i := range sec.Checks {
			if sec.Checks[i].ID == "ssh_key_auth" {
				keyAuthCheck = &sec.Checks[i]
				break
			}
		}
	}
	if keyAuthCheck == nil {
		t.Fatal("ssh_key_auth check not found")
	}
	if keyAuthCheck.Status != domain.HealthCheckWarn {
		t.Fatalf("ssh_key_auth status=%q, want %q", keyAuthCheck.Status, domain.HealthCheckWarn)
	}
	if got := keyAuthCheck.Details["auth_mode"]; got != string(sshidentity.AuthModeAgent) {
		t.Fatalf("ssh_key_auth details.auth_mode=%q, want %q", got, sshidentity.AuthModeAgent)
	}

	found := false
	for _, rec := range resp.Recommendations {
		if rec.Category != "ssh" {
			continue
		}
		if rec.Command == "scenario-to-cloud ssh bootstrap 1.2.3.4 --user root --non-interactive" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected bootstrap recommendation for unpinned ssh auth")
	}
}

func TestComputeHealth_ExplicitUnauthorizedIsFail(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	liveState.System.SSH.AuthMode = string(sshidentity.AuthModeExplicitKey)
	liveState.System.SSH.VerificationState = string(sshidentity.VerificationUnauthorized)
	identity := newExplicitIdentity(sshidentity.VerificationUnauthorized)

	resp := ComputeHealth(dep, manifest, identity, liveState, newHealthyDNSEval(), newHealthyTLSSnapshot(), nil)

	var keyAuthCheck *domain.HealthCheck
	for _, sec := range resp.Sections {
		if sec.Category != "ssh" {
			continue
		}
		for i := range sec.Checks {
			if sec.Checks[i].ID == "ssh_key_auth" {
				keyAuthCheck = &sec.Checks[i]
				break
			}
		}
	}
	if keyAuthCheck == nil {
		t.Fatal("ssh_key_auth check not found")
	}
	if keyAuthCheck.Status != domain.HealthCheckFail {
		t.Fatalf("ssh_key_auth status=%q, want %q", keyAuthCheck.Status, domain.HealthCheckFail)
	}
}

func TestComputeHealth_Degraded_TLSExpiring(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{
			Valid:         true,
			Issuer:        "Let's Encrypt",
			DaysRemaining: 22,
			NotAfter:      "Feb 28 12:00:00 2026 UTC",
		},
		ALPN: tlsinfo.ALPNCheck{Status: tlsinfo.ALPNPass},
	}

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthDegraded {
		t.Errorf("expected health=degraded, got %s", resp.Health)
	}
	if len(resp.Recommendations) == 0 {
		t.Fatal("expected at least 1 recommendation for expiring TLS")
	}
	found := false
	for _, r := range resp.Recommendations {
		if r.Category == "tls" {
			found = true
			if r.Command == "" {
				t.Error("expected recommendation to have a command")
			}
		}
	}
	if !found {
		t.Error("expected a TLS recommendation")
	}
}

func TestComputeHealth_ALPNWarnHealthyCertIsInformational(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{
			Valid:         true,
			Issuer:        "Let's Encrypt",
			DaysRemaining: 60,
			NotAfter:      "Apr 15 12:00:00 2026 UTC",
		},
		ALPN: tlsinfo.ALPNCheck{
			Status:  tlsinfo.ALPNWarn,
			Message: "TLS-ALPN probe failed",
			Error:   "remote error: tls: internal error",
		},
	}

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthHealthy {
		t.Fatalf("expected healthy when ALPN warns but cert is healthy, got %s", resp.Health)
	}
	alpnCheck := findCheck(resp, "tls", "tls_alpn")
	if alpnCheck == nil {
		t.Fatal("tls_alpn check not found")
	}
	if alpnCheck.Status != domain.HealthCheckPass {
		t.Fatalf("tls_alpn status=%q, want %q", alpnCheck.Status, domain.HealthCheckPass)
	}
	for _, rec := range resp.Recommendations {
		if rec.Category == "tls" && strings.Contains(strings.ToLower(rec.Summary), "alpn") {
			t.Fatalf("unexpected ALPN recommendation for healthy cert: %+v", rec)
		}
	}
}

func TestComputeHealth_ALPNWarnWithinRenewalWindowIsWarn(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{
			Valid:         true,
			Issuer:        "Let's Encrypt",
			DaysRemaining: 20,
			NotAfter:      "Feb 28 12:00:00 2026 UTC",
		},
		ALPN: tlsinfo.ALPNCheck{
			Status:  tlsinfo.ALPNWarn,
			Message: "TLS-ALPN probe failed",
		},
	}

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthDegraded {
		t.Fatalf("expected degraded when ALPN warns in renewal window, got %s", resp.Health)
	}
	alpnCheck := findCheck(resp, "tls", "tls_alpn")
	if alpnCheck == nil {
		t.Fatal("tls_alpn check not found")
	}
	if alpnCheck.Status != domain.HealthCheckWarn {
		t.Fatalf("tls_alpn status=%q, want %q", alpnCheck.Status, domain.HealthCheckWarn)
	}
	found := false
	for _, rec := range resp.Recommendations {
		if rec.Category == "tls" && strings.Contains(strings.ToLower(rec.Summary), "alpn") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ALPN recommendation in renewal window")
	}
}

func TestComputeHealth_ALPNWarnNearExpiryIsFail(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := &tlsinfo.Snapshot{
		Probe: tlsinfo.ProbeResult{
			Valid:         true,
			Issuer:        "Let's Encrypt",
			DaysRemaining: 10,
			NotAfter:      "Feb 18 12:00:00 2026 UTC",
		},
		ALPN: tlsinfo.ALPNCheck{
			Status:  tlsinfo.ALPNWarn,
			Message: "TLS-ALPN probe failed",
		},
	}

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthUnhealthy {
		t.Fatalf("expected unhealthy when ALPN fails near expiry, got %s", resp.Health)
	}
	alpnCheck := findCheck(resp, "tls", "tls_alpn")
	if alpnCheck == nil {
		t.Fatal("tls_alpn check not found")
	}
	if alpnCheck.Status != domain.HealthCheckFail {
		t.Fatalf("tls_alpn status=%q, want %q", alpnCheck.Status, domain.HealthCheckFail)
	}
}

func TestComputeHealth_Unhealthy_ProcessMissing(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	// Remove redis from running processes
	liveState.Processes.Resources = []domain.ResourceProcess{
		{ID: "postgres", Status: "running", PID: 5678, Port: 5432},
	}
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthUnhealthy {
		t.Errorf("expected health=unhealthy, got %s", resp.Health)
	}

	// Should have a process recommendation
	found := false
	for _, r := range resp.Recommendations {
		if r.Category == "processes" {
			found = true
		}
	}
	if !found {
		t.Error("expected a process restart recommendation")
	}
}

func TestComputeHealth_Failed(t *testing.T) {
	errMsg := "setup script failed"
	dep := newTestDeployment(domain.StatusFailed)
	dep.ErrorMessage = &errMsg
	manifest := newTestManifest()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationUnknown), nil, nil, nil, nil)

	if resp.Health != domain.HealthFailed {
		t.Errorf("expected health=failed, got %s", resp.Health)
	}
}

func TestComputeHealth_Stopped(t *testing.T) {
	dep := newTestDeployment(domain.StatusStopped)
	manifest := newTestManifest()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationUnknown), nil, nil, nil, nil)

	if resp.Health != domain.HealthStopped {
		t.Errorf("expected health=stopped, got %s", resp.Health)
	}
	// Should recommend starting
	found := false
	for _, r := range resp.Recommendations {
		if r.Category == "deployment" {
			found = true
		}
	}
	if !found {
		t.Error("expected a deployment start recommendation")
	}
}

func TestComputeHealth_Unknown_SSHUnreachable(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := &domain.LiveStateResult{
		OK:    false,
		Error: "ssh: connect to host 1.2.3.4 port 22: Connection timed out",
	}

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationUnknown), liveState, nil, nil, nil)

	if resp.Health != domain.HealthUnknown {
		t.Errorf("expected health=unknown, got %s", resp.Health)
	}
}

func TestComputeHealth_Starting(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeploying)
	manifest := newTestManifest()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationUnknown), nil, nil, nil, nil)

	if resp.Health != domain.HealthStarting {
		t.Errorf("expected health=starting, got %s", resp.Health)
	}
}

func TestComputeHealth_SystemWarnings(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	liveState.System.CPU.UsagePercent = 92
	liveState.System.Memory.UsagePercent = 88
	liveState.System.Disk.UsagePercent = 82
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthDegraded {
		t.Errorf("expected health=degraded, got %s", resp.Health)
	}
	// Should have system recommendations
	if len(resp.Recommendations) < 3 {
		t.Errorf("expected at least 3 recommendations for high system usage, got %d", len(resp.Recommendations))
	}
}

func TestComputeHealth_SectionCounts(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	for _, sec := range resp.Sections {
		total := sec.PassCount + sec.WarnCount + sec.FailCount + sec.ErrorCount
		skipCount := 0
		for _, c := range sec.Checks {
			if c.Status == domain.HealthCheckSkip {
				skipCount++
			}
		}
		if total+skipCount != len(sec.Checks) {
			t.Errorf("section %s: count mismatch — pass=%d warn=%d fail=%d error=%d skip=%d total checks=%d",
				sec.Category, sec.PassCount, sec.WarnCount, sec.FailCount, sec.ErrorCount, skipCount, len(sec.Checks))
		}
	}
}

func TestComputeHealth_CPUSpikeWithoutLoadPressureIsWarning(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	liveState.System.CPU.Cores = 1
	liveState.System.CPU.UsagePercent = 99
	liveState.System.CPU.LoadAverage = []float64{0.20, 0.25, 0.30}
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthDegraded {
		t.Fatalf("expected degraded for transient CPU spike, got %s", resp.Health)
	}
}

func TestComputeHealth_CPUSustainedHighLoadFails(t *testing.T) {
	dep := newTestDeployment(domain.StatusDeployed)
	manifest := newTestManifest()
	liveState := newHealthyLiveState()
	liveState.System.CPU.Cores = 1
	liveState.System.CPU.UsagePercent = 99
	liveState.System.CPU.LoadAverage = []float64{1.10, 1.30, 1.25}
	dnsEval := newHealthyDNSEval()
	tlsSnap := newHealthyTLSSnapshot()

	resp := ComputeHealth(dep, manifest, newExplicitIdentity(sshidentity.VerificationAuthorized), liveState, dnsEval, tlsSnap, nil)

	if resp.Health != domain.HealthUnhealthy {
		t.Fatalf("expected unhealthy for sustained CPU pressure, got %s", resp.Health)
	}
}
