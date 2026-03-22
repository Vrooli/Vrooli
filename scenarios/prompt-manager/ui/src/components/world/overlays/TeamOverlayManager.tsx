/**
 * TeamOverlayManager — renders TeamCountdownOverlay for each active team.
 * Placed inside the R3F scene (WorldScene).
 */

import { useMemo } from 'react'
import { useTeamActivityStore } from '@/stores/teamActivityStore'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { usePendingDecisionsStore } from '@/stores/pendingDecisionsStore'
import { TeamCountdownOverlay } from './TeamCountdownOverlay'

interface TeamOverlayManagerProps {
  onTeamClick?: (teamId: string) => void
}

/** Base Y offset above furniture; stacks add 0.5 each */
const BASE_Y_OFFSET = 1.5
const STACK_GAP = 0.5

export function TeamOverlayManager({ onTeamClick }: TeamOverlayManagerProps) {
  const activities = useTeamActivityStore((s) => s.activities)
  const allocations = useTeamActivityStore((s) => s.allocations)
  const getFurniture = useFurnitureStore((s) => s.getFurniture)
  const getTeamPendingCount = usePendingDecisionsStore((s) => s.getTeamPendingCount)

  // Group activities by their allocated furniture (or fallback position key)
  const overlayGroups = useMemo(() => {
    const allocationMap = new Map(allocations.map((allocation) => [allocation.teamId, allocation]))
    const groups = new Map<string, {
      position: [number, number, number]
      activities: typeof activities
    }>()

    for (const activity of activities) {
      const allocation = allocationMap.get(activity.teamId)
      if (!allocation) continue

      let position: [number, number, number]
      let groupKey: string

      if (allocation.furnitureId) {
        const furn = getFurniture(allocation.furnitureId)
        if (!furn) continue
        position = furn.position
        groupKey = allocation.furnitureId
      } else if (allocation.fallbackPosition) {
        position = allocation.fallbackPosition
        groupKey = `fallback-${activity.teamId}`
      } else {
        continue
      }

      const existing = groups.get(groupKey)
      if (existing) {
        existing.activities.push(activity)
      } else {
        groups.set(groupKey, { position, activities: [activity] })
      }
    }

    return groups
  }, [activities, allocations, getFurniture])

  return (
    <>
      {[...overlayGroups.entries()].map(([groupKey, group]) =>
        group.activities.map((activity, idx) => (
          <TeamCountdownOverlay
            key={`${groupKey}-${activity.teamId}`}
            activity={activity}
            position={group.position}
            yOffset={BASE_Y_OFFSET + idx * STACK_GAP}
            onClick={() => onTeamClick?.(activity.teamId)}
            pendingDecisionCount={getTeamPendingCount(activity.teamId)}
          />
        )),
      )}
    </>
  )
}
