/**
 * ViewportPicker Component
 *
 * A shared component for selecting viewport/canvas dimensions.
 * Used across execution settings, workflow creation, and replay settings.
 *
 * Features:
 * - Preset dimension options (YouTube, Desktop, Instagram, TikTok)
 * - Optional custom dimensions toggle
 * - Width/height inputs for custom values
 * - Light and dark theme variants
 */

import { useMemo, useCallback } from 'react';
import { Monitor, Smartphone, Tv, Settings2 } from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

export interface ViewportDimensions {
  width: number;
  height: number;
}

export interface ViewportPresetOption {
  id: string;
  label: string;
  description: string;
  width: number;
  height: number;
  icon: React.ReactNode;
  ratioLabel: string;
}

export interface ViewportPickerProps {
  /** Current viewport dimensions */
  value: ViewportDimensions;
  /** Callback when dimensions change */
  onChange: (dimensions: ViewportDimensions) => void;
  /** Whether custom dimensions mode is enabled (shows width/height inputs) */
  useCustom?: boolean;
  /** Callback when custom mode changes */
  onUseCustomChange?: (useCustom: boolean) => void;
  /** Show the custom dimensions toggle */
  showCustomToggle?: boolean;
  /** Visual variant */
  variant?: 'light' | 'dark';
  /** Additional CSS classes */
  className?: string;
  /** Number of columns for preset grid */
  columns?: 2 | 4;
}

// ============================================================================
// Constants
// ============================================================================

export const VIEWPORT_PRESETS: ViewportPresetOption[] = [
  {
    id: 'widescreen-720',
    label: '1280 × 720',
    description: 'YouTube, demos',
    width: 1280,
    height: 720,
    icon: <Monitor size={14} />,
    ratioLabel: '16:9',
  },
  {
    id: 'full-hd',
    label: '1920 × 1080',
    description: 'Desktop shares',
    width: 1920,
    height: 1080,
    icon: <Tv size={14} />,
    ratioLabel: '16:9',
  },
  {
    id: 'instagram-feed',
    label: '1080 × 1350',
    description: 'Instagram feed',
    width: 1080,
    height: 1350,
    icon: <Smartphone size={14} />,
    ratioLabel: '4:5',
  },
  {
    id: 'tiktok',
    label: '1080 × 1920',
    description: 'TikTok, Reels',
    width: 1080,
    height: 1920,
    icon: <Smartphone size={14} />,
    ratioLabel: '9:16',
  },
];

const MIN_DIMENSION = 320;
const MAX_DIMENSION = 3840;

// Base height for aspect ratio preview (in pixels)
const PREVIEW_BASE = 32;

// ============================================================================
// Helper Functions
// ============================================================================

const clampDimension = (value: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback;
  return Math.min(Math.max(Math.round(value), MIN_DIMENSION), MAX_DIMENSION);
};

const findMatchingPreset = (width: number, height: number): ViewportPresetOption | null => {
  return VIEWPORT_PRESETS.find(
    (preset) => preset.width === width && preset.height === height
  ) ?? null;
};

/**
 * Calculate preview dimensions that accurately represent the aspect ratio.
 * Uses a fixed area approach so all previews feel visually balanced.
 */
const getPreviewDimensions = (width: number, height: number) => {
  const aspectRatio = width / height;
  // For landscape: width is based on height * ratio
  // For portrait: height is based on width / ratio
  if (aspectRatio >= 1) {
    // Landscape or square
    const h = PREVIEW_BASE;
    const w = Math.round(h * aspectRatio);
    return { width: w, height: h };
  } else {
    // Portrait
    const w = PREVIEW_BASE;
    const h = Math.round(w / aspectRatio);
    return { width: w, height: h };
  }
};

// ============================================================================
// Component
// ============================================================================

export function ViewportPicker({
  value,
  onChange,
  useCustom = false,
  onUseCustomChange,
  showCustomToggle = true,
  variant = 'dark',
  className = '',
  columns = 2,
}: ViewportPickerProps) {
  const activePreset = useMemo(
    () => findMatchingPreset(value.width, value.height),
    [value.width, value.height]
  );

  const handlePresetSelect = useCallback(
    (preset: ViewportPresetOption) => {
      onChange({ width: preset.width, height: preset.height });
      onUseCustomChange?.(false);
    },
    [onChange, onUseCustomChange]
  );

  const handleWidthChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const newWidth = clampDimension(parseInt(e.target.value, 10) || 0, 1280);
      onChange({ ...value, width: newWidth });
    },
    [onChange, value]
  );

  const handleHeightChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const newHeight = clampDimension(parseInt(e.target.value, 10) || 0, 720);
      onChange({ ...value, height: newHeight });
    },
    [onChange, value]
  );

  const isDark = variant === 'dark';

  // Theme-specific styles
  const styles = {
    card: (isActive: boolean) => {
      const base = 'group relative flex items-center gap-3 p-3 rounded-lg border transition-all cursor-pointer';
      if (isDark) {
        return `${base} ${
          isActive
            ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/40'
            : 'border-gray-700 hover:border-gray-500 bg-gray-800/40 hover:bg-gray-800/60'
        }`;
      }
      return `${base} ${
        isActive
          ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 ring-1 ring-blue-500/40'
          : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 bg-gray-50 dark:bg-gray-800/40'
      }`;
    },
    previewContainer: (isActive: boolean) => {
      const base = 'flex items-center justify-center rounded transition-colors';
      if (isDark) {
        return `${base} ${isActive ? 'bg-flow-accent/20' : 'bg-gray-700/50 group-hover:bg-gray-700/70'}`;
      }
      return `${base} ${isActive ? 'bg-blue-100 dark:bg-blue-900/30' : 'bg-gray-200 dark:bg-gray-700/50'}`;
    },
    previewBox: (isActive: boolean) => {
      if (isDark) {
        return `rounded-sm border ${isActive ? 'border-flow-accent/60 bg-flow-accent/10' : 'border-gray-500/60 bg-gray-900/40'}`;
      }
      return `rounded-sm border ${isActive ? 'border-blue-400/60 bg-blue-50 dark:bg-blue-900/20' : 'border-gray-400/60 dark:border-gray-500/60 bg-white dark:bg-gray-900/40'}`;
    },
    label: isDark ? 'text-sm font-medium text-surface' : 'text-sm font-medium text-gray-900 dark:text-white',
    dimensions: isDark ? 'text-xs text-gray-400' : 'text-xs text-gray-500 dark:text-gray-400',
    description: isDark ? 'text-[11px] text-gray-500' : 'text-[11px] text-gray-400 dark:text-gray-500',
    badge: (isActive: boolean) => {
      const base = 'absolute top-2 right-2 text-[10px] font-medium px-1.5 py-0.5 rounded';
      if (isDark) {
        return `${base} ${isActive ? 'bg-flow-accent/20 text-flow-accent' : 'bg-gray-700 text-gray-400'}`;
      }
      return `${base} ${isActive ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400' : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'}`;
    },
    toggleContainer: 'flex items-center justify-between py-3 mt-2 border-t ' + (isDark ? 'border-gray-700' : 'border-gray-200 dark:border-gray-700'),
    toggleLabel: isDark ? 'text-sm text-gray-300' : 'text-sm text-gray-700 dark:text-gray-300',
    toggleDescription: isDark ? 'text-xs text-gray-500' : 'text-xs text-gray-500 dark:text-gray-400',
    toggleButton: (checked: boolean) => {
      const base = 'relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none focus:ring-2';
      if (isDark) {
        return `${base} focus:ring-flow-accent/50 ${checked ? 'bg-flow-accent' : 'bg-gray-600'}`;
      }
      return `${base} focus:ring-blue-500/50 ${checked ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'}`;
    },
    inputLabel: isDark ? 'text-xs text-gray-400 flex flex-col gap-1.5' : 'text-xs text-gray-600 dark:text-gray-400 flex flex-col gap-1.5',
    input: isDark
      ? 'w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50 focus:border-flow-accent'
      : 'w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500',
  };

  const gridCols = columns === 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-2';

  return (
    <div className={`space-y-1 ${className}`}>
      {/* Preset Grid */}
      <div className={`grid ${gridCols} gap-2`}>
        {VIEWPORT_PRESETS.map((preset) => {
          const isActive = !useCustom && activePreset?.id === preset.id;
          const preview = getPreviewDimensions(preset.width, preset.height);
          return (
            <button
              key={preset.id}
              type="button"
              onClick={() => handlePresetSelect(preset)}
              className={styles.card(isActive)}
            >
              {/* Aspect ratio badge */}
              <span className={styles.badge(isActive)}>{preset.ratioLabel}</span>

              {/* Preview box with actual aspect ratio */}
              <div
                className={styles.previewContainer(isActive)}
                style={{ width: 48, height: 48 }}
              >
                <div
                  className={styles.previewBox(isActive)}
                  style={{ width: preview.width, height: preview.height }}
                />
              </div>

              {/* Text content */}
              <div className="flex flex-col items-start min-w-0 pr-8">
                <span className={styles.label}>{preset.label}</span>
                <span className={styles.description}>{preset.description}</span>
              </div>
            </button>
          );
        })}
      </div>

      {/* Custom Dimensions Toggle */}
      {showCustomToggle && onUseCustomChange && (
        <div className={styles.toggleContainer}>
          <div className="flex items-center gap-2">
            <Settings2 size={14} className={isDark ? 'text-gray-500' : 'text-gray-400 dark:text-gray-500'} />
            <div>
              <label className={styles.toggleLabel}>Custom dimensions</label>
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={useCustom}
            onClick={() => onUseCustomChange(!useCustom)}
            className={styles.toggleButton(useCustom)}
          >
            <span
              className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow-sm transition-transform ${
                useCustom ? 'translate-x-[18px]' : 'translate-x-[3px]'
              }`}
            />
          </button>
        </div>
      )}

      {/* Custom Dimension Inputs */}
      {useCustom && (
        <div className="grid grid-cols-2 gap-3 pt-2">
          <label className={styles.inputLabel}>
            Width (px)
            <input
              type="number"
              min={MIN_DIMENSION}
              max={MAX_DIMENSION}
              step={1}
              value={value.width}
              onChange={handleWidthChange}
              className={styles.input}
            />
          </label>
          <label className={styles.inputLabel}>
            Height (px)
            <input
              type="number"
              min={MIN_DIMENSION}
              max={MAX_DIMENSION}
              step={1}
              value={value.height}
              onChange={handleHeightChange}
              className={styles.input}
            />
          </label>
        </div>
      )}
    </div>
  );
}

export default ViewportPicker;
