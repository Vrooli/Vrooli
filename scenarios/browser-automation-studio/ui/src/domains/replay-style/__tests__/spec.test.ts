import { describe, expect, it } from 'vitest';
import type { ReplaySpec as ReplayMovieSpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { applyReplayStyleToSpec, resolveReplayStyleFromSpec } from '../adapters/spec';
import { getReplayBackgroundThemeId } from '../model';

describe('replay style spec adapter', () => {
  it('hydrates replay style from movie spec', () => {
    const spec = {
      decor: {
        chromeTheme: 'chromium',
        backgroundTheme: 'ocean',
        background: { type: 'theme', id: 'ocean' },
        cursorTheme: 'white',
        cursorInitialPosition: 'top-left',
        cursorClickAnimation: 'pulse',
        cursorScale: 1.3,
      },
      cursorMotion: {
        initialPosition: 'bottom-right',
        clickAnimation: 'ripple',
        cursorScale: 1.1,
      },
      cursor: {
        initialPosition: 'center',
        clickAnimation: 'none',
        scale: 1,
      },
    } as ReplayMovieSpec;

    const style = resolveReplayStyleFromSpec(spec);

    expect(style.chromeTheme).toBe('chromium');
    expect(getReplayBackgroundThemeId(style.background)).toBe('ocean');
    expect(style.cursorTheme).toBe('white');
    expect(style.cursorInitialPosition).toBe('top-left');
    expect(style.cursorClickAnimation).toBe('pulse');
    expect(style.cursorScale).toBe(1.3);
  });

  it('applies replay style into spec decor and presentation', () => {
    const spec = {
      decor: {},
      cursorMotion: {},
      cursor: {},
      presentation: {
        canvas: { width: 1000, height: 800 },
      },
    } as ReplayMovieSpec;

    const updated = applyReplayStyleToSpec(spec, {
      chromeTheme: 'midnight',
      background: { type: 'theme', id: 'nebula' },
      cursorTheme: 'aura',
      cursorInitialPosition: 'bottom-right',
      cursorClickAnimation: 'ripple',
      cursorScale: 1.2,
      browserScale: 0.5,
    });

    expect(updated.decor?.chromeTheme).toBe('midnight');
    expect(updated.decor?.backgroundTheme).toBe('nebula');
    expect(updated.decor?.background).toMatchObject({ type: 'theme', id: 'nebula' });
    expect(updated.decor?.cursorTheme).toBe('aura');
    expect(updated.cursor?.scale).toBe(1.2);
    // A 0.5 request clamps to the supported 0.6 minimum. The frame also
    // contains browser chrome; its height cannot equal the image alone.
    expect(updated.decor?.browserScale).toBe(0.6);
    expect(updated.presentation?.browserFrame?.width).toBe(600);
    expect(updated.presentation?.browserFrame?.height).toBeGreaterThan(480);
  });
  it('preserves explicit disabled chrome and device framing through export spec', () => {
    const updated = applyReplayStyleToSpec({ presentation: { canvas: { width: 1080, height: 1920 }, viewport: { width: 430, height: 880 } } } as ReplayMovieSpec, {
      presentation: { showDesktop: true, showBrowserFrame: false, showDeviceFrame: true },
      deviceFrameTheme: 'slim', browserScale: 0.7, cursorTheme: 'arrowNeon',
    });
    const restored = resolveReplayStyleFromSpec(updated);
    expect(restored.presentation).toEqual({ showDesktop: true, showBrowserFrame: false, showDeviceFrame: true });
    expect(restored.deviceFrameTheme).toBe('slim');
    expect(restored.browserScale).toBe(0.7);
    expect(restored.cursorTheme).toBe('arrowNeon');
  });

});
