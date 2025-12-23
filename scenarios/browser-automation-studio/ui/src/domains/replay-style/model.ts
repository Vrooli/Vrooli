import { MAX_BROWSER_SCALE, MIN_BROWSER_SCALE, MAX_CURSOR_SCALE, MIN_CURSOR_SCALE } from '@/domains/exports/replay/constants';

export const REPLAY_STYLE_VERSION = 1 as const;

export const REPLAY_CHROME_THEME_IDS = ['aurora', 'chromium', 'midnight', 'minimal'] as const;
export const REPLAY_BACKGROUND_THEME_IDS = [
  'aurora',
  'sunset',
  'ocean',
  'nebula',
  'grid',
  'charcoal',
  'steel',
  'emerald',
  'none',
  'geoPrism',
  'geoOrbit',
  'geoMosaic',
] as const;
export const REPLAY_CURSOR_THEME_IDS = [
  'disabled',
  'white',
  'black',
  'aura',
  'arrowLight',
  'arrowDark',
  'arrowNeon',
  'handNeutral',
  'handAura',
] as const;
export const REPLAY_CURSOR_INITIAL_POSITIONS = [
  'center',
  'top-left',
  'top-right',
  'bottom-left',
  'bottom-right',
  'random',
] as const;
export const REPLAY_CURSOR_CLICK_ANIMATIONS = ['none', 'pulse', 'ripple'] as const;

export type ReplayChromeTheme = (typeof REPLAY_CHROME_THEME_IDS)[number];
export type ReplayBackgroundTheme = (typeof REPLAY_BACKGROUND_THEME_IDS)[number];
export type ReplayCursorTheme = (typeof REPLAY_CURSOR_THEME_IDS)[number];
export type ReplayCursorInitialPosition = (typeof REPLAY_CURSOR_INITIAL_POSITIONS)[number];
export type ReplayCursorClickAnimation = (typeof REPLAY_CURSOR_CLICK_ANIMATIONS)[number];

export interface ReplayStyleConfig {
  version: typeof REPLAY_STYLE_VERSION;
  chromeTheme: ReplayChromeTheme;
  backgroundTheme: ReplayBackgroundTheme;
  cursorTheme: ReplayCursorTheme;
  cursorInitialPosition: ReplayCursorInitialPosition;
  cursorClickAnimation: ReplayCursorClickAnimation;
  cursorScale: number;
  browserScale: number;
}

export type ReplayStyleOverrides = Partial<Omit<ReplayStyleConfig, 'version'>> & {
  version?: number;
};

export const REPLAY_STYLE_DEFAULTS: ReplayStyleConfig = {
  version: REPLAY_STYLE_VERSION,
  chromeTheme: 'aurora',
  backgroundTheme: 'aurora',
  cursorTheme: 'white',
  cursorInitialPosition: 'center',
  cursorClickAnimation: 'pulse',
  cursorScale: 1,
  browserScale: 1,
};

export const clampReplayCursorScale = (value: number): number => {
  return Math.min(MAX_CURSOR_SCALE, Math.max(MIN_CURSOR_SCALE, value));
};

export const clampReplayBrowserScale = (value: number): number => {
  return Math.min(MAX_BROWSER_SCALE, Math.max(MIN_BROWSER_SCALE, value));
};

export const isReplayChromeTheme = (value: unknown): value is ReplayChromeTheme =>
  REPLAY_CHROME_THEME_IDS.includes(value as ReplayChromeTheme);

export const isReplayBackgroundTheme = (value: unknown): value is ReplayBackgroundTheme =>
  REPLAY_BACKGROUND_THEME_IDS.includes(value as ReplayBackgroundTheme);

export const isReplayCursorTheme = (value: unknown): value is ReplayCursorTheme =>
  REPLAY_CURSOR_THEME_IDS.includes(value as ReplayCursorTheme);

export const isReplayCursorInitialPosition = (value: unknown): value is ReplayCursorInitialPosition =>
  REPLAY_CURSOR_INITIAL_POSITIONS.includes(value as ReplayCursorInitialPosition);

export const isReplayCursorClickAnimation = (value: unknown): value is ReplayCursorClickAnimation =>
  REPLAY_CURSOR_CLICK_ANIMATIONS.includes(value as ReplayCursorClickAnimation);

const readString = (source: Record<string, unknown>, keys: string[]): string | undefined => {
  for (const key of keys) {
    const value = source[key];
    if (typeof value === 'string' && value.trim().length > 0) {
      return value.trim();
    }
  }
  return undefined;
};

const readNumber = (source: Record<string, unknown>, keys: string[]): number | undefined => {
  for (const key of keys) {
    const value = source[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
    if (typeof value === 'string' && value.trim().length > 0) {
      const parsed = Number.parseFloat(value);
      if (Number.isFinite(parsed)) {
        return parsed;
      }
    }
  }
  return undefined;
};

export const normalizeReplayStyle = (
  raw: unknown,
  fallback: ReplayStyleConfig = REPLAY_STYLE_DEFAULTS,
): ReplayStyleConfig => {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {};

  const chrome = readString(source, ['chromeTheme', 'replayChromeTheme', 'chrome_theme']);
  const background = readString(source, ['backgroundTheme', 'replayBackgroundTheme', 'background_theme']);
  const cursor = readString(source, ['cursorTheme', 'replayCursorTheme', 'cursor_theme']);
  const cursorInitial = readString(source, [
    'cursorInitialPosition',
    'replayCursorInitialPosition',
    'cursor_initial_position',
  ]);
  const cursorClick = readString(source, [
    'cursorClickAnimation',
    'replayCursorClickAnimation',
    'cursor_click_animation',
  ]);
  const cursorScale = readNumber(source, ['cursorScale', 'replayCursorScale', 'cursor_scale']);
  const browserScale = readNumber(source, ['browserScale', 'replayBrowserScale', 'browser_scale']);

  return {
    version: REPLAY_STYLE_VERSION,
    chromeTheme: isReplayChromeTheme(chrome) ? chrome : fallback.chromeTheme,
    backgroundTheme: isReplayBackgroundTheme(background) ? background : fallback.backgroundTheme,
    cursorTheme: isReplayCursorTheme(cursor) ? cursor : fallback.cursorTheme,
    cursorInitialPosition: isReplayCursorInitialPosition(cursorInitial)
      ? cursorInitial
      : fallback.cursorInitialPosition,
    cursorClickAnimation: isReplayCursorClickAnimation(cursorClick)
      ? cursorClick
      : fallback.cursorClickAnimation,
    cursorScale:
      typeof cursorScale === 'number'
        ? clampReplayCursorScale(cursorScale)
        : fallback.cursorScale,
    browserScale:
      typeof browserScale === 'number'
        ? clampReplayBrowserScale(browserScale)
        : fallback.browserScale,
  };
};

export const resolveReplayStyle = (options: {
  overrides?: ReplayStyleOverrides;
  stored?: ReplayStyleOverrides;
  defaults?: ReplayStyleConfig;
}): ReplayStyleConfig => {
  const defaults = options.defaults ?? REPLAY_STYLE_DEFAULTS;
  const merged = {
    ...defaults,
    ...(options.stored ?? {}),
    ...(options.overrides ?? {}),
  };
  return normalizeReplayStyle(merged, defaults);
};
