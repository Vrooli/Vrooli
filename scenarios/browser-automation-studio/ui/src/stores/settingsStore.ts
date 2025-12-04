import { create } from 'zustand';
import type {
  ReplayChromeTheme,
  ReplayBackgroundTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
  CursorSpeedProfile,
  CursorPathStyle,
} from '../features/execution/ReplayPlayer';
import {
  isReplayChromeTheme,
  isReplayBackgroundTheme,
  isReplayCursorTheme,
  isReplayCursorInitialPosition,
  isReplayCursorClickAnimation,
  CURSOR_SCALE_MIN,
  CURSOR_SCALE_MAX,
  REPLAY_CHROME_OPTIONS,
  REPLAY_BACKGROUND_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
} from '../features/execution/replay/replayThemeOptions';

const STORAGE_PREFIX = 'browserAutomation.settings.';
const PRESETS_STORAGE_KEY = `${STORAGE_PREFIX}replay.userPresets`;
const WORKFLOW_DEFAULTS_KEY = `${STORAGE_PREFIX}workflowDefaults`;
const API_KEYS_KEY = `${STORAGE_PREFIX}apiKeys`;
const DISPLAY_SETTINGS_KEY = `${STORAGE_PREFIX}display`;

// Storage helpers with error handling
const safeGetItem = (key: string): string | null => {
  try {
    if (typeof window === 'undefined') return null;
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
};

const safeSetItem = (key: string, value: string): void => {
  try {
    if (typeof window === 'undefined') return;
    window.localStorage.setItem(key, value);
  } catch {
    // Ignore storage errors (e.g., quota exceeded, private mode)
  }
};

// Validation helpers
const isValidSpeedProfile = (value: unknown): value is CursorSpeedProfile => {
  return ['instant', 'linear', 'easeIn', 'easeOut', 'easeInOut'].includes(value as string);
};

const isValidPathStyle = (value: unknown): value is CursorPathStyle => {
  return ['linear', 'parabolicUp', 'parabolicDown', 'cubic', 'pseudorandom'].includes(value as string);
};

const clampCursorScale = (value: number): number => {
  return Math.min(CURSOR_SCALE_MAX, Math.max(CURSOR_SCALE_MIN, value));
};

// Default values
const DEFAULT_CHROME_THEME: ReplayChromeTheme = 'aurora';
const DEFAULT_BACKGROUND_THEME: ReplayBackgroundTheme = 'aurora';
const DEFAULT_CURSOR_THEME: ReplayCursorTheme = 'white';
const DEFAULT_CURSOR_POSITION: ReplayCursorInitialPosition = 'center';
const DEFAULT_CURSOR_SCALE = 1;
const DEFAULT_CLICK_ANIMATION: ReplayCursorClickAnimation = 'pulse';
const DEFAULT_SPEED_PROFILE: CursorSpeedProfile = 'easeInOut';
const DEFAULT_PATH_STYLE: CursorPathStyle = 'linear';
const DEFAULT_FRAME_DURATION = 1600;

// Default branding settings
const DEFAULT_WATERMARK_SETTINGS: WatermarkSettings = {
  enabled: false,
  assetId: null,
  position: 'bottom-right',
  size: 12,
  opacity: 80,
  margin: 16,
};

const DEFAULT_INTRO_CARD_SETTINGS: IntroCardSettings = {
  enabled: false,
  title: '',
  subtitle: '',
  logoAssetId: null,
  backgroundAssetId: null,
  backgroundColor: '#0f172a',
  textColor: '#ffffff',
  duration: 2000,
};

const DEFAULT_OUTRO_CARD_SETTINGS: OutroCardSettings = {
  enabled: false,
  title: 'Thanks for watching!',
  ctaText: 'Learn More',
  ctaUrl: '',
  logoAssetId: null,
  backgroundAssetId: null,
  backgroundColor: '#0f172a',
  textColor: '#ffffff',
  duration: 3000,
};

// Watermark position options
export type WatermarkPosition = 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right' | 'center';

export interface WatermarkSettings {
  enabled: boolean;
  assetId: string | null;
  position: WatermarkPosition;
  size: number; // 5-30 (percentage of container width)
  opacity: number; // 0-100
  margin: number; // Pixels from edge (8-48)
}

export interface IntroCardSettings {
  enabled: boolean;
  title: string;
  subtitle: string;
  logoAssetId: string | null;
  backgroundAssetId: string | null;
  backgroundColor: string; // Hex color fallback
  textColor: string; // Hex color
  duration: number; // ms (1000-5000)
}

export interface OutroCardSettings {
  enabled: boolean;
  title: string;
  ctaText: string;
  ctaUrl: string;
  logoAssetId: string | null;
  backgroundAssetId: string | null;
  backgroundColor: string;
  textColor: string;
  duration: number; // ms (2000-8000)
}

export interface ReplaySettings {
  chromeTheme: ReplayChromeTheme;
  backgroundTheme: ReplayBackgroundTheme;
  cursorTheme: ReplayCursorTheme;
  cursorInitialPosition: ReplayCursorInitialPosition;
  cursorScale: number;
  cursorClickAnimation: ReplayCursorClickAnimation;
  cursorSpeedProfile: CursorSpeedProfile;
  cursorPathStyle: CursorPathStyle;
  frameDuration: number;
  autoPlay: boolean;
  loop: boolean;
  watermark: WatermarkSettings;
  introCard: IntroCardSettings;
  outroCard: OutroCardSettings;
}

export interface WorkflowDefaultSettings {
  defaultTimeout: number; // in seconds
  stepTimeout: number; // in seconds
  retryAttempts: number;
  retryDelay: number; // in ms
  screenshotOnFailure: boolean;
  screenshotOnSuccess: boolean;
  headless: boolean;
  slowMo: number; // in ms
}

export interface ApiKeySettings {
  browserlessApiKey: string;
  openaiApiKey: string;
  anthropicApiKey: string;
  customApiEndpoint: string;
}

export type ThemeMode = 'light' | 'dark' | 'system';
export type FontSize = 'small' | 'medium' | 'large';
export type FontFamily = 'mono' | 'sans' | 'system';
export type SidebarWidth = 'narrow' | 'default' | 'wide';

export interface DisplaySettings {
  themeMode: ThemeMode;
  fontSize: FontSize;
  fontFamily: FontFamily;
  reducedMotion: boolean;
  highContrast: boolean;
  compactMode: boolean;
  sidebarWidth: SidebarWidth;
}

export interface ReplayPreset {
  id: string;
  name: string;
  settings: ReplaySettings;
  isBuiltIn?: boolean;
  createdAt?: string;
}

// Built-in presets that ship with the application
export const BUILT_IN_PRESETS: ReplayPreset[] = [
  {
    id: 'default',
    name: 'Default',
    isBuiltIn: true,
    settings: {
      chromeTheme: 'aurora',
      backgroundTheme: 'aurora',
      cursorTheme: 'white',
      cursorInitialPosition: 'center',
      cursorScale: 1,
      cursorClickAnimation: 'pulse',
      cursorSpeedProfile: 'easeInOut',
      cursorPathStyle: 'linear',
      frameDuration: 1600,
      autoPlay: false,
      loop: true,
      watermark: { ...DEFAULT_WATERMARK_SETTINGS },
      introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
      outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
    },
  },
  {
    id: 'cinematic',
    name: 'Cinematic',
    isBuiltIn: true,
    settings: {
      chromeTheme: 'midnight',
      backgroundTheme: 'nebula',
      cursorTheme: 'aura',
      cursorInitialPosition: 'center',
      cursorScale: 1.2,
      cursorClickAnimation: 'ripple',
      cursorSpeedProfile: 'easeInOut',
      cursorPathStyle: 'cubic',
      frameDuration: 2500,
      autoPlay: true,
      loop: true,
      watermark: { ...DEFAULT_WATERMARK_SETTINGS },
      introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
      outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
    },
  },
  {
    id: 'minimal',
    name: 'Minimal',
    isBuiltIn: true,
    settings: {
      chromeTheme: 'minimal',
      backgroundTheme: 'charcoal',
      cursorTheme: 'arrowLight',
      cursorInitialPosition: 'top-left',
      cursorScale: 0.9,
      cursorClickAnimation: 'none',
      cursorSpeedProfile: 'linear',
      cursorPathStyle: 'linear',
      frameDuration: 1200,
      autoPlay: false,
      loop: false,
      watermark: { ...DEFAULT_WATERMARK_SETTINGS },
      introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
      outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
    },
  },
  {
    id: 'presentation',
    name: 'Presentation',
    isBuiltIn: true,
    settings: {
      chromeTheme: 'chromium',
      backgroundTheme: 'grid',
      cursorTheme: 'arrowNeon',
      cursorInitialPosition: 'center',
      cursorScale: 1.4,
      cursorClickAnimation: 'pulse',
      cursorSpeedProfile: 'easeOut',
      cursorPathStyle: 'parabolicUp',
      frameDuration: 2000,
      autoPlay: true,
      loop: true,
      watermark: { ...DEFAULT_WATERMARK_SETTINGS },
      introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
      outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
    },
  },
  {
    id: 'ocean-vibes',
    name: 'Ocean Vibes',
    isBuiltIn: true,
    settings: {
      chromeTheme: 'aurora',
      backgroundTheme: 'ocean',
      cursorTheme: 'white',
      cursorInitialPosition: 'bottom-right',
      cursorScale: 1.1,
      cursorClickAnimation: 'ripple',
      cursorSpeedProfile: 'easeInOut',
      cursorPathStyle: 'parabolicDown',
      frameDuration: 2200,
      autoPlay: true,
      loop: true,
      watermark: { ...DEFAULT_WATERMARK_SETTINGS },
      introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
      outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
    },
  },
];

interface SettingsStore {
  replay: ReplaySettings;
  workflowDefaults: WorkflowDefaultSettings;
  apiKeys: ApiKeySettings;
  display: DisplaySettings;
  userPresets: ReplayPreset[];
  activePresetId: string | null;
  setReplaySetting: <K extends keyof ReplaySettings>(key: K, value: ReplaySettings[K]) => void;
  resetReplaySettings: () => void;
  loadPreset: (presetId: string) => void;
  saveAsPreset: (name: string) => ReplayPreset;
  deletePreset: (presetId: string) => void;
  randomizeSettings: () => void;
  getAllPresets: () => ReplayPreset[];
  setWorkflowDefault: <K extends keyof WorkflowDefaultSettings>(key: K, value: WorkflowDefaultSettings[K]) => void;
  resetWorkflowDefaults: () => void;
  setApiKey: <K extends keyof ApiKeySettings>(key: K, value: ApiKeySettings[K]) => void;
  clearApiKeys: () => void;
  setDisplaySetting: <K extends keyof DisplaySettings>(key: K, value: DisplaySettings[K]) => void;
  resetDisplaySettings: () => void;
  getEffectiveTheme: () => 'light' | 'dark';
}

// Load branding settings from localStorage
const loadBrandingSettings = <T extends object>(key: string, defaults: T): T => {
  try {
    const stored = safeGetItem(`${STORAGE_PREFIX}replay.${key}`);
    if (!stored) return defaults;
    const parsed = JSON.parse(stored);
    return { ...defaults, ...parsed };
  } catch {
    return defaults;
  }
};

const loadReplaySettings = (): ReplaySettings => {
  const storedChrome = safeGetItem(`${STORAGE_PREFIX}replay.chromeTheme`);
  const storedBackground = safeGetItem(`${STORAGE_PREFIX}replay.backgroundTheme`);
  const storedCursor = safeGetItem(`${STORAGE_PREFIX}replay.cursorTheme`);
  const storedPosition = safeGetItem(`${STORAGE_PREFIX}replay.cursorInitialPosition`);
  const storedScale = safeGetItem(`${STORAGE_PREFIX}replay.cursorScale`);
  const storedClickAnim = safeGetItem(`${STORAGE_PREFIX}replay.cursorClickAnimation`);
  const storedSpeed = safeGetItem(`${STORAGE_PREFIX}replay.cursorSpeedProfile`);
  const storedPath = safeGetItem(`${STORAGE_PREFIX}replay.cursorPathStyle`);
  const storedDuration = safeGetItem(`${STORAGE_PREFIX}replay.frameDuration`);
  const storedAutoPlay = safeGetItem(`${STORAGE_PREFIX}replay.autoPlay`);
  const storedLoop = safeGetItem(`${STORAGE_PREFIX}replay.loop`);

  return {
    chromeTheme: isReplayChromeTheme(storedChrome) ? storedChrome : DEFAULT_CHROME_THEME,
    backgroundTheme: isReplayBackgroundTheme(storedBackground) ? storedBackground : DEFAULT_BACKGROUND_THEME,
    cursorTheme: isReplayCursorTheme(storedCursor) ? storedCursor : DEFAULT_CURSOR_THEME,
    cursorInitialPosition: isReplayCursorInitialPosition(storedPosition) ? storedPosition : DEFAULT_CURSOR_POSITION,
    cursorScale: storedScale ? clampCursorScale(parseFloat(storedScale)) : DEFAULT_CURSOR_SCALE,
    cursorClickAnimation: isReplayCursorClickAnimation(storedClickAnim) ? storedClickAnim : DEFAULT_CLICK_ANIMATION,
    cursorSpeedProfile: isValidSpeedProfile(storedSpeed) ? storedSpeed : DEFAULT_SPEED_PROFILE,
    cursorPathStyle: isValidPathStyle(storedPath) ? storedPath : DEFAULT_PATH_STYLE,
    frameDuration: storedDuration ? Math.max(800, Math.min(6000, parseInt(storedDuration, 10))) : DEFAULT_FRAME_DURATION,
    autoPlay: storedAutoPlay === 'true',
    loop: storedLoop !== 'false', // Default to true
    watermark: loadBrandingSettings('watermark', DEFAULT_WATERMARK_SETTINGS),
    introCard: loadBrandingSettings('introCard', DEFAULT_INTRO_CARD_SETTINGS),
    outroCard: loadBrandingSettings('outroCard', DEFAULT_OUTRO_CARD_SETTINGS),
  };
};

const saveReplaySetting = <K extends keyof ReplaySettings>(key: K, value: ReplaySettings[K]): void => {
  // For nested objects (watermark, introCard, outroCard), serialize as JSON
  if (typeof value === 'object' && value !== null) {
    safeSetItem(`${STORAGE_PREFIX}replay.${key}`, JSON.stringify(value));
  } else {
    safeSetItem(`${STORAGE_PREFIX}replay.${key}`, String(value));
  }
};

const getDefaultReplaySettings = (): ReplaySettings => ({
  chromeTheme: DEFAULT_CHROME_THEME,
  backgroundTheme: DEFAULT_BACKGROUND_THEME,
  cursorTheme: DEFAULT_CURSOR_THEME,
  cursorInitialPosition: DEFAULT_CURSOR_POSITION,
  cursorScale: DEFAULT_CURSOR_SCALE,
  cursorClickAnimation: DEFAULT_CLICK_ANIMATION,
  cursorSpeedProfile: DEFAULT_SPEED_PROFILE,
  cursorPathStyle: DEFAULT_PATH_STYLE,
  frameDuration: DEFAULT_FRAME_DURATION,
  autoPlay: false,
  loop: true,
  watermark: { ...DEFAULT_WATERMARK_SETTINGS },
  introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
  outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
});

const getDefaultWorkflowSettings = (): WorkflowDefaultSettings => ({
  defaultTimeout: 30,
  stepTimeout: 10,
  retryAttempts: 2,
  retryDelay: 1000,
  screenshotOnFailure: true,
  screenshotOnSuccess: false,
  headless: true,
  slowMo: 0,
});

const getDefaultApiKeySettings = (): ApiKeySettings => ({
  browserlessApiKey: '',
  openaiApiKey: '',
  anthropicApiKey: '',
  customApiEndpoint: '',
});

const getDefaultDisplaySettings = (): DisplaySettings => ({
  themeMode: 'dark',
  fontSize: 'medium',
  fontFamily: 'sans',
  reducedMotion: false,
  highContrast: false,
  compactMode: false,
  sidebarWidth: 'default',
});

const loadWorkflowDefaults = (): WorkflowDefaultSettings => {
  try {
    const stored = safeGetItem(WORKFLOW_DEFAULTS_KEY);
    if (!stored) return getDefaultWorkflowSettings();
    const parsed = JSON.parse(stored);
    return { ...getDefaultWorkflowSettings(), ...parsed };
  } catch {
    return getDefaultWorkflowSettings();
  }
};

const saveWorkflowDefaults = (settings: WorkflowDefaultSettings): void => {
  safeSetItem(WORKFLOW_DEFAULTS_KEY, JSON.stringify(settings));
};

const loadApiKeySettings = (): ApiKeySettings => {
  try {
    const stored = safeGetItem(API_KEYS_KEY);
    if (!stored) return getDefaultApiKeySettings();
    const parsed = JSON.parse(stored);
    return { ...getDefaultApiKeySettings(), ...parsed };
  } catch {
    return getDefaultApiKeySettings();
  }
};

const saveApiKeySettings = (settings: ApiKeySettings): void => {
  safeSetItem(API_KEYS_KEY, JSON.stringify(settings));
};

const loadDisplaySettings = (): DisplaySettings => {
  try {
    const stored = safeGetItem(DISPLAY_SETTINGS_KEY);
    if (!stored) return getDefaultDisplaySettings();
    const parsed = JSON.parse(stored);
    return { ...getDefaultDisplaySettings(), ...parsed };
  } catch {
    return getDefaultDisplaySettings();
  }
};

const saveDisplaySettings = (settings: DisplaySettings): void => {
  safeSetItem(DISPLAY_SETTINGS_KEY, JSON.stringify(settings));
};

// Detect system color scheme preference
const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
};

// Detect system reduced motion preference
const getSystemReducedMotion = (): boolean => {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
};

// Apply theme to document
const applyTheme = (settings: DisplaySettings): void => {
  if (typeof window === 'undefined') return;

  const root = document.documentElement;

  // Determine effective theme
  const effectiveTheme = settings.themeMode === 'system' ? getSystemTheme() : settings.themeMode;
  root.setAttribute('data-theme', effectiveTheme);

  // Apply font size class
  root.setAttribute('data-font-size', settings.fontSize);

  // Apply font family class
  root.setAttribute('data-font-family', settings.fontFamily);

  // Apply reduced motion
  const effectiveReducedMotion = settings.reducedMotion || getSystemReducedMotion();
  root.setAttribute('data-reduced-motion', String(effectiveReducedMotion));

  // Apply high contrast
  root.setAttribute('data-high-contrast', String(settings.highContrast));

  // Apply compact mode
  root.setAttribute('data-compact', String(settings.compactMode));
};

// Load user presets from localStorage
const loadUserPresets = (): ReplayPreset[] => {
  try {
    const stored = safeGetItem(PRESETS_STORAGE_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((p: unknown) =>
      p && typeof p === 'object' && 'id' in p && 'name' in p && 'settings' in p
    );
  } catch {
    return [];
  }
};

// Save user presets to localStorage
const saveUserPresets = (presets: ReplayPreset[]): void => {
  safeSetItem(PRESETS_STORAGE_KEY, JSON.stringify(presets));
};

// Generate a random ID for new presets
const generatePresetId = (): string => {
  return `user-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
};

// Pick a random item from an array
const randomChoice = <T,>(arr: readonly T[]): T => {
  return arr[Math.floor(Math.random() * arr.length)];
};

// Valid speed profiles for randomization
const SPEED_PROFILES: CursorSpeedProfile[] = ['instant', 'linear', 'easeIn', 'easeOut', 'easeInOut'];
const PATH_STYLES: CursorPathStyle[] = ['linear', 'parabolicUp', 'parabolicDown', 'cubic', 'pseudorandom'];

// Generate random replay settings
const generateRandomSettings = (): ReplaySettings => {
  return {
    chromeTheme: randomChoice(REPLAY_CHROME_OPTIONS).id,
    backgroundTheme: randomChoice(REPLAY_BACKGROUND_OPTIONS).id,
    cursorTheme: randomChoice(REPLAY_CURSOR_OPTIONS).id,
    cursorInitialPosition: randomChoice(REPLAY_CURSOR_POSITIONS).id,
    cursorScale: Math.round((CURSOR_SCALE_MIN + Math.random() * (CURSOR_SCALE_MAX - CURSOR_SCALE_MIN)) * 10) / 10,
    cursorClickAnimation: randomChoice(REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS).id,
    cursorSpeedProfile: randomChoice(SPEED_PROFILES),
    cursorPathStyle: randomChoice(PATH_STYLES),
    frameDuration: Math.round((800 + Math.random() * 5200) / 100) * 100, // 800-6000 in 100ms steps
    autoPlay: Math.random() > 0.5,
    loop: Math.random() > 0.3, // 70% chance of loop
    // Preserve current branding settings when randomizing
    watermark: { ...DEFAULT_WATERMARK_SETTINGS },
    introCard: { ...DEFAULT_INTRO_CARD_SETTINGS },
    outroCard: { ...DEFAULT_OUTRO_CARD_SETTINGS },
  };
};

// Initialize display settings and apply theme on load
const initialDisplaySettings = loadDisplaySettings();
if (typeof window !== 'undefined') {
  applyTheme(initialDisplaySettings);

  // Listen for system theme changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    const current = loadDisplaySettings();
    if (current.themeMode === 'system') {
      applyTheme(current);
    }
  });
}

export const useSettingsStore = create<SettingsStore>((set, get) => ({
  replay: loadReplaySettings(),
  workflowDefaults: loadWorkflowDefaults(),
  apiKeys: loadApiKeySettings(),
  display: initialDisplaySettings,
  userPresets: loadUserPresets(),
  activePresetId: null,

  setReplaySetting: (key, value) => {
    saveReplaySetting(key, value);
    set((state) => ({
      replay: {
        ...state.replay,
        [key]: value,
      },
      // Clear active preset since settings were manually changed
      activePresetId: null,
    }));
  },

  resetReplaySettings: () => {
    const defaults = getDefaultReplaySettings();
    // Clear all replay settings from storage
    Object.keys(defaults).forEach((key) => {
      try {
        if (typeof window !== 'undefined') {
          window.localStorage.removeItem(`${STORAGE_PREFIX}replay.${key}`);
        }
      } catch {
        // Ignore
      }
    });
    set({ replay: defaults, activePresetId: 'default' });
  },

  loadPreset: (presetId: string) => {
    // Find in built-in presets first
    let preset = BUILT_IN_PRESETS.find((p) => p.id === presetId);
    // Then check user presets
    if (!preset) {
      preset = get().userPresets.find((p) => p.id === presetId);
    }
    if (!preset) return;

    // Apply all settings
    const { settings } = preset;
    Object.entries(settings).forEach(([key, value]) => {
      saveReplaySetting(key as keyof ReplaySettings, value);
    });

    set({ replay: { ...settings }, activePresetId: presetId });
  },

  saveAsPreset: (name: string) => {
    const currentSettings = get().replay;
    const newPreset: ReplayPreset = {
      id: generatePresetId(),
      name,
      settings: { ...currentSettings },
      createdAt: new Date().toISOString(),
    };

    const updatedPresets = [...get().userPresets, newPreset];
    saveUserPresets(updatedPresets);
    set({ userPresets: updatedPresets, activePresetId: newPreset.id });

    return newPreset;
  },

  deletePreset: (presetId: string) => {
    const updatedPresets = get().userPresets.filter((p) => p.id !== presetId);
    saveUserPresets(updatedPresets);

    // Clear active preset if it was the deleted one
    const newActiveId = get().activePresetId === presetId ? null : get().activePresetId;
    set({ userPresets: updatedPresets, activePresetId: newActiveId });
  },

  randomizeSettings: () => {
    const randomSettings = generateRandomSettings();

    // Apply all settings to storage
    Object.entries(randomSettings).forEach(([key, value]) => {
      saveReplaySetting(key as keyof ReplaySettings, value);
    });

    set({ replay: randomSettings, activePresetId: null });
  },

  getAllPresets: () => {
    return [...BUILT_IN_PRESETS, ...get().userPresets];
  },

  setWorkflowDefault: (key, value) => {
    const updated = { ...get().workflowDefaults, [key]: value };
    saveWorkflowDefaults(updated);
    set({ workflowDefaults: updated });
  },

  resetWorkflowDefaults: () => {
    const defaults = getDefaultWorkflowSettings();
    saveWorkflowDefaults(defaults);
    set({ workflowDefaults: defaults });
  },

  setApiKey: (key, value) => {
    const updated = { ...get().apiKeys, [key]: value };
    saveApiKeySettings(updated);
    set({ apiKeys: updated });
  },

  clearApiKeys: () => {
    const defaults = getDefaultApiKeySettings();
    saveApiKeySettings(defaults);
    set({ apiKeys: defaults });
  },

  setDisplaySetting: (key, value) => {
    const updated = { ...get().display, [key]: value };
    saveDisplaySettings(updated);
    applyTheme(updated);
    set({ display: updated });
  },

  resetDisplaySettings: () => {
    const defaults = getDefaultDisplaySettings();
    saveDisplaySettings(defaults);
    applyTheme(defaults);
    set({ display: defaults });
  },

  getEffectiveTheme: () => {
    const { themeMode } = get().display;
    return themeMode === 'system' ? getSystemTheme() : themeMode;
  },
}));

export default useSettingsStore;
