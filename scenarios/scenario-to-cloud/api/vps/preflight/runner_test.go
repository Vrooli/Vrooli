package preflight

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/reachtest"
)

type mapResolver struct{}

func (mapResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("empty host")
	}
	return []string{"138.197.95.182"}, nil
}

// healthyHost scripts a bare Ubuntu host answering every observation program
// preflight issues. Keys are argv strings (reachtest.Key), never shell.
func healthyHost(ramKB string, portsInUse bool, portProcess string) *reachtest.Scripted {
	answer := func(stdout string) reachtest.Answer { return reachtest.Answer{Result: reach.Result{Stdout: stdout}} }
	edge := ""
	if portsInUse {
		if portProcess == "" {
			portProcess = "caddy"
		}
		edge = fmt.Sprintf(`LISTEN 0 4096 *:80 *:* users:(("%s",pid=123,fd=7))`, portProcess)
	}
	return &reachtest.Scripted{
		Strict: true,
		Answers: map[string]reachtest.Answer{
			"uname -s":            answer("Linux"),
			"sudo -n -l":          answer("User root may run the following commands..."),
			"cat /etc/os-release": answer("ID=ubuntu\nVERSION_ID=\"24.04\""),
			"ss -ltnpH ( sport = :80 or sport = :443 )": answer(edge),
			"cat /etc/ufw/ufw.conf":                     answer("ENABLED=no"),
			"df -Pk /":                                  answer("Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 40000000 30562816 9437184 77% /"),
			"grep MemTotal /proc/meminfo":               answer("MemTotal:       " + ramKB + " kB"),
			"stat -- /var/run/docker.sock":              answer("  File: /var/run/docker.sock"),
			"stat -- /run/systemd/system":               answer("  File: /run/systemd/system"),
			"pgrep -a -f landing-page-business-suite":   {Result: reach.Result{ExitCode: 1}},
		},
		Prefixes: map[string]reachtest.Answer{
			"find /usr/bin": answer("/usr/bin/curl\n/usr/bin/git\n/usr/bin/unzip\n/usr/bin/tar\n/usr/bin/jq\n/usr/bin/apt-get\n/usr/bin/docker\n/usr/bin/systemctl"),
		},
	}
}

func testManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "138.197.95.182", Port: 22, User: "root", Workdir: domain.DefaultVPSWorkdir},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Edge: domain.ManifestEdge{
			Domain:    "vrooli.com",
			DNSPolicy: domain.DNSPolicyRequired,
			Caddy:     domain.ManifestCaddy{Enabled: true},
		},
	}
}

func testOptions(requirements ScenarioRequirementsFetcher) RunOptions {
	if requirements == nil {
		requirements = func(context.Context, string) (*ScenarioRequirements, error) { return nil, fmt.Errorf("unavailable") }
	}
	return RunOptions{
		Requirements: requirements,
		PortProbe:    func(context.Context, string, int, time.Duration) error { return nil },
		TLSALPNProbe: func(context.Context, string, string, int, time.Duration) (string, error) { return "acme-tls/1", nil },
	}
}

func runPreflight(t *testing.T, host *reachtest.Scripted, opts RunOptions) domain.PreflightResponse {
	t.Helper()
	manifest := testManifest()
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	return Run(context.Background(), manifest, dnsService, host, domain.TargetRefFromManifest(manifest), opts)
}

func runPreflightAsUser(t *testing.T, host *reachtest.Scripted, user string, opts RunOptions) domain.PreflightResponse {
	t.Helper()
	manifest := testManifest()
	manifest.Target.VPS.User = user
	dnsService := dns.NewService(mapResolver{}, dns.WithTimeout(2*time.Second))
	return Run(context.Background(), manifest, dnsService, host, domain.TargetRefFromManifest(manifest), opts)
}

func checkByID(resp domain.PreflightResponse, id string) (domain.PreflightCheck, bool) {
	for _, c := range resp.Checks {
		if c.ID == id {
			return c, true
		}
	}
	return domain.PreflightCheck{}, false
}

// [REQ:STC-P0-024] Preflight inspects a bare host only through read-only
// observation programs: no shell string, no vrooli verb, nothing effectful.
func TestRun_HappyPathUsesOnlyObservationPrograms(t *testing.T) {
	t.Parallel()
	host := healthyHost("2097152", false, "")
	resp := runPreflight(t, host, testOptions(nil))
	if !resp.OK {
		t.Fatalf("expected preflight OK=true, got false: %+v", resp.Checks)
	}
	if len(host.Calls) == 0 {
		t.Fatal("preflight issued no observations")
	}
	for _, call := range host.Calls {
		if !call.IsObservation() || call.Effectful || call.Verb != "" {
			t.Fatalf("preflight must only observe; got %+v", call)
		}
		if !reach.ObservationPrograms[call.Program] {
			t.Fatalf("program %q is outside the observation set", call.Program)
		}
	}
	for _, id := range []string{domain.PreflightSSHConnectID, domain.PreflightPrivilegeID, domain.PreflightOSReleaseID, domain.PreflightFirewallID, domain.PreflightDiskFreeID, domain.PreflightRAMTotalID, domain.PreflightDockerID, domain.PreflightSystemdID, domain.PreflightAptAccessID, domain.PreflightStaleProcessesID} {
		c, found := checkByID(resp, id)
		if !found || c.Status != domain.PreflightPass {
			t.Fatalf("check %s = %+v, want pass", id, c)
		}
	}
	if c, _ := checkByID(resp, domain.PreflightOutboundNetworkID); c.Status != domain.PreflightWarn {
		t.Fatalf("outbound network must be reported as unobserved (warn), got %+v", c)
	}
}

func TestRun_VerifiesNonRootPrivilegeStrategy(t *testing.T) {
	t.Parallel()
	host := healthyHost("2097152", false, "")
	resp := runPreflightAsUser(t, host, "deploy", testOptions(nil))
	if !resp.OK {
		t.Fatalf("expected non-root user with sudo to pass preflight, got: %+v", resp.Checks)
	}
	check, found := checkByID(resp, domain.PreflightPrivilegeID)
	if !found || check.Status != domain.PreflightPass || check.Data["strategy"] != "sudo_non_interactive" {
		t.Fatalf("privilege check = %+v, want sudo_non_interactive pass", check)
	}

	host.Answers["sudo -n -l"] = reachtest.Answer{Result: reach.Result{ExitCode: 1, Stderr: "deploy is not allowed to run sudo"}}
	resp = runPreflightAsUser(t, host, "deploy", testOptions(nil))
	check, found = checkByID(resp, domain.PreflightPrivilegeID)
	if !found || check.Status != domain.PreflightFail || !strings.Contains(check.Hint, "target owner") {
		t.Fatalf("privilege refusal = %+v, want actionable target-owner guidance", check)
	}
}

func TestRun_FailsOnLowRAMAndBusyEdgePorts(t *testing.T) {
	t.Parallel()
	host := healthyHost("262144", true, "nginx")
	resp := runPreflight(t, host, testOptions(nil))
	if resp.OK {
		t.Fatal("expected preflight OK=false")
	}
	if c, _ := checkByID(resp, domain.PreflightRAMTotalID); c.Status != domain.PreflightFail {
		t.Fatalf("ram check = %+v, want fail", c)
	}
	c, _ := checkByID(resp, domain.PreflightPortsEdgeID)
	if c.Status != domain.PreflightFail || !strings.Contains(c.Data["processes"], "nginx") {
		t.Fatalf("edge ports check = %+v, want fail naming nginx", c)
	}
}

func TestRun_AllowsBusyEdgePortsWhenOwnedByCaddy(t *testing.T) {
	t.Parallel()
	resp := runPreflight(t, healthyHost("2097152", true, "caddy"), testOptions(nil))
	if !resp.OK {
		t.Fatalf("expected preflight OK=true when caddy owns edge ports, got false: %+v", resp.Checks)
	}
}

func TestRun_FailsWhenAnalyzerRequirementExceedsStaticFloor(t *testing.T) {
	t.Parallel()
	resp := runPreflight(t, healthyHost("2097152", false, ""), testOptions(func(context.Context, string) (*ScenarioRequirements, error) {
		return &ScenarioRequirements{RAMKB: 3 * 1024 * 1024, CPUCores: 2, Tier: "tier-4-saas", Source: "scenario-dependency-analyzer", Confidence: "medium"}, nil
	}))
	if resp.OK {
		t.Fatal("expected preflight OK=false when graph RAM requirement is unmet")
	}
	c, found := checkByID(resp, domain.PreflightRAMTotalID)
	if !found || c.Status != domain.PreflightFail || c.Data["required_by_graph_ram_kb"] != "3145728" {
		t.Fatalf("ram check = %+v, want fail with graph requirement", c)
	}
}

// An unreachable target is one failing reachability check with the typed
// reach refusal in its details; the remaining host probes are not attempted.
func TestRun_UnreachableTargetStopsAfterReachability(t *testing.T) {
	t.Parallel()
	host := &reachtest.Scripted{Strict: true, Answers: map[string]reachtest.Answer{
		"uname -s": {Err: &reach.Error{Kind: reach.KindTargetOffline, Transport: "ssh", Detail: "ssh connection failed"}},
	}}
	resp := runPreflight(t, host, testOptions(nil))
	if resp.OK {
		t.Fatal("expected preflight OK=false")
	}
	c, found := checkByID(resp, domain.PreflightSSHConnectID)
	if !found || c.Status != domain.PreflightFail || !strings.Contains(c.Details, "target_offline") {
		t.Fatalf("reachability check = %+v, want fail carrying the reach kind", c)
	}
	if len(host.Calls) != 1 {
		t.Fatalf("expected exactly one observation before stopping, got %v", host.CallKeys())
	}
}

// UFW state is read from its own files: an active firewall without 80/443
// allow tuples fails; with them it passes.
func TestRun_FirewallReadFromUFWFiles(t *testing.T) {
	t.Parallel()
	host := healthyHost("2097152", false, "")
	host.Answers["cat /etc/ufw/ufw.conf"] = reachtest.Answer{Result: reach.Result{Stdout: "ENABLED=yes\nLOGLEVEL=low"}}
	host.Answers["cat /etc/ufw/user.rules"] = reachtest.Answer{Result: reach.Result{Stdout: "### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in\n-A ufw-user-input -p tcp --dport 22 -j ACCEPT"}}
	resp := runPreflight(t, host, testOptions(nil))
	if c, _ := checkByID(resp, domain.PreflightFirewallID); c.Status != domain.PreflightFail {
		t.Fatalf("firewall check = %+v, want fail", c)
	}
	host.Answers["cat /etc/ufw/user.rules"] = reachtest.Answer{Result: reach.Result{Stdout: "### tuple ### allow tcp 80 0.0.0.0/0 any 0.0.0.0/0 in\n### tuple ### allow tcp 443 0.0.0.0/0 any 0.0.0.0/0 in"}}
	resp = runPreflight(t, host, testOptions(nil))
	if c, _ := checkByID(resp, domain.PreflightFirewallID); c.Status != domain.PreflightPass {
		t.Fatalf("firewall check = %+v, want pass", c)
	}
}

func TestStaleProcessLinesExcludeObservers(t *testing.T) {
	t.Parallel()
	lines := staleProcessLines("100 bash -c pgrep -a -f app\n200 node /root/Vrooli/scenarios/app/api/server.js\n300 vim app.go\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "PID 200") {
		t.Fatalf("stale lines = %v", lines)
	}
}
