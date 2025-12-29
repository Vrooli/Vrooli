package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StopPortServicesRequest is the request body for stopping services on ports 80/443
type StopPortServicesRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
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
	OK            bool   `json:"ok"`
	SpaceFreed    string `json:"space_freed"`
	SpaceFreedKB  int64  `json:"space_freed_kb"`
	Message       string `json:"message"`
	ActionsRun    []string `json:"actions_run"`
	ActionsFailed []string `json:"actions_failed,omitempty"`
	Timestamp     string `json:"timestamp"`
}

func (s *Server) handleStopPortServices(w http.ResponseWriter, r *http.Request) {
	var req StopPortServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Message: "Invalid request body",
		})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}

	cfg := SSHConfig{
		Host:    req.Host,
		Port:    req.Port,
		User:    req.User,
		KeyPath: req.KeyPath,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	sshRunner := ExecSSHRunner{}

	// First, get list of processes on ports 80/443
	ssRes, err := sshRunner.Run(ctx, cfg, `ss -ltnpH '( sport = :80 or sport = :443 )' 2>/dev/null || ss -ltnH '( sport = :80 or sport = :443 )'`)
	if err != nil {
		writeJSON(w, http.StatusOK, StopPortServicesResponse{
			OK:        false,
			Message:   "Failed to check ports: " + err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if strings.TrimSpace(ssRes.Stdout) == "" {
		writeJSON(w, http.StatusOK, StopPortServicesResponse{
			OK:        true,
			Stopped:   []string{},
			Message:   "Ports 80/443 are already free",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Parse PIDs from ss output
	pids := extractPIDsFromSS(ssRes.Stdout)

	var stopped, failed []string

	// Try to stop each process
	for _, pid := range pids {
		killCmd := fmt.Sprintf("kill %s 2>/dev/null && sleep 1 && ! kill -0 %s 2>/dev/null", pid, pid)
		_, err := sshRunner.Run(ctx, cfg, killCmd)
		if err != nil {
			// Try SIGKILL
			killCmd = fmt.Sprintf("kill -9 %s 2>/dev/null", pid)
			_, err = sshRunner.Run(ctx, cfg, killCmd)
			if err != nil {
				failed = append(failed, "pid "+pid)
				continue
			}
		}
		stopped = append(stopped, "pid "+pid)
	}

	// Also try to stop common services
	commonServices := []string{"nginx", "apache2", "httpd", "caddy"}
	for _, svc := range commonServices {
		checkCmd := fmt.Sprintf("systemctl is-active %s 2>/dev/null", svc)
		checkRes, _ := sshRunner.Run(ctx, cfg, checkCmd)
		if strings.TrimSpace(checkRes.Stdout) == "active" {
			stopCmd := fmt.Sprintf("systemctl stop %s 2>/dev/null", svc)
			_, err := sshRunner.Run(ctx, cfg, stopCmd)
			if err != nil {
				failed = append(failed, svc)
			} else {
				stopped = append(stopped, svc)
			}
		}
	}

	msg := "Ports 80/443 should now be free"
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

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	var req DiskUsageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Message: "Invalid request body",
		})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}

	cfg := SSHConfig{
		Host:    req.Host,
		Port:    req.Port,
		User:    req.User,
		KeyPath: req.KeyPath,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	sshRunner := ExecSSHRunner{}

	// Get disk usage stats
	dfRes, err := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $2, $3, $4, $5}'`)
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
	duRes, _ := sshRunner.Run(ctx, cfg, `du -sk /var/* /home/* /root/* /opt/* /tmp/* 2>/dev/null | sort -rn | head -10`)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Message: "Invalid request body",
		})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}
	if len(req.Actions) == 0 {
		req.Actions = []string{"apt_clean", "journal_vacuum"}
	}

	cfg := SSHConfig{
		Host:    req.Host,
		Port:    req.Port,
		User:    req.User,
		KeyPath: req.KeyPath,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	sshRunner := ExecSSHRunner{}

	// Get initial free space
	beforeRes, _ := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
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
		_, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			actionsFailed = append(actionsFailed, action)
		} else {
			actionsRun = append(actionsRun, action)
		}
	}

	// Get final free space
	afterRes, _ := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
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
