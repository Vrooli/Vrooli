/**
 * Pure functions for dimension scaling calculations.
 *
 * These functions handle the mathematical transformations for resizing
 * replay movie specs to different output dimensions. They are pure functions
 * with no side effects, making them easy to test and reason about.
 */

import type { ReplayMovieFrameRect, ReplayMoviePresentation } from "@/types/export";

// =============================================================================
// Types
// =============================================================================

export interface Dimensions {
  width: number;
  height: number;
}

export interface ScaleFactors {
  scaleX: number;
  scaleY: number;
}

export interface DimensionPreset {
  width: number;
  height: number;
}

export type DimensionPresetId = "spec" | "1080p" | "720p" | "custom";

// =============================================================================
// Constants
// =============================================================================

export const PRESET_DIMENSIONS: Record<"1080p" | "720p", DimensionPreset> = {
  "1080p": { width: 1920, height: 1080 },
  "720p": { width: 1280, height: 720 },
};

export const DEFAULT_DIMENSIONS: Dimensions = {
  width: 1280,
  height: 720,
};

// =============================================================================
// Pure Functions
// =============================================================================

/**
 * Calculates scale factors for transforming from source to target dimensions.
 */
export function calculateScaleFactors(
  source: Dimensions,
  target: Dimensions,
): ScaleFactors {
  const sourceWidth = source.width > 0 ? source.width : DEFAULT_DIMENSIONS.width;
  const sourceHeight = source.height > 0 ? source.height : DEFAULT_DIMENSIONS.height;

  return {
    scaleX: sourceWidth > 0 ? target.width / sourceWidth : 1,
    scaleY: sourceHeight > 0 ? target.height / sourceHeight : 1,
  };
}

/**
 * Scales a dimension object by the given scale factors.
 */
export function scaleDimensions(
  dims: Partial<Dimensions> | undefined,
  scaleFactors: ScaleFactors,
  fallback: Dimensions,
): Dimensions {
  const sourceWidth = dims?.width && dims.width > 0 ? dims.width : fallback.width;
  const sourceHeight = dims?.height && dims.height > 0 ? dims.height : fallback.height;

  return {
    width: Math.round(sourceWidth * scaleFactors.scaleX),
    height: Math.round(sourceHeight * scaleFactors.scaleY),
  };
}

/**
 * Scales a frame rect (position + dimensions + radius) by the given scale factors.
 */
export function scaleFrameRect(
  rect: Partial<ReplayMovieFrameRect> | null | undefined,
  scaleFactors: ScaleFactors,
  fallbackDimensions: Dimensions,
  defaultRadius = 24,
): ReplayMovieFrameRect {
  const source: ReplayMovieFrameRect = rect
    ? { ...rect } as ReplayMovieFrameRect
    : {
        x: 0,
        y: 0,
        width: fallbackDimensions.width,
        height: fallbackDimensions.height,
        radius: defaultRadius,
      };

  return {
    x: Math.round((source.x ?? 0) * scaleFactors.scaleX),
    y: Math.round((source.y ?? 0) * scaleFactors.scaleY),
    width: Math.round((source.width ?? fallbackDimensions.width) * scaleFactors.scaleX),
    height: Math.round((source.height ?? fallbackDimensions.height) * scaleFactors.scaleY),
    radius: source.radius ?? defaultRadius,
  };
}

/**
 * Resolves the target dimensions based on a preset selection.
 */
export function resolveDimensionPreset(
  preset: DimensionPresetId,
  specDimensions: Dimensions,
  customDimensions: Dimensions,
): Dimensions {
  switch (preset) {
    case "custom":
      return {
        width: customDimensions.width > 0 ? customDimensions.width : DEFAULT_DIMENSIONS.width,
        height: customDimensions.height > 0 ? customDimensions.height : DEFAULT_DIMENSIONS.height,
      };
    case "1080p":
      return PRESET_DIMENSIONS["1080p"];
    case "720p":
      return PRESET_DIMENSIONS["720p"];
    case "spec":
    default:
      return {
        width: specDimensions.width > 0 ? specDimensions.width : DEFAULT_DIMENSIONS.width,
        height: specDimensions.height > 0 ? specDimensions.height : DEFAULT_DIMENSIONS.height,
      };
  }
}

/**
 * Extracts canvas dimensions from a presentation object.
 */
export function extractCanvasDimensions(
  presentation: ReplayMoviePresentation | null | undefined,
): Dimensions {
  return {
    width:
      presentation?.canvas?.width ??
      presentation?.viewport?.width ??
      DEFAULT_DIMENSIONS.width,
    height:
      presentation?.canvas?.height ??
      presentation?.viewport?.height ??
      DEFAULT_DIMENSIONS.height,
  };
}

/**
 * Parses a dimension input string to a number, with fallback.
 */
export function parseDimensionInput(input: string, fallback: number): number {
  const parsed = Number.parseInt(input, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}
