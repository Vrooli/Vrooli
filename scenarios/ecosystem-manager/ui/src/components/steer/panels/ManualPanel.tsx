import { Compass } from 'lucide-react';
import { Label } from '@/components/ui/label';
import { PhasePicker } from '../PhasePicker';
import { formatPhaseName } from '@/lib/utils';
import type { PhaseInfo } from '@/types/api';

interface ManualPanelProps {
  value?: string;
  onChange: (mode: string | undefined) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
}

export function ManualPanel({ value, onChange, phaseNames, isLoading }: ManualPanelProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 mb-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500/10 shrink-0">
          <Compass className="h-5 w-5 text-amber-400" />
        </div>
        <div>
          <h3 className="text-sm font-medium text-slate-200">Manual Steering</h3>
          <p className="text-sm text-slate-400 mt-0.5">
            Select a single focus mode. The task will use this mode for every execution until
            changed.
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <Label>Focus Mode</Label>
        <PhasePicker
          value={value}
          onChange={onChange}
          phaseNames={phaseNames}
          isLoading={isLoading}
          placeholder="Select a focus mode"
          dialogTitle="Select Focus Mode"
          dialogDescription="Choose a steering phase for manual mode."
        />
        {value && (
          <p className="text-xs text-slate-500">
            The task will focus on <span className="text-amber-400">{formatPhaseName(value)}</span>{' '}
            improvements.
          </p>
        )}
      </div>
    </div>
  );
}
