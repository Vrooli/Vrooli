/**
 * AgentListPanel - Panel for listing and managing agents.
 */

import { Plus, User, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAgentData } from '@/hooks/useAgentData'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { selectors } from '@/constants/selectors'

interface AgentListPanelProps {
  selectedAgentId: string | null
  onSelectAgent: (id: string) => void
  className?: string
}

/**
 * Agent list panel for the sidebar.
 */
export function AgentListPanel({
  selectedAgentId,
  onSelectAgent,
  className,
}: AgentListPanelProps) {
  const { agents, isLoading, isError, createAgent, deleteAgent } = useAgentData()

  const handleCreateAgent = async () => {
    const name = `Agent ${agents.length + 1}`
    const newAgent = await createAgent({
      displayName: name,
      appearance: {
        body: DEFAULT_AGENT_COLORS.body,
        head: DEFAULT_AGENT_COLORS.head,
        accent: DEFAULT_AGENT_COLORS.accent,
      },
    })
    // Auto-select the newly created agent
    onSelectAgent(newAgent.id)
  }

  const handleDeleteAgent = async (id: string) => {
    await deleteAgent(id)
  }

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <p className="text-sm text-destructive">Failed to load agents</p>
      </div>
    )
  }

  return (
    <div
      className={cn('flex flex-col h-full', className)}
      data-testid={selectors.agents.list}
    >
      {/* Agent list */}
      <div className="flex-1 overflow-y-auto py-1">
        {agents.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <User className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No agents yet</p>
            <button
              type="button"
              onClick={() => void handleCreateAgent()}
              className="text-xs text-primary hover:underline"
            >
              Create your first agent
            </button>
          </div>
        ) : (
          agents.map((agent) => (
            <button
              key={agent.id}
              type="button"
              onClick={() => onSelectAgent(agent.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedAgentId === agent.id && 'bg-primary/10'
              )}
              data-testid={selectors.agents.row}
              data-agent-id={agent.id}
            >
              {/* Agent color badge */}
              <AgentColorBadge appearance={agent.appearance} size="sm" />

              {/* Agent info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {agent.displayName}
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    void handleDeleteAgent(agent.id)
                  }}
                  className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                  title="Delete agent"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer - New agent button */}
      <div className="flex-shrink-0 px-3 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleCreateAgent()}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
          )}
          data-testid={selectors.agents.newButton}
        >
          <Plus className="h-4 w-4" />
          New Agent
        </button>
      </div>
    </div>
  )
}
