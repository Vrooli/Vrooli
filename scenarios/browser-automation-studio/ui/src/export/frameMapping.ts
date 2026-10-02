/**
 * Frame mapping utilities for the replay export page.
 *
 * Pure functions for converting movie spec data to replay frames
 * and resolving asset URLs.
 */

import type { ReplayAsset as ReplayMovieAsset, ReplayFrame as ReplayMovieFrame, ReplayIntroCard as ExportIntroCard, ReplayOutroCard as ExportOutroCard, ReplayWatermark as ExportWatermark } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import type { ReplayFrame } from "@/domains/exports/replay/ReplayPlayer";
import type {
  IntroCardSettings,
  OutroCardSettings,
  WatermarkSettings,
  WatermarkPosition,
} from "@/stores/settingsStore";
import {
  mapAssertion,
  mapRegions,
  mapRetryHistory,
  mapTrail,
  resolveUrl,
  toBoundingBox,
  toNumber,
  toPoint,
} from "@/utils/executionTypeMappers";
import { DEFAULT_FRAME_DURATION_MS } from "./timeline";

/** Default watermark settings */
export const DEFAULT_WATERMARK_SETTINGS: WatermarkSettings = {
  enabled: false,
  assetId: null,
  position: "bottom-right",
  size: 12,
  opacity: 80,
  margin: 16,
};

/** Default intro card settings */
export const DEFAULT_INTRO_CARD_SETTINGS: IntroCardSettings = {
  enabled: false,
  title: "",
  subtitle: "",
  logoAssetId: null,
  backgroundAssetId: null,
  backgroundColor: "#0f172a",
  textColor: "#ffffff",
  duration: 2000,
};

/** Default outro card settings */
export const DEFAULT_OUTRO_CARD_SETTINGS: OutroCardSettings = {
  enabled: false,
  title: "Thanks for watching!",
  ctaText: "Learn More",
  ctaUrl: "",
  logoAssetId: null,
  backgroundAssetId: null,
  backgroundColor: "#0f172a",
  textColor: "#ffffff",
  duration: 3000,
};

/**
 * Type guard for watermark position.
 */
export const isWatermarkPosition = (value: unknown): value is WatermarkPosition =>
  value === "top-left" ||
  value === "top-right" ||
  value === "bottom-left" ||
  value === "bottom-right" ||
  value === "center";

/**
 * Resolves the URL from an asset.
 */
export const resolveAssetUrl = (
  asset: ReplayMovieAsset | undefined,
): string | undefined => {
  if (!asset) {
    return undefined;
  }
  const candidates = [asset.source, asset.thumbnail];
  for (const candidate of candidates) {
    if (!candidate || typeof candidate !== "string") {
      continue;
    }
    const resolved = resolveUrl(candidate) ?? candidate;
    if (resolved) {
      return resolved;
    }
  }
  return undefined;
};

/**
 * Maps watermark spec to settings.
 */
export const mapWatermarkSettings = (
  watermark?: ExportWatermark | null,
): WatermarkSettings | undefined => {
  if (!watermark) {
    return undefined;
  }
  const position = isWatermarkPosition(watermark.position)
    ? watermark.position
    : DEFAULT_WATERMARK_SETTINGS.position;
  return {
    ...DEFAULT_WATERMARK_SETTINGS,
    enabled: Boolean(watermark.enabled),
    assetId: watermark.assetId ?? DEFAULT_WATERMARK_SETTINGS.assetId,
    position,
    size: watermark.size ?? DEFAULT_WATERMARK_SETTINGS.size,
    opacity: watermark.opacity ?? DEFAULT_WATERMARK_SETTINGS.opacity,
    margin: watermark.margin ?? DEFAULT_WATERMARK_SETTINGS.margin,
  };
};

/**
 * Maps intro card spec to settings.
 */
export const mapIntroCardSettings = (
  intro?: ExportIntroCard | null,
): IntroCardSettings | undefined => {
  if (!intro) {
    return undefined;
  }
  return {
    ...DEFAULT_INTRO_CARD_SETTINGS,
    enabled: Boolean(intro.enabled),
    title: intro.title ?? DEFAULT_INTRO_CARD_SETTINGS.title,
    subtitle: intro.subtitle ?? DEFAULT_INTRO_CARD_SETTINGS.subtitle,
    logoAssetId: intro.logoAssetId ?? DEFAULT_INTRO_CARD_SETTINGS.logoAssetId,
    backgroundAssetId:
      intro.backgroundAssetId ?? DEFAULT_INTRO_CARD_SETTINGS.backgroundAssetId,
    backgroundColor:
      intro.backgroundColor ?? DEFAULT_INTRO_CARD_SETTINGS.backgroundColor,
    textColor: intro.textColor ?? DEFAULT_INTRO_CARD_SETTINGS.textColor,
    duration: intro.durationMs ?? DEFAULT_INTRO_CARD_SETTINGS.duration,
  };
};

/**
 * Maps outro card spec to settings.
 */
export const mapOutroCardSettings = (
  outro?: ExportOutroCard | null,
): OutroCardSettings | undefined => {
  if (!outro) {
    return undefined;
  }
  return {
    ...DEFAULT_OUTRO_CARD_SETTINGS,
    enabled: Boolean(outro.enabled),
    title: outro.title ?? DEFAULT_OUTRO_CARD_SETTINGS.title,
    ctaText: outro.ctaText ?? DEFAULT_OUTRO_CARD_SETTINGS.ctaText,
    ctaUrl: outro.ctaUrl ?? DEFAULT_OUTRO_CARD_SETTINGS.ctaUrl,
    logoAssetId: outro.logoAssetId ?? DEFAULT_OUTRO_CARD_SETTINGS.logoAssetId,
    backgroundAssetId:
      outro.backgroundAssetId ?? DEFAULT_OUTRO_CARD_SETTINGS.backgroundAssetId,
    backgroundColor:
      outro.backgroundColor ?? DEFAULT_OUTRO_CARD_SETTINGS.backgroundColor,
    textColor: outro.textColor ?? DEFAULT_OUTRO_CARD_SETTINGS.textColor,
    duration: outro.durationMs ?? DEFAULT_OUTRO_CARD_SETTINGS.duration,
  };
};

/**
 * Converts a movie frame to a replay frame.
 */
export const toReplayFrame = (
  frame: ReplayMovieFrame,
  index: number,
  assetMap: Map<string, ReplayMovieAsset>,
): ReplayFrame => {
  const screenshotId =
    typeof frame.screenshotAssetId === "string"
      ? frame.screenshotAssetId
      : undefined;
  const asset = screenshotId ? assetMap.get(screenshotId) : undefined;
  const screenshotUrl = resolveAssetUrl(asset);
  const durationMs = toNumber(frame.durationMs) ?? DEFAULT_FRAME_DURATION_MS;
  const holdMs = toNumber(frame.holdMs) ?? 0;
  const totalDurationMs = durationMs + holdMs;
  const boundingBox = toBoundingBox(frame.elementBoundingBox);
  const focusedElementBox = frame.focusedElement?.boundingBox;
  const focusedBoundingBox = toBoundingBox(focusedElementBox);
  const clickPosition = toPoint(frame.clickPosition);
  const cursorTrail = mapTrail(
    frame.cursorTrail ?? frame.normalizedCursorTrail,
  );
  const retry = frame.resilience;

  return {
    id: frame.index != null ? String(frame.index) : `frame-${index}`,
    stepIndex: toNumber(frame.stepIndex) ?? index,
    nodeId: typeof frame.nodeId === "string" ? frame.nodeId : undefined,
    stepType: typeof frame.stepType === "string" ? frame.stepType : undefined,
    status: typeof frame.status === "string" ? frame.status : undefined,
    success: (frame.status ?? "").toLowerCase() !== "failed",
    durationMs,
    totalDurationMs,
    progress: 0,
    finalUrl: typeof frame.finalUrl === "string" ? frame.finalUrl : undefined,
    error: typeof frame.error === "string" ? frame.error : undefined,
    extractedDataPreview: undefined,
    consoleLogCount: toNumber(frame.consoleLogCount),
    networkEventCount: toNumber(frame.networkEventCount),
    screenshot: screenshotUrl
      ? {
          artifactId: screenshotId ?? `artifact-${index}`,
          url: screenshotUrl,
          thumbnailUrl: resolveAssetUrl(asset),
          width: toNumber(asset?.width),
          height: toNumber(asset?.height),
          contentType:
            typeof asset?.type === "string" ? asset?.type : undefined,
          sizeBytes: toNumber(asset?.sizeBytes),
        }
      : undefined,
    highlightRegions: mapRegions(frame.highlightRegions),
    maskRegions: mapRegions(frame.maskRegions),
    focusedElement: focusedBoundingBox
      ? {
          selector:
            typeof frame.focusedElement?.selector === "string"
              ? frame.focusedElement.selector
              : undefined,
          boundingBox: focusedBoundingBox,
        }
      : null,
    elementBoundingBox: boundingBox ?? null,
    clickPosition: clickPosition ?? null,
    cursorTrail,
    zoomFactor: toNumber(frame.zoomFactor),
    assertion: mapAssertion(frame.assertion),
    retryAttempt: toNumber(retry?.attempt),
    retryMaxAttempts: toNumber(retry?.maxAttempts),
    retryConfigured: toNumber(retry?.configuredRetries),
    retryDelayMs: toNumber(retry?.delayMs),
    retryBackoffFactor: toNumber(retry?.backoffFactor),
    retryHistory: mapRetryHistory(retry?.history),
    domSnapshotPreview: undefined,
    domSnapshotHtml: undefined,
    domSnapshotArtifactId: undefined,
  };
};
