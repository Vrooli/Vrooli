/**
 * TaskBoardView - Responsive task board with inline editing.
 * Card-based on mobile, table on desktop.
 */

import { useEffect, useState, useCallback, useRef } from 'react'
import { Plus, Trash2, ChevronDown, Loader2, ListTodo, X, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { TeamTask } from '@/services/heartbeatService'

interface TaskBoardViewProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

const STATUS_OPTIONS = ['todo', 'in-progress', 'blocked', 'done'] as const
type TaskStatus = (typeof STATUS_OPTIONS)[number]

const STATUS_STYLES: Record<TaskStatus, { bg: string; text: string; dot: string }> = {
  todo: { bg: 'bg-slate-500/20', text: 'text-slate-300', dot: 'bg-slate-400' },
  'in-progress': { bg: 'bg-amber-500/20', text: 'text-amber-300', dot: 'bg-amber-400' },
  blocked: { bg: 'bg-red-500/20', text: 'text-red-300', dot: 'bg-red-400' },
  done: { bg: 'bg-emerald-500/20', text: 'text-emerald-300', dot: 'bg-emerald-400' },
}

const STATUS_LABELS: Record<TaskStatus, string> = {
  todo: 'To Do',
  'in-progress': 'In Progress',
  blocked: 'Blocked',
  done: 'Done',
}

const PRIORITY_STYLES: Record<string, { bg: string; text: string }> = {
  P1: { bg: 'bg-red-500/20', text: 'text-red-300' },
  P2: { bg: 'bg-orange-500/20', text: 'text-orange-300' },
  P3: { bg: 'bg-yellow-500/20', text: 'text-yellow-300' },
  P4: { bg: 'bg-blue-500/20', text: 'text-blue-300' },
  P5: { bg: 'bg-slate-500/20', text: 'text-slate-300' },
}

// ---------------------------------------------------------------------------
// StatusPill - colored pill with dropdown to change status
// ---------------------------------------------------------------------------
function StatusPill({
  status,
  onChange,
}: {
  status: string
  onChange: (s: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const styles = STATUS_STYLES[status as TaskStatus] ?? STATUS_STYLES.todo

  return (
    <div ref={ref} className="relative inline-block">
      <button
        onClick={() => setOpen(o => !o)}
        className={cn(
          'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
          styles.bg,
          styles.text,
          'hover:brightness-125 cursor-pointer select-none',
        )}
      >
        <span className={cn('h-1.5 w-1.5 rounded-full', styles.dot)} />
        {STATUS_LABELS[status as TaskStatus] ?? status}
        <ChevronDown className={cn('h-3 w-3 transition-transform', open && 'rotate-180')} />
      </button>

      {open && (
        <div className="absolute left-0 top-full z-50 mt-1 min-w-[140px] rounded-md border border-border bg-card py-1 shadow-lg">
          {STATUS_OPTIONS.map(s => {
            const st = STATUS_STYLES[s]
            return (
              <button
                key={s}
                onClick={() => { onChange(s); setOpen(false) }}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-1.5 text-xs transition-colors hover:bg-muted',
                  s === status ? 'font-semibold' : 'font-normal',
                )}
              >
                <span className={cn('h-2 w-2 rounded-full', st.dot)} />
                <span className={st.text}>{STATUS_LABELS[s]}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
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
      // Defer focus so the input is mounted
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
        className="w-full rounded border border-border bg-background px-1.5 py-0.5 text-sm text-foreground outline-none focus:ring-1 focus:ring-primary"
      />
    )
  }

  return (
    <span
      onClick={() => setEditing(true)}
      className="cursor-pointer text-sm hover:underline hover:decoration-dotted"
      title="Click to edit"
    >
      {value}
    </span>
  )
}

// ---------------------------------------------------------------------------
// InlineDropdown - generic inline <select>-style editor
// ---------------------------------------------------------------------------
function InlineDropdown({
  value,
  options,
  onChange,
  placeholder,
}: {
  value: string
  options: { value: string; label: string }[]
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <select
      value={value}
      onChange={e => onChange(e.target.value)}
      className="rounded border border-border bg-background px-1.5 py-0.5 text-xs text-foreground outline-none focus:ring-1 focus:ring-primary"
    >
      {placeholder && <option value="">{placeholder}</option>}
      {options.map(o => (
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </select>
  )
}

// ---------------------------------------------------------------------------
// PriorityBadge
// ---------------------------------------------------------------------------
function PriorityBadge({ priority }: { priority: string }) {
  const s = PRIORITY_STYLES[priority] ?? { bg: 'bg-yellow-500/20', text: 'text-yellow-300' }
  return (
    <span className={cn('rounded px-1.5 py-0.5 text-xs font-mono font-semibold', s.bg, s.text)}>
      {priority}
    </span>
  )
}

// ---------------------------------------------------------------------------
// NotesSection - expandable notes block
// ---------------------------------------------------------------------------
function NotesSection({ notes }: { notes: TeamTask['notes'] }) {
  if (!notes?.length) return null
  return (
    <div className="space-y-1 pt-1">
      {notes.map((note, i) => (
        <div key={i} className="text-xs text-muted-foreground">
          <span className="font-mono">{note.at ? formatRelativePastTime(new Date(note.at)) : ''}</span>
          {note.by && <span className="ml-1 opacity-70">({note.by})</span>}
          <span className="ml-1">&mdash; {note.text}</span>
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------
export function TaskBoardView({ teamId, members, allAgents }: TaskBoardViewProps) {
  const [tasks, setTasks] = useState<TeamTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [assigneeFilter, setAssigneeFilter] = useState('')
  const [sortBy, setSortBy] = useState<'priority' | 'status' | 'updatedAt'>('priority')
  const [showAddForm, setShowAddForm] = useState(false)
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null)

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<TeamTask | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  // Add form state
  const [newTitle, setNewTitle] = useState('')
  const [newAssignee, setNewAssignee] = useState('')
  const [newPriority, setNewPriority] = useState('P3')
  const addInputRef = useRef<HTMLInputElement>(null)

  const loadTasks = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getTaskBoard(teamId)
      setTasks(resp.tasks ?? [])
    } catch (err) {
      console.error('[TaskBoardView] Failed to load tasks:', err)
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

  // Focus the add-form input when it opens
  useEffect(() => {
    if (showAddForm) {
      requestAnimationFrame(() => addInputRef.current?.focus())
    }
  }, [showAddForm])

  const getAgentAppearance = (agentId: string) => {
    return allAgents?.find(a => a.id === agentId)?.appearance ?? null
  }

  const memberOptions = members.map(m => ({
    value: m.agentId,
    label: m.displayName ?? m.agentId,
  }))

  const priorityOptions = ['P1', 'P2', 'P3', 'P4', 'P5'].map(p => ({ value: p, label: p }))

  // --- Mutation helpers with error feedback ---

  const handleMutationError = (label: string, err: unknown) => {
    const msg = err instanceof Error ? err.message : 'Unknown error'
    console.error(`[TaskBoardView] ${label}:`, err)
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

  // Filter and sort
  const filtered = tasks
    .filter(t => !statusFilter || t.status === statusFilter)
    .filter(t => !assigneeFilter || t.assignee === assigneeFilter)
    .sort((a, b) => {
      if (sortBy === 'priority') return a.priority.localeCompare(b.priority)
      if (sortBy === 'status') {
        const order: Record<string, number> = { todo: 0, 'in-progress': 1, blocked: 2, done: 3 }
        return (order[a.status] ?? 4) - (order[b.status] ?? 4)
      }
      return b.updatedAt.localeCompare(a.updatedAt)
    })

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

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
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
        <div className="flex items-center gap-2">
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className="rounded border border-border bg-background px-2 py-1 text-xs text-foreground outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">All statuses</option>
            {STATUS_OPTIONS.map(s => (
              <option key={s} value={s}>{STATUS_LABELS[s]}</option>
            ))}
          </select>
          <select
            value={assigneeFilter}
            onChange={e => setAssigneeFilter(e.target.value)}
            className="rounded border border-border bg-background px-2 py-1 text-xs text-foreground outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">All agents</option>
            {members.map(m => (
              <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Mutation error banner */}
      {mutationError && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive flex items-center justify-between">
          <span>{mutationError}</span>
          <button onClick={() => setMutationError(null)} className="ml-2 hover:text-foreground">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* Add form */}
      {showAddForm && (
        <div className="rounded-lg border border-border bg-card p-4 space-y-3 shadow-sm">
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
              {members.map(m => (
                <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
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

      {/* Empty state */}
      {filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
          <ListTodo className="h-10 w-10 opacity-40" />
          <p className="text-sm">No tasks yet. Create one to start tracking work.</p>
        </div>
      ) : (
        <>
          {/* ========== DESKTOP TABLE (hidden below md) ========== */}
          <div className="hidden md:block rounded-lg border border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th
                    className="px-3 py-2 text-left text-xs font-medium cursor-pointer select-none hover:text-primary transition-colors"
                    onClick={() => setSortBy('priority')}
                  >
                    Prio {sortBy === 'priority' && <ChevronDown className="inline h-3 w-3" />}
                  </th>
                  <th className="px-3 py-2 text-left text-xs font-medium">Title</th>
                  <th
                    className="px-3 py-2 text-left text-xs font-medium cursor-pointer select-none hover:text-primary transition-colors"
                    onClick={() => setSortBy('status')}
                  >
                    Status {sortBy === 'status' && <ChevronDown className="inline h-3 w-3" />}
                  </th>
                  <th className="px-3 py-2 text-left text-xs font-medium">Assignee</th>
                  <th className="px-3 py-2 text-left text-xs font-medium">Notes</th>
                  <th
                    className="px-3 py-2 text-left text-xs font-medium cursor-pointer select-none hover:text-primary transition-colors"
                    onClick={() => setSortBy('updatedAt')}
                  >
                    Updated {sortBy === 'updatedAt' && <ChevronDown className="inline h-3 w-3" />}
                  </th>
                  <th className="w-8 px-3 py-2" />
                </tr>
              </thead>
              <tbody>
                {filtered.map(task => {
                  const isExpanded = expandedTaskId === task.id
                  return (
                    <tr key={task.id} className="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors">
                      <td className="px-3 py-2">
                        <InlineDropdown
                          value={task.priority}
                          options={priorityOptions}
                          onChange={v => void handleUpdateTask(task.id, { priority: v })}
                        />
                      </td>
                      <td className="px-3 py-2">
                        <InlineEditableTitle
                          value={task.title}
                          onSave={v => void handleUpdateTask(task.id, { title: v })}
                        />
                      </td>
                      <td className="px-3 py-2">
                        <StatusPill
                          status={task.status}
                          onChange={s => void handleUpdateTask(task.id, { status: s })}
                        />
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex items-center gap-1.5">
                          {task.assignee && (
                            <AgentColorBadge appearance={getAgentAppearance(task.assignee)} size="xs" />
                          )}
                          <InlineDropdown
                            value={task.assignee ?? ''}
                            options={memberOptions}
                            onChange={v => void handleUpdateTask(task.id, { assignee: v || undefined })}
                            placeholder="Unassigned"
                          />
                        </div>
                      </td>
                      <td className="px-3 py-2">
                        {(task.notes?.length ?? 0) > 0 && (
                          <button
                            onClick={() => setExpandedTaskId(isExpanded ? null : task.id)}
                            className="flex items-center gap-0.5 text-xs text-primary hover:underline"
                          >
                            {task.notes!.length}
                            <ChevronDown className={cn('h-3 w-3 transition-transform', isExpanded && 'rotate-180')} />
                          </button>
                        )}
                      </td>
                      <td className="px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
                        {task.updatedAt ? formatRelativePastTime(new Date(task.updatedAt)) : '\u2014'}
                      </td>
                      <td className="px-3 py-2">
                        <button
                          onClick={() => setDeleteTarget(task)}
                          className="text-muted-foreground hover:text-destructive transition-colors"
                          title="Delete task"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>

            {/* Expanded notes panel (desktop) */}
            {expandedTaskId && (() => {
              const task = filtered.find(t => t.id === expandedTaskId)
              if (!task?.notes?.length) return null
              return (
                <div className="border-t border-border px-4 py-3 bg-muted/20 space-y-1">
                  <div className="text-xs font-medium text-muted-foreground mb-1">Notes for: {task.title}</div>
                  <NotesSection notes={task.notes} />
                </div>
              )
            })()}
          </div>

          {/* ========== MOBILE CARDS (visible below md) ========== */}
          <div className="flex flex-col gap-2 md:hidden">
            {filtered.map(task => {
              const isExpanded = expandedTaskId === task.id
              const hasNotes = (task.notes?.length ?? 0) > 0
              return (
                <div
                  key={task.id}
                  className="rounded-lg border border-border bg-card p-3 space-y-2"
                >
                  {/* Top row: priority badge + title */}
                  <div className="flex items-start gap-2">
                    <PriorityBadge priority={task.priority} />
                    <div className="flex-1 min-w-0">
                      <InlineEditableTitle
                        value={task.title}
                        onSave={v => void handleUpdateTask(task.id, { title: v })}
                      />
                    </div>
                  </div>

                  {/* Middle row: status pill + assignee */}
                  <div className="flex items-center gap-3 flex-wrap">
                    <StatusPill
                      status={task.status}
                      onChange={s => void handleUpdateTask(task.id, { status: s })}
                    />
                    <div className="flex items-center gap-1.5">
                      {task.assignee && (
                        <AgentColorBadge appearance={getAgentAppearance(task.assignee)} size="xs" />
                      )}
                      <InlineDropdown
                        value={task.assignee ?? ''}
                        options={memberOptions}
                        onChange={v => void handleUpdateTask(task.id, { assignee: v || undefined })}
                        placeholder="Unassigned"
                      />
                    </div>
                  </div>

                  {/* Notes inline (mobile) */}
                  {hasNotes && (
                    <div>
                      <button
                        onClick={() => setExpandedTaskId(isExpanded ? null : task.id)}
                        className="flex items-center gap-1 text-xs text-primary hover:underline"
                      >
                        {task.notes!.length} note{task.notes!.length !== 1 && 's'}
                        <ChevronDown className={cn('h-3 w-3 transition-transform', isExpanded && 'rotate-180')} />
                      </button>
                      {isExpanded && (
                        <div className="mt-1.5 rounded-md bg-muted/20 px-2.5 py-2 border border-border/50">
                          <NotesSection notes={task.notes} />
                        </div>
                      )}
                    </div>
                  )}

                  {/* Bottom row: updated time + actions */}
                  <div className="flex items-center justify-between pt-1 border-t border-border/50">
                    <span className="text-xs text-muted-foreground">
                      {task.updatedAt ? formatRelativePastTime(new Date(task.updatedAt)) : '\u2014'}
                    </span>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => setDeleteTarget(task)}
                        className="rounded p-1 text-muted-foreground hover:text-destructive hover:bg-muted transition-colors"
                        title="Delete task"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </>
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
