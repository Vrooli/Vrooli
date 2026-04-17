package preflight

import (
	"context"
	"fmt"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"strings"
	"testing"
	"time"
)

type mapResolver struct{}

func (mapResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	_ = ctx
	if strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("empty host")
	}
	return []string{"138.197.95.182"}, nil
}

type fakeRunner struct {
	ramKB       string
	portsInUse  bool
	portProcess string
}

func (f fakeRunner) Run(ctx context.Context, cfg ssh.Config, command string, opts ssh.RunOptions) (ssh.Result, error) {
	_ = ctx
	_ = cfg
	_ = opts

	switch {
	case strings.Contains(command, "echo ok"):
		return ssh.Result{Stdout: "ok", ExitCode: 0}, nil
	case strings.Contains(command, "cat /etc/os-release"):
		return ssh.Result{Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"", ExitCode: 0}, nil
	case strings.Contains(command, "sport = :80 or sport = :443"):
		if f.portsInUse {
			process := f.portProcess
			if strings.TrimSpace(process) == "" {
				process = "caddy"
			}
			return ssh.Result{Stdout: fmt.Sprintf(`LISTEN 0 4096 *:80 *:* users:(("%s",pid=123,fd=7))`, process), ExitCode: 0}, nil
		}
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	case strings.Contains(command, "ufw status"):
		return ssh.Result{Stdout: "Status: inactive", ExitCode: 0}, nil
	case strings.Contains(command, "curl -fsS --max-time 5 https://example.com"):
		return ssh.Result{ExitCode: 0}, nil
	case strings.Contains(command, "df -Pk / | tail -n 1 | awk '{print $4}'"):
		return ssh.Result{Stdout: "9437184", ExitCode: 0}, nil
	case strings.Contains(command, "awk '/MemTotal/ {print $2}' /proc/meminfo"):
		return ssh.Result{Stdout: f.ramKB, ExitCode: 0}, nil
	case strings.Contains(command, "which curl"):
		return ssh.Result{Stdout: "/usr/bin/curl", ExitCode: 0}, nil
	case strings.Contains(command, "which git"):
		return ssh.Result{Stdout: "/usr/bin/git", ExitCode: 0}, nil
	case strings.Contains(command, "which unzip"):
		return ssh.Result{Stdout: "/usr/bin/unzip", ExitCode: 0}, nil
	case strings.Contains(command, "which tar"):
		return ssh.Result{Stdout: "/usr/bin/tar", ExitCode: 0}, nil
	case strings.Contains(command, "which jq"):
		return ssh.Result{Stdout: "/usr/bin/jq", ExitCode: 0}, nil
	case strings.Contains(command, "apt-get update --print-uris"):
		return ssh.Result{Stdout: "'http://archive.ubuntu.com/ubuntu'", ExitCode: 0}, nil
	case strings.Contains(command, "command -v docker && docker info"):
		return ssh.Result{Stdout: "/usr/bin/docker\n29.1.3", ExitCode: 0}, nil
	case strings.Contains(command, "command -v systemctl && systemctl --version"):
		return ssh.Result{Stdout: "/usr/bin/systemctl\nsystemd 255", ExitCode: 0}, nil
	case strings.Contains(command, "ps aux --no-headers | grep -E"):
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	default:
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	}
}

func testManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "138.197.95.182",
				Port:    22,
				User:    "root",
				Workdir: domain.DefaultVPSWorkdir,
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Edge: domain.ManifestEdge{
			Domain:    "vrooli.com",
			DNSPolicy: domain.DNSPolicyRequired,
			Caddy:     domain.ManifestCaddy{Enabled: true},
		},
	}
}

func TestRun_HappyPath(t *testing.T) {
	t.Parallel()

	manifest := testManifest()
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	runner := fakeRunner{ramKB: "2097152", portsInUse: false}

	resp := Run(context.Background(), manifest, dnsService, runner, RunOptions{
		Requirements: func(ctx context.Context, scenarioID string) (*ScenarioRequirements, error) {
			_ = ctx
			_ = scenarioID
			return nil, fmt.Errorf("unavailable")
		},
		PortProbe: func(ctx context.Context, host string, port int, timeout time.Duration) error {
			_ = ctx
			_ = host
			_ = port
			_ = timeout
			return nil
		},
		TLSALPNProbe: func(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error) {
			_ = ctx
			_ = host
			_ = serverName
			_ = port
			_ = timeout
			return "acme-tls/1", nil
		},
	})

	if !resp.OK {
		t.Fatalf("expected preflight OK=true, got false: %+v", resp.Checks)
	}
}

func TestRun_FailsOnLowRAMAndBusyEdgePorts(t *testing.T) {
	t.Parallel()

	manifest := testManifest()
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	runner := fakeRunner{ramKB: "500000", portsInUse: true, portProcess: "nginx"}

	resp := Run(context.Background(), manifest, dnsService, runner, RunOptions{
		Requirements: func(ctx context.Context, scenarioID string) (*ScenarioRequirements, error) {
			_ = ctx
			_ = scenarioID
			return nil, fmt.Errorf("unavailable")
		},
		PortProbe: func(ctx context.Context, host string, port int, timeout time.Duration) error {
			_ = ctx
			_ = host
			_ = port
			_ = timeout
			return nil
		},
		TLSALPNProbe: func(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error) {
			_ = ctx
			_ = host
			_ = serverName
			_ = port
			_ = timeout
			return "acme-tls/1", nil
		},
	})

	if resp.OK {
		t.Fatalf("expected preflight OK=false for low RAM/occupied ports")
	}

	failIDs := map[string]bool{}
	for _, check := range resp.Checks {
		if check.Status == domain.PreflightFail {
			failIDs[check.ID] = true
		}
	}
	if !failIDs[domain.PreflightRAMTotalID] {
		t.Fatalf("expected %s to fail, got checks: %+v", domain.PreflightRAMTotalID, resp.Checks)
	}
	if !failIDs[domain.PreflightPortsEdgeID] {
		t.Fatalf("expected %s to fail, got checks: %+v", domain.PreflightPortsEdgeID, resp.Checks)
	}
}

func TestRun_AllowsBusyEdgePortsWhenOwnedByCaddy(t *testing.T) {
	t.Parallel()

	manifest := testManifest()
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	runner := fakeRunner{ramKB: "2097152", portsInUse: true, portProcess: "caddy"}

	resp := Run(context.Background(), manifest, dnsService, runner, RunOptions{
		Requirements: func(ctx context.Context, scenarioID string) (*ScenarioRequirements, error) {
			_ = ctx
			_ = scenarioID
			return nil, fmt.Errorf("unavailable")
		},
		PortProbe: func(ctx context.Context, host string, port int, timeout time.Duration) error {
			_ = ctx
			_ = host
			_ = port
			_ = timeout
			return nil
		},
		TLSALPNProbe: func(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error) {
			_ = ctx
			_ = host
			_ = serverName
			_ = port
			_ = timeout
			return "acme-tls/1", nil
		},
	})

	if !resp.OK {
		t.Fatalf("expected preflight OK=true when caddy owns edge ports, got false: %+v", resp.Checks)
	}
}

func TestRun_FailsWhenAnalyzerRequirementExceedsStaticFloor(t *testing.T) {
	t.Parallel()

	manifest := testManifest()
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	runner := fakeRunner{ramKB: "2097152", portsInUse: false} // 2 GiB

	resp := Run(context.Background(), manifest, dnsService, runner, RunOptions{
		Requirements: func(ctx context.Context, scenarioID string) (*ScenarioRequirements, error) {
			_ = ctx
			_ = scenarioID
			return &ScenarioRequirements{
				RAMKB:      3 * 1024 * 1024, // 3 GiB requirement from dependency graph
				DiskKB:     0,
				CPUCores:   2,
				Tier:       "tier-4-saas",
				Source:     "scenario-dependency-analyzer",
				Confidence: "medium",
			}, nil
		},
		PortProbe: func(ctx context.Context, host string, port int, timeout time.Duration) error {
			_ = ctx
			_ = host
			_ = port
			_ = timeout
			return nil
		},
		TLSALPNProbe: func(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error) {
			_ = ctx
			_ = host
			_ = serverName
			_ = port
			_ = timeout
			return "acme-tls/1", nil
		},
	})

	if resp.OK {
		t.Fatalf("expected preflight OK=false when graph RAM requirement is unmet")
	}

	for _, check := range resp.Checks {
		if check.ID != domain.PreflightRAMTotalID {
			continue
		}
		if check.Status != domain.PreflightFail {
			t.Fatalf("expected %s to fail, got status=%s", domain.PreflightRAMTotalID, check.Status)
		}
		if check.Data["required_by_graph_ram_kb"] != "3145728" {
			t.Fatalf("expected graph RAM requirement in check data, got %+v", check.Data)
		}
		return
	}

	t.Fatalf("expected %s check in response", domain.PreflightRAMTotalID)
}
