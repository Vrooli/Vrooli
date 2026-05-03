/**
 * useResizablePanel – pointer-drag panel resizing.
 *
 * The hook drives the target element's width imperatively during drag (writing
 * `targetRef.current.style.width` on every pointermove) and only commits the
 * final size to React state on pointerup. This avoids re-rendering the React
 * tree on every pointer tick — a 60–100 Hz storm that otherwise cascades into
 * every sibling component. Callers spread `style={{ width: size }}` on the
 * target for the initial render (and for after-drag reconciliation); the hook
 * takes over during the drag itself.
 */

import { useState, useCallback, useEffect, useRef, type RefObject, type PointerEvent } from "react";

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

function loadPersistedSize(storageKey: string | undefined, fallback: number, min: number, max: number): number {
  if (!storageKey || typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(storageKey);
    if (!raw) return fallback;
    const parsed = Number(raw);
    if (!Number.isFinite(parsed)) return fallback;
    return clamp(parsed, min, max);
  } catch {
    return fallback;
  }
}

function savePersistedSize(storageKey: string | undefined, size: number): void {
  if (!storageKey || typeof window === "undefined") return;
  try {
    window.localStorage.setItem(storageKey, String(Math.round(size)));
  } catch {
    // Ignore persistence failures.
  }
}

export interface UseResizablePanelOptions {
  /** Ref to the outer container whose width constrains the resize range. */
  containerRef: RefObject<HTMLElement | null>;
  /** Ref to the element whose width is being controlled. The hook writes
   *  `style.width` directly during drag. */
  targetRef: RefObject<HTMLElement | null>;
  /** Resize axis. Only "horizontal" is supported today. */
  direction?: "horizontal";
  /** Minimum panel width in px. */
  minSize: number;
  /** Maximum panel width in px. */
  maxSize: number;
  /** Initial width in px. */
  defaultSize: number;
  /** localStorage key for persisted panel width. */
  storageKey?: string;
  /** Minimum width of the adjacent pane (prevents collapsing the other side). */
  adjacentMinSize?: number;
  /** Width of the resize handle in px (subtracted from available space). */
  handleWidth?: number;
}

export interface UseResizablePanelReturn {
  /** Current panel width in px. Updated only on pointerup, so React renders
   *  driven by this value happen at most once per drag. */
  size: number;
  /** Whether a drag is in progress (useful for adding `select-none` to the container). */
  isResizing: boolean;
  /** Props to spread on the resize-handle element. */
  resizeHandleProps: {
    onPointerDown: (e: PointerEvent<HTMLDivElement>) => void;
    role: "separator";
    "aria-orientation": "vertical";
    "aria-valuenow": number;
    "aria-valuemin": number;
    "aria-valuemax": number;
  };
}

export function useResizablePanel({
  containerRef,
  targetRef,
  minSize,
  maxSize,
  defaultSize,
  adjacentMinSize = 0,
  handleWidth = 0,
  storageKey,
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const [size, setSize] = useState(() => loadPersistedSize(storageKey, defaultSize, minSize, maxSize));
  const [isResizing, setIsResizing] = useState(false);

  // Tracks the latest size during drag so pointerup can commit it without a
  // round-trip through React state mid-drag.
  const liveSizeRef = useRef(size);
  useEffect(() => {
    liveSizeRef.current = size;
  }, [size]);

  const handlePointerDown = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (!containerRef.current) return;
      const bounds = containerRef.current.getBoundingClientRect();
      const effectiveMax = Math.max(
        minSize,
        Math.min(maxSize, bounds.width - adjacentMinSize - handleWidth),
      );
      const nextSize = clamp(event.clientX - bounds.left, minSize, effectiveMax);
      liveSizeRef.current = nextSize;
      const target = targetRef.current;
      if (target) {
        target.style.width = `${nextSize}px`;
      }
    };

    const handlePointerUp = () => {
      const finalSize = liveSizeRef.current;
      setIsResizing(false);
      setSize(finalSize);
      savePersistedSize(storageKey, finalSize);
    };

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);

    return () => {
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
    };
  }, [isResizing, containerRef, targetRef, minSize, maxSize, adjacentMinSize, handleWidth, storageKey]);

  return {
    size,
    isResizing,
    resizeHandleProps: {
      onPointerDown: handlePointerDown,
      role: "separator" as const,
      "aria-orientation": "vertical" as const,
      "aria-valuenow": Math.round(size),
      "aria-valuemin": minSize,
      "aria-valuemax": maxSize,
    },
  };
}
