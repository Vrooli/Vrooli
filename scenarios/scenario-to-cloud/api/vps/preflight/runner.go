package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tlsinfo"
)

// RunOptions configures optional behavior for VPS preflight.
type RunOptions struct {
	ProvidedSecrets map[string]string
	PortProbe       tlsinfo.PortProbeFunc
	TLSALPNProbe    tlsinfo.ALPNProbeFunc
}

// Run executes all VPS preflight checks and returns the combined results.
func Run(
	ctx context.Context,
	manifest domain.CloudManifest,
	dnsService dns.Service,
	sshRunner ssh.Runner,
	opts RunOptions,
) domain.PreflightResponse {
	if opts.PortProbe == nil {
		opts.PortProbe = tlsinfo.DefaultPortProbe
	}
	if opts.TLSALPNProbe == nil {
		opts.TLSALPNProbe = tlsinfo.DefaultALPNProbe
	}

	cfg := ssh.ConfigFromManifest(manifest)
	checks := make([]domain.PreflightCheck, 0, 8)
	fail := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightFail,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}
	pass := func(id, title, details string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightPass,
			Details: details,
			Data:    data,
		})
	}
	warn := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightWarn,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}

	dnsEval := dns.Evaluate(ctx, dnsService, manifest.Edge.Domain, cfg.Host)
	checks = append(checks, dns.PreflightChecksFromEvaluation(dnsEval, manifest.Edge.DNSPolicy)...)

	if manifest.Edge.Domain != "" {
		proxiedDomains := []string{}
		for _, status := range dnsEval.Statuses {
			if status.Role != "apex" && status.Role != "www" && status.Role != "edge" {
				continue
			}
			if status.Proxied {
				proxiedDomains = append(proxiedDomains, status.Host)
			}
		}

		dns01Token := ""
		if opts.ProvidedSecrets != nil {
			dns01Token = strings.TrimSpace(opts.ProvidedSecrets[domain.CloudflareAPITokenKey])
		}

		recordProxyIssue := func(details, hint string, data map[string]string) {
			if manifest.Edge.DNSPolicy == domain.DNSPolicyWarn {
				warn(domain.PreflightDNSProxyModeID, "Proxy mode vs issuance", details, hint, data)
				return
			}
			fail(domain.PreflightDNSProxyModeID, "Proxy mode vs issuance", details, hint, data)
		}

		switch {
		case manifest.Edge.DNSPolicy == domain.DNSPolicySkip:
			warn(domain.PreflightDNSProxyModeID, "Proxy mode vs issuance", "DNS checks skipped by policy", "Set edge.dns_policy to required to enforce DNS validation.", nil)
		case len(proxiedDomains) == 0:
			pass(domain.PreflightDNSProxyModeID, "Proxy mode vs issuance", "No Cloudflare proxy detected for edge/apex/www", nil)
		case dns01Token != "":
			pass(domain.PreflightDNSProxyModeID, "Proxy mode vs issuance", "Cloudflare proxy detected and DNS-01 credentials provided", map[string]string{"proxied_domains": strings.Join(proxiedDomains, ","), "dns01": "enabled"})
		default:
			recordProxyIssue(
				"Cloudflare proxy detected without DNS-01 credentials",
				"Disable proxying (DNS-only) during issuance or provide a Cloudflare API token for DNS-01.",
				map[string]string{"proxied_domains": strings.Join(proxiedDomains, ",")},
			)
		}
	}

	publicPorts := []int{80, 443}
	var unreachable []string
	portTimeout := 3 * time.Second
	for _, port := range publicPorts {
		if err := opts.PortProbe(ctx, cfg.Host, port, portTimeout); err != nil {
			unreachable = append(unreachable, strconv.Itoa(port))
		}
	}
	if len(unreachable) > 0 {
		fail(
			domain.PreflightPublicPortsID,
			"Public ports 80/443 reachability",
			fmt.Sprintf("Unable to reach ports %s on %s from the deployment runner", strings.Join(unreachable, ","), cfg.Host),
			"Open inbound 80/443 at the VPS firewall and provider security group, or verify the host IP.",
			map[string]string{"host": cfg.Host, "ports": strings.Join(unreachable, ",")},
		)
	} else {
		pass(
			domain.PreflightPublicPortsID,
			"Public ports 80/443 reachability",
			"Ports 80/443 reachable from the deployment runner",
			map[string]string{"host": cfg.Host, "ports": "80,443"},
		)
	}

	if manifest.Edge.Caddy.Enabled && strings.TrimSpace(manifest.Edge.Domain) != "" {
		domainName := strings.TrimSpace(manifest.Edge.Domain)
		alpnTimeout := 4 * time.Second
		reachErr := opts.PortProbe(ctx, domainName, 443, portTimeout)
		proto := ""
		var probeErr error
		if reachErr == nil {
			proto, probeErr = opts.TLSALPNProbe(ctx, domainName, domainName, 443, alpnTimeout)
		}
		alpnCheck := tlsinfo.EvaluateALPN(domainName, reachErr, proto, probeErr)
		data := map[string]string{"domain": domainName}
		if alpnCheck.Protocol != "" {
			data["protocol"] = alpnCheck.Protocol
		}
		if alpnCheck.Error != "" {
			data["error"] = alpnCheck.Error
		}
		if alpnCheck.Status == tlsinfo.ALPNPass {
			pass(domain.PreflightTLSALPNID, "TLS-ALPN compatibility", alpnCheck.Message, data)
		} else {
			warn(domain.PreflightTLSALPNID, "TLS-ALPN compatibility", alpnCheck.Message, alpnCheck.Hint, data)
		}
	}

	if _, err := sshRunner.Run(ctx, cfg, "echo ok"); err != nil {
		fail(
			"ssh_connect",
			"SSH connectivity",
			"Unable to run a remote command over SSH",
			"Confirm SSH key auth works (root login for P0) and that port 22 is reachable.",
			map[string]string{"host": cfg.Host, "user": cfg.User, "port": strconv.Itoa(cfg.Port)},
		)
	} else {
		pass("ssh_connect", "SSH connectivity", "SSH command executed successfully", map[string]string{"host": cfg.Host, "user": cfg.User})
	}

	osRes, osErr := sshRunner.Run(ctx, cfg, "cat /etc/os-release")
	if osErr != nil || osRes.ExitCode != 0 {
		fail(
			"os_release",
			"Ubuntu version",
			"Unable to read /etc/os-release",
			"Ensure the VPS is running Ubuntu and that /etc/os-release is readable.",
			map[string]string{"stderr": osRes.Stderr},
		)
	} else {
		id, ver := parseOSRelease(osRes.Stdout)
		if id != "ubuntu" {
			fail(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Unsupported OS: %s", id),
				"scenario-to-cloud requires Ubuntu. Debian may work but is untested.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else if ver == "24.04" {
			pass("os_release", "Ubuntu version", "Ubuntu 24.04 detected", map[string]string{"id": id, "version_id": ver})
		} else if ver == "22.04" || ver == "20.04" {
			// Older LTS versions should work but aren't officially tested
			warn(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (24.04 recommended)", ver),
				"Ubuntu 22.04/20.04 should work but 24.04 LTS is recommended for best compatibility.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else {
			warn(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (24.04 recommended)", ver),
				"This Ubuntu version is untested. Consider using Ubuntu 24.04 LTS.",
				map[string]string{"id": id, "version_id": ver},
			)
		}
	}

	portsRes, portsErr := sshRunner.Run(ctx, cfg, `ss -ltnpH '( sport = :80 or sport = :443 )' 2>/dev/null || ss -ltnH '( sport = :80 or sport = :443 )'`)
	if portsErr != nil {
		warn(
			"ports_80_443",
			"Ports 80/443 availability",
			"Unable to check ports 80/443 via ss",
			"Ensure ports 80 and 443 are free for Caddy/Let's Encrypt HTTP-01.",
			map[string]string{"stderr": portsRes.Stderr},
		)
	} else if strings.TrimSpace(portsRes.Stdout) != "" {
		bindings := parsePortBindings(portsRes.Stdout)
		details := "Port 80 and/or 443 appears to already be in use"
		if len(bindings) > 0 {
			details = fmt.Sprintf("Ports in use: %s", formatPortBindings(bindings))
		}
		hint := "Ports 80/443 must be free for Caddy to complete Let's Encrypt HTTP-01 challenges."
		if len(bindings) > 0 {
			hint = hint + " Use the Free Ports action or run: sudo systemctl stop <service> or sudo kill <pid>."
		}
		data := map[string]string{"ss": portsRes.Stdout}
		if len(bindings) > 0 {
			if encoded, err := json.Marshal(bindings); err == nil {
				data["port_bindings"] = string(encoded)
			}
			data["ports_in_use"] = strings.Join(portBindingPorts(bindings), ",")
			data["processes"] = strings.Join(portBindingProcessList(bindings), ", ")
		}
		fail(
			"ports_80_443",
			"Ports 80/443 availability",
			details,
			hint,
			data,
		)
	} else {
		pass("ports_80_443", "Ports 80/443 availability", "Ports 80/443 appear free", nil)
	}

	ufwRes, ufwErr := sshRunner.Run(ctx, cfg, "ufw status")
	if ufwErr != nil {
		warn(
			domain.PreflightFirewallID,
			"Inbound firewall rules",
			"Unable to check UFW status",
			"Confirm inbound firewall rules allow ports 80/443 (UFW, iptables, or cloud security group).",
			map[string]string{"stderr": ufwRes.Stderr},
		)
	} else {
		statusLine := ""
		lines := strings.Split(strings.TrimSpace(ufwRes.Stdout), "\n")
		if len(lines) > 0 {
			statusLine = strings.ToLower(strings.TrimSpace(lines[0]))
		}
		if strings.Contains(statusLine, "inactive") {
			pass(domain.PreflightFirewallID, "Inbound firewall rules", "UFW is inactive", nil)
		} else if strings.Contains(statusLine, "active") {
			allow80 := false
			allow443 := false
			for _, line := range lines[1:] {
				line = strings.ToLower(line)
				if !strings.Contains(line, "allow") {
					continue
				}
				if ufwAllowsPort(line, 80) {
					allow80 = true
				}
				if ufwAllowsPort(line, 443) {
					allow443 = true
				}
			}
			if allow80 && allow443 {
				pass(domain.PreflightFirewallID, "Inbound firewall rules", "UFW allows inbound 80/443", nil)
			} else {
				fail(
					domain.PreflightFirewallID,
					"Inbound firewall rules",
					"UFW is active but does not allow inbound 80/443",
					"Run: sudo ufw allow 80/tcp && sudo ufw allow 443/tcp (or update firewall/security group rules).",
					map[string]string{"ufw_status": ufwRes.Stdout},
				)
			}
		} else if strings.Contains(strings.ToLower(ufwRes.Stderr), "command not found") {
			warn(
				domain.PreflightFirewallID,
				"Inbound firewall rules",
				"UFW not installed",
				"Confirm inbound firewall rules allow ports 80/443 (UFW, iptables, or cloud security group).",
				nil,
			)
		} else {
			warn(
				domain.PreflightFirewallID,
				"Inbound firewall rules",
				"Unable to determine firewall status",
				"Confirm inbound firewall rules allow ports 80/443 (UFW, iptables, or cloud security group).",
				map[string]string{"ufw_status": ufwRes.Stdout},
			)
		}
	}

	netRes, netErr := sshRunner.Run(ctx, cfg, `curl -fsS --max-time 5 https://example.com >/dev/null`)
	if netErr != nil || netRes.ExitCode != 0 {
		warn(
			"outbound_network",
			"Outbound network",
			"Unable to confirm outbound HTTPS access with curl",
			"Ensure outbound network access is allowed (apt/pnpm downloads, Let's Encrypt).",
			map[string]string{"stderr": netRes.Stderr},
		)
	} else {
		pass("outbound_network", "Outbound network", "Outbound HTTPS access looks OK", nil)
	}

	diskRes, diskErr := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
	if diskErr != nil || diskRes.ExitCode != 0 {
		warn("disk_free", "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", map[string]string{"stderr": diskRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(diskRes.Stdout), 10, 64)
		const minKB = 5 * 1024 * 1024 // 5 GiB
		if kb > 0 && kb < minKB {
			fail(
				"disk_free",
				"Disk free space",
				fmt.Sprintf("Low free disk space: %s", formatBytes(kb)),
				"At least 5 GB free space is recommended. Run: sudo apt clean && sudo journalctl --vacuum-size=100M",
				map[string]string{"free_kb": diskRes.Stdout, "free_human": formatBytes(kb)},
			)
		} else {
			pass("disk_free", "Disk free space", fmt.Sprintf("Free space: %s", formatBytes(kb)), map[string]string{"free_kb": diskRes.Stdout, "free_human": formatBytes(kb)})
		}
	}

	ramRes, ramErr := sshRunner.Run(ctx, cfg, `awk '/MemTotal/ {print $2}' /proc/meminfo`)
	if ramErr != nil || ramRes.ExitCode != 0 {
		warn("ram_total", "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", map[string]string{"stderr": ramRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(ramRes.Stdout), 10, 64)
		const minKB = 1024 * 1024      // 1 GiB
		const warnKB = 2 * 1024 * 1024 // 2 GiB
		if kb > 0 && kb < minKB {
			fail(
				"ram_total",
				"RAM",
				fmt.Sprintf("Low RAM: %s", formatBytes(kb)),
				"At least 1 GB RAM is required. 2+ GB is recommended for most scenarios.",
				map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)},
			)
		} else if kb > 0 && kb < warnKB {
			warn(
				"ram_total",
				"RAM",
				fmt.Sprintf("RAM: %s (2+ GB recommended)", formatBytes(kb)),
				"Your VPS has limited RAM. Consider upgrading for better performance.",
				map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)},
			)
		} else {
			pass("ram_total", "RAM", fmt.Sprintf("RAM: %s", formatBytes(kb)), map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)})
		}
	}

	// Check: required system commands (bootstrap will install if missing)
	requiredCmds := []struct {
		name string
		id   string
	}{
		{"curl", "cmd_curl"},
		{"git", "cmd_git"},
		{"unzip", "cmd_unzip"},
		{"tar", "cmd_tar"},
	}
	for _, cmd := range requiredCmds {
		res, err := sshRunner.Run(ctx, cfg, "which "+cmd.name)
		if err != nil || res.ExitCode != 0 {
			warn(cmd.id, cmd.name+" available",
				cmd.name+" not found on VPS",
				"Bootstrap phase will install this automatically",
				nil)
		} else {
			pass(cmd.id, cmd.name+" available",
				"Found at "+strings.TrimSpace(res.Stdout), nil)
		}
	}

	// Check: jq (nice to have, warn only)
	jqRes, jqErr := sshRunner.Run(ctx, cfg, "which jq")
	if jqErr != nil || jqRes.ExitCode != 0 {
		warn("cmd_jq", "jq available",
			"jq not found on VPS",
			"Bootstrap phase will install this automatically",
			nil)
	} else {
		pass("cmd_jq", "jq available",
			"Found at "+strings.TrimSpace(jqRes.Stdout), nil)
	}

	// Check: apt access (required for bootstrap to work)
	aptRes, aptErr := sshRunner.Run(ctx, cfg, "apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log")
	if aptErr != nil {
		fail("apt_access", "apt accessible",
			"Unable to run apt-get",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			map[string]string{"error": aptErr.Error()})
	} else if strings.Contains(aptRes.Stderr, "Permission denied") || strings.Contains(aptRes.Stdout, "Permission denied") {
		fail("apt_access", "apt accessible",
			"apt-get permission denied",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			nil)
	} else {
		pass("apt_access", "apt accessible", "apt-get is accessible", nil)
	}

	// Check: stale scenario processes that might have outdated credentials
	checkStaleScenarioProcesses(ctx, cfg, sshRunner, manifest, warn, pass)

	// Check: credential validation for required resources (postgres, redis, etc.)
	workdir := manifest.Target.VPS.Workdir
	credentialChecks := RunCredentialValidation(ctx, cfg, sshRunner, manifest, workdir)
	checks = append(checks, credentialChecks...)

	ok := true
	for _, c := range checks {
		if c.Status == domain.PreflightFail {
			ok = false
			break
		}
	}

	return domain.PreflightResponse{
		OK:        ok,
		Checks:    checks,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
