package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeResolver struct {
	hosts map[string][]string
	err   error
}

func (f fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	ips, ok := f.hosts[host]
	if !ok {
		return nil, errors.New("not found")
	}
	return ips, nil
}

type fakeSSHRunner struct {
	responses map[string]SSHResult
	errs      map[string]error
}

func (f fakeSSHRunner) Run(_ context.Context, _ SSHConfig, command string) (SSHResult, error) {
	if err, ok := f.errs[command]; ok {
		return SSHResult{ExitCode: 255}, err
	}
	if res, ok := f.responses[command]; ok {
		return res, nil
	}
	return SSHResult{ExitCode: 127, Stderr: "unknown command: " + command}, errors.New("unknown command")
}

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
		Ports:  ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	resp := RunVPSPreflight(
		ctx,
		manifest,
		fakeResolver{hosts: map[string][]string{
			"203.0.113.10": {"203.0.113.10"},
			"example.com":  {"203.0.113.10"},
		}},
		fakeSSHRunner{responses: map[string]SSHResult{
			"echo ok":                       {ExitCode: 0, Stdout: "ok"},
			"cat /etc/os-release":           {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
			"ss -ltnH '( sport = :80 or sport = :443 )'": {ExitCode: 0, Stdout: ""},
			"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
			"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
			"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
		}},
	)

	if !resp.OK {
		t.Fatalf("expected OK, got: %+v", resp)
	}
	if len(resp.Checks) < 6 {
		t.Fatalf("expected checks, got: %+v", resp.Checks)
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
		Ports:  ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	resp := RunVPSPreflight(
		ctx,
		manifest,
		fakeResolver{hosts: map[string][]string{
			"203.0.113.10": {"203.0.113.10"},
			"example.com":  {"198.51.100.5"},
		}},
		fakeSSHRunner{responses: map[string]SSHResult{
			"echo ok":             {ExitCode: 0, Stdout: "ok"},
			"cat /etc/os-release": {ExitCode: 0, Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"},
			"ss -ltnH '( sport = :80 or sport = :443 )'": {ExitCode: 0, Stdout: ""},
			"curl -fsS --max-time 5 https://example.com >/dev/null": {ExitCode: 0, Stdout: ""},
			"df -Pk / | tail -n 1 | awk '{print $4}'":               {ExitCode: 0, Stdout: "9999999"},
			"awk '/MemTotal/ {print $2}' /proc/meminfo":             {ExitCode: 0, Stdout: "2097152"},
		}},
	)

	var found bool
	for _, c := range resp.Checks {
		if c.ID == "dns_points_to_vps" {
			found = true
			if c.Status != PreflightFail {
				t.Fatalf("expected dns_points_to_vps to fail, got: %+v", c)
			}
			if !strings.Contains(c.Hint, "Update DNS") {
				t.Fatalf("expected actionable hint, got: %q", c.Hint)
			}
		}
	}
	if !found {
		t.Fatalf("expected dns_points_to_vps check")
	}
}

