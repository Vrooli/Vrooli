/**
 * StreamSettings Component
 *
 * Allows users to configure stream quality settings for the live preview.
 * Settings are applied when a new session is created (not mid-stream).
 *
 * Presets:
 * - Fast: Lower quality, lower FPS - good for slow connections
 * - Balanced: Default settings - good balance of quality and performance
 * - Sharp: Higher quality - for when you need crisp visuals
 * - HiDPI: Device pixel ratio capture - for Retina/HiDPI displays
 */

import { useCallback, useEffect, useRef, useState } from 'react';

/** Stream quality preset identifiers */
export type StreamPreset = 'fast' | 'balanced' | 'sharp' | 'hidpi';

/** Resolved stream settings values */
export interface StreamSettingsValues {
  quality: number;
  fps: number;
  /** 'css' = 1x scale, 'device' = device pixel ratio */
  scale: 'css' | 'device';
}

/** Preset configurations */
const PRESETS: Record<StreamPreset, { label: string; description: string; settings: StreamSettingsValues }> = {
  fast: {
    label: 'Fast',
    description: 'Lower quality, faster streaming',
    settings: { quality: 40, fps: 4, scale: 'css' },
  },
  balanced: {
    label: 'Balanced',
    description: 'Good quality and performance',
    settings: { quality: 55, fps: 6, scale: 'css' },
  },
  sharp: {
    label: 'Sharp',
    description: 'Higher quality preview',
    settings: { quality: 70, fps: 6, scale: 'css' },
  },
  hidpi: {
    label: 'HiDPI',
    description: 'Crisp on Retina displays',
    settings: { quality: 60, fps: 4, scale: 'device' },
  },
};

const STORAGE_KEY = 'browser-automation-studio:stream-preset';
const SHOW_STATS_STORAGE_KEY = 'browser-automation-studio:show-stream-stats';
const DEFAULT_PRESET: StreamPreset = 'balanced';
const DEFAULT_SHOW_STATS = false;

/** Load preset from localStorage */
function loadStoredPreset(): StreamPreset {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && stored in PRESETS) {
      return stored as StreamPreset;
    }
  } catch {
    // localStorage may be unavailable
  }
  return DEFAULT_PRESET;
}

/** Save preset to localStorage */
function savePreset(preset: StreamPreset): void {
  try {
    localStorage.setItem(STORAGE_KEY, preset);
  } catch {
    // localStorage may be unavailable
  }
}

/** Load showStats preference from localStorage */
function loadShowStats(): boolean {
  try {
    const stored = localStorage.getItem(SHOW_STATS_STORAGE_KEY);
    if (stored !== null) {
      return stored === 'true';
    }
  } catch {
    // localStorage may be unavailable
  }
  return DEFAULT_SHOW_STATS;
}

/** Save showStats preference to localStorage */
function saveShowStats(show: boolean): void {
  try {
    localStorage.setItem(SHOW_STATS_STORAGE_KEY, String(show));
  } catch {
    // localStorage may be unavailable
  }
}

interface StreamSettingsProps {
  /** Current session ID - used to show "applies on next session" hint */
  hasActiveSession: boolean;
  /** Callback when settings change */
  onSettingsChange: (settings: StreamSettingsValues) => void;
  /** Current preset (controlled) */
  preset?: StreamPreset;
  /** Callback when preset changes (controlled) */
  onPresetChange?: (preset: StreamPreset) => void;
  /** Whether to show performance stats (controlled) */
  showStats?: boolean;
  /** Callback when showStats changes (controlled) */
  onShowStatsChange?: (show: boolean) => void;
}

export function StreamSettings({
  hasActiveSession,
  onSettingsChange,
  preset: controlledPreset,
  onPresetChange,
  showStats: controlledShowStats,
  onShowStatsChange,
}: StreamSettingsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [internalPreset, setInternalPreset] = useState<StreamPreset>(loadStoredPreset);
  const [internalShowStats, setInternalShowStats] = useState<boolean>(loadShowStats);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Use controlled preset if provided, otherwise internal state
  const preset = controlledPreset ?? internalPreset;
  const setPreset = onPresetChange ?? setInternalPreset;

  // Use controlled showStats if provided, otherwise internal state
  const showStats = controlledShowStats ?? internalShowStats;
  const setShowStats = onShowStatsChange ?? setInternalShowStats;

  // Notify parent of initial settings on mount
  useEffect(() => {
    onSettingsChange(PRESETS[preset].settings);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handlePresetSelect = useCallback(
    (newPreset: StreamPreset) => {
      setPreset(newPreset);
      savePreset(newPreset);
      onSettingsChange(PRESETS[newPreset].settings);
      setIsOpen(false);
    },
    [setPreset, onSettingsChange]
  );

  const handleShowStatsToggle = useCallback(() => {
    const newValue = !showStats;
    setShowStats(newValue);
    saveShowStats(newValue);
  }, [showStats, setShowStats]);

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Settings button */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="p-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
        title="Stream quality settings"
        aria-label="Stream quality settings"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
          />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className="absolute right-0 top-full mt-1 w-64 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50 overflow-hidden"
          role="listbox"
          aria-label="Stream quality presets"
        >
          <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700">
            <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
              Stream Quality
            </p>
          </div>

          <div className="py-1">
            {(Object.entries(PRESETS) as [StreamPreset, (typeof PRESETS)[StreamPreset]][]).map(
              ([presetKey, config]) => (
                <button
                  key={presetKey}
                  type="button"
                  role="option"
                  aria-selected={preset === presetKey}
                  onClick={() => handlePresetSelect(presetKey)}
                  className={`w-full px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors flex items-start gap-3 ${
                    preset === presetKey ? 'bg-blue-50 dark:bg-blue-900/20' : ''
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
          </div>

          {/* Show stats toggle */}
          <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={handleShowStatsToggle}
              className="w-full flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 -mx-1 px-1 py-1 rounded transition-colors"
            >
              <span className="text-sm text-gray-700 dark:text-gray-200">Show performance stats</span>
              <span
                className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out ${
                  showStats ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    showStats ? 'translate-x-4' : 'translate-x-0'
                  }`}
                />
              </span>
            </button>
          </div>

          {/* Application hint */}
          {hasActiveSession && (
            <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-amber-50 dark:bg-amber-900/20">
              <p className="text-xs text-amber-700 dark:text-amber-300 flex items-center gap-1.5">
                <svg className="w-3.5 h-3.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
                Applies on next session
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/** Hook to use stream settings with localStorage persistence */
export function useStreamSettings() {
  const [preset, setPreset] = useState<StreamPreset>(loadStoredPreset);
  const [settings, setSettings] = useState<StreamSettingsValues>(PRESETS[loadStoredPreset()].settings);
  const [showStats, setShowStats] = useState<boolean>(loadShowStats);

  const handlePresetChange = useCallback((newPreset: StreamPreset) => {
    setPreset(newPreset);
    setSettings(PRESETS[newPreset].settings);
    savePreset(newPreset);
  }, []);

  const handleShowStatsChange = useCallback((show: boolean) => {
    setShowStats(show);
    saveShowStats(show);
  }, []);

  return {
    preset,
    settings,
    showStats,
    setPreset: handlePresetChange,
    setSettings,
    setShowStats: handleShowStatsChange,
  };
}

/** Get settings for a preset */
export function getPresetSettings(preset: StreamPreset): StreamSettingsValues {
  return PRESETS[preset].settings;
}

/** Export preset keys for type safety */
export const STREAM_PRESETS = PRESETS;
