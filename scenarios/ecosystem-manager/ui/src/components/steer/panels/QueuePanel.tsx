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
import { GripVertical, ListOrdered, X } from 'lucide-react';
import { PhasePicker } from '../PhasePicker';
import { formatPhaseName } from '@/lib/utils';
import type { PhaseInfo } from '@/types/api';

interface QueuePanelProps {
  value: string[];
  onChange: (queue: string[]) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
}

interface SortableQueueItemProps {
  id: string;
  mode: string;
  index: number;
  onRemove: () => void;
}

function SortableQueueItem({ id, mode, index, onRemove }: SortableQueueItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`
        flex items-center gap-2 px-3 py-2 rounded-md border
        bg-cyan-500/5 border-cyan-500/20
        ${isDragging ? 'opacity-50 shadow-lg ring-2 ring-cyan-500/50' : ''}
      `}
    >
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-300 touch-none"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      <span className="text-xs text-slate-500 font-mono w-5">{index + 1}.</span>
      <span className="flex-1 text-sm text-cyan-200">{formatPhaseName(mode)}</span>
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

export function QueuePanel({ value, onChange, phaseNames, isLoading }: QueuePanelProps) {
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

  return (
    <div className="space-y-4">
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

      {/* Add mode controls */}
      <PhasePicker
        value={undefined}
        onChange={handleAddMode}
        phaseNames={phaseNames}
        isLoading={isLoading}
        placeholder="Add a phase to queue..."
        dialogTitle="Add Phase to Queue"
        dialogDescription="Select a phase to add to the steering queue."
      />

      {/* Queue list with drag-and-drop */}
      <div className="min-h-[120px]">
        {value.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 border border-dashed border-slate-700 rounded-md">
            <ListOrdered className="h-8 w-8 text-slate-600 mb-2" />
            <p className="text-sm text-slate-500">Add modes to build your queue</p>
          </div>
        ) : (
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
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </div>

      {/* Preview */}
      {value.length > 0 && (
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
    </div>
  );
}
