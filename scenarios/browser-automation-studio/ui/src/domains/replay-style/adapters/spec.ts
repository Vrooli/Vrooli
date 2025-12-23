import type { ReplayMovieSpec } from '@/types/export';
import type { ReplayStyleConfig, ReplayStyleOverrides } from '../model';
import { clampReplayBrowserScale, normalizeReplayStyle, REPLAY_STYLE_DEFAULTS } from '../model';

export const resolveReplayStyleFromSpec = (
  spec: ReplayMovieSpec | null | undefined,
): ReplayStyleConfig => {
  if (!spec) {
    return REPLAY_STYLE_DEFAULTS;
  }
  const decor = spec.decor ?? {};
  const motion = spec.cursor_motion ?? {};
  const cursor = spec.cursor ?? {};
  return normalizeReplayStyle({
    chromeTheme: decor.chrome_theme,
    backgroundTheme: decor.background_theme,
    cursorTheme: decor.cursor_theme,
    cursorInitialPosition:
      decor.cursor_initial_position ?? motion.initial_position ?? cursor.initial_position,
    cursorClickAnimation:
      decor.cursor_click_animation ?? motion.click_animation ?? cursor.click_animation,
    cursorScale: decor.cursor_scale ?? motion.cursor_scale ?? cursor.scale,
  });
};

export const applyReplayStyleToSpec = (
  spec: ReplayMovieSpec,
  styleOverrides: ReplayStyleOverrides,
): ReplayMovieSpec => {
  const style = normalizeReplayStyle(styleOverrides, REPLAY_STYLE_DEFAULTS);
  const cursorSpec = spec.cursor ?? {};
  const decor = spec.decor ?? {};
  const motion = spec.cursor_motion ?? {};
  const presentation = spec.presentation ?? {};

  const canvasWidth =
    presentation.canvas?.width ??
    presentation.viewport?.width ??
    0;
  const canvasHeight =
    presentation.canvas?.height ??
    presentation.viewport?.height ??
    0;
  const clampedBrowserScale = clampReplayBrowserScale(style.browserScale);
  const browserWidth = canvasWidth > 0 ? Math.round(canvasWidth * clampedBrowserScale) : 0;
  const browserHeight = canvasHeight > 0 ? Math.round(canvasHeight * clampedBrowserScale) : 0;
  const browserFrameRadius = presentation.browser_frame?.radius ?? 24;
  const browserFrame = canvasWidth > 0 && canvasHeight > 0
    ? {
        x: Math.round((canvasWidth - browserWidth) / 2),
        y: Math.round((canvasHeight - browserHeight) / 2),
        width: browserWidth,
        height: browserHeight,
        radius: browserFrameRadius,
      }
    : presentation.browser_frame;

  return {
    ...spec,
    cursor: {
      ...cursorSpec,
      scale: style.cursorScale,
      initial_position: style.cursorInitialPosition,
      click_animation: style.cursorClickAnimation,
    },
    decor: {
      ...decor,
      chrome_theme: style.chromeTheme,
      background_theme: style.backgroundTheme,
      cursor_theme: style.cursorTheme,
      cursor_initial_position: style.cursorInitialPosition,
      cursor_click_animation: style.cursorClickAnimation,
      cursor_scale: style.cursorScale,
    },
    cursor_motion: {
      ...motion,
      initial_position: style.cursorInitialPosition,
      click_animation: style.cursorClickAnimation,
      cursor_scale: style.cursorScale,
    },
    presentation: browserFrame
      ? {
          ...presentation,
          browser_frame: browserFrame,
        }
      : presentation,
  };
};
