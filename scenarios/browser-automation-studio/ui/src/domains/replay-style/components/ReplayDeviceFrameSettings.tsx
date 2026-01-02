import clsx from 'clsx';
import { Check } from 'lucide-react';
import { REPLAY_DEVICE_FRAME_OPTIONS } from '../catalog';
import type { ReplayDeviceFrameTheme } from '../model';

type ReplayDeviceFrameSettingsVariant = 'settings' | 'compact';

interface ReplayDeviceFrameSettingsProps {
  deviceFrameTheme: ReplayDeviceFrameTheme;
  onDeviceFrameThemeChange: (value: ReplayDeviceFrameTheme) => void;
  variant?: ReplayDeviceFrameSettingsVariant;
}

export function ReplayDeviceFrameSettings({
  deviceFrameTheme,
  onDeviceFrameThemeChange,
  variant = 'settings',
}: ReplayDeviceFrameSettingsProps) {
  const isCompact = variant === 'compact';

  return (
    <div
      className={clsx(
        'grid gap-3',
        isCompact ? 'grid-cols-2 sm:grid-cols-3' : 'grid-cols-2 sm:grid-cols-3',
      )}
      role="radiogroup"
      aria-label="Device frame style"
    >
      {REPLAY_DEVICE_FRAME_OPTIONS.map((option) => {
        const isActive = deviceFrameTheme === option.id;
        return (
          <button
            key={option.id}
            type="button"
            role="radio"
            aria-checked={isActive}
            onClick={() => onDeviceFrameThemeChange(option.id)}
            title={option.subtitle}
            className={clsx(
              'relative flex flex-col items-center rounded-xl border p-3 transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
              isActive
                ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
                : 'border-gray-700 bg-gray-800/30 hover:border-gray-600 hover:bg-gray-800/50',
            )}
          >
            {/* Selection indicator */}
            <div
              className={clsx(
                'absolute right-2 top-2 flex h-4 w-4 items-center justify-center rounded-full border-2 transition-colors',
                isActive
                  ? 'border-flow-accent bg-flow-accent'
                  : 'border-slate-600 bg-transparent',
              )}
            >
              {isActive && <Check className="h-2.5 w-2.5 text-white" strokeWidth={3} />}
            </div>

            {/* Device frame preview */}
            <div className="mb-2 flex h-16 items-center justify-center">
              {option.preview}
            </div>

            {/* Label and subtitle */}
            <div className="text-center">
              <div className={clsx(
                'font-medium',
                isCompact ? 'text-xs' : 'text-sm',
                isActive ? 'text-white' : 'text-gray-300',
              )}>
                {option.label}
              </div>
              {!isCompact && (
                <div className="mt-0.5 text-xs text-gray-500">{option.subtitle}</div>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
}

export default ReplayDeviceFrameSettings;
