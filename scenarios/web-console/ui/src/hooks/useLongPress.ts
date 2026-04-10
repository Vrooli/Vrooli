import { useCallback, useRef } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

const LONG_PRESS_MS = 500;

interface UseLongPressOptions {
  onPress: () => void;
  onLongPress: () => void;
}

/**
 * Detects click vs long-press (500ms) vs right-click on a button.
 * Uses pointerDown/pointerUp events so it doesn't conflict with
 * useDraggablePosition's handleClickCapture suppression.
 */
export function useLongPress({ onPress, onLongPress }: UseLongPressOptions) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const firedRef = useRef(false);

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const onPointerDown = useCallback(
    (e: ReactPointerEvent) => {
      if (e.pointerType === "mouse" && e.button !== 0) return;
      firedRef.current = false;
      clear();
      timerRef.current = setTimeout(() => {
        firedRef.current = true;
        onLongPress();
      }, LONG_PRESS_MS);
    },
    [clear, onLongPress],
  );

  const onPointerUp = useCallback(() => {
    if (!firedRef.current && timerRef.current) {
      onPress();
    }
    clear();
  }, [clear, onPress]);

  const onPointerCancel = useCallback(() => {
    clear();
  }, [clear]);

  const onContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      clear();
      if (!firedRef.current) {
        firedRef.current = true;
        onLongPress();
      }
    },
    [clear, onLongPress],
  );

  return { onPointerDown, onPointerUp, onPointerCancel, onContextMenu };
}
