package vps

import (
	"fmt"
	"strings"
	"time"

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

	// If SSH unreachable and no live state data
	if liveState == nil || !liveState.OK {
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
	sec := domain.HealthSection{
		Category: "ssh",
		Title:    "SSH & Connectivity",
	}

	if liveState == nil || !liveState.OK {
		msg := "VPS unreachable via SSH"
		if liveState != nil && liveState.Error != "" {
			msg = fmt.Sprintf("VPS unreachable: %s", liveState.Error)
		}
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "ssh_connected",
			Title:   "SSH connectivity",
			Status:  domain.HealthCheckFail,
			Message: msg,
		})
		return sec
	}

	system := liveState.System
	if system == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "ssh_connected",
			Title:   "SSH connectivity",
			Status:  domain.HealthCheckFail,
			Message: "No system data available",
		})
		return sec
	}

	// Connection check
	details := map[string]string{
		"latency_ms": fmt.Sprintf("%d", system.SSH.LatencyMs),
	}
	status := domain.HealthCheckPass
	msg := fmt.Sprintf("Connected (latency: %dms)", system.SSH.LatencyMs)
	if system.SSH.LatencyMs > 2000 {
		status = domain.HealthCheckWarn
		msg = fmt.Sprintf("Connected but slow (latency: %dms)", system.SSH.LatencyMs)
	}
	addCheckToSection(&sec, domain.HealthCheck{
		ID:      "ssh_connected",
		Title:   "SSH connectivity",
		Status:  status,
		Message: msg,
		Details: details,
	})

	// Canonical identity check
	var keyStatus domain.HealthCheckStatus
	var keyMsg string
	keyDetails := map[string]string{
		"auth_mode":          string(identity.AuthMode),
		"verification_state": system.SSH.VerificationState,
	}
	switch identity.AuthMode {
	case sshidentity.AuthModeExplicitKey:
		keyDetails["key_path"] = identity.KeyPath
		if identity.PublicKeyFingerprint != "" {
			keyDetails["public_key_fingerprint"] = identity.PublicKeyFingerprint
		}
		switch sshidentity.VerificationState(system.SSH.VerificationState) {
		case sshidentity.VerificationAuthorized:
			keyStatus = domain.HealthCheckPass
			keyMsg = "Explicit SSH key is authorized on VPS"
		case sshidentity.VerificationUnauthorized:
			keyStatus = domain.HealthCheckFail
			keyMsg = "Explicit SSH key is not authorized on VPS"
		default:
			keyStatus = domain.HealthCheckFail
			keyMsg = "Explicit SSH key verification is unknown"
		}
	case sshidentity.AuthModeAgent:
		keyStatus = domain.HealthCheckWarn
		keyMsg = "Connected via SSH agent; deployment key is not pinned"
	case sshidentity.AuthModeDefaultSSH:
		keyStatus = domain.HealthCheckWarn
		keyMsg = "Connected via default SSH identity; deployment key is not pinned"
	default:
		keyStatus = domain.HealthCheckWarn
		keyMsg = "SSH identity mode is unknown"
	}
	addCheckToSection(&sec, domain.HealthCheck{
		ID:      "ssh_key_auth",
		Title:   "Key authorization",
		Status:  keyStatus,
		Message: keyMsg,
		Details: keyDetails,
	})

	return sec
}

func buildProcessesSection(liveState *domain.LiveStateResult, manifest domain.CloudManifest) domain.HealthSection {
	sec := domain.HealthSection{
		Category: "processes",
		Title:    "Processes",
	}

	if liveState == nil || !liveState.OK || liveState.Processes == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "processes_unavailable",
			Title:   "Process state",
			Status:  domain.HealthCheckSkip,
			Message: "Process state unavailable (SSH unreachable)",
		})
		return sec
	}

	processes := liveState.Processes

	// Check main scenario
	scenarioRunning := false
	for _, s := range processes.Scenarios {
		if s.ID == manifest.Scenario.ID {
			scenarioRunning = s.Status == "running"
			details := map[string]string{
				"pid":    fmt.Sprintf("%d", s.PID),
				"cpu":    fmt.Sprintf("%.1f%%", s.Resources.CPUPercent),
				"memory": fmt.Sprintf("%dMB", s.Resources.MemoryMB),
			}
			status := domain.HealthCheckPass
			msg := fmt.Sprintf("%s running (PID %d)", s.ID, s.PID)
			if s.Status != "running" {
				status = domain.HealthCheckFail
				msg = fmt.Sprintf("%s %s", s.ID, s.Status)
			}
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "process_scenario_" + s.ID,
				Title:   s.ID,
				Status:  status,
				Message: msg,
				Details: details,
			})
			break
		}
	}
	if !scenarioRunning {
		// Check if it's not in the list at all
		found := false
		for _, s := range processes.Scenarios {
			if s.ID == manifest.Scenario.ID {
				found = true
				break
			}
		}
		if !found {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "process_scenario_" + manifest.Scenario.ID,
				Title:   manifest.Scenario.ID,
				Status:  domain.HealthCheckFail,
				Message: fmt.Sprintf("%s not running", manifest.Scenario.ID),
			})
		}
	}

	// Check expected resources
	runningResources := make(map[string]bool)
	for _, r := range processes.Resources {
		runningResources[r.ID] = r.Status == "running"
	}

	for _, expectedRes := range manifest.Dependencies.Resources {
		found := false
		for _, r := range processes.Resources {
			if r.ID == expectedRes {
				found = true
				details := map[string]string{
					"pid": fmt.Sprintf("%d", r.PID),
				}
				if r.Port > 0 {
					details["port"] = fmt.Sprintf("%d", r.Port)
				}
				status := domain.HealthCheckPass
				msg := fmt.Sprintf("%s running (PID %d)", r.ID, r.PID)
				if r.Status != "running" {
					status = domain.HealthCheckFail
					msg = fmt.Sprintf("%s %s", r.ID, r.Status)
				}
				addCheckToSection(&sec, domain.HealthCheck{
					ID:      "process_resource_" + r.ID,
					Title:   r.ID,
					Status:  status,
					Message: msg,
					Details: details,
				})
				break
			}
		}
		if !found {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "process_resource_" + expectedRes,
				Title:   expectedRes,
				Status:  domain.HealthCheckFail,
				Message: fmt.Sprintf("%s not running", expectedRes),
			})
		}
	}

	// Update title with running count
	totalExpected := 1 + len(manifest.Dependencies.Resources) // scenario + resources
	sec.Title = fmt.Sprintf("Processes (%d/%d running)", sec.PassCount, totalExpected)

	return sec
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
	sec := domain.HealthSection{
		Category: "tls",
		Title:    "TLS / Edge",
	}

	// Caddy status
	if liveState != nil && liveState.Caddy != nil {
		caddy := liveState.Caddy
		if caddy.Running {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "caddy_running",
				Title:   "Caddy reverse proxy",
				Status:  domain.HealthCheckPass,
				Message: "Caddy: running",
			})
		} else {
			addCheckToSection(&sec, domain.HealthCheck{
				ID:      "caddy_running",
				Title:   "Caddy reverse proxy",
				Status:  domain.HealthCheckFail,
				Message: "Caddy: not running",
			})
		}
	}

	// TLS certificate
	if tlsErr != nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_cert",
			Title:   "TLS certificate",
			Status:  domain.HealthCheckFail,
			Message: fmt.Sprintf("TLS probe failed: %v", tlsErr),
		})
		return sec
	}

	if tlsSnap == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_cert",
			Title:   "TLS certificate",
			Status:  domain.HealthCheckSkip,
			Message: "TLS check skipped",
		})
		return sec
	}

	probe := tlsSnap.Probe
	details := map[string]string{}
	if probe.Issuer != "" {
		details["issuer"] = probe.Issuer
	}
	if probe.NotAfter != "" {
		details["expires"] = probe.NotAfter
	}
	details["days_remaining"] = fmt.Sprintf("%d", probe.DaysRemaining)

	if !probe.Valid {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_cert",
			Title:   "TLS certificate",
			Status:  domain.HealthCheckFail,
			Message: "Certificate invalid or expired",
			Details: details,
		})
	} else if probe.DaysRemaining < 30 {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_cert",
			Title:   "TLS certificate",
			Status:  domain.HealthCheckWarn,
			Message: fmt.Sprintf("Certificate valid (%s, expires in %dd)", probe.Issuer, probe.DaysRemaining),
			Details: details,
		})
	} else {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_cert",
			Title:   "TLS certificate",
			Status:  domain.HealthCheckPass,
			Message: fmt.Sprintf("Certificate valid (%s, expires in %dd)", probe.Issuer, probe.DaysRemaining),
			Details: details,
		})
	}

	// ALPN check
	alpn := tlsSnap.ALPN
	if alpn.Status == tlsinfo.ALPNPass {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_alpn",
			Title:   "TLS-ALPN readiness",
			Status:  domain.HealthCheckPass,
			Message: alpn.Message,
		})
	} else if alpn.Status == tlsinfo.ALPNWarn {
		status := domain.HealthCheckWarn
		message := "TLS-ALPN probe failed"
		if probe.Valid && probe.DaysRemaining >= tlsALPNWarnDaysThreshold {
			status = domain.HealthCheckPass
			message = "TLS-ALPN readiness check is informational while certificate is healthy"
		} else if probe.Valid && probe.DaysRemaining < tlsALPNFailDaysThreshold {
			status = domain.HealthCheckFail
			message = fmt.Sprintf("TLS-ALPN probe failed with certificate near expiry (%dd remaining)", probe.DaysRemaining)
		} else if probe.Valid && probe.DaysRemaining < tlsALPNWarnDaysThreshold {
			status = domain.HealthCheckWarn
			message = fmt.Sprintf("TLS-ALPN probe failed within renewal window (%dd remaining)", probe.DaysRemaining)
		}
		if strings.TrimSpace(alpn.Message) != "" {
			if status == domain.HealthCheckPass {
				message = fmt.Sprintf("%s (%s)", message, strings.TrimSpace(alpn.Message))
			} else {
				message = fmt.Sprintf("%s: %s", message, strings.TrimSpace(alpn.Message))
			}
		}
		details := map[string]string{
			"cert_days_remaining": fmt.Sprintf("%d", probe.DaysRemaining),
		}
		if alpn.Protocol != "" {
			details["protocol"] = alpn.Protocol
		}
		if alpn.Error != "" {
			details["error"] = alpn.Error
		}
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "tls_alpn",
			Title:   "TLS-ALPN readiness",
			Status:  status,
			Message: message,
			Details: details,
		})
	}

	return sec
}

func buildSystemSection(liveState *domain.LiveStateResult) domain.HealthSection {
	sec := domain.HealthSection{
		Category: "system",
		Title:    "System Resources",
	}

	if liveState == nil || !liveState.OK || liveState.System == nil {
		addCheckToSection(&sec, domain.HealthCheck{
			ID:      "system_unavailable",
			Title:   "System metrics",
			Status:  domain.HealthCheckSkip,
			Message: "System metrics unavailable (SSH unreachable)",
		})
		return sec
	}

	sys := liveState.System

	// CPU
	cpuDetails := map[string]string{
		"cores": fmt.Sprintf("%d", sys.CPU.Cores),
		"usage": fmt.Sprintf("%.1f%%", sys.CPU.UsagePercent),
	}
	cpuStatus := domain.HealthCheckPass
	cpuMsg := fmt.Sprintf("CPU: %.1f%% (%d cores)", sys.CPU.UsagePercent, sys.CPU.Cores)
	load5PerCore := 0.0
	hasLoad5PerCore := sys.CPU.Cores > 0 && len(sys.CPU.LoadAverage) >= 2
	if hasLoad5PerCore {
		load5PerCore = sys.CPU.LoadAverage[1] / float64(sys.CPU.Cores)
		cpuDetails["load5_per_core"] = fmt.Sprintf("%.2f", load5PerCore)
	}
	if sys.CPU.UsagePercent > 95 {
		// High instantaneous CPU can spike briefly; require sustained load pressure before failing.
		if hasLoad5PerCore && load5PerCore >= 1.0 {
			cpuStatus = domain.HealthCheckFail
			cpuMsg = fmt.Sprintf("CPU: %.1f%% (%d cores, load5/core %.2f)", sys.CPU.UsagePercent, sys.CPU.Cores, load5PerCore)
		} else {
			cpuStatus = domain.HealthCheckWarn
		}
	} else if sys.CPU.UsagePercent > 90 {
		cpuStatus = domain.HealthCheckWarn
	}
	addCheckToSection(&sec, domain.HealthCheck{
		ID:      "system_cpu",
		Title:   "CPU",
		Status:  cpuStatus,
		Message: cpuMsg,
		Details: cpuDetails,
	})

	// Memory
	memDetails := map[string]string{
		"total_mb": fmt.Sprintf("%d", sys.Memory.TotalMB),
		"used_mb":  fmt.Sprintf("%d", sys.Memory.UsedMB),
		"usage":    fmt.Sprintf("%.1f%%", sys.Memory.UsagePercent),
	}
	memStatus := domain.HealthCheckPass
	memMsg := fmt.Sprintf("Memory: %d/%d MB (%.0f%%)", sys.Memory.UsedMB, sys.Memory.TotalMB, sys.Memory.UsagePercent)
	if sys.Memory.UsagePercent > 95 {
		memStatus = domain.HealthCheckFail
	} else if sys.Memory.UsagePercent > 85 {
		memStatus = domain.HealthCheckWarn
	}
	addCheckToSection(&sec, domain.HealthCheck{
		ID:      "system_memory",
		Title:   "Memory",
		Status:  memStatus,
		Message: memMsg,
		Details: memDetails,
	})

	// Disk
	diskDetails := map[string]string{
		"total_gb": fmt.Sprintf("%d", sys.Disk.TotalGB),
		"used_gb":  fmt.Sprintf("%d", sys.Disk.UsedGB),
		"usage":    fmt.Sprintf("%.1f%%", sys.Disk.UsagePercent),
	}
	diskStatus := domain.HealthCheckPass
	diskMsg := fmt.Sprintf("Disk: %d/%d GB (%.0f%%)", sys.Disk.UsedGB, sys.Disk.TotalGB, sys.Disk.UsagePercent)
	if sys.Disk.UsagePercent > 90 {
		diskStatus = domain.HealthCheckFail
	} else if sys.Disk.UsagePercent > 80 {
		diskStatus = domain.HealthCheckWarn
	}
	addCheckToSection(&sec, domain.HealthCheck{
		ID:      "system_disk",
		Title:   "Disk",
		Status:  diskStatus,
		Message: diskMsg,
		Details: diskDetails,
	})

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
	sshHost := ""
	sshUser := "root"
	if manifest.Target.VPS != nil {
		sshHost = strings.TrimSpace(manifest.Target.VPS.Host)
		if strings.TrimSpace(manifest.Target.VPS.User) != "" {
			sshUser = strings.TrimSpace(manifest.Target.VPS.User)
		}
	}
	sshTestCmd := "scenario-to-cloud ssh test <host>"
	sshBootstrapCmd := "scenario-to-cloud ssh bootstrap <host> --user <user> --non-interactive"
	if sshHost != "" {
		sshTestCmd = fmt.Sprintf("scenario-to-cloud ssh test %s --user %s", sshHost, sshUser)
		sshBootstrapCmd = fmt.Sprintf("scenario-to-cloud ssh bootstrap %s --user %s --non-interactive", sshHost, sshUser)
	}

	switch {
	case category == "ssh" && check.ID == "ssh_connected" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "ssh",
			Summary:  "VPS unreachable via SSH",
			Command:  sshTestCmd,
		}
	case category == "ssh" && check.ID == "ssh_key_auth" && (check.Status == domain.HealthCheckWarn || check.Status == domain.HealthCheckFail):
		summary := "SSH transport is not pinned to an explicit deployment key"
		if strings.TrimSpace(check.Details["auth_mode"]) == string(sshidentity.AuthModeUnknown) {
			summary = "SSH identity mode is unknown; run bootstrap and redeploy with explicit key"
		}
		if check.Status == domain.HealthCheckFail {
			summary = "Explicit SSH deployment key is not authorized on VPS"
		}
		return &domain.Recommendation{
			Priority: 1,
			Category: "ssh",
			Summary:  summary,
			Command:  sshBootstrapCmd,
		}
	case category == "processes" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "processes",
			Summary:  check.Message,
			Command:  fmt.Sprintf("scenario-to-cloud process control %s restart", deploymentID),
		}
	case category == "dns" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 2,
			Category: "dns",
			Summary:  check.Message,
			Command:  fmt.Sprintf("scenario-to-cloud edge dns-check %s", deploymentID),
		}
	case check.ID == "tls_cert" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "tls",
			Summary:  "TLS certificate invalid or expired",
			Command:  fmt.Sprintf("scenario-to-cloud edge tls-renew %s", deploymentID),
		}
	case check.ID == "tls_cert" && check.Status == domain.HealthCheckWarn:
		days := check.Details["days_remaining"]
		return &domain.Recommendation{
			Priority: 2,
			Category: "tls",
			Summary:  fmt.Sprintf("TLS certificate expiring soon (%s days remaining)", days),
			Command:  fmt.Sprintf("scenario-to-cloud edge tls-renew %s", deploymentID),
		}
	case check.ID == "tls_alpn" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "tls",
			Summary:  "TLS-ALPN challenge path is failing with certificate near expiry",
			Command:  fmt.Sprintf("scenario-to-cloud edge tls-renew %s", deploymentID),
		}
	case check.ID == "tls_alpn" && check.Status == domain.HealthCheckWarn:
		days := strings.TrimSpace(check.Details["cert_days_remaining"])
		if days == "" {
			days = "unknown"
		}
		return &domain.Recommendation{
			Priority: 2,
			Category: "tls",
			Summary:  fmt.Sprintf("TLS-ALPN challenge path is degraded (%s days remaining)", days),
			Command:  fmt.Sprintf("scenario-to-cloud edge tls-renew %s", deploymentID),
		}
	case check.ID == "caddy_running" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "tls",
			Summary:  "Caddy reverse proxy is not running",
			Command:  fmt.Sprintf("scenario-to-cloud edge caddy %s restart", deploymentID),
		}
	case check.ID == "system_disk" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "system",
			Summary:  "Disk usage critical (>90%)",
			Command:  fmt.Sprintf("scenario-to-cloud inspect files %s", deploymentID),
		}
	case check.ID == "system_disk" && check.Status == domain.HealthCheckWarn:
		return &domain.Recommendation{
			Priority: 3,
			Category: "system",
			Summary:  "Disk usage high (>80%)",
			Command:  fmt.Sprintf("scenario-to-cloud inspect files %s", deploymentID),
		}
	case check.ID == "system_cpu" && (check.Status == domain.HealthCheckFail || check.Status == domain.HealthCheckWarn):
		return &domain.Recommendation{
			Priority: 2,
			Category: "system",
			Summary:  "High CPU usage",
			Command:  fmt.Sprintf("scenario-to-cloud inspect live %s", deploymentID),
		}
	case check.ID == "system_memory" && (check.Status == domain.HealthCheckFail || check.Status == domain.HealthCheckWarn):
		return &domain.Recommendation{
			Priority: 2,
			Category: "system",
			Summary:  "High memory usage",
			Command:  fmt.Sprintf("scenario-to-cloud inspect live %s", deploymentID),
		}
	case category == "deployment" && check.Status == domain.HealthCheckFail:
		return &domain.Recommendation{
			Priority: 1,
			Category: "deployment",
			Summary:  "Deployment in failed state",
			Command:  fmt.Sprintf("scenario-to-cloud deployment history %s", deploymentID),
		}
	case category == "deployment" && check.Status == domain.HealthCheckWarn:
		if strings.Contains(check.Message, "stopped") {
			return &domain.Recommendation{
				Priority: 2,
				Category: "deployment",
				Summary:  "Deployment is stopped",
				Command:  fmt.Sprintf("scenario-to-cloud deployment start %s", deploymentID),
			}
		}
		return nil
	}

	return nil
}
