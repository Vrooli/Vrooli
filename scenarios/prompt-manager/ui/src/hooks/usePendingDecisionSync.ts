/**
 * usePendingDecisionSync - Syncs pending decisions to stores for cross-component access.
 * Called once in SkillManagerLayout, distributes data to pendingDecisionsStore and accessoryStore.
 */
import { useEffect } from 'react'
import { usePendingDecisions, type UsePendingDecisionsResult } from './usePendingDecisions'
import { usePendingDecisionsStore } from '@/stores/pendingDecisionsStore'
import { useAccessoryStore } from '@/stores/accessoryStore'

export function usePendingDecisionSync(): UsePendingDecisionsResult {
  const result = usePendingDecisions()
  const { groupedByTeam, count } = result

  useEffect(() => {
    // Sync to pending decisions store for world view consumption
    usePendingDecisionsStore.getState().setGroups(
      groupedByTeam,
      count,
    )

    // Sync to accessory store: set pending-decision status for submitting agents
    const accessoryStore = useAccessoryStore.getState()
    const agentIds = new Set<string>()

    for (const group of groupedByTeam) {
      for (const entry of group.entries) {
        if (entry.by) {
          agentIds.add(entry.by)
          const truncatedDecision = entry.decision.length > 60
            ? entry.decision.slice(0, 57) + '...'
            : entry.decision
          accessoryStore.setAgentStatus(entry.by, {
            type: 'pending-decision',
            message: `Decision awaiting review: ${truncatedDecision}`,
            source: 'decision',
          })
        }
      }
    }

    // Cleanup: clear statuses for agents no longer in pending list
    return () => {
      for (const agentId of agentIds) {
        const currentStatus = accessoryStore.getAgentStatus(agentId)
        if (currentStatus?.source === 'decision') {
          accessoryStore.clearAgentStatus(agentId)
        }
      }
    }
  }, [groupedByTeam, count])

  // Clear store on unmount
  useEffect(() => {
    return () => {
      usePendingDecisionsStore.getState().clear()
    }
  }, [])

  return result
}
