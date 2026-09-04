/**
 * @libraryId react-component-library:useSwipe
 * @displayName useSwipe
 * @version 2.0.4
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-swipe */
import { useCallback, useRef, type PointerEvent as ReactPointerEvent } from "react";

/**
 * Pointer capture keeps a drag attached to the element it started on when the
 * finger leaves that element, which is exactly what a dismissing drag does.
 * It is also absent in jsdom and in older WebKit, where an unguarded call is a
 * TypeError on the first touch — the gesture does not degrade, it throws. Both
 * calls are guarded, and the capture is released rather than left held.
 */
const withPointerCapture = (
  target: EventTarget & Element,
  pointerId: number,
  method: "setPointerCapture" | "releasePointerCapture",
) => {
  const capture = (target as Partial<Element>)[method];
  if (typeof capture !== "function") return;
  try {
    capture.call(target, pointerId);
  } catch {
    // A pointer that is already released, or was never captured, is not an
    // error the gesture needs to react to.
  }
};

export type SwipeDirection = "left" | "right" | "up" | "down";
export interface SwipeOptions {
  direction: SwipeDirection;
  threshold?: number;
  velocity?: number;
  onProgress?: (progress: number) => void;
  onCommit: () => void;
  onCancel?: () => void;
}

export function useSwipe({
  direction,
  threshold = 96,
  velocity = 0.5,
  onProgress,
  onCommit,
  onCancel,
}: SwipeOptions) {
  const start = useRef({
    x: 0,
    y: 0,
    time: 0,
    pointerId: -1,
    target: null as Element | null,
  });
  const distance = (event: ReactPointerEvent) => {
    const dx = event.clientX - start.current.x;
    const dy = event.clientY - start.current.y;
    if (direction === "left") return -dx;
    if (direction === "right") return dx;
    if (direction === "up") return -dy;
    return dy;
  };
  const cancel = useCallback(() => {
    const gesture = start.current;
    if (gesture.pointerId < 0) return;
    if (gesture.target)
      withPointerCapture(gesture.target, gesture.pointerId, "releasePointerCapture");
    gesture.pointerId = -1;
    gesture.target = null;
    onProgress?.(0);
    onCancel?.();
  }, [onCancel, onProgress]);

  return {
    cancel,
    onPointerDown: (event: ReactPointerEvent) => {
      start.current = {
        x: event.clientX,
        y: event.clientY,
        time: performance.now(),
        pointerId: event.pointerId,
        target: event.currentTarget,
      };
      withPointerCapture(event.currentTarget, event.pointerId, "setPointerCapture");
    },
    onPointerMove: (event: ReactPointerEvent) => {
      if (event.pointerId !== start.current.pointerId) return;
      onProgress?.(Math.max(0, Math.min(1, distance(event) / threshold)));
    },
    onPointerUp: (event: ReactPointerEvent) => {
      if (event.pointerId !== start.current.pointerId) return;
      withPointerCapture(event.currentTarget, event.pointerId, "releasePointerCapture");
      const moved = Math.max(0, distance(event));
      const elapsed = Math.max(1, performance.now() - start.current.time);
      start.current.pointerId = -1;
      start.current.target = null;
      onProgress?.(0);
      if (moved >= threshold || moved / elapsed >= velocity) onCommit();
      else onCancel?.();
    },
    onPointerCancel: () => {
      cancel();
    },
  };
}
