import { useCallback, useEffect, useRef, useState, type PointerEvent, type RefObject } from "react";

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

function loadPersistedSize(storageKey: string | undefined, fallback: number, min: number, max: number): number {
  if (!storageKey || typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(storageKey);
    if (!raw) return fallback;
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? clamp(parsed, min, max) : fallback;
  } catch {
    return fallback;
  }
}

function savePersistedSize(storageKey: string | undefined, size: number): void {
  if (!storageKey || typeof window === "undefined") return;
  try {
    window.localStorage.setItem(storageKey, String(Math.round(size)));
  } catch {
    // Ignore local persistence failures.
  }
}

interface UseResizablePanelOptions {
  containerRef: RefObject<HTMLElement | null>;
  targetRef: RefObject<HTMLElement | null>;
  minSize: number;
  maxSize: number;
  defaultSize: number;
  storageKey?: string;
  adjacentMinSize?: number;
  handleWidth?: number;
  resizeEdge?: "left" | "right";
}

export interface UseResizablePanelReturn {
  size: number;
  isResizing: boolean;
  resizeHandleProps: {
    onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
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
  resizeEdge = "right",
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const [size, setSize] = useState(() => loadPersistedSize(storageKey, defaultSize, minSize, maxSize));
  const [isResizing, setIsResizing] = useState(false);
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
      const container = containerRef.current;
      if (!container) return;
      const bounds = container.getBoundingClientRect();
      const effectiveMax = Math.max(
        minSize,
        Math.min(maxSize, bounds.width - adjacentMinSize - handleWidth),
      );
      const rawSize = resizeEdge === "left" ? bounds.right - event.clientX : event.clientX - bounds.left;
      const nextSize = clamp(rawSize, minSize, effectiveMax);
      liveSizeRef.current = nextSize;
      if (targetRef.current) {
        targetRef.current.style.width = `${nextSize}px`;
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
    window.addEventListener("pointercancel", handlePointerUp);

    return () => {
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
      window.removeEventListener("pointercancel", handlePointerUp);
    };
  }, [adjacentMinSize, containerRef, handleWidth, isResizing, maxSize, minSize, resizeEdge, storageKey, targetRef]);

  return {
    size,
    isResizing,
    resizeHandleProps: {
      onPointerDown: handlePointerDown,
      role: "separator",
      "aria-orientation": "vertical",
      "aria-valuenow": Math.round(size),
      "aria-valuemin": minSize,
      "aria-valuemax": maxSize,
    },
  };
}
