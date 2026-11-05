/**
 * TaskKanbanCard - Draggable task card for the Kanban board.
 * Modeled on ecosystem-manager TaskCard with prompt-manager task fields.
 */

import { useRef, useState, useEffect, type PointerEvent as ReactPointerEvent } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Trash2, ChevronDown, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { MarkdownRenderer } from '@/components/markdown'
import { formatRelativePastTime } from '@/lib/timeUtils'
import type { TeamTask } from '@/services/heartbeatService'

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const PRIORITY_STYLES: Record<string, { bg: string; text: string }> = {
  P1: { bg: 'bg-red-500/20', text: 'text-red-300' },
  P2: { bg: 'bg-orange-500/20', text: 'text-orange-300' },
  P3: { bg: 'bg-yellow-500/20', text: 'text-yellow-300' },
  P4: { bg: 'bg-blue-500/20', text: 'text-blue-300' },
  P5: { bg: 'bg-slate-500/20', text: 'text-slate-300' },
}

// ---------------------------------------------------------------------------
// InlineEditableTitle
// ---------------------------------------------------------------------------

function InlineEditableTitle({
  value,
  onSave,
}: {
  value: string
  onSave: (v: string) => void
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) {
      setDraft(value)
      requestAnimationFrame(() => inputRef.current?.focus())
    }
  }, [editing, value])

  const commit = () => {
    const trimmed = draft.trim()
    if (trimmed && trimmed !== value) onSave(trimmed)
    setEditing(false)
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        value={draft}
        onChange={e => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={e => {
          if (e.key === 'Enter') commit()
          if (e.key === 'Escape') setEditing(false)
        }}
        onClick={e => e.stopPropagation()}
        onPointerDown={e => e.stopPropagation()}
        className="w-full rounded border border-border bg-background px-1.5 py-0.5 text-sm text-foreground outline-none focus:ring-1 focus:ring-primary"
      />
    )
  }

  return (
    <span
      onClick={e => { e.stopPropagation(); setEditing(true) }}
      onPointerDown={e => e.stopPropagation()}
      className="cursor-text break-words text-sm hover:underline hover:decoration-dotted line-clamp-2"
      title="Click to edit"
    >
      {value}
    </span>
  )
}

// ---------------------------------------------------------------------------
// TaskKanbanCard
// ---------------------------------------------------------------------------

export interface TaskKanbanCardProps {
  task: TeamTask
  members: TeamMember[]
  allAgents?: Agent[]
  onUpdateTask: (taskId: string, update: Record<string, unknown>) => void
  onDeleteTask: (task: TeamTask) => void
  dragOverlay?: boolean
}

export function TaskKanbanCard({
  task,
  members,
  allAgents,
  onUpdateTask,
  onDeleteTask,
  dragOverlay = false,
}: TaskKanbanCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: task.id,
    data: { type: 'task', task },
  })

  const [notesExpanded, setNotesExpanded] = useState(false)

  const pointerStartRef = useRef<{ x: number; y: number } | null>(null)
  const dragIntentRef = useRef(false)

  const { onPointerDown, onPointerMove, onPointerUp, ...restListeners } = listeners ?? {}

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const handlePointerDown = (event: ReactPointerEvent) => {
    pointerStartRef.current = { x: event.clientX, y: event.clientY }
    dragIntentRef.current = false
    onPointerDown?.(event)
  }

  const handlePointerMove = (event: ReactPointerEvent) => {
    if (pointerStartRef.current && !dragIntentRef.current) {
      const dx = event.clientX - pointerStartRef.current.x
      const dy = event.clientY - pointerStartRef.current.y
      if (Math.hypot(dx, dy) > 6) {
        dragIntentRef.current = true
      }
    }
    onPointerMove?.(event)
  }

  const handlePointerUp = (event: ReactPointerEvent) => {
    onPointerUp?.(event)
    pointerStartRef.current = null
  }

  const getAgentAppearance = (agentId: string) =>
    allAgents?.find(a => a.id === agentId)?.appearance ?? null

  const memberName = members.find(m => m.agentId === task.assignee)?.displayName ?? task.assignee
  const priorityStyle = PRIORITY_STYLES[task.priority] ?? { bg: 'bg-yellow-500/20', text: 'text-yellow-300' }
  const noteCount = task.notes?.length ?? 0

  const cardContent = (
    <>
      {/* Top: Priority + Title */}
      <div className="flex items-start gap-2">
        <span className={cn('shrink-0 rounded px-1.5 py-0.5 text-xs font-mono font-semibold', priorityStyle.bg, priorityStyle.text)}>
          {task.priority}
        </span>
        <div className="flex-1 min-w-0">
          <InlineEditableTitle
            value={task.title}
            onSave={v => onUpdateTask(task.id, { title: v })}
          />
        </div>
      </div>

      {/* Middle: Assignee */}
      {task.assignee && (
        <div className="flex items-center gap-1.5 mt-2">
          <AgentColorBadge appearance={getAgentAppearance(task.assignee)} size="xs" />
          <span className="text-xs text-muted-foreground truncate">{memberName}</span>
        </div>
      )}

      {/* Bottom: Timestamp + Notes + Delete */}
      <div className="flex items-center justify-between mt-2 pt-2 border-t border-border/50">
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {task.updatedAt ? formatRelativePastTime(new Date(task.updatedAt)) : '\u2014'}
          </span>
          {noteCount > 0 && (
            <button
              onClick={e => { e.stopPropagation(); setNotesExpanded(v => !v) }}
              onPointerDown={e => e.stopPropagation()}
              className="flex items-center gap-0.5 text-xs text-primary hover:underline"
            >
              <MessageSquare className="h-3 w-3" />
              {noteCount}
              <ChevronDown className={cn('h-3 w-3 transition-transform', notesExpanded && 'rotate-180')} />
            </button>
          )}
        </div>
        <button
          onClick={e => { e.stopPropagation(); onDeleteTask(task) }}
          onPointerDown={e => e.stopPropagation()}
          className="rounded p-1 text-muted-foreground hover:text-destructive hover:bg-red-500/10 transition-colors"
          title="Delete task"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Expanded notes */}
      {notesExpanded && task.notes && task.notes.length > 0 && (
        <div className="mt-2 overflow-hidden rounded-md bg-muted/20 px-2.5 py-2 border border-border/50 space-y-1">
          {task.notes.map((note, i) => (
            <div key={i} className="min-w-0 text-xs text-muted-foreground">
              <span className="font-mono">{note.at ? formatRelativePastTime(new Date(note.at)) : ''}</span>
              {note.by && <span className="ml-1 opacity-70">({note.by})</span>}
              <MarkdownRenderer
                content={note.text}
                className="ml-1 inline break-words text-xs text-muted-foreground [&_*]:break-words [&_pre]:max-w-full [&_pre]:overflow-x-auto"
              />
            </div>
          ))}
        </div>
      )}
    </>
  )

  if (dragOverlay) {
    return (
      <div className="bg-card border border-border rounded-lg p-3 shadow-xl ring-2 ring-primary/40 rotate-3 w-72">
        {cardContent}
      </div>
    )
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...restListeners}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerLeave={() => { pointerStartRef.current = null }}
      className={cn(
        'bg-card border border-border rounded-lg p-3',
        'hover:border-primary/30 hover:bg-muted/60 transition-all',
        'cursor-grab active:cursor-grabbing',
        isDragging && 'shadow-xl ring-2 ring-primary/40',
      )}
    >
      {cardContent}
    </div>
  )
}
