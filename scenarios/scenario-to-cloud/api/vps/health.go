package vps

import (
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/apphealth"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/tlsinfo"
)

const (
	tlsALPNWarnDaysThreshold = 30
	tlsALPNFailDaysThreshold = 14
)

// ComputeHealth composes live state, DNS, and TLS results into a unified health report.
// This is a pure function with no I/O — all data is passed in as parameters.
func ComputeHealth(
	dep *domain.Deployment,
	manifest domain.CloudManifest,
	identity sshidentity.DeploymentSSHIdentity,
	liveState *domain.LiveStateResult,
	dnsEval *dns.Evaluation,
	tlsSnap *tlsinfo.Snapshot,
	tlsErr error,
	app *apphealth.Report,
) domain.HealthResponse {
	resp := domain.HealthResponse{
		OK:             true,
		DeploymentID:   dep.ID,
		DeploymentName: dep.Name,
		ScenarioID:     dep.ScenarioID,
		Domain:         manifest.Edge.Domain,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
	if manifest.Target.VPS != nil {
		resp.Host = manifest.Target.VPS.Host
	}

	var allSections []domain.HealthSection

	// --- Deployment section ---
	allSections = append(allSections, buildDeploymentSection(dep))

	// --- SSH section ---
	allSections = append(allSections, buildSSHSection(identity, liveState))

	// --- Processes section ---
	allSections = append(allSections, buildProcessesSection(liveState, manifest))

	// --- DNS section ---
	allSections = append(allSections, buildDNSSection(dnsEval, manifest))

	// --- TLS / Edge section ---
	allSections = append(allSections, buildTLSSection(liveState, tlsSnap, tlsErr))

	// --- System section ---
	allSections = append(allSections, buildSystemSection(liveState))

	// --- Application dependencies section ---
	allSections = append(allSections, buildAppDependenciesSection(app))

	resp.Sections = allSections

	// Compute totals and overall health
	var totalPass, totalWarn, totalFail int
	for _, sec := range allSections {
		totalPass += sec.PassCount
		totalWarn += sec.WarnCount
		totalFail += sec.FailCount
	}

	// Determine overall health level
	resp.Health = computeOverallHealth(dep, liveState, totalFail, totalWarn)

	// Build summary string
	resp.Summary = fmt.Sprintf("%d passed  |  %d warning  |  %d failed", totalPass, totalWarn, totalFail)

	// Build recommendations
	resp.Recommendations = buildRecommendations(dep.ID, manifest, allSections)

	return resp
}

// LiveStateReachable reports whether the live-state inspection actually
// reached the target: the inspection ran, produced system data and the SSH
// ping succeeded. RunLiveStateInspection returns OK=true even when every
// command failed (only cancellation sets OK=false), so OK alone is not
// evidence of reach; a report built on unreachable evidence must be unknown,
// never unhealthy or healthy.
func LiveStateReachable(liveState *domain.LiveStateResult) bool {
	return liveState != nil && liveState.OK && liveState.System != nil && liveState.System.SSH.Connected
}

// computeOverallHealth determines the overall health level based on deployment status and check results.
func computeOverallHealth(dep *domain.Deployment, liveState *domain.LiveStateResult, fails, warns int) domain.HealthLevel {
	// Status overrides
	switch dep.Status {
	case domain.StatusFailed:
		return domain.HealthFailed
	case domain.StatusStopped:
		return domain.HealthStopped
	case domain.StatusPending:
		return domain.HealthPending
	case domain.StatusSetupRunning, domain.StatusDeploying:
		return domain.HealthStarting
	case domain.StatusSetupComplete:
		return domain.HealthStarting
	}

	// If SSH unreachable, nothing below is evidence about the deployment.
	if !LiveStateReachable(liveState) {
		return domain.HealthUnknown
	}

	if fails > 0 {
		return domain.HealthUnhealthy
	}
	if warns > 0 {
		return domain.HealthDegraded
	}
	return domain.HealthHealthy
}

// --- Section builders ---

func buildDeploymentSection(dep *domain.Deployment) domain.HealthSection {
	sec := domain.HealthSection{
		Category: "deployment",
		Title:    "Deployment Status",
	}

	details := map[string]string{
		"status": string(dep.Status),
	}
	if dep.LastDeployedAt != nil {
		details["last_deployed"] = dep.LastDeployedAt.Format("2006-01-02 15:04 UTC")
	}

	var check domain.HealthCheck
	switch dep.Status {
	case domain.StatusDeployed:
		check = domain.HealthCheck{
			ID:      "deployment_status",
			Title:   "Deployment status",
			Status:  domain.HealthCheckPass,
			Message: "Status: deployed",
			Details: details,
		}
	case domain.StatusStopped:
		check = domain.HealthCheck{
			ID:      "deployment_status",
			Title:   "Deployment status",
			Status:  domain.HealthCheckWarn,
			Message: "Status: stopped",
			Details: details,
		}
	case domain.StatusFailed:
		msg := "Status: failed"
		if dep.ErrorMessage != nil {
			msg = fmt.Sprintf("Status: failed — %s", *dep.ErrorMessage)
		}
		check = domain.HealthCheck{
			ID:      "deployment_status",
			Title:   "Deployment status",
			Status:  domain.HealthCheckFail,
			Message: msg,
			Details: details,
		}
	default:
		check = domain.HealthCheck{
			ID:      "deployment_status",
			Title:   "Deployment status",
			Status:  domain.HealthCheckWarn,
			Message: fmt.Sprintf("Status: %s", dep.Status),
			Details: details,
		}
	}

	addCheckToSection(&sec, check)
	return sec
}

func buildSSHSection(identity sshidentity.DeploymentSSHIdentity, liveState *domain.LiveStateResult) domain.HealthSection {
	section := domain.HealthSection{Category: "ssh", Title: "SSH & Connectivity"}
	if unavailable := sshAvailabilityCheck(identity, liveState); unavailable != nil {
		addCheckToSection(&section, *unavailable)
		return section
	}
	ssh := liveState.System.SSH
	addCheckToSection(&section, sshConnectionCheck(ssh.LatencyMs))
	addCheckToSection(&section, sshIdentityCheck(identity, ssh.VerificationState))
	return section
}

func sshAvailabilityCheck(identity sshidentity.DeploymentSSHIdentity, liveState *domain.LiveStateResult) *domain.HealthCheck {
	if liveState == nil || !liveState.OK {
		message := "VPS unreachable via SSH"
		if liveState != nil && liveState.Error != "" {
			message = fmt.Sprintf("VPS unreachable: %s", liveState.Error)
		}
		return &domain.HealthCheck{ID: "ssh_connected", Title: "SSH connectivity", Status: domain.HealthCheckFail, Message: message}
	}
	if liveState.System == nil {
		return &domain.HealthCheck{ID: "ssh_connected", Title: "SSH connectivity", Status: domain.HealthCheckFail, Message: "No system data available"}
	}
	if !liveState.System.SSH.Connected {
		return &domain.HealthCheck{ID: "ssh_connected", Title: "SSH connectivity", Status: domain.HealthCheckFail,
			Message: "VPS unreachable via SSH (connectivity probe failed)", Details: map[string]string{"auth_mode": string(identity.AuthMode)}}
	}
	return nil
}

func sshConnectionCheck(latency int64) domain.HealthCheck {
	status := domain.HealthCheckPass
	message := fmt.Sprintf("Connected (latency: %dms)", latency)
	if latency > 2000 {
		status = domain.HealthCheckWarn
		message = fmt.Sprintf("Connected but slow (latency: %dms)", latency)
	}
	return domain.HealthCheck{ID: "ssh_connected", Title: "SSH connectivity", Status: status, Message: message,
		Details: map[string]string{"latency_ms": fmt.Sprintf("%d", latency)}}
}

func sshIdentityCheck(identity sshidentity.DeploymentSSHIdentity, verification string) domain.HealthCheck {
	status, message := sshIdentityStatus(identity.AuthMode, sshidentity.VerificationState(verification))
	details := map[string]string{"auth_mode": string(identity.AuthMode), "verification_state": verification}
	if identity.AuthMode == sshidentity.AuthModeExplicitKey {
		details["key_path"] = identity.KeyPath
		if identity.PublicKeyFingerprint != "" {
			details["public_key_fingerprint"] = identity.PublicKeyFingerprint
		}
	}
	return domain.HealthCheck{ID: "ssh_key_auth", Title: "Key authorization", Status: status, Message: message, Details: details}
}

func sshIdentityStatus(mode sshidentity.AuthMode, verification sshidentity.VerificationState) (domain.HealthCheckStatus, string) {
	if mode == sshidentity.AuthModeExplicitKey {
		switch verification {
		case sshidentity.VerificationAuthorized:
			return domain.HealthCheckPass, "Explicit SSH key is authorized on VPS"
		case sshidentity.VerificationUnauthorized:
			return domain.HealthCheckFail, "Explicit SSH key is not authorized on VPS"
		default:
			return domain.HealthCheckFail, "Explicit SSH key verification is unknown"
		}
	}
	switch mode {
	case sshidentity.AuthModeAgent:
		return domain.HealthCheckWarn, "Connected via SSH agent; deployment key is not pinned"
	case sshidentity.AuthModeDefaultSSH:
		return domain.HealthCheckWarn, "Connected via default SSH identity; deployment key is not pinned"
	default:
		return domain.HealthCheckWarn, "SSH identity mode is unknown"
	}
}

func buildProcessesSection(liveState *domain.LiveStateResult, manifest domain.CloudManifest) domain.HealthSection {
	sec := domain.HealthSection{Category: "processes", Title: "Processes"}
	if !LiveStateReachable(liveState) || liveState.Processes == nil {
		addCheckToSection(&sec, unavailableProcessesCheck())
		return sec
	}
	processes := liveState.Processes
	addCheckToSection(&sec, scenarioProcessCheck(processes.Scenarios, manifest.Scenario.ID))
	for _, expectedRes := range manifest.Dependencies.Resources {
		addCheckToSection(&sec, resourceProcessCheck(expectedRes, processes.Resources))
	}
	sec.Title = fmt.Sprintf("Processes (%d/%d running)", sec.PassCount, 1+len(manifest.Dependencies.Resources))
	return sec
}

func unavailableProcessesCheck() domain.HealthCheck {
	return domain.HealthCheck{ID: "processes_unavailable", Title: "Process state", Status: domain.HealthCheckSkip, Message: "Process state unavailable (SSH unreachable)"}
}

func scenarioProcessCheck(scenarios []domain.ScenarioProcess, scenarioID string) domain.HealthCheck {
	for _, scenario := range scenarios {
		if scenario.ID == scenarioID {
			return processStatusCheck("process_scenario_", scenario.ID, scenario.Status, scenario.PID, map[string]string{
				"cpu": fmt.Sprintf("%.1f%%", scenario.Resources.CPUPercent), "memory": fmt.Sprintf("%dMB", scenario.Resources.MemoryMB),
			})
		}
	}
	return domain.HealthCheck{ID: "process_scenario_" + scenarioID, Title: scenarioID, Status: domain.HealthCheckFail, Message: fmt.Sprintf("%s not running", scenarioID)}
}

func resourceProcessCheck(resourceID string, resources []domain.ResourceProcess) domain.HealthCheck {
	for _, resource := range resources {
		if resource.ID != resourceID {
			continue
		}
		details := map[string]string{"pid": fmt.Sprintf("%d", resource.PID)}
		if resource.Port > 0 {
			details["port"] = fmt.Sprintf("%d", resource.Port)
		}
		return processStatusCheck("process_resource_", resource.ID, resource.Status, resource.PID, details)
	}
	return domain.HealthCheck{ID: "process_resource_" + resourceID, Title: resourceID, Status: domain.HealthCheckFail, Message: fmt.Sprintf("%s not running", resourceID)}
}

func processStatusCheck(prefix, id, state string, pid int, details map[string]string) domain.HealthCheck {
	details["pid"] = fmt.Sprintf("%d", pid)
	status := domain.HealthCheckPass
	message := fmt.Sprintf("%s running (PID %d)", id, pid)
	if state != "running" {
		status = domain.HealthCheckFail
		message = fmt.Sprintf("%s %s", id, state)
	}
	return domain.HealthCheck{ID: prefix + id, Title: id, Status: status, Message: message, Details: details}
}

func buildDNSSection(dnsEval *dns.Evaluation, manifest domain.CloudManifest) domain.HealthSection {
	sec := domain.HealthSection{
		Category: "dns",
		Title:    "DNS",
	}

	if dnsEval == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "dns_unavailable",
			Title:   "DNS evaluation",
			Status:  domain.HealthCheckSkip,
			Message: "DNS check skipped",
		})
		return sec
	}

	vpsIP := ""
	if dnsEval.VPS.Error == nil && len(dnsEval.VPS.IPs) > 0 {
		vpsIP = dnsEval.VPS.IPs[0]
	}

	for _, ds := range dnsEval.Statuses {
		details := map[string]string{
			"role": ds.Role,
			"host": ds.Host,
		}

		if ds.Lookup.Error != nil {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "dns_" + ds.Role,
				Title:   ds.Host,
				Status:  domain.HealthCheckFail,
				Message: fmt.Sprintf("%s does not resolve", ds.Host),
				Details: details,
			})
			continue
		}

		ips := strings.Join(ds.Lookup.IPs, ", ")
		details["ips"] = ips

		if ds.Proxied {
			details["proxied"] = "cloudflare"
			if ds.AllowProxy {
				addCheckToSection(&sec, domain.HealthCheck{
					ID:      "dns_" + ds.Role,
					Title:   ds.Host,
					Status:  domain.HealthCheckPass,
					Message: fmt.Sprintf("%s -> Cloudflare proxy", ds.Host),
					Details: details,
				})
			} else {
				addCheckToSection(&sec, domain.HealthCheck{
					ID:      "dns_" + ds.Role,
					Title:   ds.Host,
					Status:  domain.HealthCheckWarn,
					Message: fmt.Sprintf("%s proxied through Cloudflare (origin record should be DNS-only)", ds.Host),
					Details: details,
				})
			}
			continue
		}

		if ds.PointsToVPS {
			target := vpsIP
			if target == "" {
				target = "VPS"
			}
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "dns_" + ds.Role,
				Title:   ds.Host,
				Status:  domain.HealthCheckPass,
				Message: fmt.Sprintf("%s -> %s (VPS)", ds.Host, target),
				Details: details,
			})
		} else {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "dns_" + ds.Role,
				Title:   ds.Host,
				Status:  domain.HealthCheckFail,
				Message: fmt.Sprintf("%s -> %s (not VPS IP %s)", ds.Host, ips, vpsIP),
				Details: details,
			})
		}
	}

	return sec
}

func buildTLSSection(liveState *domain.LiveStateResult, tlsSnap *tlsinfo.Snapshot, tlsErr error) domain.HealthSection {
	sec := domain.HealthSection{Category: "tls", Title: "TLS / Edge"}
	if check := caddyHealthCheck(liveState); check != nil {
		addCheckToSection(&sec, *check)
	}
	certificate, hasProbe := tlsCertificateHealthCheck(tlsSnap, tlsErr)
	addCheckToSection(&sec, certificate)
	if hasProbe {
		if check := alpnHealthCheck(tlsSnap); check != nil {
			addCheckToSection(&sec, *check)
		}
	}
	return sec
}

func caddyHealthCheck(liveState *domain.LiveStateResult) *domain.HealthCheck {
	if liveState == nil || liveState.Caddy == nil {
		return nil
	}
	status, message := domain.HealthCheckFail, "Caddy: not running"
	if liveState.Caddy.Running {
		status, message = domain.HealthCheckPass, "Caddy: running"
	}
	return &domain.HealthCheck{ID: "caddy_running", Title: "Caddy reverse proxy", Status: status, Message: message}
}

func tlsCertificateHealthCheck(snapshot *tlsinfo.Snapshot, probeErr error) (domain.HealthCheck, bool) {
	if probeErr != nil {
		return domain.HealthCheck{ID: "tls_cert", Title: "TLS certificate", Status: domain.HealthCheckFail, Message: fmt.Sprintf("TLS probe failed: %v", probeErr)}, false
	}
	if snapshot == nil {
		return domain.HealthCheck{ID: "tls_cert", Title: "TLS certificate", Status: domain.HealthCheckSkip, Message: "TLS check skipped"}, false
	}
	return tlsProbeCertificateHealthCheck(snapshot.Probe), true
}

func tlsProbeCertificateHealthCheck(probe tlsinfo.ProbeResult) domain.HealthCheck {
	details := map[string]string{}
	if probe.Issuer != "" {
		details["issuer"] = probe.Issuer
	}
	if probe.NotAfter != "" {
		details["expires"] = probe.NotAfter
	}
	details["days_remaining"] = fmt.Sprintf("%d", probe.DaysRemaining)
	if !probe.Valid {
		return domain.HealthCheck{ID: "tls_cert", Title: "TLS certificate", Status: domain.HealthCheckFail, Message: "Certificate invalid or expired", Details: details}
	}
	status := domain.HealthCheckPass
	if probe.DaysRemaining < 30 {
		status = domain.HealthCheckWarn
	}
	message := fmt.Sprintf("Certificate valid (%s, expires in %dd)", probe.Issuer, probe.DaysRemaining)
	return domain.HealthCheck{ID: "tls_cert", Title: "TLS certificate", Status: status, Message: message, Details: details}
}

func alpnHealthCheck(snapshot *tlsinfo.Snapshot) *domain.HealthCheck {
	alpn := snapshot.ALPN
	if alpn.Status == tlsinfo.ALPNPass {
		return &domain.HealthCheck{ID: "tls_alpn", Title: "TLS-ALPN readiness", Status: domain.HealthCheckPass, Message: alpn.Message}
	}
	if alpn.Status != tlsinfo.ALPNWarn {
		return nil
	}
	probe := snapshot.Probe
	status, message := alpnWarningOutcome(probe)
	if detail := strings.TrimSpace(alpn.Message); detail != "" {
		if status == domain.HealthCheckPass {
			message = fmt.Sprintf("%s (%s)", message, detail)
		} else {
			message = fmt.Sprintf("%s: %s", message, detail)
		}
	}
	details := map[string]string{"cert_days_remaining": fmt.Sprintf("%d", probe.DaysRemaining)}
	if alpn.Protocol != "" {
		details["protocol"] = alpn.Protocol
	}
	if alpn.Error != "" {
		details["error"] = alpn.Error
	}
	return &domain.HealthCheck{ID: "tls_alpn", Title: "TLS-ALPN readiness", Status: status, Message: message, Details: details}
}

func alpnWarningOutcome(probe tlsinfo.ProbeResult) (domain.HealthCheckStatus, string) {
	status, message := domain.HealthCheckWarn, "TLS-ALPN probe failed"
	if !probe.Valid {
		return status, message
	}
	if probe.DaysRemaining >= tlsALPNWarnDaysThreshold {
		return domain.HealthCheckPass, "TLS-ALPN readiness check is informational while certificate is healthy"
	}
	if probe.DaysRemaining < tlsALPNFailDaysThreshold {
		status = domain.HealthCheckFail
		message = fmt.Sprintf("TLS-ALPN probe failed with certificate near expiry (%dd remaining)", probe.DaysRemaining)
	} else if probe.DaysRemaining < tlsALPNWarnDaysThreshold {
		status = domain.HealthCheckWarn
		message = fmt.Sprintf("TLS-ALPN probe failed within renewal window (%dd remaining)", probe.DaysRemaining)
	}
	return status, message
}

func buildSystemSection(liveState *domain.LiveStateResult) domain.HealthSection {
	sec := domain.HealthSection{Category: "system", Title: "System Resources"}
	if !LiveStateReachable(liveState) {
		addCheckToSection(&sec, domain.HealthCheck{ID: "system_unavailable", Title: "System metrics", Status: domain.HealthCheckSkip, Message: "System metrics unavailable (SSH unreachable)"})
		return sec
	}
	sys := liveState.System
	addCheckToSection(&sec, cpuHealthCheck(sys.CPU))
	addCheckToSection(&sec, memoryHealthCheck(sys.Memory))
	addCheckToSection(&sec, diskHealthCheck(sys.Disk))
	return sec
}

func cpuHealthCheck(cpu domain.CPUInfo) domain.HealthCheck {
	details := map[string]string{"cores": fmt.Sprintf("%d", cpu.Cores), "usage": fmt.Sprintf("%.1f%%", cpu.UsagePercent)}
	cpuStatus := domain.HealthCheckPass
	message := fmt.Sprintf("CPU: %.1f%% (%d cores)", cpu.UsagePercent, cpu.Cores)
	load5PerCore := 0.0
	hasLoad5PerCore := cpu.Cores > 0 && len(cpu.LoadAverage) >= 2
	if hasLoad5PerCore {
		load5PerCore = cpu.LoadAverage[1] / float64(cpu.Cores)
		details["load5_per_core"] = fmt.Sprintf("%.2f", load5PerCore)
	}
	if cpu.UsagePercent > 95 {
		// High instantaneous CPU can spike briefly; require sustained load pressure before failing.
		if hasLoad5PerCore && load5PerCore >= 1.0 {
			cpuStatus = domain.HealthCheckFail
			message = fmt.Sprintf("CPU: %.1f%% (%d cores, load5/core %.2f)", cpu.UsagePercent, cpu.Cores, load5PerCore)
		} else {
			cpuStatus = domain.HealthCheckWarn
		}
	} else if cpu.UsagePercent > 90 {
		cpuStatus = domain.HealthCheckWarn
	}
	return domain.HealthCheck{ID: "system_cpu", Title: "CPU", Status: cpuStatus, Message: message, Details: details}
}

func memoryHealthCheck(memory domain.MemoryInfo) domain.HealthCheck {
	status := domain.HealthCheckPass
	if memory.UsagePercent > 95 {
		status = domain.HealthCheckFail
	} else if memory.UsagePercent > 85 {
		status = domain.HealthCheckWarn
	}
	message := fmt.Sprintf("Memory: %d/%d MB (%.0f%%)", memory.UsedMB, memory.TotalMB, memory.UsagePercent)
	details := map[string]string{"total_mb": fmt.Sprintf("%d", memory.TotalMB), "used_mb": fmt.Sprintf("%d", memory.UsedMB), "usage": fmt.Sprintf("%.1f%%", memory.UsagePercent)}
	return domain.HealthCheck{ID: "system_memory", Title: "Memory", Status: status, Message: message, Details: details}
}

func diskHealthCheck(disk domain.DiskInfo) domain.HealthCheck {
	status := domain.HealthCheckPass
	if disk.UsagePercent > 90 {
		status = domain.HealthCheckFail
	} else if disk.UsagePercent > 80 {
		status = domain.HealthCheckWarn
	}
	message := fmt.Sprintf("Disk: %d/%d GB (%.0f%%)", disk.UsedGB, disk.TotalGB, disk.UsagePercent)
	details := map[string]string{"total_gb": fmt.Sprintf("%d", disk.TotalGB), "used_gb": fmt.Sprintf("%d", disk.UsedGB), "usage": fmt.Sprintf("%.1f%%", disk.UsagePercent)}
	return domain.HealthCheck{ID: "system_disk", Title: "Disk", Status: status, Message: message, Details: details}
}

// appDependenciesCategory is the section category for what the deployed
// application says about its own dependencies.
const appDependenciesCategory = "app_dependencies"

// appDependencyCheckID is the check id prefix for one application dependency.
const appDependencyCheckID = "app_dependency_"

// buildAppDependenciesSection reports the deployed application's own view of
// its dependencies. A failing dependency is a warning, never a failure: the
// application is serving, so the deployment is not broken, but a capability
// the operator paid for is unavailable and nothing else in this report would
// say so. A report that could not be observed is a skip, so "not observed"
// never renders as "everything passes".
func buildAppDependenciesSection(app *apphealth.Report) domain.HealthSection {
	sec := domain.HealthSection{
		Category: appDependenciesCategory,
		Title:    "Application Dependencies",
	}

	if app == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "app_dependencies_unavailable",
			Title:   "Application dependencies",
			Status:  domain.HealthCheckSkip,
			Message: "The application health body was not read",
		})
		return sec
	}
	if app.Unavailable {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "app_dependencies_unavailable",
			Title:   "Application dependencies",
			Status:  domain.HealthCheckSkip,
			Message: app.Summary(),
			Details: map[string]string{"url": app.URL},
		})
		return sec
	}

	for _, dep := range app.Dependencies {
		status := domain.HealthCheckPass
		message := dep.Name + " is working"
		if dep.Status == apphealth.DependencyWarn {
			status = domain.HealthCheckWarn
			message = dep.Name + " is failing"
			if strings.TrimSpace(dep.Detail) != "" {
				message += ": " + strings.TrimSpace(dep.Detail)
			}
		}
		details := map[string]string{"dependency": dep.Name}
		if strings.TrimSpace(app.Status) != "" {
			details["app_status"] = strings.TrimSpace(app.Status)
		}
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      appDependencyCheckID + dep.Name,
			Title:   dep.Name,
			Status:  status,
			Message: message,
			Details: details,
		})
	}
	if len(sec.Checks) == 0 {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "app_dependencies_none",
			Title:   "Application dependencies",
			Status:  domain.HealthCheckSkip,
			Message: app.Summary(),
			Details: map[string]string{"url": app.URL},
		})
	}
	return sec
}

// --- Helpers ---

// addCheckToSection appends a check and updates the section's counters and status.
func addCheckToSection(sec *domain.HealthSection, check domain.HealthCheck) {
	sec.Checks = append(sec.Checks, check)
	switch check.Status {
	case domain.HealthCheckPass:
		sec.PassCount++
	case domain.HealthCheckWarn:
		sec.WarnCount++
	case domain.HealthCheckFail:
		sec.FailCount++
	case domain.HealthCheckError:
		sec.ErrorCount++
	}
	// Update section-level status (worst wins)
	sec.Status = worstStatus(sec.Status, check.Status)
}

// worstStatus returns the worse of two statuses.
func worstStatus(a, b domain.HealthCheckStatus) domain.HealthCheckStatus {
	order := map[domain.HealthCheckStatus]int{
		"":                      0,
		domain.HealthCheckSkip:  0,
		domain.HealthCheckPass:  1,
		domain.HealthCheckWarn:  2,
		domain.HealthCheckFail:  3,
		domain.HealthCheckError: 4,
	}
	if order[b] > order[a] {
		return b
	}
	return a
}

// buildRecommendations generates actionable recommendations from failed/warned checks.
func buildRecommendations(deploymentID string, manifest domain.CloudManifest, sections []domain.HealthSection) []domain.Recommendation {
	var recs []domain.Recommendation

	for _, sec := range sections {
		for _, check := range sec.Checks {
			if check.Status != domain.HealthCheckFail && check.Status != domain.HealthCheckWarn {
				continue
			}

			rec := checkToRecommendation(deploymentID, manifest, sec.Category, check)
			if rec != nil {
				recs = append(recs, *rec)
			}
		}
	}

	return recs
}

// checkToRecommendation maps a failed/warned check to an actionable recommendation.
func checkToRecommendation(deploymentID string, manifest domain.CloudManifest, category string, check domain.HealthCheck) *domain.Recommendation {
	if rec := sshRecommendation(manifest, category, check); rec != nil {
		return rec
	}
	if rec := processRecommendation(deploymentID, category, check); rec != nil {
		return rec
	}
	if rec := tlsRecommendation(deploymentID, check); rec != nil {
		return rec
	}
	if rec := systemRecommendation(deploymentID, check); rec != nil {
		return rec
	}
	return deploymentRecommendation(deploymentID, category, check)
}

func sshRecommendation(manifest domain.CloudManifest, category string, check domain.HealthCheck) *domain.Recommendation {
	if rec := sshConnectivityRecommendation(manifest, category, check); rec != nil {
		return rec
	}
	return sshIdentityRecommendation(manifest, category, check)
}

func sshConnectivityRecommendation(manifest domain.CloudManifest, category string, check domain.HealthCheck) *domain.Recommendation {
	if category != "ssh" || check.ID != "ssh_connected" || check.Status != domain.HealthCheckFail {
		return nil
	}
	sshHost, sshUser := sshTarget(manifest)
	sshTestCmd := "scenario-to-cloud ssh test <host>"
	if sshHost != "" {
		sshTestCmd = fmt.Sprintf("scenario-to-cloud ssh test %s --user %s", sshHost, sshUser)
	}
	return &domain.Recommendation{Priority: 1, Category: "ssh", Summary: "VPS unreachable via SSH", Command: sshTestCmd}
}

func sshIdentityRecommendation(manifest domain.CloudManifest, category string, check domain.HealthCheck) *domain.Recommendation {
	if category != "ssh" || check.ID != "ssh_key_auth" || (check.Status != domain.HealthCheckWarn && check.Status != domain.HealthCheckFail) {
		return nil
	}
	sshHost, sshUser := sshTarget(manifest)
	sshBootstrapCmd := "scenario-to-cloud ssh bootstrap <host> --user <user> --non-interactive"
	if sshHost != "" {
		sshBootstrapCmd = fmt.Sprintf("scenario-to-cloud ssh bootstrap %s --user %s --non-interactive", sshHost, sshUser)
	}
	summary := "SSH transport is not pinned to an explicit deployment key"
	if strings.TrimSpace(check.Details["auth_mode"]) == string(sshidentity.AuthModeUnknown) {
		summary = "SSH identity mode is unknown; run bootstrap and redeploy with explicit key"
	}
	if check.Status == domain.HealthCheckFail {
		summary = "Explicit SSH deployment key is not authorized on VPS"
	}
	return &domain.Recommendation{Priority: 1, Category: "ssh", Summary: summary, Command: sshBootstrapCmd}
}

func sshTarget(manifest domain.CloudManifest) (host, user string) {
	user = "root"
	if manifest.Target.VPS == nil {
		return "", user
	}
	host = strings.TrimSpace(manifest.Target.VPS.Host)
	if configured := strings.TrimSpace(manifest.Target.VPS.User); configured != "" {
		user = configured
	}
	return host, user
}

func processRecommendation(deploymentID, category string, check domain.HealthCheck) *domain.Recommendation {
	if category == "processes" && check.Status == domain.HealthCheckFail {
		return &domain.Recommendation{Priority: 1, Category: "processes", Summary: check.Message,
			// Process control stays target-bound. Deployment start is the owner-level repair path.
			Command: fmt.Sprintf("scenario-to-cloud deployment start %s", deploymentID)}
	}
	if category == "dns" && check.Status == domain.HealthCheckFail {
		return &domain.Recommendation{Priority: 2, Category: "dns", Summary: check.Message, Command: fmt.Sprintf("scenario-to-cloud edge dns-check %s", deploymentID)}
	}
	return nil
}

func tlsRecommendation(deploymentID string, check domain.HealthCheck) *domain.Recommendation {
	command := fmt.Sprintf("scenario-to-cloud edge tls-renew %s", deploymentID)
	switch check.ID {
	case "tls_cert":
		switch check.Status {
		case domain.HealthCheckFail:
			return &domain.Recommendation{Priority: 1, Category: "tls", Summary: "TLS certificate invalid or expired", Command: command}
		case domain.HealthCheckWarn:
			return &domain.Recommendation{Priority: 2, Category: "tls", Summary: fmt.Sprintf("TLS certificate expiring soon (%s days remaining)", check.Details["days_remaining"]), Command: command}
		}
	case "tls_alpn":
		switch check.Status {
		case domain.HealthCheckFail:
			return &domain.Recommendation{Priority: 1, Category: "tls", Summary: "TLS-ALPN challenge path is failing with certificate near expiry", Command: command}
		case domain.HealthCheckWarn:
			days := strings.TrimSpace(check.Details["cert_days_remaining"])
			if days == "" {
				days = "unknown"
			}
			return &domain.Recommendation{Priority: 2, Category: "tls", Summary: fmt.Sprintf("TLS-ALPN challenge path is degraded (%s days remaining)", days), Command: command}
		}
	case "caddy_running":
		if check.Status == domain.HealthCheckFail {
			return &domain.Recommendation{Priority: 1, Category: "tls", Summary: "Caddy reverse proxy is not running", Command: fmt.Sprintf("scenario-to-cloud edge caddy %s restart", deploymentID)}
		}
	}
	return nil
}

func systemRecommendation(deploymentID string, check domain.HealthCheck) *domain.Recommendation {
	if check.ID == "system_disk" {
		if check.Status == domain.HealthCheckFail {
			return &domain.Recommendation{Priority: 1, Category: "system", Summary: "Disk usage critical (>90%)", Command: fmt.Sprintf("scenario-to-cloud inspect files %s", deploymentID)}
		}
		if check.Status == domain.HealthCheckWarn {
			return &domain.Recommendation{Priority: 3, Category: "system", Summary: "Disk usage high (>80%)", Command: fmt.Sprintf("scenario-to-cloud inspect files %s", deploymentID)}
		}
	}
	if check.ID != "system_cpu" && check.ID != "system_memory" || (check.Status != domain.HealthCheckFail && check.Status != domain.HealthCheckWarn) {
		return nil
	}
	summary := "High CPU usage"
	if check.ID == "system_memory" {
		summary = "High memory usage"
	}
	return &domain.Recommendation{Priority: 2, Category: "system", Summary: summary, Command: fmt.Sprintf("scenario-to-cloud inspect live %s", deploymentID)}
}

func deploymentRecommendation(deploymentID, category string, check domain.HealthCheck) *domain.Recommendation {
	if category == "deployment" && check.Status == domain.HealthCheckFail {
		return &domain.Recommendation{Priority: 1, Category: "deployment", Summary: "Deployment in failed state", Command: fmt.Sprintf("scenario-to-cloud deployment history %s", deploymentID)}
	}
	if category == appDependenciesCategory && check.Status == domain.HealthCheckWarn {
		return &domain.Recommendation{Priority: 2, Category: appDependenciesCategory, Summary: "The deployed application reports a failing dependency: " + strings.TrimPrefix(check.ID, appDependencyCheckID), Command: fmt.Sprintf("scenario-to-cloud deployment health %s", deploymentID)}
	}
	if category != "deployment" || check.Status != domain.HealthCheckWarn || !strings.Contains(check.Message, "stopped") {
		return nil
	}
	return &domain.Recommendation{Priority: 2, Category: "deployment", Summary: "Deployment is stopped", Command: fmt.Sprintf("scenario-to-cloud deployment start %s", deploymentID)}
}
