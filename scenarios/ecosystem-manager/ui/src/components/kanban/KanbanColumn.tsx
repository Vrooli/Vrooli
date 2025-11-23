/**
 * KanbanColumn Component
 * Displays a column for a specific task status with droppable area
 */

import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { Eye } from 'lucide-react';
import { TaskCard } from './TaskCard';
import type { Task, TaskStatus } from '../../types/api';

interface KanbanColumnProps {
  status: TaskStatus;
  title: string;
  tasks: Task[];
  isVisible: boolean;
  onToggleVisibility: () => void;
  onViewDetails?: (task: Task) => void;
  onDeleteTask?: (taskId: string) => void;
}

const STATUS_COLORS: Record<TaskStatus, { bg: string; border: string; header: string }> = {
  'pending': {
    bg: 'bg-slate-50 dark:bg-slate-900/30',
    border: 'border-slate-200 dark:border-slate-700/50',
    header: 'bg-slate-100 text-slate-700 dark:bg-slate-800/50 dark:text-slate-300',
  },
  'in-progress': {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    border: 'border-blue-200 dark:border-blue-700/50',
    header: 'bg-blue-100 text-blue-800 dark:bg-blue-800/30 dark:text-blue-200',
  },
  'completed': {
    bg: 'bg-green-50 dark:bg-green-900/20',
    border: 'border-green-200 dark:border-green-700/50',
    header: 'bg-green-100 text-green-800 dark:bg-green-800/30 dark:text-green-200',
  },
  'completed-finalized': {
    bg: 'bg-emerald-50 dark:bg-emerald-900/20',
    border: 'border-emerald-200 dark:border-emerald-700/50',
    header: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-800/30 dark:text-emerald-200',
  },
  'failed': {
    bg: 'bg-red-50 dark:bg-red-900/20',
    border: 'border-red-200 dark:border-red-700/50',
    header: 'bg-red-100 text-red-800 dark:bg-red-800/30 dark:text-red-200',
  },
  'failed-blocked': {
    bg: 'bg-orange-50 dark:bg-orange-900/20',
    border: 'border-orange-200 dark:border-orange-700/50',
    header: 'bg-orange-100 text-orange-800 dark:bg-orange-800/30 dark:text-orange-200',
  },
  'archived': {
    bg: 'bg-slate-50 dark:bg-slate-900/40',
    border: 'border-slate-200 dark:border-slate-600/50',
    header: 'bg-slate-100 text-slate-700 dark:bg-slate-700/30 dark:text-slate-400',
  },
  'review': {
    bg: 'bg-purple-50 dark:bg-purple-900/20',
    border: 'border-purple-200 dark:border-purple-700/50',
    header: 'bg-purple-100 text-purple-800 dark:bg-purple-800/30 dark:text-purple-200',
  },
};

export function KanbanColumn({
  status,
  title,
  tasks,
  isVisible,
  onToggleVisibility,
  onViewDetails,
  onDeleteTask,
}: KanbanColumnProps) {
  const { setNodeRef, isOver } = useDroppable({
    id: status,
    data: {
      type: 'column',
      status,
    },
  });

  const colors = STATUS_COLORS[status];
  const taskIds = tasks.map(t => t.id);

  if (!isVisible) {
    return null;
  }

  return (
    <div className="shrink-0 w-80 flex flex-col h-screen max-h-screen min-h-0">
      {/* Column Header */}
      <div className={`px-4 py-3 border ${colors.border} ${colors.header}`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">{title}</h2>
            <span className="text-xs px-2 py-0.5 rounded-full bg-foreground/10 text-foreground">
              {tasks.length}
            </span>
          </div>
          <button
            onClick={onToggleVisibility}
            className="p-1 rounded hover:bg-foreground/10 text-muted-foreground transition-colors"
            title="Hide column"
          >
            <Eye className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Column Content (Droppable Area) */}
      <div
        ref={setNodeRef}
        className={`
          kanban-column-body flex-1 min-h-0 overflow-y-auto
          px-3 py-3 border-x border-b
          ${colors.bg} ${colors.border}
          ${isOver ? 'ring-2 ring-blue-500/50 bg-blue-500/10 dark:bg-blue-500/10' : ''}
          transition-all
        `}
      >
        <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
          <div className="space-y-3">
            {tasks.length === 0 ? (
              <div className="text-center py-8 text-sm text-muted-foreground">
                No tasks
              </div>
            ) : (
              tasks.map(task => (
                <TaskCard
                  key={task.id}
                  task={task}
                  onViewDetails={onViewDetails}
                  onDelete={onDeleteTask}
                />
              ))
            )}
          </div>
        </SortableContext>
      </div>
    </div>
  );
}
