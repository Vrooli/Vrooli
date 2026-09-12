import { useState, useEffect, useRef, useCallback } from "react";

export interface UseResizablePanelOptions {
  /** Storage key for localStorage persistence (will be prefixed with "agm.") */
  storageKey: string;
  /** Optional full localStorage key override (for legacy key compatibility) */
  persistKey?: string;
  /** Resize direction */
  axis?: "horizontal" | "vertical";
  /** Default width in pixels */
  defaultWidth?: number;
  /** Minimum width in pixels */
  minWidth?: number;
  /** Minimum space required for the other panel */
  minOtherWidth?: number;
  /** Default size in pixels (alias of defaultWidth, preferred for vertical resizing) */
  defaultSize?: number;
  /** Minimum size in pixels (alias of minWidth, preferred for vertical resizing) */
  minSize?: number;
  /** Minimum space required for the other panel (alias of minOtherWidth) */
  minOtherSize?: number;
}

export interface UseResizablePanelReturn {
  /** Current panel size in pixels */
  size: number;
  /** Current width in pixels (backward-compatible alias of size) */
  width: number;
  /** Whether resize is in progress */
  isResizing: boolean;
  /** Mouse down handler for the resize divider */
  handleResizeStart: (e: React.MouseEvent) => void;
  /** Ref to attach to the container element */
  containerRef: React.RefObject<HTMLDivElement>;
}

interface ResizeRef {
  startPointer: number;
  startSize: number;
  containerSize: number;
  lastSize: number;
}

const STORAGE_PREFIX = "agm.panel.";

/**
 * Hook for managing resizable panel width with localStorage persistence.
 * Based on git-control-tower resize pattern.
 */
export function useResizablePanel({
  storageKey,
  persistKey,
  axis = "horizontal",
  defaultWidth,
  minWidth,
  minOtherWidth,
  defaultSize,
  minSize,
  minOtherSize,
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const resolvedDefaultSize = defaultSize ?? defaultWidth ?? 320;
  const resolvedMinSize = minSize ?? minWidth ?? 200;
  const resolvedMinOtherSize = minOtherSize ?? minOtherWidth ?? 200;
  const storageSuffix = axis === "vertical" ? "height" : "width";
  const fullStorageKey = persistKey ?? `${STORAGE_PREFIX}${storageKey}.${storageSuffix}`;
  const containerRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<ResizeRef | null>(null);
  const animationFrameRef = useRef<number | null>(null);
  const pendingSizeRef = useRef<number | null>(null);

  // Initialize from localStorage or default
  const [size, setSize] = useState(() => {
    if (typeof window === "undefined") return resolvedDefaultSize;
    const stored = localStorage.getItem(fullStorageKey);
    if (stored) {
      const parsed = Number(stored);
      if (Number.isFinite(parsed) && parsed > 0) {
        return parsed;
      }
    }
    return resolvedDefaultSize;
  });

  const [isResizing, setIsResizing] = useState(false);
  const [previewSize, setPreviewSize] = useState<number | null>(null);

  const displayedSize = previewSize ?? size;

  // Save only committed sizes. Drag previews update React at most once per frame
  // and persist on mouseup instead of writing localStorage on every pointer move.
  useEffect(() => {
    localStorage.setItem(fullStorageKey, String(size));
  }, [fullStorageKey, size]);

  // Clamp width when container resizes (e.g., window resize)
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const clamp = () => {
      const containerSize = axis === "vertical" ? container.clientHeight : container.clientWidth;
      const maxSize = containerSize - resolvedMinOtherSize;
      if (size > maxSize) {
        setSize(Math.max(resolvedMinSize, maxSize));
      }
    };

    // Initial clamp
    clamp();

    const observer = new ResizeObserver(clamp);
    observer.observe(container);

    return () => observer.disconnect();
  }, [axis, resolvedMinSize, resolvedMinOtherSize, size]);

  const handleResizeStart = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      const container = containerRef.current;
      if (!container) return;

      resizeRef.current = {
        startPointer: axis === "vertical" ? event.clientY : event.clientX,
        startSize: displayedSize,
        containerSize: axis === "vertical" ? container.clientHeight : container.clientWidth,
        lastSize: displayedSize,
      };
      pendingSizeRef.current = displayedSize;
      setPreviewSize(displayedSize);
      setIsResizing(true);
    },
    [axis, displayedSize]
  );

  // Handle mouse move and up during resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMove = (event: MouseEvent) => {
      if (!resizeRef.current) return;

      const pointer = axis === "vertical" ? event.clientY : event.clientX;
      const { startPointer, startSize, containerSize } = resizeRef.current;
      const delta = pointer - startPointer;
      const newSize = startSize + delta;

      // Calculate max width based on container and minimum other panel width
      const maxSize = containerSize - resolvedMinOtherSize;
      const clampedSize = Math.max(resolvedMinSize, Math.min(maxSize, newSize));

      resizeRef.current.lastSize = clampedSize;
      pendingSizeRef.current = clampedSize;

      if (animationFrameRef.current === null) {
        animationFrameRef.current = window.requestAnimationFrame(() => {
          animationFrameRef.current = null;
          if (pendingSizeRef.current !== null) {
            setPreviewSize(pendingSizeRef.current);
          }
        });
      }
    };

    const handleUp = () => {
      const finalSize = resizeRef.current?.lastSize ?? pendingSizeRef.current ?? size;
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
      pendingSizeRef.current = null;
      setSize(finalSize);
      setPreviewSize(null);
      setIsResizing(false);
      resizeRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    // Set cursor and prevent text selection during resize
    document.body.style.cursor = axis === "vertical" ? "row-resize" : "col-resize";
    document.body.style.userSelect = "none";

    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [axis, isResizing, resolvedMinSize, resolvedMinOtherSize, size]);

  return {
    size: displayedSize,
    width: displayedSize,
    isResizing,
    handleResizeStart,
    containerRef,
  };
}
