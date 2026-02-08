package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	Requirements    ScenarioRequirementsFetcher
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
	if opts.Requirements == nil {
		opts.Requirements = fetchScenarioRequirementsFromAnalyzer
	}

	cfg := ssh.ConfigFromManifest(manifest)
	diskRequiredKB := MinDiskFreeKB
	ramRequiredKB := MinRAMKB
	ramRecommendedKB := RecommendedRAMKB
	requirementData := map[string]string{
		"static_floor_disk_kb":    strconv.FormatInt(MinDiskFreeKB, 10),
		"static_floor_ram_kb":     strconv.FormatInt(MinRAMKB, 10),
		"static_floor_ram_rec_kb": strconv.FormatInt(RecommendedRAMKB, 10),
	}
	if manifest.Scenario.ID != "" && opts.Requirements != nil {
		estimate, err := opts.Requirements(ctx, manifest.Scenario.ID)
		if err != nil {
			requirementData["requirement_source"] = "static_fallback"
			requirementData["requirement_error"] = err.Error()
		} else if estimate != nil {
			requirementData["requirement_source"] = estimate.Source
			requirementData["requirement_confidence"] = estimate.Confidence
			requirementData["requirement_tier"] = estimate.Tier
			requirementData["required_by_graph_ram_kb"] = strconv.FormatInt(estimate.RAMKB, 10)
			requirementData["required_by_graph_disk_kb"] = strconv.FormatInt(estimate.DiskKB, 10)
			requirementData["required_by_graph_cpu_cores"] = strconv.FormatFloat(estimate.CPUCores, 'f', -1, 64)
			if estimate.RAMKB > ramRequiredKB {
				ramRequiredKB = estimate.RAMKB
			}
			if estimate.RAMKB > ramRecommendedKB {
				ramRecommendedKB = estimate.RAMKB
			}
			if estimate.DiskKB > diskRequiredKB {
				diskRequiredKB = estimate.DiskKB
			}
		}
	}
	requirementData["effective_required_ram_kb"] = strconv.FormatInt(ramRequiredKB, 10)
	requirementData["effective_required_disk_kb"] = strconv.FormatInt(diskRequiredKB, 10)
	requirementData["effective_recommended_ram_kb"] = strconv.FormatInt(ramRecommendedKB, 10)
	requirementData["effective_required_ram_human"] = formatBytes(ramRequiredKB)
	requirementData["effective_required_disk_human"] = formatBytes(diskRequiredKB)
	requirementData["effective_recommended_ram_human"] = formatBytes(ramRecommendedKB)

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
		dns01Token := ""
		if opts.ProvidedSecrets != nil {
			dns01Token = strings.TrimSpace(opts.ProvidedSecrets[domain.CloudflareAPITokenKey])
		}
		checks = append(checks, dns.ProxyModeCheck(dnsEval, manifest.Edge.DNSPolicy, dns01Token))
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
		alpnCheck := tlsinfo.RunALPNCheck(ctx, domainName, opts.PortProbe, opts.TLSALPNProbe, portTimeout, alpnTimeout)
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

	if _, err := sshRunner.Run(ctx, cfg, "echo ok", ssh.DefaultRunOptions()); err != nil {
		fail(
			domain.PreflightSSHConnectID,
			"SSH connectivity",
			"Unable to run a remote command over SSH",
			"Confirm SSH key auth works (root login for P0) and that port 22 is reachable.",
			map[string]string{"host": cfg.Host, "user": cfg.User, "port": strconv.Itoa(cfg.Port)},
		)
	} else {
		pass(domain.PreflightSSHConnectID, "SSH connectivity", "SSH command executed successfully", map[string]string{"host": cfg.Host, "user": cfg.User})
	}

	osRes, osErr := sshRunner.Run(ctx, cfg, "cat /etc/os-release", ssh.DefaultRunOptions())
	if osErr != nil || osRes.ExitCode != 0 {
		fail(
			domain.PreflightOSReleaseID,
			"Ubuntu version",
			"Unable to read /etc/os-release",
			"Ensure the VPS is running Ubuntu and that /etc/os-release is readable.",
			map[string]string{"stderr": osRes.Stderr},
		)
	} else {
		id, ver := parseOSRelease(osRes.Stdout)
		if id != SupportedOSID {
			fail(
				domain.PreflightOSReleaseID,
				"Ubuntu version",
				fmt.Sprintf("Unsupported OS: %s", id),
				"scenario-to-cloud requires Ubuntu. Non-Ubuntu systems are not supported.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else if ver == RecommendedUbuntuVersion {
			pass(domain.PreflightOSReleaseID, "Ubuntu version", "Ubuntu 24.04 detected", map[string]string{"id": id, "version_id": ver})
		} else if ver == SupportedUbuntuAltVersion || ver == LegacyUbuntuAltVersion {
			// Older LTS versions should work but aren't officially tested
			warn(
				domain.PreflightOSReleaseID,
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (%s recommended)", ver, RecommendedUbuntuVersion),
				fmt.Sprintf("Ubuntu %s/%s should work but %s LTS is recommended for best compatibility.", SupportedUbuntuAltVersion, LegacyUbuntuAltVersion, RecommendedUbuntuVersion),
				map[string]string{"id": id, "version_id": ver},
			)
		} else {
			warn(
				domain.PreflightOSReleaseID,
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (%s recommended)", ver, RecommendedUbuntuVersion),
				fmt.Sprintf("This Ubuntu version is untested. Consider using Ubuntu %s LTS.", RecommendedUbuntuVersion),
				map[string]string{"id": id, "version_id": ver},
			)
		}
	}

	portsRes, portsErr := sshRunner.Run(ctx, cfg, `ss -ltnpH '( sport = :80 or sport = :443 )' 2>/dev/null || ss -ltnH '( sport = :80 or sport = :443 )'`, ssh.DefaultRunOptions())
	if portsErr != nil {
		warn(
			domain.PreflightPortsEdgeID,
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
		data := map[string]string{"ss": portsRes.Stdout}
		if len(bindings) > 0 {
			if encoded, err := json.Marshal(bindings); err == nil {
				data["port_bindings"] = string(encoded)
			}
			data["ports_in_use"] = strings.Join(portBindingPorts(bindings), ",")
			data["processes"] = strings.Join(portBindingProcessList(bindings), ", ")
		}

		// Caddy is the expected edge owner for deployed hosts; don't block convergence.
		allCaddy := len(bindings) > 0
		for _, binding := range bindings {
			if !strings.EqualFold(binding.Process, "caddy") && !strings.EqualFold(binding.Service, "caddy") {
				allCaddy = false
				break
			}
		}
		if allCaddy {
			pass(
				domain.PreflightPortsEdgeID,
				"Ports 80/443 availability",
				fmt.Sprintf("%s (expected edge owner: caddy)", details),
				data,
			)
		} else {
			hint := "Ports 80/443 must be free for Caddy to complete Let's Encrypt HTTP-01 challenges."
			if len(bindings) > 0 {
				hint = hint + " Use the Free Ports action or run: sudo systemctl stop <service> or sudo kill <pid>."
			}
			fail(
				domain.PreflightPortsEdgeID,
				"Ports 80/443 availability",
				details,
				hint,
				data,
			)
		}
	} else {
		pass(domain.PreflightPortsEdgeID, "Ports 80/443 availability", "Ports 80/443 appear free", nil)
	}

	ufwRes, ufwErr := sshRunner.Run(ctx, cfg, "ufw status", ssh.DefaultRunOptions())
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

	netRes, netErr := sshRunner.Run(ctx, cfg, `curl -fsS --max-time 5 https://example.com >/dev/null`, ssh.DefaultRunOptions())
	if netErr != nil || netRes.ExitCode != 0 {
		warn(
			domain.PreflightOutboundNetworkID,
			"Outbound network",
			"Unable to confirm outbound HTTPS access with curl",
			"Ensure outbound network access is allowed (apt/pnpm downloads, Let's Encrypt).",
			map[string]string{"stderr": netRes.Stderr},
		)
	} else {
		pass(domain.PreflightOutboundNetworkID, "Outbound network", "Outbound HTTPS access looks OK", nil)
	}

	diskRes, diskErr := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`, ssh.DefaultRunOptions())
	if diskErr != nil || diskRes.ExitCode != 0 {
		warn(domain.PreflightDiskFreeID, "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", map[string]string{"stderr": diskRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(diskRes.Stdout), 10, 64)
		detailsData := map[string]string{
			"free_kb":            diskRes.Stdout,
			"free_human":         formatBytes(kb),
			"required_min_kb":    strconv.FormatInt(diskRequiredKB, 10),
			"required_min_human": formatBytes(diskRequiredKB),
		}
		for k, v := range requirementData {
			detailsData[k] = v
		}
		if kb > 0 && kb < diskRequiredKB {
			fail(
				domain.PreflightDiskFreeID,
				"Disk free space",
				fmt.Sprintf("Low free disk space: %s", formatBytes(kb)),
				fmt.Sprintf("At least %s free space is required for this deployment. Run: sudo apt clean && sudo journalctl --vacuum-size=100M", formatBytes(diskRequiredKB)),
				detailsData,
			)
		} else {
			pass(domain.PreflightDiskFreeID, "Disk free space", fmt.Sprintf("Free space: %s", formatBytes(kb)), detailsData)
		}
	}

	ramRes, ramErr := sshRunner.Run(ctx, cfg, `awk '/MemTotal/ {print $2}' /proc/meminfo`, ssh.DefaultRunOptions())
	if ramErr != nil || ramRes.ExitCode != 0 {
		warn(domain.PreflightRAMTotalID, "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", map[string]string{"stderr": ramRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(ramRes.Stdout), 10, 64)
		detailsData := map[string]string{
			"memtotal_kb":              ramRes.Stdout,
			"memtotal_human":           formatBytes(kb),
			"required_min_kb":          strconv.FormatInt(ramRequiredKB, 10),
			"required_min_human":       formatBytes(ramRequiredKB),
			"recommended_min_kb":       strconv.FormatInt(ramRecommendedKB, 10),
			"recommended_min_human":    formatBytes(ramRecommendedKB),
			"required_by_graph_ram_mb": strconv.FormatFloat(math.Ceil(float64(ramRequiredKB)/1024), 'f', -1, 64),
		}
		for k, v := range requirementData {
			detailsData[k] = v
		}
		if kb > 0 && kb < ramRequiredKB {
			fail(
				domain.PreflightRAMTotalID,
				"RAM",
				fmt.Sprintf("Low RAM: %s", formatBytes(kb)),
				fmt.Sprintf("At least %s RAM is required for this deployment. %s is recommended.", formatBytes(ramRequiredKB), formatBytes(ramRecommendedKB)),
				detailsData,
			)
		} else if kb > 0 && kb < ramRecommendedKB {
			warn(
				"ram_total",
				"RAM",
				fmt.Sprintf("RAM: %s (%s recommended)", formatBytes(kb), formatBytes(ramRecommendedKB)),
				"Your VPS has limited RAM for this deployment profile. Consider upgrading for better performance.",
				detailsData,
			)
		} else {
			pass(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("RAM: %s", formatBytes(kb)), detailsData)
		}
	}

	// Check: required system commands (bootstrap will install if missing)
	requiredCmds := []struct {
		name string
		id   string
	}{
		{"curl", domain.PreflightCmdCurlID},
		{"git", domain.PreflightCmdGitID},
		{"unzip", domain.PreflightCmdUnzipID},
		{"tar", domain.PreflightCmdTarID},
	}
	for _, cmd := range requiredCmds {
		res, err := sshRunner.Run(ctx, cfg, "which "+cmd.name, ssh.DefaultRunOptions())
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
	jqRes, jqErr := sshRunner.Run(ctx, cfg, "which jq", ssh.DefaultRunOptions())
	if jqErr != nil || jqRes.ExitCode != 0 {
		warn(domain.PreflightCmdJqID, "jq available",
			"jq not found on VPS",
			"Bootstrap phase will install this automatically",
			nil)
	} else {
		pass(domain.PreflightCmdJqID, "jq available",
			"Found at "+strings.TrimSpace(jqRes.Stdout), nil)
	}

	// Check: apt access (required for bootstrap to work)
	aptRes, aptErr := sshRunner.Run(ctx, cfg, "apt-get update --print-uris &> /tmp/apt-check.log && head -1 /tmp/apt-check.log", ssh.DefaultRunOptions())
	if aptErr != nil {
		fail(domain.PreflightAptAccessID, "apt accessible",
			"Unable to run apt-get",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			map[string]string{"error": aptErr.Error()})
	} else if strings.Contains(aptRes.Stderr, "Permission denied") || strings.Contains(aptRes.Stdout, "Permission denied") {
		fail(domain.PreflightAptAccessID, "apt accessible",
			"apt-get permission denied",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			nil)
	} else {
		pass(domain.PreflightAptAccessID, "apt accessible", "apt-get is accessible", nil)
	}

	// Check: Docker available and running
	dockerRes, dockerErr := sshRunner.Run(ctx, cfg, "command -v docker && docker info --format '{{.ServerVersion}}'", ssh.DefaultRunOptions())
	if dockerErr != nil || dockerRes.ExitCode != 0 {
		warn(domain.PreflightDockerID, "Docker available",
			"Docker not found or not running",
			"Bootstrap will attempt to install Docker. If already installed, verify dockerd is running.",
			map[string]string{"stderr": dockerRes.Stderr})
	} else {
		pass(domain.PreflightDockerID, "Docker available",
			"Docker "+strings.TrimSpace(dockerRes.Stdout), nil)
	}

	// Check: systemd init system
	systemdRes, systemdErr := sshRunner.Run(ctx, cfg, "command -v systemctl && systemctl --version | head -1", ssh.DefaultRunOptions())
	if systemdErr != nil || systemdRes.ExitCode != 0 {
		warn(domain.PreflightSystemdID, "systemd available",
			"systemctl not found",
			"Deployment expects systemd for service management. Non-systemd systems (Alpine/OpenRC) are not supported.",
			nil)
	} else {
		pass(domain.PreflightSystemdID, "systemd available",
			strings.TrimSpace(systemdRes.Stdout), nil)
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
