import clsx from 'clsx';
import type { ReplayPresentationSettings } from '../model';

type ReplayPresentationModeVariant = 'settings' | 'compact';

interface ReplayPresentationModeSettingsProps {
  presentation: ReplayPresentationSettings;
  onChange: (value: ReplayPresentationSettings) => void;
  variant?: ReplayPresentationModeVariant;
}

const MODE_OPTIONS: Array<{
  key: keyof ReplayPresentationSettings;
  label: string;
  description: string;
}> = [
  {
    key: 'showDesktop',
    label: 'Desktop',
    description: 'Background stage behind the replay.',
  },
  {
    key: 'showBrowserFrame',
    label: 'Browser',
    description: 'Window chrome + frame styling.',
  },
  {
    key: 'showDeviceFrame',
    label: 'Device Frame',
    description: 'Outer hardware frame around the stage.',
  },
];

export function ReplayPresentationModeSettings({
  presentation,
  onChange,
  variant = 'settings',
}: ReplayPresentationModeSettingsProps) {
  const isCompact = variant === 'compact';
  const handleToggle = (key: keyof ReplayPresentationSettings) => {
    onChange({ ...presentation, [key]: !presentation[key] });
  };

  return (
    <div
      className={clsx('grid gap-2', isCompact ? 'sm:grid-cols-3' : 'sm:grid-cols-3')}
      role="group"
      aria-label="Replay presentation"
    >
      {MODE_OPTIONS.map((option) => {
        const isActive = presentation[option.key];
        return (
          <button
            key={option.key}
            type="button"
            role="checkbox"
            aria-checked={isActive}
            onClick={() => handleToggle(option.key)}
            className={clsx(
              'flex flex-col items-start rounded-xl border text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
              isCompact ? 'px-3 py-2' : 'px-4 py-3',
              isActive
                ? 'border-flow-accent bg-flow-accent/10 text-white shadow-[0_12px_35px_rgba(59,130,246,0.25)]'
                : isCompact
                  ? 'border-white/10 bg-slate-900/60 text-slate-300 hover:border-white/20'
                  : 'border-gray-800 bg-gray-900/60 text-gray-300 hover:border-gray-700',
            )}
          >
            <span className={clsx(isCompact ? 'text-[11px] font-semibold' : 'text-sm font-medium')}>
              {option.label}
            </span>
            <span className={clsx('mt-1', isCompact ? 'text-[10px] text-slate-400' : 'text-xs text-gray-500')}>
              {option.description}
            </span>
          </button>
        );
      })}
    </div>
  );
}

export default ReplayPresentationModeSettings;
