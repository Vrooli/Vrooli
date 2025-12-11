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
 * - Custom: User-configurable quality, FPS, and scale settings
 */

import { useCallback, useEffect, useRef, useState } from 'react';

/** Stream quality preset identifiers */
export type StreamPreset = 'fast' | 'balanced' | 'sharp' | 'hidpi' | 'custom';

/** Resolved stream settings values */
export interface StreamSettingsValues {
  quality: number;
  fps: number;
  /** 'css' = 1x scale, 'device' = device pixel ratio */
  scale: 'css' | 'device';
}

/** Preset configurations (excludes 'custom' which uses user-defined values) */
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

/** Custom preset metadata (settings come from user state) */
const CUSTOM_PRESET_META = {
  label: 'Custom',
  description: 'Configure your own settings',
};

/** Default custom settings (used when switching to custom for the first time) */
const DEFAULT_CUSTOM_SETTINGS: StreamSettingsValues = { quality: 55, fps: 20, scale: 'css' };

const STORAGE_KEY = 'browser-automation-studio:stream-preset';
const CUSTOM_SETTINGS_STORAGE_KEY = 'browser-automation-studio:stream-custom-settings';
const SHOW_STATS_STORAGE_KEY = 'browser-automation-studio:show-stream-stats';
const DEBUG_PERF_STORAGE_KEY = 'browser-automation-studio:debug-perf-mode';
const DEFAULT_PRESET: StreamPreset = 'balanced';
const DEFAULT_SHOW_STATS = false;
const DEFAULT_DEBUG_PERF = false;

/** Load preset from localStorage */
function loadStoredPreset(): StreamPreset {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && (stored in PRESETS || stored === 'custom')) {
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

/** Load custom settings from localStorage */
function loadCustomSettings(): StreamSettingsValues {
  try {
    const stored = localStorage.getItem(CUSTOM_SETTINGS_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Partial<StreamSettingsValues>;
      return {
        quality: typeof parsed.quality === 'number' ? Math.min(100, Math.max(1, parsed.quality)) : DEFAULT_CUSTOM_SETTINGS.quality,
        fps: typeof parsed.fps === 'number' ? Math.min(60, Math.max(1, parsed.fps)) : DEFAULT_CUSTOM_SETTINGS.fps,
        scale: parsed.scale === 'device' ? 'device' : 'css',
      };
    }
  } catch {
    // localStorage may be unavailable or invalid JSON
  }
  return DEFAULT_CUSTOM_SETTINGS;
}

/** Save custom settings to localStorage */
function saveCustomSettings(settings: StreamSettingsValues): void {
  try {
    localStorage.setItem(CUSTOM_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
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

/** Load debugPerfMode preference from localStorage */
function loadDebugPerfMode(): boolean {
  try {
    const stored = localStorage.getItem(DEBUG_PERF_STORAGE_KEY);
    if (stored !== null) {
      return stored === 'true';
    }
  } catch {
    // localStorage may be unavailable
  }
  return DEFAULT_DEBUG_PERF;
}

/** Save debugPerfMode preference to localStorage */
function saveDebugPerfMode(enabled: boolean): void {
  try {
    localStorage.setItem(DEBUG_PERF_STORAGE_KEY, String(enabled));
  } catch {
    // localStorage may be unavailable
  }
}

interface StreamSettingsProps {
  /** Current session ID - enables live settings update when dropdown closes */
  sessionId?: string | null;
  /** @deprecated Use sessionId instead. Shows "applies on next session" hint */
  hasActiveSession?: boolean;
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
  /** Whether debug performance mode is enabled (controlled) */
  debugPerfMode?: boolean;
  /** Callback when debugPerfMode changes (controlled) */
  onDebugPerfModeChange?: (enabled: boolean) => void;
}

export function StreamSettings({
  sessionId,
  hasActiveSession,
  onSettingsChange,
  preset: controlledPreset,
  onPresetChange,
  showStats: controlledShowStats,
  onShowStatsChange,
  debugPerfMode: controlledDebugPerfMode,
  onDebugPerfModeChange,
}: StreamSettingsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [internalPreset, setInternalPreset] = useState<StreamPreset>(loadStoredPreset);
  const [internalShowStats, setInternalShowStats] = useState<boolean>(loadShowStats);
  const [internalDebugPerfMode, setInternalDebugPerfMode] = useState<boolean>(loadDebugPerfMode);
  const [customSettings, setCustomSettings] = useState<StreamSettingsValues>(loadCustomSettings);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Track settings when dropdown opens for change detection
  const settingsOnOpenRef = useRef<{ quality: number; fps: number } | null>(null);
  // Track if we're currently updating settings (to show loading state)
  const [isUpdating, setIsUpdating] = useState(false);

  // Derive effective hasActiveSession from sessionId if not explicitly provided
  const effectiveHasActiveSession = hasActiveSession ?? !!sessionId;

  // Use controlled preset if provided, otherwise internal state
  const preset = controlledPreset ?? internalPreset;
  const setPreset = onPresetChange ?? setInternalPreset;

  // Use controlled showStats if provided, otherwise internal state
  const showStats = controlledShowStats ?? internalShowStats;
  const setShowStats = onShowStatsChange ?? setInternalShowStats;

  // Use controlled debugPerfMode if provided, otherwise internal state
  const debugPerfMode = controlledDebugPerfMode ?? internalDebugPerfMode;
  const setDebugPerfMode = onDebugPerfModeChange ?? setInternalDebugPerfMode;

  // Get current effective settings based on preset
  const getCurrentSettings = useCallback((): StreamSettingsValues => {
    if (preset === 'custom') {
      return customSettings;
    }
    return PRESETS[preset].settings;
  }, [preset, customSettings]);

  // Notify parent of initial settings on mount
  useEffect(() => {
    onSettingsChange(getCurrentSettings());
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Update live stream settings when dropdown closes (if settings changed)
  const updateLiveStreamSettings = useCallback(async (settings: { quality: number; fps: number }) => {
    if (!sessionId) return;

    setIsUpdating(true);
    try {
      const response = await fetch(`/api/v1/recordings/live/${sessionId}/stream-settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          quality: settings.quality,
          fps: settings.fps,
        }),
      });

      if (!response.ok) {
        console.warn('Failed to update stream settings:', await response.text());
      }
    } catch (err) {
      console.warn('Failed to update stream settings:', err);
    } finally {
      setIsUpdating(false);
    }
  }, [sessionId]);

  // Handle dropdown open/close with live update
  const handleDropdownToggle = useCallback(() => {
    if (!isOpen) {
      // Opening: capture current settings
      const current = getCurrentSettings();
      settingsOnOpenRef.current = { quality: current.quality, fps: current.fps };
      setIsOpen(true);
    } else {
      // Closing: check if settings changed and update live
      const current = getCurrentSettings();
      const prev = settingsOnOpenRef.current;
      if (sessionId && prev && (prev.quality !== current.quality || prev.fps !== current.fps)) {
        void updateLiveStreamSettings({ quality: current.quality, fps: current.fps });
      }
      settingsOnOpenRef.current = null;
      setIsOpen(false);
    }
  }, [isOpen, getCurrentSettings, sessionId, updateLiveStreamSettings]);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        // Same logic as handleDropdownToggle close
        const current = getCurrentSettings();
        const prev = settingsOnOpenRef.current;
        if (sessionId && prev && (prev.quality !== current.quality || prev.fps !== current.fps)) {
          void updateLiveStreamSettings({ quality: current.quality, fps: current.fps });
        }
        settingsOnOpenRef.current = null;
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen, getCurrentSettings, sessionId, updateLiveStreamSettings]);

  const handlePresetSelect = useCallback(
    (newPreset: StreamPreset) => {
      setPreset(newPreset);
      savePreset(newPreset);
      if (newPreset === 'custom') {
        onSettingsChange(customSettings);
      } else {
        onSettingsChange(PRESETS[newPreset].settings);
      }
      // Don't close dropdown when selecting custom - allow user to configure
      if (newPreset !== 'custom') {
        setIsOpen(false);
      }
    },
    [setPreset, onSettingsChange, customSettings]
  );

  const handleCustomSettingChange = useCallback(
    (key: keyof StreamSettingsValues, value: number | 'css' | 'device') => {
      const newSettings = { ...customSettings, [key]: value };
      setCustomSettings(newSettings);
      saveCustomSettings(newSettings);
      // Only notify parent if we're currently on custom preset
      if (preset === 'custom') {
        onSettingsChange(newSettings);
      }
    },
    [customSettings, preset, onSettingsChange]
  );

  const handleShowStatsToggle = useCallback(() => {
    const newValue = !showStats;
    setShowStats(newValue);
    saveShowStats(newValue);
  }, [showStats, setShowStats]);

  // Handle debug perf mode toggle with API call
  const handleDebugPerfModeToggle = useCallback(async () => {
    const newValue = !debugPerfMode;
    setDebugPerfMode(newValue);
    saveDebugPerfMode(newValue);

    // Send to API if we have an active session
    if (sessionId) {
      try {
        const response = await fetch(`/api/v1/recordings/live/${sessionId}/stream-settings`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ perfMode: newValue }),
        });
        if (!response.ok) {
          console.warn('Failed to update perfMode:', await response.text());
        }
      } catch (err) {
        console.warn('Failed to update perfMode:', err);
      }
    }
  }, [debugPerfMode, setDebugPerfMode, sessionId]);

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Settings button */}
      <button
        type="button"
        onClick={handleDropdownToggle}
        className={`p-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors ${isUpdating ? 'opacity-50' : ''}`}
        title="Stream quality settings"
        aria-label="Stream quality settings"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        disabled={isUpdating}
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
          className="absolute right-0 top-full mt-1 w-72 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50 overflow-hidden"
          role="listbox"
          aria-label="Stream quality presets"
        >
          <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700">
            <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
              Stream Quality
            </p>
          </div>

          <div className="py-1">
            {/* Standard presets */}
            {(Object.entries(PRESETS) as [Exclude<StreamPreset, 'custom'>, (typeof PRESETS)[Exclude<StreamPreset, 'custom'>]][]).map(
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

            {/* Custom preset option */}
            <button
              type="button"
              role="option"
              aria-selected={preset === 'custom'}
              onClick={() => handlePresetSelect('custom')}
              className={`w-full px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors flex items-start gap-3 ${
                preset === 'custom' ? 'bg-blue-50 dark:bg-blue-900/20' : ''
              }`}
            >
              {/* Radio indicator */}
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

          {/* Custom settings controls - shown when custom preset is selected */}
          {preset === 'custom' && (
            <div className="px-3 py-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 space-y-4">
              {/* Quality slider */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label htmlFor="stream-quality" className="text-xs font-medium text-gray-600 dark:text-gray-300">
                    Quality
                  </label>
                  <span className="text-xs tabular-nums text-gray-500 dark:text-gray-400">
                    {customSettings.quality}%
                  </span>
                </div>
                <input
                  id="stream-quality"
                  type="range"
                  min={10}
                  max={100}
                  step={5}
                  value={customSettings.quality}
                  onChange={(e) => handleCustomSettingChange('quality', parseInt(e.target.value, 10))}
                  className="w-full h-1.5 bg-gray-200 dark:bg-gray-600 rounded-full appearance-none cursor-pointer accent-blue-500"
                />
                <div className="flex justify-between text-[10px] text-gray-400 dark:text-gray-500 mt-0.5">
                  <span>Faster</span>
                  <span>Sharper</span>
                </div>
              </div>

              {/* FPS slider */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label htmlFor="stream-fps" className="text-xs font-medium text-gray-600 dark:text-gray-300">
                    Frame Rate
                  </label>
                  <span className="text-xs tabular-nums text-gray-500 dark:text-gray-400">
                    {customSettings.fps} fps
                  </span>
                </div>
                <input
                  id="stream-fps"
                  type="range"
                  min={1}
                  max={60}
                  step={1}
                  value={customSettings.fps}
                  onChange={(e) => handleCustomSettingChange('fps', parseInt(e.target.value, 10))}
                  className="w-full h-1.5 bg-gray-200 dark:bg-gray-600 rounded-full appearance-none cursor-pointer accent-blue-500"
                />
                <div className="flex justify-between text-[10px] text-gray-400 dark:text-gray-500 mt-0.5">
                  <span>Less bandwidth</span>
                  <span>Smoother</span>
                </div>
              </div>

              {/* Scale selector */}
              <div>
                <label className="text-xs font-medium text-gray-600 dark:text-gray-300 block mb-1.5">
                  Resolution
                </label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => handleCustomSettingChange('scale', 'css')}
                    className={`flex-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                      customSettings.scale === 'css'
                        ? 'bg-blue-500 text-white'
                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'
                    }`}
                  >
                    Standard (1x)
                  </button>
                  <button
                    type="button"
                    onClick={() => handleCustomSettingChange('scale', 'device')}
                    className={`flex-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                      customSettings.scale === 'device'
                        ? 'bg-blue-500 text-white'
                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'
                    }`}
                  >
                    HiDPI (2x)
                  </button>
                </div>
                <p className="text-[10px] text-gray-400 dark:text-gray-500 mt-1">
                  {customSettings.scale === 'device' ? 'Crisp on Retina, uses more bandwidth' : 'Good for most displays'}
                </p>
              </div>
            </div>
          )}

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

          {/* Debug performance mode toggle */}
          <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={handleDebugPerfModeToggle}
              className="w-full flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 -mx-1 px-1 py-1 rounded transition-colors"
            >
              <div className="flex flex-col items-start">
                <span className="text-sm text-gray-700 dark:text-gray-200">Debug performance</span>
                <span className="text-xs text-gray-400 dark:text-gray-500">Instrument frame timing pipeline</span>
              </div>
              <span
                className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out ${
                  debugPerfMode ? 'bg-orange-500' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    debugPerfMode ? 'translate-x-4' : 'translate-x-0'
                  }`}
                />
              </span>
            </button>
          </div>

          {/* Application hint */}
          {effectiveHasActiveSession && (
            <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-blue-50 dark:bg-blue-900/20">
              <p className="text-xs text-blue-700 dark:text-blue-300 flex items-center gap-1.5">
                <svg className="w-3.5 h-3.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                    clipRule="evenodd"
                  />
                </svg>
                {sessionId ? 'Quality & FPS apply on close. Scale requires new session.' : 'Applies on next session'}
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
  const [customSettings, setCustomSettings] = useState<StreamSettingsValues>(loadCustomSettings);
  const [showStats, setShowStats] = useState<boolean>(loadShowStats);
  const [debugPerfMode, setDebugPerfMode] = useState<boolean>(loadDebugPerfMode);

  // Compute effective settings based on preset
  const settings = preset === 'custom' ? customSettings : PRESETS[preset].settings;

  const handlePresetChange = useCallback((newPreset: StreamPreset) => {
    setPreset(newPreset);
    savePreset(newPreset);
  }, []);

  const handleCustomSettingsChange = useCallback((newSettings: StreamSettingsValues) => {
    setCustomSettings(newSettings);
    saveCustomSettings(newSettings);
  }, []);

  const handleShowStatsChange = useCallback((show: boolean) => {
    setShowStats(show);
    saveShowStats(show);
  }, []);

  const handleDebugPerfModeChange = useCallback((enabled: boolean) => {
    setDebugPerfMode(enabled);
    saveDebugPerfMode(enabled);
  }, []);

  return {
    preset,
    settings,
    customSettings,
    showStats,
    debugPerfMode,
    setPreset: handlePresetChange,
    setCustomSettings: handleCustomSettingsChange,
    setShowStats: handleShowStatsChange,
    setDebugPerfMode: handleDebugPerfModeChange,
  };
}

/** Get settings for a preset */
export function getPresetSettings(preset: StreamPreset): StreamSettingsValues {
  if (preset === 'custom') {
    return loadCustomSettings();
  }
  return PRESETS[preset].settings;
}

/** Export preset keys for type safety */
export const STREAM_PRESETS = PRESETS;
