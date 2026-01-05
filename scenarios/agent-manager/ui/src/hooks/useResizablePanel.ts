import { useState, useEffect, useRef, useCallback } from "react";

export interface UseResizablePanelOptions {
  /** Storage key for localStorage persistence (will be prefixed with "agm.") */
  storageKey: string;
  /** Default width in pixels */
  defaultWidth: number;
  /** Minimum width in pixels */
  minWidth: number;
  /** Minimum space required for the other panel */
  minOtherWidth: number;
}

export interface UseResizablePanelReturn {
  /** Current width in pixels */
  width: number;
  /** Whether resize is in progress */
  isResizing: boolean;
  /** Mouse down handler for the resize divider */
  handleResizeStart: (e: React.MouseEvent) => void;
  /** Ref to attach to the container element */
  containerRef: React.RefObject<HTMLDivElement>;
}

interface ResizeRef {
  startX: number;
  startWidth: number;
  containerWidth: number;
}

const STORAGE_PREFIX = "agm.panel.";

/**
 * Hook for managing resizable panel width with localStorage persistence.
 * Based on git-control-tower resize pattern.
 */
export function useResizablePanel({
  storageKey,
  defaultWidth,
  minWidth,
  minOtherWidth,
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const fullStorageKey = `${STORAGE_PREFIX}${storageKey}.width`;
  const containerRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<ResizeRef | null>(null);

  // Initialize from localStorage or default
  const [width, setWidth] = useState(() => {
    if (typeof window === "undefined") return defaultWidth;
    const stored = localStorage.getItem(fullStorageKey);
    if (stored) {
      const parsed = Number(stored);
      if (Number.isFinite(parsed) && parsed > 0) {
        return parsed;
      }
    }
    return defaultWidth;
  });

  const [isResizing, setIsResizing] = useState(false);

  // Save to localStorage when width changes
  useEffect(() => {
    localStorage.setItem(fullStorageKey, String(width));
  }, [fullStorageKey, width]);

  // Clamp width when container resizes (e.g., window resize)
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const clamp = () => {
      const containerWidth = container.clientWidth;
      const maxWidth = containerWidth - minOtherWidth;
      if (width > maxWidth) {
        setWidth(Math.max(minWidth, maxWidth));
      }
    };

    // Initial clamp
    clamp();

    const observer = new ResizeObserver(clamp);
    observer.observe(container);

    return () => observer.disconnect();
  }, [width, minWidth, minOtherWidth]);

  const handleResizeStart = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      const container = containerRef.current;
      if (!container) return;

      resizeRef.current = {
        startX: event.clientX,
        startWidth: width,
        containerWidth: container.clientWidth,
      };
      setIsResizing(true);
    },
    [width]
  );

  // Handle mouse move and up during resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMove = (event: MouseEvent) => {
      if (!resizeRef.current) return;

      const { startX, startWidth, containerWidth } = resizeRef.current;
      const delta = event.clientX - startX;
      const newWidth = startWidth + delta;

      // Calculate max width based on container and minimum other panel width
      const maxWidth = containerWidth - minOtherWidth;
      const clampedWidth = Math.max(minWidth, Math.min(maxWidth, newWidth));

      setWidth(clampedWidth);
    };

    const handleUp = () => {
      setIsResizing(false);
      resizeRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    // Set cursor and prevent text selection during resize
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, minWidth, minOtherWidth]);

  return {
    width,
    isResizing,
    handleResizeStart,
    containerRef,
  };
}
