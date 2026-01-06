/**
 * Record Mode API Types
 *
 * Request and response types for all Record Mode endpoints.
 * These define the HTTP contract for the recording feature.
 */

import type { TimelineEntry } from '../../proto/recording';

/**
 * POST /session/:id/record/start
 */
export interface StartRecordingRequest {
  /**
   * Optional recording ID to use for this recording.
   * If provided and recording is already active with this ID, the request is
   * treated as idempotent (returns success without starting a new recording).
   * This enables safe retries when network issues cause response loss.
   */
  recording_id?: string;
  /** Optional callback URL to stream actions to (for API integration) */
  callback_url?: string;
  /** Optional callback URL to stream frames to (for WebSocket broadcasting) */
  frame_callback_url?: string;
  /** Optional callback URL to stream page lifecycle events to (for multi-tab support) */
  page_callback_url?: string;
  /** Frame quality (1-100), default 55 (from API config) */
  frame_quality?: number;
  /** Target FPS for frame streaming (1-60), default 30 (from API config) */
  /** Note: For CDP screencast, Chrome controls actual FPS. This is a target/hint. */
  frame_fps?: number;
}

export interface StartRecordingResponse {
  recording_id: string;
  session_id: string;
  started_at: string;
  /**
   * Pipeline verification performed before recording started.
   * Always present when auto_verify is true (default).
   * Use this to diagnose issues if recording appears to work but captures no events.
   */
  verification?: {
    /** Recording script loaded on page */
    script_loaded: boolean;
    /** Recording script fully initialized */
    script_ready: boolean;
    /** Script running in MAIN context (required for History API) */
    in_main_context: boolean;
    /** Number of event handlers registered (expect 7+) */
    handlers_count: number;
    /** Script version */
    version?: string | null;
    /** Any warnings about the verification */
    warnings?: string[];
  };
}

/**
 * POST /session/:id/record/stop
 */
export interface StopRecordingResponse {
  recording_id: string;
  session_id: string;
  action_count: number;
  stopped_at: string;
}

/**
 * GET /session/:id/record/status
 */
export interface RecordingStatusResponse {
  session_id: string;
  is_recording: boolean;
  recording_id?: string;
  action_count: number;
  started_at?: string;
}

/**
 * POST /session/:id/record/validate-selector
 */
export interface ValidateSelectorRequest {
  selector: string;
}

export interface ValidateSelectorResponse {
  valid: boolean;
  match_count: number;
  selector: string;
  error?: string;
}

/**
 * POST /session/:id/record/replay-preview
 */
export interface ReplayPreviewRequest {
  entries: TimelineEntry[];
  limit?: number;
  stop_on_failure?: boolean;
  action_timeout?: number;
}

export interface ReplayPreviewResponse {
  success: boolean;
  total_actions: number;
  passed_actions: number;
  failed_actions: number;
  results: Array<{
    entry_id: string;
    sequence_num: number;
    action_type: number; // ActionType enum
    success: boolean;
    duration_ms: number;
    error?: {
      message: string;
      code: string;
      match_count?: number;
      selector?: string;
    };
    screenshot_on_error?: string;
  }>;
  total_duration_ms: number;
  stopped_early: boolean;
}

/**
 * POST /session/:id/record/navigate
 */
export interface NavigateRequest {
  url: string;
  wait_until?: 'load' | 'domcontentloaded' | 'networkidle' | 'commit';
  timeout_ms?: number;
  capture?: boolean;
}

export interface NavigateResponse {
  session_id: string;
  url: string;
  title: string;
  can_go_back: boolean;
  can_go_forward: boolean;
  screenshot?: string;
}

/**
 * POST /session/:id/record/reload
 */
export interface ReloadRequest {
  wait_until?: 'load' | 'domcontentloaded' | 'networkidle' | 'commit';
  timeout_ms?: number;
}

export interface ReloadResponse {
  session_id: string;
  url: string;
  title: string;
  can_go_back: boolean;
  can_go_forward: boolean;
}

/**
 * POST /session/:id/record/go-back
 */
export interface GoBackRequest {
  wait_until?: 'load' | 'domcontentloaded' | 'networkidle' | 'commit';
  timeout_ms?: number;
}

export interface GoBackResponse {
  session_id: string;
  url: string;
  title: string;
  can_go_back: boolean;
  can_go_forward: boolean;
}

/**
 * POST /session/:id/record/go-forward
 */
export interface GoForwardRequest {
  wait_until?: 'load' | 'domcontentloaded' | 'networkidle' | 'commit';
  timeout_ms?: number;
}

export interface GoForwardResponse {
  session_id: string;
  url: string;
  title: string;
  can_go_back: boolean;
  can_go_forward: boolean;
}

/**
 * GET /session/:id/record/navigation-state
 */
export interface NavigationStateResponse {
  session_id: string;
  url: string;
  title: string;
  can_go_back: boolean;
  can_go_forward: boolean;
}

/**
 * POST /session/:id/record/screenshot
 */
export interface ScreenshotRequest {
  full_page?: boolean;
  quality?: number;
}

export interface ScreenshotResponse {
  session_id: string;
  screenshot: string;
}

/**
 * POST /session/:id/record/input
 */
export type PointerAction = 'move' | 'down' | 'up' | 'click';
export type InputType = 'pointer' | 'keyboard' | 'wheel';

export interface InputRequest {
  type: InputType;
  session_id?: string;
  // Pointer
  action?: PointerAction;
  x?: number;
  y?: number;
  button?: 'left' | 'right' | 'middle';
  modifiers?: string[];
  // Keyboard
  key?: string;
  text?: string;
  // Wheel
  delta_x?: number;
  delta_y?: number;
}

/**
 * GET /session/:id/record/frame
 *
 * NOTE: Playwright only supports 'png' and 'jpeg' screenshot formats.
 * WebP would provide ~25% better compression but is NOT supported.
 * Do not attempt to use type: 'webp' - it fails at runtime.
 */
export interface FrameResponse {
  session_id: string;
  /** JPEG format - Playwright only supports png/jpeg, NOT webp */
  mime: 'image/jpeg';
  image: string;
  width: number;
  height: number;
  captured_at: string;
  /** MD5 hash of the raw frame buffer for reliable ETag generation */
  content_hash: string;
  /** Current page title (document.title) */
  page_title?: string;
  /** Current page URL */
  page_url?: string;
}

/**
 * POST /session/:id/record/viewport
 */
export interface ViewportRequest {
  width: number;
  height: number;
}

export interface ViewportResponse {
  session_id: string;
  width: number;
  height: number;
}

/**
 * POST /session/:id/record/stream-settings
 *
 * Updates stream settings for an active session.
 * Quality and FPS can be updated live. Scale changes require session restart.
 */
export interface StreamSettingsRequest {
  /** JPEG quality 1-100 */
  quality?: number;
  /** Target FPS 1-60 */
  fps?: number;
  /** Screenshot scale - can only be set at session start */
  scale?: 'css' | 'device';
  /** Enable/disable debug performance mode for this session */
  perfMode?: boolean;
}

export interface StreamSettingsResponse {
  session_id: string;
  /** Current quality setting */
  quality: number;
  /** Target FPS */
  fps: number;
  /** Current adaptive FPS (may be lower than target) */
  current_fps: number;
  /** Scale setting (cannot be changed mid-session) */
  scale: 'css' | 'device';
  /** Whether stream is currently active */
  is_streaming: boolean;
  /** Whether any settings were actually changed */
  updated: boolean;
  /** Warning if scale change was requested (requires restart) */
  scale_warning?: string;
  /** Whether debug performance mode is enabled */
  perf_mode: boolean;
}

/**
 * POST /session/:id/record/active-page
 *
 * Switch the active page for frame streaming and input forwarding.
 * Used in multi-tab recording sessions.
 */
export interface ActivePageRequest {
  /** Playwright page ID - the internal identifier for the target page */
  page_id: string;
}

export interface ActivePageResponse {
  session_id: string;
  /** The now-active page ID */
  active_page_id: string;
  /** Current URL of the active page */
  url: string;
  /** Title of the active page */
  title: string;
}

/**
 * Page event sent to the page callback URL.
 * Matches the Go DriverPageEvent structure.
 */
export interface DriverPageEvent {
  /** Session ID */
  sessionId: string;
  /** Driver's internal page ID (UUID) */
  driverPageId: string;
  /** Vrooli's page ID (echoed back if provided) */
  vrooliPageId: string;
  /** Event type: "created" | "navigated" | "closed" | "initial" */
  eventType: 'created' | 'navigated' | 'closed' | 'initial';
  /** Current URL of the page */
  url: string;
  /** Title of the page */
  title: string;
  /** Driver page ID of the opener page (if any) */
  openerDriverPageId?: string;
  /** ISO 8601 timestamp */
  timestamp: string;
}

// ========================================================================
// Browser History Types
// ========================================================================

/**
 * A single entry in the browser navigation history.
 */
export interface HistoryEntry {
  /** Unique ID for this entry */
  id: string;
  /** Page URL */
  url: string;
  /** Page title at time of navigation */
  title: string;
  /** ISO 8601 timestamp when navigation occurred */
  timestamp: string;
  /** Optional base64-encoded JPEG thumbnail (~150x100, quality 60) */
  thumbnail?: string;
}

/**
 * Settings for history capture behavior.
 */
export interface HistorySettings {
  /** Maximum number of entries to retain (default: 100) */
  maxEntries: number;
  /** TTL in days - entries older than this are pruned (default: 30, 0 = no TTL) */
  retentionDays: number;
  /** Whether to capture thumbnails (default: true) */
  captureThumbnails: boolean;
}

/**
 * Callback payload for history entry events.
 * Sent to the API when navigation occurs to persist history.
 */
export interface HistoryEntryCallback {
  session_id: string;
  entry: HistoryEntry;
  /** Navigation type: 'navigate' | 'back' | 'forward' */
  navigation_type: 'navigate' | 'back' | 'forward';
}
