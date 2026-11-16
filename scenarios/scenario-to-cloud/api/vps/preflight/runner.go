package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/tlsinfo"
)

// RunOptions configures optional behavior for VPS preflight.
type RunOptions struct {
	ProvidedSecrets map[string]string
	// BundleSizeBytes is the compressed release artifact that will be staged
	// on the target. A zero value uses the bounded default cushion.
	BundleSizeBytes int64
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
	configureRunOptions(&opts)
	if target.Locator.Host == "" {
		target = domain.TargetRefFromManifest(manifest)
	}
	obs := observer{reach: rr, target: target}
	checks := checkCollector{items: make([]domain.PreflightCheck, 0, 16)}
	requirements := estimateRequirements(ctx, manifest.Scenario.ID, opts)
	appendDNSChecks(ctx, dnsService, manifest, target.Locator.Host, opts, &checks)
	checkPublicPorts(ctx, target.Locator.Host, opts.PortProbe, &checks)
	checkTLSALPN(ctx, manifest, opts, &checks)
	if !checkTargetReachability(ctx, obs, target, &checks) {
		return checks.response()
	}
	checkPrivilege(ctx, obs, target, &checks)
	checkOSRelease(ctx, obs, &checks)
	checkEdgePorts(ctx, obs, &checks)
	firewallCheck(ctx, obs, checks.fail, checks.pass, checks.warn)
	checks.warn(domain.PreflightOutboundNetworkID, "Outbound network",
		"Outbound reachability is not observed before enrollment",
		"The read-only observation set cannot open connections; host.prepare reports a blocked egress at apply time (apt.packages.ensure). Ensure outbound HTTPS is allowed.", nil)
	checkCapacity(ctx, obs, manifest, target.Locator.Host, requirements, &checks)
	tools := checkHostTools(ctx, obs, target, &checks)
	checkHostServices(ctx, obs, tools, &checks)
	checkStaleScenarioProcesses(ctx, obs, manifest, checks.warn, checks.pass)
	checks.append(RunCredentialValidation(ctx, obs, manifest)...)
	return checks.response()
}

type checkCollector struct{ items []domain.PreflightCheck }

func (c *checkCollector) append(checks ...domain.PreflightCheck) {
	c.items = append(c.items, checks...)
}
func (c *checkCollector) fail(id, title, details, hint string, data map[string]string) {
	c.append(domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightFail, Details: details, Hint: hint, Data: data})
}
func (c *checkCollector) pass(id, title, details string, data map[string]string) {
	c.append(domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightPass, Details: details, Data: data})
}
func (c *checkCollector) warn(id, title, details, hint string, data map[string]string) {
	c.append(domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightWarn, Details: details, Hint: hint, Data: data})
}
func (c *checkCollector) response() domain.PreflightResponse { return finish(c.items) }

type requirementBudget struct {
	diskRequiredKB, diskRecommendedKB int64
	ramRequiredKB, ramRecommendedKB   int64
	data                              map[string]string
}

func configureRunOptions(opts *RunOptions) {
	if opts.PortProbe == nil {
		opts.PortProbe = tlsinfo.DefaultPortProbe
	}
	if opts.TLSALPNProbe == nil {
		opts.TLSALPNProbe = tlsinfo.DefaultALPNProbe
	}
	if opts.Requirements == nil {
		opts.Requirements = fetchScenarioRequirementsFromAnalyzer
	}
}

func estimateRequirements(ctx context.Context, scenarioID string, opts RunOptions) requirementBudget {
	budget := requirementBudget{
		diskRequiredKB: RequiredDiskFreeKB(opts.BundleSizeBytes), diskRecommendedKB: MinDiskFreeKB,
		ramRequiredKB: MinRAMKB, ramRecommendedKB: RecommendedRAMKB,
		data: map[string]string{
			"static_floor_disk_kb":    strconv.FormatInt(MinDiskFreeKB, 10),
			"static_floor_ram_kb":     strconv.FormatInt(MinRAMKB, 10),
			"static_floor_ram_rec_kb": strconv.FormatInt(RecommendedRAMKB, 10),
		},
	}
	if scenarioID == "" || opts.Requirements == nil {
		return finalizeRequirementBudget(budget)
	}
	estimate, err := opts.Requirements(ctx, scenarioID)
	if err != nil {
		budget.data["requirement_source"] = "static_fallback"
		budget.data["requirement_error"] = err.Error()
		return finalizeRequirementBudget(budget)
	}
	if estimate != nil {
		budget.data["requirement_source"] = estimate.Source
		budget.data["requirement_confidence"] = estimate.Confidence
		budget.data["requirement_tier"] = estimate.Tier
		budget.data["required_by_graph_ram_kb"] = strconv.FormatInt(estimate.RAMKB, 10)
		budget.data["required_by_graph_disk_kb"] = strconv.FormatInt(estimate.DiskKB, 10)
		budget.data["required_by_graph_cpu_cores"] = strconv.FormatFloat(estimate.CPUCores, 'f', -1, 64)
		confidence := strings.ToLower(strings.TrimSpace(estimate.Confidence))
		if confidence != "low" && estimate.RAMKB > budget.ramRequiredKB {
			budget.ramRequiredKB = estimate.RAMKB
		}
		budget.ramRecommendedKB = max(budget.ramRecommendedKB, estimate.RAMKB)
		budget.diskRecommendedKB = max(budget.diskRecommendedKB, estimate.DiskKB)
	}
	return finalizeRequirementBudget(budget)
}

func finalizeRequirementBudget(b requirementBudget) requirementBudget {
	b.data["effective_required_ram_kb"] = strconv.FormatInt(b.ramRequiredKB, 10)
	b.data["effective_required_disk_kb"] = strconv.FormatInt(b.diskRequiredKB, 10)
	b.data["effective_recommended_disk_kb"] = strconv.FormatInt(b.diskRecommendedKB, 10)
	b.data["effective_recommended_ram_kb"] = strconv.FormatInt(b.ramRecommendedKB, 10)
	b.data["effective_required_ram_human"] = formatBytes(b.ramRequiredKB)
	b.data["effective_required_disk_human"] = formatBytes(b.diskRequiredKB)
	b.data["effective_recommended_disk_human"] = formatBytes(b.diskRecommendedKB)
	b.data["effective_recommended_ram_human"] = formatBytes(b.ramRecommendedKB)
	return b
}

func appendDNSChecks(ctx context.Context, service dns.Service, manifest domain.CloudManifest, host string, opts RunOptions, checks *checkCollector) {
	evaluation := dns.Evaluate(ctx, service, manifest.Edge.Domain, host)
	checks.append(dns.PreflightChecksFromEvaluation(evaluation, manifest.Edge.DNSPolicy)...)
	requirements := make([]maildns.ProviderRequirement, 0, len(manifest.Edge.MailDNS))
	for _, requirement := range manifest.Edge.MailDNS {
		converted := maildns.ProviderRequirement{Name: requirement.Name, SPFMechanism: requirement.SPFMechanism, SPFHost: requirement.SPFHost}
		for _, dkim := range requirement.DKIM {
			converted.DKIM = append(converted.DKIM, maildns.DKIMRequirement{Selector: dkim.Selector, Type: dkim.Type, Expected: dkim.Expected})
		}
		requirements = append(requirements, converted)
	}
	checks.append(dns.MailDNSChecks(ctx, manifest.Edge.Domain, requirements, manifest.Edge.DNSPolicy)...)
	if manifest.Edge.Domain != "" {
		token := strings.TrimSpace(opts.ProvidedSecrets[domain.CloudflareAPITokenKey])
		checks.append(dns.ProxyModeCheck(evaluation, manifest.Edge.DNSPolicy, token))
	}
}

func checkPublicPorts(ctx context.Context, host string, probe tlsinfo.PortProbeFunc, checks *checkCollector) {
	var unreachable []string
	for _, port := range []int{80, 443} {
		if probe(ctx, host, port, 3*time.Second) != nil {
			unreachable = append(unreachable, strconv.Itoa(port))
		}
	}
	if len(unreachable) == 0 {
		checks.pass(domain.PreflightPublicPortsID, "Public ports 80/443 reachability", "Ports 80/443 reachable from the deployment runner", map[string]string{"host": host, "ports": "80,443"})
		return
	}
	checks.warn(domain.PreflightPublicPortsID, "Public ports 80/443 reachability",
		fmt.Sprintf("Unable to reach ports %s on %s from the deployment runner", strings.Join(unreachable, ","), host),
		"This is expected before the edge service is installed. The deployment will configure the edge; verify the ports after activation and open them in the provider security group if they remain unreachable.",
		map[string]string{"host": host, "ports": strings.Join(unreachable, ",")})
}

func checkTLSALPN(ctx context.Context, manifest domain.CloudManifest, opts RunOptions, checks *checkCollector) {
	if !manifest.Edge.Caddy.Enabled || strings.TrimSpace(manifest.Edge.Domain) == "" {
		return
	}
	domainName := strings.TrimSpace(manifest.Edge.Domain)
	result := tlsinfo.RunALPNCheck(ctx, domainName, opts.PortProbe, opts.TLSALPNProbe, 3*time.Second, 4*time.Second)
	data := map[string]string{"domain": domainName}
	if result.Protocol != "" {
		data["protocol"] = result.Protocol
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	if result.Status == tlsinfo.ALPNPass {
		checks.pass(domain.PreflightTLSALPNID, "TLS-ALPN compatibility", result.Message, data)
		return
	}
	checks.warn(domain.PreflightTLSALPNID, "TLS-ALPN compatibility", result.Message, result.Hint, data)
}

func checkTargetReachability(ctx context.Context, obs observer, target identity.TargetRef, checks *checkCollector) bool {
	data := map[string]string{"host": target.Locator.Host, "user": target.Locator.User, "port": strconv.Itoa(target.Locator.Port), "transport": target.Transport}
	result, err := obs.observe(ctx, "uname", "-s")
	if ok(result, err) {
		checks.pass(domain.PreflightSSHConnectID, "Target reachability", "Remote observation executed successfully", data)
		return true
	}
	detail := "Unable to run a remote observation over the bound transport"
	if err != nil {
		detail += ": " + err.Error()
	}
	checks.fail(domain.PreflightSSHConnectID, "Target reachability", detail,
		"Confirm the target is enrolled (vrooli-bridge onboard) or that the bound SSH credential and port 22 work.", data)
	return false
}

func checkPrivilege(ctx context.Context, obs observer, target identity.TargetRef, checks *checkCollector) {
	user := target.Locator.User
	if strings.EqualFold(strings.TrimSpace(user), "root") {
		checks.pass(domain.PreflightPrivilegeID, "Privilege strategy", "The bound SSH user is root; host preparation can use the target owner", map[string]string{"user": user, "strategy": "root"})
		return
	}
	result, err := obs.observe(ctx, "sudo", "-n", "-l")
	if ok(result, err) {
		checks.pass(domain.PreflightPrivilegeID, "Privilege strategy", fmt.Sprintf("Bound SSH user %q has non-interactive elevation", user), map[string]string{"user": user, "strategy": "sudo_non_interactive"})
		return
	}
	detail := fmt.Sprintf("Bound SSH user %q does not have verified non-interactive elevation", user)
	if err != nil {
		detail += ": " + err.Error()
	} else if strings.TrimSpace(result.Stderr) != "" {
		detail += ": " + strings.TrimSpace(result.Stderr)
	}
	checks.fail(domain.PreflightPrivilegeID, "Privilege strategy", detail,
		"Grant the bound user the required non-interactive elevation through the target owner, or connect as an approved privileged user before host preparation.",
		map[string]string{"user": user, "strategy": "sudo_non_interactive"})
}

func checkOSRelease(ctx context.Context, obs observer, checks *checkCollector) {
	result, err := obs.observe(ctx, "cat", "/etc/os-release")
	if !ok(result, err) {
		checks.fail(domain.PreflightOSReleaseID, "Ubuntu version", "Unable to read /etc/os-release", "Ensure the VPS is running Ubuntu and that /etc/os-release is readable.", map[string]string{"stderr": result.Stderr})
		return
	}
	id, version := parseOSRelease(result.Stdout)
	data := map[string]string{"id": id, "version_id": version}
	switch {
	case id != SupportedOSID:
		checks.fail(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Unsupported OS: %s", id), "scenario-to-cloud requires Ubuntu. Non-Ubuntu systems are not supported.", data)
	case version == RecommendedUbuntuVersion:
		checks.pass(domain.PreflightOSReleaseID, "Ubuntu version", "Ubuntu 24.04 detected", data)
	case version == SupportedUbuntuAltVersion || version == LegacyUbuntuAltVersion:
		checks.warn(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Ubuntu %s detected (%s recommended)", version, RecommendedUbuntuVersion),
			fmt.Sprintf("Ubuntu %s/%s should work but %s LTS is recommended for best compatibility.", SupportedUbuntuAltVersion, LegacyUbuntuAltVersion, RecommendedUbuntuVersion), data)
	default:
		checks.warn(domain.PreflightOSReleaseID, "Ubuntu version", fmt.Sprintf("Ubuntu %s detected (%s recommended)", version, RecommendedUbuntuVersion),
			fmt.Sprintf("This Ubuntu version is untested. Consider using Ubuntu %s LTS.", RecommendedUbuntuVersion), data)
	}
}

func checkEdgePorts(ctx context.Context, obs observer, checks *checkCollector) {
	result, err := obs.listeningSockets(ctx, 80, 443)
	if !ok(result, err) {
		checks.warn(domain.PreflightPortsEdgeID, "Ports 80/443 availability", "Unable to check ports 80/443 via ss", "Ensure ports 80 and 443 are free for Caddy/Let's Encrypt HTTP-01.", map[string]string{"stderr": result.Stderr})
		return
	}
	if strings.TrimSpace(result.Stdout) == "" {
		checks.pass(domain.PreflightPortsEdgeID, "Ports 80/443 availability", "Ports 80/443 appear free", nil)
		return
	}
	bindings := parsePortBindings(result.Stdout)
	details := "Port 80 and/or 443 appears to already be in use"
	if len(bindings) > 0 {
		details = fmt.Sprintf("Ports in use: %s", formatPortBindings(bindings))
	}
	data := map[string]string{"ss": result.Stdout}
	if len(bindings) > 0 {
		if encoded, marshalErr := json.Marshal(bindings); marshalErr == nil {
			data["port_bindings"] = string(encoded)
		}
		data["ports_in_use"] = strings.Join(portBindingPorts(bindings), ",")
		data["processes"] = strings.Join(portBindingProcessList(bindings), ", ")
	}
	if allCaddyBindings(bindings) {
		checks.pass(domain.PreflightPortsEdgeID, "Ports 80/443 availability", fmt.Sprintf("%s (expected edge owner: caddy)", details), data)
		return
	}
	hint := "Ports 80/443 must be free for Caddy to complete Let's Encrypt HTTP-01 challenges."
	if len(bindings) > 0 {
		hint += " Stop the owning scenario through its lifecycle owner (Stop Processes) or the unit through the host's service manager."
	}
	checks.fail(domain.PreflightPortsEdgeID, "Ports 80/443 availability", details, hint, data)
}

func allCaddyBindings(bindings []EdgePortBinding) bool {
	if len(bindings) == 0 {
		return false
	}
	for _, binding := range bindings {
		if !strings.EqualFold(binding.Process, "caddy") && !strings.EqualFold(binding.Service, "caddy") {
			return false
		}
	}
	return true
}

func checkCapacity(ctx context.Context, obs observer, manifest domain.CloudManifest, host string, budget requirementBudget, checks *checkCollector) {
	checkDiskCapacity(ctx, obs, manifest, host, budget, checks)
	checkRAMCapacity(ctx, obs, budget, checks)
}

func checkDiskCapacity(ctx context.Context, obs observer, manifest domain.CloudManifest, host string, budget requirementBudget, checks *checkCollector) {
	result, err := obs.observe(ctx, "df", "-Pk", "/")
	_, _, available, _, found := dfRoot(result)
	if !ok(result, err) || !found {
		checks.warn(domain.PreflightDiskFreeID, "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", map[string]string{"stderr": result.Stderr})
		return
	}
	data := map[string]string{"free_kb": strconv.FormatInt(available, 10), "free_human": formatBytes(available), "required_min_kb": strconv.FormatInt(budget.diskRequiredKB, 10), "required_min_human": formatBytes(budget.diskRequiredKB)}
	for key, value := range budget.data {
		data[key] = value
	}
	if available > 0 && available < budget.diskRequiredKB {
		hint := fmt.Sprintf("At least %s free space is required for this deployment. Free space with the disk cleanup action (journal vacuum, docker prune) or release garbage collection: scenario-to-cloud bundle vps-gc --host %s --scenario %s --keep 2", formatBytes(budget.diskRequiredKB), host, manifest.Scenario.ID)
		checks.fail(domain.PreflightDiskFreeID, "Disk free space", fmt.Sprintf("Low free disk space: %s", formatBytes(available)), hint, data)
	} else if available > 0 && available < budget.diskRecommendedKB {
		details := fmt.Sprintf("Free space: %s (below the analyzer's advisory estimate of %s)", formatBytes(available), formatBytes(budget.diskRecommendedKB))
		checks.warn(domain.PreflightDiskFreeID, "Disk free space", details, "The analyzer estimate has low confidence and is advisory. Deployment may proceed, but clean unused releases and package caches before or after activation if space becomes tight.", data)
	} else {
		checks.pass(domain.PreflightDiskFreeID, "Disk free space", fmt.Sprintf("Free space: %s", formatBytes(available)), data)
	}
}

func checkRAMCapacity(ctx context.Context, obs observer, budget requirementBudget, checks *checkCollector) {
	result, err := obs.observe(ctx, "grep", "MemTotal", "/proc/meminfo")
	kb, found := memTotalKB(result)
	if !ok(result, err) || !found {
		checks.warn(domain.PreflightRAMTotalID, "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", map[string]string{"stderr": result.Stderr})
		return
	}
	data := map[string]string{"memtotal_kb": strconv.FormatInt(kb, 10), "memtotal_human": formatBytes(kb), "required_min_kb": strconv.FormatInt(budget.ramRequiredKB, 10), "required_min_human": formatBytes(budget.ramRequiredKB), "recommended_min_kb": strconv.FormatInt(budget.ramRecommendedKB, 10), "recommended_min_human": formatBytes(budget.ramRecommendedKB), "required_by_graph_ram_mb": strconv.FormatFloat(math.Ceil(float64(budget.ramRequiredKB)/1024), 'f', -1, 64)}
	for key, value := range budget.data {
		data[key] = value
	}
	switch {
	case kb > 0 && kb < budget.ramRequiredKB:
		checks.fail(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("Low RAM: %s", formatBytes(kb)), fmt.Sprintf("At least %s RAM is required for this deployment. %s is recommended.", formatBytes(budget.ramRequiredKB), formatBytes(budget.ramRecommendedKB)), data)
	case kb > 0 && kb < budget.ramRecommendedKB:
		checks.warn(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("RAM: %s (%s recommended)", formatBytes(kb), formatBytes(budget.ramRecommendedKB)), "Your VPS has limited RAM for this deployment profile. Consider upgrading for better performance.", data)
	default:
		checks.pass(domain.PreflightRAMTotalID, "RAM", fmt.Sprintf("RAM: %s", formatBytes(kb)), data)
	}
}

func checkHostTools(ctx context.Context, obs observer, target identity.TargetRef, checks *checkCollector) map[string]string {
	tools := obs.findTools(ctx, "curl", "git", "unzip", "tar", "jq", "apt-get", "docker", "systemctl")
	for _, cmd := range []struct{ name, id string }{{"curl", domain.PreflightCmdCurlID}, {"git", domain.PreflightCmdGitID}, {"unzip", domain.PreflightCmdUnzipID}, {"tar", domain.PreflightCmdTarID}, {"jq", domain.PreflightCmdJqID}} {
		if path, found := tools[cmd.name]; found {
			checks.pass(cmd.id, cmd.name+" available", "Found at "+path, nil)
		} else {
			checks.warn(cmd.id, cmd.name+" available", cmd.name+" not found on VPS", "host.prepare installs it through apt.packages.ensure", nil)
		}
	}
	if _, found := tools["apt-get"]; !found {
		checks.fail(domain.PreflightAptAccessID, "apt accessible", "apt-get not found on the target", "Bootstrap installs packages through apt.packages.ensure; the target must be Ubuntu.", nil)
	} else if user := target.Locator.User; user != "" && user != "root" {
		checks.warn(domain.PreflightAptAccessID, "apt accessible", "apt-get present; the bound user is not root", "Package installs run through the privilege broker; the bound user needs its elevation grant.", map[string]string{"user": user})
	} else {
		checks.pass(domain.PreflightAptAccessID, "apt accessible", "apt-get is present", nil)
	}
	return tools
}

func checkHostServices(ctx context.Context, obs observer, tools map[string]string, checks *checkCollector) {
	dockerPath, found := tools["docker"]
	switch {
	case found && obs.exists(ctx, "/var/run/docker.sock"):
		checks.pass(domain.PreflightDockerID, "Docker available", "Docker present at "+dockerPath+" (daemon socket present)", nil)
	case found:
		checks.warn(domain.PreflightDockerID, "Docker available", "Docker is installed but its daemon socket is absent", "Verify dockerd is running (systemctl status docker).", map[string]string{"path": dockerPath})
	default:
		checks.warn(domain.PreflightDockerID, "Docker available", "Docker not found", "Bootstrap will attempt to install Docker through apt.packages.ensure.", nil)
	}
	if obs.exists(ctx, "/run/systemd/system") {
		checks.pass(domain.PreflightSystemdID, "systemd available", "systemd is the running init system", nil)
	} else {
		checks.warn(domain.PreflightSystemdID, "systemd available", "systemd is not the running init system", "Deployment expects systemd for service management. Non-systemd systems (Alpine/OpenRC) are not supported.", nil)
	}
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
