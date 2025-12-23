import type { ReplayStyleConfig, ReplayStyleOverrides } from '../model';
import { normalizeReplayStyle, REPLAY_STYLE_DEFAULTS } from '../model';

const STORAGE_KEY = 'browserAutomation.replayStyle';
const ALLOWED_KEYS = new Set([
  'chromeTheme',
  'background',
  'cursorTheme',
  'cursorInitialPosition',
  'cursorClickAnimation',
  'cursorScale',
  'browserScale',
  'version',
]);

const sanitizeStoredStyle = (value: unknown): ReplayStyleOverrides => {
  if (!value || typeof value !== 'object') {
    return {};
  }
  const source = value as Record<string, unknown>;
  return Object.fromEntries(
    Object.entries(source).filter(([key]) => ALLOWED_KEYS.has(key)),
  ) as ReplayStyleOverrides;
};

export const readReplayStyleFromStorage = (): ReplayStyleConfig => {
  if (typeof window === 'undefined') {
    return REPLAY_STYLE_DEFAULTS;
  }

  let parsed: ReplayStyleOverrides | null = null;
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      parsed = sanitizeStoredStyle(JSON.parse(stored));
    }
  } catch {
    parsed = null;
  }

  return normalizeReplayStyle(parsed ?? {});
};

export const writeReplayStyleToStorage = (config: ReplayStyleConfig) => {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
  } catch {
    // Ignore storage failures.
  }

};
