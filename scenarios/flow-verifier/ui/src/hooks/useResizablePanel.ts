/**
 * useResizablePanel — pointer-drag panel resizing.
 *
 * Writes `targetRef.current.style.width` imperatively during drag and only
 * commits to React state on pointerup, so the React tree re-renders at most
 * once per drag instead of on every 60–100 Hz pointer tick. Callers spread
 * `style={{ width: size }}` on the target for initial render and after-drag
 * reconciliation.
 */
import {
  type PointerEvent,
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

const clamp = (value: number, min: number, max: number) =>
  Math.min(max, Math.max(min, value));

function loadPersistedSize(
  storageKey: string | undefined,
  fallback: number,
  min: number,
  max: number,
): number {
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
    // ignore persistence failures
  }
}

export interface UseResizablePanelOptions {
  containerRef: RefObject<HTMLElement | null>;
  targetRef: RefObject<HTMLElement | null>;
  minSize: number;
  maxSize: number;
  defaultSize: number;
  storageKey?: string;
  adjacentMinSize?: number;
  handleWidth?: number;
  resizeEdge?: "left" | "right";
  onSizeCommit?: (size: number) => void;
}

export interface UseResizablePanelReturn {
  size: number;
  isResizing: boolean;
  resizeHandleProps: {
    onPointerDown: (e: PointerEvent<HTMLDivElement>) => void;
    role: "separator";
    "aria-orientation": "vertical";
    "aria-valuenow": number;
    "aria-valuemin": number;
    "aria-valuemax": number;
    tabIndex: 0;
    "aria-label": string;
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
  onSizeCommit,
}: UseResizablePanelOptions): UseResizablePanelReturn {
  const [size, setSize] = useState(() =>
    loadPersistedSize(storageKey, defaultSize, minSize, maxSize),
  );
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
      if (!containerRef.current) return;
      const bounds = containerRef.current.getBoundingClientRect();
      const effectiveMax = Math.max(
        minSize,
        Math.min(maxSize, bounds.width - adjacentMinSize - handleWidth),
      );
      const raw =
        resizeEdge === "left"
          ? bounds.right - event.clientX
          : event.clientX - bounds.left;
      const next = clamp(raw, minSize, effectiveMax);
      liveSizeRef.current = next;
      const t = targetRef.current;
      if (t) t.style.width = `${next}px`;
    };

    const handlePointerUp = () => {
      const final = liveSizeRef.current;
      setIsResizing(false);
      setSize(final);
      savePersistedSize(storageKey, final);
      onSizeCommit?.(final);
    };

    const prevCursor = document.body.style.cursor;
    const prevSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);

    return () => {
      document.body.style.cursor = prevCursor;
      document.body.style.userSelect = prevSelect;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
    };
  }, [
    isResizing,
    containerRef,
    targetRef,
    minSize,
    maxSize,
    adjacentMinSize,
    handleWidth,
    storageKey,
    resizeEdge,
    onSizeCommit,
  ]);

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
      tabIndex: 0 as const,
      "aria-label": "Resize sidebar",
    },
  };
}
