import { useEffect, useRef, useState, type RefObject } from "react";

/** Bounds and default of the reading text size, shared by every surface that offers it. */
export const READER_FONT_MIN = 12;
export const READER_FONT_MAX = 32;
export const READER_FONT_DEFAULT = 16;

export function clampReaderFontSize(size: number): number {
  return Math.min(READER_FONT_MAX, Math.max(READER_FONT_MIN, Math.round(size)));
}

function distance(touches: TouchList): number {
  const first = touches[0];
  const second = touches[1];
  return first && second ? Math.hypot(first.clientX - second.clientX, first.clientY - second.clientY) : 0;
}

/**
 * Two-finger pinch to resize text. Returns the live size while a gesture is in
 * progress (null otherwise) and calls `onCommit` once, with the final size,
 * when the fingers lift, so a persisted preference is written once per pinch.
 */
export function useTypographyPinch(
  targetRef: RefObject<HTMLElement | null>,
  fontSize: number,
  onCommit: (size: number) => void,
): number | null {
  const [preview, setPreview] = useState<number | null>(null);
  const sizeRef = useRef(fontSize);
  const commitRef = useRef(onCommit);
  sizeRef.current = fontSize;
  commitRef.current = onCommit;

  useEffect(() => {
    const target = targetRef.current;
    if (!target) return;
    let gesture: { startDistance: number; startSize: number; latest: number | null } | null = null;
    const onStart = (event: TouchEvent) => {
      if (event.touches.length === 2) {
        const startDistance = distance(event.touches);
        if (startDistance > 0) gesture = { startDistance, startSize: sizeRef.current, latest: null };
      }
    };
    const onMove = (event: TouchEvent) => {
      if (!gesture || event.touches.length !== 2) return;
      const currentDistance = distance(event.touches);
      if (currentDistance <= 0) return;
      event.preventDefault();
      gesture.latest = clampReaderFontSize(gesture.startSize * currentDistance / gesture.startDistance);
      setPreview(gesture.latest);
    };
    const onEnd = (event: TouchEvent) => {
      if (!gesture || event.touches.length >= 2) return;
      if (gesture.latest !== null) commitRef.current(gesture.latest);
      gesture = null;
      setPreview(null);
    };
    target.addEventListener("touchstart", onStart, { passive: true });
    target.addEventListener("touchmove", onMove, { passive: false });
    target.addEventListener("touchend", onEnd);
    target.addEventListener("touchcancel", onEnd);
    return () => {
      target.removeEventListener("touchstart", onStart);
      target.removeEventListener("touchmove", onMove);
      target.removeEventListener("touchend", onEnd);
      target.removeEventListener("touchcancel", onEnd);
    };
  }, [targetRef]);

  return preview;
}
