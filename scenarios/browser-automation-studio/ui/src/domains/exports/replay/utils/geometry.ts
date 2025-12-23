/**
 * Geometry utilities for the ReplayPlayer component
 *
 * Contains functions for coordinate transformations and zoom selection:
 * - Normalized/absolute point conversion
 * - Zoom anchor selection
 */

import type { ReplayPoint, NormalizedPoint, Dimensions, ReplayBoundingBox, ReplayFrame } from '../types';
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

export const toAbsolutePoint = (point: NormalizedPoint, dims: Dimensions): { x: number; y: number } => {
  const width = dims.width || FALLBACK_DIMENSIONS.width;
  const height = dims.height || FALLBACK_DIMENSIONS.height;
  return {
    x: clamp01(point.x) * width,
    y: clamp01(point.y) * height,
  };
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
