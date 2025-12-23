import { MAX_BROWSER_SCALE, MIN_BROWSER_SCALE, MAX_CURSOR_SCALE, MIN_CURSOR_SCALE } from './constants';

export const REPLAY_STYLE_VERSION = 2 as const;

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

export type ReplayBackgroundImageFit = 'cover' | 'contain';

export interface ReplayGradientStop {
  color: string;
  position?: number;
}

export interface ReplayGradientSpec {
  type: 'linear' | 'radial';
  angle?: number;
  center?: { x: number; y: number };
  stops: ReplayGradientStop[];
}

export type ReplayBackgroundSource =
  | { type: 'theme'; id: ReplayBackgroundTheme }
  | { type: 'gradient'; value: ReplayGradientSpec }
  | { type: 'image'; assetId?: string; url?: string; fit?: ReplayBackgroundImageFit };

export const isReplayBackgroundImage = (
  background: ReplayBackgroundSource,
): background is Extract<ReplayBackgroundSource, { type: 'image' }> => background.type === 'image';

export interface ReplayStyleConfig {
  version: typeof REPLAY_STYLE_VERSION;
  chromeTheme: ReplayChromeTheme;
  background: ReplayBackgroundSource;
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
  background: { type: 'theme', id: 'aurora' },
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

const isReplayBackgroundImageFit = (value: unknown): value is ReplayBackgroundImageFit =>
  value === 'cover' || value === 'contain';

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

const readObject = (source: Record<string, unknown>, keys: string[]): Record<string, unknown> | undefined => {
  for (const key of keys) {
    const value = source[key];
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return value as Record<string, unknown>;
    }
  }
  return undefined;
};

const normalizeGradientStop = (value: unknown): ReplayGradientStop | null => {
  if (!value || typeof value !== 'object') {
    return null;
  }
  const stop = value as Record<string, unknown>;
  const color = readString(stop, ['color', 'value']);
  if (!color) {
    return null;
  }
  const position = readNumber(stop, ['position', 'pos']);
  return typeof position === 'number' ? { color, position } : { color };
};

const normalizeGradientSpec = (value: unknown): ReplayGradientSpec | null => {
  if (!value || typeof value !== 'object') {
    return null;
  }
  const spec = value as Record<string, unknown>;
  const type = readString(spec, ['type', 'kind']);
  const rawStops = spec.stops;
  const stops = Array.isArray(rawStops)
    ? rawStops.map(normalizeGradientStop).filter(Boolean)
    : [];
  if (stops.length < 2) {
    return null;
  }
  const angle = readNumber(spec, ['angle', 'rotation']);
  const centerObject = readObject(spec, ['center', 'origin']);
  const centerX = centerObject ? readNumber(centerObject, ['x', 'left']) : readNumber(spec, ['centerX', 'center_x']);
  const centerY = centerObject ? readNumber(centerObject, ['y', 'top']) : readNumber(spec, ['centerY', 'center_y']);
  const normalizedType = type === 'radial' ? 'radial' : 'linear';
  const clampPercent = (value: number): number => Math.min(100, Math.max(0, value));
  const center =
    normalizedType === 'radial' && typeof centerX === 'number' && typeof centerY === 'number'
      ? { x: clampPercent(centerX), y: clampPercent(centerY) }
      : undefined;
  return {
    type: normalizedType,
    angle: normalizedType === 'linear' && typeof angle === 'number' ? angle : undefined,
    center,
    stops: stops as ReplayGradientStop[],
  };
};

const normalizeBackgroundSource = (
  raw: Record<string, unknown> | undefined,
  fallback: ReplayBackgroundSource,
  legacyTheme?: string,
): ReplayBackgroundSource => {
  if (raw) {
    const type = readString(raw, ['type']);
    if (type === 'theme') {
      const theme = readString(raw, ['id', 'theme', 'value']);
      if (isReplayBackgroundTheme(theme)) {
        return { type: 'theme', id: theme };
      }
    }
    if (type === 'gradient') {
      const gradient = normalizeGradientSpec(raw.value ?? raw.gradient ?? raw.spec);
      if (gradient) {
        return { type: 'gradient', value: gradient };
      }
    }
    if (type === 'image') {
      const assetId = readString(raw, ['assetId', 'asset_id']);
      const url = readString(raw, ['url', 'src']);
      const fit = readString(raw, ['fit', 'objectFit']);
      if (assetId || url) {
        return {
          type: 'image',
          assetId: assetId ?? undefined,
          url: url ?? undefined,
          fit: isReplayBackgroundImageFit(fit) ? fit : fallback.type === 'image' ? fallback.fit : 'cover',
        };
      }
    }
  }

  if (isReplayBackgroundTheme(legacyTheme)) {
    return { type: 'theme', id: legacyTheme };
  }

  return fallback;
};

export const getReplayBackgroundThemeId = (
  background: ReplayBackgroundSource,
  fallback: ReplayBackgroundTheme = REPLAY_STYLE_DEFAULTS.background.type === 'theme'
    ? REPLAY_STYLE_DEFAULTS.background.id
    : 'aurora',
): ReplayBackgroundTheme => {
  if (background.type === 'theme') {
    return background.id;
  }
  return fallback;
};

export const normalizeReplayStyle = (
  raw: unknown,
  fallback: ReplayStyleConfig = REPLAY_STYLE_DEFAULTS,
): ReplayStyleConfig => {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {};

  const chrome = readString(source, ['chromeTheme', 'replayChromeTheme', 'chrome_theme']);
  const backgroundTheme = readString(source, ['backgroundTheme', 'replayBackgroundTheme', 'background_theme']);
  const backgroundSource = readObject(source, ['background', 'replayBackground', 'background_source']);
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
    background: normalizeBackgroundSource(backgroundSource, fallback.background, backgroundTheme),
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
