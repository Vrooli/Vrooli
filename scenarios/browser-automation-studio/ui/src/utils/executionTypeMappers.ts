/**
 * Type mapping utilities for converting execution timeline data
 * from API responses to strongly-typed frontend structures.
 */

import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { getCachedConfig } from "../config";
import type {
  ReplayFrame,
  ReplayPoint,
  ReplayRetryHistoryEntry,
} from "../features/execution/ReplayPlayer";

declare global {
  interface Window {
    __BAS_EXPORT_API_BASE__?: string;
  }
}

const getForcedExportApiBase = (): string | undefined => {
  if (typeof window === "undefined") {
    return undefined;
  }
  const candidate = window.__BAS_EXPORT_API_BASE__;
  if (typeof candidate !== "string") {
    return undefined;
  }
  const trimmed = candidate.trim();
  return trimmed.length > 0 ? trimmed : undefined;
};

export const toNumber = (value: unknown): number | undefined => {
  return typeof value === "number" ? value : undefined;
};

export const toBoundingBox = (
  value: unknown,
): ReplayFrame["elementBoundingBox"] => {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  const x = toNumber(obj.x);
  const y = toNumber(obj.y);
  const width = toNumber(obj.width);
  const height = toNumber(obj.height);
  if (x == null && y == null && width == null && height == null) {
    return undefined;
  }
  return { x, y, width, height };
};

export const toPoint = (value: unknown): ReplayPoint | undefined => {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  const x = toNumber(obj.x);
  const y = toNumber(obj.y);
  if (x == null || y == null) {
    return undefined;
  }
  return { x, y };
};

export const mapTrail = (value: unknown): ReplayPoint[] => {
  if (!Array.isArray(value)) {
    return [];
  }
  const points: ReplayPoint[] = [];
  for (const entry of value) {
    const point = toPoint(entry);
    if (point) {
      points.push(point);
    }
  }
  return points;
};

export const mapRegion = (
  value: unknown,
): NonNullable<ReplayFrame["highlightRegions"]>[number] | undefined => {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  const boundingBox = toBoundingBox(obj.bounding_box ?? obj.boundingBox);
  const selector = typeof obj.selector === "string" ? obj.selector : undefined;
  if (!selector && !boundingBox) {
    return undefined;
  }
  return {
    selector,
    boundingBox: boundingBox ?? undefined,
    padding: toNumber(obj.padding),
    color: typeof obj.color === "string" ? obj.color : undefined,
    opacity: toNumber(obj.opacity),
  };
};

export const mapRegions = (
  value: unknown,
): NonNullable<ReplayFrame["highlightRegions"]> => {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map(mapRegion).filter(Boolean) as NonNullable<
    ReplayFrame["highlightRegions"]
  >;
};

export const mapRetryHistory = (
  value: unknown,
): ReplayRetryHistoryEntry[] | undefined => {
  if (!Array.isArray(value)) {
    return undefined;
  }
  const entries: ReplayRetryHistoryEntry[] = [];
  for (const item of value) {
    if (!item || typeof item !== "object") {
      continue;
    }
    const obj = item as Record<string, unknown>;
    const attempt = toNumber(obj.attempt ?? obj.attempt_number);
    const success = typeof obj.success === "boolean" ? obj.success : undefined;
    const durationMs = toNumber(obj.duration_ms ?? obj.durationMs);
    const callDurationMs = toNumber(obj.call_duration_ms ?? obj.callDurationMs);
    const error = typeof obj.error === "string" ? obj.error : undefined;
    entries.push({ attempt, success, durationMs, callDurationMs, error });
  }
  return entries.length > 0 ? entries : undefined;
};

export const mapAssertion = (value: unknown): ReplayFrame["assertion"] => {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  return {
    mode: typeof obj.mode === "string" ? obj.mode : undefined,
    selector: typeof obj.selector === "string" ? obj.selector : undefined,
    expected: obj.expected,
    actual: obj.actual,
    success: typeof obj.success === "boolean" ? obj.success : undefined,
    message: typeof obj.message === "string" ? obj.message : undefined,
    negated: typeof obj.negated === "boolean" ? obj.negated : undefined,
    caseSensitive:
      typeof obj.caseSensitive === "boolean" ? obj.caseSensitive : undefined,
  };
};

const ABSOLUTE_URL_PATTERN = /^[a-zA-Z][a-zA-Z\d+.-]*:/;

const API_SUFFIX = "/api/v1";

const isLoopbackHost = (hostname?: string | null): boolean => {
  if (!hostname) {
    return false;
  }
  const value = hostname.toLowerCase();
  return (
    value === "localhost" ||
    value === "127.0.0.1" ||
    value === "0.0.0.0" ||
    value === "::1" ||
    value === "[::1]"
  );
};

const rewriteLoopbackUrl = (url: URL): string => {
  if (typeof window === "undefined" || !window.location) {
    return url.toString();
  }
  const currentHost = window.location.hostname;
  if (
    !currentHost ||
    isLoopbackHost(currentHost) ||
    !isLoopbackHost(url.hostname)
  ) {
    return url.toString();
  }

  // Use canonical info (proxy-aware) instead of window.location.origin
  const canonical = getCanonicalInfo();
  const origin = canonical?.origin ?? window.location.origin.replace(/\/$/, "");
  const pathname = url.pathname.startsWith("/")
    ? url.pathname
    : `/${url.pathname}`;
  const search = url.search ?? "";
  const hash = url.hash ?? "";
  return `${origin}${pathname}${search}${hash}`;
};

type CanonicalInfo = {
  origin: string;
  fullPath: string; // includes suffix when present (no trailing slash)
  rootPath: string; // path before suffix (can be empty string)
};

const getCanonicalInfo = (): CanonicalInfo | null => {
  const forcedBase = getForcedExportApiBase();
  const candidateBases: string[] = [];
  if (forcedBase) {
    const normalized = forcedBase.endsWith(API_SUFFIX)
      ? forcedBase
      : `${forcedBase.replace(/\/$/, "")}${API_SUFFIX}`;
    candidateBases.push(normalized);
  }
  candidateBases.push(resolveApiBase({ appendSuffix: true }));

  for (const base of candidateBases) {
    try {
      const canonicalUrl = new URL(base);
      const normalizedFullPath = canonicalUrl.pathname.endsWith("/")
        ? canonicalUrl.pathname.slice(0, -1)
        : canonicalUrl.pathname;
      const suffixIndex = normalizedFullPath.endsWith(API_SUFFIX)
        ? normalizedFullPath.length - API_SUFFIX.length
        : -1;
      const rootPath =
        suffixIndex >= 0
          ? normalizedFullPath.slice(0, suffixIndex)
          : normalizedFullPath;
      return {
        origin: canonicalUrl.origin,
        fullPath: normalizedFullPath,
        rootPath,
      };
    } catch {
      // try next candidate
    }
  }

  return null;
};

const ensureLeadingSlash = (value: string) =>
  value.startsWith("/") ? value : `/${value}`;

const splitPathAndSuffix = (value: string) => {
  const match = value.match(/^[^?#]*/u);
  const path = match ? match[0] : "";
  const suffix = value.slice(path.length);
  return { path, suffix };
};

const alignPathToCanonical = (
  rawPath: string,
  canonical: CanonicalInfo,
): string => {
  const normalized = ensureLeadingSlash(rawPath);
  if (
    canonical.fullPath &&
    canonical.fullPath.length &&
    normalized.startsWith(canonical.fullPath)
  ) {
    return normalized;
  }
  if (
    canonical.rootPath &&
    canonical.rootPath.length &&
    normalized.startsWith(canonical.rootPath)
  ) {
    return normalized;
  }
  if (canonical.rootPath && canonical.rootPath.length) {
    if (normalized.startsWith(API_SUFFIX)) {
      return `${canonical.rootPath}${normalized}`;
    }
    return `${canonical.rootPath}${normalized}`;
  }
  return normalized;
};

const attemptBuildWithBase = (
  path: string,
  base?: string,
): string | undefined => {
  if (!base) {
    return undefined;
  }
  try {
    const urlString = buildApiUrl(path.startsWith("/") ? path : `/${path}`, {
      baseUrl: base,
      appendSuffix: false,
    });
    return rewriteLoopbackUrl(new URL(urlString));
  } catch {
    return undefined;
  }
};

export const resolveUrl = (url?: string | null): string | undefined => {
  if (!url) {
    return undefined;
  }

  const trimmed = url.trim();
  if (!trimmed) {
    return undefined;
  }

  const canonical = getCanonicalInfo();
  const forcedBase = getForcedExportApiBase();

  if (ABSOLUTE_URL_PATTERN.test(trimmed)) {
    if (typeof window !== "undefined" && window.location) {
      try {
        const parsed = new URL(trimmed);
        if (forcedBase && isLoopbackHost(parsed.hostname)) {
          try {
            const forced = new URL(forcedBase);
            parsed.protocol = forced.protocol;
            parsed.hostname = forced.hostname;
            parsed.port = forced.port;
          } catch {
            // ignore malformed forced base
          }
        }
        if (canonical && parsed.origin === canonical.origin) {
          parsed.pathname = alignPathToCanonical(parsed.pathname, canonical);
          return rewriteLoopbackUrl(parsed);
        }
        return rewriteLoopbackUrl(parsed);
      } catch {
        // fall through and return trimmed value below
      }
    }
    return trimmed;
  }

  if (trimmed.startsWith("//")) {
    const protocol =
      typeof window !== "undefined" && window.location?.protocol
        ? window.location.protocol
        : "https:";
    return `${protocol}${trimmed}`;
  }

  if (forcedBase) {
    const fromForced = attemptBuildWithBase(trimmed, forcedBase);
    if (fromForced) {
      return fromForced;
    }
  }

  if (canonical) {
    const { path, suffix } = splitPathAndSuffix(trimmed);
    const alignedPath = alignPathToCanonical(path, canonical);
    return `${canonical.origin}${alignedPath}${suffix}`;
  }

  const configBase = getCachedConfig()?.API_URL;
  const fromConfig = attemptBuildWithBase(trimmed, configBase);
  if (fromConfig) {
    return fromConfig;
  }

  const resolvedBase = (() => {
    try {
      return resolveApiBase({ appendSuffix: false });
    } catch {
      return undefined;
    }
  })();
  const fromResolved = attemptBuildWithBase(trimmed, resolvedBase);
  if (fromResolved) {
    return fromResolved;
  }

  if (typeof window !== "undefined" && window.location?.origin) {
    const fallback = attemptBuildWithBase(trimmed, window.location.origin);
    if (fallback) {
      return fallback;
    }
  }

  const localFallback = attemptBuildWithBase(trimmed, "http://localhost");
  return localFallback ?? trimmed;
};
