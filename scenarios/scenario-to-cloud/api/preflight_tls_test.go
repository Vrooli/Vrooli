package main

import (
	"context"
	"testing"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/sshadapter"
	"scenario-to-cloud/vps/preflight"
)

func TestVPSPreflightProxyModeRequiresDNS01(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root"},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge: domain.ManifestEdge{
			Domain:    "example.com",
			DNSPolicy: domain.DNSPolicyRequired,
			Caddy:     domain.ManifestCaddy{Enabled: true},
		},
	}

	resp := preflight.Run(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"example.com":           {"104.16.0.1"},
			"www.example.com":       {"104.16.0.2"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		preflightReach(&FakeSSHRunner{Responses: preflightObservationResponses()}),
		domain.TargetRefFromManifest(manifest),
		preflight.RunOptions{
			PortProbe: func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _, _ string, _ int, _ time.Duration) (string, error) {
				return "acme-tls/1", nil
			},
		},
	)

	var proxyCheck *domain.PreflightCheck
	for i := range resp.Checks {
		if resp.Checks[i].ID == domain.PreflightDNSProxyModeID {
			proxyCheck = &resp.Checks[i]
			break
		}
	}
	if proxyCheck == nil {
		t.Fatalf("expected dns_proxy_mode check")
	}
	if proxyCheck.Status != domain.PreflightFail {
		t.Fatalf("expected dns_proxy_mode to fail, got: %+v", proxyCheck)
	}
	if resp.OK {
		t.Fatalf("expected preflight to fail when proxy mode check fails")
	}
}

func TestVPSPreflightProxyModeWarnPolicy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root"},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge: domain.ManifestEdge{
			Domain:    "example.com",
			DNSPolicy: domain.DNSPolicyWarn,
			Caddy:     domain.ManifestCaddy{Enabled: true},
		},
	}

	resp := preflight.Run(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"example.com":           {"104.16.0.1"},
			"www.example.com":       {"104.16.0.2"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		preflightReach(&FakeSSHRunner{Responses: preflightObservationResponses()}),
		domain.TargetRefFromManifest(manifest),
		preflight.RunOptions{
			PortProbe: func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _, _ string, _ int, _ time.Duration) (string, error) {
				return "acme-tls/1", nil
			},
		},
	)

	var proxyCheck *domain.PreflightCheck
	for i := range resp.Checks {
		if resp.Checks[i].ID == domain.PreflightDNSProxyModeID {
			proxyCheck = &resp.Checks[i]
			break
		}
	}
	if proxyCheck == nil {
		t.Fatalf("expected dns_proxy_mode check")
	}
	if proxyCheck.Status != domain.PreflightWarn {
		t.Fatalf("expected dns_proxy_mode to warn, got: %+v", proxyCheck)
	}
	if !resp.OK {
		t.Fatalf("expected preflight to pass when proxy mode check is warn")
	}
}

func TestVPSPreflightALPNWarnOnWrongProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root"},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge: domain.ManifestEdge{
			Domain:    "example.com",
			DNSPolicy: domain.DNSPolicyRequired,
			Caddy:     domain.ManifestCaddy{Enabled: true},
		},
	}

	resp := preflight.Run(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"example.com":           {"203.0.113.10"},
			"www.example.com":       {"203.0.113.10"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		preflightReach(&FakeSSHRunner{Responses: preflightObservationResponses()}),
		domain.TargetRefFromManifest(manifest),
		preflight.RunOptions{
			PortProbe:    func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _, _ string, _ int, _ time.Duration) (string, error) { return "h2", nil },
		},
	)

	var alpnCheck *domain.PreflightCheck
	for i := range resp.Checks {
		if resp.Checks[i].ID == domain.PreflightTLSALPNID {
			alpnCheck = &resp.Checks[i]
			break
		}
	}
	if alpnCheck == nil {
		t.Fatalf("expected tls_alpn_compat check")
	}
	if alpnCheck.Status != domain.PreflightWarn {
		t.Fatalf("expected tls_alpn_compat to warn, got: %+v", alpnCheck)
	}
}

func preflightObservationResponses() map[string]sshadapter.Result {
	quoted := func(argv ...string) string { return sshadapter.ObservationCommand(argv) }
	return map[string]sshadapter.Result{
		quoted("uname", "-s"):            {ExitCode: 0, Stdout: "Linux"},
		quoted("cat", "/etc/os-release"): {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
		quoted("ss", "-ltnpH", "(", "sport", "=", ":80", "or", "sport", "=", ":443", ")"): {ExitCode: 0, Stdout: ""},
		quoted("cat", "/etc/ufw/ufw.conf"):                                                {ExitCode: 0, Stdout: "ENABLED=no\n"},
		quoted("df", "-Pk", "/"):                                                          {ExitCode: 0, Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 40000000 30000001 9999999 76% /"},
		quoted("grep", "MemTotal", "/proc/meminfo"):                                       {ExitCode: 0, Stdout: "MemTotal:        2097152 kB"},
		quoted("find", "/usr/bin", "/usr/local/bin", "/bin", "/usr/sbin", "/sbin", "/snap/bin", "-maxdepth", "1", "(", "-name", "curl", "-o", "-name", "git", "-o", "-name", "unzip", "-o", "-name", "tar", "-o", "-name", "jq", "-o", "-name", "apt-get", "-o", "-name", "docker", "-o", "-name", "systemctl", ")"): {ExitCode: 0, Stdout: "/usr/bin/curl\n/usr/bin/git\n/usr/bin/unzip\n/bin/tar\n/usr/bin/jq\n/usr/bin/apt-get\n/usr/bin/docker\n/usr/bin/systemctl"},
		quoted("stat", "--", "/var/run/docker.sock"):               {ExitCode: 0, Stdout: "  File: /var/run/docker.sock"},
		quoted("stat", "--", "/run/systemd/system"):                {ExitCode: 0, Stdout: "  File: /run/systemd/system"},
		quoted("pgrep", "-a", "-f", "landing-page-business-suite"): {ExitCode: 1, Stdout: ""},
	}
}

// preflightReach runs preflight through the bounded SSH adapter over a fake
// runner, exactly as the server does when no router is wired.
func preflightReach(runner sshadapter.Runner) reach.Reach {
	return &sshadapter.Adapter{Runner: runner, Config: func(_ context.Context, target identity.TargetRef) (sshadapter.ConnectionConfig, error) {
		return sshadapter.NewConfig(target.Locator.Host, target.Locator.Port, target.Locator.User, ""), nil
	}}
}
