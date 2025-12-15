/**
 * Record Mode Routes
 *
 * Public API for Record Mode functionality. This module re-exports all
 * record mode handlers organized by responsibility:
 *
 * - recording-lifecycle: Start/stop/status/actions management
 * - recording-validation: Selector validation and replay preview
 * - recording-interaction: Browser interaction during recording
 */

// Types
export type {
  StartRecordingRequest,
  StartRecordingResponse,
  StopRecordingResponse,
  RecordingStatusResponse,
  ValidateSelectorRequest,
  ValidateSelectorResponse,
  ReplayPreviewRequest,
  ReplayPreviewResponse,
  NavigateRequest,
  NavigateResponse,
  ScreenshotRequest,
  ScreenshotResponse,
  InputRequest,
  InputType,
  PointerAction,
  FrameResponse,
  ViewportRequest,
  ViewportResponse,
  StreamSettingsRequest,
  StreamSettingsResponse,
} from './types';

// Recording lifecycle handlers
export {
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
  handleStreamSettings,
} from './recording-lifecycle';

// Recording validation handlers
export {
  handleValidateSelector,
  handleReplayPreview,
} from './recording-validation';

// Recording interaction handlers
export {
  handleRecordNavigate,
  handleRecordScreenshot,
  handleRecordInput,
  handleRecordFrame,
  handleRecordViewport,
  clearFrameCache,
  clearAllFrameCaches,
} from './recording-interaction';

// Cleanup utility
import { removeRecordingBuffer } from '../../recording/buffer';
import { clearFrameCache } from './recording-interaction';

/**
 * Clean up recording buffer and frame cache for a session
 */
export function cleanupSessionRecording(sessionId: string): void {
  removeRecordingBuffer(sessionId);
  clearFrameCache(sessionId);
}
