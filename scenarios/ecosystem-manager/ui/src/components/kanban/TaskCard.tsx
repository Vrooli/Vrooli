/**
 * TaskCard Component
 * Main task card container with drag-and-drop support
 */

import { useRef, type PointerEvent as ReactPointerEvent } from 'react';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { TaskCardHeader } from './TaskCardHeader';
import { TaskCardBody } from './TaskCardBody';
import { TaskCardFooter } from './TaskCardFooter';
import type { Task, AutoSteerProfile } from '../../types/api';

interface TaskCardProps {
  task: Task;
  onViewDetails?: (task: Task) => void;
  onDelete?: (task: Task) => void;
  autoSteerProfile?: AutoSteerProfile;
  autoSteerPhaseIndex?: number;
  dragOverlay?: boolean;
}

export function TaskCard({
  task,
  onViewDetails,
  onDelete,
  autoSteerProfile,
  autoSteerPhaseIndex,
  dragOverlay = false,
}: TaskCardProps) {
  if (dragOverlay) {
    return (
      <div
        className="
          bg-card border border-border rounded-lg p-3
          shadow-xl ring-2 ring-primary/40 rotate-3
        "
      >
        <TaskCardHeader task={task} />
        <TaskCardBody
          task={task}
          autoSteerProfile={autoSteerProfile}
          autoSteerPhaseIndex={autoSteerPhaseIndex}
        />
        <TaskCardFooter task={task} />
      </div>
    );
  }

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

  const pointerStartRef = useRef<{ x: number; y: number } | null>(null);
  const dragIntentRef = useRef(false);

  const { onPointerDown, onPointerMove, onPointerUp, ...restListeners } = listeners ?? {};

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const handlePointerDown = (event: ReactPointerEvent) => {
    pointerStartRef.current = { x: event.clientX, y: event.clientY };
    dragIntentRef.current = false;
    onPointerDown?.(event);
  };

  const handlePointerMove = (event: ReactPointerEvent) => {
    if (pointerStartRef.current && !dragIntentRef.current) {
      const dx = event.clientX - pointerStartRef.current.x;
      const dy = event.clientY - pointerStartRef.current.y;
      if (Math.hypot(dx, dy) > 6) {
        dragIntentRef.current = true;
      }
    }
    onPointerMove?.(event);
  };

  const handlePointerUp = (event: ReactPointerEvent) => {
    onPointerUp?.(event);
    pointerStartRef.current = null;
  };

  const handleClick = () => {
    if (!dragIntentRef.current && !isDragging) {
      onViewDetails?.(task);
    }
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...restListeners}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerLeave={() => {
        pointerStartRef.current = null;
      }}
      onClick={handleClick}
      className={`
        bg-card border border-border rounded-lg p-3
        hover:border-primary/30 hover:bg-muted/60 transition-all
        cursor-grab active:cursor-grabbing
        ${isDragging ? 'shadow-xl ring-2 ring-primary/40' : ''}
      `}
    >
      <TaskCardHeader task={task} />
      <TaskCardBody
        task={task}
        autoSteerProfile={autoSteerProfile}
        autoSteerPhaseIndex={autoSteerPhaseIndex}
      />
      <TaskCardFooter
        task={task}
        onViewDetails={onViewDetails ? () => onViewDetails(task) : undefined}
        onDelete={onDelete ? () => onDelete(task) : undefined}
      />
    </div>
  );
}
