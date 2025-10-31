/**
 * Type mapping utilities for converting execution timeline data
 * from API responses to strongly-typed frontend structures.
 */

import { resolveApiBase } from '@vrooli/api-base';
import type { ReplayFrame, ReplayPoint, ReplayRetryHistoryEntry } from '../components/ReplayPlayer';

export const toNumber = (value: unknown): number | undefined => {
  return typeof value === 'number' ? value : undefined;
};

export const toBoundingBox = (value: unknown): ReplayFrame['elementBoundingBox'] => {
  if (!value || typeof value !== 'object') {
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
  if (!value || typeof value !== 'object') {
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

export const mapRegion = (value: unknown): NonNullable<ReplayFrame['highlightRegions']>[number] | undefined => {
  if (!value || typeof value !== 'object') {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  const boundingBox = toBoundingBox(obj.bounding_box ?? obj.boundingBox);
  const selector = typeof obj.selector === 'string' ? obj.selector : undefined;
  if (!selector && !boundingBox) {
    return undefined;
  }
  return {
    selector,
    boundingBox: boundingBox ?? undefined,
    padding: toNumber(obj.padding),
    color: typeof obj.color === 'string' ? obj.color : undefined,
    opacity: toNumber(obj.opacity),
  };
};

export const mapRegions = (value: unknown): NonNullable<ReplayFrame['highlightRegions']> => {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map(mapRegion).filter(Boolean) as NonNullable<ReplayFrame['highlightRegions']>;
};

export const mapRetryHistory = (value: unknown): ReplayRetryHistoryEntry[] | undefined => {
  if (!Array.isArray(value)) {
    return undefined;
  }
  const entries: ReplayRetryHistoryEntry[] = [];
  for (const item of value) {
    if (!item || typeof item !== 'object') {
      continue;
    }
    const obj = item as Record<string, unknown>;
    const attempt = toNumber(obj.attempt ?? obj.attempt_number);
    const success = typeof obj.success === 'boolean' ? obj.success : undefined;
    const durationMs = toNumber(obj.duration_ms ?? obj.durationMs);
    const callDurationMs = toNumber(obj.call_duration_ms ?? obj.callDurationMs);
    const error = typeof obj.error === 'string' ? obj.error : undefined;
    entries.push({ attempt, success, durationMs, callDurationMs, error });
  }
  return entries.length > 0 ? entries : undefined;
};

export const mapAssertion = (value: unknown): ReplayFrame['assertion'] => {
  if (!value || typeof value !== 'object') {
    return undefined;
  }
  const obj = value as Record<string, unknown>;
  return {
    mode: typeof obj.mode === 'string' ? obj.mode : undefined,
    selector: typeof obj.selector === 'string' ? obj.selector : undefined,
    expected: obj.expected,
    actual: obj.actual,
    success: typeof obj.success === 'boolean' ? obj.success : undefined,
    message: typeof obj.message === 'string' ? obj.message : undefined,
    negated: typeof obj.negated === 'boolean' ? obj.negated : undefined,
    caseSensitive: typeof obj.caseSensitive === 'boolean' ? obj.caseSensitive : undefined,
  };
};

const ABSOLUTE_URL_PATTERN = /^[a-zA-Z][a-zA-Z\d+.-]*:/;

const normalizeBase = (value: string): string => value.replace(/\/+$/, '');

export const resolveUrl = (url?: string | null): string | undefined => {
  if (!url) {
    return undefined;
  }

  const trimmed = url.trim();
  if (!trimmed) {
    return undefined;
  }

  if (ABSOLUTE_URL_PATTERN.test(trimmed)) {
    return trimmed;
  }

  if (trimmed.startsWith('//')) {
    const protocol = typeof window !== 'undefined' && window.location?.protocol ? window.location.protocol : 'https:';
    return `${protocol}${trimmed}`;
  }

  try {
    const apiBase = resolveApiBase({ appendSuffix: false });
    if (apiBase && apiBase.length) {
      const base = normalizeBase(apiBase);
      if (trimmed.startsWith('/')) {
        return `${base}${trimmed}`;
      }
      return `${base}/${trimmed}`;
    }
  } catch (_) {
    // fall back to window-based resolution
  }

  try {
    const fallbackBase = typeof window !== 'undefined' && window.location ? window.location.href : 'http://localhost';
    return new URL(trimmed, fallbackBase).toString();
  } catch {
    return trimmed;
  }
};
