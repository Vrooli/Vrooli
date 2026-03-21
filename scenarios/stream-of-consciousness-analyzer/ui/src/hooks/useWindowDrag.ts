import { useRef, useEffect, useCallback } from "react";

/** Callback invoked when a drag completes with the total displacement. */
export type DragEndHandler = (dx: number, dy: number) => void;

/** Callback invoked continuously during drag with current displacement. */
export type DragMoveHandler = (dx: number, dy: number) => void;

interface UseWindowDragOptions {
  /** Called on each mousemove during drag. */
  onMove?: DragMoveHandler;
  /** Called on mouseup with final displacement. */
  onEnd?: DragEndHandler;
  /** Scaling factor applied to displacement (e.g. 1/zoom). Default: 1 */
  scale?: number;
}

interface UseWindowDragReturn {
  /** Call from a mousedown handler to begin tracking a drag. */
  startDrag: (e: React.MouseEvent | MouseEvent) => void;
  /** Ref that is true while a drag is active. */
  isDragging: React.RefObject<boolean>;
}

/**
 * Manages window-level mouse drag interactions.
 *
 * Attaches mousemove/mouseup to `window` so drags that leave the
 * originating element are still captured. Cleans up listeners on
 * unmount to prevent leaks.
 */
export function useWindowDrag({
  onMove,
  onEnd,
  scale = 1,
}: UseWindowDragOptions): UseWindowDragReturn {
  const isDragging = useRef(false);
  const cleanupRef = useRef<(() => void) | null>(null);

  // Clean up any active drag on unmount
  useEffect(() => {
    return () => {
      cleanupRef.current?.();
    };
  }, []);

  const startDrag = useCallback(
    (e: React.MouseEvent | MouseEvent) => {
      const startX = e.clientX;
      const startY = e.clientY;
      isDragging.current = true;

      const handleMove = (me: MouseEvent) => {
        const dx = (me.clientX - startX) * scale;
        const dy = (me.clientY - startY) * scale;
        onMove?.(dx, dy);
      };

      const handleUp = (me: MouseEvent) => {
        window.removeEventListener("mousemove", handleMove);
        window.removeEventListener("mouseup", handleUp);
        cleanupRef.current = null;
        isDragging.current = false;
        const dx = (me.clientX - startX) * scale;
        const dy = (me.clientY - startY) * scale;
        onEnd?.(dx, dy);
      };

      window.addEventListener("mousemove", handleMove);
      window.addEventListener("mouseup", handleUp);
      cleanupRef.current = () => {
        window.removeEventListener("mousemove", handleMove);
        window.removeEventListener("mouseup", handleUp);
        isDragging.current = false;
      };
    },
    [onMove, onEnd, scale],
  );

  return { startDrag, isDragging };
}
