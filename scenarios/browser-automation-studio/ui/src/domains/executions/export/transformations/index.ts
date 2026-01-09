/**
 * Export transformations - pure functions for modifying replay movie specs.
 *
 * This module provides pure, testable functions for:
 * - Dimension calculations and scaling
 * - Movie spec transformations
 *
 * All functions are side-effect free and suitable for unit testing.
 */

export {
  // Types
  type Dimensions,
  type DimensionPreset,
  type DimensionPresetId,
  type ScaleFactors,
  // Constants
  DEFAULT_DIMENSIONS,
  PRESET_DIMENSIONS,
  // Functions
  calculateScaleFactors,
  extractCanvasDimensions,
  parseDimensionInput,
  resolveDimensionPreset,
  scaleDimensions,
  scaleFrameRect,
} from "./scaleDimensions";

export {
  // Types
  type ScaleMovieSpecOptions,
  // Functions
  isScalingNeeded,
  scaleFrames,
  scaleFrameViewport,
  scaleMovieSpec,
  scalePresentation,
} from "./scaleMovieSpec";
