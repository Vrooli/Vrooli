/**
 * TaskCard Component
 * Main task card container with drag-and-drop support
 */

import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { TaskCardHeader } from './TaskCardHeader';
import { TaskCardBody } from './TaskCardBody';
import { TaskCardFooter } from './TaskCardFooter';
import type { Task } from '../../types/api';

interface TaskCardProps {
  task: Task;
  onViewDetails?: (task: Task) => void;
  onDelete?: (taskId: string) => void;
}

export function TaskCard({ task, onViewDetails, onDelete }: TaskCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: task.id,
    data: {
      type: 'task',
      task,
    },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`
        bg-card border border-border rounded-lg p-3
        hover:border-primary/30 hover:bg-muted/60 transition-all
        cursor-grab active:cursor-grabbing
        ${isDragging ? 'shadow-xl ring-2 ring-primary/40' : ''}
      `}
    >
      <TaskCardHeader task={task} />
      <TaskCardBody task={task} />
      <TaskCardFooter
        task={task}
        onViewDetails={onViewDetails ? () => onViewDetails(task) : undefined}
        onDelete={onDelete ? () => onDelete(task.id) : undefined}
      />
    </div>
  );
}
