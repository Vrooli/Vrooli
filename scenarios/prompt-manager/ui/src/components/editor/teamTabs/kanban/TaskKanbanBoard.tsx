/**
 * TaskKanbanBoard - 4-column Kanban board for team tasks.
 * Modeled on ecosystem-manager KanbanBoard with prompt-manager task model.
 */

import { useEffect, useState, useCallback, useMemo, useRef } from 'react'
import {
  DndContext,
  DragOverlay,
  closestCenter,
  pointerWithin,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { Plus, X, Check, Loader2, ListTodo } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import * as heartbeatService from '@/services/heartbeatService'
import type { TeamTask } from '@/services/heartbeatService'
import { TaskKanbanColumn } from './TaskKanbanColumn'
import { TaskKanbanCard } from './TaskKanbanCard'

// ---------------------------------------------------------------------------
// Column definitions
// ---------------------------------------------------------------------------

type TaskStatus = 'todo' | 'in-progress' | 'blocked' | 'done'

const COLUMNS: Array<{ status: TaskStatus; title: string }> = [
  { status: 'todo', title: 'To Do' },
  { status: 'in-progress', title: 'In Progress' },
  { status: 'blocked', title: 'Blocked' },
  { status: 'done', title: 'Done' },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface TaskKanbanBoardProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TaskKanbanBoard({ teamId, members, allAgents }: TaskKanbanBoardProps) {
  const [tasks, setTasks] = useState<TeamTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
  const [activeTask, setActiveTask] = useState<TeamTask | null>(null)

  // Add form
  const [showAddForm, setShowAddForm] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newAssignee, setNewAssignee] = useState('')
  const [newPriority, setNewPriority] = useState('P3')
  const addInputRef = useRef<HTMLInputElement>(null)

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<TeamTask | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  // Horizontal scroll ref
  const boardRef = useRef<HTMLDivElement | null>(null)
  const scrollLockRef = useRef<{ timeout: number | null }>({ timeout: null })

  // ---- Data fetching ----

  const loadTasks = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getTaskBoard(teamId)
      setTasks(resp.tasks)
    } catch (err) {
      console.error('[TaskKanbanBoard] Failed to load tasks:', err)
      setError(err instanceof Error ? err.message : 'Failed to load tasks')
    } finally {
      setLoading(false)
    }
  }, [teamId])

  useEffect(() => {
    void loadTasks()
    const interval = setInterval(() => void loadTasks(), 30_000)
    return () => clearInterval(interval)
  }, [loadTasks])

  // Focus add form input when opened
  useEffect(() => {
    if (showAddForm) {
      requestAnimationFrame(() => addInputRef.current?.focus())
    }
  }, [showAddForm])

  // ---- Group tasks by status ----

  const tasksByStatus = useMemo(() => {
    const grouped: Record<TaskStatus, TeamTask[]> = {
      todo: [],
      'in-progress': [],
      blocked: [],
      done: [],
    }
    for (const task of tasks) {
      const status = task.status as TaskStatus
      if (status in grouped) {
        grouped[status].push(task)
      }
    }
    return grouped
  }, [tasks])

  // ---- Drag-and-drop ----

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
  )

  const collisionDetection = useCallback((args: Parameters<typeof pointerWithin>[0]) => {
    const intersections = pointerWithin(args)
    if (intersections.length > 0) return intersections
    return closestCenter(args)
  }, [])

  const handleDragStart = (event: DragStartEvent) => {
    const task = event.active.data.current?.task as TeamTask | undefined
    if (task) setActiveTask(task)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveTask(null)

    if (!over) return

    const taskId = active.id as string
    const overData = over.data.current as Record<string, unknown> | undefined
    const overType = overData?.type as string | undefined
    let newStatus: TaskStatus | null = null

    if (overType === 'column') {
      newStatus = (overData?.status as TaskStatus | undefined) ?? null
    } else if (overType === 'task') {
      const overTask = overData?.task as TeamTask | undefined
      newStatus = (overTask?.status as TaskStatus | undefined) ?? null
      if (!newStatus) {
        const targetTask = tasks.find(t => t.id === over.id)
        newStatus = (targetTask?.status as TaskStatus | undefined) ?? null
      }
    } else if (typeof over.id === 'string') {
      const maybeStatus = over.id as TaskStatus
      if (COLUMNS.some(col => col.status === maybeStatus)) {
        newStatus = maybeStatus
      }
    }

    if (!newStatus) return

    const task = tasks.find(t => t.id === taskId)
    if (task && task.status !== newStatus) {
      void handleUpdateTask(taskId, { status: newStatus })
    }
  }

  // ---- Horizontal wheel scroll ----

  useEffect(() => {
    const board = boardRef.current
    if (!board) return

    const handleWheel = (event: WheelEvent) => {
      if (event.defaultPrevented) return

      const scrollLock = scrollLockRef.current
      const horizontalLockActive = scrollLock.timeout !== null

      const columnBody = (event.target as HTMLElement | null)?.closest('.kanban-column-body') as HTMLElement | null
      if (columnBody && !horizontalLockActive) {
        const canScrollVertically = columnBody.scrollHeight > columnBody.clientHeight
        if (canScrollVertically) {
          const scrollTop = columnBody.scrollTop
          const atTop = scrollTop <= 0
          const atBottom = (columnBody.scrollHeight - columnBody.clientHeight - scrollTop) <= 1
          if ((event.deltaY < 0 && !atTop) || (event.deltaY > 0 && !atBottom)) {
            return
          }
        }
      }

      if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
        event.preventDefault()
        board.scrollLeft += event.deltaY

        if (scrollLock.timeout !== null) {
          clearTimeout(scrollLock.timeout)
        }
        scrollLock.timeout = window.setTimeout(() => {
          scrollLock.timeout = null
        }, 250)
      }
    }

    board.addEventListener('wheel', handleWheel, { passive: false })
    return () => board.removeEventListener('wheel', handleWheel)
  }, [loading])

  // ---- Mutation helpers ----

  const handleMutationError = (label: string, err: unknown) => {
    const msg = err instanceof Error ? err.message : 'Unknown error'
    console.error(`[TaskKanbanBoard] ${label}:`, err)
    setMutationError(`${label}: ${msg}`)
    setTimeout(() => setMutationError(null), 5000)
  }

  const handleUpdateTask = async (taskId: string, update: Parameters<typeof heartbeatService.updateTask>[2]) => {
    try {
      setMutationError(null)
      await heartbeatService.updateTask(teamId, taskId, update)
      void loadTasks()
    } catch (err) {
      handleMutationError('Failed to update task', err)
    }
  }

  const handleAddTask = async () => {
    if (!newTitle.trim()) return
    try {
      setMutationError(null)
      await heartbeatService.addTask(teamId, {
        title: newTitle,
        assignee: newAssignee || undefined,
        priority: newPriority,
        from: 'ui-user',
      })
      setNewTitle('')
      setNewAssignee('')
      setNewPriority('P3')
      setShowAddForm(false)
      void loadTasks()
    } catch (err) {
      handleMutationError('Failed to add task', err)
    }
  }

  const handleDeleteTask = async () => {
    if (!deleteTarget) return
    try {
      setDeleteLoading(true)
      setMutationError(null)
      await heartbeatService.deleteTask(teamId, deleteTarget.id)
      setDeleteTarget(null)
      void loadTasks()
    } catch (err) {
      handleMutationError('Failed to delete task', err)
      setDeleteTarget(null)
    } finally {
      setDeleteLoading(false)
    }
  }

  const memberOptions = members.map(m => ({
    value: m.agentId,
    label: m.displayName || m.agentId,
  }))

  // ---- Loading state ----

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // ---- Error state ----

  if (error) {
    return (
      <div className="rounded-md border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
        Error loading tasks: {error}
        <button
          onClick={() => void loadTasks()}
          className="ml-2 text-xs underline hover:no-underline"
        >
          Retry
        </button>
      </div>
    )
  }

  const allEmpty = COLUMNS.every(col => tasksByStatus[col.status].length === 0)

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap shrink-0 px-3 py-2">
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className={cn(
            'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
            showAddForm
              ? 'bg-muted text-muted-foreground'
              : 'bg-primary text-primary-foreground hover:bg-primary/90',
          )}
        >
          {showAddForm ? <X className="h-3.5 w-3.5" /> : <Plus className="h-3.5 w-3.5" />}
          {showAddForm ? 'Close' : 'Add Task'}
        </button>
      </div>

      {/* Mutation error banner */}
      {mutationError && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 mx-3 px-3 py-2 text-xs text-destructive flex items-center justify-between shrink-0">
          <span>{mutationError}</span>
          <button onClick={() => setMutationError(null)} className="ml-2 hover:text-foreground">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* Add form */}
      {showAddForm && (
        <div className="rounded-lg border border-border bg-card mx-3 p-4 space-y-3 shadow-sm shrink-0">
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">New Task</div>
          <input
            ref={addInputRef}
            type="text"
            placeholder="What needs to be done?"
            value={newTitle}
            onChange={e => setNewTitle(e.target.value)}
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
            onKeyDown={e => { if (e.key === 'Enter') void handleAddTask() }}
          />
          <div className="flex items-center gap-2 flex-wrap">
            <select
              value={newAssignee}
              onChange={e => setNewAssignee(e.target.value)}
              className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">Unassigned</option>
              {memberOptions.map(m => (
                <option key={m.value} value={m.value}>{m.label}</option>
              ))}
            </select>
            <select
              value={newPriority}
              onChange={e => setNewPriority(e.target.value)}
              className="rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground outline-none focus:ring-1 focus:ring-primary"
            >
              {['P1', 'P2', 'P3', 'P4', 'P5'].map(p => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
            <div className="flex-1" />
            <button
              onClick={() => { setShowAddForm(false); setNewTitle(''); setNewAssignee(''); setNewPriority('P3') }}
              className="rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => void handleAddTask()}
              disabled={!newTitle.trim()}
              className={cn(
                'rounded-md px-4 py-1.5 text-xs font-medium transition-colors',
                newTitle.trim()
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-muted text-muted-foreground cursor-not-allowed',
              )}
            >
              <Check className="mr-1 inline-block h-3 w-3" />
              Create
            </button>
          </div>
        </div>
      )}

      {/* Empty state (all columns empty) */}
      {allEmpty && !showAddForm && (
        <div className="flex flex-col items-center justify-center gap-3 py-12 px-3 text-muted-foreground">
          <ListTodo className="h-10 w-10 opacity-40" />
          <p className="text-sm">No tasks yet. Create one to start tracking work.</p>
        </div>
      )}

      {/* Kanban columns */}
      {!allEmpty && (
        <DndContext
          sensors={sensors}
          collisionDetection={collisionDetection}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <div
            ref={boardRef}
            className="flex gap-0 flex-1 min-h-0 overflow-x-auto"
          >
            {COLUMNS.map(({ status }) => (
              <TaskKanbanColumn
                key={status}
                status={status}
                tasks={tasksByStatus[status]}
                members={members}
                allAgents={allAgents}
                onUpdateTask={(taskId, update) => void handleUpdateTask(taskId, update)}
                onDeleteTask={setDeleteTarget}
              />
            ))}
          </div>

          {/* Drag Overlay */}
          <DragOverlay dropAnimation={null}>
            {activeTask ? (
              <TaskKanbanCard
                task={activeTask}
                members={members}
                allAgents={allAgents}
                onUpdateTask={() => {}}
                onDeleteTask={() => {}}
                dragOverlay
              />
            ) : null}
          </DragOverlay>
        </DndContext>
      )}

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void handleDeleteTask()}
        title="Delete Task"
        message={`Are you sure you want to delete "${deleteTarget?.title ?? ''}"? This action cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        isLoading={deleteLoading}
      />
    </div>
  )
}
