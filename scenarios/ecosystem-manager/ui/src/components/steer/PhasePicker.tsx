import { useState } from 'react';
import { ChevronRight, Compass } from 'lucide-react';
import { cn, formatPhaseName } from '@/lib/utils';
import { usePhaseNames } from '@/hooks/usePromptFiles';
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
}

/**
 * PhasePicker - A reusable phase selection component with search, sort, and grid view.
 *
 * Can either receive phaseNames as a prop or fetch them internally via usePhaseNames().
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
}: PhasePickerProps) {
  const [dialogOpen, setDialogOpen] = useState(false);

  // Use external data if provided, otherwise fetch internally
  const { data: internalPhaseNames = [], isLoading: internalLoading } = usePhaseNames();

  const phaseNames = externalPhaseNames ?? internalPhaseNames;
  const isLoading = externalLoading ?? internalLoading;

  // Find selected phase for display
  const selectedPhase = phaseNames.find((p) => p.name === value);
  const displayName = selectedPhase ? formatPhaseName(selectedPhase.name) : null;
  const displayDescription = selectedPhase?.description;

  const handleSelect = (phaseName: string) => {
    onChange(phaseName);
  };

  return (
    <>
      <button
        type="button"
        onClick={() => !disabled && setDialogOpen(true)}
        disabled={disabled}
        className={cn(
          'flex items-center gap-2 px-3 py-2 rounded-md border transition-colors text-left w-full overflow-hidden',
          'bg-slate-800/50 border-slate-700 hover:bg-slate-800 hover:border-slate-600',
          disabled && 'opacity-50 cursor-not-allowed',
          value && 'border-amber-500/30 bg-amber-500/5',
          className
        )}
      >
        <Compass className="h-4 w-4 shrink-0 text-slate-400" />
        <div className="flex-1 min-w-0 overflow-hidden">
          {displayName ? (
            <>
              <div className="text-sm font-medium text-slate-100 truncate">{displayName}</div>
              {displayDescription && (
                <div className="text-xs text-slate-400 line-clamp-1 break-all">{displayDescription}</div>
              )}
            </>
          ) : (
            <div className="text-sm text-slate-400">{isLoading ? 'Loading...' : placeholder}</div>
          )}
        </div>
        <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" />
      </button>

      <PhasePickerDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        value={value}
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
