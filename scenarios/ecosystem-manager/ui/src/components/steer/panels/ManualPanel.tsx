import { Compass } from 'lucide-react';
import { Label } from '@/components/ui/label';
import { SkillPicker } from '../SkillPicker';
import { formatSkillSetLabel, formatSkillSetTooltip } from '@/lib/utils';
import type { SkillInfo } from '@/types/api';

interface ManualPanelProps {
  value: string[];
  onChange: (set: string[]) => void;
  skillNames: SkillInfo[];
  isLoading?: boolean;
}

export function ManualPanel({ value, onChange, skillNames, isLoading }: ManualPanelProps) {
  const displayName = formatSkillSetLabel(value, skillNames, { maxVisible: 1, emptyLabel: '' });
  const displayTooltip = formatSkillSetTooltip(value, skillNames);

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 mb-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500/10 shrink-0">
          <Compass className="h-5 w-5 text-amber-400" />
        </div>
        <div>
          <h3 className="text-sm font-medium text-slate-200">Manual Steering</h3>
          <p className="text-sm text-slate-400 mt-0.5">
            Select one or more focus skills. The task will use this skill set for every execution until
            changed.
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <Label>Focus Skills</Label>
        <SkillPicker
          values={value}
          onChange={onChange}
          skillNames={skillNames}
          isLoading={isLoading}
          selectionMode="multiple"
          placeholder="Select focus skills"
          dialogTitle="Select Focus Skills"
          dialogDescription="Choose one or more steering skills for manual mode."
        />
        {value.length > 0 && displayName && (
          <p className="text-xs text-slate-500">
            The task will focus on <span className="text-amber-400" title={displayTooltip}>{displayName}</span>{' '}
            improvements.
          </p>
        )}
      </div>
    </div>
  );
}
