import { describe, expect, it } from 'vitest';
import {
  normalizeReplayStyle,
  resolveReplayStyle,
  REPLAY_STYLE_DEFAULTS,
} from '../model';
import {
  MAX_BROWSER_SCALE,
  MIN_BROWSER_SCALE,
  MAX_CURSOR_SCALE,
  MIN_CURSOR_SCALE,
} from '@/domains/exports/replay/constants';

describe('normalizeReplayStyle', () => {
  it('normalizes legacy keys and clamps scale values', () => {
    const normalized = normalizeReplayStyle({
      replayChromeTheme: 'midnight',
      background_theme: 'ocean',
      cursor_scale: '999',
      browser_scale: '0.1',
      cursor_click_animation: 'pulse',
      cursor_initial_position: 'top-left',
    });

    expect(normalized.chromeTheme).toBe('midnight');
    expect(normalized.backgroundTheme).toBe('ocean');
    expect(normalized.cursorClickAnimation).toBe('pulse');
    expect(normalized.cursorInitialPosition).toBe('top-left');
    expect(normalized.cursorScale).toBe(MAX_CURSOR_SCALE);
    expect(normalized.browserScale).toBe(MIN_BROWSER_SCALE);
  });

  it('falls back to defaults for invalid values', () => {
    const fallback = {
      ...REPLAY_STYLE_DEFAULTS,
      chromeTheme: 'chromium',
      cursorTheme: 'black',
    };
    const normalized = normalizeReplayStyle(
      { chromeTheme: 'nope', cursorTheme: 'bad', cursorScale: -10 },
      fallback,
    );

    expect(normalized.chromeTheme).toBe('chromium');
    expect(normalized.cursorTheme).toBe('black');
    expect(normalized.cursorScale).toBe(MIN_CURSOR_SCALE);
  });
});

describe('resolveReplayStyle', () => {
  it('applies overrides over stored values and defaults', () => {
    const resolved = resolveReplayStyle({
      defaults: REPLAY_STYLE_DEFAULTS,
      stored: { chromeTheme: 'midnight', cursorScale: 1.4 },
      overrides: { chromeTheme: 'chromium' },
    });

    expect(resolved.chromeTheme).toBe('chromium');
    expect(resolved.cursorScale).toBe(1.4);
  });
});
