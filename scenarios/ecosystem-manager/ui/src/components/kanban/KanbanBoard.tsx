/**
 * KanbanBoard Component
 * Main Kanban board with 7 status columns and drag-and-drop functionality
 */

import { useMemo } from 'react';
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core';
import { useState } from 'react';
import { KanbanColumn } from './KanbanColumn';
import { TaskCard } from './TaskCard';
import { SkeletonTaskCard } from './SkeletonTaskCard';
import { useTasks } from '../../hooks/useTasks';
import { useUpdateTaskStatus } from '../../hooks/useTaskMutations';
import { useTaskUpdates } from '../../hooks/useTaskUpdates';
import { useAppState } from '../../contexts/AppStateContext';
import type { Task, TaskStatus } from '../../types/api';

// Column definitions with display labels
const COLUMNS: Array<{ status: TaskStatus; title: string }> = [
  { status: 'pending', title: 'Pending' },
  { status: 'in-progress', title: 'Active' },
  { status: 'completed', title: 'Completed' },
  { status: 'completed-finalized', title: 'Finished' },
  { status: 'failed', title: 'Failed' },
  { status: 'failed-blocked', title: 'Blocked' },
  { status: 'archived', title: 'Archived' },
];

interface KanbanBoardProps {
  onViewTaskDetails?: (task: Task) => void;
  onDeleteTask?: (taskId: string) => void;
}

export function KanbanBoard({ onViewTaskDetails, onDeleteTask }: KanbanBoardProps) {
  const [activeTask, setActiveTask] = useState<Task | null>(null);

  const { columnVisibility, toggleColumnVisibility, filters } = useAppState();
  const { data: tasks = [], isLoading, error } = useTasks(filters);
  const updateTaskStatus = useUpdateTaskStatus();

  // Subscribe to WebSocket updates
  useTaskUpdates();

  // Set up drag sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // 8px movement required to start drag
      },
    })
  );

  // Group tasks by status
  const tasksByStatus = useMemo(() => {
    const grouped: Record<TaskStatus, Task[]> = {
      'pending': [],
      'in-progress': [],
      'completed': [],
      'completed-finalized': [],
      'failed': [],
      'failed-blocked': [],
      'archived': [],
      'review': [],
    };

    tasks.forEach(task => {
      if (grouped[task.status]) {
        grouped[task.status].push(task);
      }
    });

    return grouped;
  }, [tasks]);

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;
    const task = active.data.current?.task as Task | undefined;
    if (task) {
      setActiveTask(task);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveTask(null);

    if (!over) return;

    const taskId = active.id as string;
    const newStatus = over.id as TaskStatus;
    const task = tasks.find(t => t.id === taskId);

    // Only update if status actually changed
    if (task && task.status !== newStatus) {
      updateTaskStatus.mutate({ id: taskId, status: newStatus });
    }
  };

  if (isLoading) {
    return (
      <div className="flex h-full gap-0 overflow-x-auto">
        {COLUMNS.slice(0, 4).map(({ status, title }) => (
          <div key={status} className="flex-shrink-0 w-80">
            <div className="bg-slate-800/50 border border-white/10">
              {/* Column Header */}
              <div className="px-4 py-3 border-b border-white/10">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-sm text-slate-300">{title}</h3>
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 text-xs bg-slate-700 rounded-full">...</span>
                  </div>
                </div>
              </div>
              {/* Skeleton Cards */}
              <div className="p-3 space-y-3 max-h-[calc(100vh-300px)] overflow-y-auto">
                <SkeletonTaskCard />
                <SkeletonTaskCard />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-red-400">Error loading tasks: {error.message}</div>
      </div>
    );
  }

  return (
    <div className="h-full min-h-0 flex flex-col">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="flex h-full min-h-0 gap-0 overflow-x-auto">
          {COLUMNS.map(({ status, title }) => (
            <KanbanColumn
              key={status}
              status={status}
              title={title}
              tasks={tasksByStatus[status]}
              isVisible={columnVisibility[status]}
              onToggleVisibility={() => toggleColumnVisibility(status)}
              onViewDetails={onViewTaskDetails}
              onDeleteTask={onDeleteTask}
            />
          ))}
        </div>

        {/* Drag Overlay - Shows the task being dragged */}
        <DragOverlay>
          {activeTask ? (
            <div className="rotate-3">
              <TaskCard task={activeTask} />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}
