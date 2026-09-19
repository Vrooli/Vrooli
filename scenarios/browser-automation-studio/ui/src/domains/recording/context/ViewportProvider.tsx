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
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { useViewportSyncManager } from '../utils/ViewportSyncManager';
import type {
  ViewportDimensions,
} from '../types/viewport';
import { ViewportContext } from './viewportContext';
import type { ActualViewportWithSource, ViewportContextValue } from './viewportContext';

// Re-export types for backward compatibility
export type { ViewportDimensions, ViewportSyncState } from '../types/viewport';

/**
 * Viewport with source attribution from the driver.
 * Alias for ActualViewportOptional (width, height, optional source/reason).
 */
export type { ActualViewportWithSource, ViewportContextActions, ViewportContextState, ViewportContextValue } from './viewportContext';

// =============================================================================
// Types
// =============================================================================

// =============================================================================
// Context
// =============================================================================

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

// Hooks are exported from viewportHooks.ts
