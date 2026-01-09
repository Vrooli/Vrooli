import { describe, expect, it } from 'vitest';
import type { ReplayMovieSpec } from '@/types/export';
import { applyReplayStyleToSpec, resolveReplayStyleFromSpec } from '../adapters/spec';
import { getReplayBackgroundThemeId } from '../model';

describe('replay style spec adapter', () => {
  it('hydrates replay style from movie spec', () => {
    const spec = {
      decor: {
        chrome_theme: 'chromium',
        background_theme: 'ocean',
        cursor_theme: 'white',
        cursor_initial_position: 'top-left',
        cursor_click_animation: 'pulse',
        cursor_scale: 1.3,
      },
      cursor_motion: {
        initial_position: 'bottom-right',
        click_animation: 'ripple',
        cursor_scale: 1.1,
      },
      cursor: {
        initial_position: 'center',
        click_animation: 'none',
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
      cursor_motion: {},
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

    expect(updated.decor?.chrome_theme).toBe('midnight');
    expect(updated.decor?.background_theme).toBe('nebula');
    expect(updated.decor?.background).toMatchObject({ type: 'theme', id: 'nebula' });
    expect(updated.decor?.cursor_theme).toBe('aura');
    expect(updated.cursor?.scale).toBe(1.2);
    expect(updated.presentation?.browser_frame?.width).toBe(500);
    expect(updated.presentation?.browser_frame?.height).toBe(400);
  });
});
