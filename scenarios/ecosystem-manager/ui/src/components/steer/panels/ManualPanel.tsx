import { Compass } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import type { PhaseInfo } from '@/types/api';

interface ManualPanelProps {
  value?: string;
  onChange: (mode: string | undefined) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
}

function formatPhaseName(name: string): string {
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
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
        <Label htmlFor="manual-mode">Focus Mode</Label>
        <Select
          value={value || ''}
          onValueChange={(val) => onChange(val || undefined)}
          disabled={isLoading}
        >
          <SelectTrigger id="manual-mode" className="w-full">
            <SelectValue placeholder={isLoading ? 'Loading modes...' : 'Select a focus mode'} />
          </SelectTrigger>
          <SelectContent>
            {phaseNames.map((phase) => (
              <SelectItem key={phase.name} value={phase.name}>
                {formatPhaseName(phase.name)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
