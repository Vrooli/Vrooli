import type { ReplayBackgroundTheme, ReplayStyleConfig } from './model';
import { getReplayBackgroundThemeId, resolveReplayPresentationStyle } from './model';
import type { BackgroundDecor, ChromeDecor, CursorDecor } from './catalog';
import { buildChromeDecor, buildCursorDecor, resolveBackgroundDecor } from './catalog';

const DEVICE_FRAME_CLASS =
  'ring-1 ring-white/12 ring-offset-8 ring-offset-slate-950/65 shadow-[0_28px_80px_rgba(15,23,42,0.45)]';

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
  const baseBackgroundDecor = resolveBackgroundDecor(presentationStyle.background, fallbackTheme);
  const chromeDecorBase = buildChromeDecor(presentationStyle.chromeTheme, options.title);
  const hasDesktop = style.presentation.showDesktop;
  const hasDeviceFrame = hasDesktop && style.presentation.showDeviceFrame;
  const hasFullBleed = !hasDesktop
    || (presentationStyle.background.type === 'theme' && presentationStyle.background.id === 'none');
  const backgroundDecor = hasDeviceFrame
    ? { ...baseBackgroundDecor, containerClass: `${baseBackgroundDecor.containerClass} ${DEVICE_FRAME_CLASS}` }
    : baseBackgroundDecor;
  const needsSquareFrame = hasFullBleed || !style.presentation.showBrowserFrame;
  return {
    backgroundDecor: hasFullBleed
      ? { ...backgroundDecor, containerClass: `${backgroundDecor.containerClass} rounded-none` }
      : backgroundDecor,
    chromeDecor: needsSquareFrame
      ? { ...chromeDecorBase, frameClass: `${chromeDecorBase.frameClass} rounded-none` }
      : chromeDecorBase,
    cursorDecor: buildCursorDecor(style.cursorTheme),
  };
};
