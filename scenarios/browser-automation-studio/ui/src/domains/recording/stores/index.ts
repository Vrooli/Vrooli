/**
 * Recording Domain Stores
 *
 * Central exports for all recording-related Zustand stores.
 */

export {
  useSessionStore,
  // Selector hooks
  useSessionId,
  useIsSessionValidated,
  useIsCreatingSession,
  useIsValidatingSession,
  useSessionError,
  useRetryState,
  useActualViewport,
  usePages,
  usePageList,
  useOpenPages,
  useActivePageId,
  useActivePage,
  usePageColorMap,
  useTimelineEntries,
  useTimelineLoading,
  useFrameDimensions,
  useDisplayDimensions,
  useSessionState,
  // Types
  type PageColor,
  type FrameDimensions,
  type DisplayDimensions,
} from './sessionStore';
