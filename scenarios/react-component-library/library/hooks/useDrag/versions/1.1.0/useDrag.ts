/** @vrooliComponentSource hooks.use-drag */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";

export interface DragStart {
  x: number;
  y: number;
}

export interface UseDragOptions {
  disabled?: boolean;
  /** Pixels moved per arrow press. */
  step?: number;
  /** Pixels moved per arrow press while Shift is held. */
  coarseStep?: number;
  onStart?: (start: DragStart) => void;
  onMove?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onEnd?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onCancel?: () => void;
  /** Receives pixel deltas, resolved from `step` / `coarseStep`. */
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

export function useDrag(onMove?: (event: globalThis.PointerEvent) => void): {
  onPointerMove: (event: PointerEvent) => void;
};
export function useDrag(options: UseDragOptions): UseDragHandlers;
export function useDrag(
  input: UseDragOptions | ((event: globalThis.PointerEvent) => void) = {},
) {
  const legacy = typeof input === "function";
  const [isDragging, setIsDragging] = useState(false);
  const active = useRef<DragStart | undefined>(undefined);
  // Handlers keep a stable identity across renders. Callers pass an inline
  // options object, so memoizing on `input` itself would rebuild every handler
  // on every render; the latest options are read through a ref instead.
  const latest = useRef(input);
  const draggingRef = useRef(false);
  useEffect(() => {
    latest.current = input;
  });

  const options = (): UseDragOptions => (typeof latest.current === "function" ? {} : latest.current);
  const isLegacy = () => typeof latest.current === "function";

  const setDragging = useCallback((next: boolean) => {
    draggingRef.current = next;
    setIsDragging(next);
  }, []);

  const onPointerDown = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const opts = options();
      if (isLegacy() || opts.disabled || event.button !== 0) return;
      active.current = { x: event.clientX, y: event.clientY };
      event.currentTarget.setPointerCapture(event.pointerId);
      setDragging(true);
      opts.onStart?.(active.current);
    },
    [setDragging],
  );

  const onPointerMove = useCallback((event: PointerEvent<HTMLElement>) => {
    const current = latest.current;
    if (typeof current === "function") {
      current(event.nativeEvent);
      return;
    }
    if (current.disabled || !active.current || !draggingRef.current) return;
    current.onMove?.(event.nativeEvent, active.current);
  }, []);

  const onPointerUp = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const opts = options();
      if (isLegacy() || !active.current) return;
      const start = active.current;
      active.current = undefined;
      setDragging(false);
      opts.onEnd?.(event.nativeEvent, start);
    },
    [setDragging],
  );

  const onPointerCancel = useCallback(
    (_event: PointerEvent<HTMLElement>) => {
      if (isLegacy() || !active.current) return;
      active.current = undefined;
      setDragging(false);
      options().onCancel?.();
    },
    [setDragging],
  );

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      const opts = options();
      if (isLegacy() || opts.disabled) return;
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
        active.current = { x: 0, y: 0 };
        setDragging(true);
        opts.onStart?.(active.current);
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

  return legacy
    ? { onPointerMove }
    : { onPointerDown, onPointerMove, onPointerUp, onPointerCancel, onKeyDown, isDragging };
}
