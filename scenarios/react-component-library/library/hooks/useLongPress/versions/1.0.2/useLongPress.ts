/**
 * @libraryId react-component-library:useLongPress
 * @displayName useLongPress
 * @description A token-backed long-press hook with movement tolerance, pointer cancellation, origin capture, and click/context-menu coordination.
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useLongPress
 * @vrooliComponentSourceSlot hooks.use-long-press */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";

export interface LongPressOrigin {
  x: number;
  y: number;
  pointerType: string;
}
export interface UseLongPressOptions {
  onLongPress: (origin: LongPressOrigin) => void;
  delay?: number;
  moveTolerance?: number;
  disabled?: boolean;
  onCancel?: (reason: "moved" | "released" | "aborted") => void;
}

interface ActiveGesture {
  id: number;
  x: number;
  y: number;
  pointerType: string;
  timer: ReturnType<typeof setTimeout>;
  detach: () => void;
}

/**
 * Detects a stationary press held for the gesture-token delay.
 *
 * The hook never captures the pointer. Capture retargets the resulting
 * `click` to the capturing element, so a long-press surface wrapping other
 * controls would swallow every ordinary click on them. The gesture instead
 * follows its pointer through window listeners for as long as it is pending,
 * which keeps movement tolerance and release working after the pointer
 * leaves the element.
 */
export function useLongPress(options: UseLongPressOptions): {
  longPressProps: Pick<
    React.DOMAttributes<HTMLElement>,
    | "onPointerDown"
    | "onPointerMove"
    | "onPointerUp"
    | "onPointerCancel"
    | "onContextMenu"
    | "onClick"
  >;
  fired: boolean;
} {
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const active = useRef<ActiveGesture | null>(null);
  const firedRef = useRef(false);
  const [fired, setFired] = useState(false);
  const end = useCallback(() => {
    const gesture = active.current;
    if (!gesture) return null;
    clearTimeout(gesture.timer);
    gesture.detach();
    active.current = null;
    return gesture;
  }, []);
  const cancel = useCallback(
    (reason: "moved" | "released" | "aborted") => {
      if (end()) optionsRef.current.onCancel?.(reason);
    },
    [end],
  );
  const track = useCallback(
    (pointerId: number, x: number, y: number) => {
      const gesture = active.current;
      if (!gesture || gesture.id !== pointerId) return;
      const tolerance =
        optionsRef.current.moveTolerance ?? resolveGestureFeel().longPressMoveTolerance;
      if (Math.hypot(x - gesture.x, y - gesture.y) > tolerance) cancel("moved");
    },
    [cancel],
  );
  const release = useCallback(
    (pointerId: number, reason: "released" | "aborted") => {
      if (active.current?.id === pointerId) cancel(reason);
    },
    [cancel],
  );
  const onPointerDown = useCallback(
    (event: ReactPointerEvent) => {
      const current = optionsRef.current;
      if (
        current.disabled ||
        active.current ||
        (event.pointerType === "mouse" && event.button !== 0)
      )
        return;
      firedRef.current = false;
      setFired(false);
      const pointerId = event.pointerId;
      const onWindowMove = (move: PointerEvent) =>
        track(move.pointerId, move.clientX, move.clientY);
      const onWindowUp = (up: PointerEvent) => release(up.pointerId, "released");
      const onWindowCancel = (abort: PointerEvent) => release(abort.pointerId, "aborted");
      window.addEventListener("pointermove", onWindowMove);
      window.addEventListener("pointerup", onWindowUp);
      window.addEventListener("pointercancel", onWindowCancel);
      const timer = setTimeout(() => {
        const gesture = end();
        if (!gesture) return;
        firedRef.current = true;
        setFired(true);
        optionsRef.current.onLongPress({
          x: gesture.x,
          y: gesture.y,
          pointerType: gesture.pointerType,
        });
      }, current.delay ?? resolveGestureFeel().longPressDelay);
      active.current = {
        id: pointerId,
        x: event.clientX,
        y: event.clientY,
        pointerType: event.pointerType,
        timer,
        detach: () => {
          window.removeEventListener("pointermove", onWindowMove);
          window.removeEventListener("pointerup", onWindowUp);
          window.removeEventListener("pointercancel", onWindowCancel);
        },
      };
    },
    [end, release, track],
  );
  const onPointerMove = useCallback(
    (event: ReactPointerEvent) => track(event.pointerId, event.clientX, event.clientY),
    [track],
  );
  const onPointerUp = useCallback(
    (event: ReactPointerEvent) => release(event.pointerId, "released"),
    [release],
  );
  const onPointerCancel = useCallback(
    (event: ReactPointerEvent) => release(event.pointerId, "aborted"),
    [release],
  );
  const onContextMenu = useCallback((event: React.MouseEvent) => {
    if (firedRef.current) {
      event.preventDefault();
      firedRef.current = false;
    }
  }, []);
  const onClick = useCallback((event: React.MouseEvent) => {
    if (firedRef.current) {
      event.preventDefault();
      event.stopPropagation();
      firedRef.current = false;
    }
  }, []);
  useEffect(() => () => void end(), [end]);
  return {
    longPressProps: {
      onPointerDown,
      onPointerMove,
      onPointerUp,
      onPointerCancel,
      onContextMenu,
      onClick,
    },
    fired,
  };
}
