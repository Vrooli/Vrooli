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
 * These match the proto ActionType enum values in action.proto.
 * NOTE: 'type' is an alias for 'input', 'keypress' is an alias for 'keyboard'.
 */
export type ActionType =
  | 'navigate'
  | 'click'
  | 'input'
  | 'wait'
  | 'assert'
  | 'scroll'
  | 'select'
  | 'evaluate'
  | 'keyboard'
  | 'hover'
  | 'screenshot'
  | 'focus'
  | 'blur';

export type RecordedActionKind =
  | 'RECORDED_ACTION_TYPE_UNSPECIFIED'
  | 'RECORDED_ACTION_TYPE_NAVIGATE'
  | 'RECORDED_ACTION_TYPE_CLICK'
  | 'RECORDED_ACTION_TYPE_INPUT'
  | 'RECORDED_ACTION_TYPE_WAIT'
  | 'RECORDED_ACTION_TYPE_ASSERT'
  | 'RECORDED_ACTION_TYPE_CUSTOM_SCRIPT';

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
 * A single recorded user action.
 */
export interface RecordedAction {
  id: string;
  sessionId: string;
  sequenceNum: number;
  timestamp: string;
  durationMs?: number;
  actionType: ActionType;
   /** Proto-aligned action kind; prefer over actionType for downstream typing. */
  actionKind?: RecordedActionKind;
  confidence: number;
  selector?: SelectorSet;
  elementMeta?: ElementMeta;
  boundingBox?: BoundingBox;
  payload?: ActionPayload;
  /** Typed action payloads; prefer over payload when present. */
  typedAction?: TypedActionPayload;
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

export interface RecordingSessionProfile {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
  lastUsedAt: string;
  hasStorageState: boolean;
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

/**
 * Error details for a failed action replay.
 */
export interface ActionReplayError {
  message: string;
  code: 'SELECTOR_NOT_FOUND' | 'SELECTOR_AMBIGUOUS' | 'ELEMENT_NOT_VISIBLE' | 'ELEMENT_NOT_ENABLED' | 'TIMEOUT' | 'NAVIGATION_FAILED' | 'UNKNOWN';
  match_count?: number;
  selector?: string;
}

/**
 * Result of replaying a single action.
 */
export interface ActionReplayResult {
  action_id: string;
  sequence_num: number;
  action_type: ActionType;
  success: boolean;
  duration_ms: number;
  error?: ActionReplayError;
  screenshot_on_error?: string;
}

/**
 * Response from replay preview.
 */
export interface ReplayPreviewResponse {
  success: boolean;
  total_actions: number;
  passed_actions: number;
  failed_actions: number;
  results: ActionReplayResult[];
  total_duration_ms: number;
  stopped_early: boolean;
}
