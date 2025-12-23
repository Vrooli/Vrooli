import type { ReplayStyleConfig, ReplayStyleOverrides } from '../model';
import { normalizeReplayStyle, REPLAY_STYLE_DEFAULTS } from '../model';

const STORAGE_KEY = 'browserAutomation.replayStyle';

export const readReplayStyleFromStorage = (): ReplayStyleConfig => {
  if (typeof window === 'undefined') {
    return REPLAY_STYLE_DEFAULTS;
  }

  let parsed: ReplayStyleOverrides | null = null;
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      parsed = JSON.parse(stored) as ReplayStyleOverrides;
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
