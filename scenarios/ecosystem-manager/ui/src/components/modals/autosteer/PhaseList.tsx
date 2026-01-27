/**
 * PhaseList Component
 * Manage the list of phases in an Auto Steer profile
 */

import { useEffect, useMemo, useState } from 'react';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
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
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { AutoSteerPhase } from '@/types/api';
import { PhaseEditor } from './PhaseEditor';

interface PhaseListProps {
  phases: AutoSteerPhase[];
  onChange: (phases: AutoSteerPhase[]) => void;
}

interface SortablePhaseItemProps {
  id: string;
  phase: AutoSteerPhase;
  index: number;
  onChange: (phase: AutoSteerPhase) => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDuplicate: () => void;
  isFirst: boolean;
  isLast: boolean;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

function SortablePhaseItem({
  id,
  phase,
  index,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
  onDuplicate,
  isFirst,
  isLast,
  isCollapsed,
  onToggleCollapse,
}: SortablePhaseItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const style = {
    transform: transform ? CSS.Transform.toString({ ...transform, x: 0 }) : undefined,
    transition,
  };

  return (
    <div ref={setNodeRef} style={style}>
      <PhaseEditor
        phase={phase}
        index={index}
        onChange={onChange}
        onRemove={onRemove}
        onMoveUp={onMoveUp}
        onMoveDown={onMoveDown}
        onDuplicate={onDuplicate}
        isFirst={isFirst}
        isLast={isLast}
        isCollapsed={isCollapsed}
        onToggleCollapse={onToggleCollapse}
        dragHandleProps={{ ...attributes, ...listeners }}
        isDragging={isDragging}
      />
    </div>
  );
}

export function PhaseList({ phases, onChange }: PhaseListProps) {
  const [collapsedById, setCollapsedById] = useState<Record<string, boolean>>({});

  const phaseIds = useMemo(
    () => phases.map((phase, index) => phase.id ?? `phase-${index}`),
    [phases]
  );

  useEffect(() => {
    if (phases.length === 0) return;
    let needsUpdate = false;
    const normalized = phases.map((phase, index) => {
      if (phase.id) return phase;
      needsUpdate = true;
      return { ...phase, id: `phase-${Date.now()}-${index}` };
    });
    if (needsUpdate) {
      onChange(normalized);
    }
  }, [phases, onChange]);

  useEffect(() => {
    if (phases.length === 0) return;
    setCollapsedById((prev) => {
      let changed = false;
      const next = { ...prev };
      const currentIds = new Set(phaseIds);

      phaseIds.forEach((id) => {
        if (next[id] === undefined) {
          next[id] = true;
          changed = true;
        }
      });

      Object.keys(next).forEach((id) => {
        if (!currentIds.has(id)) {
          delete next[id];
          changed = true;
        }
      });

      return changed ? next : prev;
    });
  }, [phaseIds, phases.length]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = phaseIds.indexOf(active.id as string);
    const newIndex = phaseIds.indexOf(over.id as string);
    if (oldIndex === -1 || newIndex === -1) return;

    onChange(arrayMove(phases, oldIndex, newIndex));
  };

  const handleAddPhase = () => {
    const newPhaseId = `phase-${Date.now()}-${phases.length}`;
    const newPhase: AutoSteerPhase = {
      id: newPhaseId,
      skill_id: 'progress',
      skill_name: 'Progress',
      modes: [],
      max_iterations: 10,
      description: '',
      stop_conditions: [],
    };
    onChange([...phases, newPhase]);
    setCollapsedById(() => {
      const next: Record<string, boolean> = {};
      phaseIds.forEach((id) => {
        next[id] = true;
      });
      next[newPhaseId] = false;
      return next;
    });
  };

  const handleUpdatePhase = (index: number, updatedPhase: AutoSteerPhase) => {
    const newPhases = [...phases];
    newPhases[index] = updatedPhase;
    onChange(newPhases);
  };

  const handleRemovePhase = (index: number) => {
    onChange(phases.filter((_, i) => i !== index));
  };

  const handleDuplicatePhase = (index: number) => {
    const phaseToCopy = phases[index];
    if (!phaseToCopy) return;
    const copyId = `phase-${Date.now()}-${index}-copy`;
    const copy: AutoSteerPhase = {
      ...JSON.parse(JSON.stringify(phaseToCopy)),
      id: copyId,
    };
    onChange([...phases.slice(0, index + 1), copy, ...phases.slice(index + 1)]);
    setCollapsedById(() => {
      const next: Record<string, boolean> = {};
      phaseIds.forEach((id) => {
        next[id] = true;
      });
      next[copyId] = false;
      return next;
    });
  };

  const handleMovePhase = (from: number, to: number) => {
    if (to < 0 || to >= phases.length) return;
    const newPhases = [...phases];
    const [moved] = newPhases.splice(from, 1);
    newPhases.splice(to, 0, moved);
    onChange(newPhases);
  };

  const handleToggleCollapse = (id: string) => {
    setCollapsedById((prev) => {
      const isCollapsed = prev[id] ?? true;
      const next: Record<string, boolean> = {};
      phaseIds.forEach((phaseId) => {
        next[phaseId] = phaseId === id ? !isCollapsed : true;
      });
      return next;
    });
  };

  const handleCollapseAll = () => {
    setCollapsedById((prev) => {
      const next = { ...prev };
      phaseIds.forEach((id) => {
        next[id] = true;
      });
      return next;
    });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-slate-400">
          {phases.length} phase{phases.length !== 1 ? 's' : ''} configured
        </p>
        <div className="flex items-center gap-2">
          <Button type="button" size="sm" variant="ghost" onClick={handleCollapseAll} disabled={phases.length === 0}>
            Collapse all
          </Button>
          <Button type="button" size="sm" onClick={handleAddPhase}>
            <Plus className="h-4 w-4 mr-2" />
            Add Phase
          </Button>
        </div>
      </div>

      {phases.length === 0 ? (
        <div className="text-center py-8 border border-dashed border-slate-700 rounded-lg">
          <p className="text-slate-400 mb-2">No phases yet</p>
          <p className="text-sm text-slate-500">Add a phase to get started</p>
        </div>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={phaseIds} strategy={verticalListSortingStrategy}>
            <div className="space-y-3">
              {phases.map((phase, index) => {
                const phaseId = phaseIds[index];
                return (
                  <SortablePhaseItem
                    key={phaseId}
                    id={phaseId}
                    phase={phase}
                    index={index}
                    onChange={(updated) => handleUpdatePhase(index, updated)}
                    onRemove={() => handleRemovePhase(index)}
                    onMoveUp={() => handleMovePhase(index, index - 1)}
                    onMoveDown={() => handleMovePhase(index, index + 1)}
                    onDuplicate={() => handleDuplicatePhase(index)}
                    isFirst={index === 0}
                    isLast={index === phases.length - 1}
                    isCollapsed={collapsedById[phaseId] ?? true}
                    onToggleCollapse={() => handleToggleCollapse(phaseId)}
                  />
                );
              })}
            </div>
          </SortableContext>
        </DndContext>
      )}
    </div>
  );
}
