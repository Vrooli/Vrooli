/**
 * Record Mode Types
 *
 * Type definitions for the Record Mode feature.
 */

/**
 * Selector candidate for element targeting.
 */
export interface SelectorCandidate {
  type: 'data-testid' | 'id' | 'aria' | 'text' | 'data-attr' | 'css' | 'xpath';
  value: string;
  confidence: number;
  specificity: number;
}

/**
 * Set of selector strategies for an element.
 */
export interface SelectorSet {
  primary: string;
  candidates: SelectorCandidate[];
}

/**
 * Metadata about the target element.
 */
export interface ElementMeta {
  tagName: string;
  id?: string;
  className?: string;
  innerText?: string;
  attributes?: Record<string, string>;
  isVisible: boolean;
  isEnabled: boolean;
  role?: string;
  ariaLabel?: string;
}

/**
 * Bounding box for element position.
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
 * Action types that can be recorded.
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
 * Payload data for recorded actions.
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

  // Allow other fields
  [key: string]: unknown;
}

/**
 * A single recorded user action.
 */
export interface RecordedAction {
  id: string;
  sessionId: string;
  sequenceNum: number;
  timestamp: string;
  durationMs?: number;
  actionType: ActionType;
  confidence: number;
  selector?: SelectorSet;
  elementMeta?: ElementMeta;
  boundingBox?: BoundingBox;
  payload?: ActionPayload;
  url: string;
  frameId?: string;
  cursorPos?: Point;
}

/**
 * Recording session state.
 */
export interface RecordingState {
  isRecording: boolean;
  recordingId?: string;
  sessionId: string;
  actionCount: number;
  startedAt?: string;
}

/**
 * Response from start recording endpoint.
 */
export interface StartRecordingResponse {
  recording_id: string;
  session_id: string;
  started_at: string;
}

/**
 * Response from stop recording endpoint.
 */
export interface StopRecordingResponse {
  recording_id: string;
  session_id: string;
  action_count: number;
  stopped_at: string;
}

/**
 * Response from get actions endpoint.
 */
export interface GetActionsResponse {
  session_id: string;
  actions: RecordedAction[];
  count: number;
}

/**
 * Response from generate workflow endpoint.
 */
export interface GenerateWorkflowResponse {
  workflow_id: string;
  project_id: string;
  name: string;
  node_count: number;
  action_count: number;
}

/**
 * Selector validation result.
 */
export interface SelectorValidation {
  valid: boolean;
  match_count: number;
  selector: string;
  error?: string;
}
