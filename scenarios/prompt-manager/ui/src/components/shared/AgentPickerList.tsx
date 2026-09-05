/**
 * AgentPickerList - Lightweight agent list for the mobile bottom-sheet picker.
 * Tapping an agent focuses it in the world view through the route.
 */

import { useNavigate } from 'react-router-dom'
import { useAgentData } from '@/hooks/useAgentData'
import { worldPath } from '@/app/routes/route-paths'
import { AgentColorBadge } from './AgentColorBadge'

interface AgentPickerListProps {
  /** Called after an agent is selected (e.g. to close the sheet) */
  onSelect?: () => void
}

export function AgentPickerList({ onSelect }: AgentPickerListProps) {
  const { agents, isLoading } = useAgentData()
  const navigate = useNavigate()

  if (isLoading) {
    return <p className="text-sm text-muted-foreground px-2 py-4">Loading agents...</p>
  }

  if (agents.length === 0) {
    return <p className="text-sm text-muted-foreground px-2 py-4">No agents found.</p>
  }

  return (
    <div className="flex flex-col gap-0.5">
      {agents.map((agent) => (
        <button
          key={agent.id}
          type="button"
          className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted/50 active:bg-muted transition-colors text-left w-full"
          onClick={() => {
            navigate(worldPath({ focus: agent.id }))
            onSelect?.()
          }}
        >
          <AgentColorBadge appearance={agent.appearance} size="xs" className="flex-shrink-0" />
          <span className="text-sm font-medium text-foreground truncate flex-1">
            {agent.displayName}
          </span>
        </button>
      ))}
    </div>
  )
}
