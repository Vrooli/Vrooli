import { describe, expect, it } from 'vitest';
import {
  REPLAY_BACKGROUND_OPTIONS,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
} from '../catalog';
import {
  REPLAY_BACKGROUND_THEME_IDS,
  REPLAY_CHROME_THEME_IDS,
  REPLAY_CURSOR_CLICK_ANIMATIONS,
  REPLAY_CURSOR_INITIAL_POSITIONS,
  REPLAY_CURSOR_THEME_IDS,
  REPLAY_STYLE_DEFAULTS,
} from '../model';

const assertUniqueIds = (ids: string[]) => {
  expect(new Set(ids).size).toBe(ids.length);
};

describe('replay style catalog', () => {
  it('exposes unique theme ids', () => {
    assertUniqueIds(REPLAY_CHROME_OPTIONS.map((option) => option.id));
    assertUniqueIds(REPLAY_BACKGROUND_OPTIONS.map((option) => option.id));
    assertUniqueIds(REPLAY_CURSOR_OPTIONS.map((option) => option.id));
    assertUniqueIds(REPLAY_CURSOR_POSITIONS.map((option) => option.id));
    assertUniqueIds(REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option) => option.id));
  });

  it('covers every model theme id and defaults', () => {
    const chromeIds = REPLAY_CHROME_OPTIONS.map((option) => option.id);
    const backgroundIds = REPLAY_BACKGROUND_OPTIONS.map((option) => option.id);
    const cursorIds = REPLAY_CURSOR_OPTIONS.map((option) => option.id);
    const positionIds = REPLAY_CURSOR_POSITIONS.map((option) => option.id);
    const clickIds = REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option) => option.id);

    for (const id of REPLAY_CHROME_THEME_IDS) {
      expect(chromeIds).toContain(id);
    }
    for (const id of REPLAY_BACKGROUND_THEME_IDS) {
      expect(backgroundIds).toContain(id);
    }
    for (const id of REPLAY_CURSOR_THEME_IDS) {
      expect(cursorIds).toContain(id);
    }
    for (const id of REPLAY_CURSOR_INITIAL_POSITIONS) {
      expect(positionIds).toContain(id);
    }
    for (const id of REPLAY_CURSOR_CLICK_ANIMATIONS) {
      expect(clickIds).toContain(id);
    }

    expect(chromeIds).toContain(REPLAY_STYLE_DEFAULTS.chromeTheme);
    expect(backgroundIds).toContain(REPLAY_STYLE_DEFAULTS.backgroundTheme);
    expect(cursorIds).toContain(REPLAY_STYLE_DEFAULTS.cursorTheme);
    expect(positionIds).toContain(REPLAY_STYLE_DEFAULTS.cursorInitialPosition);
    expect(clickIds).toContain(REPLAY_STYLE_DEFAULTS.cursorClickAnimation);
  });
});
