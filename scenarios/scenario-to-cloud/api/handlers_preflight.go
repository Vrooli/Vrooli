package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// handlePreflight runs VPS preflight checks to validate the target is ready for deployment.
// POST /api/v1/preflight
func (s *Server) handlePreflight(w http.ResponseWriter, r *http.Request) {
	manifest, err := decodeJSON[CloudManifest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if hasBlockingIssues(issues) {
		writeJSON(w, http.StatusUnprocessableEntity, ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	resp := RunVPSPreflight(ctx, normalized, s.dnsService, s.sshRunner)
	resp.Issues = issues
	writeJSON(w, http.StatusOK, resp)
}

// StopPortServicesRequest is the request body for stopping services on ports 80/443
type StopPortServicesRequest struct {
	Host              string `json:"host"`
	Port              int    `json:"port,omitempty"`
	User              string `json:"user,omitempty"`
	KeyPath            string `json:"key_path"`
	Ports             []int  `json:"ports,omitempty"`
	PIDs              []int  `json:"pids,omitempty"`
	Services          []string `json:"services,omitempty"`
	PreferServiceStop *bool  `json:"prefer_service_stop,omitempty"`
}

// StopPortServicesResponse is the response from stopping port services
type StopPortServicesResponse struct {
	OK        bool     `json:"ok"`
	Stopped   []string `json:"stopped"`
	Failed    []string `json:"failed,omitempty"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
}

// DiskUsageRequest is the request body for getting disk usage details
type DiskUsageRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
}

// DiskUsageEntry represents a single disk usage entry
type DiskUsageEntry struct {
	Path  string `json:"path"`
	Size  string `json:"size"`
	Bytes int64  `json:"bytes"`
}

// DiskUsageResponse is the response from disk usage check
type DiskUsageResponse struct {
	OK          bool             `json:"ok"`
	FreeSpace   string           `json:"free_space"`
	FreeBytes   int64            `json:"free_bytes"`
	TotalSpace  string           `json:"total_space"`
	TotalBytes  int64            `json:"total_bytes"`
	UsedPercent int              `json:"used_percent"`
	LargestDirs []DiskUsageEntry `json:"largest_dirs"`
	Timestamp   string           `json:"timestamp"`
}

// DiskCleanupRequest is the request body for running disk cleanup
type DiskCleanupRequest struct {
	Host    string   `json:"host"`
	Port    int      `json:"port,omitempty"`
	User    string   `json:"user,omitempty"`
	KeyPath string   `json:"key_path"`
	Actions []string `json:"actions"` // apt_clean, journal_vacuum, docker_prune
}

// DiskCleanupResponse is the response from disk cleanup
type DiskCleanupResponse struct {
	OK            bool     `json:"ok"`
	SpaceFreed    string   `json:"space_freed"`
	SpaceFreedKB  int64    `json:"space_freed_kb"`
	Message       string   `json:"message"`
	ActionsRun    []string `json:"actions_run"`
	ActionsFailed []string `json:"actions_failed,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

func (s *Server) handleStopPortServices(w http.ResponseWriter, r *http.Request) {
	var req StopPortServicesRequest
	if !decodeRequestBody(w, r, &req) {
		return
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	preferServiceStop := req.PreferServiceStop == nil || *req.PreferServiceStop

	pids, portsDetected, err := s.collectPortStopPIDs(ctx, cfg, req)
	if err != nil {
		writeJSON(w, http.StatusOK, StopPortServicesResponse{
			OK:        false,
			Message:   "Failed to check ports: " + err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var stopped, failed []string
	stoppedUnits := make(map[string]bool)

	if len(pids) == 0 && len(req.Services) == 0 && len(req.Ports) == 0 && !portsDetected {
		writeJSON(w, http.StatusOK, StopPortServicesResponse{
			OK:        true,
			Stopped:   []string{},
			Message:   "Ports 80/443 are already free",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	for _, svc := range req.Services {
		unit := normalizeServiceName(svc)
		if unit == "" {
			continue
		}
		if stopUnit(ctx, s.sshRunner, cfg, unit) {
			stoppedUnits[unit] = true
			stopped = append(stopped, "service "+trimServiceSuffix(unit))
		} else {
			failed = append(failed, "service "+trimServiceSuffix(unit))
		}
	}

	for _, pid := range pids {
		if preferServiceStop {
			unit := resolveUnitForPID(ctx, s.sshRunner, cfg, pid)
			if unit != "" {
				if stoppedUnits[unit] {
					continue
				}
				if stopUnit(ctx, s.sshRunner, cfg, unit) {
					stoppedUnits[unit] = true
					stopped = append(stopped, fmt.Sprintf("service %s (pid %s)", trimServiceSuffix(unit), pid))
					continue
				}
			}
		}
		if stopPID(ctx, s.sshRunner, cfg, pid) {
			stopped = append(stopped, "pid "+pid)
		} else {
			failed = append(failed, "pid "+pid)
		}
	}

	if len(req.Services) == 0 && len(req.PIDs) == 0 && len(req.Ports) == 0 {
		commonServices := []string{"nginx", "apache2", "httpd", "caddy"}
		for _, svc := range commonServices {
			unit := normalizeServiceName(svc)
			if unit == "" || stoppedUnits[unit] {
				continue
			}
			if stopUnitIfActive(ctx, s.sshRunner, cfg, unit) {
				stoppedUnits[unit] = true
				stopped = append(stopped, "service "+trimServiceSuffix(unit))
			}
		}
	}

	msg := "Ports 80/443 should now be free"
	if len(req.Services) > 0 || len(req.PIDs) > 0 || len(req.Ports) > 0 {
		msg = "Requested stop actions completed"
	}
	if len(failed) > 0 {
		msg = "Some processes could not be stopped"
	}

	writeJSON(w, http.StatusOK, StopPortServicesResponse{
		OK:        len(failed) == 0,
		Stopped:   stopped,
		Failed:    failed,
		Message:   msg,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) collectPortStopPIDs(ctx context.Context, cfg SSHConfig, req StopPortServicesRequest) ([]string, bool, error) {
	if len(req.Ports) == 0 && len(req.PIDs) == 0 && len(req.Services) == 0 {
		ssRes, err := s.sshRunner.Run(ctx, cfg, `ss -ltnpH '( sport = :80 or sport = :443 )' 2>/dev/null || ss -ltnH '( sport = :80 or sport = :443 )'`)
		if err != nil {
			return nil, false, err
		}
		if strings.TrimSpace(ssRes.Stdout) == "" {
			return nil, false, nil
		}
		return extractPIDsFromSS(ssRes.Stdout), true, nil
	}

	seen := make(map[string]bool)
	var pids []string
	for _, pid := range req.PIDs {
		pidStr := strconv.Itoa(pid)
		if !seen[pidStr] {
			seen[pidStr] = true
			pids = append(pids, pidStr)
		}
	}

	for _, port := range req.Ports {
		cmd := fmt.Sprintf("ss -ltnpH 'sport = :%d' 2>/dev/null || ss -ltnH 'sport = :%d'", port, port)
		res, err := s.sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			return nil, true, err
		}
		for _, pid := range extractPIDsFromSS(res.Stdout) {
			if !seen[pid] {
				seen[pid] = true
				pids = append(pids, pid)
			}
		}
	}

	return pids, len(req.Ports) > 0 || len(req.PIDs) > 0 || len(req.Services) > 0, nil
}

func resolveUnitForPID(ctx context.Context, runner SSHRunner, cfg SSHConfig, pid string) string {
	cmd := fmt.Sprintf("systemctl status %s --no-pager 2>/dev/null | head -n 1", pid)
	res, err := runner.Run(ctx, cfg, cmd)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(res.Stdout)
	if line == "" {
		return ""
	}
	for _, field := range strings.Fields(line) {
		if strings.HasSuffix(field, ".service") {
			return strings.TrimPrefix(field, "●")
		}
	}
	return ""
}

func stopUnit(ctx context.Context, runner SSHRunner, cfg SSHConfig, unit string) bool {
	cmd := fmt.Sprintf("systemctl stop %s 2>/dev/null", unit)
	res, err := runner.Run(ctx, cfg, cmd)
	if err != nil {
		return false
	}
	return res.ExitCode == 0
}

func stopUnitIfActive(ctx context.Context, runner SSHRunner, cfg SSHConfig, unit string) bool {
	checkCmd := fmt.Sprintf("systemctl is-active %s 2>/dev/null", unit)
	checkRes, _ := runner.Run(ctx, cfg, checkCmd)
	if strings.TrimSpace(checkRes.Stdout) != "active" {
		return false
	}
	return stopUnit(ctx, runner, cfg, unit)
}

func stopPID(ctx context.Context, runner SSHRunner, cfg SSHConfig, pid string) bool {
	killCmd := fmt.Sprintf("kill %s 2>/dev/null && sleep 1 && ! kill -0 %s 2>/dev/null", pid, pid)
	_, err := runner.Run(ctx, cfg, killCmd)
	if err != nil {
		killCmd = fmt.Sprintf("kill -9 %s 2>/dev/null", pid)
		_, err = runner.Run(ctx, cfg, killCmd)
		if err != nil {
			return false
		}
	}
	return true
}

func normalizeServiceName(service string) string {
	service = strings.TrimSpace(service)
	if service == "" {
		return ""
	}
	if strings.HasSuffix(service, ".service") {
		return service
	}
	return service + ".service"
}

func trimServiceSuffix(service string) string {
	return strings.TrimSuffix(service, ".service")
}

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	var req DiskUsageRequest
	if !decodeRequestBody(w, r, &req) {
		return
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get disk usage stats
	dfRes, err := s.sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $2, $3, $4, $5}'`)
	if err != nil {
		writeJSON(w, http.StatusOK, DiskUsageResponse{
			OK:        false,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	parts := strings.Fields(dfRes.Stdout)
	var totalKB, freeKB int64
	var usedPct int
	if len(parts) >= 4 {
		totalKB, _ = strconv.ParseInt(parts[0], 10, 64)
		// parts[1] is used space (not needed for response)
		freeKB, _ = strconv.ParseInt(parts[2], 10, 64)
		pctStr := strings.TrimSuffix(parts[3], "%")
		usedPct, _ = strconv.Atoi(pctStr)
	}

	// Get largest directories
	duRes, _ := s.sshRunner.Run(ctx, cfg, `du -sk /var/* /home/* /root/* /opt/* /tmp/* 2>/dev/null | sort -rn | head -10`)
	var largestDirs []DiskUsageEntry
	for _, line := range strings.Split(duRes.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
			largestDirs = append(largestDirs, DiskUsageEntry{
				Path:  fields[1],
				Size:  formatBytes(sizeKB),
				Bytes: sizeKB * 1024,
			})
		}
	}

	writeJSON(w, http.StatusOK, DiskUsageResponse{
		OK:          true,
		FreeSpace:   formatBytes(freeKB),
		FreeBytes:   freeKB * 1024,
		TotalSpace:  formatBytes(totalKB),
		TotalBytes:  totalKB * 1024,
		UsedPercent: usedPct,
		LargestDirs: largestDirs,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleDiskCleanup(w http.ResponseWriter, r *http.Request) {
	var req DiskCleanupRequest
	if !decodeRequestBody(w, r, &req) {
		return
	}

	if len(req.Actions) == 0 {
		req.Actions = []string{"apt_clean", "journal_vacuum"}
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Get initial free space
	beforeRes, _ := s.sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
	beforeKB, _ := strconv.ParseInt(strings.TrimSpace(beforeRes.Stdout), 10, 64)

	var actionsRun, actionsFailed []string

	actionCommands := map[string]string{
		"apt_clean":      "apt-get clean && apt-get autoremove -y",
		"journal_vacuum": "journalctl --vacuum-size=100M",
		"docker_prune":   "docker system prune -af 2>/dev/null || true",
		"tmp_clean":      "find /tmp -type f -atime +7 -delete 2>/dev/null || true",
	}

	for _, action := range req.Actions {
		cmd, ok := actionCommands[action]
		if !ok {
			continue
		}
		_, err := s.sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			actionsFailed = append(actionsFailed, action)
		} else {
			actionsRun = append(actionsRun, action)
		}
	}

	// Get final free space
	afterRes, _ := s.sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
	afterKB, _ := strconv.ParseInt(strings.TrimSpace(afterRes.Stdout), 10, 64)

	freedKB := afterKB - beforeKB
	if freedKB < 0 {
		freedKB = 0
	}

	msg := fmt.Sprintf("Freed %s of disk space", formatBytes(freedKB))
	if freedKB == 0 {
		msg = "No significant space was freed"
	}

	writeJSON(w, http.StatusOK, DiskCleanupResponse{
		OK:            len(actionsFailed) == 0,
		SpaceFreed:    formatBytes(freedKB),
		SpaceFreedKB:  freedKB,
		Message:       msg,
		ActionsRun:    actionsRun,
		ActionsFailed: actionsFailed,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	})
}

// StopScenarioProcessesRequest is the request body for stopping stale scenario processes
type StopScenarioProcessesRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	KeyPath    string `json:"key_path"`
	Workdir    string `json:"workdir"`
	ScenarioID string `json:"scenario_id,omitempty"` // If empty, stops all vrooli processes
}

// StopScenarioProcessesResponse is the response from stopping scenario processes
type StopScenarioProcessesResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"` // "stop_scenario" or "stop_all"
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleStopScenarioProcesses(w http.ResponseWriter, r *http.Request) {
	var req StopScenarioProcessesRequest
	if !decodeRequestBody(w, r, &req) {
		return
	}

	if req.Workdir == "" {
		req.Workdir = DefaultVPSWorkdir
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Verify SSH connectivity first
	testResult, testErr := s.sshRunner.Run(ctx, cfg, "echo ok")
	if testErr != nil {
		writeStopScenarioResponse(w, false, "test_ssh",
			fmt.Sprintf("SSH connection test failed: %v (stdout: %s, stderr: %s)", testErr, testResult.Stdout, testResult.Stderr),
			testResult.Stderr)
		return
	}

	// Verify workdir exists (common check for both branches)
	if !s.workdirExists(ctx, cfg, req.Workdir) {
		writeStopScenarioResponse(w, false, "check_workdir",
			fmt.Sprintf("Workdir %s does not exist on VPS", req.Workdir), "")
		return
	}

	// Branch: Stop specific scenario
	if req.ScenarioID != "" {
		result := StopExistingScenario(ctx, s.sshRunner, cfg, req.Workdir, req.ScenarioID, nil)
		writeStopScenarioResponse(w, result.OK, "stop_scenario", result.Message,
			fmt.Sprintf("Stopped scenario %s: %s", req.ScenarioID, result.Message))
		return
	}

	// Branch: Stop all vrooli processes
	outputs := s.stopAllVrooliProcesses(ctx, cfg, req.Workdir)
	writeStopScenarioResponse(w, true, "stop_all",
		"Stopped all vrooli processes and killed orphans",
		strings.Join(outputs, "\n"))
}

// writeStopScenarioResponse is a helper to reduce response construction boilerplate.
func writeStopScenarioResponse(w http.ResponseWriter, ok bool, action, message, output string) {
	writeJSON(w, http.StatusOK, StopScenarioProcessesResponse{
		OK:        ok,
		Action:    action,
		Message:   message,
		Output:    output,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// workdirExists checks if the specified directory exists on the remote VPS.
func (s *Server) workdirExists(ctx context.Context, cfg SSHConfig, workdir string) bool {
	result, _ := s.sshRunner.Run(ctx, cfg, fmt.Sprintf("test -d %s && echo exists || echo missing", shellQuoteSingle(workdir)))
	return strings.TrimSpace(result.Stdout) == "exists"
}

// stopAllVrooliProcesses attempts to stop all vrooli processes using CLI (if available) and pkill.
func (s *Server) stopAllVrooliProcesses(ctx context.Context, cfg SSHConfig, workdir string) []string {
	var outputs []string

	// Try vrooli stop first (if CLI available)
	checkCliResult, _ := s.sshRunner.Run(ctx, cfg, vrooliCommand(workdir, "which vrooli || echo notfound"))
	if !strings.Contains(checkCliResult.Stdout, "notfound") {
		vrooliResult, _ := s.sshRunner.Run(ctx, cfg, vrooliCommand(workdir, "vrooli stop"))
		if vrooliResult.Stdout != "" {
			outputs = append(outputs, vrooliResult.Stdout)
		}
	}

	// Kill orphaned processes that may not be managed by vrooli lifecycle
	killOrphansCmd := "pkill -f 'pm2' 2>/dev/null; pkill -f '-api$' 2>/dev/null; pkill -f 'scenario.*api' 2>/dev/null; true"
	if _, err := s.sshRunner.Run(ctx, cfg, killOrphansCmd); err != nil {
		outputs = append(outputs, "Failed to kill orphaned processes: "+err.Error())
	} else {
		outputs = append(outputs, "Killed orphaned processes")
	}

	return outputs
}

// extractPIDsFromSS parses PIDs from ss -ltnp output
// Example: LISTEN 0 4096 *:80 *:* users:(("nginx",pid=1234,fd=6))
func extractPIDsFromSS(output string) []string {
	var pids []string
	seen := make(map[string]bool)

	pidRegex := regexp.MustCompile(`pid=(\d+)`)

	for _, line := range strings.Split(output, "\n") {
		matches := pidRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pid := match[1]
				if !seen[pid] {
					seen[pid] = true
					pids = append(pids, pid)
				}
			}
		}
	}

	return pids
}
