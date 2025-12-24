import clsx from 'clsx';
import { useMemo } from 'react';
import { REPLAY_DEVICE_FRAME_OPTIONS } from '../catalog';
import type { ReplayDeviceFrameTheme } from '../model';

type ReplayDeviceFrameSettingsVariant = 'settings' | 'compact';

interface ReplayDeviceFrameSettingsProps {
  deviceFrameTheme: ReplayDeviceFrameTheme;
  onDeviceFrameThemeChange: (value: ReplayDeviceFrameTheme) => void;
  variant?: ReplayDeviceFrameSettingsVariant;
}

const getStyles = (variant: ReplayDeviceFrameSettingsVariant) => {
  const isCompact = variant === 'compact';
  return {
    isCompact,
    grid: isCompact ? 'flex flex-wrap items-center gap-2' : 'grid grid-cols-2 sm:grid-cols-3 gap-3',
    cardBase: clsx(
      'rounded-xl border transition-all text-left',
      isCompact ? 'px-3 py-2 text-xs font-medium' : 'flex flex-col gap-2 p-3',
    ),
    cardActive: isCompact
      ? 'bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]'
      : 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50',
    cardIdle: isCompact
      ? 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80'
      : 'border-gray-700 hover:border-gray-600 bg-gray-800/30 text-gray-300',
    cardLabel: isCompact ? 'text-xs' : 'text-sm font-medium text-surface',
    cardSubtitle: isCompact ? 'text-[11px] text-slate-400' : 'text-xs text-gray-500',
    previewBox: clsx(
      'relative h-8 w-14 rounded-2xl bg-slate-950/70',
      isCompact ? 'ring-offset-2' : 'ring-offset-4',
    ),
  };
};

export function ReplayDeviceFrameSettings({
  deviceFrameTheme,
  onDeviceFrameThemeChange,
  variant = 'settings',
}: ReplayDeviceFrameSettingsProps) {
  const styles = useMemo(() => getStyles(variant), [variant]);

  return (
    <div className={styles.grid}>
      {REPLAY_DEVICE_FRAME_OPTIONS.map((option) => (
        <button
          key={option.id}
          type="button"
          onClick={() => onDeviceFrameThemeChange(option.id)}
          title={option.subtitle}
          className={clsx(
            styles.cardBase,
            deviceFrameTheme === option.id ? styles.cardActive : styles.cardIdle,
          )}
        >
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className={styles.cardLabel}>{option.label}</div>
              {!styles.isCompact && <div className={styles.cardSubtitle}>{option.subtitle}</div>}
            </div>
            <div className={clsx(styles.previewBox, option.previewClass)}>
              {option.previewNode}
            </div>
          </div>
        </button>
      ))}
    </div>
  );
}

export default ReplayDeviceFrameSettings;
