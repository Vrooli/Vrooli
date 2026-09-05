/**
 * ActionListPanel - Sidebar browse surface for executable Action contracts.
 */

import { useMemo, useState } from 'react'
import { Archive, Bolt, CheckCircle2, CircleDashed, ExternalLink, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useActionsData } from '@/hooks/useActionsData'
import type { Action, CreateActionRequest } from '@/types'

interface ActionListPanelProps {
  selectedActionId: string | null
  onSelectAction: (id: string) => void
  searchQuery?: string
  className?: string
  isSelectMode?: boolean
  selectedIds?: Set<string>
  onToggleSelection?: (id: string) => void
}

export function ActionListPanel({
  selectedActionId,
  onSelectAction,
  searchQuery,
  className,
  isSelectMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
}: ActionListPanelProps) {
  const { actions, isLoading, isError, createAction, isCreating } = useActionsData()
  const [createError, setCreateError] = useState<string | null>(null)

  const filteredActions = useMemo(() => {
    if (!searchQuery) return actions
    const lower = searchQuery.toLowerCase()
    return actions.filter((action) => {
      return actionSearchText(action).includes(lower)
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
              checked={selectedIds.has(action.id)}
              isSelectMode={isSelectMode}
              onSelect={() => onSelectAction(action.id)}
              onToggle={() => onToggleSelection?.(action.id)}
            />
          ))
        )}
      </div>

      {createError && (
        <div className="px-3 py-2 border-t border-border text-xs text-destructive">
          {createError}
        </div>
      )}

      {!isSelectMode && (
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
      )}
    </div>
  )
}

function ActionRow({
  action,
  selected,
  checked,
  isSelectMode,
  onSelect,
  onToggle,
}: {
  action: Action
  selected: boolean
  checked: boolean
  isSelectMode: boolean
  onSelect: () => void
  onToggle: () => void
}) {
  const StatusIcon = action.status === 'active'
    ? CheckCircle2
    : action.status === 'archived'
      ? Archive
      : CircleDashed

  return (
    <div
      className={cn(
        'w-full flex items-start gap-3 px-3 py-2 text-left group',
        'hover:bg-muted/50 transition-colors',
        selected && 'bg-primary/10'
      )}
      data-action-id={action.id}
    >
      {isSelectMode && (
        <button
          type="button"
          onClick={onToggle}
          aria-pressed={checked}
          aria-label={`${checked ? 'Deselect' : 'Select'} ${action.name}`}
          className="flex-shrink-0 pt-1 focus:outline-none focus:ring-2 focus:ring-primary/30 rounded"
        >
          <span
            className={cn(
              'flex h-4 w-4 items-center justify-center rounded border transition-colors',
              checked ? 'bg-primary border-primary' : 'border-border bg-background'
            )}
          >
            {checked && (
              <svg viewBox="0 0 16 16" className="h-4 w-4 text-primary-foreground" fill="currentColor">
                <path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
              </svg>
            )}
          </span>
        </button>
      )}
      <div className="flex-shrink-0 w-7 h-7 rounded-md bg-muted flex items-center justify-center mt-0.5">
        <Bolt className="h-3.5 w-3.5 text-muted-foreground" />
      </div>
      <button
        type="button"
        onClick={isSelectMode ? onToggle : onSelect}
        aria-current={selected ? 'true' : undefined}
        aria-pressed={isSelectMode ? checked : undefined}
        aria-label={`${action.name}, ${action.status} Action owned by ${action.owner.type}:${action.owner.id}`}
        className={cn(
          'min-w-0 flex-1 text-left rounded-md',
          'focus:outline-none focus:bg-muted/50 focus:ring-2 focus:ring-primary/30'
        )}
      >
        <div className="flex items-center gap-1.5 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">{action.name}</p>
          <StatusIcon className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
        </div>
        <p className="mt-0.5 text-[11px] text-muted-foreground truncate">{action.id}</p>
        <p className="mt-1 text-[11px] text-muted-foreground truncate">
          {action.command.argv.join(' ')}
        </p>
      </button>
      {isSelectMode && (
        <button
          type="button"
          onClick={onSelect}
          className="flex-shrink-0 p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          title="Go to action"
          aria-label={`Go to ${action.name}`}
        >
          <ExternalLink className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}

function actionSearchText(action: Action): string {
  const chunks: string[] = [
    action.id,
    action.name,
    action.description,
    action.status,
    action.owner.type,
    action.owner.id,
    action.command.argv.join(' '),
    ...action.tags,
  ]

  for (const [name, input] of Object.entries(action.inputs)) {
    chunks.push(name, input.type, input.description, input.pattern, ...input.enum)
    if (input.default !== undefined) {
      const defaultText = typeof input.default === 'string' ? input.default : JSON.stringify(input.default)
      if (defaultText) chunks.push(defaultText)
    }
  }
  for (const [name, output] of Object.entries(action.outputs)) {
    chunks.push(name, output.type, output.description)
  }
  for (const [permission, enabled] of Object.entries(action.permissions)) {
    if (enabled) chunks.push(permission)
  }
  for (const example of action.examples) {
    chunks.push(example.description, JSON.stringify(example.input))
  }

  return chunks.join(' ').toLowerCase()
}

function nextDraftId(actions: Action[]): string {
  const existing = new Set(actions.map((action) => action.id))
  for (let i = 1; i < 1000; i += 1) {
    const id = `action.draft.${i}`
    if (!existing.has(id)) return id
  }
  return `action.draft.${Date.now()}`
}
