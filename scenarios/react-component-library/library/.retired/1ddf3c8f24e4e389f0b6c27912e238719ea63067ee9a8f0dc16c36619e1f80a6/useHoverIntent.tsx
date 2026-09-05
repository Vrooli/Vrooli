/**
 * @libraryId react-component-library:useHoverIntent
 * @displayName useHoverIntent
 * @description Safe-polygon hover intent for fine pointers.
 * @version 0.1.1
 * @tags ["runtime","gesture","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-hover-intent */
import { useCallback, useEffect, useRef } from "react";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";

export interface HoverPoint {
  x: number;
  y: number;
}
export interface HoverRect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}
export interface UseHoverIntentOptions {
  childRect?: () => HoverRect | null;
  onOpen: () => void;
  onClose: () => void;
  openDelay?: number;
  closeDelay?: number;
  fuse?: number;
}

const insideTriangle = (point: HoverPoint, a: HoverPoint, b: HoverPoint, c: HoverPoint) => {
  const sign = (p1: HoverPoint, p2: HoverPoint, p3: HoverPoint) =>
    (p1.x - p3.x) * (p2.y - p3.y) - (p2.x - p3.x) * (p1.y - p3.y);
  const d1 = sign(point, a, b);
  const d2 = sign(point, b, c);
  const d3 = sign(point, c, a);
  return !((d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0));
};

export function useHoverIntent(options: UseHoverIntentOptions) {
  const latest = useRef(options);
  latest.current = options;
  const previous = useRef<HoverPoint | null>(null);
  const opened = useRef(false);
  const openTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clear = useCallback(() => {
    if (openTimer.current) clearTimeout(openTimer.current);
    if (closeTimer.current) clearTimeout(closeTimer.current);
    openTimer.current = null;
    closeTimer.current = null;
  }, []);
  useEffect(() => clear, [clear]);
  const canHover = () =>
    typeof window === "undefined" ||
    window.matchMedia?.("(hover: hover) and (pointer: fine)").matches === true;
  const open = useCallback(() => {
    if (!canHover()) return;
    clear();
    openTimer.current = setTimeout(() => {
      opened.current = true;
      latest.current.onOpen();
    }, latest.current.openDelay ?? resolveGestureFeel().hoverOpenDelay);
  }, [clear]);
  const close = useCallback(() => {
    clear();
    closeTimer.current = setTimeout(() => {
      if (opened.current) {
        opened.current = false;
        latest.current.onClose();
      }
    }, latest.current.closeDelay ?? resolveGestureFeel().hoverCloseDelay);
  }, [clear]);
  const onPointerEnter = useCallback(
    (event: { clientX: number; clientY: number; pointerType?: string }) => {
      if (event.pointerType && event.pointerType !== "mouse") return;
      previous.current = { x: event.clientX, y: event.clientY };
      open();
    },
    [open],
  );
  const onPointerMove = useCallback(
    (event: { clientX: number; clientY: number }) => {
      if (!opened.current || !previous.current) return;
      const rect = latest.current.childRect?.();
      const point = { x: event.clientX, y: event.clientY };
      const last = previous.current;
      previous.current = point;
      if (!rect) return close();
      const towardChild = insideTriangle(
        point,
        last,
        { x: rect.left, y: rect.top },
        { x: rect.left, y: rect.bottom },
      );
      if (!towardChild) close();
      else {
        clear();
        closeTimer.current = setTimeout(
          close,
          latest.current.fuse ?? resolveGestureFeel().safePolygonFuse,
        );
      }
    },
    [clear, close],
  );
  const onPointerLeave = useCallback(() => close(), [close]);
  const onChildEnter = useCallback(() => clear(), [clear]);
  return { onPointerEnter, onPointerMove, onPointerLeave, onChildEnter, cancel: clear };
}

export { insideTriangle };
export default useHoverIntent;
