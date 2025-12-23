import type { ReplayMovieSpec } from '@/types/export';
import { computeReplayLayout } from '@/domains/replay-layout';
import type { ReplayStyleConfig, ReplayStyleOverrides } from '../model';
import { getReplayBackgroundThemeId, normalizeReplayStyle, REPLAY_STYLE_DEFAULTS } from '../model';
import { buildChromeDecor } from '../catalog';

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
    background: decor.background,
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
  const browserFrameRadius = presentation.browser_frame?.radius ?? 24;
  const chromeDecor = buildChromeDecor(style.chromeTheme, '');
  const viewportWidth =
    presentation.viewport?.width ??
    presentation.canvas?.width ??
    0;
  const viewportHeight =
    presentation.viewport?.height ??
    presentation.canvas?.height ??
    0;
  const layout = canvasWidth > 0 && canvasHeight > 0
    ? computeReplayLayout({
        canvas: { width: canvasWidth, height: canvasHeight },
        viewport: {
          width: viewportWidth > 0 ? viewportWidth : canvasWidth,
          height: viewportHeight > 0 ? viewportHeight : canvasHeight,
        },
        browserScale: style.browserScale,
        chromeHeaderHeight: chromeDecor.headerHeight,
        fit: 'none',
      })
    : null;
  const browserFrame = layout
    ? {
        x: Math.round(layout.frameRect.x),
        y: Math.round(layout.frameRect.y),
        width: Math.round(layout.frameRect.width),
        height: Math.round(layout.frameRect.height),
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
      background_theme: getReplayBackgroundThemeId(style.background),
      background: style.background,
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
