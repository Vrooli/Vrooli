/**
 * AgentListPanel - Panel for listing and managing agents.
 */

import { useState, useMemo, useCallback } from 'react'
import { Plus, User, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAgentData } from '@/hooks/useAgentData'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { AgentContextMenu } from '@/components/agent/AgentContextMenu'
import { selectors } from '@/constants/selectors'

interface AgentListPanelProps {
  selectedAgentId: string | null
  onSelectAgent: (id: string) => void
  /** Filter agents by display name */
  searchQuery?: string
  className?: string
  /** Called when user requests to duplicate an agent via context menu */
  onDuplicateAgent?: (agentId: string) => void
  /** Called when user requests to customize an agent via context menu */
  onCustomizeAgent?: (agentId: string) => void
  /** Called when user requests to preview an agent's prompt via context menu */
  onPreviewPrompt?: (agentId: string) => void
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
}

/**
 * Agent list panel for the sidebar.
 */
export function AgentListPanel({
  selectedAgentId,
  onSelectAgent,
  searchQuery,
  className,
  onDuplicateAgent,
  onCustomizeAgent,
  onPreviewPrompt,
  isSelectMode,
  selectedIds,
  onToggleSelection,
}: AgentListPanelProps) {
  const { agents, isLoading, isError, createAgent, deleteAgent } = useAgentData()

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    agentId: string
    agentName: string
  } | null>(null)

  const filteredAgents = useMemo(() => {
    if (!searchQuery) return agents
    const lower = searchQuery.toLowerCase()
    return agents.filter((a) => a.displayName.toLowerCase().includes(lower))
  }, [agents, searchQuery])

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

  const handleContextMenu = useCallback((e: React.MouseEvent, agentId: string, agentName: string) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY, agentId, agentName })
  }, [])

  const handleCloseContextMenu = useCallback(() => {
    setContextMenu(null)
  }, [])

  const handleItemClick = useCallback((id: string) => {
    if (isSelectMode && onToggleSelection) {
      onToggleSelection(id)
    } else {
      onSelectAgent(id)
    }
  }, [isSelectMode, onToggleSelection, onSelectAgent])

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
        ) : filteredAgents.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <User className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-60" />
            <p className="text-xs text-muted-foreground">No matching agents</p>
          </div>
        ) : (
          filteredAgents.map((agent) => (
            <button
              key={agent.id}
              type="button"
              onClick={() => handleItemClick(agent.id)}
              onContextMenu={(e) => handleContextMenu(e, agent.id, agent.displayName)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                !isSelectMode && selectedAgentId === agent.id && 'bg-primary/10',
                isSelectMode && selectedIds?.has(agent.id) && 'bg-primary/10'
              )}
              data-testid={selectors.agents.row}
              data-agent-id={agent.id}
            >
              {/* Selection checkbox */}
              {isSelectMode && (
                <div className="flex-shrink-0">
                  <div
                    className={cn(
                      'h-4 w-4 rounded border transition-colors',
                      selectedIds?.has(agent.id)
                        ? 'bg-primary border-primary'
                        : 'border-border bg-background'
                    )}
                  >
                    {selectedIds?.has(agent.id) && (
                      <svg viewBox="0 0 16 16" className="h-4 w-4 text-primary-foreground" fill="currentColor">
                        <path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
                      </svg>
                    )}
                  </div>
                </div>
              )}

              {/* Agent color badge */}
              <AgentColorBadge appearance={agent.appearance} size="sm" />

              {/* Agent info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {agent.displayName}
                </p>
              </div>

              {/* Actions (hidden in select mode) */}
              {!isSelectMode && (
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
              )}
            </button>
          ))
        )}
      </div>

      {/* Footer - New agent button (hidden in select mode) */}
      {!isSelectMode && (
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
      )}

      {/* Context menu */}
      {contextMenu && (
        <AgentContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          agentId={contextMenu.agentId}
          agentName={contextMenu.agentName}
          onClose={handleCloseContextMenu}
          onDuplicate={onDuplicateAgent ?? (() => {})}
          onCustomize={onCustomizeAgent ?? (() => {})}
          onPreviewPrompt={onPreviewPrompt ?? (() => {})}
          onDelete={(id) => void handleDeleteAgent(id)}
        />
      )}
    </div>
  )
}
