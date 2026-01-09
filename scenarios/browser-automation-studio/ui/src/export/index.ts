/**
 * Export module - utilities for replay export page and composer.
 *
 * Structure:
 * - types.ts: Type definitions for timeline, metadata, payloads
 * - bootstrap.ts: Bootstrap utilities for composer initialization
 * - timeline.ts: Timeline computation utilities
 * - frameMapping.ts: Frame mapping and settings utilities
 * - status.ts: Status normalization utilities
 */

// Types
export type {
  BootstrapPayload,
  ExportMetadata,
  ExportPreviewPayload,
  FrameTimeline,
  FrameWaiter,
  PresentationBounds,
} from "./types";

// Bootstrap utilities
export {
  addPadding,
  decodeExportPayload,
  decodeJsonSpec,
  ensureBasExportBootstrap,
  getBootstrapPayload,
  resolveBootstrapSpec,
} from "./bootstrap";

// Timeline utilities
export {
  DEFAULT_FRAME_DURATION_MS,
  buildTimeline,
  clampProgress,
  computeTotalDuration,
  findFrameForTime,
} from "./timeline";

// Frame mapping utilities
export {
  DEFAULT_INTRO_CARD_SETTINGS,
  DEFAULT_OUTRO_CARD_SETTINGS,
  DEFAULT_WATERMARK_SETTINGS,
  isWatermarkPosition,
  mapIntroCardSettings,
  mapOutroCardSettings,
  mapWatermarkSettings,
  resolveAssetUrl,
  toReplayFrame,
} from "./frameMapping";

// Status utilities
export { defaultStatusMessage, normalizeStatus } from "./status";
