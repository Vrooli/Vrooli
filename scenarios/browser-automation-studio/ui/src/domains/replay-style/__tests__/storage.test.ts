import { beforeEach, describe, expect, it } from 'vitest';
import {
  readReplayStyleFromStorage,
  writeReplayStyleToStorage,
} from '../adapters/storage';
import { REPLAY_STYLE_DEFAULTS } from '../model';

describe('replay style storage adapter', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('returns defaults when storage is empty', () => {
    const style = readReplayStyleFromStorage();
    expect(style).toEqual(REPLAY_STYLE_DEFAULTS);
  });

  it('writes canonical storage keys', () => {
    const config = { ...REPLAY_STYLE_DEFAULTS, chromeTheme: 'chromium', cursorScale: 1.25 };

    writeReplayStyleToStorage(config);

    const stored = JSON.parse(
      window.localStorage.getItem('browserAutomation.replayStyle') ?? '{}',
    ) as Record<string, unknown>;
    expect(stored.chromeTheme).toBe('chromium');
    expect(window.localStorage.getItem('browserAutomation.replayChromeTheme')).toBeNull();
    expect(window.localStorage.getItem('browserAutomation.replayCursorScale')).toBeNull();
  });
});
