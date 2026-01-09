/**
 * KanbanBoard Component
 * Main Kanban board with 7 status columns and drag-and-drop functionality
 */

import { useEffect, useMemo, useRef, useState, useCallback } from 'react';
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
} from '@dnd-kit/core';
import { KanbanColumn } from './KanbanColumn';
import { TaskCard } from './TaskCard';
import { SkeletonTaskCard } from './SkeletonTaskCard';
import { useTasks } from '../../hooks/useTasks';
import { useUpdateTaskStatus } from '../../hooks/useTaskMutations';
import { useTaskUpdates } from '../../hooks/useTaskUpdates';
import { useAutoSteerProfiles } from '../../hooks/useAutoSteer';
import { useAppState } from '../../contexts/AppStateContext';
import type { AutoSteerProfile, Task, TaskStatus } from '../../types/api';

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
  onDeleteTask?: (task: Task) => void;
}

export function KanbanBoard({ onViewTaskDetails, onDeleteTask }: KanbanBoardProps) {
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const kanbanGridRef = useRef<HTMLDivElement | null>(null);
  const scrollLockRef = useRef<{ timeout: number | null }>({ timeout: null });

  const { columnVisibility, toggleColumnVisibility, filters } = useAppState();
  const { data: tasks = [], isLoading, error } = useTasks(filters);
  const updateTaskStatus = useUpdateTaskStatus();
  const { data: autoSteerProfiles = [] } = useAutoSteerProfiles();

  const autoSteerProfileMap = useMemo<Record<string, AutoSteerProfile>>(() => {
    const map: Record<string, AutoSteerProfile> = {};
    autoSteerProfiles.forEach(profile => {
      map[profile.id] = profile;
    });
    return map;
  }, [autoSteerProfiles]);

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

  // Prefer pointer collisions to avoid snapping back to the source column
  const collisionDetection = useCallback((args: Parameters<typeof pointerWithin>[0]) => {
    const intersections = pointerWithin(args);
    if (intersections.length > 0) {
      return intersections;
    }
    return closestCenter(args);
  }, []);

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
    const overType = over.data.current?.type;
    let newStatus: TaskStatus | null = null;

    if (overType === 'column') {
      newStatus = over.data.current?.status as TaskStatus | null;
    } else if (overType === 'task') {
      // Dropped onto another task - infer status from that task
      const overTask = over.data.current?.task as Task | undefined;
      newStatus = overTask?.status ?? null;
      if (!newStatus) {
        const targetTask = tasks.find(t => t.id === over.id);
        newStatus = targetTask?.status ?? null;
      }
    } else if (typeof over.id === 'string') {
      // Fallback: if over.id matches a column status
      const maybeStatus = over.id as TaskStatus;
      if (COLUMNS.some(col => col.status === maybeStatus)) {
        newStatus = maybeStatus;
      }
    }

    if (!newStatus) return;

    const task = tasks.find(t => t.id === taskId);

    // Only update if status actually changed
    if (task && task.status !== newStatus) {
      updateTaskStatus.mutate({ id: taskId, status: newStatus });
    }
  };

  // Match legacy scroll behavior: convert vertical wheel scroll to horizontal unless a column can scroll vertically
  useEffect(() => {
    const kanbanGrid = kanbanGridRef.current;
    if (!kanbanGrid) return;

    const handleWheel = (event: WheelEvent) => {
      if (event.defaultPrevented) return;

      const scrollLock = scrollLockRef.current;
      const horizontalLockActive = scrollLock.timeout !== null;

      const columnBody = (event.target as HTMLElement | null)?.closest('.kanban-column-body') as HTMLElement | null;
      if (columnBody && !horizontalLockActive) {
        const canScrollVertically = columnBody.scrollHeight > columnBody.clientHeight;
        if (canScrollVertically) {
          const scrollTop = columnBody.scrollTop;
          const atTop = scrollTop <= 0;
          const atBottom = (columnBody.scrollHeight - columnBody.clientHeight - scrollTop) <= 1;
          if ((event.deltaY < 0 && !atTop) || (event.deltaY > 0 && !atBottom)) {
            return;
          }
        }
      }

      if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
        event.preventDefault();
        kanbanGrid.scrollLeft += event.deltaY;

        if (scrollLock.timeout !== null) {
          clearTimeout(scrollLock.timeout);
        }
        scrollLock.timeout = window.setTimeout(() => {
          scrollLock.timeout = null;
        }, 250);
      }
    };

    kanbanGrid.addEventListener('wheel', handleWheel, { passive: false });
    return () => kanbanGrid.removeEventListener('wheel', handleWheel);
  }, [isLoading]);

  if (isLoading) {
    return (
      <div className="flex h-full min-h-0 gap-0 overflow-x-auto">
        {COLUMNS.slice(0, 4).map(({ status, title }) => (
          <div key={status} className="flex-shrink-0 w-80 h-screen max-h-screen">
            <div className="bg-card/80 border border-border/60 rounded-lg overflow-hidden flex flex-col h-full">
              {/* Column Header */}
              <div className="px-4 py-3 border-b border-border/60 bg-muted/40">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-sm text-muted-foreground">{title}</h3>
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 text-xs bg-muted rounded-full text-muted-foreground">...</span>
                  </div>
                </div>
              </div>
              {/* Skeleton Cards */}
              <div className="p-3 space-y-3 flex-1 min-h-0 overflow-y-auto">
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
        <div className="text-destructive">Error loading tasks: {error.message}</div>
      </div>
    );
  }

  return (
    <div className="h-full min-h-0 flex flex-col overflow-hidden">
      <DndContext
        sensors={sensors}
        collisionDetection={collisionDetection}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
      <div ref={kanbanGridRef} className="flex h-full min-h-0 gap-0 overflow-x-auto">
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
              autoSteerProfilesById={autoSteerProfileMap}
            />
          ))}
        </div>

        {/* Drag Overlay - Shows the task being dragged */}
        <DragOverlay dropAnimation={null}>
          {activeTask ? (
            <TaskCard
              task={activeTask}
              dragOverlay
              autoSteerProfile={
                activeTask.auto_steer_profile_id
                  ? autoSteerProfileMap[activeTask.auto_steer_profile_id]
                  : undefined
              }
              autoSteerPhaseIndex={activeTask.auto_steer_phase_index}
            />
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}
