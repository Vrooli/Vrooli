/**
 * @libraryId react-component-library:useSwipe
 * @displayName useSwipe
 * @description Production-ready useSwipe hook with SSR-safe lifecycle behavior.
 * @version 2.0.1
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-swipe */
import { useRef, type PointerEvent as ReactPointerEvent } from "react";

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
  const start = useRef({ x: 0, y: 0, time: 0, pointerId: -1 });
  const distance = (event: ReactPointerEvent) => {
    const dx = event.clientX - start.current.x;
    const dy = event.clientY - start.current.y;
    if (direction === "left") return -dx;
    if (direction === "right") return dx;
    if (direction === "up") return -dy;
    return dy;
  };
  return {
    onPointerDown: (event: ReactPointerEvent) => {
      start.current = {
        x: event.clientX,
        y: event.clientY,
        time: performance.now(),
        pointerId: event.pointerId,
      };
      event.currentTarget.setPointerCapture(event.pointerId);
    },
    onPointerMove: (event: ReactPointerEvent) => {
      if (event.pointerId !== start.current.pointerId) return;
      onProgress?.(Math.max(0, Math.min(1, distance(event) / threshold)));
    },
    onPointerUp: (event: ReactPointerEvent) => {
      if (event.pointerId !== start.current.pointerId) return;
      const moved = Math.max(0, distance(event));
      const elapsed = Math.max(1, performance.now() - start.current.time);
      start.current.pointerId = -1;
      onProgress?.(0);
      if (moved >= threshold || moved / elapsed >= velocity) onCommit();
      else onCancel?.();
    },
    onPointerCancel: () => {
      start.current.pointerId = -1;
      onProgress?.(0);
      onCancel?.();
    },
  };
}
