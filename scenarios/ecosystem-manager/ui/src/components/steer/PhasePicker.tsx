import { useMemo, useState } from 'react';
import { ChevronRight, Compass } from 'lucide-react';
import {
  cn,
  formatSkillSetLabel,
  formatSkillSetTooltip,
  getPhaseDisplayName,
  normalizeSkillId,
} from '@/lib/utils';
import { useMergedPhaseNames } from '@/hooks/usePromptFiles';
import { PhasePickerDialog } from './PhasePickerDialog';
import type { PhaseInfo } from '@/types/api';

interface PhasePickerProps {
  values?: string[];
  onChange: (phaseIds: string[]) => void;
  phaseNames?: PhaseInfo[];
  isLoading?: boolean;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  dialogTitle?: string;
  dialogDescription?: string;
  confirmLabel?: string;
  variant?: 'default' | 'compact';
  selectionMode?: 'single' | 'multiple';
}

export function PhasePicker({
  values,
  onChange,
  phaseNames: externalPhaseNames,
  isLoading: externalLoading,
  placeholder = 'Select focus skills',
  disabled,
  className,
  dialogTitle,
  dialogDescription,
  confirmLabel,
  variant = 'default',
  selectionMode = 'single',
}: PhasePickerProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const isCompact = variant === 'compact';

  const { data: internalPhaseNames = [], isLoading: internalLoading } = useMergedPhaseNames();

  const phaseNames = externalPhaseNames ?? internalPhaseNames;
  const isLoading = externalLoading ?? internalLoading;
  const normalizedValues = useMemo(
    () => (values ?? []).map((id) => normalizeSkillId(id)).filter(Boolean),
    [values]
  );

  const displayName = formatSkillSetLabel(normalizedValues, phaseNames, {
    maxVisible: isCompact ? 1 : 2,
    emptyLabel: '',
  });
  const displayDescription = formatSkillSetTooltip(normalizedValues, phaseNames);

  const handleConfirm = (phaseIds: string[]) => {
    const normalized = phaseIds.map((id) => normalizeSkillId(id)).filter(Boolean);
    if (selectionMode === 'single') {
      onChange(normalized.slice(0, 1));
      return;
    }
    onChange(normalized);
  };

  const singleDisplayName =
    normalizedValues.length > 0
      ? getPhaseDisplayName(normalizedValues[0], phaseNames)
      : undefined;
  const buttonLabel =
    selectionMode === 'single' ? singleDisplayName ?? '' : displayName;

  return (
    <>
      <button
        type="button"
        onClick={() => !disabled && setDialogOpen(true)}
        disabled={disabled}
        className={cn(
          'flex items-center gap-2 rounded-md border transition-colors text-left w-full overflow-hidden',
          'bg-slate-800/50 border-slate-700 hover:bg-slate-800 hover:border-slate-600',
          disabled && 'opacity-50 cursor-not-allowed',
          normalizedValues.length > 0 && 'border-amber-500/30 bg-amber-500/5',
          isCompact ? 'px-2 py-1.5' : 'px-3 py-2',
          className
        )}
      >
        <Compass className={cn('shrink-0 text-slate-400', isCompact ? 'h-3.5 w-3.5' : 'h-4 w-4')} />
        <div className="flex-1 min-w-0 overflow-hidden">
          {buttonLabel ? (
            <>
              <div className={cn('font-medium text-slate-100 truncate', isCompact ? 'text-xs' : 'text-sm')}>
                {buttonLabel}
              </div>
              {!isCompact && displayDescription && (
                <div className="text-xs text-slate-400 line-clamp-1 break-all">{displayDescription}</div>
              )}
            </>
          ) : (
            <div className={cn('text-slate-400', isCompact ? 'text-xs' : 'text-sm')}>
              {isLoading ? 'Loading...' : placeholder}
            </div>
          )}
        </div>
        <ChevronRight className={cn('shrink-0 text-slate-500', isCompact ? 'h-3.5 w-3.5' : 'h-4 w-4')} />
      </button>

      <PhasePickerDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        values={normalizedValues}
        onConfirm={handleConfirm}
        phaseNames={phaseNames}
        isLoading={isLoading}
        title={dialogTitle}
        description={dialogDescription}
        selectionMode={selectionMode}
        confirmLabel={confirmLabel}
      />
    </>
  );
}

export { PhasePickerDialog } from './PhasePickerDialog';
