/** @vrooliComponentSource hooks.use-swipe */
import { useRef, type PointerEvent as ReactPointerEvent } from "react";

export function useSwipe(
  onSwipe: (direction: "left" | "right" | "up" | "down") => void,
) {
  const start = useRef({ x: 0, y: 0 });
  return {
    onPointerDown: (event: ReactPointerEvent) => {
      start.current = { x: event.clientX, y: event.clientY };
    },
    onPointerUp: (event: ReactPointerEvent) => {
      const dx = event.clientX - start.current.x;
      const dy = event.clientY - start.current.y;
      if (Math.abs(dx) >= Math.abs(dy)) onSwipe(dx >= 0 ? "right" : "left");
      else onSwipe(dy >= 0 ? "down" : "up");
    },
  };
}
