/**
 * TaskKanbanColumn - A single status column in the Kanban board.
 * Columns are flush (no gaps, no border radius) to maximize space,
 * matching the ecosystem-manager layout.
 */

import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { cn } from '@/lib/utils'
import { TaskKanbanCard, type TaskKanbanCardProps } from './TaskKanbanCard'
import type { TeamTask } from '@/services/heartbeatService'

type TaskStatus = 'todo' | 'in-progress' | 'blocked' | 'done'

const STATUS_COLORS: Record<TaskStatus, { bg: string; border: string; header: string }> = {
  todo: {
    bg: 'bg-slate-50 dark:bg-slate-900/30',
    border: 'border-slate-200 dark:border-slate-700/50',
    header: 'bg-slate-100 text-slate-700 dark:bg-slate-800/50 dark:text-slate-300',
  },
  'in-progress': {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    border: 'border-blue-200 dark:border-blue-700/50',
    header: 'bg-blue-100 text-blue-800 dark:bg-blue-800/30 dark:text-blue-200',
  },
  blocked: {
    bg: 'bg-orange-50 dark:bg-orange-900/20',
    border: 'border-orange-200 dark:border-orange-700/50',
    header: 'bg-orange-100 text-orange-800 dark:bg-orange-800/30 dark:text-orange-200',
  },
  done: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    border: 'border-green-200 dark:border-green-700/50',
    header: 'bg-green-100 text-green-800 dark:bg-green-800/30 dark:text-green-200',
  },
}

const STATUS_LABELS: Record<TaskStatus, string> = {
  todo: 'To Do',
  'in-progress': 'In Progress',
  blocked: 'Blocked',
  done: 'Done',
}

interface TaskKanbanColumnProps {
  status: TaskStatus
  tasks: TeamTask[]
  members: TaskKanbanCardProps['members']
  allAgents: TaskKanbanCardProps['allAgents']
  onUpdateTask: TaskKanbanCardProps['onUpdateTask']
  onDeleteTask: TaskKanbanCardProps['onDeleteTask']
}

export function TaskKanbanColumn({
  status,
  tasks,
  members,
  allAgents,
  onUpdateTask,
  onDeleteTask,
}: TaskKanbanColumnProps) {
  const { setNodeRef, isOver } = useDroppable({
    id: status,
    data: { type: 'column', status },
  })

  const colors = STATUS_COLORS[status]
  const taskIds = tasks.map(t => t.id)

  return (
    <div className="shrink-0 w-80 flex flex-col h-full min-h-0">
      {/* Column Header — no border radius, flush with neighbors */}
      <div className={cn('px-4 py-3 border', colors.border, colors.header)}>
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold">{STATUS_LABELS[status]}</h2>
          <span className="text-xs px-2 py-0.5 rounded-full bg-foreground/10 text-foreground">
            {tasks.length}
          </span>
        </div>
      </div>

      {/* Column Content (Droppable Area) — no border radius */}
      <div
        ref={setNodeRef}
        className={cn(
          'kanban-column-body flex-1 min-h-0 overflow-y-auto',
          'px-3 py-3 border-x border-b',
          colors.bg, colors.border,
          isOver && 'ring-2 ring-blue-500/50 bg-blue-500/10 dark:bg-blue-500/10',
          'transition-all',
        )}
      >
        <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
          <div className="space-y-3">
            {tasks.length === 0 ? (
              <div className="text-center py-8 text-sm text-muted-foreground">
                No tasks
              </div>
            ) : (
              tasks.map(task => (
                <TaskKanbanCard
                  key={task.id}
                  task={task}
                  members={members}
                  allAgents={allAgents}
                  onUpdateTask={onUpdateTask}
                  onDeleteTask={onDeleteTask}
                />
              ))
            )}
          </div>
        </SortableContext>
      </div>
    </div>
  )
}
