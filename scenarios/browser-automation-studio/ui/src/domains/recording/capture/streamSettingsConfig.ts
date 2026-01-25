export type StreamPreset = 'fast' | 'balanced' | 'sharp' | 'hidpi' | 'custom';

export interface StreamSettingsValues {
  quality: number;
  fps: number;
  /** 'css' = 1x scale, 'device' = device pixel ratio */
  scale: 'css' | 'device';
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

export const PRESETS: Record<Exclude<StreamPreset, 'custom'>, { label: string; description: string; settings: StreamSettingsValues }> = {
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

export const CUSTOM_PRESET_META = {
  label: 'Custom',
  description: 'Configure your own settings',
};

export const DEFAULT_CUSTOM_SETTINGS: StreamSettingsValues = { quality: 55, fps: 20, scale: 'css' };

const STORAGE_KEY = 'browser-automation-studio:stream-preset';
const CUSTOM_SETTINGS_STORAGE_KEY = 'browser-automation-studio:stream-custom-settings';
const SHOW_STATS_STORAGE_KEY = 'browser-automation-studio:show-stream-stats';
const DEFAULT_PRESET: StreamPreset = 'balanced';
const DEFAULT_SHOW_STATS = true;

export function loadStoredPreset(): StreamPreset {
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

export function savePreset(preset: StreamPreset): void {
  try {
    localStorage.setItem(STORAGE_KEY, preset);
  } catch {
    // localStorage may be unavailable
  }
}

export function loadCustomSettings(): StreamSettingsValues {
  try {
    const stored = localStorage.getItem(CUSTOM_SETTINGS_STORAGE_KEY);
    if (stored) {
      const parsed: unknown = JSON.parse(stored);
      if (!isRecord(parsed)) {
        return DEFAULT_CUSTOM_SETTINGS;
      }
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

export function saveCustomSettings(settings: StreamSettingsValues): void {
  try {
    localStorage.setItem(CUSTOM_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // localStorage may be unavailable
  }
}

export function loadShowStats(): boolean {
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

export function saveShowStats(show: boolean): void {
  try {
    localStorage.setItem(SHOW_STATS_STORAGE_KEY, String(show));
  } catch {
    // localStorage may be unavailable
  }
}
