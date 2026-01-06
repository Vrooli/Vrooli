/**
 * ViewportSyncManager - Centralized viewport coordination for recording sessions
 *
 * This module provides viewport state management and debounced sync to backend.
 *
 * Architecture:
 * - Uses React hooks for state management
 * - Debounces rapid viewport changes (e.g., during sidebar drag)
 * - Provides isResizing flag for UI transition states
 * - Syncs viewport to backend API which updates Playwright and frame streaming
 *
 * Types are imported from the consolidated types/viewport.ts module.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import type { ViewportDimensions, ViewportSyncState } from '../types/viewport';

// Re-export types for backward compatibility
export type { ViewportDimensions, ViewportSyncState } from '../types/viewport';

// Re-export utility functions for backward compatibility
export { viewportsEqual, getAspectRatio, fitViewportToBounds } from '../types/viewport';

// =============================================================================
// Types (local to this module)
// =============================================================================

export interface ViewportSyncConfig {
  /** Session ID for API calls */
  sessionId: string | null;
  /** Debounce delay for viewport sync (default: 200ms) */
  debounceMs?: number;
  /** Threshold for detecting rapid resize (default: 100ms) */
  resizeThresholdMs?: number;
  /** Minimum viewport dimension (default: 320) */
  minDimension?: number;
  /** Maximum viewport dimension (default: 3840) */
  maxDimension?: number;
}

export interface ViewportSyncManager {
  /** Current viewport state */
  state: ViewportSyncState;
  /** Update viewport from container bounds (triggers debounced sync) */
  updateFromBounds: (bounds: ViewportDimensions) => void;
  /** Force immediate sync to backend */
  forceSync: () => Promise<void>;
  /** Reset state (call on session change) */
  reset: () => void;
  /** Get clamped viewport dimensions */
  getClampedViewport: (bounds: ViewportDimensions) => ViewportDimensions;
}

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_DEBOUNCE_MS = 200;
const DEFAULT_RESIZE_THRESHOLD_MS = 100;
const DEFAULT_MIN_DIMENSION = 320;
const DEFAULT_MAX_DIMENSION = 3840;

// =============================================================================
// Hook Implementation
// =============================================================================

/**
 * Hook to manage viewport synchronization with the backend.
 *
 * Usage:
 * ```tsx
 * const { state, updateFromBounds, forceSync } = useViewportSyncManager({
 *   sessionId,
 *   debounceMs: 200,
 * });
 *
 * // In ResizeObserver callback:
 * updateFromBounds({ width, height });
 *
 * // Show loading state during resize:
 * {state.isResizing && <LoadingOverlay />}
 * ```
 */
export function useViewportSyncManager(config: ViewportSyncConfig): ViewportSyncManager {
  const {
    sessionId,
    debounceMs = DEFAULT_DEBOUNCE_MS,
    resizeThresholdMs = DEFAULT_RESIZE_THRESHOLD_MS,
    minDimension = DEFAULT_MIN_DIMENSION,
    maxDimension = DEFAULT_MAX_DIMENSION,
  } = config;

  // State
  const [viewport, setViewport] = useState<ViewportDimensions | null>(null);
  const [isResizing, setIsResizing] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [lastSyncTime, setLastSyncTime] = useState<number | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);

  // Refs for tracking resize behavior
  const lastUpdateTimeRef = useRef<number>(0);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const resizeEndTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingViewportRef = useRef<ViewportDimensions | null>(null);
  const lastSyncedViewportRef = useRef<ViewportDimensions | null>(null);

  // Clamp viewport dimensions
  const getClampedViewport = useCallback(
    (bounds: ViewportDimensions): ViewportDimensions => ({
      width: Math.min(maxDimension, Math.max(minDimension, Math.round(bounds.width))),
      height: Math.min(maxDimension, Math.max(minDimension, Math.round(bounds.height))),
    }),
    [minDimension, maxDimension]
  );

  // Sync viewport to backend
  const syncToBackend = useCallback(
    async (viewportToSync: ViewportDimensions): Promise<void> => {
      if (!sessionId) return;

      // Skip if viewport hasn't changed from last sync
      if (
        lastSyncedViewportRef.current &&
        lastSyncedViewportRef.current.width === viewportToSync.width &&
        lastSyncedViewportRef.current.height === viewportToSync.height
      ) {
        return;
      }

      setIsSyncing(true);
      setSyncError(null);

      try {
        const appConfig = await getConfig();
        const response = await fetch(`${appConfig.API_URL}/recordings/live/${sessionId}/viewport`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            width: viewportToSync.width,
            height: viewportToSync.height,
          }),
        });

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `Viewport sync failed (${response.status})`);
        }

        lastSyncedViewportRef.current = viewportToSync;
        setLastSyncTime(Date.now());
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Viewport sync failed';
        setSyncError(message);
        console.warn('[ViewportSyncManager] Sync failed:', message);
      } finally {
        setIsSyncing(false);
      }
    },
    [sessionId]
  );

  // Update viewport from bounds (with debouncing and resize detection)
  const updateFromBounds = useCallback(
    (bounds: ViewportDimensions) => {
      const now = performance.now();
      const timeSinceLastUpdate = now - lastUpdateTimeRef.current;
      lastUpdateTimeRef.current = now;

      // Detect rapid resize (e.g., sidebar drag)
      if (timeSinceLastUpdate < resizeThresholdMs) {
        setIsResizing(true);

        // Clear existing resize end timer
        if (resizeEndTimerRef.current) {
          clearTimeout(resizeEndTimerRef.current);
        }

        // Set new resize end timer
        resizeEndTimerRef.current = setTimeout(() => {
          setIsResizing(false);
          resizeEndTimerRef.current = null;
        }, resizeThresholdMs * 2);
      }

      // Compute clamped viewport
      const clampedViewport = getClampedViewport(bounds);

      // Update local state immediately for responsive UI
      setViewport(clampedViewport);
      pendingViewportRef.current = clampedViewport;

      // Debounce backend sync
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }

      debounceTimerRef.current = setTimeout(() => {
        if (pendingViewportRef.current) {
          void syncToBackend(pendingViewportRef.current);
        }
        debounceTimerRef.current = null;
      }, debounceMs);
    },
    [getClampedViewport, syncToBackend, debounceMs, resizeThresholdMs]
  );

  // Force immediate sync
  const forceSync = useCallback(async () => {
    if (pendingViewportRef.current) {
      // Clear debounce timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
        debounceTimerRef.current = null;
      }

      await syncToBackend(pendingViewportRef.current);
    }
  }, [syncToBackend]);

  // Reset state (call on session change)
  const reset = useCallback(() => {
    setViewport(null);
    setIsResizing(false);
    setIsSyncing(false);
    setLastSyncTime(null);
    setSyncError(null);
    pendingViewportRef.current = null;
    lastSyncedViewportRef.current = null;

    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }

    if (resizeEndTimerRef.current) {
      clearTimeout(resizeEndTimerRef.current);
      resizeEndTimerRef.current = null;
    }
  }, []);

  // Reset when session changes
  useEffect(() => {
    reset();
  }, [sessionId, reset]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (resizeEndTimerRef.current) {
        clearTimeout(resizeEndTimerRef.current);
      }
    };
  }, []);

  // Memoize state object
  const state = useMemo<ViewportSyncState>(
    () => ({
      viewport,
      isResizing,
      isSyncing,
      lastSyncTime,
      syncError,
    }),
    [viewport, isResizing, isSyncing, lastSyncTime, syncError]
  );

  return {
    state,
    updateFromBounds,
    forceSync,
    reset,
    getClampedViewport,
  };
}

