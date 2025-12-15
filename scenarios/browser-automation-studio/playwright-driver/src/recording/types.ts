/**
 * Recording Types
 *
 * PROTO-FIRST ARCHITECTURE:
 * All recording types now come from proto. This file re-exports proto types
 * and defines recording-specific types that don't belong in the proto schema.
 *
 * TYPE SOURCES:
 * - Proto types (TimelineEntry, ActionType, etc.): Import from '../proto'
 * - Selector configuration: Import from './selector-config'
 * - Recording state types: Defined here (driver-specific, not in proto)
 */

import { SELECTOR_DEFAULTS, getUnstableClassPatterns } from './selector-config';

// =============================================================================
// PROTO TYPE RE-EXPORTS
// =============================================================================

// Re-export proto types for use by other modules
export type {
  TimelineEntry,
  RawBrowserEvent,
  RawSelectorSet,
  RawSelectorCandidate,
  RawElementMeta,
  ConversionContext,
} from '../proto/recording';

export {
  rawBrowserEventToTimelineEntry,
  timelineEntryToJson,
  createNavigateTimelineEntry,
  TimelineEntrySchema,
  ActionType,
} from '../proto/recording';

// Re-export domain types from proto
export type {
  BoundingBox,
  Point,
  SelectorCandidate,
  ElementMeta,
} from '../proto';
export { SelectorType } from '../proto';

// Re-export action executor types
export type {
  SelectorValidation,
  ActionReplayResult,
  ExecutorContext,
  ActionErrorCode,
} from './action-executor';

// =============================================================================
// RECORDING-SPECIFIC TYPES (driver-only, not in proto)
// =============================================================================

/**
 * Recording session state.
 * Tracks the current state of a recording session in the driver.
 */
export interface RecordingState {
  isRecording: boolean;
  recordingId?: string;
  sessionId: string;
  actionCount: number;
  startedAt?: string;
}

/**
 * Selector generation options.
 * Configuration for how selectors are generated from DOM elements.
 */
export interface SelectorGeneratorOptions {
  maxCssDepth?: number;
  includeXPath?: boolean;
  preferTestIds?: boolean;
  unstableClassPatterns?: RegExp[];
  minConfidence?: number;
}

/**
 * Default options for selector generation.
 */
export const DEFAULT_SELECTOR_OPTIONS: Required<SelectorGeneratorOptions> = {
  maxCssDepth: SELECTOR_DEFAULTS.maxCssDepth,
  includeXPath: SELECTOR_DEFAULTS.includeXPath,
  preferTestIds: SELECTOR_DEFAULTS.preferTestIds,
  unstableClassPatterns: getUnstableClassPatterns(),
  minConfidence: SELECTOR_DEFAULTS.minConfidence,
};
