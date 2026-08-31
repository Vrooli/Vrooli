/** @vrooliComponentSource hooks.use-drag */
import {
  useCallback,
  useMemo,
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
  onStart?: (start: DragStart) => void;
  onMove?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onEnd?: (event: globalThis.PointerEvent, start: DragStart) => void;
  onCancel?: () => void;
  onKeyboardMove?: (dx: number, dy: number) => void;
  onKeyboardEnd?: () => void;
}

export function useDrag(onMove?: (event: globalThis.PointerEvent) => void): {
  onPointerMove: (event: PointerEvent) => void;
};
export function useDrag(options: UseDragOptions): {
  onPointerDown: (event: PointerEvent<HTMLElement>) => void;
  onPointerMove: (event: PointerEvent<HTMLElement>) => void;
  onPointerUp: (event: PointerEvent<HTMLElement>) => void;
  onPointerCancel: (event: PointerEvent<HTMLElement>) => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
  isDragging: boolean;
};
export function useDrag(
  input: UseDragOptions | ((event: globalThis.PointerEvent) => void) = {},
) {
  const legacy = typeof input === "function";
  const options = useMemo(
    () => (typeof input === "function" ? {} : input),
    [input],
  );
  const [isDragging, setIsDragging] = useState(false);
  const active = useRef<DragStart>();
  const onPointerDown = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      if (legacy || options.disabled || event.button !== 0) return;
      active.current = { x: event.clientX, y: event.clientY };
      event.currentTarget.setPointerCapture(event.pointerId);
      setIsDragging(true);
      const start = active.current;
      options.onStart?.(start);
    },
    [legacy, options],
  );
  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      if (typeof input === "function") {
        input(event.nativeEvent);
        return;
      }
      if (active.current && isDragging)
        options.onMove?.(event.nativeEvent, active.current);
    },
    [input, isDragging, options],
  );
  const onPointerUp = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      if (legacy || !active.current) return;
      options.onEnd?.(event.nativeEvent, active.current);
      active.current = undefined;
      setIsDragging(false);
    },
    [legacy, options],
  );
  const onPointerCancel = useCallback(
    (_event: PointerEvent<HTMLElement>) => {
      if (legacy || !active.current) return;
      active.current = undefined;
      setIsDragging(false);
      options.onCancel?.();
    },
    [legacy, options],
  );
  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      if (legacy || options.disabled) return;
      if (event.key === "Escape" && isDragging) {
        event.preventDefault();
        active.current = undefined;
        setIsDragging(false);
        options.onCancel?.();
        return;
      }
      if (
        (event.key === " " || event.key === "Space" || event.key === "Enter") &&
        !isDragging
      ) {
        event.preventDefault();
        active.current = { x: 0, y: 0 };
        setIsDragging(true);
        options.onStart?.(active.current);
        return;
      }
      if (isDragging) {
        const step = event.shiftKey ? 10 : 1;
        const moves: Record<string, [number, number]> = {
          ArrowUp: [0, -step],
          ArrowDown: [0, step],
          ArrowLeft: [-step, 0],
          ArrowRight: [step, 0],
        };
        const move = moves[event.key];
        if (move) {
          event.preventDefault();
          options.onKeyboardMove?.(move[0], move[1]);
        }
        if (
          (event.key === "Enter" ||
            event.key === " " ||
            event.key === "Space") &&
          active.current
        ) {
          event.preventDefault();
          setIsDragging(false);
          active.current = undefined;
          options.onKeyboardEnd?.();
        }
      }
    },
    [isDragging, legacy, options],
  );
  return legacy
    ? { onPointerMove }
    : {
        onPointerDown,
        onPointerMove,
        onPointerUp,
        onPointerCancel,
        onKeyDown,
        isDragging,
      };
}
