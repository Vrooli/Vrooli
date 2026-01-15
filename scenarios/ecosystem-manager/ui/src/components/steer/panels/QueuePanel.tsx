import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Check, GripVertical, ListOrdered, X } from 'lucide-react';
import { PhasePicker } from '../PhasePicker';
import { formatPhaseName } from '@/lib/utils';
import type { PhaseInfo } from '@/types/api';

type ItemStatus = 'completed' | 'current' | 'pending';

interface QueuePanelProps {
  value: string[];
  onChange: (queue: string[]) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
  /** Current execution position (0-indexed) */
  currentIndex?: number;
  /** Whether the queue is fully processed */
  isExhausted?: boolean;
  /** Disables editing controls when true */
  readOnly?: boolean;
}

interface SortableQueueItemProps {
  id: string;
  mode: string;
  index: number;
  onRemove: () => void;
  status?: ItemStatus;
}

interface ReadOnlyQueueItemProps {
  mode: string;
  index: number;
  status: ItemStatus;
}

function getStatusStyles(status: ItemStatus): string {
  switch (status) {
    case 'completed':
      return 'bg-slate-800/30 border-slate-700/50 opacity-60';
    case 'current':
      return 'bg-cyan-500/10 border-cyan-500/40 ring-2 ring-cyan-500/50';
    case 'pending':
    default:
      return 'bg-cyan-500/5 border-cyan-500/20';
  }
}

function ReadOnlyQueueItem({ mode, index, status }: ReadOnlyQueueItemProps) {
  return (
    <div
      className={`
        flex items-center gap-2 px-3 py-2 rounded-md border
        ${getStatusStyles(status)}
      `}
    >
      {status === 'completed' ? (
        <div className="flex h-4 w-4 items-center justify-center">
          <Check className="h-3.5 w-3.5 text-green-500" />
        </div>
      ) : status === 'current' ? (
        <div className="flex h-4 w-4 items-center justify-center">
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-cyan-500"></span>
          </span>
        </div>
      ) : (
        <div className="w-4" />
      )}
      <span className="text-xs text-slate-500 font-mono w-5">{index + 1}.</span>
      <span
        className={`flex-1 text-sm ${
          status === 'completed'
            ? 'text-slate-400 line-through'
            : status === 'current'
              ? 'text-cyan-100 font-medium'
              : 'text-cyan-200'
        }`}
      >
        {formatPhaseName(mode)}
      </span>
    </div>
  );
}

function SortableQueueItem({ id, mode, index, onRemove, status }: SortableQueueItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  // Combine base styles with status-specific styles
  const getContainerStyles = () => {
    const base = 'flex items-center gap-2 px-3 py-2 rounded-md border';
    const drag = isDragging ? 'opacity-50 shadow-lg ring-2 ring-cyan-500/50' : '';

    if (!status || status === 'pending') {
      return `${base} bg-cyan-500/5 border-cyan-500/20 ${drag}`;
    }
    if (status === 'completed') {
      return `${base} bg-slate-800/30 border-slate-700/50 opacity-60 ${drag}`;
    }
    if (status === 'current') {
      return `${base} bg-cyan-500/10 border-cyan-500/40 ring-2 ring-cyan-500/50 ${drag}`;
    }
    return `${base} bg-cyan-500/5 border-cyan-500/20 ${drag}`;
  };

  return (
    <div ref={setNodeRef} style={style} className={getContainerStyles()}>
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-300 touch-none"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      {/* Status indicator */}
      {status === 'completed' ? (
        <div className="flex h-4 w-4 items-center justify-center">
          <Check className="h-3.5 w-3.5 text-green-500" />
        </div>
      ) : status === 'current' ? (
        <div className="flex h-4 w-4 items-center justify-center">
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-cyan-500"></span>
          </span>
        </div>
      ) : null}
      <span className="text-xs text-slate-500 font-mono w-5">{index + 1}.</span>
      <span
        className={`flex-1 text-sm ${
          status === 'completed'
            ? 'text-slate-400 line-through'
            : status === 'current'
              ? 'text-cyan-100 font-medium'
              : 'text-cyan-200'
        }`}
      >
        {formatPhaseName(mode)}
      </span>
      <button
        type="button"
        onClick={onRemove}
        className="p-1 rounded hover:bg-red-500/20 text-slate-400 hover:text-red-400 transition-colors"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

export function QueuePanel({
  value,
  onChange,
  phaseNames,
  isLoading,
  currentIndex,
  isExhausted,
  readOnly,
}: QueuePanelProps) {
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Generate unique IDs for each queue item (mode + index for uniqueness)
  const itemIds = value.map((mode, idx) => `${mode}-${idx}`);

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = itemIds.indexOf(active.id as string);
      const newIndex = itemIds.indexOf(over.id as string);

      if (oldIndex !== -1 && newIndex !== -1) {
        onChange(arrayMove(value, oldIndex, newIndex));
      }
    }
  };

  const handleAddMode = (mode: string | undefined) => {
    if (mode) {
      onChange([...value, mode]);
    }
  };

  const handleRemove = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  const getItemStatus = (index: number): ItemStatus => {
    if (isExhausted) return 'completed';
    if (currentIndex === undefined) return 'pending';
    if (index < currentIndex) return 'completed';
    if (index === currentIndex) return 'current';
    return 'pending';
  };

  return (
    <div className="space-y-4">
      {!readOnly && (
        <div className="flex items-start gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-cyan-500/10 shrink-0">
            <ListOrdered className="h-5 w-5 text-cyan-400" />
          </div>
          <div>
            <h3 className="text-sm font-medium text-slate-200">Steering Queue</h3>
            <p className="text-sm text-slate-400 mt-0.5">
              Build an ordered list of focus modes. Each mode runs once in sequence, then the task
              completes.
            </p>
          </div>
        </div>
      )}

      {/* Add mode controls - only in edit mode */}
      {!readOnly && (
        <PhasePicker
          value={undefined}
          onChange={handleAddMode}
          phaseNames={phaseNames}
          isLoading={isLoading}
          placeholder="Add a phase to queue..."
          dialogTitle="Add Phase to Queue"
          dialogDescription="Select a phase to add to the steering queue."
        />
      )}

      {/* Queue list */}
      <div className={readOnly ? '' : 'min-h-[120px]'}>
        {value.length === 0 ? (
          !readOnly && (
            <div className="flex flex-col items-center justify-center py-8 border border-dashed border-slate-700 rounded-md">
              <ListOrdered className="h-8 w-8 text-slate-600 mb-2" />
              <p className="text-sm text-slate-500">Add modes to build your queue</p>
            </div>
          )
        ) : readOnly ? (
          // Read-only mode: simple list with status indicators
          <div className="space-y-2">
            {value.map((mode, index) => (
              <ReadOnlyQueueItem
                key={`${mode}-${index}`}
                mode={mode}
                index={index}
                status={getItemStatus(index)}
              />
            ))}
          </div>
        ) : (
          // Edit mode: drag-and-drop sortable list
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={itemIds} strategy={verticalListSortingStrategy}>
              <div className="space-y-2">
                {value.map((mode, index) => (
                  <SortableQueueItem
                    key={itemIds[index]}
                    id={itemIds[index]}
                    mode={mode}
                    index={index}
                    onRemove={() => handleRemove(index)}
                    status={getItemStatus(index)}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </div>

      {/* Preview - only in edit mode */}
      {!readOnly && value.length > 0 && (
        <div className="text-xs text-slate-500 border-t border-slate-700/50 pt-3">
          <span className="font-medium text-slate-400">Order: </span>
          {value.map((mode, idx) => (
            <span key={idx}>
              <span className="text-cyan-400">{mode}</span>
              {idx < value.length - 1 && <span className="text-slate-600"> → </span>}
            </span>
          ))}
        </div>
      )}

      {/* Progress summary - show when there's progress data */}
      {currentIndex !== undefined && (
        <div className="text-xs text-slate-500 border-t border-slate-700/50 pt-3">
          {isExhausted ? (
            <span className="text-green-400">Queue completed</span>
          ) : (
            <span>
              Progress: <span className="text-cyan-400">{currentIndex + 1}</span> of{' '}
              <span className="text-cyan-400">{value.length}</span>
            </span>
          )}
        </div>
      )}
    </div>
  );
}
