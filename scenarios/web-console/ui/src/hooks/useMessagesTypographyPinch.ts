import { useEffect, useRef, useState, type RefObject } from "react";
import { MESSAGES_FONT_MAX, MESSAGES_FONT_MIN } from "../stores/useMessagesViewStore";

export function clampMessagesFont(size: number): number {
  return Math.min(MESSAGES_FONT_MAX, Math.max(MESSAGES_FONT_MIN, Math.round(size)));
}

function distance(touches: TouchList): number {
  const first = touches[0];
  const second = touches[1];
  return first && second ? Math.hypot(first.clientX - second.clientX, first.clientY - second.clientY) : 0;
}

/** Shares the Messages pinch contract between the virtualized list and reader. */
export function useMessagesTypographyPinch(
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
      gesture.latest = clampMessagesFont(gesture.startSize * currentDistance / gesture.startDistance);
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
