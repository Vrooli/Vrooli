/**
 * ActionListPanel - Sidebar browse surface for executable Action contracts.
 */

import { useMemo, useState } from 'react'
import { Bolt, CheckCircle2, CircleDashed, Archive, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useActionsData } from '@/hooks/useActionsData'
import type { Action, CreateActionRequest } from '@/types'

interface ActionListPanelProps {
  selectedActionId: string | null
  onSelectAction: (id: string) => void
  searchQuery?: string
  className?: string
}

export function ActionListPanel({
  selectedActionId,
  onSelectAction,
  searchQuery,
  className,
}: ActionListPanelProps) {
  const { actions, isLoading, isError, createAction, isCreating } = useActionsData()
  const [createError, setCreateError] = useState<string | null>(null)

  const filteredActions = useMemo(() => {
    if (!searchQuery) return actions
    const lower = searchQuery.toLowerCase()
    return actions.filter((action) => {
      const owner = `${action.owner.type}:${action.owner.id}`
      const argv = action.command.argv.join(' ')
      return (
        action.id.toLowerCase().includes(lower) ||
        action.name.toLowerCase().includes(lower) ||
        action.description.toLowerCase().includes(lower) ||
        action.status.toLowerCase().includes(lower) ||
        owner.toLowerCase().includes(lower) ||
        argv.toLowerCase().includes(lower) ||
        action.tags.some((tag) => tag.toLowerCase().includes(lower))
      )
    })
  }, [actions, searchQuery])

  const handleCreate = async () => {
    setCreateError(null)
    const id = nextDraftId(actions)
    const request: CreateActionRequest = {
      kind: 'action',
      schemaVersion: 1,
      id,
      name: 'New Action',
      description: '',
      status: 'draft',
      owner: { type: 'scenario', id: 'prompt-manager' },
      command: { argv: ['prompt-manager', 'action', 'list'] },
      inputs: {},
      outputs: {},
      permissions: {
        filesystemRead: false,
        filesystemWrite: false,
        localhostNetwork: false,
        externalNetwork: false,
        apiRead: true,
        apiWrite: false,
        processStart: false,
        processStop: false,
        hostConfigure: false,
        secretRead: false,
        secretWrite: false,
        destructive: false,
      },
      examples: [],
      tags: [],
      validation: { mode: 'contract', argv: [] },
      pack: 'drafts',
    }

    try {
      const response = await createAction(request)
      onSelectAction(response.action.id)
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : 'Action creation failed')
    }
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
        <p className="text-sm text-destructive">Failed to load actions</p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col min-h-0', className)}>
      <div className="flex-1 overflow-y-auto py-1">
        {actions.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Bolt className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No actions yet</p>
            <button
              type="button"
              onClick={() => void handleCreate()}
              className="text-xs text-primary hover:underline"
              disabled={isCreating}
            >
              Create your first Action
            </button>
          </div>
        ) : filteredActions.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Bolt className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-60" />
            <p className="text-xs text-muted-foreground">No matching actions</p>
          </div>
        ) : (
          filteredActions.map((action) => (
            <ActionRow
              key={action.id}
              action={action}
              selected={selectedActionId === action.id}
              onSelect={() => onSelectAction(action.id)}
            />
          ))
        )}
      </div>

      {createError && (
        <div className="px-3 py-2 border-t border-border text-xs text-destructive">
          {createError}
        </div>
      )}

      <div className="flex-shrink-0 px-3 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleCreate()}
          disabled={isCreating}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors',
            isCreating && 'opacity-50 cursor-not-allowed'
          )}
        >
          <Plus className="h-4 w-4" />
          New Action
        </button>
      </div>
    </div>
  )
}

function ActionRow({
  action,
  selected,
  onSelect,
}: {
  action: Action
  selected: boolean
  onSelect: () => void
}) {
  const StatusIcon = action.status === 'active'
    ? CheckCircle2
    : action.status === 'archived'
      ? Archive
      : CircleDashed

  return (
    <button
      type="button"
      onClick={onSelect}
      aria-current={selected ? 'true' : undefined}
      aria-label={`${action.name}, ${action.status} Action owned by ${action.owner.type}:${action.owner.id}`}
      className={cn(
        'w-full flex items-start gap-3 px-3 py-2 text-left group',
        'hover:bg-muted/50 transition-colors',
        'focus:outline-none focus:bg-muted/50 focus:ring-2 focus:ring-primary/30',
        selected && 'bg-primary/10'
      )}
      data-action-id={action.id}
    >
      <div className="flex-shrink-0 w-7 h-7 rounded-md bg-muted flex items-center justify-center mt-0.5">
        <Bolt className="h-3.5 w-3.5 text-muted-foreground" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">{action.name}</p>
          <StatusIcon className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
        </div>
        <p className="mt-0.5 text-[11px] text-muted-foreground truncate">{action.id}</p>
        <p className="mt-1 text-[11px] text-muted-foreground truncate">
          {action.command.argv.join(' ')}
        </p>
      </div>
    </button>
  )
}

function nextDraftId(actions: Action[]): string {
  const existing = new Set(actions.map((action) => action.id))
  for (let i = 1; i < 1000; i += 1) {
    const id = `action.draft.${i}`
    if (!existing.has(id)) return id
  }
  return `action.draft.${Date.now()}`
}
