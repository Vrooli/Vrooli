import clsx from 'clsx';
import type { ReplayPresentationMode } from '../model';

type ReplayPresentationModeVariant = 'settings' | 'compact';

interface ReplayPresentationModeSettingsProps {
  mode: ReplayPresentationMode;
  onChange: (value: ReplayPresentationMode) => void;
  variant?: ReplayPresentationModeVariant;
}

const MODE_OPTIONS: Array<{
  id: ReplayPresentationMode;
  label: string;
  description: string;
}> = [
  {
    id: 'desktop',
    label: 'Desktop',
    description: 'Browser frame inside a styled background.',
  },
  {
    id: 'frame',
    label: 'Frame Only',
    description: 'Maximized browser window without a background.',
  },
  {
    id: 'content',
    label: 'Content Only',
    description: 'Captured content only, no chrome or background.',
  },
];

export function ReplayPresentationModeSettings({
  mode,
  onChange,
  variant = 'settings',
}: ReplayPresentationModeSettingsProps) {
  if (variant === 'compact') {
    return (
      <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Replay mode">
        {MODE_OPTIONS.map((option) => {
          const isActive = option.id === mode;
          return (
            <button
              key={option.id}
              type="button"
              role="radio"
              aria-checked={isActive}
              onClick={() => onChange(option.id)}
              className={clsx(
                'rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
                isActive
                  ? 'bg-flow-accent text-white'
                  : 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80',
              )}
            >
              {option.label}
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <div className="grid gap-2 sm:grid-cols-3" role="radiogroup" aria-label="Replay mode">
      {MODE_OPTIONS.map((option) => {
        const isActive = option.id === mode;
        return (
          <button
            key={option.id}
            type="button"
            role="radio"
            aria-checked={isActive}
            onClick={() => onChange(option.id)}
            className={clsx(
              'flex flex-col items-start rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
              isActive
                ? 'border-flow-accent bg-flow-accent/10 text-white shadow-[0_12px_35px_rgba(59,130,246,0.25)]'
                : 'border-gray-800 bg-gray-900/60 text-gray-300 hover:border-gray-700',
            )}
          >
            <span className="text-sm font-medium">{option.label}</span>
            <span className="mt-1 text-xs text-gray-500">{option.description}</span>
          </button>
        );
      })}
    </div>
  );
}

export default ReplayPresentationModeSettings;
