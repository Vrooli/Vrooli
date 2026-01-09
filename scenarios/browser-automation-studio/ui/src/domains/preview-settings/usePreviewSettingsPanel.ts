/**
 * usePreviewSettingsPanel Hook
 *
 * Manages the preview settings panel state including:
 * - Panel width and resize handling
 * - Active section
 *
 * Width is persisted to localStorage for user preference retention.
 */

import { useState, useCallback, useEffect, useRef } from 'react';

// ============================================================================
// Constants
// ============================================================================

/** Minimum width of the panel */
export const PANEL_MIN_WIDTH = 360;

/** Maximum width of the panel */
export const PANEL_MAX_WIDTH = 600;

/** Default width of the panel */
export const PANEL_DEFAULT_WIDTH = 480;

/** Storage key for panel width */
const STORAGE_KEY_WIDTH = 'preview-settings-panel-width';

// ============================================================================
// LocalStorage Helpers
// ============================================================================

function getStoredNumber(key: string, defaultValue: number): number {
  if (typeof window === 'undefined') return defaultValue;
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
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(key, String(value));
  } catch {
    // Ignore storage errors
  }
}

// ============================================================================
// Hook Return Type
// ============================================================================

export interface UsePreviewSettingsPanelReturn {
  /** Current panel width in pixels */
  width: number;
  /** Minimum panel width */
  minWidth: number;
  /** Maximum panel width */
  maxWidth: number;
  /** Whether the panel is currently being resized */
  isResizing: boolean;
  /** Handler to start resizing (attach to resize handle) */
  handleResizeStart: (e: React.MouseEvent) => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function usePreviewSettingsPanel(): UsePreviewSettingsPanelReturn {
  // Width state
  const [width, setWidth] = useState(() =>
    getStoredNumber(STORAGE_KEY_WIDTH, PANEL_DEFAULT_WIDTH)
  );

  // Resize state
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(0);

  // Resize handling - for right panel, we drag from the left edge
  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
    resizeStartXRef.current = e.clientX;
    resizeStartWidthRef.current = width;
  }, [width]);

  // Mouse move handler for resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      // For a right-side panel, dragging left increases width
      const delta = resizeStartXRef.current - e.clientX;
      const newWidth = Math.min(
        PANEL_MAX_WIDTH,
        Math.max(PANEL_MIN_WIDTH, resizeStartWidthRef.current + delta)
      );
      setWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
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
    minWidth: PANEL_MIN_WIDTH,
    maxWidth: PANEL_MAX_WIDTH,
    isResizing,
    handleResizeStart,
  };
}
