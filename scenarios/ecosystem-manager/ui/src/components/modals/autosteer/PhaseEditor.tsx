/**
 * PhaseEditor Component
 * Edit a single phase in an Auto Steer profile
 */

import { useState } from 'react';
import { Trash2, CodeIcon, ArrowUp, ArrowDown, Copy } from 'lucide-react';
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
import { Slider } from '@/components/ui/slider';
import { AVAILABLE_METRICS } from './ConditionNode';

interface PhaseEditorProps {
  phase: AutoSteerPhase;
  index: number;
  onChange: (phase: AutoSteerPhase) => void;
  onRemove: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  onDuplicate?: () => void;
  isFirst: boolean;
  isLast: boolean;
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

const MODE_DESCRIPTIONS: Record<string, string> = {
  progress: 'Structured, step-by-step execution focused on momentum.',
  ux: 'UI/UX polish and usability-first execution.',
  refactor: 'Technical cleanup, readability, and maintainability.',
  test: 'Validation-heavy; adds/updates coverage and reliability.',
  explore: 'Discovery and research without strict output constraints.',
  polish: 'Tighten copy, micro-interactions, and quality details.',
  integration: 'Wire external/internal systems together safely.',
  performance: 'Optimize throughput, memory, and responsiveness.',
  security: 'Harden surfaces, permissions, and guardrails.',
};

const MODE_LABELS = Object.fromEntries(PHASE_MODES.map((m) => [m.value, m.label]));

export function PhaseEditor({
  phase,
  index,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
  onDuplicate,
  isFirst,
  isLast,
}: PhaseEditorProps) {
  const [isConditionBuilderOpen, setIsConditionBuilderOpen] = useState(false);

  const updateField = (field: keyof AutoSteerPhase, value: any) => {
    onChange({ ...phase, [field]: value });
  };

  const updateIterations = (value: number) => {
    const normalized = Number.isFinite(value) ? Math.min(Math.max(value, 1), 100) : 1;
    updateField('max_iterations', normalized);
  };

  const conditionCount = phase.stop_conditions?.length || 0;
  const conditionSummary = (phase.stop_conditions || []).map((condition) =>
    summarizeCondition(condition)
  );

  const modeLabel = MODE_LABELS[phase.mode] || 'Select a mode';
  const modeDescription = MODE_DESCRIPTIONS[phase.mode] || 'Choose a focus for this phase.';

  return (
    <div className="p-4 border border-slate-700 rounded-lg bg-slate-800/50 space-y-4 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span className="text-xs uppercase tracking-wide text-slate-400">
            Phase #{index + 1}
          </span>

          <div className="flex items-center gap-2 text-xs text-slate-300">
            <span className="rounded-full bg-slate-900 px-2 py-1 border border-slate-700">
              {modeLabel}
            </span>
            <span className="rounded-full bg-slate-900 px-2 py-1 border border-slate-700">
              {phase.max_iterations || 10} iterations
            </span>
            <span className="rounded-full bg-slate-900 px-2 py-1 border border-slate-700">
              {conditionCount} condition{conditionCount === 1 ? '' : 's'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onMoveUp}
            disabled={isFirst}
            title="Move phase up"
            className="h-10 w-10 p-0 text-slate-100"
          >
            <ArrowUp className="h-5 w-5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onMoveDown}
            disabled={isLast}
            title="Move phase down"
            className="h-10 w-10 p-0 text-slate-100"
          >
            <ArrowDown className="h-5 w-5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onDuplicate}
            title="Duplicate phase"
            className="h-10 w-10 p-0 text-slate-100"
          >
            <Copy className="h-5 w-5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            onClick={onRemove}
            title="Remove phase"
            size="sm"
            className="h-10 w-10 p-0 text-red-400 hover:text-red-300"
          >
            <Trash2 className="h-5 w-5" />
          </Button>
        </div>
      </div>

      <p className="text-xs text-slate-500">
        Use the arrows to reorder phases; duplicate to branch small variations quickly.
      </p>

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
          <p className="text-xs text-slate-500 mt-1">{modeDescription}</p>
        </div>

        <div>
          <Label className="flex items-center justify-between">
            <span>Max Iterations *</span>
            <span className="text-xs text-slate-400">
              {phase.max_iterations || 10} per run
            </span>
          </Label>
          <div className="flex items-center gap-3 mt-2">
            <Slider
              min={1}
              max={50}
              step={1}
              value={[phase.max_iterations || 10]}
              onValueChange={(value) => updateIterations(value[0])}
            />
            <Input
              type="number"
              min={1}
              max={100}
              className="w-20"
              value={phase.max_iterations || 10}
              onChange={(e) => updateIterations(parseInt(e.target.value, 10))}
              required
            />
          </div>
          <p className="text-xs text-slate-500 mt-1">
            Keep between 3-20 for focused steps; raise for exploratory phases.
          </p>
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
          <div className="mt-2 space-y-2 rounded border border-slate-700 bg-slate-900/60 p-3 text-xs text-slate-200">
            {conditionSummary.map((summary, idx) => (
              <div key={idx} className="flex items-start gap-2">
                <span className="text-slate-500">#{idx + 1}</span>
                <span className="leading-5">{summary}</span>
              </div>
            ))}
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

function summarizeCondition(condition: any): string {
  if (!condition) return 'Unknown condition';

  if (condition.type === 'simple') {
    const metricLabel =
      AVAILABLE_METRICS.find((m) => m.value === condition.metric)?.label ||
      condition.metric ||
      'Metric';
    return `${metricLabel} ${condition.compare_operator || '>'} ${condition.value ?? 0}`;
  }

  if (condition.type === 'compound' && condition.conditions?.length) {
    const inner = condition.conditions.map((child: any) => summarizeCondition(child)).join(' • ');
    return `${condition.logic_operator || 'AND'}: ${inner}`;
  }

  return 'Empty condition';
}
