package main

import (
	"context"
	"testing"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
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
		&FakeSSHRunner{Responses: basePreflightResponses()},
		preflight.RunOptions{
			PortProbe: func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _ string, _ string, _ int, _ time.Duration) (string, error) {
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
		&FakeSSHRunner{Responses: basePreflightResponses()},
		preflight.RunOptions{
			PortProbe: func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _ string, _ string, _ int, _ time.Duration) (string, error) {
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
		&FakeSSHRunner{Responses: basePreflightResponses()},
		preflight.RunOptions{
			PortProbe:    func(_ context.Context, _ string, _ int, _ time.Duration) error { return nil },
			TLSALPNProbe: func(_ context.Context, _ string, _ string, _ int, _ time.Duration) (string, error) { return "h2", nil },
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

func basePreflightResponses() map[string]ssh.Result {
	return map[string]ssh.Result{
		"echo ok":             {ExitCode: 0, Stdout: "ok"},
		"cat /etc/os-release": {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
		"ss -ltnH '( sport = :80 or sport = :443 )'": {ExitCode: 0, Stdout: ""},
		"ufw status": {ExitCode: 0, Stdout: "Status: inactive\n"},
		"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
		"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
		"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
		"which curl":  {ExitCode: 0, Stdout: "/usr/bin/curl"},
		"which git":   {ExitCode: 0, Stdout: "/usr/bin/git"},
		"which unzip": {ExitCode: 0, Stdout: "/usr/bin/unzip"},
		"which tar":   {ExitCode: 0, Stdout: "/bin/tar"},
		"which jq":    {ExitCode: 0, Stdout: "/usr/bin/jq"},
		"apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log": {ExitCode: 0, Stdout: ""},
	}
}
