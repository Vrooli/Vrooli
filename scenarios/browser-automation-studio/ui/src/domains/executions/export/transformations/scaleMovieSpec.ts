/**
 * Pure functions for transforming ReplayMovieSpec dimensions.
 *
 * These functions handle resizing a movie spec to different output dimensions
 * while maintaining proper aspect ratios and scaling all frame viewports.
 */

import type {
  ReplayMovieFrame,
  ReplayMoviePresentation,
  ReplayMovieSpec,
} from "@/types/export";
import {
  calculateScaleFactors,
  extractCanvasDimensions,
  scaleDimensions,
  scaleFrameRect,
  type Dimensions,
} from "./scaleDimensions";

// =============================================================================
// Types
// =============================================================================

export interface ScaleMovieSpecOptions {
  /**
   * The target output dimensions.
   */
  targetDimensions: Dimensions;
}

// =============================================================================
// Pure Functions
// =============================================================================

/**
 * Scales a movie spec's presentation to the target dimensions.
 *
 * This creates a new presentation object with:
 * - Canvas set to target dimensions
 * - Viewport scaled proportionally
 * - Browser frame scaled proportionally
 */
export function scalePresentation(
  basePresentation: ReplayMoviePresentation | undefined,
  targetDimensions: Dimensions,
): ReplayMoviePresentation {
  const baseDimensions = extractCanvasDimensions(basePresentation);
  const scaleFactors = calculateScaleFactors(baseDimensions, targetDimensions);

  const viewportFallback: Dimensions = {
    width: basePresentation?.viewport?.width ?? baseDimensions.width,
    height: basePresentation?.viewport?.height ?? baseDimensions.height,
  };

  const browserFrameFallback: Dimensions = {
    width: basePresentation?.browser_frame?.width ?? baseDimensions.width,
    height: basePresentation?.browser_frame?.height ?? baseDimensions.height,
  };

  return {
    canvas: {
      width: targetDimensions.width,
      height: targetDimensions.height,
    },
    viewport: scaleDimensions(basePresentation?.viewport, scaleFactors, viewportFallback),
    browser_frame: scaleFrameRect(
      basePresentation?.browser_frame,
      scaleFactors,
      browserFrameFallback,
      basePresentation?.browser_frame?.radius ?? 24,
    ),
    device_scale_factor:
      basePresentation?.device_scale_factor && basePresentation.device_scale_factor > 0
        ? basePresentation.device_scale_factor
        : 1,
  };
}

/**
 * Scales a single frame's viewport to match the new dimensions.
 */
export function scaleFrameViewport(
  frame: ReplayMovieFrame,
  scaleFactors: { scaleX: number; scaleY: number },
  baseDimensions: Dimensions,
  baseViewportDimensions: Dimensions,
): ReplayMovieFrame {
  const fallback: Dimensions = {
    width: frame.viewport?.width ?? baseViewportDimensions.width ?? baseDimensions.width,
    height: frame.viewport?.height ?? baseViewportDimensions.height ?? baseDimensions.height,
  };

  return {
    ...frame,
    viewport: scaleDimensions(frame.viewport, scaleFactors, fallback),
  };
}

/**
 * Scales all frames in an array to match the new dimensions.
 */
export function scaleFrames(
  frames: ReplayMovieFrame[] | undefined,
  baseDimensions: Dimensions,
  targetDimensions: Dimensions,
  baseViewportDimensions: Dimensions,
): ReplayMovieFrame[] {
  if (!Array.isArray(frames)) {
    return [];
  }

  const scaleFactors = calculateScaleFactors(baseDimensions, targetDimensions);

  return frames.map((frame) =>
    scaleFrameViewport(frame, scaleFactors, baseDimensions, baseViewportDimensions),
  );
}

/**
 * Creates a new movie spec with dimensions scaled to the target.
 *
 * This is a pure function that returns a new spec without modifying the original.
 * All presentation, viewport, and frame dimensions are scaled proportionally.
 */
export function scaleMovieSpec(
  spec: ReplayMovieSpec,
  options: ScaleMovieSpecOptions,
): ReplayMovieSpec {
  const { targetDimensions } = options;
  const baseDimensions = extractCanvasDimensions(spec.presentation);

  const baseViewportDimensions: Dimensions = {
    width: spec.presentation?.viewport?.width ?? baseDimensions.width,
    height: spec.presentation?.viewport?.height ?? baseDimensions.height,
  };

  const scaledPresentation = scalePresentation(spec.presentation, targetDimensions);
  const scaledFrames = scaleFrames(
    spec.frames,
    baseDimensions,
    targetDimensions,
    baseViewportDimensions,
  );

  return {
    ...spec,
    presentation: scaledPresentation,
    frames: scaledFrames,
  };
}

/**
 * Checks if scaling is needed (i.e., target differs from current dimensions).
 */
export function isScalingNeeded(
  currentDimensions: Dimensions,
  targetDimensions: Dimensions,
): boolean {
  return (
    currentDimensions.width !== targetDimensions.width ||
    currentDimensions.height !== targetDimensions.height
  );
}
