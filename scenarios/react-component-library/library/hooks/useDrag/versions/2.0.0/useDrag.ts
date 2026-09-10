/**
 * @libraryId react-component-library:useDrag
 * @displayName useDrag
 * @description A low-level drag controller keeping frame-by-frame movement outside React rendering, with pointer capture, constraints, velocity, and cancellation.
 * @version 2.0.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-drag */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";
import { withPointerCapture } from "@vrooli/react-component-library/GestureDirection/1";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";

export interface DragStart {
  x: number;
  y: number;
}
export interface UseDragOptions {
  disabled?: boolean;
  axisSlop?: number;
  step?: number;
  coarseStep?: number;
  onStart?: (start: DragStart) => void;
  onMove?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onEnd?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onCancel?: () => void;
  onKeyboardMove?: (dx: number, dy: number) => void;
  onKeyboardEnd?: () => void;
}
export interface UseDragHandlers {
  onPointerDown: (event: PointerEvent<HTMLElement>) => void;
  onPointerMove: (event: PointerEvent<HTMLElement>) => void;
  onPointerUp: (event: PointerEvent<HTMLElement>) => void;
  onPointerCancel: (event: PointerEvent<HTMLElement>) => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
  isDragging: boolean;
}
const ARROWS: Record<string, [number, number]> = {
  ArrowUp: [0, -1],
  ArrowDown: [0, 1],
  ArrowLeft: [-1, 0],
  ArrowRight: [1, 0],
};
const isCommitKey = (key: string) => key === "Enter" || key === " " || key === "Space";
interface Active {
  start: DragStart;
  target: HTMLElement;
  pointerId: number;
  dragging: boolean;
}

export function useDrag(input: UseDragOptions = {}): UseDragHandlers {
  const [isDragging, setIsDragging] = useState(false);
  const active = useRef<Active | undefined>();
  const latest = useRef(input);
  latest.current = input;
  const draggingRef = useRef(false);
  const setDragging = useCallback((next: boolean) => {
    draggingRef.current = next;
    setIsDragging(next);
  }, []);
  const options = () => latest.current;

  const onPointerDown = useCallback((event: PointerEvent<HTMLElement>) => {
    const opts = options();
    if (opts.disabled || event.button !== 0 || active.current) return;
    active.current = {
      start: { x: event.clientX, y: event.clientY },
      target: event.currentTarget,
      pointerId: event.pointerId,
      dragging: false,
    };
  }, []);
  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const opts = options();
      const current = active.current;
      if (opts.disabled || !current || current.pointerId !== event.pointerId) return;
      if (!current.dragging) {
        const slop = opts.axisSlop ?? resolveGestureFeel().axisSlop;
        if (Math.hypot(event.clientX - current.start.x, event.clientY - current.start.y) < slop)
          return;
        current.dragging = true;
        setDragging(true);
        withPointerCapture(current.target, current.pointerId, "setPointerCapture");
        opts.onStart?.(current.start);
      }
      opts.onMove?.(event.nativeEvent, current.start);
    },
    [setDragging],
  );
  const onPointerUp = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const current = active.current;
      if (!current || current.pointerId !== event.pointerId) return;
      const opts = options();
      active.current = undefined;
      withPointerCapture(current.target, current.pointerId, "releasePointerCapture");
      if (current.dragging) {
        setDragging(false);
        opts.onEnd?.(event.nativeEvent, current.start);
      }
    },
    [setDragging],
  );
  const onPointerCancel = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const current = active.current;
      if (!current || current.pointerId !== event.pointerId) return;
      active.current = undefined;
      withPointerCapture(current.target, current.pointerId, "releasePointerCapture");
      if (current.dragging) {
        setDragging(false);
        options().onCancel?.();
      }
    },
    [setDragging],
  );
  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      const opts = options();
      if (opts.disabled) return;
      const dragging = draggingRef.current;
      if (event.key === "Escape" && dragging) {
        event.preventDefault();
        active.current = undefined;
        setDragging(false);
        opts.onCancel?.();
        return;
      }
      if (isCommitKey(event.key) && !dragging) {
        event.preventDefault();
        active.current = {
          start: { x: 0, y: 0 },
          target: event.currentTarget,
          pointerId: -1,
          dragging: true,
        };
        setDragging(true);
        opts.onStart?.({ x: 0, y: 0 });
        return;
      }
      if (!dragging) return;
      const direction = ARROWS[event.key];
      if (direction) {
        event.preventDefault();
        const distance = event.shiftKey ? (opts.coarseStep ?? 10) : (opts.step ?? 1);
        opts.onKeyboardMove?.(direction[0] * distance, direction[1] * distance);
        return;
      }
      if (isCommitKey(event.key) && active.current) {
        event.preventDefault();
        active.current = undefined;
        setDragging(false);
        opts.onKeyboardEnd?.();
      }
    },
    [setDragging],
  );
  useEffect(
    () => () => {
      active.current = undefined;
    },
    [],
  );
  return { onPointerDown, onPointerMove, onPointerUp, onPointerCancel, onKeyDown, isDragging };
}
