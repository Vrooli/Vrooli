/**
 * AllowedSkillsPicker
 * Selects which steer skills the controller is permitted to choose from for a
 * profile. The controller will never select a skill outside this allow-set.
 */

import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SkillOption {
  id: string;
  name: string;
}

interface AllowedSkillsPickerProps {
  selected: string[];
  onChange: (skills: string[]) => void;
  options: SkillOption[];
  isLoading?: boolean;
}

export function AllowedSkillsPicker({
  selected,
  onChange,
  options,
  isLoading,
}: AllowedSkillsPickerProps) {
  const selectedSet = new Set(selected);

  // Surface any selected skill that isn't in the catalog so it's never silently dropped.
  const knownIds = new Set(options.map((o) => o.id));
  const orphanOptions: SkillOption[] = selected
    .filter((id) => !knownIds.has(id))
    .map((id) => ({ id, name: id }));
  const allOptions = [...options, ...orphanOptions];

  const toggle = (id: string) => {
    if (selectedSet.has(id)) {
      onChange(selected.filter((s) => s !== id));
    } else {
      onChange([...selected, id]);
    }
  };

  if (isLoading) {
    return <p className="text-xs text-slate-500">Loading skills...</p>;
  }

  if (allOptions.length === 0) {
    return <p className="text-xs text-slate-500">No steer skills available.</p>;
  }

  return (
    <div className="flex flex-wrap gap-2" role="group" aria-label="Allowed skills">
      {allOptions.map((opt) => {
        const isSelected = selectedSet.has(opt.id);
        return (
          <button
            key={opt.id}
            type="button"
            role="checkbox"
            aria-checked={isSelected}
            onClick={() => toggle(opt.id)}
            className={cn(
              'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs border transition-colors',
              isSelected
                ? 'bg-indigo-500/15 text-indigo-200 border-indigo-500/40'
                : 'bg-slate-800/40 text-slate-300 border-white/10 hover:border-indigo-500/40',
            )}
            title={opt.id}
          >
            {isSelected && <Check className="h-3 w-3" />}
            {opt.name}
          </button>
        );
      })}
    </div>
  );
}
