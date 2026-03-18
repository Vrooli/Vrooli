/**
 * TaskBoardView - Table view of the team's task board.
 */

import { useEffect, useState, useCallback } from 'react'
import { Plus, Trash2, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { TeamTask } from '@/services/heartbeatService'

interface TaskBoardViewProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

const STATUS_OPTIONS = ['todo', 'in-progress', 'blocked', 'done'] as const
const STATUS_ICONS: Record<string, string> = {
  todo: '\u26AA',
  'in-progress': '\uD83D\uDFE1',
  blocked: '\uD83D\uDD34',
  done: '\uD83D\uDFE2',
}

export function TaskBoardView({ teamId, members, allAgents }: TaskBoardViewProps) {
  const [tasks, setTasks] = useState<TeamTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [assigneeFilter, setAssigneeFilter] = useState('')
  const [sortBy, setSortBy] = useState<'priority' | 'status' | 'updatedAt'>('priority')
  const [showAddForm, setShowAddForm] = useState(false)
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null)

  // Add form state
  const [newTitle, setNewTitle] = useState('')
  const [newAssignee, setNewAssignee] = useState('')
  const [newPriority, setNewPriority] = useState('P3')

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

  const getAgentName = (agentId: string) => {
    const member = members.find(m => m.agentId === agentId)
    return member?.displayName ?? agentId
  }

  const getAgentAppearance = (agentId: string) => {
    return allAgents?.find(a => a.id === agentId)?.appearance ?? null
  }

  const handleStatusChange = async (taskId: string, newStatus: string) => {
    try {
      await heartbeatService.updateTask(teamId, taskId, { status: newStatus })
      void loadTasks()
    } catch {
      // silently fail
    }
  }

  const handleAddTask = async () => {
    if (!newTitle.trim()) return
    try {
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
    } catch {
      // silently fail
    }
  }

  const handleDeleteTask = async (taskId: string) => {
    try {
      await heartbeatService.deleteTask(teamId, taskId)
      void loadTasks()
    } catch {
      // silently fail
    }
  }

  // Filter and sort
  const filtered = tasks
    .filter(t => !statusFilter || t.status === statusFilter)
    .filter(t => !assigneeFilter || t.assignee === assigneeFilter)
    .sort((a, b) => {
      if (sortBy === 'priority') return a.priority.localeCompare(b.priority)
      if (sortBy === 'status') {
        const order = { todo: 0, 'in-progress': 1, blocked: 2, done: 3 }
        return (order[a.status] ?? 4) - (order[b.status] ?? 4)
      }
      return b.updatedAt.localeCompare(a.updatedAt)
    })

  if (loading) {
    return <div className="text-sm text-muted-foreground">Loading tasks...</div>
  }

  if (error) {
    return (
      <div className="text-sm text-destructive">
        Error loading tasks: {error}
        <button onClick={() => void loadTasks()} className="ml-2 text-xs underline hover:no-underline">Retry</button>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="flex items-center gap-1 text-xs px-2 py-1 border rounded hover:bg-muted"
        >
          <Plus className="h-3 w-3" /> Add Task
        </button>
        <div className="flex items-center gap-2">
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className="text-xs border rounded px-2 py-1 bg-background text-foreground"
          >
            <option value="">All statuses</option>
            {STATUS_OPTIONS.map(s => (
              <option key={s} value={s}>{STATUS_ICONS[s]} {s}</option>
            ))}
          </select>
          <select
            value={assigneeFilter}
            onChange={e => setAssigneeFilter(e.target.value)}
            className="text-xs border rounded px-2 py-1 bg-background text-foreground"
          >
            <option value="">All agents</option>
            {members.map(m => (
              <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Add form */}
      {showAddForm && (
        <div className="border rounded-lg p-3 bg-card space-y-2">
          <input
            type="text"
            placeholder="Task title..."
            value={newTitle}
            onChange={e => setNewTitle(e.target.value)}
            className="w-full text-sm border rounded px-2 py-1 bg-background text-foreground"
            onKeyDown={e => { if (e.key === 'Enter') void handleAddTask() }}
          />
          <div className="flex items-center gap-2">
            <select
              value={newAssignee}
              onChange={e => setNewAssignee(e.target.value)}
              className="text-xs border rounded px-2 py-1 bg-background text-foreground"
            >
              <option value="">Unassigned</option>
              {members.map(m => (
                <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
              ))}
            </select>
            <select
              value={newPriority}
              onChange={e => setNewPriority(e.target.value)}
              className="text-xs border rounded px-2 py-1 bg-background text-foreground"
            >
              {['P1', 'P2', 'P3', 'P4', 'P5'].map(p => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
            <button
              onClick={() => void handleAddTask()}
              className="text-xs px-3 py-1 bg-primary text-primary-foreground rounded hover:bg-primary/90"
            >
              Create
            </button>
          </div>
        </div>
      )}

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="text-sm text-muted-foreground">No tasks found.</div>
      ) : (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th
                  className="text-left px-3 py-2 font-medium text-xs cursor-pointer hover:text-primary"
                  onClick={() => setSortBy('priority')}
                >
                  Prio {sortBy === 'priority' && '\u25BC'}
                </th>
                <th className="text-left px-3 py-2 font-medium text-xs">Title</th>
                <th
                  className="text-left px-3 py-2 font-medium text-xs cursor-pointer hover:text-primary"
                  onClick={() => setSortBy('status')}
                >
                  Status {sortBy === 'status' && '\u25BC'}
                </th>
                <th className="text-left px-3 py-2 font-medium text-xs">Assignee</th>
                <th className="text-left px-3 py-2 font-medium text-xs">Notes</th>
                <th
                  className="text-left px-3 py-2 font-medium text-xs cursor-pointer hover:text-primary"
                  onClick={() => setSortBy('updatedAt')}
                >
                  Updated {sortBy === 'updatedAt' && '\u25BC'}
                </th>
                <th className="px-3 py-2 w-8"></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(task => (
                <tr key={task.id} className="border-b last:border-b-0 hover:bg-muted/30">
                  <td className="px-3 py-2 text-xs font-mono">{task.priority}</td>
                  <td className="px-3 py-2">
                    <span className="text-sm">{task.title}</span>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={task.status}
                      onChange={e => void handleStatusChange(task.id, e.target.value)}
                      className="text-xs border rounded px-1 py-0.5 bg-background text-foreground"
                    >
                      {STATUS_OPTIONS.map(s => (
                        <option key={s} value={s}>{STATUS_ICONS[s]} {s}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    {task.assignee && (
                      <div className="flex items-center gap-1">
                        <AgentColorBadge appearance={getAgentAppearance(task.assignee)} size="xs" />
                        <span className="text-xs">{getAgentName(task.assignee)}</span>
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    {(task.notes?.length ?? 0) > 0 && (
                      <button
                        onClick={() => setExpandedTaskId(expandedTaskId === task.id ? null : task.id)}
                        className="text-xs text-primary flex items-center gap-0.5 hover:underline"
                      >
                        {task.notes!.length}
                        <ChevronDown className={cn('h-3 w-3 transition-transform', expandedTaskId === task.id && 'rotate-180')} />
                      </button>
                    )}
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">
                    {task.updatedAt ? formatRelativePastTime(new Date(task.updatedAt)) : '\u2014'}
                  </td>
                  <td className="px-3 py-2">
                    <button
                      onClick={() => void handleDeleteTask(task.id)}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {/* Expanded notes */}
          {expandedTaskId && (() => {
            const task = filtered.find(t => t.id === expandedTaskId)
            if (!task?.notes?.length) return null
            return (
              <div className="border-t px-4 py-2 bg-muted/20 space-y-1">
                <div className="text-xs font-medium text-muted-foreground mb-1">Notes for: {task.title}</div>
                {task.notes.map((note, i) => (
                  <div key={i} className="text-xs text-muted-foreground">
                    <span className="font-mono">{note.at ? formatRelativePastTime(new Date(note.at)) : ''}</span>
                    {note.by && <span className="ml-1">({note.by})</span>}
                    <span className="ml-1">&mdash; {note.text}</span>
                  </div>
                ))}
              </div>
            )
          })()}
        </div>
      )}
    </div>
  )
}
