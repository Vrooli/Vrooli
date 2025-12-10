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
  /** Optional callback URL to stream actions to (for API integration) */
  callback_url?: string;
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
 */
export interface FrameResponse {
  session_id: string;
  mime: 'image/jpeg';
  image: string;
  width: number;
  height: number;
  captured_at: string;
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
