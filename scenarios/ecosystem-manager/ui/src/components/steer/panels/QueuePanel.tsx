import { useMemo, useState } from 'react';
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
import { Check, ChevronDown, GripVertical, ListOrdered, Loader2, X } from 'lucide-react';
import { SkillPicker } from '../SkillPicker';
import { getQueueStepDisplay } from '@/lib/utils';
import type { SkillInfo } from '@/types/api';

type ItemStatus = 'completed' | 'current' | 'pending';

interface QueuePanelProps {
  value: string[][];
  onChange: (queue: string[][]) => void;
  skillNames: SkillInfo[];
  isLoading?: boolean;
  currentIndex?: number;
  isExhausted?: boolean;
  readOnly?: boolean;
  onPositionChange?: (position: number) => void;
  pendingPosition?: number | null;
}

interface SortableQueueItemProps {
  id: string;
  step: string[];
  index: number;
  skillNames: SkillInfo[];
  onRemove: () => void;
  onClick?: () => void;
  status?: ItemStatus;
  isPending?: boolean;
  readOnly?: boolean;
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

function QueueItemContent({
  step,
  index,
  skillNames,
  status = 'pending',
  isPending,
  onClick,
}: {
  step: string[];
  index: number;
  skillNames: SkillInfo[];
  status?: ItemStatus;
  isPending?: boolean;
  onClick?: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const summary = getQueueStepDisplay(step, skillNames);
  const isClickable = !!onClick && !isPending;

  return (
    <div className="flex-1 min-w-0">
      <div
        className={isClickable ? 'cursor-pointer' : undefined}
        onClick={isClickable ? onClick : undefined}
        role={isClickable ? 'button' : undefined}
      >
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500 font-mono w-5">{index + 1}.</span>
          <span
            className={`flex-1 text-sm ${
              isPending
                ? 'text-amber-200 font-medium'
                : status === 'completed'
                  ? 'text-slate-400 line-through'
                  : status === 'current'
                    ? 'text-cyan-100 font-medium'
                    : 'text-cyan-200'
            }`}
            title={summary.tooltip}
          >
            {summary.label}
          </span>
          {step.length > 1 && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setExpanded((v) => !v);
              }}
              className="text-slate-500 hover:text-slate-300"
              title={expanded ? 'Collapse set' : 'Expand set'}
            >
              <ChevronDown className={`h-3.5 w-3.5 transition-transform ${expanded ? 'rotate-180' : ''}`} />
            </button>
          )}
        </div>
      </div>

      {expanded && (
        <div className="mt-2 ml-7 flex flex-wrap gap-1.5">
          {step.map((skillId, idx) => (
            <span
              key={`${skillId}-${idx}`}
              className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-cyan-500/10 text-cyan-200 border border-cyan-500/20"
            >
              {getQueueStepDisplay([skillId], skillNames).label}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function SortableQueueItem({
  id,
  step,
  index,
  skillNames,
  onRemove,
  onClick,
  status,
  isPending,
  readOnly,
}: SortableQueueItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  const style = { transform: CSS.Transform.toString(transform), transition };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-start gap-2 px-3 py-2 rounded-md border ${
        isPending
          ? 'ring-2 ring-amber-500/50 bg-amber-500/10 border-amber-500/40'
          : getStatusStyles(status ?? 'pending')
      } ${isDragging ? 'opacity-50 shadow-lg ring-2 ring-cyan-500/50' : ''}`}
    >
      {!readOnly && (
        <button
          type="button"
          className="cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-300 touch-none mt-0.5"
          {...attributes}
          {...listeners}
        >
          <GripVertical className="h-4 w-4" />
        </button>
      )}

      {isPending ? (
        <div className="flex h-4 w-4 items-center justify-center mt-0.5">
          <Loader2 className="h-3.5 w-3.5 text-amber-400 animate-spin" />
        </div>
      ) : status === 'completed' ? (
        <div className="flex h-4 w-4 items-center justify-center mt-0.5">
          <Check className="h-3.5 w-3.5 text-green-500" />
        </div>
      ) : status === 'current' ? (
        <div className="flex h-4 w-4 items-center justify-center mt-0.5">
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-cyan-500" />
          </span>
        </div>
      ) : (
        <div className="w-4 mt-0.5" />
      )}

      <QueueItemContent
        step={step}
        index={index}
        skillNames={skillNames}
        status={status}
        isPending={isPending}
        onClick={onClick}
      />

      {!readOnly && (
        <button
          type="button"
          onClick={onRemove}
          className="p-1 rounded hover:bg-red-500/20 text-slate-400 hover:text-red-400 transition-colors"
          title="Remove step"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}

export function QueuePanel({
  value,
  onChange,
  skillNames,
  isLoading,
  currentIndex,
  isExhausted,
  readOnly,
  onPositionChange,
  pendingPosition,
}: QueuePanelProps) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const itemIds = useMemo(
    () => value.map((step, idx) => `${step.join('|') || 'empty'}-${idx}`),
    [value]
  );

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

  const handleAddSet = (set: string[]) => {
    if (set.length > 0) {
      onChange([...value, set]);
    }
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
              Build an ordered list of steering skill sets. Each set runs once in sequence.
            </p>
          </div>
        </div>
      )}

      {!readOnly && (
        <SkillPicker
          values={[]}
          onChange={handleAddSet}
          skillNames={skillNames}
          isLoading={isLoading}
          selectionMode="multiple"
          placeholder="Add a skill set to queue..."
          dialogTitle="Add Queue Step"
          dialogDescription="Select one or more skills for this queue step."
          confirmLabel="Add Step"
        />
      )}

      <div className={readOnly ? '' : 'min-h-[120px]'}>
        {value.length === 0 ? (
          !readOnly && (
            <div className="flex flex-col items-center justify-center py-8 border border-dashed border-slate-700 rounded-md">
              <ListOrdered className="h-8 w-8 text-slate-600 mb-2" />
              <p className="text-sm text-slate-500">Add skill sets to build your queue</p>
            </div>
          )
        ) : (
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={itemIds} strategy={verticalListSortingStrategy}>
              <div className="space-y-2">
                {value.map((step, index) => (
                  <SortableQueueItem
                    key={itemIds[index]}
                    id={itemIds[index]}
                    step={step}
                    index={index}
                    skillNames={skillNames}
                    onRemove={() => onChange(value.filter((_, i) => i !== index))}
                    status={getItemStatus(index)}
                    onClick={onPositionChange ? () => onPositionChange(index) : undefined}
                    isPending={pendingPosition === index}
                    readOnly={readOnly}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </div>

      {!readOnly && value.length > 0 && (
        <div className="text-xs text-slate-500 border-t border-slate-700/50 pt-3">
          <span className="font-medium text-slate-400">Order: </span>
          {value.map((step, idx) => {
            const summary = getQueueStepDisplay(step, skillNames);
            return (
              <span key={idx}>
                <span className="text-cyan-400" title={summary.tooltip}>{summary.label}</span>
                {idx < value.length - 1 && <span className="text-slate-600"> → </span>}
              </span>
            );
          })}
        </div>
      )}

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
