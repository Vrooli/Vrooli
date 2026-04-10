import { useRef, useCallback } from "react";

interface UseLongPressOptions {
  onLongPress: () => void;
  onTap?: () => void;
  threshold?: number;
  moveTolerance?: number;
}

export function useLongPress({
  onLongPress,
  onTap,
  threshold = 450,
  moveTolerance = 10,
}: UseLongPressOptions) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startPos = useRef<{ x: number; y: number } | null>(null);
  const firedRef = useRef(false);
  const cancelledRef = useRef(false);

  const clear = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const onTouchStart = useCallback(
    (e: React.TouchEvent) => {
      clear();
      firedRef.current = false;
      cancelledRef.current = false;
      const touch = e.touches[0];
      if (!touch) return;
      startPos.current = { x: touch.clientX, y: touch.clientY };
      timerRef.current = setTimeout(() => {
        firedRef.current = true;
        timerRef.current = null;
        try {
          navigator.vibrate(50);
        } catch {
          // iOS ignores vibrate; swallow errors
        }
        onLongPress();
      }, threshold);
    },
    [onLongPress, threshold, clear],
  );

  const onTouchMove = useCallback(
    (e: React.TouchEvent) => {
      if (!startPos.current) return;
      const touch = e.touches[0];
      if (!touch) return;
      const dx = touch.clientX - startPos.current.x;
      const dy = touch.clientY - startPos.current.y;
      if (Math.abs(dx) > moveTolerance || Math.abs(dy) > moveTolerance) {
        clear();
        cancelledRef.current = true;
      }
    },
    [moveTolerance, clear],
  );

  const onTouchEnd = useCallback(() => {
    clear();
    if (!firedRef.current && !cancelledRef.current && onTap) {
      onTap();
    }
    startPos.current = null;
  }, [clear, onTap]);

  const onTouchCancel = useCallback(() => {
    clear();
    cancelledRef.current = true;
    startPos.current = null;
  }, [clear]);

  return { onTouchStart, onTouchMove, onTouchEnd, onTouchCancel };
}
