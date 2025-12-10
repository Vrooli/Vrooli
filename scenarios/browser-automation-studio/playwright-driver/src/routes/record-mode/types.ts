/**
 * Record Mode API Types
 *
 * Request and response types for all Record Mode endpoints.
 * These define the HTTP contract for the recording feature.
 */

import type { RecordedAction } from '../../recording/types';

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
  /** Frame quality (0-100), default 65 */
  frame_quality?: number;
  /** Target FPS for frame streaming, default 6 */
  frame_fps?: number;
}

export interface StartRecordingResponse {
  recording_id: string;
  session_id: string;
  started_at: string;
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
  actions: RecordedAction[];
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
    action_id: string;
    sequence_num: number;
    action_type: string;
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
  screenshot?: string;
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
