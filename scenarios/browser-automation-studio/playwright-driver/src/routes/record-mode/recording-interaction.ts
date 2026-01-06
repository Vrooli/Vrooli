/**
 * Recording Interaction (Barrel Export)
 *
 * This file has been refactored - functionality is now split into:
 * - recording-navigation.ts: Navigation handlers (navigate, reload, go-back, go-forward)
 * - recording-frames.ts: Frame capture and screenshot handlers
 * - recording-input.ts: Input forwarding (pointer, keyboard, wheel, viewport)
 * - recording-pages.ts: Multi-tab page management and history callbacks
 *
 * This file re-exports all handlers for backward compatibility.
 */

// Re-export from recording-navigation.ts
export {
  handleRecordNavigate,
  handleRecordReload,
  handleRecordGoBack,
  handleRecordGoForward,
  handleRecordNavigationState,
  handleRecordNavigationStack,
  getNavigationStack,
  clearNavigationState,
} from './recording-navigation';

// Re-export from recording-frames.ts
export {
  handleRecordFrame,
  handleRecordScreenshot,
  clearFrameCache,
  clearAllFrameCaches,
} from './recording-frames';

// Re-export from recording-input.ts
export {
  handleRecordInput,
  handleRecordViewport,
} from './recording-input';

// Re-export from recording-pages.ts
export {
  handleRecordNewPage,
  handleRecordActivePage,
  captureThumbnail,
  emitHistoryCallback,
} from './recording-pages';
