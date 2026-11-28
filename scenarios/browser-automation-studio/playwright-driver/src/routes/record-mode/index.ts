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
 * - recording-stream-settings-route: Recording stream configuration
 * - recording-diagnostics-routes: Debug and external URL injection checks
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
  NavigationResponse,
  HistoryNavigationRequest,
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
  handleRecordActionsAck,
} from './recording-lifecycle';

// Recording stream configuration
export { handleStreamSettings } from './recording-stream-settings-route';

// Recording diagnostics handlers (debug, testing)
export {
  handleRecordDebug,
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
} from './recording-navigation';

// Recording frame handlers
export {
  handleRecordFrame,
  handleRecordScreenshot,
} from './recording-frames';
export { clearFrameCache, clearAllFrameCaches } from '../../session/frame-cache';

// Recording input handlers
export {
  handleRecordInput,
  handleRecordViewport,
} from './recording-input';

// Recording page handlers
export {
  handleRecordNewPage,
  handleRecordActivePage,
  handleRecordClosePage,
} from './recording-pages';
