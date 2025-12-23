import type { ReplayBackgroundTheme, ReplayStyleConfig } from './model';
import { getReplayBackgroundThemeId, resolveReplayPresentationStyle } from './model';
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
  const presentationStyle = resolveReplayPresentationStyle(style);
  const fallbackTheme = options.fallbackBackgroundTheme ?? getReplayBackgroundThemeId(presentationStyle.background);
  const backgroundDecor = resolveBackgroundDecor(presentationStyle.background, fallbackTheme);
  const chromeDecor = buildChromeDecor(presentationStyle.chromeTheme, options.title);
  const hasFullBleed = style.presentationMode !== 'desktop'
    || (presentationStyle.background.type === 'theme' && presentationStyle.background.id === 'none');
  return {
    backgroundDecor: hasFullBleed
      ? { ...backgroundDecor, containerClass: `${backgroundDecor.containerClass} rounded-none` }
      : backgroundDecor,
    chromeDecor: hasFullBleed
      ? { ...chromeDecor, frameClass: `${chromeDecor.frameClass} rounded-none` }
      : chromeDecor,
    cursorDecor: buildCursorDecor(style.cursorTheme),
  };
};
