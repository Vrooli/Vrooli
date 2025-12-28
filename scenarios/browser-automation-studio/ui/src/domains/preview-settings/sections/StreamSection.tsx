/**
 * StreamSection Component
 *
 * Stream quality settings for the live preview. Includes:
 * - Quality presets (fast, balanced, sharp, hidpi, custom)
 * - Custom quality/FPS sliders
 * - Resolution toggle (Standard/HiDPI)
 * - Performance stats toggle
 */

import { useCallback } from 'react';
import type { StreamPreset, StreamSettingsValues } from '@/domains/recording/capture/StreamSettings';

/** Preset configurations */
const PRESETS: Record<Exclude<StreamPreset, 'custom'>, { label: string; description: string; settings: StreamSettingsValues }> = {
  fast: {
    label: 'Fast',
    description: 'Lower quality, faster streaming',
    settings: { quality: 40, fps: 10, scale: 'css' },
  },
  balanced: {
    label: 'Balanced',
    description: 'Good quality and performance',
    settings: { quality: 55, fps: 20, scale: 'css' },
  },
  sharp: {
    label: 'Sharp',
    description: 'Higher quality preview',
    settings: { quality: 70, fps: 15, scale: 'css' },
  },
  hidpi: {
    label: 'HiDPI',
    description: 'Crisp on Retina displays',
    settings: { quality: 60, fps: 30, scale: 'device' },
  },
};

const CUSTOM_PRESET_META = {
  label: 'Custom',
  description: 'Configure your own settings',
};

interface StreamSectionProps {
  /** Current preset selection */
  preset: StreamPreset;
  /** Current stream settings values */
  settings: StreamSettingsValues;
  /** Whether performance stats are shown */
  showStats: boolean;
  /** Whether there's an active session (affects resolution hint) */
  hasActiveSession: boolean;
  /** Callback when preset changes */
  onPresetChange: (preset: StreamPreset) => void;
  /** Callback when custom settings change */
  onSettingsChange: (settings: StreamSettingsValues) => void;
  /** Callback when show stats toggles */
  onShowStatsChange: (show: boolean) => void;
}

export function StreamSection({
  preset,
  settings,
  showStats,
  hasActiveSession,
  onPresetChange,
  onSettingsChange,
  onShowStatsChange,
}: StreamSectionProps) {
  const handlePresetSelect = useCallback(
    (newPreset: StreamPreset) => {
      onPresetChange(newPreset);
      if (newPreset !== 'custom') {
        onSettingsChange(PRESETS[newPreset].settings);
      }
    },
    [onPresetChange, onSettingsChange]
  );

  const handleCustomSettingChange = useCallback(
    (key: keyof StreamSettingsValues, value: number | 'css' | 'device') => {
      onSettingsChange({ ...settings, [key]: value });
    },
    [settings, onSettingsChange]
  );

  return (
    <div className="space-y-6">
      {/* Preset Selection */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
          Quality Preset
        </h3>
        <div className="space-y-2">
          {(Object.entries(PRESETS) as [Exclude<StreamPreset, 'custom'>, typeof PRESETS[Exclude<StreamPreset, 'custom'>]][]).map(
            ([presetKey, config]) => (
              <button
                key={presetKey}
                type="button"
                onClick={() => handlePresetSelect(presetKey)}
                className={`w-full px-4 py-3 text-left rounded-lg border transition-colors flex items-start gap-3 ${
                  preset === presetKey
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                {/* Radio indicator */}
                <span
                  className={`mt-0.5 w-4 h-4 rounded-full border-2 flex-shrink-0 flex items-center justify-center ${
                    preset === presetKey
                      ? 'border-blue-500 bg-blue-500'
                      : 'border-gray-300 dark:border-gray-600'
                  }`}
                >
                  {preset === presetKey && <span className="w-1.5 h-1.5 rounded-full bg-white" />}
                </span>

                <div className="flex-1 min-w-0">
                  <p
                    className={`text-sm font-medium ${
                      preset === presetKey
                        ? 'text-blue-700 dark:text-blue-300'
                        : 'text-gray-900 dark:text-gray-100'
                    }`}
                  >
                    {config.label}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{config.description}</p>
                </div>
              </button>
            )
          )}

          {/* Custom preset option */}
          <button
            type="button"
            onClick={() => handlePresetSelect('custom')}
            className={`w-full px-4 py-3 text-left rounded-lg border transition-colors flex items-start gap-3 ${
              preset === 'custom'
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
            }`}
          >
            <span
              className={`mt-0.5 w-4 h-4 rounded-full border-2 flex-shrink-0 flex items-center justify-center ${
                preset === 'custom'
                  ? 'border-blue-500 bg-blue-500'
                  : 'border-gray-300 dark:border-gray-600'
              }`}
            >
              {preset === 'custom' && <span className="w-1.5 h-1.5 rounded-full bg-white" />}
            </span>

            <div className="flex-1 min-w-0">
              <p
                className={`text-sm font-medium ${
                  preset === 'custom'
                    ? 'text-blue-700 dark:text-blue-300'
                    : 'text-gray-900 dark:text-gray-100'
                }`}
              >
                {CUSTOM_PRESET_META.label}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">{CUSTOM_PRESET_META.description}</p>
            </div>
          </button>
        </div>
      </div>

      {/* Custom settings controls */}
      {preset === 'custom' && (
        <div className="space-y-5 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
          {/* Quality slider */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="stream-quality" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Quality
              </label>
              <span className="text-sm tabular-nums text-gray-500 dark:text-gray-400">
                {settings.quality}%
              </span>
            </div>
            <input
              id="stream-quality"
              type="range"
              min={10}
              max={100}
              step={5}
              value={settings.quality}
              onChange={(e) => handleCustomSettingChange('quality', parseInt(e.target.value, 10))}
              className="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-full appearance-none cursor-pointer accent-blue-500"
            />
            <div className="flex justify-between text-xs text-gray-400 dark:text-gray-500 mt-1">
              <span>Faster</span>
              <span>Sharper</span>
            </div>
          </div>

          {/* FPS slider */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="stream-fps" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Frame Rate
              </label>
              <span className="text-sm tabular-nums text-gray-500 dark:text-gray-400">
                {settings.fps} fps
              </span>
            </div>
            <input
              id="stream-fps"
              type="range"
              min={1}
              max={60}
              step={1}
              value={settings.fps}
              onChange={(e) => handleCustomSettingChange('fps', parseInt(e.target.value, 10))}
              className="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-full appearance-none cursor-pointer accent-blue-500"
            />
            <div className="flex justify-between text-xs text-gray-400 dark:text-gray-500 mt-1">
              <span>Less bandwidth</span>
              <span>Smoother</span>
            </div>
          </div>

          {/* Scale selector */}
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-2">
              Resolution
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => handleCustomSettingChange('scale', 'css')}
                className={`flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                  settings.scale === 'css'
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'
                }`}
              >
                Standard (1x)
              </button>
              <button
                type="button"
                onClick={() => handleCustomSettingChange('scale', 'device')}
                className={`flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                  settings.scale === 'device'
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'
                }`}
              >
                HiDPI (2x)
              </button>
            </div>
            <p className="text-xs text-gray-400 dark:text-gray-500 mt-2">
              {settings.scale === 'device' ? 'Crisp on Retina, uses more bandwidth' : 'Good for most displays'}
            </p>
          </div>
        </div>
      )}

      {/* Divider */}
      <div className="border-t border-gray-200 dark:border-gray-700" />

      {/* Performance stats toggle */}
      <div>
        <button
          type="button"
          onClick={() => onShowStatsChange(!showStats)}
          className="w-full flex items-center justify-between px-4 py-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
        >
          <div className="flex flex-col items-start">
            <span className="text-sm font-medium text-gray-700 dark:text-gray-200">Show performance stats</span>
            <span className="text-xs text-gray-400 dark:text-gray-500">FPS, latency, and bottleneck analysis</span>
          </div>
          <span
            className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out ${
              showStats ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
            }`}
          >
            <span
              className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                showStats ? 'translate-x-5' : 'translate-x-0'
              }`}
            />
          </span>
        </button>
      </div>

      {/* Active session hint for resolution */}
      {hasActiveSession && (
        <div className="flex items-center gap-2 px-3 py-2 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
          <svg className="w-4 h-4 flex-shrink-0 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
              clipRule="evenodd"
            />
          </svg>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Resolution (Standard/HiDPI) changes require a new session
          </p>
        </div>
      )}
    </div>
  );
}
