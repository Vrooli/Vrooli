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
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/tlsinfo"
)

// RunOptions configures optional behavior for VPS preflight.
type RunOptions struct {
	ProvidedSecrets map[string]string
	PortProbe       tlsinfo.PortProbeFunc
	TLSALPNProbe    tlsinfo.ALPNProbeFunc
	Requirements    ScenarioRequirementsFetcher
}

// Run executes every preflight check for a target and returns the combined
// result. Host facts come from read-only observation programs through the
// bound reach (the host may not carry a vrooli binary yet); nothing here can
// change the target.
func Run(
	ctx context.Context,
	manifest domain.CloudManifest,
	dnsService dns.Service,
	rr reach.Reach,
	target identity.TargetRef,
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
	if target.Locator.Host == "" {
		target = domain.TargetRefFromManifest(manifest)
	}
	host := target.Locator.Host
	obs := observer{reach: rr, target: target}

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

	checks := make([]domain.PreflightCheck, 0, 16)
	fail := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightFail, Details: details, Hint: hint, Data: data})
	}
	pass := func(id, title, details string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightPass, Details: details, Data: data})
	}
	warn := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightWarn, Details: details, Hint: hint, Data: data})
	}

	dnsEval := dns.Evaluate(ctx, dnsService, manifest.Edge.Domain, host)
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
		if err := opts.PortProbe(ctx, host, port, portTimeout); err != nil {
			unreachable = append(unreachable, strconv.Itoa(port))
		}
	}
	if len(unreachable) > 0 {
		fail(domain.PreflightPublicPortsID, "Public ports 80/443 reachability",
			fmt.Sprintf("Unable to reach ports %s on %s from the deployment runner", strings.Join(unreachable, ","), host),
			"Open inbound 80/443 at the VPS firewall and provider security group, or verify the host IP.",
			map[string]string{"host": host, "ports": strings.Join(unreachable, ",")})
	} else {
		pass(domain.PreflightPublicPortsID, "Public ports 80/443 reachability", "Ports 80/443 reachable from the deployment runner", map[string]string{"host": host, "ports": "80,443"})
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

	connData := map[string]string{"host": host, "user": target.Locator.User, "port": strconv.Itoa(target.Locator.Port), "transport": target.Transport}
	if res, err := obs.observe(ctx, "uname", "-s"); !ok(res, err) {
		detail := "Unable to run a remote observation over the bound transport"
		if err != nil {
			detail += ": " + err.Error()
		}
		fail(domain.PreflightSSHConnectID, "Target reachability", detail,
			"Confirm the target is enrolled (vrooli-bridge onboard) or that the bound SSH credential and port 22 work.", connData)
		return finish(checks)
	}
	pass(domain.PreflightSSHConnectID, "Target reachability", "Remote observation executed successfully", connData)

	if strings.EqualFold(strings.TrimSpace(target.Locator.User), "root") {
		pass(domain.PreflightPrivilegeID, "Privilege strategy", "The bound SSH user is root; host preparation can use the target owner", map[string]string{"user": target.Locator.User, "strategy": "root"})
	} else {
		privilegeRes, privilegeErr := obs.observe(ctx, "sudo", "-n", "-l")
		if !ok(privilegeRes, privilegeErr) {
			detail := fmt.Sprintf("Bound SSH user %q does not have verified non-interactive elevation", target.Locator.User)
			if privilegeErr != nil {
				detail += ": " + privilegeErr.Error()
			} else if strings.TrimSpace(privilegeRes.Stderr) != "" {
				detail += ": " + strings.TrimSpace(privilegeRes.Stderr)
			}
			fail(domain.PreflightPrivilegeID, "Privilege strategy", detail,
				"Grant the bound user the required non-interactive elevation through the target owner, or connect as an approved privileged user before host preparation.",
				map[string]string{"user": target.Locator.User, "strategy": "sudo_non_interactive"})
		} else {
			pass(domain.PreflightPrivilegeID, "Privilege strategy", fmt.Sprintf("Bound SSH user %q has non-interactive elevation", target.Locator.User), map[string]string{"user": target.Locator.User, "strategy": "sudo_non_interactive"})
		}
	}

	osRes, osErr := obs.observe(ctx, "cat", "/etc/os-release")
	if !ok(osRes, osErr) {
		fail(domain.PreflightOSReleaseID, "Ubuntu version", "Unable to read /etc/os-release",
			"Ensure the VPS is running Ubuntu and that /etc/os-release is readable.", map[string]string{"stderr": osRes.Stderr})
	} else {
		id, ver := parseOSRelease(osRes.Stdout)
		switch {
		case id != SupportedOSID:
			fail(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Unsupported OS: %s", id),
				"scenario-to-cloud requires Ubuntu. Non-Ubuntu systems are not supported.", map[string]string{"id": id, "version_id": ver})
		case ver == RecommendedUbuntuVersion:
			pass(domain.PreflightOSReleaseID, "Ubuntu version", "Ubuntu 24.04 detected", map[string]string{"id": id, "version_id": ver})
		case ver == SupportedUbuntuAltVersion || ver == LegacyUbuntuAltVersion:
			warn(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Ubuntu %s detected (%s recommended)", ver, RecommendedUbuntuVersion),
				fmt.Sprintf("Ubuntu %s/%s should work but %s LTS is recommended for best compatibility.", SupportedUbuntuAltVersion, LegacyUbuntuAltVersion, RecommendedUbuntuVersion),
				map[string]string{"id": id, "version_id": ver})
		default:
			warn(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Ubuntu %s detected (%s recommended)", ver, RecommendedUbuntuVersion),
				fmt.Sprintf("This Ubuntu version is untested. Consider using Ubuntu %s LTS.", RecommendedUbuntuVersion), map[string]string{"id": id, "version_id": ver})
		}
	}

	portsRes, portsErr := obs.listeningSockets(ctx, 80, 443)
	if !ok(portsRes, portsErr) {
		warn(domain.PreflightPortsEdgeID, "Ports 80/443 availability", "Unable to check ports 80/443 via ss",
			"Ensure ports 80 and 443 are free for Caddy/Let's Encrypt HTTP-01.", map[string]string{"stderr": portsRes.Stderr})
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
			pass(domain.PreflightPortsEdgeID, "Ports 80/443 availability", fmt.Sprintf("%s (expected edge owner: caddy)", details), data)
		} else {
			hint := "Ports 80/443 must be free for Caddy to complete Let's Encrypt HTTP-01 challenges."
			if len(bindings) > 0 {
				hint += " Stop the owning scenario through its lifecycle owner (Stop Processes) or the unit through the host's service manager."
			}
			fail(domain.PreflightPortsEdgeID, "Ports 80/443 availability", details, hint, data)
		}
	} else {
		pass(domain.PreflightPortsEdgeID, "Ports 80/443 availability", "Ports 80/443 appear free", nil)
	}

	firewallCheck(ctx, obs, fail, pass, warn)

	warn(domain.PreflightOutboundNetworkID, "Outbound network",
		"Outbound reachability is not observed before enrollment",
		"The read-only observation set cannot open connections; host.prepare reports a blocked egress at apply time (apt.packages.ensure). Ensure outbound HTTPS is allowed.",
		nil)

	// Remote VPS snapshot command. Local host inventory probing belongs in internal/hostinventory.
	// hostinventory:remote-snapshot-parser
	diskRes, diskErr := obs.observe(ctx, "df", "-Pk", "/")
	if _, _, availKB, _, found := dfRoot(diskRes); !ok(diskRes, diskErr) || !found {
		warn(domain.PreflightDiskFreeID, "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", map[string]string{"stderr": diskRes.Stderr})
	} else {
		detailsData := map[string]string{
			"free_kb":            strconv.FormatInt(availKB, 10),
			"free_human":         formatBytes(availKB),
			"required_min_kb":    strconv.FormatInt(diskRequiredKB, 10),
			"required_min_human": formatBytes(diskRequiredKB),
		}
		for k, v := range requirementData {
			detailsData[k] = v
		}
		if availKB > 0 && availKB < diskRequiredKB {
			fail(domain.PreflightDiskFreeID, "Disk free space", fmt.Sprintf("Low free disk space: %s", formatBytes(availKB)),
				fmt.Sprintf("At least %s free space is required for this deployment. Free space with the disk cleanup action (journal vacuum, docker prune) or release garbage collection: scenario-to-cloud bundle vps-gc --host %s --scenario %s --keep 2",
					formatBytes(diskRequiredKB), host, manifest.Scenario.ID), detailsData)
		} else {
			pass(domain.PreflightDiskFreeID, "Disk free space", fmt.Sprintf("Free space: %s", formatBytes(availKB)), detailsData)
		}
	}

	// hostinventory:remote-snapshot-parser
	ramRes, ramErr := obs.observe(ctx, "grep", "MemTotal", "/proc/meminfo")
	if kb, found := memTotalKB(ramRes); !ok(ramRes, ramErr) || !found {
		warn(domain.PreflightRAMTotalID, "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", map[string]string{"stderr": ramRes.Stderr})
	} else {
		detailsData := map[string]string{
			"memtotal_kb":              strconv.FormatInt(kb, 10),
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
		switch {
		case kb > 0 && kb < ramRequiredKB:
			fail(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("Low RAM: %s", formatBytes(kb)),
				fmt.Sprintf("At least %s RAM is required for this deployment. %s is recommended.", formatBytes(ramRequiredKB), formatBytes(ramRecommendedKB)), detailsData)
		case kb > 0 && kb < ramRecommendedKB:
			warn(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("RAM: %s (%s recommended)", formatBytes(kb), formatBytes(ramRecommendedKB)),
				"Your VPS has limited RAM for this deployment profile. Consider upgrading for better performance.", detailsData)
		default:
			pass(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("RAM: %s", formatBytes(kb)), detailsData)
		}
	}

	tools := obs.findTools(ctx, "curl", "git", "unzip", "tar", "jq", "apt-get", "docker", "systemctl")
	for _, cmd := range []struct{ name, id string }{
		{"curl", domain.PreflightCmdCurlID},
		{"git", domain.PreflightCmdGitID},
		{"unzip", domain.PreflightCmdUnzipID},
		{"tar", domain.PreflightCmdTarID},
		{"jq", domain.PreflightCmdJqID},
	} {
		if path, found := tools[cmd.name]; found {
			pass(cmd.id, cmd.name+" available", "Found at "+path, nil)
		} else {
			warn(cmd.id, cmd.name+" available", cmd.name+" not found on VPS", "host.prepare installs it through apt.packages.ensure", nil)
		}
	}

	if _, found := tools["apt-get"]; !found {
		fail(domain.PreflightAptAccessID, "apt accessible", "apt-get not found on the target",
			"Bootstrap installs packages through apt.packages.ensure; the target must be Ubuntu.", nil)
	} else if target.Locator.User != "" && target.Locator.User != "root" {
		warn(domain.PreflightAptAccessID, "apt accessible", "apt-get present; the bound user is not root",
			"Package installs run through the privilege broker; the bound user needs its elevation grant.", map[string]string{"user": target.Locator.User})
	} else {
		pass(domain.PreflightAptAccessID, "apt accessible", "apt-get is present", nil)
	}

	// Remote VPS service check. This is not Docker GPU runtime inventory for the local host.
	switch dockerPath, found := tools["docker"]; {
	case found && obs.exists(ctx, "/var/run/docker.sock"):
		pass(domain.PreflightDockerID, "Docker available", "Docker present at "+dockerPath+" (daemon socket present)", nil)
	case found:
		warn(domain.PreflightDockerID, "Docker available", "Docker is installed but its daemon socket is absent",
			"Verify dockerd is running (systemctl status docker).", map[string]string{"path": dockerPath})
	default:
		warn(domain.PreflightDockerID, "Docker available", "Docker not found",
			"Bootstrap will attempt to install Docker through apt.packages.ensure.", nil)
	}

	if obs.exists(ctx, "/run/systemd/system") {
		pass(domain.PreflightSystemdID, "systemd available", "systemd is the running init system", nil)
	} else {
		warn(domain.PreflightSystemdID, "systemd available", "systemd is not the running init system",
			"Deployment expects systemd for service management. Non-systemd systems (Alpine/OpenRC) are not supported.", nil)
	}

	checkStaleScenarioProcesses(ctx, obs, manifest, warn, pass)

	checks = append(checks, RunCredentialValidation(ctx, obs, manifest)...)

	return finish(checks)
}

// finish computes the overall verdict: any failing check fails preflight.
func finish(checks []domain.PreflightCheck) domain.PreflightResponse {
	ok := true
	for _, c := range checks {
		if c.Status == domain.PreflightFail {
			ok = false
			break
		}
	}
	return domain.PreflightResponse{OK: ok, Checks: checks, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// firewallCheck reads UFW's own state files instead of running ufw: the
// observation set carries no program that can change firewall state.
func firewallCheck(ctx context.Context, obs observer, fail func(id, title, details, hint string, data map[string]string), pass func(id, title, details string, data map[string]string), warn func(id, title, details, hint string, data map[string]string)) {
	const hint = "Confirm inbound firewall rules allow ports 80/443 (UFW, iptables, or cloud security group)."
	confRes, confErr := obs.observe(ctx, "cat", "/etc/ufw/ufw.conf")
	if !ok(confRes, confErr) {
		warn(domain.PreflightFirewallID, "Inbound firewall rules", "UFW not installed or its configuration is unreadable", hint, map[string]string{"stderr": confRes.Stderr})
		return
	}
	if !ufwEnabled(confRes.Stdout) {
		pass(domain.PreflightFirewallID, "Inbound firewall rules", "UFW is inactive", nil)
		return
	}
	rulesRes, rulesErr := obs.observe(ctx, "cat", "/etc/ufw/user.rules")
	if !ok(rulesRes, rulesErr) {
		warn(domain.PreflightFirewallID, "Inbound firewall rules", "UFW is active but its rule set is unreadable", hint, map[string]string{"stderr": rulesRes.Stderr})
		return
	}
	allow80, allow443 := ufwRulesAllow(rulesRes.Stdout)
	if allow80 && allow443 {
		pass(domain.PreflightFirewallID, "Inbound firewall rules", "UFW allows inbound 80/443", nil)
		return
	}
	fail(domain.PreflightFirewallID, "Inbound firewall rules", "UFW is active but does not allow inbound 80/443",
		"Use the Open Firewall Ports action (edge.ufw.allow through the target owner) or update the security group rules.", map[string]string{"ufw_rules": rulesRes.Stdout})
}
