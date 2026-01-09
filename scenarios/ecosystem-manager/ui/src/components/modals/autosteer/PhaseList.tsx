/**
 * PhaseList Component
 * Manage the list of phases in an Auto Steer profile
 */

import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { AutoSteerPhase } from '@/types/api';
import { PhaseEditor } from './PhaseEditor';

interface PhaseListProps {
  phases: AutoSteerPhase[];
  onChange: (phases: AutoSteerPhase[]) => void;
}

export function PhaseList({ phases, onChange }: PhaseListProps) {
  const handleAddPhase = () => {
    const newPhase: AutoSteerPhase = {
      id: `phase-${Date.now()}-${phases.length}`,
      mode: 'progress',
      max_iterations: 10,
      description: '',
      stop_conditions: [],
    };
    onChange([...phases, newPhase]);
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
    const copy: AutoSteerPhase = {
      ...JSON.parse(JSON.stringify(phaseToCopy)),
      id: `phase-${Date.now()}-${index}-copy`,
    };
    onChange([...phases.slice(0, index + 1), copy, ...phases.slice(index + 1)]);
  };

  const handleMovePhase = (from: number, to: number) => {
    if (to < 0 || to >= phases.length) return;
    const newPhases = [...phases];
    const [moved] = newPhases.splice(from, 1);
    newPhases.splice(to, 0, moved);
    onChange(newPhases);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-slate-400">
          {phases.length} phase{phases.length !== 1 ? 's' : ''} configured
        </p>
        <Button type="button" size="sm" onClick={handleAddPhase}>
          <Plus className="h-4 w-4 mr-2" />
          Add Phase
        </Button>
      </div>

      {phases.length === 0 ? (
        <div className="text-center py-8 border border-dashed border-slate-700 rounded-lg">
          <p className="text-slate-400 mb-2">No phases yet</p>
          <p className="text-sm text-slate-500">Add a phase to get started</p>
        </div>
      ) : (
        <div className="space-y-4">
          {phases.map((phase, index) => (
            <PhaseEditor
              key={phase.id ?? index}
              phase={phase}
              index={index}
              onChange={(updated) => handleUpdatePhase(index, updated)}
              onRemove={() => handleRemovePhase(index)}
              onMoveUp={() => handleMovePhase(index, index - 1)}
              onMoveDown={() => handleMovePhase(index, index + 1)}
              onDuplicate={() => handleDuplicatePhase(index)}
              isFirst={index === 0}
              isLast={index === phases.length - 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}
