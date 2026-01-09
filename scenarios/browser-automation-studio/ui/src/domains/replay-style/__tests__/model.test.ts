import { describe, expect, it } from 'vitest';
import {
  normalizeReplayStyle,
  resolveReplayStyle,
  REPLAY_STYLE_DEFAULTS,
  getReplayBackgroundThemeId,
} from '../model';
import {
  MAX_BROWSER_SCALE,
  MIN_BROWSER_SCALE,
  MAX_CURSOR_SCALE,
  MIN_CURSOR_SCALE,
} from '@/domains/replay-style';
import { buildGradientCss, parseGradientCss } from '../gradient';

describe('normalizeReplayStyle', () => {
  it('clamps scale values', () => {
    const normalized = normalizeReplayStyle({
      chromeTheme: 'midnight',
      background: { type: 'theme', id: 'ocean' },
      cursorScale: '999',
      browserScale: '0.1',
      cursorClickAnimation: 'pulse',
      cursorInitialPosition: 'top-left',
    });

    expect(normalized.chromeTheme).toBe('midnight');
    expect(getReplayBackgroundThemeId(normalized.background)).toBe('ocean');
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

describe('gradient css helpers', () => {
  it('parses linear gradient css into a spec', () => {
    const parsed = parseGradientCss('linear-gradient(120deg, #111111 0%, #222222 100%)');
    expect(parsed?.type).toBe('linear');
    expect(parsed?.angle).toBe(120);
    expect(parsed?.stops.length).toBeGreaterThanOrEqual(2);
  });

  it('serializes radial gradient css', () => {
    const css = buildGradientCss({
      type: 'radial',
      center: { x: 20, y: 40 },
      stops: [
        { color: '#111111', position: 0 },
        { color: '#222222', position: 100 },
      ],
    });
    expect(css).toContain('radial-gradient');
    expect(css).toContain('20% 40%');
  });
});
