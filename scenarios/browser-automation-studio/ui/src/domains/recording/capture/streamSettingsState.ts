import { useCallback, useState } from 'react';
import {
  PRESETS,
  loadStoredPreset,
  loadCustomSettings,
  loadShowStats,
  savePreset,
  saveCustomSettings,
  saveShowStats,
  type StreamPreset,
  type StreamSettingsValues,
} from './streamSettingsConfig';

/** Hook to use stream settings with localStorage persistence */
export function useStreamSettings() {
  const [preset, setPreset] = useState<StreamPreset>(loadStoredPreset);
  const [customSettings, setCustomSettings] = useState<StreamSettingsValues>(loadCustomSettings);
  const [showStats, setShowStats] = useState<boolean>(loadShowStats);

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

  return {
    preset,
    settings,
    customSettings,
    showStats,
    setPreset: handlePresetChange,
    setCustomSettings: handleCustomSettingsChange,
    setShowStats: handleShowStatsChange,
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
