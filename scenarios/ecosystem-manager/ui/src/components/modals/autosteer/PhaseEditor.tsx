/**
 * PhaseEditor Component
 * Edit a single phase in an Auto Steer profile
 */

import { useState, type HTMLAttributes, type KeyboardEvent, type MouseEvent } from 'react';
import { ArrowDown, ArrowUp, ChevronDown, CodeIcon, Copy, GripVertical, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { AutoSteerPhase } from '@/types/api';
import { ConditionBuilderModal } from './ConditionBuilderModal';
import { Slider } from '@/components/ui/slider';
import { AVAILABLE_METRICS } from './ConditionNode';
import { useMergedPhaseNames } from '@/hooks/usePromptFiles';
import { PhasePicker } from '@/components/steer/PhasePicker';
import { formatSkillSetLabel, formatSkillSetTooltip } from '@/lib/utils';

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
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
  dragHandleProps?: HTMLAttributes<HTMLButtonElement>;
  isDragging?: boolean;
}

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
  isCollapsed = true,
  onToggleCollapse,
  dragHandleProps,
  isDragging,
}: PhaseEditorProps) {
  const [isConditionBuilderOpen, setIsConditionBuilderOpen] = useState(false);
  const { data: phaseNames = [], isLoading: phasesLoading } = useMergedPhaseNames();

  const updateField = (field: keyof AutoSteerPhase, value: unknown) => {
    onChange({ ...phase, [field]: value });
  };

  const updateIterations = (value: number) => {
    const normalized = Number.isFinite(value) ? Math.min(Math.max(value, 1), 100) : 1;
    updateField('max_iterations', normalized);
  };

  const conditionCount = phase.stop_conditions?.length || 0;
  const conditionSummary = (phase.stop_conditions || []).map((condition) => summarizeCondition(condition));

  const selectedSkills = phase.skill_ids ?? [];
  const modeLabel = formatSkillSetLabel(selectedSkills, phaseNames, {
    maxVisible: isCollapsed ? 1 : 2,
    emptyLabel: 'Select skills',
  });
  const modeDescription =
    phase.description ||
    formatSkillSetTooltip(selectedSkills, phaseNames) ||
    `Skill set: ${modeLabel}`;
  const maxIterations = phase.max_iterations || 10;

  const handlePhaseSelect = (skillIds: string[]) => {
    onChange({
      ...phase,
      skill_ids: skillIds,
      skill_name: formatSkillSetLabel(skillIds, phaseNames, { maxVisible: 1, emptyLabel: '' }),
      modes: [],
    });
  };

  const containerClasses = [
    'border border-slate-700 rounded-lg shadow-sm',
    isCollapsed ? 'bg-slate-900/40 px-3 py-2' : 'bg-slate-800/50 p-4 space-y-4',
    isDragging ? 'ring-2 ring-cyan-500/40 shadow-lg' : '',
  ].join(' ');

  const handleToggleCollapse = () => {
    if (onToggleCollapse) {
      onToggleCollapse();
    }
  };

  const handleHeaderKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!onToggleCollapse) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onToggleCollapse();
    }
  };

  const stopPropagation = (event: MouseEvent) => {
    event.stopPropagation();
  };

  const handleDragHandleClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    dragHandleProps?.onClick?.(event);
  };

  const wrapAction =
    (handler?: () => void) =>
    (event: MouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      handler?.();
    };

  return (
    <div className={containerClasses}>
      <div
        className="flex items-center justify-between gap-3"
        onClick={onToggleCollapse ? handleToggleCollapse : undefined}
        role={onToggleCollapse ? 'button' : undefined}
        tabIndex={onToggleCollapse ? 0 : undefined}
        onKeyDown={onToggleCollapse ? handleHeaderKeyDown : undefined}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <button
            type="button"
            className="cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-300 touch-none"
            {...dragHandleProps}
            onClick={handleDragHandleClick}
            aria-label="Drag to reorder phase"
          >
            <GripVertical className="h-4 w-4" />
          </button>

          <span className="text-xs text-slate-500 font-mono w-6">{index + 1}.</span>

          {isCollapsed && (
            <div className="flex items-center gap-2 flex-1 min-w-0 flex-wrap" onClick={stopPropagation}>
              <div className="min-w-[220px] flex-1">
                <PhasePicker
                  values={selectedSkills}
                  onChange={handlePhaseSelect}
                  phaseNames={phaseNames}
                  isLoading={phasesLoading}
                  selectionMode="multiple"
                  placeholder="Select skills"
                  dialogTitle="Select Phase Skills"
                  dialogDescription="Choose one or more steering skills for this phase."
                  variant="compact"
                />
              </div>
              <div className="flex items-center gap-1">
                <span className="text-[10px] uppercase text-slate-500">Iter</span>
                <Input
                  type="number"
                  min={1}
                  max={100}
                  className="w-16 h-8 text-xs"
                  value={maxIterations}
                  onClick={stopPropagation}
                  onChange={(e) => updateIterations(parseInt(e.target.value, 10))}
                  required
                />
              </div>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1" onClick={stopPropagation}>
          {!isCollapsed && (
            <>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={wrapAction(onMoveUp)}
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
                onClick={wrapAction(onMoveDown)}
                disabled={isLast}
                title="Move phase down"
                className="h-10 w-10 p-0 text-slate-100"
              >
                <ArrowDown className="h-5 w-5" />
              </Button>
            </>
          )}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={wrapAction(onDuplicate)}
            title="Duplicate phase"
            className="h-10 w-10 p-0 text-slate-100"
          >
            <Copy className="h-5 w-5" />
          </Button>
          {!isCollapsed && (
            <Button
              type="button"
              variant="ghost"
              onClick={wrapAction(onRemove)}
              title="Remove phase"
              size="sm"
              className="h-10 w-10 p-0 text-red-400 hover:text-red-300"
            >
              <Trash2 className="h-5 w-5" />
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={wrapAction(handleToggleCollapse)}
            title={isCollapsed ? 'Expand phase' : 'Collapse phase'}
            className="h-10 w-10 p-0 text-slate-100"
          >
            <ChevronDown className={`h-5 w-5 transition-transform ${isCollapsed ? '' : 'rotate-180'}`} />
          </Button>
        </div>
      </div>

      {!isCollapsed && (
        <>
          <p className="text-xs text-slate-500">
            Use the arrows to reorder phases; duplicate to branch small variations quickly.
          </p>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Skills *</Label>
              <PhasePicker
                values={selectedSkills}
                onChange={handlePhaseSelect}
                phaseNames={phaseNames}
                isLoading={phasesLoading}
                selectionMode="multiple"
                placeholder="Select skills"
                dialogTitle="Select Phase Skills"
                dialogDescription="Choose one or more steering skills for this phase."
              />
              {modeDescription && (
                <p className="text-xs text-slate-500 mt-1 line-clamp-2" title={modeDescription}>{modeDescription}</p>
              )}
            </div>

            <div>
              <Label className="flex items-center justify-between">
                <span>Max Iterations *</span>
                <span className="text-xs text-slate-400">{maxIterations} per run</span>
              </Label>
              <div className="flex items-center gap-3 mt-2">
                <Slider
                  min={1}
                  max={50}
                  step={1}
                  value={[maxIterations]}
                  onValueChange={(value) => updateIterations(value[0])}
                />
                <Input
                  type="number"
                  min={1}
                  max={100}
                  className="w-20"
                  value={maxIterations}
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
        </>
      )}

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

  if (condition.type === 'compound') {
    const operator = condition.operator || 'AND';
    const nested = Array.isArray(condition.conditions) ? condition.conditions.length : 0;
    return `${operator} group with ${nested} condition${nested === 1 ? '' : 's'}`;
  }

  const metric = condition.metric || 'metric';
  const metricMeta = AVAILABLE_METRICS.find((item) => item.value === metric);
  const metricLabel = metricMeta?.label || metric;
  const operator = condition.compare_operator || condition.operator || '=';
  const value = condition.value ?? '0';
  return `${metricLabel} ${operator} ${value}`;
}
