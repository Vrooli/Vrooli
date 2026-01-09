import { describe, expect, it } from 'vitest';
import {
  REPLAY_BACKGROUND_OPTIONS,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
} from '../catalog';
import { REPLAY_STYLE_DEFAULTS } from '../model';

describe('replay style registry', () => {
  it('includes default theme IDs', () => {
    const chromeIds = REPLAY_CHROME_OPTIONS.map((option) => option.id);
    const backgroundIds = REPLAY_BACKGROUND_OPTIONS.map((option) => option.id);
    const cursorIds = REPLAY_CURSOR_OPTIONS.map((option) => option.id);

    expect(chromeIds).toContain(REPLAY_STYLE_DEFAULTS.chromeTheme);
    expect(backgroundIds).toContain(REPLAY_STYLE_DEFAULTS.background.type === 'theme'
      ? REPLAY_STYLE_DEFAULTS.background.id
      : 'aurora');
    expect(cursorIds).toContain(REPLAY_STYLE_DEFAULTS.cursorTheme);
  });
});
