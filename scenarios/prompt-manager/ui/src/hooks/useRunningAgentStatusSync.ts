/**
 * useRunningAgentStatusSync - Syncs running agent data into the accessory store
 * (for 3D world StatusIcons) and the runningAgentsStore (for MemberDetailPanel).
 *
 * Should be called once in SkillManagerLayout (always mounted).
 * Returns the same shape as useRunningAgents so the popover can consume it.
 */

import { useEffect, useRef } from 'react'
import { useRunningAgents, type UseRunningAgentsResult } from './useRunningAgents'
import { useAccessoryStore } from '@/stores/accessoryStore'
import { useRunningAgentsStore } from '@/stores/runningAgentsStore'

export function useRunningAgentStatusSync(): UseRunningAgentsResult {
  const result = useRunningAgents()
  const { runningAgents } = result

  // Track which agent IDs we've set status for, so we can clear them when they stop
  const managedIdsRef = useRef<Set<string>>(new Set())

  useEffect(() => {
    const { setAgentStatus, clearAgentStatus, agentAccessories } = useAccessoryStore.getState()

    // Update runningAgentsStore
    useRunningAgentsStore.getState().setAgents(runningAgents)

    // Build set of currently running agent IDs
    const currentIds = new Set<string>()
    for (const agent of runningAgents) {
      currentIds.add(agent.agentId)
      setAgentStatus(agent.agentId, {
        type: 'thinking',
        message: `Running heartbeat (${agent.duration})`,
        source: 'heartbeat',
      })
    }

    // Clear status for agents that stopped — only if source was 'heartbeat'
    for (const agentId of managedIdsRef.current) {
      if (!currentIds.has(agentId)) {
        const existing = agentAccessories[agentId]?.status
        if (!existing || existing.source === 'heartbeat') {
          clearAgentStatus(agentId)
        }
      }
    }

    managedIdsRef.current = currentIds
  }, [runningAgents])

  // Cleanup on unmount: clear all managed statuses
  useEffect(() => {
    return () => {
      const { clearAgentStatus, agentAccessories } = useAccessoryStore.getState()
      for (const agentId of managedIdsRef.current) {
        const existing = agentAccessories[agentId]?.status
        if (!existing || existing.source === 'heartbeat') {
          clearAgentStatus(agentId)
        }
      }
      useRunningAgentsStore.getState().setAgents([])
    }
  }, [])

  return result
}
