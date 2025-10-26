package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// AgentDetectionMethod specifies how to detect active Claude agents
type AgentDetectionMethod int

const (
	// DetectRegistry uses the resource-claude-code agent registry
	DetectRegistry AgentDetectionMethod = iota
	// DetectProcessList scans running processes
	DetectProcessList
	// DetectBoth uses both registry and process list
	DetectBoth
)

// stopClaudeAgent attempts to stop a Claude Code agent
func (qp *Processor) stopClaudeAgent(agentIdentifier string, pid int) {
	if agentIdentifier != "" {
		cmd := exec.Command(ClaudeCodeResourceCommand, "agents", "stop", agentIdentifier)
		if qp.vrooliRoot != "" {
			cmd.Dir = qp.vrooliRoot
		}
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to stop claude-code agent %s via CLI: %v", agentIdentifier, err)
		} else {
			log.Printf("Successfully stopped claude-code agent %s", agentIdentifier)
			return
		}
	}

	if pid == 0 && agentIdentifier != "" {
		if pids := qp.agentProcessPIDs(agentIdentifier); len(pids) > 0 {
			pid = pids[0]
			for _, extra := range pids[1:] {
				qp.terminateProcessByPID(extra)
			}
		}
	}

	if pid > 0 {
		qp.terminateProcessByPID(pid)
	}
}

// cleanupClaudeAgentRegistry runs the agent cleanup command
func (qp *Processor) cleanupClaudeAgentRegistry() {
	cleanupCmd := exec.Command(ClaudeCodeResourceCommand, "agents", "cleanup")
	if err := cleanupCmd.Run(); err != nil {
		log.Printf("Warning: Failed to cleanup claude-code agents: %v", err)
	}
}

// agentProcessPIDs finds all process IDs for a given agent tag
func (qp *Processor) agentProcessPIDs(agentTag string) []int {
	pattern := fmt.Sprintf("%s run --tag %s", ClaudeCodeResourceCommand, agentTag)
	cmd := exec.Command("pgrep", "-f", pattern)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(output))
	var pids []int
	for _, field := range fields {
		if pid, err := strconv.Atoi(strings.TrimSpace(field)); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// waitForAgentShutdown waits for an agent to fully terminate
func (qp *Processor) waitForAgentShutdown(agentTag string, primaryPID int) {
	deadline := time.Now().Add(AgentShutdownTimeout)

	for {
		agentActive := qp.agentExists(agentTag)
		primaryAlive := primaryPID > 0 && qp.isProcessAlive(primaryPID)

		if !agentActive && !primaryAlive {
			return
		}

		pids := qp.agentProcessPIDs(agentTag)
		for _, pid := range pids {
			if pid == primaryPID {
				continue
			}
			if err := KillProcessGroup(pid); err != nil {
				log.Printf("Warning: failed to terminate agent process group for %s (pid %d): %v", agentTag, pid, err)
			}
			// Best-effort SIGKILL - ignore errors as process may already be dead
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				log.Printf("Warning: failed to SIGKILL process %d: %v", pid, err)
			}
		}

		if primaryAlive {
			if err := KillProcessGroup(primaryPID); err != nil {
				log.Printf("Warning: failed to terminate primary agent process group for %s (pid %d): %v", agentTag, primaryPID, err)
			}
			// Best-effort SIGKILL - ignore errors as process may already be dead
			if err := syscall.Kill(primaryPID, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				log.Printf("Warning: failed to SIGKILL primary process %d: %v", primaryPID, err)
			}
		}

		if time.Now().After(deadline) {
			log.Printf("Warning: agent %s still appears active after forced shutdown", agentTag)
			return
		}

		time.Sleep(AgentShutdownPollInterval)
	}
}

// agentExists checks if an agent is registered in the external registry
func (qp *Processor) agentExists(agentTag string) bool {
	tags := qp.getExternalActiveTaskIDs()
	_, exists := tags[parseAgentTag(agentTag)]
	return exists
}

// terminateProcessByPID terminates a process by PID with graceful fallback to force kill
func (qp *Processor) terminateProcessByPID(pid int) {
	if err := qp.killProcessGracefully(pid); err != nil {
		log.Printf("Warning: failed to terminate PID %d: %v", pid, err)
	}
}

// getExternalActiveTaskIDs returns task IDs with active Claude agents
func (qp *Processor) getExternalActiveTaskIDs() map[string]struct{} {
	// Use both detection methods for comprehensive agent discovery
	tags, err := qp.detectActiveAgentTags(DetectBoth)
	if err != nil {
		log.Printf("Warning: failed to detect active agents: %v", err)
		return make(map[string]struct{})
	}

	active := make(map[string]struct{}, len(tags))
	for agentTag := range tags {
		taskID := parseAgentTag(agentTag)
		active[taskID] = struct{}{}
	}
	return active
}

// detectActiveAgentTags discovers active Claude agents using the specified method
func (qp *Processor) detectActiveAgentTags(method AgentDetectionMethod) (map[string]struct{}, error) {
	var registryTags map[string]struct{}
	var procListTags map[string]struct{}
	var registryErr error

	if method == DetectRegistry || method == DetectBoth {
		registryTags, registryErr = qp.queryAgentRegistry()
	}

	if method == DetectProcessList || method == DetectBoth {
		procListTags = qp.scanProcessListForAgents()
	}

	if method == DetectRegistry {
		return registryTags, registryErr
	}

	if method == DetectProcessList {
		return procListTags, nil
	}

	merged := make(map[string]struct{})
	for tag := range registryTags {
		merged[tag] = struct{}{}
	}
	for tag := range procListTags {
		merged[tag] = struct{}{}
	}
	return merged, nil
}

// queryAgentRegistry queries the resource-claude-code agent registry
func (qp *Processor) queryAgentRegistry() (map[string]struct{}, error) {
	cmd := exec.Command(ClaudeCodeResourceCommand, "agents", "list", "--format", "json")
	if qp.vrooliRoot != "" {
		cmd.Dir = qp.vrooliRoot
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to query agent registry: %w (output: %s)", err, string(output))
	}

	var payload struct {
		Agents []struct {
			Tag string `json:"tag"`
		} `json:"agents"`
	}

	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse agent list JSON: %w", err)
	}

	agentTags := make(map[string]struct{}, len(payload.Agents))
	for _, agent := range payload.Agents {
		if strings.HasPrefix(agent.Tag, AgentTagPrefix) {
			agentTags[agent.Tag] = struct{}{}
		}
	}

	return agentTags, nil
}

// scanProcessListForAgents scans running processes for Claude agents
func (qp *Processor) scanProcessListForAgents() map[string]struct{} {
	pattern := ClaudeCodeResourceCommand + " run --tag " + AgentTagPrefix
	cmd := exec.Command("pgrep", "-f", pattern)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return make(map[string]struct{})
	}

	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		return make(map[string]struct{})
	}

	agentTags := make(map[string]struct{})
	for _, pidStr := range pids {
		cmdlineFile := fmt.Sprintf("/proc/%s/cmdline", pidStr)
		data, err := os.ReadFile(cmdlineFile)
		if err != nil {
			continue
		}

		cmdline := string(data)
		if !strings.Contains(cmdline, "--tag") {
			continue
		}
		parts := strings.Split(cmdline, "\x00")
		for i, part := range parts {
			if part == "--tag" && i+1 < len(parts) {
				agentTag := parts[i+1]
				if strings.HasPrefix(agentTag, AgentTagPrefix) {
					agentTags[agentTag] = struct{}{}
				}
				break
			}
		}
	}

	return agentTags
}

// ensureAgentInactive ensures no agent is running for a given tag
func (qp *Processor) ensureAgentInactive(agentTag string) error {
	if agentTag == "" {
		return nil
	}

	// Use both detection methods for comprehensive agent discovery
	tags, err := qp.detectActiveAgentTags(DetectBoth)
	if err != nil {
		return err
	}

	if _, exists := tags[agentTag]; !exists {
		return nil
	}

	qp.stopClaudeAgent(agentTag, 0)
	time.Sleep(AgentCleanupRetryDelay)

	tags, err = qp.detectActiveAgentTags(DetectBoth)
	if err != nil {
		return err
	}

	if _, exists := tags[agentTag]; exists {
		return fmt.Errorf("agent %s still active after stop attempt", agentTag)
	}

	return nil
}

// cleanupOrphanedProcesses terminates any orphaned Claude agents
func (qp *Processor) cleanupOrphanedProcesses() {
	externalActive := qp.getExternalActiveTaskIDs()
	internalRunning := qp.getInternalRunningTaskIDs()

	for taskID := range externalActive {
		if _, tracked := internalRunning[taskID]; tracked {
			continue
		}

		agentTag := makeAgentTag(taskID)
		log.Printf("Detected orphaned agent %s, terminating...", agentTag)
		systemlog.Warnf("Detected orphaned agent %s, terminating", agentTag)
		qp.stopClaudeAgent(agentTag, 0)
	}

	qp.cleanupClaudeAgentRegistry()
}

// isProcessAlive checks if a process is running
func (qp *Processor) isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// killProcessGracefully terminates a process with SIGTERM, then SIGKILL if needed
// This is the canonical process termination function - all other termination
// functions should delegate to this one for consistent behavior
func (qp *Processor) killProcessGracefully(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}

	// Try graceful termination first
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		log.Printf("Failed to send SIGTERM to PID %d: %v", pid, err)
	}

	time.Sleep(ProcessTermRetryDelay)
	if qp.isProcessAlive(pid) {
		// Process didn't exit gracefully, force kill
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return fmt.Errorf("failed to SIGKILL process %d: %w", pid, err)
		}
	}

	// Verify termination
	time.Sleep(ProcessCleanupDelay)
	if qp.isProcessAlive(pid) {
		return fmt.Errorf("process %d still alive after kill attempts", pid)
	}

	return nil
}
