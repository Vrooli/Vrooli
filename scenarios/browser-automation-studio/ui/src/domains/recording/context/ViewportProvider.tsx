/**
 * ViewportProvider - Centralized viewport state management via React Context
 *
 * This context consolidates all viewport-related state and operations:
 * - Container bounds measurement (from ResizeObserver)
 * - Browser viewport (what Playwright uses - stable, container-based)
 * - Display viewport (for CSS rendering - may differ with replay style)
 * - Actual viewport feedback (from Playwright driver, may differ due to profile)
 * - Sync status (debouncing, syncing, errors)
 *
 * Benefits:
 * - Eliminates prop drilling for viewport state
 * - Single source of truth for all viewport-related data
 * - Child components can access viewport info via useViewport() hook
 * - Centralizes sync logic with the backend
 *
 * Architecture:
 * - Provider wraps the recording session content
 * - Uses ViewportSyncManager internally for debouncing and backend sync
 * - Exposes both state and actions through context
 */

import {
  createContext,
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { useViewportSyncManager } from '../utils/ViewportSyncManager';
import type {
  ViewportDimensions,
  ViewportSyncState,
  ActualViewportOptional,
} from '../types/viewport';

// Re-export types for backward compatibility
export type { ViewportDimensions, ViewportSyncState } from '../types/viewport';

/**
 * Viewport with source attribution from the driver.
 * Alias for ActualViewportOptional (width, height, optional source/reason).
 */
export type ActualViewportWithSource = ActualViewportOptional;

// =============================================================================
// Types
// =============================================================================

export interface ViewportContextState {
  /**
   * Current browser viewport (from container bounds, synced to Playwright).
   * This is the "source of truth" for what Playwright uses.
   */
  browserViewport: ViewportDimensions | null;

  /**
   * Actual viewport reported by Playwright driver with source attribution.
   * May differ from browserViewport due to session profile overrides.
   * Includes `source` and `reason` fields for attribution.
   */
  actualViewport: ActualViewportWithSource | null;

  /**
   * Whether there's a mismatch between requested and actual viewport.
   */
  hasMismatch: boolean;

  /**
   * Reason for viewport mismatch (if any).
   */
  mismatchReason: string | null;

  /**
   * Sync status from ViewportSyncManager.
   */
  syncState: ViewportSyncState;

  /**
   * Session ID being managed.
   */
  sessionId: string | null;
}

export interface ViewportContextActions {
  /**
   * Update viewport from container bounds (triggers debounced sync).
   * Call this from ResizeObserver callbacks.
   */
  updateFromBounds: (bounds: ViewportDimensions) => void;

  /**
   * Set the actual viewport reported by the driver.
   * Call this when session creation or viewport update returns actual dimensions.
   */
  setActualViewport: (viewport: ViewportDimensions | null) => void;

  /**
   * Force immediate sync to backend (bypasses debounce).
   */
  forceSync: () => Promise<void>;

  /**
   * Reset all viewport state (call on session change).
   */
  reset: () => void;

  /**
   * Get clamped viewport dimensions.
   */
  getClampedViewport: (bounds: ViewportDimensions) => ViewportDimensions;
}

export interface ViewportContextValue extends ViewportContextState, ViewportContextActions {}

// =============================================================================
// Context
// =============================================================================

const ViewportContext = createContext<ViewportContextValue | null>(null);

// =============================================================================
// Provider
// =============================================================================

export interface ViewportProviderProps {
  /**
   * Session ID for API calls. When null, viewport won't sync to backend.
   */
  sessionId: string | null;

  /**
   * Actual viewport from external source (e.g., session creation response).
   * This prop is reactive - changes will update the internal state.
   * Includes optional source and reason fields for attribution.
   */
  actualViewport?: ActualViewportWithSource | null;

  /**
   * Debounce delay for viewport sync (default: 200ms).
   */
  debounceMs?: number;

  /**
   * Threshold for detecting rapid resize (default: 100ms).
   */
  resizeThresholdMs?: number;

  /**
   * Children to render.
   */
  children: ReactNode;
}

const DIMENSION_TOLERANCE = 5; // pixels

export function ViewportProvider({
  sessionId,
  actualViewport: externalActualViewport = null,
  debounceMs = 200,
  resizeThresholdMs = 100,
  children,
}: ViewportProviderProps) {
  // Actual viewport from driver (may differ from requested due to profile)
  const [actualViewport, setActualViewportState] = useState<ActualViewportWithSource | null>(
    externalActualViewport
  );

  // Sync external actualViewport prop to internal state
  useEffect(() => {
    setActualViewportState(externalActualViewport);
  }, [externalActualViewport]);

  // Use ViewportSyncManager for debouncing and backend sync
  const syncManager = useViewportSyncManager({
    sessionId,
    debounceMs,
    resizeThresholdMs,
  });

  // Compute mismatch info
  const hasMismatch = useMemo(() => {
    const browserVp = syncManager.state.viewport;
    if (!browserVp || !actualViewport) return false;

    const widthDiff = Math.abs(browserVp.width - actualViewport.width);
    const heightDiff = Math.abs(browserVp.height - actualViewport.height);

    return widthDiff > DIMENSION_TOLERANCE || heightDiff > DIMENSION_TOLERANCE;
  }, [syncManager.state.viewport, actualViewport]);

  const mismatchReason = useMemo(() => {
    if (!hasMismatch) return null;
    // Use the reason from the driver if available, otherwise a default message
    return actualViewport?.reason || 'Session profile has viewport override configured';
  }, [hasMismatch, actualViewport?.reason]);

  // Action: Set actual viewport (accepts full type with source attribution)
  const setActualViewport = useCallback((viewport: ActualViewportWithSource | ViewportDimensions | null) => {
    setActualViewportState(viewport as ActualViewportWithSource | null);
  }, []);

  // Combine all state and actions
  const value = useMemo<ViewportContextValue>(
    () => ({
      // State
      browserViewport: syncManager.state.viewport,
      actualViewport,
      hasMismatch,
      mismatchReason,
      syncState: syncManager.state,
      sessionId,

      // Actions
      updateFromBounds: syncManager.updateFromBounds,
      setActualViewport,
      forceSync: syncManager.forceSync,
      reset: syncManager.reset,
      getClampedViewport: syncManager.getClampedViewport,
    }),
    [
      syncManager.state,
      syncManager.updateFromBounds,
      syncManager.forceSync,
      syncManager.reset,
      syncManager.getClampedViewport,
      actualViewport,
      hasMismatch,
      mismatchReason,
      sessionId,
      setActualViewport,
    ]
  );

  return <ViewportContext.Provider value={value}>{children}</ViewportContext.Provider>;
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to access viewport context.
 * Must be used within a ViewportProvider.
 */
export function useViewport(): ViewportContextValue {
  const context = useContext(ViewportContext);
  if (!context) {
    throw new Error('useViewport must be used within a ViewportProvider');
  }
  return context;
}

/**
 * Hook to access viewport context, returning null if not within provider.
 * Useful for components that may be used outside the viewport context.
 */
export function useViewportOptional(): ViewportContextValue | null {
  return useContext(ViewportContext);
}

