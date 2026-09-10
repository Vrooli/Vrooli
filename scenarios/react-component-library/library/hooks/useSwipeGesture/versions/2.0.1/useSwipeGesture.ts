/**
 * @libraryId react-component-library:useSwipeGesture
 * @displayName Use Swipe Gesture
 * @description Tracks staged pointer movement and reports deterministic swipe outcomes without rerendering every frame.
 * @version 2.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-swipe-gesture */
import { useCallback, useEffect, useRef, type PointerEvent as ReactPointerEvent } from "react";
import {
  axisOf,
  edgeSign,
  coordinateOf,
  withPointerCapture,
  type Axis,
  type PhysicalEdge,
} from "@vrooli/react-component-library/GestureDirection/1";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";

export type SwipeOutcome = "commit" | "rest" | "return" | "abort";
export type SwipeReleaseMode = "commit" | "rest";

export interface SwipeGestureFrame {
  distance: number;
  offset: number;
  translate: number;
  stage: number;
}
export interface SwipeGestureRelease extends SwipeGestureFrame {
  outcome: SwipeOutcome;
  velocity: number;
  startedOpen: boolean;
}
export interface UseSwipeGestureOptions {
  direction: PhysicalEdge;
  stages: readonly number[];
  releaseMode?: SwipeReleaseMode;
  axisSlop?: number;
  flickVelocity?: number;
  resistance?: number;
  startOffset?: () => number;
  disabled?: boolean;
  onStart?: () => void;
  onMove?: (frame: SwipeGestureFrame) => void;
  onStageChange?: (stage: number, previous: number) => void;
  onRelease?: (release: SwipeGestureRelease) => void;
  onSettle?: (release: SwipeGestureRelease) => void;
}

type GestureAxis = Axis | "undecided";
interface Sample {
  position: number;
  time: number;
}
interface ActiveGesture {
  pointerId: number;
  target: Element;
  startPrimary: number;
  startOrthogonal: number;
  base: number;
  axis: GestureAxis;
  stage: number;
  frame: SwipeGestureFrame;
  samples: Sample[];
}

const stageFor = (stages: readonly number[], distance: number) => {
  let armed = 0;
  for (let index = 0; index < stages.length; index += 1) {
    const threshold = stages[index];
    if (threshold !== undefined && distance >= threshold) armed = index + 1;
  }
  return armed;
};

/**
 * A staged, axis-locked drag. Movement is reported in pixels through a
 * callback, never through React state, because a fraction cannot drive a
 * surface that tracks a finger and per-frame state re-renders subscribers.
 * A pointercancel aborts and restores the starting position; it never performs
 * an armed action because the browser taking a gesture away is not a request.
 */
export function useSwipeGesture(options: UseSwipeGestureOptions) {
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const active = useRef<ActiveGesture | null>(null);
  const detach = useRef<(() => void) | null>(null);
  const finish = useCallback((outcome: SwipeOutcome, velocity: number) => {
    const gesture = active.current;
    if (!gesture) return;
    const release = { ...gesture.frame, outcome, velocity, startedOpen: gesture.base > 0 };
    detach.current?.();
    detach.current = null;
    withPointerCapture(gesture.target, gesture.pointerId, "releasePointerCapture");
    active.current = null;
    optionsRef.current.onRelease?.(release);
    optionsRef.current.onSettle?.(release);
  }, []);
  const cancel = useCallback(() => finish("abort", 0), [finish]);
  useEffect(
    () => () => {
      detach.current?.();
    },
    [],
  );

  const onPointerDown = useCallback(
    (event: ReactPointerEvent) => {
      const current = optionsRef.current;
      if (current.disabled || active.current) return;
      if (event.button !== 0 && event.pointerType === "mouse") return;
      const primaryKey = coordinateOf(current.direction);
      const primary = event[primaryKey];
      const orthogonal = primaryKey === "clientX" ? event.clientY : event.clientX;
      const base = current.startOffset?.() ?? 0;
      const axis = axisOf(current.direction);
      active.current = {
        pointerId: event.pointerId,
        target: event.currentTarget,
        startPrimary: primary,
        startOrthogonal: orthogonal,
        base,
        axis: "undecided",
        stage: stageFor(current.stages, base),
        frame: {
          distance: base,
          offset: base,
          translate: base * edgeSign(current.direction),
          stage: 0,
        },
        samples: [{ position: primary, time: event.nativeEvent.timeStamp }],
      };
      const view = event.currentTarget.ownerDocument.defaultView ?? window;
      const handleMove = (native: PointerEvent) => {
        const gesture = active.current;
        const settings = optionsRef.current;
        if (!gesture || native.pointerId !== gesture.pointerId) return;
        const nextPrimary = primaryKey === "clientX" ? native.clientX : native.clientY;
        const nextOrthogonal = primaryKey === "clientX" ? native.clientY : native.clientX;
        const primaryDelta = nextPrimary - gesture.startPrimary;
        const orthogonalDelta = nextOrthogonal - gesture.startOrthogonal;
        if (gesture.axis === "undecided") {
          const slop = settings.axisSlop ?? resolveGestureFeel().axisSlop;
          if (Math.abs(primaryDelta) < slop && Math.abs(orthogonalDelta) < slop) return;
          if (Math.abs(orthogonalDelta) > Math.abs(primaryDelta)) {
            finish("abort", 0);
            return;
          }
          gesture.axis = axis;
          withPointerCapture(gesture.target, gesture.pointerId, "setPointerCapture");
          settings.onStart?.();
        }
        if (gesture.axis !== axis) return;
        if (native.cancelable) native.preventDefault();
        gesture.samples.push({ position: nextPrimary, time: native.timeStamp });
        while (gesture.samples.length > 2 && native.timeStamp - gesture.samples[0]!.time > 80)
          gesture.samples.shift();
        const sign = edgeSign(settings.direction);
        const distance = Math.max(0, (primaryDelta + gesture.base * sign) * sign);
        const ceiling = settings.stages.at(-1) ?? 0;
        const resistance = settings.resistance ?? resolveGestureFeel().resistance;
        const offset = distance > ceiling ? ceiling + (distance - ceiling) * resistance : distance;
        const stage = stageFor(settings.stages, distance);
        gesture.frame = { distance, offset, translate: offset * sign, stage };
        settings.onMove?.(gesture.frame);
        if (stage !== gesture.stage) {
          const previous = gesture.stage;
          gesture.stage = stage;
          settings.onStageChange?.(stage, previous);
        }
      };
      const handleUp = (native: PointerEvent) => {
        const gesture = active.current;
        const settings = optionsRef.current;
        if (!gesture || native.pointerId !== gesture.pointerId) return;
        if (gesture.axis !== axis) {
          finish("abort", 0);
          return;
        }
        const nextPrimary = primaryKey === "clientX" ? native.clientX : native.clientY;
        const oldest = gesture.samples[0] ?? {
          position: gesture.startPrimary,
          time: native.timeStamp,
        };
        const elapsed = Math.max(1, native.timeStamp - oldest.time);
        const sign = edgeSign(settings.direction);
        const velocity = ((nextPrimary - oldest.position) * sign) / elapsed;
        const movement = gesture.frame.distance - gesture.base;
        const threshold = settings.stages[0] ?? 0;
        const flick = settings.flickVelocity ?? resolveGestureFeel().flickVelocity;
        const startedOpen = gesture.base > 0;
        const settleOpen =
          velocity >= flick
            ? true
            : velocity <= -flick
              ? false
              : startedOpen
                ? -movement < threshold
                : gesture.frame.stage > 0;
        if (!settleOpen) {
          finish("return", velocity);
          return;
        }
        finish((settings.releaseMode ?? "rest") === "commit" ? "commit" : "rest", velocity);
      };
      const handleCancel = (native: PointerEvent) => {
        if (active.current?.pointerId === native.pointerId) finish("abort", 0);
      };
      view.addEventListener("pointermove", handleMove, { passive: false });
      view.addEventListener("pointerup", handleUp);
      view.addEventListener("pointercancel", handleCancel);
      detach.current = () => {
        view.removeEventListener("pointermove", handleMove);
        view.removeEventListener("pointerup", handleUp);
        view.removeEventListener("pointercancel", handleCancel);
      };
    },
    [finish],
  );
  return { onPointerDown, cancel };
}
