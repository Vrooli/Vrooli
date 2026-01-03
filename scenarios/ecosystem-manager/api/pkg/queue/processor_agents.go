package queue

// ==============================================================================
// AGENT STATUS AND TRACKING
// ==============================================================================
//
// This file provides agent status queries for ecosystem-manager.
// With agent-manager integration, all execution and cleanup is handled by
// agent-manager via agentSvc. This file contains only:
//
//   1. Internal execution tracking (getInternalRunningTaskIDs)
//   2. External agent detection (getExternalActiveTaskIDs) - returns empty,
//      agent-manager is the source of truth for external state
//   3. Startup cleanup (cleanupOrphanedProcesses) - simplified to reset
//      internal tracking; agent-manager handles actual agent lifecycle
//
// All cleanup functions have been removed - agent-manager handles:
//   - stopClaudeAgent() -> agentSvc.StopRun()
//   - terminateAgent() -> agentSvc.StopRun()
//   - ensureAgentRemoved() -> agent-manager verifies cleanup
//   - forceRemoveAgentWithRetry() -> agent-manager handles retries
//
// ==============================================================================

import (
	"log"

	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// getExternalActiveTaskIDs returns task IDs with active agents.
// With agent-manager integration, we rely on internal tracking only.
// Agent-manager is the source of truth for external agent state.
func (qp *Processor) getExternalActiveTaskIDs() map[string]struct{} {
	// Return empty - agent-manager handles external state.
	// Internal tracking via getInternalRunningTaskIDs is sufficient.
	return make(map[string]struct{})
}

// getInternalRunningTaskIDs returns task IDs currently tracked as running internally.
func (qp *Processor) getInternalRunningTaskIDs() map[string]struct{} {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	result := make(map[string]struct{}, len(qp.executions))
	for taskID := range qp.executions {
		result[taskID] = struct{}{}
	}
	return result
}

// cleanupOrphanedProcesses clears stale internal tracking at startup.
// With agent-manager integration, actual agent lifecycle is handled by agent-manager.
// This function just ensures our internal state is clean on startup.
func (qp *Processor) cleanupOrphanedProcesses() {
	qp.executionsMu.Lock()
	orphanCount := len(qp.executions)
	qp.executions = make(map[string]*taskExecution)
	qp.executionsMu.Unlock()

	if orphanCount > 0 {
		log.Printf("Cleared %d stale execution records from previous run", orphanCount)
		systemlog.Infof("Startup: cleared %d stale execution tracking records", orphanCount)
	}
}
