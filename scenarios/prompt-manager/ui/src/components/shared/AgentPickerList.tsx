/**
 * AgentPickerList - Lightweight agent list for the mobile bottom-sheet picker.
 * Tapping an agent zooms the camera to it in the world view.
 */

import { useAgentData } from '@/hooks/useAgentData'
import { useAgentPositionStore } from '@/stores/agentPositionStore'
import { useCameraStore } from '@/stores/cameraStore'
import { useAccessoryStore } from '@/stores/accessoryStore'
import { AgentColorBadge } from './AgentColorBadge'

interface AgentPickerListProps {
  /** Called after an agent is selected (e.g. to close the sheet) */
  onSelect?: () => void
}

export function AgentPickerList({ onSelect }: AgentPickerListProps) {
  const { agents, isLoading } = useAgentData()
  const getPosition = useAgentPositionStore((s) => s.getPosition)
  const zoomToAgent = useCameraStore((s) => s.zoomToAgent)
  const agentAccessories = useAccessoryStore((s) => s.agentAccessories)

  if (isLoading) {
    return <p className="text-sm text-muted-foreground px-2 py-4">Loading agents...</p>
  }

  if (agents.length === 0) {
    return <p className="text-sm text-muted-foreground px-2 py-4">No agents found.</p>
  }

  return (
    <div className="flex flex-col gap-0.5">
      {agents.map((agent) => {
        const position = getPosition(agent.id)
        const status = agentAccessories[agent.id]?.status
        const hasPendingDecision = status?.type === 'pending-decision'

        return (
          <button
            key={agent.id}
            type="button"
            className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted/50 active:bg-muted transition-colors text-left w-full"
            onClick={() => {
              if (position) {
                zoomToAgent(agent.id, position)
              }
              onSelect?.()
            }}
          >
            <AgentColorBadge appearance={agent.appearance} size="xs" className="flex-shrink-0" />
            <span className="text-sm font-medium text-foreground truncate flex-1">
              {agent.displayName}
            </span>
            {hasPendingDecision && (
              <span className="flex-shrink-0 w-2 h-2 rounded-full bg-amber-500" title="Has pending decision" />
            )}
          </button>
        )
      })}
    </div>
  )
}
