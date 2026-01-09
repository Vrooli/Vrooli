/**
 * Types for the replay export page and composer.
 *
 * These types define the structures used for:
 * - Timeline and frame tracking during playback
 * - Export metadata returned to the parent frame
 * - Bootstrap payloads for initializing the composer
 * - Message communication with parent frames
 */

import type { ReplayMovieSpec } from "@/types/export";

/**
 * Timeline entry for a single frame.
 */
export interface FrameTimeline {
  index: number;
  startMs: number;
  durationMs: number;
}

/**
 * Waiter for async frame seek operations.
 */
export interface FrameWaiter {
  index: number;
  progress: number;
  resolve: () => void;
  reject: (error: Error) => void;
  timeoutId: number;
}

/**
 * Metadata about the export for external consumers.
 */
export interface ExportMetadata {
  totalDurationMs: number;
  frameCount: number;
  timeline: FrameTimeline[];
  width: number;
  height: number;
  canvasWidth: number;
  canvasHeight: number;
  browserFrame: {
    x: number;
    y: number;
    width: number;
    height: number;
    radius?: number;
  };
  deviceScaleFactor?: number;
  assetCount: number;
  specId?: string | null;
}

/**
 * Payload structure for export preview API responses.
 */
export interface ExportPreviewPayload {
  status?: string | null;
  message?: string | null;
  package?: ReplayMovieSpec | null;
  execution_id?: string | null;
}

/**
 * Container dimensions for presentation bounds.
 */
export interface PresentationBounds {
  width: number;
  height: number;
}

/**
 * Bootstrap payload for initializing the composer.
 */
export interface BootstrapPayload {
  payloadJson?: string | null;
  payloadBase64?: string | null;
  apiBase?: string | null;
  executionId?: string | null;
}

/**
 * Global window augmentation for the export API.
 */
declare global {
  interface Window {
    basExport?: {
      ready: boolean;
      error?: string | null;
      seekTo: (ms: number) => Promise<void>;
      play: () => void;
      pause: () => void;
      getViewportRect: () => {
        x: number;
        y: number;
        width: number;
        height: number;
      };
      getMetadata: () => ExportMetadata;
      getCurrentState: () => { frameIndex: number; progress: number };
    };
    __BAS_EXPORT_API_BASE__?: string;
    __BAS_EXPORT_BOOTSTRAP__?: BootstrapPayload | null;
  }
}
