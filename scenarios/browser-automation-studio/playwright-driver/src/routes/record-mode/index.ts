/**
 * Record Mode Routes
 *
 * Public API for Record Mode functionality. This module re-exports all
 * record mode handlers organized by responsibility:
 *
 * - recording-lifecycle: Start/stop/status/actions management
 * - recording-validation: Selector validation and replay preview
 * - recording-navigation: URL navigation, back/forward, reload
 * - recording-frames: Frame capture and screenshots
 * - recording-input: Pointer, keyboard, wheel, viewport input
 * - recording-pages: Multi-tab page management
 * - recording-diagnostics-routes: Debug endpoints, pipeline testing
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

// Recording lifecycle handlers (core start/stop/status/actions)
export {
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
} from './recording-lifecycle';

// Recording diagnostics handlers (stream settings, debug, testing)
export {
  handleStreamSettings,
  handleRecordDebug,
  handleRecordPipelineTest,
  handleRecordExternalUrlTest,
} from './recording-diagnostics-routes';

// Recording validation handlers
export {
  handleValidateSelector,
  handleReplayPreview,
} from './recording-validation';

// Recording navigation handlers
export {
  handleRecordNavigate,
  handleRecordReload,
  handleRecordGoBack,
  handleRecordGoForward,
  handleRecordNavigationState,
  handleRecordNavigationStack,
  clearNavigationState,
} from './recording-navigation';

// Recording frame handlers
export {
  handleRecordFrame,
  handleRecordScreenshot,
  clearFrameCache,
  clearAllFrameCaches,
} from './recording-frames';

// Recording input handlers
export {
  handleRecordInput,
  handleRecordViewport,
} from './recording-input';

// Recording page handlers
export {
  handleRecordNewPage,
  handleRecordActivePage,
} from './recording-pages';

// Cleanup utility
import { removeRecordingBuffer } from '../../recording';
import { clearFrameCache } from './recording-frames';
import { clearNavigationState } from './recording-navigation';

/**
 * Clean up recording buffer, frame cache, and navigation state for a session
 */
export function cleanupSessionRecording(sessionId: string): void {
  removeRecordingBuffer(sessionId);
  clearFrameCache(sessionId);
  clearNavigationState(sessionId);
}
