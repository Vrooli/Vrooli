// @ts-check

/**
 * Shared utility helpers for the web-console UI.
 */

import { resolveApiBase, resolveWsBase, buildApiUrl } from '@vrooli/api-base'

const sharedTextDecoder = new TextDecoder();
const sharedTextEncoder = new TextEncoder();

export const textDecoder = sharedTextDecoder;
export const textEncoder = sharedTextEncoder;

/** @type {ReturnType<typeof setTimeout> | null} */
let resourceTimingCleanupTimer = null;

function scheduleResourceTimingCleanup() {
  if (typeof window === "undefined" || typeof performance === "undefined") {
    return;
  }
  if (typeof performance.clearResourceTimings !== "function") {
    return;
  }
  if (resourceTimingCleanupTimer) {
    return;
  }
  resourceTimingCleanupTimer = window.setTimeout(() => {
    resourceTimingCleanupTimer = null;
    try {
      performance.clearResourceTimings();
    } catch (error) {
      console.warn("Failed to clear resource timings:", error);
    }
  }, 15000);
}

// Resolve API base URL once at module load
const API_BASE = resolveApiBase({ appendSuffix: false })

/**
 * @param {string} path
 * @param {{ method?: string; json?: unknown; headers?: HeadersInit }} [options]
 * @returns {Promise<Response>}
 */
export function proxyToApi(path, { method = "GET", json, headers } = {}) {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const requestHeaders = headers ? new Headers(headers) : new Headers();
  /** @type {RequestInit} */
  const requestInit = {
    method,
    headers: requestHeaders,
  };
  if (json !== undefined) {
    requestHeaders.set("Content-Type", "application/json");
    requestInit.body = JSON.stringify(json);
  }
  const request = fetch(url, requestInit);
  scheduleResourceTimingCleanup();
  return request;
}

/**
 * @param {string} path
 * @returns {string}
 */
export function buildWebSocketUrl(path) {
  // Use resolveWsBase to get the WebSocket base URL
  const WS_BASE = resolveWsBase({ appendSuffix: false });
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${WS_BASE}${normalized}`;
}

/**
 * @param {unknown} value
 * @returns {value is ArrayBufferView}
 */
export function isArrayBufferView(value) {
  return (
    typeof value === "object" &&
    value !== null &&
    ArrayBuffer.isView(/** @type {ArrayBufferView} */ (value))
  );
}

/**
 * @param {unknown} data
 * @param {(error: unknown) => void} [onError]
 * @returns {Promise<string>}
 */
export async function normalizeSocketData(data, onError) {
  if (typeof data === "string") {
    return data;
  }
  try {
    if (data instanceof Blob && typeof data.text === "function") {
      return await data.text();
    }
    if (data instanceof ArrayBuffer) {
      return sharedTextDecoder.decode(data);
    }
    if (isArrayBufferView(data)) {
      const view = data;
      const buffer =
        view.byteOffset === 0 && view.byteLength === view.buffer.byteLength
          ? view.buffer
          : view.buffer.slice(
              view.byteOffset,
              view.byteOffset + view.byteLength,
            );
      return sharedTextDecoder.decode(buffer);
    }
  } catch (error) {
    if (typeof onError === "function") {
      onError(error);
    } else {
      console.error("Failed to normalize socket data:", error);
    }
  }
  return "";
}

/**
 * @param {string} command
 * @param {unknown} args
 * @returns {string}
 */
export function formatCommandLabel(command, args) {
  if (!command) return "—";
  if (Array.isArray(args) && args.length > 0) {
    return `${command} ${args.join(" ")}`;
  }
  return command;
}

/**
 * @param {unknown} payload
 * @returns {string}
 */
export function formatEventPayload(payload) {
  if (payload === undefined || payload === null) {
    return "";
  }
  if (typeof payload === "string") {
    return payload;
  }
  try {
    return JSON.stringify(payload, null, 2);
  } catch (_error) {
    return "[unserializable payload]";
  }
}

/**
 * @param {number | string | Date} value
 * @param {Intl.RelativeTimeFormat} [formatter]
 * @returns {string}
 */
export function formatRelativeTimestamp(value, formatter) {
  if (!value) return "";
  try {
    const timestamp =
      typeof value === "number"
        ? value
        : value instanceof Date
          ? value.getTime()
          : Date.parse(String(value));
    if (Number.isNaN(timestamp)) {
      return "";
    }
    if (!formatter) {
      return new Date(timestamp).toLocaleTimeString();
    }
    const diffSeconds = Math.round((timestamp - Date.now()) / 1000);
    const absSeconds = Math.abs(diffSeconds);
    if (absSeconds < 60) {
      return formatter.format(diffSeconds, "second");
    }
    if (absSeconds < 3600) {
      return formatter.format(Math.round(diffSeconds / 60), "minute");
    }
    if (absSeconds < 86400) {
      return formatter.format(Math.round(diffSeconds / 3600), "hour");
    }
    return formatter.format(Math.round(diffSeconds / 86400), "day");
  } catch (_error) {
    return "";
  }
}

/**
 * @param {number | string | Date} value
 * @returns {string}
 */
export function formatAbsoluteTime(value) {
  if (!value) return "";
  try {
    const timestamp =
      typeof value === "number"
        ? value
        : value instanceof Date
          ? value.getTime()
          : Date.parse(String(value));
    if (Number.isNaN(timestamp)) {
      return "";
    }
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch (_error) {
    return "";
  }
}

/**
 * @param {unknown} raw
 * @param {string} fallback
 * @returns {{ message: string; raw?: unknown }}
 */
export function parseApiError(raw, fallback) {
  if (!raw) {
    return { message: fallback };
  }
  const trimmed = typeof raw === "string" ? raw.trim() : "";
  if (!trimmed) {
    return { message: fallback };
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (
      parsed &&
      typeof parsed === "object" &&
      typeof parsed.message === "string"
    ) {
      return { message: parsed.message.trim() || fallback, raw: parsed };
    }
  } catch (_error) {
    // Ignore JSON parse failures
  }
  return { message: trimmed || fallback };
}

/** @type {(callback: () => void) => void} */
export const scheduleMicrotask =
  typeof queueMicrotask === "function"
    ? queueMicrotask
    : (cb) => {
        Promise.resolve().then(() => {
          cb();
        });
      };
