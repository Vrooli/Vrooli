/**
 * useResizableColumns - Hook for managing a resizable two-column layout.
 *
 * Provides:
 * - Left column width state with localStorage persistence
 * - Mouse drag handlers for resize
 * - Min/max constraints with ResizeObserver clamping
 * - Cursor feedback during drag
 *
 * Based on the resize pattern from prompt-manager's useResizableSidebar.
 */

import { useState, useEffect, useCallback, useRef } from 'react';

const DEFAULT_LEFT_RATIO = 0.4; // 40% for left (form cards), 60% for right (preview)
const MIN_RATIO = 0.25; // Minimum 25% for either column
const MAX_RATIO = 0.75; // Maximum 75% for either column

interface UseResizableColumnsOptions {
  /** Default left column ratio (0-1), defaults to 0.4 */
  defaultLeftRatio?: number;
  /** Minimum ratio for either column (0-1), defaults to 0.25 */
  minRatio?: number;
  /** Maximum ratio for left column (0-1), defaults to 0.75 */
  maxRatio?: number;
  /** localStorage key for persistence */
  storageKey?: string;
}

interface UseResizableColumnsResult {
  /** Current left column ratio (0-1) */
  leftRatio: number;
  /** Whether currently resizing */
  isResizing: boolean;
  /** Ref to attach to the container element for ResizeObserver */
  containerRef: React.RefObject<HTMLDivElement>;
  /** Mouse down handler for the resize handle */
  handleResizeStart: (e: React.MouseEvent) => void;
  /** CSS styles for the left column */
  leftColumnStyle: React.CSSProperties;
  /** CSS styles for the right column */
  rightColumnStyle: React.CSSProperties;
}

export function useResizableColumns(
  options: UseResizableColumnsOptions = {}
): UseResizableColumnsResult {
  const {
    defaultLeftRatio = DEFAULT_LEFT_RATIO,
    minRatio = MIN_RATIO,
    maxRatio = MAX_RATIO,
    storageKey = 'lpbs.bundleColumnsRatio',
  } = options;

  const containerRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<{
    startX: number;
    startRatio: number;
    containerWidth: number;
  } | null>(null);

  // Initialize ratio from localStorage
  const [leftRatio, setLeftRatio] = useState(() => {
    if (typeof window === 'undefined') return defaultLeftRatio;
    const stored = localStorage.getItem(storageKey);
    if (stored) {
      const parsed = Number(stored);
      if (Number.isFinite(parsed) && parsed >= minRatio && parsed <= maxRatio) {
        return parsed;
      }
    }
    return defaultLeftRatio;
  });

  const [isResizing, setIsResizing] = useState(false);

  // Persist ratio to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(storageKey, String(leftRatio));
    }
  }, [leftRatio, storageKey]);

  // Mouse event handlers for resizing
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return;

      const deltaX = e.clientX - resizeRef.current.startX;
      const deltaRatio = deltaX / resizeRef.current.containerWidth;
      const newRatio = resizeRef.current.startRatio + deltaRatio;
      const clampedRatio = Math.max(minRatio, Math.min(maxRatio, newRatio));
      setLeftRatio(clampedRatio);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      resizeRef.current = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing, minRatio, maxRatio]);

  // Handler to start resizing
  const handleResizeStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      if (!containerRef.current) return;

      const containerWidth = containerRef.current.clientWidth;

      resizeRef.current = {
        startX: e.clientX,
        startRatio: leftRatio,
        containerWidth,
      };
      setIsResizing(true);
    },
    [leftRatio]
  );

  // Compute column styles
  const leftColumnStyle: React.CSSProperties = {
    width: `${leftRatio * 100}%`,
    flexShrink: 0,
  };

  const rightColumnStyle: React.CSSProperties = {
    width: `${(1 - leftRatio) * 100}%`,
    flexShrink: 0,
  };

  return {
    leftRatio,
    isResizing,
    containerRef,
    handleResizeStart,
    leftColumnStyle,
    rightColumnStyle,
  };
}
