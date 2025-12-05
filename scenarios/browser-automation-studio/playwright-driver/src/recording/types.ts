/**
 * Recording Types
 *
 * Type definitions for Record Mode feature.
 * These types define the data structures for capturing user actions
 * and generating reliable selectors.
 */

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
 */
export type SelectorType =
  | 'data-testid'  // Explicit test ID (highest confidence)
  | 'id'           // Unique DOM ID
  | 'aria'         // ARIA labels/attributes
  | 'text'         // Tag + text content (Playwright :has-text())
  | 'data-attr'    // Other data-* attributes
  | 'css'          // CSS selector path
  | 'xpath';       // XPath (fallback)

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

/**
 * BoundingBox for element position on screen.
 */
export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

/**
 * Point for cursor/click positions.
 */
export interface Point {
  x: number;
  y: number;
}

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
 */
export const DEFAULT_SELECTOR_OPTIONS: Required<SelectorGeneratorOptions> = {
  maxCssDepth: 5,
  includeXPath: true,
  preferTestIds: true,
  unstableClassPatterns: [
    /^css-[a-z0-9]+$/i,      // CSS-in-JS (Emotion, etc.)
    /^sc-[a-zA-Z]+$/,        // styled-components
    /^_[a-zA-Z0-9]+$/,       // CSS modules
    /^[a-zA-Z]+-[0-9]+$/,    // Generic hash patterns
    /^jsx-[a-z0-9]+$/i,      // Next.js styled-jsx
    /^svelte-[a-z0-9]+$/i,   // Svelte scoped styles
    /^v-[a-z0-9]+$/i,        // Vue scoped styles
  ],
  minConfidence: 0.3,
};
