/**
 * Geometry utilities for the ReplayPlayer component
 *
 * Contains functions for coordinate transformations and style generation:
 * - Normalized/absolute point conversion
 * - CSS style generation for rects and points
 * - Zoom anchor selection
 */

import type {
  ReplayPoint,
  NormalizedPoint,
  Dimensions,
  ReplayBoundingBox,
  ReplayFrame,
} from '../types';
import { FALLBACK_DIMENSIONS } from '../constants';
import { clamp01, clampNormalizedPoint } from './cursorMath';

// =============================================================================
// Point Conversion
// =============================================================================

export const toNormalizedPoint = (
  point: ReplayPoint | null | undefined,
  dims: Dimensions,
): NormalizedPoint | undefined => {
  if (!point || typeof point.x !== 'number' || typeof point.y !== 'number') {
    return undefined;
  }
  const width = dims.width || FALLBACK_DIMENSIONS.width;
  const height = dims.height || FALLBACK_DIMENSIONS.height;
  if (!width || !height) {
    return undefined;
  }
  return clampNormalizedPoint({ x: point.x / width, y: point.y / height });
};

export const toAbsolutePoint = (point: NormalizedPoint, dims: Dimensions): ReplayPoint => {
  const width = dims.width || FALLBACK_DIMENSIONS.width;
  const height = dims.height || FALLBACK_DIMENSIONS.height;
  return {
    x: clamp01(point.x) * width,
    y: clamp01(point.y) * height,
  };
};

// =============================================================================
// CSS Style Generation
// =============================================================================

export const toRectStyle = (box: ReplayBoundingBox | null | undefined, dims: Dimensions) => {
  if (!box || !dims.width || !dims.height) {
    return undefined;
  }
  const clamp = (value: number | undefined, max: number) => {
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return 0;
    }
    return Math.max(0, Math.min(value, max));
  };

  const width = clamp(box.width, dims.width);
  const height = clamp(box.height, dims.height);
  const left = clamp(box.x, dims.width - width);
  const top = clamp(box.y, dims.height - height);

  return {
    left: `${(left / dims.width) * 100}%`,
    top: `${(top / dims.height) * 100}%`,
    width: `${(width / dims.width) * 100}%`,
    height: `${(height / dims.height) * 100}%`,
  } as const;
};

export const toPointStyle = (point: ReplayPoint | null | undefined, dims: Dimensions) => {
  if (!point || !dims.width || !dims.height) {
    return undefined;
  }
  if (typeof point.x !== 'number' || typeof point.y !== 'number') {
    return undefined;
  }
  return {
    left: `${(point.x / dims.width) * 100}%`,
    top: `${(point.y / dims.height) * 100}%`,
  } as const;
};

// =============================================================================
// Zoom Anchor Selection
// =============================================================================

export const pickZoomAnchor = (frame: ReplayFrame): ReplayBoundingBox | undefined => {
  if (frame.focusedElement?.boundingBox) {
    return frame.focusedElement.boundingBox;
  }
  if (frame.highlightRegions && frame.highlightRegions.length > 0) {
    const candidate = frame.highlightRegions.find((region) => region.boundingBox);
    if (candidate?.boundingBox) {
      return candidate.boundingBox;
    }
  }
  return undefined;
};
