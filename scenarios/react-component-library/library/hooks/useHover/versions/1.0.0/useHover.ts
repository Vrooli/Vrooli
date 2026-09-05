/** @vrooliComponentSource hooks.use-hover */
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FocusEvent,
  type PointerEvent,
} from "react";

export interface UseHoverOptions {
  enterDelay?: number;
  exitDelay?: number;
  disabled?: boolean;
  relatedRefs?: Array<{ current: HTMLElement | null }>;
  onChange?: (hovered: boolean) => void;
}

export interface HoverProps {
  onPointerEnter: (event: PointerEvent<HTMLElement>) => void;
  onPointerLeave: (event: PointerEvent<HTMLElement>) => void;
  onFocus: () => void;
  onBlur: (event: FocusEvent<HTMLElement>) => void;
}

export function useHover({
  enterDelay = 80,
  exitDelay = 120,
  disabled = false,
  relatedRefs = [],
  onChange,
}: UseHoverOptions = {}) {
  const [hovered, setHovered] = useState(false);
  const [pointerType, setPointerType] = useState<
    "mouse" | "pen" | "touch" | null
  >(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hoverCapable = useRef(true);
  useEffect(() => {
    const query = window.matchMedia("(hover: hover) and (pointer: fine)");
    const update = () => {
      hoverCapable.current = query.matches;
      if (!query.matches) setHovered(false);
    };
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  const clearTimer = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
  }, []);
  const commit = useCallback(
    (next: boolean) => {
      setHovered((previous) => {
        if (previous !== next) onChange?.(next);
        return next;
      });
    },
    [onChange],
  );
  const schedule = useCallback(
    (next: boolean, delay: number) => {
      clearTimer();
      if (disabled || (!hoverCapable.current && next)) return;
      timer.current = setTimeout(
        () => {
          timer.current = null;
          commit(next);
        },
        Math.max(0, delay),
      );
    },
    [clearTimer, commit, disabled],
  );
  const isRelated = useCallback(
    (target: EventTarget | null) =>
      target instanceof Node &&
      relatedRefs.some((ref) => ref.current?.contains(target)),
    [relatedRefs],
  );
  const onPointerEnter = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      const nextPointer =
        event.pointerType === "mouse"
          ? "mouse"
          : event.pointerType === "touch"
            ? "touch"
            : "pen";
      setPointerType(nextPointer);
      if (event.pointerType === "mouse" || event.pointerType === "pen")
        schedule(true, enterDelay);
    },
    [enterDelay, schedule],
  );
  const onPointerLeave = useCallback(
    (event: PointerEvent<HTMLElement>) => {
      if (isRelated(event.relatedTarget)) return;
      schedule(false, exitDelay);
    },
    [exitDelay, isRelated, schedule],
  );
  const onFocus = useCallback(() => {
    if (!disabled) commit(true);
  }, [commit, disabled]);
  const onBlur = useCallback(
    (event: FocusEvent<HTMLElement>) => {
      if (isRelated(event.relatedTarget)) return;
      schedule(false, exitDelay);
    },
    [exitDelay, isRelated, schedule],
  );
  useEffect(() => clearTimer, [clearTimer]);
  const triggerProps = useMemo<HoverProps>(
    () => ({ onPointerEnter, onPointerLeave, onFocus, onBlur }),
    [onBlur, onFocus, onPointerEnter, onPointerLeave],
  );
  return {
    hovered,
    open: hovered,
    pointerType,
    triggerProps,
    floatingProps: triggerProps,
    close: () => schedule(false, exitDelay),
    cancel: clearTimer,
  };
}
