import type { ExecutionViewportSettings, ViewportPreset } from '../types';

// ============================================================================
// Constants
// ============================================================================

export const MIN_VIEWPORT_DIMENSION = 200;
export const MAX_VIEWPORT_DIMENSION = 10000;

// ============================================================================
// Helpers
// ============================================================================

export const isPlainObject = (value: unknown): value is Record<string, unknown> => {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
};

export const clampDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION), MAX_VIEWPORT_DIMENSION);
};

export const parseViewportDimension = (value: unknown): number | null => {
  if (typeof value === 'number') {
    return value > 0 ? clampDimension(value) : null;
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return null;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (Number.isNaN(parsed)) {
      return null;
    }
    return parsed > 0 ? clampDimension(parsed) : null;
  }
  return null;
};

export const parseViewportPreset = (value: unknown): ViewportPreset | undefined => {
  if (typeof value !== 'string') {
    return undefined;
  }
  const normalized = value.trim().toLowerCase();
  if (normalized === 'desktop' || normalized === 'mobile' || normalized === 'custom') {
    return normalized;
  }
  return undefined;
};

// ============================================================================
// Viewport Settings
// ============================================================================

export const sanitizeViewportSettings = (viewport: ExecutionViewportSettings | undefined | null): ExecutionViewportSettings | undefined => {
  if (!viewport) {
    return undefined;
  }
  const width = parseViewportDimension((viewport as ExecutionViewportSettings).width);
  const height = parseViewportDimension((viewport as ExecutionViewportSettings).height);
  if (!width || !height) {
    return undefined;
  }
  const preset = parseViewportPreset((viewport as ExecutionViewportSettings).preset);
  return preset ? { width, height, preset } : { width, height };
};

export const extractExecutionViewport = (definition: unknown): ExecutionViewportSettings | undefined => {
  if (!isPlainObject(definition)) {
    return undefined;
  }
  const settingsValue = definition.settings;
  if (!isPlainObject(settingsValue)) {
    return undefined;
  }
  const viewportValue = settingsValue.executionViewport ?? settingsValue.viewport;
  if (!isPlainObject(viewportValue)) {
    return undefined;
  }
  const width = parseViewportDimension(viewportValue.width);
  const height = parseViewportDimension(viewportValue.height);
  if (!width || !height) {
    return undefined;
  }
  const preset = parseViewportPreset(viewportValue.preset ?? viewportValue.mode);
  return preset ? { width, height, preset } : { width, height };
};
