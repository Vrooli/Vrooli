/**
 * PhaseEditor Component
 * Edit a single phase in an Auto Steer profile
 */

import { useState } from 'react';
import { Trash2, CodeIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { AutoSteerPhase } from '@/types/api';
import { ConditionBuilderModal } from './ConditionBuilderModal';

interface PhaseEditorProps {
  phase: AutoSteerPhase;
  index: number;
  onChange: (phase: AutoSteerPhase) => void;
  onRemove: () => void;
}

const PHASE_MODES = [
  { value: 'progress', label: 'Progress' },
  { value: 'ux', label: 'UX' },
  { value: 'refactor', label: 'Refactor' },
  { value: 'test', label: 'Test' },
  { value: 'explore', label: 'Explore' },
  { value: 'polish', label: 'Polish' },
  { value: 'integration', label: 'Integration' },
  { value: 'performance', label: 'Performance' },
  { value: 'security', label: 'Security' },
];

export function PhaseEditor({ phase, index, onChange, onRemove }: PhaseEditorProps) {
  const [isConditionBuilderOpen, setIsConditionBuilderOpen] = useState(false);

  const updateField = (field: keyof AutoSteerPhase, value: any) => {
    onChange({ ...phase, [field]: value });
  };

  const conditionCount = phase.stop_conditions?.length || 0;

  return (
    <div className="p-4 border border-slate-700 rounded-lg bg-slate-800/50 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-slate-300">Phase #{index + 1}</h4>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onRemove}
          className="text-red-400 hover:text-red-300"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      {/* Fields */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>Mode *</Label>
          <Select value={phase.mode} onValueChange={(value) => updateField('mode', value)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PHASE_MODES.map((mode) => (
                <SelectItem key={mode.value} value={mode.value}>
                  {mode.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div>
          <Label>Max Iterations *</Label>
          <Input
            type="number"
            min={1}
            max={100}
            value={phase.max_iterations || 10}
            onChange={(e) => updateField('max_iterations', parseInt(e.target.value, 10))}
            required
          />
        </div>
      </div>

      <div>
        <Label>Description (optional)</Label>
        <Input
          value={phase.description || ''}
          onChange={(e) => updateField('description', e.target.value)}
          placeholder="Optional description for this phase"
        />
      </div>

      {/* Stop Conditions */}
      <div>
        <Label>Stop Conditions</Label>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setIsConditionBuilderOpen(true)}
          >
            <CodeIcon className="h-4 w-4 mr-2" />
            Configure Conditions
            {conditionCount > 0 && (
              <span className="ml-2 px-1.5 py-0.5 rounded-full bg-blue-500 text-xs">
                {conditionCount}
              </span>
            )}
          </Button>
        </div>

        {conditionCount > 0 && (
          <div className="mt-2 p-2 bg-slate-900 rounded text-xs text-slate-400">
            {conditionCount} condition{conditionCount !== 1 ? 's' : ''} configured
          </div>
        )}
      </div>

      {/* Condition Builder Modal */}
      <ConditionBuilderModal
        open={isConditionBuilderOpen}
        onOpenChange={setIsConditionBuilderOpen}
        conditions={phase.stop_conditions || []}
        onChange={(conditions) => updateField('stop_conditions', conditions)}
      />
    </div>
  );
}
