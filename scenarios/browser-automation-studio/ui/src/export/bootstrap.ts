/**
 * Bootstrap utilities for the replay export page.
 *
 * These utilities handle:
 * - Decoding base64 and JSON payloads
 * - Reading bootstrap payloads from the window object
 * - Initializing the basExport API on window
 */

import type { ReplayMovieSpec } from "@/types/export";
import type { BootstrapPayload, ExportMetadata } from "./types";
import { logger } from "@/utils/logger";

/**
 * Adds padding to a base64 string to make it valid.
 */
export const addPadding = (base64: string): string => {
  const padding = (4 - (base64.length % 4)) % 4;
  if (padding === 0) {
    return base64;
  }
  return `${base64}${"=".repeat(padding)}`;
};

/**
 * Decodes a base64-encoded movie spec payload.
 */
export const decodeExportPayload = (encoded: string): ReplayMovieSpec | null => {
  if (!encoded) {
    return null;
  }
  try {
    const normalized = addPadding(
      encoded.replace(/-/g, "+").replace(/_/g, "/"),
    );
    if (typeof window === "undefined" || typeof window.atob !== "function") {
      return null;
    }
    const json = window.atob(normalized);
    return JSON.parse(json) as ReplayMovieSpec;
  } catch (error) {
    logger.error(
      "Failed to decode replay export payload",
      { component: "ReplayExportPage" },
      error,
    );
    return null;
  }
};

/**
 * Parses a JSON string into a movie spec.
 */
export const decodeJsonSpec = (
  raw: string | null | undefined,
): ReplayMovieSpec | null => {
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as ReplayMovieSpec;
  } catch (error) {
    logger.error(
      "Failed to parse replay export bootstrap payload",
      { component: "ReplayExportPage" },
      error,
    );
    return null;
  }
};

/**
 * Gets the bootstrap payload from the window object.
 */
export const getBootstrapPayload = (): BootstrapPayload | null => {
  if (typeof window === "undefined") {
    return null;
  }
  const candidate = window.__BAS_EXPORT_BOOTSTRAP__;
  if (!candidate || typeof candidate !== "object") {
    return null;
  }
  const payload = candidate as BootstrapPayload;
  return {
    payloadJson:
      typeof payload.payloadJson === "string" ? payload.payloadJson : undefined,
    payloadBase64:
      typeof payload.payloadBase64 === "string"
        ? payload.payloadBase64
        : undefined,
    apiBase: typeof payload.apiBase === "string" ? payload.apiBase : undefined,
    executionId:
      typeof payload.executionId === "string" ? payload.executionId : undefined,
  };
};

/**
 * Resolves the movie spec from the bootstrap payload.
 */
export const resolveBootstrapSpec = (): {
  spec: ReplayMovieSpec | null;
  executionId: string | null;
  apiBase: string | null;
} => {
  const payload = getBootstrapPayload();
  if (!payload) {
    return {
      spec: null,
      executionId: null,
      apiBase: null,
    };
  }
  let spec = decodeJsonSpec(payload.payloadJson);
  if (!spec && payload.payloadBase64) {
    spec = decodeExportPayload(payload.payloadBase64);
  }
  const executionId = payload.executionId?.trim() || null;
  const apiBase = payload.apiBase?.trim() || null;
  return { spec, executionId, apiBase };
};

/**
 * Ensures the basExport API is initialized on window.
 */
export const ensureBasExportBootstrap = (): void => {
  if (typeof window === "undefined") {
    return;
  }
  if (!window.basExport) {
    window.basExport = {
      ready: false,
      error: "initializing",
      async seekTo() {
        throw new Error("Replay export controller not ready");
      },
      play() {
        throw new Error("Replay export controller not ready");
      },
      pause() {
        throw new Error("Replay export controller not ready");
      },
      getViewportRect() {
        return { x: 0, y: 0, width: 0, height: 0 };
      },
      getMetadata(): ExportMetadata {
        return {
          totalDurationMs: 0,
          frameCount: 0,
          timeline: [],
          width: 0,
          height: 0,
          canvasWidth: 0,
          canvasHeight: 0,
          browserFrame: {
            x: 0,
            y: 0,
            width: 0,
            height: 0,
            radius: 0,
          },
          assetCount: 0,
          specId: null,
          deviceScaleFactor: 1,
        };
      },
      getCurrentState() {
        return { frameIndex: 0, progress: 0 };
      },
    };
  }
};
