/**
 * useResizablePanel – reusable hook for pointer-drag panel resizing.
 *
 * Extracted from the identical implementations in BacklogDetailsPage and
 * PromptsPage. Returns state + handler props so callers retain full control
 * over the resize-handle JSX.
 */

import { useState, useCallback, useEffect, type RefObject, type PointerEvent } from "react";

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
  /** Current panel width in px. */
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
  minSize,
  maxSize,
  defaultSize,
  adjacentMinSize = 0,
  handleWidth = 0,
  storageKey,
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const [size, setSize] = useState(() => loadPersistedSize(storageKey, defaultSize, minSize, maxSize));
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    savePersistedSize(storageKey, size);
  }, [size, storageKey]);

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
      setSize(nextSize);
    };

    const handlePointerUp = () => {
      setIsResizing(false);
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
  }, [isResizing, containerRef, minSize, maxSize, adjacentMinSize, handleWidth]);

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
