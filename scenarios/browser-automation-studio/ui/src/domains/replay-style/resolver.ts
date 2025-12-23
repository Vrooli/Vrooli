import type { ReplayBackgroundTheme, ReplayStyleConfig } from './model';
import { getReplayBackgroundThemeId } from './model';
import type { BackgroundDecor, ChromeDecor, CursorDecor } from './catalog';
import { buildChromeDecor, buildCursorDecor, resolveBackgroundDecor } from './catalog';

export interface ReplayStyleTokens {
  backgroundDecor: BackgroundDecor;
  chromeDecor: ChromeDecor;
  cursorDecor: CursorDecor;
}

export const resolveReplayStyleTokens = (
  style: ReplayStyleConfig,
  options: { title: string; fallbackBackgroundTheme?: ReplayBackgroundTheme },
): ReplayStyleTokens => {
  const fallbackTheme = options.fallbackBackgroundTheme ?? getReplayBackgroundThemeId(style.background);
  return {
    backgroundDecor: resolveBackgroundDecor(style.background, fallbackTheme),
    chromeDecor: buildChromeDecor(style.chromeTheme, options.title),
    cursorDecor: buildCursorDecor(style.cursorTheme),
  };
};
