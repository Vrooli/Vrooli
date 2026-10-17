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

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
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
  /** Canonical selected page; null suspends browser mutations. */
  pageId: string | null;
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

function sameViewport(a: ViewportDimensions | null, b: ViewportDimensions | null): boolean {
  return Boolean(a && b && a.width === b.width && a.height === b.height);
}

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
 *   pageId,
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
  const {sessionId, pageId, debounceMs = DEFAULT_DEBOUNCE_MS,
    resizeThresholdMs = DEFAULT_RESIZE_THRESHOLD_MS, minDimension = DEFAULT_MIN_DIMENSION,
    maxDimension = DEFAULT_MAX_DIMENSION} = config;
  const [viewport, setViewport] = useState<ViewportDimensions | null>(null);
  const [isResizing, setIsResizing] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [lastSyncTime, setLastSyncTime] = useState<number | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);
  const lastUpdateTimeRef = useRef(0);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const resizeEndTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingViewportRef = useRef<ViewportDimensions | null>(null);
  const lastSyncedViewportRef = useRef<ViewportDimensions | null>(null);
  const viewportRef = useRef<ViewportDimensions | null>(null);
  const owner = useMemo(() => ({sessionId, pageId, disposed: false, request: null as AbortController | null}), [sessionId, pageId]);
  const ownerRef = useRef(owner);
  ownerRef.current = owner;

  const clearTimers = useCallback(() => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    if (resizeEndTimerRef.current) clearTimeout(resizeEndTimerRef.current);
    debounceTimerRef.current = null;
    resizeEndTimerRef.current = null;
  }, []);

  const getClampedViewport = useCallback((bounds: ViewportDimensions): ViewportDimensions => ({
    width: Math.min(maxDimension, Math.max(minDimension, Math.round(bounds.width))),
    height: Math.min(maxDimension, Math.max(minDimension, Math.round(bounds.height))),
  }), [minDimension, maxDimension]);

  const syncToBackend = useCallback(async (desired: ViewportDimensions): Promise<void> => {
    if (!owner.sessionId || !owner.pageId || owner.disposed || sameViewport(lastSyncedViewportRef.current, desired)) return;
    owner.request?.abort();
    const controller = new AbortController();
    owner.request = controller;
    const current = () => !owner.disposed && !controller.signal.aborted && owner.request === controller && sameViewport(pendingViewportRef.current, desired);
    setIsSyncing(true);
    setSyncError(null);
    try {
      const appConfig = await getConfig();
      if (!current()) return;
      // Once sent, an aborted command may still change the browser.
      lastSyncedViewportRef.current = null;
      const response = await fetch(`${appConfig.API_URL}/recordings/live/${owner.sessionId}/viewport`, {
        method: 'POST', headers: {'Content-Type': 'application/json'}, signal: controller.signal,
        body: JSON.stringify({...desired, page_id: owner.pageId}),
      });
      if (!current()) return;
      if (!response.ok) throw new Error(await response.text() || `Viewport sync failed (${response.status})`);
      const actual: unknown = await response.json();
      if (!current()) return;
      if (!actual || typeof actual !== 'object' || !('width' in actual) || !('height' in actual) || actual.width !== desired.width || actual.height !== desired.height) {
        throw new Error('Browser did not confirm the requested viewport');
      }
      lastSyncedViewportRef.current = desired;
      setLastSyncTime(Date.now());
    } catch (error) {
      if (current()) setSyncError(error instanceof Error ? error.message : 'Viewport sync failed');
    } finally {
      if (current()) setIsSyncing(false);
    }
  }, [owner]);

  const updateFromBounds = useCallback((bounds: ViewportDimensions) => {
    if (owner.disposed) return;
    const now = performance.now();
    if (now - lastUpdateTimeRef.current < resizeThresholdMs) {
      setIsResizing(true);
      if (resizeEndTimerRef.current) clearTimeout(resizeEndTimerRef.current);
      resizeEndTimerRef.current = setTimeout(() => {setIsResizing(false);resizeEndTimerRef.current = null;}, resizeThresholdMs * 2);
    }
    lastUpdateTimeRef.current = now;
    const next = getClampedViewport(bounds);
    if (sameViewport(viewportRef.current, next) && sameViewport(pendingViewportRef.current, next)) return;
    owner.request?.abort();
    setIsSyncing(false);
    viewportRef.current = next;
    pendingViewportRef.current = next;
    setViewport(next);
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    debounceTimerRef.current = setTimeout(() => {
      if (pendingViewportRef.current) void syncToBackend(pendingViewportRef.current);
      debounceTimerRef.current = null;
    }, debounceMs);
  }, [owner, getClampedViewport, resizeThresholdMs, debounceMs, syncToBackend]);

  const forceSync = useCallback(async () => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    debounceTimerRef.current = null;
    if (pendingViewportRef.current) await syncToBackend(pendingViewportRef.current);
  }, [syncToBackend]);

  const reset = useCallback(() => {
    ownerRef.current.request?.abort();
    clearTimers();
    viewportRef.current = null;
    pendingViewportRef.current = null;
    lastSyncedViewportRef.current = null;
    setViewport(null);
    setIsResizing(false);
    setIsSyncing(false);
    setLastSyncTime(null);
    setSyncError(null);
  }, [clearTimers]);

  useLayoutEffect(() => {
    owner.disposed = false;
    lastSyncedViewportRef.current = null;
    setIsSyncing(false);
    setIsResizing(false);
    setLastSyncTime(null);
    setSyncError(null);
    return () => {owner.disposed = true;owner.request?.abort();clearTimers();};
  }, [owner, clearTimers]);
  useEffect(() => {reset();}, [sessionId, reset]);
  // A new selection needs the retained container bounds even without a resize.
  useEffect(() => {
    if (pendingViewportRef.current) void syncToBackend(pendingViewportRef.current);
  }, [syncToBackend]);

  const state = useMemo<ViewportSyncState>(() => ({viewport, isResizing, isSyncing, lastSyncTime, syncError}),
    [viewport, isResizing, isSyncing, lastSyncTime, syncError]);
  return {state, updateFromBounds, forceSync, reset, getClampedViewport};
}
