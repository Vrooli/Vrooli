package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/dns"
)

func TestVPSPreflightHappyPath(t *testing.T) {
	// [REQ:STC-P0-003] VPS readiness validation
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := CloudManifest{
		Version:     "1.0.0",
		Environment: "prod",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host: "203.0.113.10",
				Port: 22,
				User: "root",
			},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
			Resources: []string{},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	resp := RunVPSPreflight(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"203.0.113.10":          {"203.0.113.10"},
			"example.com":           {"203.0.113.10"},
			"www.example.com":       {"203.0.113.10"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		&FakeSSHRunner{Responses: map[string]SSHResult{
			"echo ok":             {ExitCode: 0, Stdout: "ok"},
			"cat /etc/os-release": {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
			"ss -ltnH '( sport = :80 or sport = :443 )'":            {ExitCode: 0, Stdout: ""},
			"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
			"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
			"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
			// Bootstrap prerequisite checks
			"which curl":  {ExitCode: 0, Stdout: "/usr/bin/curl"},
			"which git":   {ExitCode: 0, Stdout: "/usr/bin/git"},
			"which unzip": {ExitCode: 0, Stdout: "/usr/bin/unzip"},
			"which tar":   {ExitCode: 0, Stdout: "/bin/tar"},
			"which jq":    {ExitCode: 0, Stdout: "/usr/bin/jq"},
			"apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log": {ExitCode: 0, Stdout: ""},
		}},
	)

	if !resp.OK {
		t.Fatalf("expected OK, got: %+v", resp)
	}
	if len(resp.Checks) < 15 {
		t.Fatalf("expected at least 15 checks (original + prerequisites), got: %d checks: %+v", len(resp.Checks), resp.Checks)
	}
}

func TestVPSPreflightDNSErrorIsActionable(t *testing.T) {
	// [REQ:STC-P0-009] preflight errors are actionable
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS:  &ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root"},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	resp := RunVPSPreflight(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"203.0.113.10":          {"203.0.113.10"},
			"example.com":           {"198.51.100.5"},
			"www.example.com":       {"198.51.100.5"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		&FakeSSHRunner{Responses: map[string]SSHResult{
			"echo ok":             {ExitCode: 0, Stdout: "ok"},
			"cat /etc/os-release": {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
			"ss -ltnH '( sport = :80 or sport = :443 )'":            {ExitCode: 0, Stdout: ""},
			"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
			"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
			"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
			// Bootstrap prerequisite checks
			"which curl":  {ExitCode: 0, Stdout: "/usr/bin/curl"},
			"which git":   {ExitCode: 0, Stdout: "/usr/bin/git"},
			"which unzip": {ExitCode: 0, Stdout: "/usr/bin/unzip"},
			"which tar":   {ExitCode: 0, Stdout: "/bin/tar"},
			"which jq":    {ExitCode: 0, Stdout: "/usr/bin/jq"},
			"apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log": {ExitCode: 0, Stdout: ""},
		}},
	)

	var found bool
	for _, c := range resp.Checks {
		if c.ID == preflightDNSEdgeApex {
			found = true
			if c.Status != PreflightFail {
				t.Fatalf("expected dns_edge_apex to fail, got: %+v", c)
			}
			if !strings.Contains(c.Hint, "Update DNS") {
				t.Fatalf("expected actionable hint, got: %q", c.Hint)
			}
		}
	}
	if !found {
		t.Fatalf("expected dns_edge_apex check")
	}
}

func TestVPSPreflightDNSPolicyWarnDowngradesFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS:  &ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root"},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge: ManifestEdge{
			Domain:    "example.com",
			DNSPolicy: DNSPolicyWarn,
			Caddy:     ManifestCaddy{Enabled: true},
		},
	}

	resp := RunVPSPreflight(
		ctx,
		manifest,
		dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{
			"203.0.113.10":          {"203.0.113.10"},
			"example.com":           {"198.51.100.5"},
			"www.example.com":       {"198.51.100.5"},
			"do-origin.example.com": {"203.0.113.10"},
		}}),
		&FakeSSHRunner{Responses: map[string]SSHResult{
			"echo ok":             {ExitCode: 0, Stdout: "ok"},
			"cat /etc/os-release": {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
			"ss -ltnH '( sport = :80 or sport = :443 )'":            {ExitCode: 0, Stdout: ""},
			"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
			"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
			"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
			"which curl":  {ExitCode: 0, Stdout: "/usr/bin/curl"},
			"which git":   {ExitCode: 0, Stdout: "/usr/bin/git"},
			"which unzip": {ExitCode: 0, Stdout: "/usr/bin/unzip"},
			"which tar":   {ExitCode: 0, Stdout: "/bin/tar"},
			"which jq":    {ExitCode: 0, Stdout: "/usr/bin/jq"},
			"apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log": {ExitCode: 0, Stdout: ""},
		}},
	)

	var pointsCheck *PreflightCheck
	for i := range resp.Checks {
		if resp.Checks[i].ID == preflightDNSEdgeApex {
			pointsCheck = &resp.Checks[i]
			break
		}
	}
	if pointsCheck == nil {
		t.Fatalf("expected dns_edge_apex check")
	}
	if pointsCheck.Status != PreflightWarn {
		t.Fatalf("expected dns_edge_apex to warn, got: %+v", pointsCheck)
	}
	if !resp.OK {
		t.Fatalf("expected OK when DNS policy is warn, got: %+v", resp)
	}
}
