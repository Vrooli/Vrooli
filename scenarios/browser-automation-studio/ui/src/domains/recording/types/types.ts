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
  /** Page ID for multi-tab recording */
  pageId?: string;
  /** Page title at the time of the action */
  pageTitle?: string;
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
  created_at: string;
  updated_at: string;
  last_used_at: string;
  has_storage_state: boolean;
  browser_profile?: BrowserProfile;
}

// ============================================================================
// Browser Profile Types - Anti-detection and Human-like Behavior Settings
// ============================================================================

/**
 * Quality preset for browser profile.
 * - stealth: Maximum anti-detection, natural human behavior
 * - balanced: Good anti-detection, reasonable speed
 * - fast: Minimal delays, basic anti-detection
 * - none: No anti-detection or behavior modifications
 */
export type ProfilePreset = 'stealth' | 'balanced' | 'fast' | 'none';

/**
 * User agent preset for common browser/OS combinations.
 */
export type UserAgentPreset =
  | 'chrome_windows'
  | 'chrome_mac'
  | 'chrome_linux'
  | 'firefox_windows'
  | 'firefox_mac'
  | 'safari_mac'
  | 'edge_windows'
  | 'custom';

/**
 * Mouse movement style for human-like behavior.
 */
export type MouseMovementStyle = 'linear' | 'bezier' | 'natural';

/**
 * Scroll behavior style.
 * Must match API validation: smooth (continuous) or stepped (discrete, human-like).
 */
export type ScrollStyle = 'smooth' | 'stepped';

/**
 * Fingerprint settings for browser identity.
 */
export interface FingerprintSettings {
  viewport_width?: number;
  viewport_height?: number;
  device_scale_factor?: number;
  hardware_concurrency?: number;
  device_memory?: number;
  user_agent?: string;
  user_agent_preset?: UserAgentPreset;
  locale?: string;
  timezone_id?: string;
  geolocation_enabled?: boolean;
  latitude?: number;
  longitude?: number;
  accuracy?: number;
  color_scheme?: 'light' | 'dark' | 'no-preference';
}

/**
 * Human-like behavior settings.
 */
export interface BehaviorSettings {
  // Typing behavior - inter-keystroke delays
  typing_delay_min?: number;
  typing_delay_max?: number;

  // Typing behavior - pre-typing delay (pause before starting to type)
  typing_start_delay_min?: number;
  typing_start_delay_max?: number;

  // Typing behavior - paste threshold (paste long text instead of typing)
  typing_paste_threshold?: number; // If text > this, paste. 0 = always type, -1 = always paste

  // Typing behavior - enhanced variance (simulate human typing patterns)
  typing_variance_enabled?: boolean; // Enable digraph/shift/symbol variance

  // Mouse movement
  mouse_movement_style?: MouseMovementStyle;
  mouse_jitter_amount?: number;

  // Click behavior
  click_delay_min?: number;
  click_delay_max?: number;

  // Scroll behavior
  scroll_style?: ScrollStyle;
  scroll_speed_min?: number;
  scroll_speed_max?: number;

  // Random micro-pauses between actions
  micro_pause_enabled?: boolean;
  micro_pause_min_ms?: number;
  micro_pause_max_ms?: number;
  micro_pause_frequency?: number;
}

/**
 * Ad blocking mode options.
 */
export type AdBlockingMode = 'none' | 'ads_only' | 'ads_and_tracking';

/**
 * Anti-detection settings for bypassing bot detection.
 */
export interface AntiDetectionSettings {
  disable_automation_controlled?: boolean;
  disable_webrtc?: boolean;
  patch_navigator_webdriver?: boolean;
  patch_navigator_plugins?: boolean;
  patch_navigator_languages?: boolean;
  patch_webgl?: boolean;
  patch_canvas?: boolean;
  headless_detection_bypass?: boolean;
  ad_blocking_mode?: AdBlockingMode;
}

/**
 * Full browser profile configuration.
 */
export interface BrowserProfile {
  preset?: ProfilePreset;
  fingerprint?: FingerprintSettings;
  behavior?: BehaviorSettings;
  anti_detection?: AntiDetectionSettings;
}

/**
 * Cookie from browser storage state.
 */
export interface StorageStateCookie {
  name: string;
  value: string;
  valueMasked: boolean;
  domain: string;
  path: string;
  expires: number;
  httpOnly: boolean;
  secure: boolean;
  sameSite: 'Strict' | 'Lax' | 'None';
}

/**
 * LocalStorage key-value pair.
 */
export interface StorageStateLocalStorageItem {
  name: string;
  value: string;
}

/**
 * LocalStorage grouped by origin.
 */
export interface StorageStateOrigin {
  origin: string;
  localStorage: StorageStateLocalStorageItem[];
}

/**
 * Summary statistics for storage state.
 */
export interface StorageStateStats {
  cookieCount: number;
  localStorageCount: number;
  originCount: number;
}

/**
 * Full storage state response from API.
 */
export interface StorageStateResponse {
  cookies: StorageStateCookie[];
  origins: StorageStateOrigin[];
  stats: StorageStateStats;
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
