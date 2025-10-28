package queue

import (
	"context"
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

// TerminationOptions controls agent termination behavior
type TerminationOptions struct {
	// ForceRemove enables aggressive retry with exponential backoff
	ForceRemove bool
	// VerifyRemoval ensures agent is fully removed before returning
	VerifyRemoval bool
	// Context provides context for logging (e.g., "timeout", "user request")
	Context string
}

// AGENT TERMINATION ARCHITECTURE:
//
// This module provides a layered approach to agent cleanup:
//
// 1. terminateAgent() - Core termination logic (CLI stop + process kill + verification)
//    Use this for standard termination cases
//
// 2. terminateAgentWithOptions() - Configurable termination with retry options
//    Use this when you need fine-grained control (e.g., force removal, extra verification)
//
// 3. forceRemoveAgentWithRetry() - Aggressive cleanup with exponential backoff
//    Use this only when standard termination has failed
//
// 4. ensureAgentRemoved() - Verification helper that triggers forced removal if needed
//    Use this after termination to guarantee cleanup
//
// Best practices:
// - For normal cleanup: Use terminateAgent()
// - For critical cleanup (e.g., after finalization failure): Use terminateAgentWithOptions() with ForceRemove
// - For verification after cleanup: Use ensureAgentRemoved()

// stopClaudeAgent attempts to stop a Claude Code agent with timeout
// Returns error if stop command fails (excluding fallback to PID termination)
func (qp *Processor) stopClaudeAgent(agentIdentifier string, pid int) error {
	if agentIdentifier != "" {
		ctx, cancel := context.WithTimeout(context.Background(), AgentStopTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, ClaudeCodeResourceCommand, "agents", "stop", agentIdentifier)
		if qp.vrooliRoot != "" {
			cmd.Dir = qp.vrooliRoot
		}
		if err := cmd.Run(); err != nil {
			stopErr := err
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("Timeout stopping claude-code agent %s (exceeded %v)", agentIdentifier, AgentStopTimeout)
				stopErr = fmt.Errorf("timeout stopping agent %s: %w", agentIdentifier, ctx.Err())
			} else {
				log.Printf("Failed to stop claude-code agent %s via CLI: %v", agentIdentifier, err)
			}
			// Continue to fallback PID termination, but track the error
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
			return stopErr
		}
		log.Printf("Successfully stopped claude-code agent %s", agentIdentifier)
		return nil
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
		return nil
	}

	return fmt.Errorf("no agent identifier or PID provided for stop")
}

// cleanupClaudeAgentRegistryOrWarn runs cleanup and logs critical errors if it fails
// This is a convenience wrapper around cleanupClaudeAgentRegistry for common use cases
// where we want consistent error logging without needing to handle the error explicitly
func (qp *Processor) cleanupClaudeAgentRegistryOrWarn(context string) {
	if err := qp.cleanupClaudeAgentRegistry(); err != nil {
		systemlog.Errorf("CRITICAL: Agent registry cleanup failed during %s: %v - tasks may appear stuck", context, err)
		log.Printf("ERROR: Agent cleanup failed during %s: %v", context, err)
	}
}

// cleanupClaudeAgentRegistry runs the agent cleanup command with timeout
// CRITICAL: This must succeed to prevent zombie agents appearing "stuck active" in UI
// Returns error if cleanup fails (even after retry)
func (qp *Processor) cleanupClaudeAgentRegistry() error {
	ctx, cancel := context.WithTimeout(context.Background(), AgentRegistryTimeout)
	defer cancel()

	cleanupCmd := exec.CommandContext(ctx, ClaudeCodeResourceCommand, "agents", "cleanup")
	if qp.vrooliRoot != "" {
		cleanupCmd.Dir = qp.vrooliRoot
	}

	if err := cleanupCmd.Run(); err != nil {
		firstErr := err
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("ERROR: Agent cleanup timed out after %v", AgentRegistryTimeout)
			systemlog.Errorf("Agent registry cleanup timed out - may cause tasks to appear stuck active")
			firstErr = fmt.Errorf("cleanup timeout after %v: %w", AgentRegistryTimeout, ctx.Err())
		} else {
			log.Printf("ERROR: Failed to cleanup claude-code agents: %v", err)
			systemlog.Errorf("Agent registry cleanup failed: %v - may cause tasks to appear stuck active", err)
		}

		// Retry cleanup once after a delay - critical for preventing stuck tasks
		time.Sleep(AgentCleanupRetryDelay)
		retryCtx, retryCancel := context.WithTimeout(context.Background(), AgentRegistryTimeout)
		defer retryCancel()

		retryCmd := exec.CommandContext(retryCtx, ClaudeCodeResourceCommand, "agents", "cleanup")
		if qp.vrooliRoot != "" {
			retryCmd.Dir = qp.vrooliRoot
		}
		if retryErr := retryCmd.Run(); retryErr != nil {
			if retryCtx.Err() == context.DeadlineExceeded {
				log.Printf("ERROR: Agent cleanup retry also timed out after %v", AgentRegistryTimeout)
				systemlog.Errorf("Agent cleanup retry timed out - zombie agents may persist")
				return fmt.Errorf("cleanup retry timeout after %v (first error: %v)", AgentRegistryTimeout, firstErr)
			}
			log.Printf("ERROR: Agent cleanup retry also failed: %v", retryErr)
			systemlog.Errorf("Agent cleanup retry failed: %v - zombie agents may persist", retryErr)
			return fmt.Errorf("cleanup retry failed: %w (first error: %v)", retryErr, firstErr)
		}
		log.Printf("✅ Agent cleanup succeeded on retry")
		return nil // Retry succeeded
	}
	return nil // First attempt succeeded
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

// terminateAgentWithOptions terminates an agent with configurable behavior
// This is the most flexible termination function - use this when you need control over behavior
func (qp *Processor) terminateAgentWithOptions(taskID, agentTag string, pid int, opts TerminationOptions) error {
	// Resolve agent tag if not provided
	if agentTag == "" {
		agentTag = makeAgentTag(taskID)
	}

	context := opts.Context
	if context == "" {
		context = "standard termination"
	}

	log.Printf("Terminating agent %s (pid %d) for task %s [%s]", agentTag, pid, taskID, context)

	// Use standard termination sequence
	err := qp.terminateAgent(taskID, agentTag, pid)

	// If termination failed and force removal is requested, retry aggressively
	if err != nil && opts.ForceRemove {
		log.Printf("Standard termination failed for %s, attempting forced removal", agentTag)
		if retryErr := qp.forceRemoveAgentWithRetry(agentTag, pid); retryErr != nil {
			return fmt.Errorf("forced removal failed after standard termination: %w", retryErr)
		}
		return nil // Forced removal succeeded
	}

	// If verification is requested and standard termination reports success, double-check
	if err == nil && opts.VerifyRemoval {
		if qp.agentExists(agentTag) {
			return fmt.Errorf("verification failed: agent %s still exists after successful termination", agentTag)
		}
	}

	return err
}

// terminateAgent is the canonical agent termination function that all other termination
// functions should delegate to for consistent behavior across the codebase
func (qp *Processor) terminateAgent(taskID, agentTag string, pid int) error {
	// Resolve agent tag if not provided
	if agentTag == "" {
		agentTag = makeAgentTag(taskID)
	}

	log.Printf("Terminating agent %s (pid %d) for task %s", agentTag, pid, taskID)

	// Step 1: Stop via CLI
	if err := qp.stopClaudeAgent(agentTag, pid); err != nil {
		log.Printf("Warning: stopClaudeAgent failed for %s: %v", agentTag, err)
	}
	qp.cleanupClaudeAgentRegistryOrWarn("terminateAgent")

	// Step 2: Verify agent was removed from registry
	if qp.agentExists(agentTag) {
		log.Printf("Warning: Agent %s still active in registry after CLI stop", agentTag)
		systemlog.Warnf("Agent %s persists in registry after stop attempt", agentTag)
	}

	// Step 3: Kill process if still alive
	if pid > 0 && qp.isProcessAlive(pid) {
		if err := qp.killProcessGracefully(pid); err != nil {
			log.Printf("Warning: graceful kill failed for pid %d, using force kill", pid)
			// Fallback to process group kill
			if err := KillProcessGroup(pid); err != nil {
				log.Printf("Warning: failed to kill process group for pid %d: %v", pid, err)
			}
		}
	}

	// Step 4: Wait for complete shutdown
	qp.waitForAgentShutdown(agentTag, pid)

	// Step 5: Final verification
	if qp.agentExists(agentTag) {
		err := fmt.Errorf("agent %s still registered after full termination sequence", agentTag)
		systemlog.Errorf("Failed to remove agent %s from registry: %v", agentTag, err)
		return err
	}

	log.Printf("Successfully terminated agent %s for task %s", agentTag, taskID)
	return nil
}

// ensureAgentRemoved is the canonical function for verifying and forcing agent removal
// This consolidates all agent cleanup verification logic into one place
// Returns error if agent still exists after all cleanup attempts
func (qp *Processor) ensureAgentRemoved(agentTag string, pid int, context string) error {
	// Check if agent exists before attempting removal
	if !qp.agentExists(agentTag) {
		return nil // Agent already removed, success
	}

	log.Printf("WARNING: Agent %s still exists after %s, initiating forced removal with retries", agentTag, context)
	systemlog.Warnf("Zombie agent %s detected after %s, attempting forced removal with retries", agentTag, context)

	// Attempt forced removal with retry loop
	if err := qp.forceRemoveAgentWithRetry(agentTag, pid); err != nil {
		systemlog.Errorf("CRITICAL: Failed to remove zombie agent %s after all retry attempts: %v", agentTag, err)
		log.Printf("ERROR: Agent %s remains stuck in registry after %s - this will cause task to appear active forever", agentTag, context)
		return fmt.Errorf("agent %s remains in registry after %s: %w", agentTag, context, err)
	}

	log.Printf("Successfully removed zombie agent %s after %s", agentTag, context)
	return nil
}

// forceRemoveAgentWithRetry attempts to remove a zombie agent with exponential backoff retries
// This is the nuclear option for when cleanupClaudeAgentRegistry fails to remove an agent
func (qp *Processor) forceRemoveAgentWithRetry(agentTag string, pid int) error {
	for attempt := 1; attempt <= MaxAgentCleanupRetries; attempt++ {
		// Strategy 1: Try stopClaudeAgent first (uses CLI)
		if err := qp.stopClaudeAgent(agentTag, pid); err != nil {
			log.Printf("Attempt %d/%d: stopClaudeAgent failed for %s: %v", attempt, MaxAgentCleanupRetries, agentTag, err)
		}

		// Strategy 2: Run cleanup registry
		if err := qp.cleanupClaudeAgentRegistry(); err != nil {
			log.Printf("Attempt %d/%d: cleanupClaudeAgentRegistry failed: %v", attempt, MaxAgentCleanupRetries, err)
		}

		// Verify removal
		if !qp.agentExists(agentTag) {
			if attempt > 1 {
				log.Printf("Agent %s successfully removed on attempt %d/%d", agentTag, attempt, MaxAgentCleanupRetries)
				systemlog.Infof("Zombie agent %s removed after %d retry attempts", agentTag, attempt)
			}
			return nil
		}

		// If still exists and not the last attempt, wait with exponential backoff
		if attempt < MaxAgentCleanupRetries {
			delay := time.Duration(attempt) * AgentCleanupBackoffBase
			log.Printf("Agent %s still exists after attempt %d/%d, retrying in %v", agentTag, attempt, MaxAgentCleanupRetries, delay)
			time.Sleep(delay)

			// Strategy 3: On second attempt, try killing any orphaned processes
			if attempt == 2 {
				pids := qp.agentProcessPIDs(agentTag)
				for _, orphanPID := range pids {
					log.Printf("Killing orphaned process %d for agent %s", orphanPID, agentTag)
					qp.terminateProcessByPID(orphanPID)
				}
			}
		}
	}

	// All attempts failed
	return fmt.Errorf("agent %s still registered after %d attempts with multiple cleanup strategies", agentTag, MaxAgentCleanupRetries)
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

// queryAgentRegistry queries the resource-claude-code agent registry with timeout
func (qp *Processor) queryAgentRegistry() (map[string]struct{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), AgentRegistryTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ClaudeCodeResourceCommand, "agents", "list", "--format", "json")
	if qp.vrooliRoot != "" {
		cmd.Dir = qp.vrooliRoot
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("agent registry query timed out after %v", AgentRegistryTimeout)
		}
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
		if isValidAgentTag(agent.Tag) {
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
				if isValidAgentTag(agentTag) {
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

	if err := qp.stopClaudeAgent(agentTag, 0); err != nil {
		log.Printf("Warning: failed to stop agent %s during ensureAgentInactive: %v", agentTag, err)
	}
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
		if err := qp.stopClaudeAgent(agentTag, 0); err != nil {
			log.Printf("Warning: failed to stop orphaned agent %s: %v", agentTag, err)
		}
	}

	qp.cleanupClaudeAgentRegistryOrWarn("orphaned process cleanup")
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
