import * as React from "react";

/**
 * useResizableSplit — controller for a draggable two-pane split.
 *
 * Returns a primary-pane percentage (0–100) and a `beginDrag` callback for
 * the drag handle. Subscribes to pointermove/pointerup on the window so a
 * single registry handles dragging regardless of where the cursor ends up.
 *
 * Orientation defaults to `horizontal` (primary pane sits to the inline-start
 * side). Use `vertical` to stack primary above secondary.
 */
export interface UseResizableSplitOptions {
  initialPercent?: number;
  min?: number;
  max?: number;
  orientation?: "horizontal" | "vertical";
}

export interface UseResizableSplitResult {
  percent: number;
  setPercent: (next: number) => void;
  /** Attach to the resize handle's onPointerDown to start dragging. */
  beginDrag: (event: React.PointerEvent | PointerEvent) => void;
  /** True while a drag is in flight (useful for cursor: col-resize on body). */
  isDragging: boolean;
}

export function useResizableSplit({
  initialPercent = 50,
  min = 15,
  max = 85,
  orientation = "horizontal",
}: UseResizableSplitOptions = {}): UseResizableSplitResult {
  const [percent, setPercentState] = React.useState<number>(
    clamp(initialPercent, min, max),
  );
  const [isDragging, setIsDragging] = React.useState(false);
  const containerRef = React.useRef<Element | null>(null);

  const setPercent = React.useCallback(
    (next: number) => setPercentState(clamp(next, min, max)),
    [min, max],
  );

  React.useEffect(() => {
    if (!isDragging) return;
    const onMove = (e: PointerEvent) => {
      const container = containerRef.current;
      if (!container) return;
      const rect = container.getBoundingClientRect();
      const ratio =
        orientation === "horizontal"
          ? (e.clientX - rect.left) / rect.width
          : (e.clientY - rect.top) / rect.height;
      setPercentState(clamp(ratio * 100, min, max));
    };
    const onUp = () => setIsDragging(false);
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [isDragging, orientation, min, max]);

  const beginDrag = React.useCallback(
    (event: React.PointerEvent | PointerEvent) => {
      const target = "currentTarget" in event ? event.currentTarget : null;
      const container = target instanceof Element ? target.parentElement : null;
      containerRef.current = container ?? null;
      setIsDragging(true);
    },
    [],
  );

  return { percent, setPercent, beginDrag, isDragging };
}

const clamp = (n: number, min: number, max: number): number =>
  Math.min(Math.max(n, min), max);
