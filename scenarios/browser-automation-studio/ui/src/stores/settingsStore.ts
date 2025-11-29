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
} from '../features/execution/replay/replayThemeOptions';

const STORAGE_PREFIX = 'browserAutomation.settings.';

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
}

interface SettingsStore {
  replay: ReplaySettings;
  setReplaySetting: <K extends keyof ReplaySettings>(key: K, value: ReplaySettings[K]) => void;
  resetReplaySettings: () => void;
}

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
  };
};

const saveReplaySetting = <K extends keyof ReplaySettings>(key: K, value: ReplaySettings[K]): void => {
  safeSetItem(`${STORAGE_PREFIX}replay.${key}`, String(value));
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
});

export const useSettingsStore = create<SettingsStore>((set) => ({
  replay: loadReplaySettings(),

  setReplaySetting: (key, value) => {
    saveReplaySetting(key, value);
    set((state) => ({
      replay: {
        ...state.replay,
        [key]: value,
      },
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
    set({ replay: defaults });
  },
}));

export default useSettingsStore;
