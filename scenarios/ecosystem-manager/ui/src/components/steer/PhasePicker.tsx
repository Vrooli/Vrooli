import { useState } from 'react';
import { ChevronRight, Compass } from 'lucide-react';
import { cn, formatPhaseName, normalizeSteerMode } from '@/lib/utils';
import { useMergedPhaseNames } from '@/hooks/usePromptFiles';
import { PhasePickerDialog } from './PhasePickerDialog';
import type { PhaseInfo } from '@/types/api';

interface PhasePickerProps {
  value?: string;
  onChange: (phaseName: string | undefined) => void;
  phaseNames?: PhaseInfo[];
  isLoading?: boolean;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  dialogTitle?: string;
  dialogDescription?: string;
  variant?: 'default' | 'compact';
}

/**
 * PhasePicker - A reusable phase selection component with search, sort, and grid view.
 *
 * Can either receive phaseNames as a prop or fetch them internally via useMergedPhaseNames().
 * Tracks usage in localStorage for recent/most-used sorting.
 */
export function PhasePicker({
  value,
  onChange,
  phaseNames: externalPhaseNames,
  isLoading: externalLoading,
  placeholder = 'Select a phase',
  disabled,
  className,
  dialogTitle,
  dialogDescription,
  variant = 'default',
}: PhasePickerProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const isCompact = variant === 'compact';

  // Use external data if provided, otherwise fetch internally
  const { data: internalPhaseNames = [], isLoading: internalLoading } = useMergedPhaseNames();

  const phaseNames = externalPhaseNames ?? internalPhaseNames;
  const isLoading = externalLoading ?? internalLoading;

  // Find selected phase for display
  const normalizedValue = value ? normalizeSteerMode(value) : '';
  const selectedPhase = normalizedValue
    ? phaseNames.find((p) => normalizeSteerMode(p.name) === normalizedValue)
    : undefined;
  const displayName = selectedPhase
    ? formatPhaseName(selectedPhase.name)
    : normalizedValue
      ? formatPhaseName(normalizedValue)
      : null;
  const displayDescription = selectedPhase?.description;

  const handleSelect = (phaseName: string) => {
    onChange(normalizeSteerMode(phaseName));
  };

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
          value && 'border-amber-500/30 bg-amber-500/5',
          isCompact ? 'px-2 py-1.5' : 'px-3 py-2',
          className
        )}
      >
        <Compass className={cn('shrink-0 text-slate-400', isCompact ? 'h-3.5 w-3.5' : 'h-4 w-4')} />
        <div className="flex-1 min-w-0 overflow-hidden">
          {displayName ? (
            <>
              <div className={cn('font-medium text-slate-100 truncate', isCompact ? 'text-xs' : 'text-sm')}>
                {displayName}
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
        value={normalizedValue || undefined}
        onSelect={handleSelect}
        phaseNames={phaseNames}
        isLoading={isLoading}
        title={dialogTitle}
        description={dialogDescription}
      />
    </>
  );
}

// Re-export the dialog for direct use if needed
export { PhasePickerDialog } from './PhasePickerDialog';
