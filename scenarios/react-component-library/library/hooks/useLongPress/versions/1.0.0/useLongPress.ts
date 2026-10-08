/**
 * @libraryId react-component-library:useLongPress
 * @displayName useLongPress
 * @description Stationary pointer gesture with cancellation and origin capture.
 * @version 1.0.0
 * @tags ["gesture","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-long-press */
import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { withPointerCapture } from "@vrooli/react-component-library/GestureDirection/1";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";

export interface LongPressOrigin { x: number; y: number; pointerType: string }
export interface UseLongPressOptions {
  onLongPress: (origin: LongPressOrigin) => void;
  delay?: number;
  moveTolerance?: number;
  disabled?: boolean;
  onCancel?: (reason: "moved" | "released" | "aborted") => void;
}

export function useLongPress(options: UseLongPressOptions): {
  longPressProps: Pick<React.DOMAttributes<HTMLElement>, "onPointerDown" | "onPointerMove" | "onPointerUp" | "onPointerCancel" | "onContextMenu" | "onClick">;
  fired: boolean;
} {
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const active = useRef<{ id: number; x: number; y: number; pointerType: string; target: Element; timer: ReturnType<typeof setTimeout> } | null>(null);
  const firedRef = useRef(false);
  const [fired, setFired] = useState(false);
  const cancel = useCallback((reason: "moved" | "released" | "aborted") => {
    const gesture = active.current;
    if (!gesture) return;
    clearTimeout(gesture.timer);
    withPointerCapture(gesture.target, gesture.id, "releasePointerCapture");
    active.current = null;
    optionsRef.current.onCancel?.(reason);
  }, []);
  const onPointerDown = useCallback((event: ReactPointerEvent) => {
    const current = optionsRef.current;
    if (current.disabled || active.current || (event.pointerType === "mouse" && event.button !== 0)) return;
    firedRef.current = false;
    setFired(false);
    const feel = resolveGestureFeel();
    const target = event.currentTarget;
    withPointerCapture(target, event.pointerId, "setPointerCapture");
    const timer = setTimeout(() => {
      const gesture = active.current;
      if (!gesture) return;
      active.current = null;
      withPointerCapture(gesture.target, gesture.id, "releasePointerCapture");
      firedRef.current = true;
      setFired(true);
      optionsRef.current.onLongPress({ x: gesture.x, y: gesture.y, pointerType: gesture.pointerType });
    }, current.delay ?? feel.longPressDelay);
    active.current = { id: event.pointerId, x: event.clientX, y: event.clientY, pointerType: event.pointerType, target, timer };
  }, []);
  const onPointerMove = useCallback((event: ReactPointerEvent) => {
    const gesture = active.current;
    const tolerance = optionsRef.current.moveTolerance ?? resolveGestureFeel().longPressMoveTolerance;
    if (gesture && (Math.hypot(event.clientX - gesture.x, event.clientY - gesture.y) > tolerance)) cancel("moved");
  }, [cancel]);
  const onPointerUp = useCallback(() => { if (active.current) cancel("released"); }, [cancel]);
  const onPointerCancel = useCallback(() => { if (active.current) cancel("aborted"); }, [cancel]);
  const onContextMenu = useCallback((event: React.MouseEvent) => { if (firedRef.current) { event.preventDefault(); firedRef.current = false; } }, []);
  const onClick = useCallback((event: React.MouseEvent) => { if (firedRef.current) { event.preventDefault(); event.stopPropagation(); firedRef.current = false; } }, []);
  useEffect(() => () => { if (active.current) { clearTimeout(active.current.timer); active.current = null; } }, []);
  return { longPressProps: { onPointerDown, onPointerMove, onPointerUp, onPointerCancel, onContextMenu, onClick }, fired };
}
