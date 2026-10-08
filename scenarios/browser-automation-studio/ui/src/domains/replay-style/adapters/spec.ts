import type { ReplaySpec as ReplayMovieSpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import type { JsonObject, JsonValue } from '@bufbuild/protobuf';
import { create } from '@bufbuild/protobuf';
import {
  ReplayCursorMotionSchema,
  ReplayCursorSchema,
  ReplayDecorSchema,
  ReplayFrameRectSchema,
  ReplayPresentationSchema,
} from '@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb';
import { computeReplayLayout } from '@/domains/replay-layout';
import type { ReplayStyleConfig, ReplayStyleOverrides } from '../model';
import { REPLAY_BACKGROUND_THEME_IDS } from '../model';
import type { ReplayBackgroundSource } from '../model';
import {
  getReplayBackgroundThemeId,
  normalizeReplayStyle,
  REPLAY_STYLE_DEFAULTS,
  resolveReplayPresentationStyle,
} from '../model';
import { buildChromeDecor } from '../catalog';

const backgroundToJson = (background: ReplayBackgroundSource): JsonObject => {
  switch (background.type) {
    case 'theme':
      return { type: background.type, id: background.id };
    case 'gradient':
      return {
        type: background.type,
        value: {
          type: background.value.type,
          ...(background.value.angle === undefined ? {} : { angle: background.value.angle }),
          ...(background.value.center === undefined ? {} : { center: background.value.center }),
          stops: background.value.stops.map((stop) => ({
            color: stop.color,
            ...(stop.position === undefined ? {} : { position: stop.position }),
          })),
        },
      };
    case 'image':
      return {
        type: background.type,
        ...(background.assetId === undefined ? {} : { assetId: background.assetId }),
        ...(background.url === undefined ? {} : { url: background.url }),
        ...(background.fit === undefined ? {} : { fit: background.fit }),
      };
  }
};

const jsonObject = (value: JsonValue | undefined): JsonObject | undefined =>
  value !== null && typeof value === 'object' && !Array.isArray(value) ? value : undefined;

const jsonString = (value: JsonValue | undefined): string | undefined =>
  typeof value === 'string' ? value : undefined;

const jsonNumber = (value: JsonValue | undefined): number | undefined =>
  typeof value === 'number' && Number.isFinite(value) ? value : undefined;

const backgroundFromJson = (value: JsonObject | undefined): ReplayBackgroundSource | undefined => {
  if (!value) return undefined;
  const type = jsonString(value.type);
  if (type === 'theme') {
    const id = jsonString(value.id);
    const theme = REPLAY_BACKGROUND_THEME_IDS.find((candidate) => candidate === id);
    return theme ? { type, id: theme } : undefined;
  }
  if (type === 'gradient') {
    const gradient = jsonObject(value.value);
    const gradientType = jsonString(gradient?.type);
    const stopsValue = gradient?.stops;
    if ((gradientType !== 'linear' && gradientType !== 'radial') || !Array.isArray(stopsValue)) return undefined;
    const stops = stopsValue.flatMap((entry) => {
      const stop = jsonObject(entry);
      const color = jsonString(stop?.color);
      if (!color) return [];
      const position = jsonNumber(stop?.position);
      return [{ color, ...(position === undefined ? {} : { position }) }];
    });
    const centerValue = jsonObject(gradient?.center);
    const centerX = jsonNumber(centerValue?.x);
    const centerY = jsonNumber(centerValue?.y);
    const angle = jsonNumber(gradient?.angle);
    return {
      type,
      value: {
        type: gradientType,
        ...(angle === undefined ? {} : { angle }),
        ...(centerX === undefined || centerY === undefined ? {} : { center: { x: centerX, y: centerY } }),
        stops,
      },
    };
  }
  if (type === 'image') {
    const fit = jsonString(value.fit);
    return {
      type,
      ...(jsonString(value.assetId) === undefined ? {} : { assetId: jsonString(value.assetId) }),
      ...(jsonString(value.url) === undefined ? {} : { url: jsonString(value.url) }),
      ...(fit === 'cover' || fit === 'contain' ? { fit } : {}),
    };
  }
  return undefined;
};

export const resolveReplayStyleFromSpec = (
  spec: ReplayMovieSpec | null | undefined,
): ReplayStyleConfig => {
  if (!spec) {
    return REPLAY_STYLE_DEFAULTS;
  }
  const decor = spec.decor;
  const motion = spec.cursorMotion;
  const cursor = spec.cursor;
  return normalizeReplayStyle({
    chromeTheme: decor?.chromeTheme,
    background: backgroundFromJson(decor?.background),
    cursorTheme: decor?.cursorTheme,
    cursorInitialPosition:
      decor?.cursorInitialPosition ?? motion?.initialPosition ?? cursor?.initialPosition,
    cursorClickAnimation:
      decor?.cursorClickAnimation ?? motion?.clickAnimation ?? cursor?.clickAnimation,
    cursorScale: decor?.cursorScale ?? motion?.cursorScale ?? cursor?.scale,
  });
};

export const applyReplayStyleToSpec = (
  spec: ReplayMovieSpec,
  styleOverrides: ReplayStyleOverrides,
): ReplayMovieSpec => {
  const style = normalizeReplayStyle(styleOverrides, REPLAY_STYLE_DEFAULTS);
  const cursorSpec = spec.cursor;
  const motion = spec.cursorMotion;
  const presentation = spec.presentation;

  const canvasWidth =
    presentation?.canvas?.width ??
    presentation?.viewport?.width ??
    0;
  const canvasHeight =
    presentation?.canvas?.height ??
    presentation?.viewport?.height ??
    0;
  const presentationStyle = resolveReplayPresentationStyle(style);
  const browserFrameRadius = presentationStyle.presentation.showBrowserFrame
    && presentationStyle.presentation.showDesktop
    ? presentation?.browserFrame?.radius ?? 24
    : 0;
  const chromeDecor = buildChromeDecor(presentationStyle.chromeTheme, '');
  const viewportWidth =
    presentation?.viewport?.width ??
    presentation?.canvas?.width ??
    0;
  const viewportHeight =
    presentation?.viewport?.height ??
    presentation?.canvas?.height ??
    0;
  const layout = canvasWidth > 0 && canvasHeight > 0
    ? computeReplayLayout({
        canvas: { width: canvasWidth, height: canvasHeight },
        viewport: {
          width: viewportWidth > 0 ? viewportWidth : canvasWidth,
          height: viewportHeight > 0 ? viewportHeight : canvasHeight,
        },
        browserScale: presentationStyle.browserScale,
        chromeHeaderHeight: chromeDecor.headerHeight,
        fit: 'none',
      })
    : null;
  const browserFrame = layout
    ? create(ReplayFrameRectSchema, {
        x: Math.round(layout.frameRect.x),
        y: Math.round(layout.frameRect.y),
        width: Math.round(layout.frameRect.width),
        height: Math.round(layout.frameRect.height),
        radius: browserFrameRadius,
      })
    : presentation?.browserFrame;

  return {
    ...spec,
    cursor: create(ReplayCursorSchema, {
      style: cursorSpec?.style,
      accentColor: cursorSpec?.accentColor,
      trail: cursorSpec?.trail,
      clickPulse: cursorSpec?.clickPulse,
      scale: style.cursorScale,
      initialPosition: style.cursorInitialPosition,
      clickAnimation: style.cursorClickAnimation,
    }),
    decor: create(ReplayDecorSchema, {
      chromeTheme: presentationStyle.chromeTheme,
      backgroundTheme: getReplayBackgroundThemeId(presentationStyle.background),
      background: backgroundToJson(presentationStyle.background),
      cursorTheme: style.cursorTheme,
      cursorInitialPosition: style.cursorInitialPosition,
      cursorClickAnimation: style.cursorClickAnimation,
      cursorScale: style.cursorScale,
    }),
    cursorMotion: create(ReplayCursorMotionSchema, {
      speedProfile: motion?.speedProfile,
      pathStyle: motion?.pathStyle,
      initialPosition: style.cursorInitialPosition,
      clickAnimation: style.cursorClickAnimation,
      cursorScale: style.cursorScale,
    }),
    presentation: browserFrame
      ? create(ReplayPresentationSchema, {
          canvas: presentation?.canvas,
          viewport: presentation?.viewport,
          browserFrame,
          deviceScaleFactor: presentation?.deviceScaleFactor,
        })
      : presentation,
  };
};
