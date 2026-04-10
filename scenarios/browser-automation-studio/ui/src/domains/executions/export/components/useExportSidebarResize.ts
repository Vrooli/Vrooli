/**
 * useExportSidebarResize Hook
 *
 * Manages the export sidebar resize state including:
 * - Sidebar width and resize handling
 *
 * Width is persisted to localStorage for user preference retention.
 * This is for a LEFT-side panel, so dragging right increases width.
 */

import { useState, useCallback, useEffect, useRef } from "react";

// ============================================================================
// Constants
// ============================================================================

/** Minimum width of the sidebar */
export const SIDEBAR_MIN_WIDTH = 280;

/** Maximum width of the sidebar */
export const SIDEBAR_MAX_WIDTH = 480;

/** Default width of the sidebar */
export const SIDEBAR_DEFAULT_WIDTH = 320;

/** Storage key for sidebar width */
const STORAGE_KEY_WIDTH = "export-stylization-sidebar-width";

// ============================================================================
// LocalStorage Helpers
// ============================================================================

function getStoredNumber(key: string, defaultValue: number): number {
  if (typeof window === "undefined") return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed)) return parsed;
    }
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function setStoredValue(key: string, value: string | number): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(key, String(value));
  } catch {
    // Ignore storage errors
  }
}

// ============================================================================
// Hook Return Type
// ============================================================================

export interface UseExportSidebarResizeReturn {
  /** Current sidebar width in pixels */
  width: number;
  /** Minimum sidebar width */
  minWidth: number;
  /** Maximum sidebar width */
  maxWidth: number;
  /** Whether the sidebar is currently being resized */
  isResizing: boolean;
  /** Handler to start resizing (attach to resize handle) */
  handleResizeStart: (e: React.MouseEvent) => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useExportSidebarResize(): UseExportSidebarResizeReturn {
  // Width state
  const [width, setWidth] = useState(() =>
    getStoredNumber(STORAGE_KEY_WIDTH, SIDEBAR_DEFAULT_WIDTH),
  );

  // Resize state
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(0);

  // Resize handling - for left panel, we drag from the right edge
  const handleResizeStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);
      resizeStartXRef.current = e.clientX;
      resizeStartWidthRef.current = width;
    },
    [width],
  );

  // Mouse move handler for resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      // For a left-side panel, dragging right increases width
      const delta = e.clientX - resizeStartXRef.current;
      const newWidth = Math.min(
        SIDEBAR_MAX_WIDTH,
        Math.max(SIDEBAR_MIN_WIDTH, resizeStartWidthRef.current + delta),
      );
      setWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing]);

  // Persist width when resize ends
  useEffect(() => {
    if (!isResizing) {
      setStoredValue(STORAGE_KEY_WIDTH, width);
    }
  }, [isResizing, width]);

  return {
    width,
    minWidth: SIDEBAR_MIN_WIDTH,
    maxWidth: SIDEBAR_MAX_WIDTH,
    isResizing,
    handleResizeStart,
  };
}
