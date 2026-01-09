import clsx from 'clsx';
import { useMemo } from 'react';
import { RangeSlider } from '@shared/ui';
import { REPLAY_CHROME_OPTIONS } from '../catalog';
import { MAX_BROWSER_SCALE, MIN_BROWSER_SCALE } from '../constants';
import { clampReplayBrowserScale, type ReplayChromeTheme } from '../model';

type ReplayChromeSettingsVariant = 'settings' | 'compact';

interface ReplayChromeSettingsProps {
  chromeTheme: ReplayChromeTheme;
  browserScale: number;
  onChromeThemeChange: (value: ReplayChromeTheme) => void;
  onBrowserScaleChange: (value: number) => void;
  variant?: ReplayChromeSettingsVariant;
}

const getStyles = (variant: ReplayChromeSettingsVariant) => {
  const isCompact = variant === 'compact';
  return {
    isCompact,
    grid: isCompact ? 'flex flex-wrap items-center gap-2' : 'grid grid-cols-2 sm:grid-cols-4 gap-2',
    cardBase: clsx(
      'rounded-lg border transition-all text-center',
      isCompact ? 'px-3 py-1.5 text-xs font-medium' : 'flex flex-col items-center p-3',
    ),
    cardActive: isCompact
      ? 'bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]'
      : 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50',
    cardIdle: isCompact
      ? 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80'
      : 'border-gray-700 hover:border-gray-600 bg-gray-800/30 text-gray-300',
    cardLabel: isCompact ? 'text-xs' : 'text-sm font-medium text-surface',
    cardSubtitle: isCompact ? 'text-[11px] text-slate-400' : 'text-xs text-gray-500 mt-0.5',
    sliderPanel: clsx(
      'rounded-xl border px-3 py-2',
      isCompact ? 'border-white/10 bg-slate-900/60' : 'border-gray-800 bg-gray-900/50',
    ),
    sliderLabel: isCompact ? 'text-[11px] uppercase tracking-[0.24em] text-slate-400' : 'text-sm text-gray-300',
    sliderValue: isCompact ? 'text-[11px] text-slate-500' : 'text-sm text-gray-400',
    sliderHint: isCompact ? 'text-[11px] text-slate-500' : 'mt-2 text-xs text-gray-500',
  };
};

export function ReplayChromeSettings({
  chromeTheme,
  browserScale,
  onChromeThemeChange,
  onBrowserScaleChange,
  variant = 'settings',
}: ReplayChromeSettingsProps) {
  const styles = useMemo(() => getStyles(variant), [variant]);

  const handleBrowserScaleChange = (value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }
    onBrowserScaleChange(clampReplayBrowserScale(value));
  };

  return (
    <div className="space-y-4">
      <div className={styles.grid}>
        {REPLAY_CHROME_OPTIONS.map((option) => (
          <button
            key={option.id}
            type="button"
            onClick={() => onChromeThemeChange(option.id)}
            title={option.subtitle}
            className={clsx(
              styles.cardBase,
              chromeTheme === option.id ? styles.cardActive : styles.cardIdle,
              !styles.isCompact && 'text-gray-300',
            )}
          >
            <span className={styles.cardLabel}>{option.label}</span>
            {!styles.isCompact && <span className={styles.cardSubtitle}>{option.subtitle}</span>}
          </button>
        ))}
      </div>
      <div className={styles.sliderPanel}>
        <div className="flex items-center justify-between">
          <span className={styles.sliderLabel}>{styles.isCompact ? 'Browser size' : 'Browser size'}</span>
          <span className={styles.sliderValue}>{Math.round(browserScale * 100)}%</span>
        </div>
        <RangeSlider
          min={MIN_BROWSER_SCALE}
          max={MAX_BROWSER_SCALE}
          step={0.05}
          value={browserScale}
          onChange={handleBrowserScaleChange}
          ariaLabel="Browser size"
          className={styles.isCompact ? 'mt-2' : ''}
        />
        <div className={styles.sliderHint}>Scale how much of the replay canvas the browser frame occupies.</div>
      </div>
    </div>
  );
}
