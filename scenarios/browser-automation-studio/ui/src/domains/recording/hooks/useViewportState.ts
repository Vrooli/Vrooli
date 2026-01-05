/**
 * useViewportState - Centralized viewport state management
 *
 * This hook separates the concepts of:
 * - Browser viewport: What Playwright/CDP actually uses (stable)
 * - Display viewport: How we render in the UI (can be scaled for replay style)
 *
 * Key design decisions:
 * 1. Browser viewport is based on container bounds and stays stable during style toggles
 * 2. Style toggle only affects visual rendering, not the actual browser viewport
 * 3. All dimension sources are tracked explicitly for debugging
 * 4. Sync status is exposed for UI feedback
 *
 * This addresses the flickering issue by ensuring the browser viewport doesn't
 * change when toggling replay style - only the CSS transform changes.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';

// =============================================================================
// Types
// =============================================================================

export interface Dimensions {
  width: number;
  height: number;
}

export interface ViewportStateConfig {
  /** Session ID for API calls */
  sessionId: string | null;
  /** Debounce delay for viewport sync (default: 200ms) */
  debounceMs?: number;
  /** Minimum viewport dimension (default: 320) */
  minDimension?: number;
  /** Maximum viewport dimension (default: 3840) */
  maxDimension?: number;
}

export interface ViewportDimensionInfo {
  /** The dimensions */
  dimensions: Dimensions | null;
  /** Source of these dimensions */
  source: 'container' | 'settings' | 'browser_profile' | 'synced' | 'unknown';
  /** When these dimensions were last updated */
  updatedAt: number;
}

export interface ViewportSyncStatus {
  /** Whether a sync is in progress */
  isSyncing: boolean;
  /** Whether a resize is being detected (rapid changes) */
  isResizing: boolean;
  /** Last successful sync timestamp */
  lastSyncTime: number | null;
  /** Error from last sync attempt */
  syncError: string | null;
}

export interface ViewportState {
  /**
   * Container bounds - measured from the DOM element.
   * This is the available space for the preview.
   */
  containerBounds: ViewportDimensionInfo;

  /**
   * Browser viewport - what Playwright/CDP is actually using.
   * Based on container bounds, stays stable during style toggles.
   */
  browserViewport: ViewportDimensionInfo;

  /**
   * Display viewport - how we render in the UI.
   * When replay style is on, this may differ from browserViewport
   * (scaled/transformed for presentation).
   */
  displayViewport: ViewportDimensionInfo;

  /**
   * Actual browser dimensions reported from the driver.
   * May differ from requested due to browser profile overrides.
   */
  actualBrowserDimensions: ViewportDimensionInfo;

  /**
   * Sync status for UI feedback
   */
  syncStatus: ViewportSyncStatus;

  /**
   * Whether there's a mismatch between requested and actual dimensions
   */
  hasDimensionMismatch: boolean;

  /**
   * Human-readable reason for dimension mismatch (if any)
   */
  mismatchReason: string | null;
}

export interface ViewportStateActions {
  /**
   * Update container bounds (call from ResizeObserver)
   */
  updateContainerBounds: (bounds: Dimensions) => void;

  /**
   * Update display settings (from replay settings, doesn't affect browser)
   */
  updateDisplaySettings: (settings: { width: number; height: number }) => void;

  /**
   * Report actual browser dimensions (from driver response)
   */
  reportActualDimensions: (dimensions: Dimensions, source?: string) => void;

  /**
   * Force immediate sync to backend
   */
  forceSync: () => Promise<void>;

  /**
   * Reset state (call on session change)
   */
  reset: () => void;

  /**
   * Get the viewport to use for coordinate mapping (always browserViewport)
   */
  getCoordinateMappingViewport: () => Dimensions | null;
}

export type UseViewportStateReturn = ViewportState & ViewportStateActions;

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_DEBOUNCE_MS = 200;
const DEFAULT_MIN_DIMENSION = 320;
const DEFAULT_MAX_DIMENSION = 3840;
const RESIZE_THRESHOLD_MS = 100;
const DIMENSION_MISMATCH_TOLERANCE = 5; // pixels

// =============================================================================
// Helper Functions
// =============================================================================

function clampDimensions(
  dims: Dimensions,
  min: number,
  max: number
): Dimensions {
  return {
    width: Math.min(max, Math.max(min, Math.round(dims.width))),
    height: Math.min(max, Math.max(min, Math.round(dims.height))),
  };
}

function dimensionsEqual(
  a: Dimensions | null,
  b: Dimensions | null,
  tolerance = DIMENSION_MISMATCH_TOLERANCE
): boolean {
  if (!a || !b) return a === b;
  return (
    Math.abs(a.width - b.width) <= tolerance &&
    Math.abs(a.height - b.height) <= tolerance
  );
}

function createDimensionInfo(
  dimensions: Dimensions | null,
  source: ViewportDimensionInfo['source']
): ViewportDimensionInfo {
  return {
    dimensions,
    source,
    updatedAt: Date.now(),
  };
}

// =============================================================================
// Hook Implementation
// =============================================================================

export function useViewportState(config: ViewportStateConfig): UseViewportStateReturn {
  const {
    sessionId,
    debounceMs = DEFAULT_DEBOUNCE_MS,
    minDimension = DEFAULT_MIN_DIMENSION,
    maxDimension = DEFAULT_MAX_DIMENSION,
  } = config;

  // Core state
  const [containerBounds, setContainerBounds] = useState<ViewportDimensionInfo>(
    createDimensionInfo(null, 'unknown')
  );
  const [browserViewport, setBrowserViewport] = useState<ViewportDimensionInfo>(
    createDimensionInfo(null, 'unknown')
  );
  const [displaySettings, setDisplaySettings] = useState<Dimensions | null>(null);
  const [actualBrowserDimensions, setActualBrowserDimensions] = useState<ViewportDimensionInfo>(
    createDimensionInfo(null, 'unknown')
  );

  // Sync status
  const [isSyncing, setIsSyncing] = useState(false);
  const [isResizing, setIsResizing] = useState(false);
  const [lastSyncTime, setLastSyncTime] = useState<number | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);

  // Refs for tracking
  const lastUpdateTimeRef = useRef<number>(0);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const resizeEndTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingViewportRef = useRef<Dimensions | null>(null);
  const lastSyncedViewportRef = useRef<Dimensions | null>(null);

  // Compute display viewport (may be scaled for replay style)
  const displayViewport = useMemo<ViewportDimensionInfo>(() => {
    if (displaySettings) {
      return createDimensionInfo(displaySettings, 'settings');
    }
    // Default to browser viewport
    return browserViewport;
  }, [displaySettings, browserViewport]);

  // Compute mismatch info
  const hasDimensionMismatch = useMemo(() => {
    if (!browserViewport.dimensions || !actualBrowserDimensions.dimensions) {
      return false;
    }
    return !dimensionsEqual(browserViewport.dimensions, actualBrowserDimensions.dimensions);
  }, [browserViewport.dimensions, actualBrowserDimensions.dimensions]);

  const mismatchReason = useMemo(() => {
    if (!hasDimensionMismatch) return null;
    if (actualBrowserDimensions.source === 'browser_profile') {
      return 'Session profile has viewport override configured';
    }
    return 'Browser using different dimensions than requested';
  }, [hasDimensionMismatch, actualBrowserDimensions.source]);

  // Sync viewport to backend
  const syncToBackend = useCallback(
    async (viewportToSync: Dimensions): Promise<void> => {
      if (!sessionId) return;

      // Skip if viewport hasn't changed from last sync
      if (
        lastSyncedViewportRef.current &&
        dimensionsEqual(lastSyncedViewportRef.current, viewportToSync, 1)
      ) {
        return;
      }

      setIsSyncing(true);
      setSyncError(null);

      try {
        const appConfig = await getConfig();
        const response = await fetch(
          `${appConfig.API_URL}/recordings/live/${sessionId}/viewport`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              width: viewportToSync.width,
              height: viewportToSync.height,
            }),
          }
        );

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `Viewport sync failed (${response.status})`);
        }

        lastSyncedViewportRef.current = viewportToSync;
        setLastSyncTime(Date.now());

        // Update browser viewport to synced value
        setBrowserViewport(createDimensionInfo(viewportToSync, 'synced'));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Viewport sync failed';
        setSyncError(message);
        console.warn('[useViewportState] Sync failed:', message);
      } finally {
        setIsSyncing(false);
      }
    },
    [sessionId]
  );

  // Update container bounds (from ResizeObserver)
  const updateContainerBounds = useCallback(
    (bounds: Dimensions) => {
      const now = performance.now();
      const timeSinceLastUpdate = now - lastUpdateTimeRef.current;
      lastUpdateTimeRef.current = now;

      // Detect rapid resize
      if (timeSinceLastUpdate < RESIZE_THRESHOLD_MS) {
        setIsResizing(true);
        if (resizeEndTimerRef.current) {
          clearTimeout(resizeEndTimerRef.current);
        }
        resizeEndTimerRef.current = setTimeout(() => {
          setIsResizing(false);
          resizeEndTimerRef.current = null;
        }, RESIZE_THRESHOLD_MS * 2);
      }

      // Clamp and round
      const clampedBounds = clampDimensions(bounds, minDimension, maxDimension);

      // Update container bounds
      setContainerBounds(createDimensionInfo(clampedBounds, 'container'));

      // Browser viewport follows container bounds (not display settings!)
      // This is the key insight: browser viewport is based on container,
      // not on replay style settings
      pendingViewportRef.current = clampedBounds;

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
    [syncToBackend, debounceMs, minDimension, maxDimension]
  );

  // Update display settings (from replay settings - doesn't sync to backend!)
  const updateDisplaySettings = useCallback((settings: { width: number; height: number }) => {
    setDisplaySettings(clampDimensions(settings, minDimension, maxDimension));
  }, [minDimension, maxDimension]);

  // Report actual dimensions from driver
  const reportActualDimensions = useCallback(
    (dimensions: Dimensions, source?: string) => {
      setActualBrowserDimensions(
        createDimensionInfo(
          clampDimensions(dimensions, minDimension, maxDimension),
          (source as ViewportDimensionInfo['source']) || 'unknown'
        )
      );
    },
    [minDimension, maxDimension]
  );

  // Force immediate sync
  const forceSync = useCallback(async () => {
    if (pendingViewportRef.current) {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
        debounceTimerRef.current = null;
      }
      await syncToBackend(pendingViewportRef.current);
    }
  }, [syncToBackend]);

  // Reset state
  const reset = useCallback(() => {
    setContainerBounds(createDimensionInfo(null, 'unknown'));
    setBrowserViewport(createDimensionInfo(null, 'unknown'));
    setDisplaySettings(null);
    setActualBrowserDimensions(createDimensionInfo(null, 'unknown'));
    setIsSyncing(false);
    setIsResizing(false);
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

  // Get viewport for coordinate mapping (always browser viewport)
  const getCoordinateMappingViewport = useCallback(() => {
    return browserViewport.dimensions;
  }, [browserViewport.dimensions]);

  // Reset on session change
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

  return {
    // State
    containerBounds,
    browserViewport,
    displayViewport,
    actualBrowserDimensions,
    syncStatus: {
      isSyncing,
      isResizing,
      lastSyncTime,
      syncError,
    },
    hasDimensionMismatch,
    mismatchReason,

    // Actions
    updateContainerBounds,
    updateDisplaySettings,
    reportActualDimensions,
    forceSync,
    reset,
    getCoordinateMappingViewport,
  };
}

export default useViewportState;
