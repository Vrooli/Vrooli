import type { ReplayBackgroundTheme, ReplayStyleConfig } from './model';
import { getReplayBackgroundThemeId, resolveReplayPresentationStyle } from './model';
import type { BackgroundDecor, ChromeDecor, CursorDecor, DeviceFrameDecor } from './catalog';
import { buildChromeDecor, buildCursorDecor, buildDeviceFrameDecor, resolveBackgroundDecor } from './catalog';

export interface ReplayStyleTokens {
  backgroundDecor: BackgroundDecor;
  chromeDecor: ChromeDecor;
  cursorDecor: CursorDecor;
  deviceFrameDecor: DeviceFrameDecor | null;
}

export const resolveReplayStyleTokens = (
  style: ReplayStyleConfig,
  options: { title: string; fallbackBackgroundTheme?: ReplayBackgroundTheme },
): ReplayStyleTokens => {
  const presentationStyle = resolveReplayPresentationStyle(style);
  const fallbackTheme = options.fallbackBackgroundTheme ?? getReplayBackgroundThemeId(presentationStyle.background);
  const baseBackgroundDecor = resolveBackgroundDecor(presentationStyle.background, fallbackTheme);
  const chromeDecorBase = buildChromeDecor(presentationStyle.chromeTheme, options.title);
  const deviceFrameDecorBase = buildDeviceFrameDecor(style.deviceFrameTheme);
  const hasDesktop = style.presentation.showDesktop;
  const hasDeviceFrame = hasDesktop && style.presentation.showDeviceFrame;
  const hasFullBleed = !hasDesktop
    || (presentationStyle.background.type === 'theme' && presentationStyle.background.id === 'none');

  // Device hardware surrounds the product window, never the export canvas.
  const backgroundDecor = baseBackgroundDecor;
  const needsSquareFrame = hasFullBleed || (!style.presentation.showBrowserFrame && !hasDeviceFrame);

  return {
    backgroundDecor: hasFullBleed
      ? { ...backgroundDecor, containerClass: `${backgroundDecor.containerClass} rounded-none` }
      : backgroundDecor,
    chromeDecor: needsSquareFrame
      ? { ...chromeDecorBase, frameClass: `${chromeDecorBase.frameClass} rounded-none` }
      : chromeDecorBase,
    cursorDecor: buildCursorDecor(style.cursorTheme),
    deviceFrameDecor: hasDeviceFrame ? deviceFrameDecorBase : null,
  };
};
