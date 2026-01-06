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
  ReloadRequest,
  ReloadResponse,
  GoBackRequest,
  GoBackResponse,
  GoForwardRequest,
  GoForwardResponse,
  NavigationStateResponse,
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
  ActivePageRequest,
  ActivePageResponse,
} from './types';

// Recording lifecycle handlers
export {
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
  handleStreamSettings,
  handleRecordDebug,
  handleRecordPipelineTest,
  handleRecordExternalUrlTest,
} from './recording-lifecycle';

// Recording validation handlers
export {
  handleValidateSelector,
  handleReplayPreview,
} from './recording-validation';

// Recording interaction handlers
export {
  handleRecordNavigate,
  handleRecordReload,
  handleRecordGoBack,
  handleRecordGoForward,
  handleRecordNavigationState,
  handleRecordNavigationStack,
  handleRecordScreenshot,
  handleRecordInput,
  handleRecordFrame,
  handleRecordViewport,
  handleRecordNewPage,
  handleRecordActivePage,
  clearFrameCache,
  clearAllFrameCaches,
  clearNavigationState,
} from './recording-interaction';

// Cleanup utility
import { removeRecordingBuffer } from '../../recording/buffer';
import { clearFrameCache, clearNavigationState } from './recording-interaction';

/**
 * Clean up recording buffer, frame cache, and navigation state for a session
 */
export function cleanupSessionRecording(sessionId: string): void {
  removeRecordingBuffer(sessionId);
  clearFrameCache(sessionId);
  clearNavigationState(sessionId);
}
