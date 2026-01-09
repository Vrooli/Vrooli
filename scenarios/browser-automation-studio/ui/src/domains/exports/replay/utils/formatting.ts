/**
 * Formatting utilities for the ReplayPlayer component
 *
 * Contains functions for formatting durations, coordinates, and color values.
 */

import type { ReplayPoint, Dimensions } from '../types';
import { DEFAULT_DURATION, MIN_DURATION, MAX_DURATION, FALLBACK_CLICK_COLOR } from '../constants';

// =============================================================================
// Duration Formatting
// =============================================================================

export const clampDuration = (value?: number): number => {
  if (!value || Number.isNaN(value)) {
    return DEFAULT_DURATION;
  }
  return Math.min(MAX_DURATION, Math.max(MIN_DURATION, value));
};

export const formatDurationSeconds = (value?: number | null): string => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '0.0s';
  }
  const seconds = Math.max(0, value) / 1000;
  return `${seconds.toFixed(1)}s`;
};

// =============================================================================
// Coordinate Formatting
// =============================================================================

export const formatCoordinate = (point?: ReplayPoint | null, dims?: Dimensions): string => {
  if (!point || typeof point.x !== 'number' || typeof point.y !== 'number' || !dims?.width || !dims?.height) {
    return 'n/a';
  }
  const percentX = (point.x / dims.width) * 100;
  const percentY = (point.y / dims.height) * 100;
  const formatPercent = (value: number) => `${Math.round(value * 10) / 10}%`;
  return `${Math.round(point.x)}px, ${Math.round(point.y)}px (${formatPercent(percentX)}, ${formatPercent(percentY)})`;
};

// =============================================================================
// Color Formatting
// =============================================================================

export const parseRgbaComponents = (value: string | undefined): [string, string, string] | null => {
  if (!value) {
    return null;
  }
  const match = value.match(/rgba?\(([^)]+)\)/i);
  if (!match) {
    return null;
  }
  const parts = match[1]
    .split(',')
    .map((part) => part.trim())
    .filter((part) => part.length > 0);
  if (parts.length < 3) {
    return null;
  }
  return [parts[0], parts[1], parts[2]];
};

export const rgbaWithAlpha = (components: [string, string, string] | null, alpha: number): string => {
  const [r, g, b] = components ?? FALLBACK_CLICK_COLOR;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
};

// =============================================================================
// Value Formatting
// =============================================================================

export const formatValue = (value: unknown): string => {
  if (value == null) {
    return 'null';
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value);
    } catch (error) {
      return String(value);
    }
  }
  return String(value);
};
