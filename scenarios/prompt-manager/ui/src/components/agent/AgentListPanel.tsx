/**
 * AgentListPanel - Panel for listing and managing agents.
 */

import { useMemo } from 'react'
import { Copy, Eye, Palette, Plus, Trash2, User } from 'lucide-react'
import { CollectionList } from '@vrooli/react-component-library/CollectionList/1.0.0'
import type { RowAction } from '@vrooli/react-component-library/useCollection/1'
import { cn } from '@/lib/utils'
import { useAgentData } from '@/hooks/useAgentData'
import { DEFAULT_AGENT_COLORS, type Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { selectors } from '@/constants/selectors'
import { toast } from '@/hooks/use-toast'

function agentActions(
  onDuplicate: (id: string) => void,
  onCustomize: (id: string) => void,
  onPreviewPrompt: (id: string) => void,
  onDelete: (id: string) => void,
): RowAction<Agent>[] {
  return [
    { id: 'duplicate', label: 'Duplicate agent', icon: <Copy />, onSelect: (rows) => rows[0] && onDuplicate(rows[0].id) },
    { id: 'customize', label: 'Customize Appearance', icon: <Palette />, onSelect: (rows) => rows[0] && onCustomize(rows[0].id) },
    { id: 'preview', label: 'Preview Prompt', icon: <Eye />, onSelect: (rows) => rows[0] && onPreviewPrompt(rows[0].id) },
    { id: 'delete', label: 'Delete Agent', icon: <Trash2 />, tone: 'destructive', separatorBefore: true, onSelect: (rows) => rows[0] && onDelete(rows[0].id) },
  ]
}

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

  const filteredAgents = useMemo(() => {
    if (!searchQuery) return agents
    const lower = searchQuery.toLowerCase()
    return agents.filter((a) => a.displayName.toLowerCase().includes(lower))
  }, [agents, searchQuery])

  const handleCreateAgent = async () => {
    // The list can be filtered or stale while another test/user creates an
    // agent. A timestamp suffix keeps the generated storage ID unique instead
    // of turning a normal "New Agent" click into a 409 conflict.
    const name = `Agent ${agents.length + 1}-${Date.now()}`
    try {
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
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Unable to create agent'
      console.error('Failed to create agent:', error)
      toast({ title: 'Agent creation failed', description, variant: 'destructive' })
    }
  }

  const handleDeleteAgent = async (id: string) => {
    await deleteAgent(id)
  }

  const actions = useMemo(
    () => agentActions(
      onDuplicateAgent ?? (() => {}),
      onCustomizeAgent ?? (() => {}),
      onPreviewPrompt ?? (() => {}),
      (id) => void handleDeleteAgent(id),
    ),
    [onCustomizeAgent, onDuplicateAgent, onPreviewPrompt, deleteAgent],
  )

  const syncSelection = (keys: string[]) => {
    if (!onToggleSelection) return
    const next = new Set(keys)
    selectedIds?.forEach((id) => { if (!next.has(id)) onToggleSelection(id) })
    next.forEach((id) => { if (!selectedIds?.has(id)) onToggleSelection(id) })
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
      className={cn('flex flex-col min-h-0', className)}
      data-testid={selectors.agents.list}
    >
      <div className="min-h-0 flex-1 overflow-y-auto py-1">
        <CollectionList
          items={filteredAgents}
          getKey={(agent) => agent.id}
          label="Agents"
          virtualize
          height="100%"
          selection={{
            mode: isSelectMode ? 'multi' : 'none',
            selected: selectedIds ? [...selectedIds] : undefined,
            onChange: syncSelection,
          }}
          onOpen={(agent) => onSelectAgent(agent.id)}
          actions={actions}
          empty={agents.length === 0 ? (
            <div className="px-3 py-8 text-center">
              <User className="mx-auto mb-2 h-8 w-8 text-muted-foreground" />
              <p className="mb-4 text-xs text-muted-foreground">No agents yet</p>
              <button type="button" onClick={() => void handleCreateAgent()} className="text-xs text-primary hover:underline">Create your first agent</button>
            </div>
          ) : (
            <div className="px-3 py-8 text-center">
              <User className="mx-auto mb-2 h-8 w-8 text-muted-foreground opacity-60" />
              <p className="text-xs text-muted-foreground">No matching agents</p>
            </div>
          )}
          renderItem={(agent, state) => {
            const selected = isSelectMode ? state.selection.selected : selectedAgentId === agent.id
            return <div className={cn('flex w-full items-center gap-3 px-3 py-2 text-left transition-colors', selected && 'bg-primary/10')} data-testid={selectors.agents.row} data-agent-id={agent.id} aria-selected={selected || undefined}>
              <AgentColorBadge appearance={agent.appearance} size="sm" />
              <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-foreground">{agent.displayName}</p></div>
            </div>
          }}
          className="h-full w-full"
        />
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
    </div>
  )
}
