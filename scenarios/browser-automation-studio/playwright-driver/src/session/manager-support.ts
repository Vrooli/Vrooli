/**
 * Internal collaborator surface for the session lifecycle coordinator.
 * SessionManager owns admission and lease state; these modules own the focused
 * browser, page, audio, diagnostics, reset, and inspection behaviors.
 */
export { buildContext, type ActualViewport } from './context-builder';
export { transition, canTransition, canAcceptInstructions } from './state-machine';
export { setupDiagnosticLogging } from './diagnostic-logger';
export {
  countActiveSessions,
  inspectSession,
  isSessionActive,
  findIdleSessions,
  listSessions,
  summarizeSessions,
  type SessionInfo,
  type SessionListEntry,
  type SessionSummary,
} from './session-inspection';
export { resetSessionState } from './session-reset';
export { DriverPageBindings } from './page-bindings';
export { teardownSessionResources } from './session-teardown';
export { resetPageInputState } from './live-input';
export { clearFrameCache } from './frame-cache';
export {
  selectAppTargetPage,
  validateAppTargetCapabilities,
  validateAppTargetSpec,
  verifyAppTargetRenderer,
} from './electron-target';
export { BrowserManager, type BrowserStatus } from './browser-manager';
export {
  applySilentSinkToCurrentPage,
  generateSilentSinkPatch,
  type AudioStrategy,
} from './audio';
export {
  createPipeWireQualificationDevice,
  PIPEWIRE_QUALIFICATION_DEVICE_NAME,
  verifyBrowserCaptureDevice,
  type BrowserCaptureDeviceEvidence,
  type PipeWireQualificationDevice,
} from './audio/device-evidence';
export { PerformanceTracer, injectWebVitalsObserver, AccessibilitySnapshotter } from '../tracing';
