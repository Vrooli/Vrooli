/**
 * Recording Types
 *
 * Type definitions for Record Mode feature.
 * These types define the data structures for capturing user actions
 * and generating reliable selectors.
 */

import type { SelectorStrategyType } from './selector-config';
import { SELECTOR_DEFAULTS, getUnstableClassPatterns } from './selector-config';
// Re-export shared geometry types from contracts (single source of truth)
import type { BoundingBox, Point } from '../types/contracts';
export type { BoundingBox, Point };

/**
 * SelectorCandidate represents a single selector strategy with metadata.
 */
export interface SelectorCandidate {
  /** Selector strategy type */
  type: SelectorType;
  /** The actual selector string (CSS, XPath, etc.) */
  value: string;
  /** Confidence score 0-1, likelihood of uniqueness and stability */
  confidence: number;
  /** Specificity score, higher = more specific (used for tie-breaking) */
  specificity: number;
}

/**
 * Supported selector types, in order of preference.
 * Derived from SELECTOR_STRATEGIES in selector-config.ts.
 */
export type SelectorType = SelectorStrategyType;

/**
 * SelectorSet contains multiple selector strategies for resilience.
 * When one selector fails, the system can fall back to alternatives.
 */
export interface SelectorSet {
  /** Best selector to use (highest confidence) */
  primary: string;
  /** All ranked alternatives */
  candidates: SelectorCandidate[];
}

/**
 * ElementMeta captures information about the target element.
 * Used for display purposes and debugging.
 */
export interface ElementMeta {
  /** HTML tag name (lowercase) */
  tagName: string;
  /** Element ID if present */
  id?: string;
  /** Class names (space-separated string) */
  className?: string;
  /** Visible text content (truncated to reasonable length) */
  innerText?: string;
  /** Relevant attributes for identification */
  attributes?: Record<string, string>;
  /** Whether element is visible in viewport */
  isVisible: boolean;
  /** Whether element is enabled (for inputs/buttons) */
  isEnabled: boolean;
  /** Role attribute or inferred role */
  role?: string;
  /** ARIA label if present */
  ariaLabel?: string;
}

// BoundingBox and Point are imported and re-exported from types/contracts.ts
// This ensures a single source of truth for geometry types across the codebase.

/**
 * RecordedAction represents a single user action captured during recording.
 * This is the primary data structure flowing through the recording pipeline.
 */
export interface RecordedAction {
  /** Unique action ID (UUID) */
  id: string;
  /** Recording session ID */
  sessionId: string;
  /** Order in recording sequence (0-based) */
  sequenceNum: number;

  /** When action occurred (ISO 8601) */
  timestamp: string;
  /** Duration in ms for actions with duration (typing) */
  durationMs?: number;

  /** Action classification */
  actionType: ActionType;
  /** Typed action kind (proto-aligned) */
  actionKind?: RecordedActionKind;
  /** Confidence in action classification (0-1) */
  confidence: number;

  /** Target element selector set */
  selector?: SelectorSet;
  /** Target element metadata */
  elementMeta?: ElementMeta;
  /** Element bounding box at action time */
  boundingBox?: BoundingBox;

  /** Action-specific parameters */
  payload?: ActionPayload;
  /** Typed action payloads (proto-aligned) */
  typedAction?: TypedActionPayload;

  /** Page URL at action time */
  url: string;
  /** Frame ID if action occurred in iframe */
  frameId?: string;

  /** Cursor position at action time */
  cursorPos?: Point;
}

/**
 * Supported action types for recording.
 */
export type ActionType =
  | 'click'
  | 'type'
  | 'scroll'
  | 'navigate'
  | 'select'
  | 'hover'
  | 'focus'
  | 'blur'
  | 'keypress';

/** Proto-aligned recorded action kinds. */
export type RecordedActionKind =
  | 'RECORDED_ACTION_TYPE_UNSPECIFIED'
  | 'RECORDED_ACTION_TYPE_NAVIGATE'
  | 'RECORDED_ACTION_TYPE_CLICK'
  | 'RECORDED_ACTION_TYPE_INPUT'
  | 'RECORDED_ACTION_TYPE_WAIT'
  | 'RECORDED_ACTION_TYPE_ASSERT'
  | 'RECORDED_ACTION_TYPE_CUSTOM_SCRIPT';

/**
 * Action-specific payload data.
 */
export interface ActionPayload {
  // Click-specific
  button?: 'left' | 'right' | 'middle';
  modifiers?: Array<'ctrl' | 'shift' | 'alt' | 'meta'>;
  clickCount?: number;

  // Type-specific
  text?: string;
  delay?: number;
  clearFirst?: boolean;

  // Scroll-specific
  scrollX?: number;
  scrollY?: number;
  deltaX?: number;
  deltaY?: number;

  // Select-specific
  value?: string;
  selectedText?: string;
  selectedIndex?: number;

  // Keypress-specific
  key?: string;
  code?: string;

  // Navigate-specific
  targetUrl?: string;

  // Generic
  [key: string]: unknown;
}

/** Typed payloads for RecordedAction. */
export type TypedActionPayload =
  | { navigate: NavigateActionPayload }
  | { click: ClickActionPayload }
  | { input: InputActionPayload }
  | { wait: WaitActionPayload }
  | { assert: AssertActionPayload }
  | { customScript: CustomScriptActionPayload };

export interface NavigateActionPayload {
  url: string;
  waitForSelector?: string;
  timeoutMs?: number;
}

export interface ClickActionPayload {
  selector?: string;
  button?: 'left' | 'right' | 'middle';
  clickCount?: number;
  delayMs?: number;
  scrollIntoView?: boolean;
}

export interface InputActionPayload {
  selector?: string;
  value: string;
  isSensitive?: boolean;
  submit?: boolean;
}

export interface WaitActionPayload {
  durationMs: number;
}

export interface AssertActionPayload {
  mode: string;
  selector: string;
  expected?: unknown;
  negated?: boolean;
  caseSensitive?: boolean;
}

export interface CustomScriptActionPayload {
  language?: string;
  source: string;
}

/**
 * Raw browser event captured by injected JavaScript.
 * This is the format received from page.exposeFunction() before normalization.
 */
export interface RawBrowserEvent {
  actionType: string;
  timestamp: number;
  selector: SelectorSet;
  elementMeta: ElementMeta;
  boundingBox?: BoundingBox;
  cursorPos?: Point;
  url: string;
  frameId?: string | null;
  payload?: Record<string, unknown>;
}

/**
 * Recording session state.
 */
export interface RecordingState {
  /** Whether recording is currently active */
  isRecording: boolean;
  /** Recording session ID */
  recordingId?: string;
  /** Session ID this recording is attached to */
  sessionId: string;
  /** Number of actions captured */
  actionCount: number;
  /** When recording started */
  startedAt?: string;
}

/**
 * Selector validation result.
 */
export interface SelectorValidation {
  /** Whether selector found exactly one element */
  valid: boolean;
  /** Number of elements found */
  matchCount: number;
  /** The selector that was validated */
  selector: string;
  /** Error message if validation failed */
  error?: string;
}

/**
 * Selector generation options.
 */
export interface SelectorGeneratorOptions {
  /** Maximum depth for CSS path generation */
  maxCssDepth?: number;
  /** Include XPath as fallback */
  includeXPath?: boolean;
  /** Prefer data-testid selectors */
  preferTestIds?: boolean;
  /** Patterns for unstable classes to filter out */
  unstableClassPatterns?: RegExp[];
  /** Minimum confidence threshold to include candidate */
  minConfidence?: number;
}

/**
 * Default options for selector generation.
 * Derived from SELECTOR_DEFAULTS in selector-config.ts (single source of truth).
 */
export const DEFAULT_SELECTOR_OPTIONS: Required<SelectorGeneratorOptions> = {
  maxCssDepth: SELECTOR_DEFAULTS.maxCssDepth,
  includeXPath: SELECTOR_DEFAULTS.includeXPath,
  preferTestIds: SELECTOR_DEFAULTS.preferTestIds,
  // Use the canonical source from selector-config.ts (converted to RegExp)
  unstableClassPatterns: getUnstableClassPatterns(),
  minConfidence: SELECTOR_DEFAULTS.minConfidence,
};

/**
 * Result of replaying a single action.
 */
export interface ActionReplayResult {
  /** The action that was replayed */
  actionId: string;
  /** Index in the action sequence */
  sequenceNum: number;
  /** Action type that was executed */
  actionType: ActionType;
  /** Whether the action succeeded */
  success: boolean;
  /** Time taken to execute in ms */
  durationMs: number;
  /** Error details if failed */
  error?: {
    /** Error message */
    message: string;
    /** Error code for categorization */
    code: 'SELECTOR_NOT_FOUND' | 'SELECTOR_AMBIGUOUS' | 'ELEMENT_NOT_VISIBLE' | 'ELEMENT_NOT_ENABLED' | 'TIMEOUT' | 'NAVIGATION_FAILED' | 'UNKNOWN';
    /** Number of elements found for selector (0 = not found, >1 = ambiguous) */
    matchCount?: number;
    /** The selector that failed */
    selector?: string;
  };
  /** Screenshot after action (base64, only on failure) */
  screenshotOnError?: string;
}

/**
 * Request for replay preview.
 */
export interface ReplayPreviewRequest {
  /** Actions to replay */
  actions: RecordedAction[];
  /** Maximum number of actions to replay (default: all) */
  limit?: number;
  /** Whether to stop on first failure (default: true) */
  stopOnFailure?: boolean;
  /** Action timeout in ms (default: config.execution.replayActionTimeoutMs) */
  actionTimeout?: number;
}

/**
 * Response from replay preview.
 */
export interface ReplayPreviewResponse {
  /** Overall success (all actions passed) */
  success: boolean;
  /** Total actions attempted */
  totalActions: number;
  /** Number of actions that passed */
  passedActions: number;
  /** Number of actions that failed */
  failedActions: number;
  /** Per-action results */
  results: ActionReplayResult[];
  /** Total duration in ms */
  totalDurationMs: number;
  /** Whether replay was stopped early due to failure */
  stoppedEarly: boolean;
}
