/**
 * @libraryId react-component-library:useSwipeGesture
 * @displayName useSwipeGesture
 * @description Staged directional drag with axis locking, resistance, and velocity-aware release, reported in pixels rather than a clamped ratio.
 * @version 1.1.3
 * @tags ["runtime","gesture","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-swipe-gesture */
import { useCallback, useEffect, useRef, type PointerEvent as ReactPointerEvent } from "react";

import { edgeSign, type PhysicalEdge } from "@vrooli/react-component-library/GestureDirection/1";

/** How a finished gesture resolved. */
export type SwipeOutcome =
  /** Past a threshold, and the consumer should perform the armed action. */
  | "commit"
  /** Past a threshold, and the consumer should hold the revealed position. */
  | "rest"
  /** Short of every threshold; return to the closed position. */
  | "return"
  /** The browser or the consumer took the gesture away; change nothing. */
  | "abort";

/** What a release should do once a threshold has been passed. */
export type SwipeReleaseMode = "commit" | "rest";

export interface SwipeGestureFrame {
  /** Travel toward the intended direction, in pixels, clamped at zero. */
  distance: number;
  /** Travel after resistance, i.e. what should actually be rendered. */
  offset: number;
  /** Signed CSS translation for `offset`, ready for `translateX()`. */
  translate: number;
  /** 0 when nothing is armed, otherwise the 1-based index into `stages`. */
  stage: number;
}

export interface SwipeGestureRelease extends SwipeGestureFrame {
  outcome: SwipeOutcome;
  /** Pixels per millisecond over the tail of the gesture. Negative travelling back. */
  velocity: number;
  /** Where the gesture began, so an abort can restore that position rather than assume closed. */
  startedOpen: boolean;
}

export interface UseSwipeGestureOptions {
  /** The physical direction travel must run for this gesture to arm. */
  direction: PhysicalEdge;
  /**
   * Ascending pixel distances at which each successive action arms. The last
   * entry also sets where resistance begins.
   */
  stages: readonly number[];
  /** What a past-threshold release means. */
  releaseMode?: SwipeReleaseMode;
  /** Movement, in pixels, before the gesture commits to an axis. */
  axisSlop?: number;
  /** Pixels per millisecond that counts as a flick regardless of distance. */
  flickVelocity?: number;
  /** Fraction of overtravel that still moves the surface. 0 pins it hard. */
  resistance?: number;
  /** Where this gesture starts from, for resuming a rested-open surface. */
  startOffset?: () => number;
  disabled?: boolean;
  onStart?: () => void;
  /** Fires on every accepted pointer move. Write to the DOM here, never setState. */
  onMove?: (frame: SwipeGestureFrame) => void;
  /** Fires only when the armed stage changes, so it is safe to setState here. */
  onStageChange?: (stage: number, previous: number) => void;
  onRelease?: (release: SwipeGestureRelease) => void;
}

const DEFAULT_AXIS_SLOP = 8;
const DEFAULT_FLICK_VELOCITY = 0.5;
const DEFAULT_RESISTANCE = 0.32;
/** Velocity is measured over the tail of the gesture, not its whole life. */
const VELOCITY_WINDOW_MS = 80;

type Axis = "undecided" | "inline" | "block";

interface Sample {
  position: number;
  time: number;
}

interface ActiveGesture {
  pointerId: number;
  target: Element;
  startX: number;
  startY: number;
  base: number;
  axis: Axis;
  stage: number;
  frame: SwipeGestureFrame;
  samples: Sample[];
}

/**
 * Pointer capture keeps a drag attached to the element it began on once the
 * finger leaves that element. It is absent in jsdom and older WebKit, where an
 * unguarded call throws on the first touch rather than degrading, so both the
 * set and the release are guarded.
 */
const withPointerCapture = (
  target: Element,
  pointerId: number,
  method: "setPointerCapture" | "releasePointerCapture",
) => {
  const capture = (target as Partial<Element>)[method];
  if (typeof capture !== "function") return;
  try {
    capture.call(target, pointerId);
  } catch {
    // A pointer that was never captured, or is already released, is not an
    // error the gesture needs to react to.
  }
};

const stageFor = (stages: readonly number[], distance: number) => {
  let armed = 0;
  for (let index = 0; index < stages.length; index += 1) {
    const threshold = stages[index];
    if (threshold !== undefined && distance >= threshold) armed = index + 1;
  }
  return armed;
};

/**
 * A staged, axis-locked drag along one direction.
 *
 * Two properties matter more than the rest and are worth stating explicitly:
 *
 * Movement is reported in **pixels**, through a callback, and never through
 * React state. The predecessor hook reported `distance / threshold` clamped to
 * one, which cannot drive a surface that tracks a finger; and holding
 * per-frame state in React re-renders every subscriber sixty times a second.
 *
 * A `pointercancel` **aborts**. Wiring cancel to the same path as `pointerup`
 * means a browser-initiated cancel past a threshold performs the action the
 * user just stopped asking for.
 */
export function useSwipeGesture(options: UseSwipeGestureOptions) {
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const active = useRef<ActiveGesture | null>(null);
  const detach = useRef<(() => void) | null>(null);

  const teardown = useCallback(() => {
    detach.current?.();
    detach.current = null;
    const gesture = active.current;
    if (gesture) {
      withPointerCapture(gesture.target, gesture.pointerId, "releasePointerCapture");
    }
    active.current = null;
  }, []);

  const finish = useCallback(
    (outcome: SwipeOutcome, velocity: number) => {
      const gesture = active.current;
      if (!gesture) return;
      const frame = gesture.frame;
      const startedOpen = gesture.base > 0;
      teardown();
      optionsRef.current.onRelease?.({ ...frame, outcome, velocity, startedOpen });
    },
    [teardown],
  );

  /** Consumers cancel from outside when something else claims the interaction. */
  const cancel = useCallback(() => {
    if (!active.current) return;
    finish("abort", 0);
  }, [finish]);

  useEffect(() => teardown, [teardown]);

  const onPointerDown = useCallback(
    (event: ReactPointerEvent) => {
      const settings = optionsRef.current;
      if (settings.disabled || active.current) return;
      // Secondary buttons open menus; they are not drags.
      if (event.button !== 0 && event.pointerType === "mouse") return;

      const target = event.currentTarget;
      const base = settings.startOffset?.() ?? 0;
      active.current = {
        pointerId: event.pointerId,
        target,
        startX: event.clientX,
        startY: event.clientY,
        base,
        axis: "undecided",
        stage: stageFor(settings.stages, base),
        frame: {
          distance: base,
          offset: base,
          translate: base * edgeSign(settings.direction),
          stage: 0,
        },
        // `event.nativeEvent`, not the synthetic event: React falls back to
        // `Date.now()` whenever a native timeStamp is falsy, and epoch
        // milliseconds are not comparable with the time-origin milliseconds
        // that every later window event reports. Mixing the two makes a slow
        // drag read as an instantaneous flick.
        samples: [{ position: event.clientX, time: event.nativeEvent.timeStamp }],
      };
      const handleMove = (native: PointerEvent) => {
        const gesture = active.current;
        const current = optionsRef.current;
        if (!gesture || native.pointerId !== gesture.pointerId) return;

        const dx = native.clientX - gesture.startX;
        const dy = native.clientY - gesture.startY;

        if (gesture.axis === "undecided") {
          if (
            Math.abs(dx) < (current.axisSlop ?? DEFAULT_AXIS_SLOP) &&
            Math.abs(dy) < (current.axisSlop ?? DEFAULT_AXIS_SLOP)
          ) {
            return;
          }
          // A drag that starts vertically belongs to the scroll container, and
          // the decision is final: re-deciding mid-gesture is what makes a
          // surface feel like it is fighting the finger.
          gesture.axis = Math.abs(dx) > Math.abs(dy) ? "inline" : "block";
          if (gesture.axis === "block") {
            finish("abort", 0);
            return;
          }
          // Capture only once the drag is ours. Capturing at pointerdown would
          // retarget the click that a plain tap still owes its button, so a
          // gesture surface wrapping real controls would swallow taps on them.
          withPointerCapture(gesture.target, gesture.pointerId, "setPointerCapture");
          current.onStart?.();
        }
        if (gesture.axis !== "inline") return;

        // Claim the gesture so the browser does not also scroll or navigate.
        // The listener is registered non-passive precisely so this is allowed.
        if (native.cancelable) native.preventDefault();

        gesture.samples.push({ position: native.clientX, time: native.timeStamp });
        while (gesture.samples.length > 2) {
          const oldest = gesture.samples[0];
          if (!oldest || native.timeStamp - oldest.time <= VELOCITY_WINDOW_MS) break;
          gesture.samples.shift();
        }

        const sign = edgeSign(current.direction);
        const distance = Math.max(0, (dx + gesture.base * sign) * sign);
        const ceiling = current.stages[current.stages.length - 1] ?? 0;
        const resistance = current.resistance ?? DEFAULT_RESISTANCE;
        const offset = distance > ceiling ? ceiling + (distance - ceiling) * resistance : distance;
        const stage = stageFor(current.stages, distance);

        gesture.frame = { distance, offset, translate: offset * sign, stage };
        current.onMove?.(gesture.frame);

        if (stage !== gesture.stage) {
          const previous = gesture.stage;
          gesture.stage = stage;
          current.onStageChange?.(stage, previous);
        }
      };

      const handleUp = (native: PointerEvent) => {
        const gesture = active.current;
        const current = optionsRef.current;
        if (!gesture || native.pointerId !== gesture.pointerId) return;
        if (gesture.axis !== "inline") {
          finish("abort", 0);
          return;
        }

        const oldest = gesture.samples[0] ?? {
          position: gesture.startX,
          time: native.timeStamp,
        };
        const elapsed = Math.max(1, native.timeStamp - oldest.time);
        const sign = edgeSign(current.direction);
        const travelled = (native.clientX - oldest.position) * sign;
        const velocity = travelled / elapsed;

        // The release test is relative to where the gesture began, not to the
        // origin. Reading `stage` alone is right opening and wrong closing: a
        // surface that starts fully revealed is already past every threshold,
        // so it stays past them until the finger has travelled almost the whole
        // way back, and a half-hearted drag springs open again. The gesture is
        // measured by how far it moved from its own starting point instead, so
        // closing asks for the same distance as opening did.
        const threshold = current.stages[0] ?? 0;
        const limit = current.flickVelocity ?? DEFAULT_FLICK_VELOCITY;
        const movement = gesture.frame.distance - gesture.base;
        const startedOpen = gesture.base > 0;

        let settleOpen: boolean;
        if (velocity >= limit) settleOpen = true;
        else if (velocity <= -limit) settleOpen = false;
        else if (startedOpen) settleOpen = -movement < threshold;
        else settleOpen = gesture.frame.stage > 0;

        if (!settleOpen) {
          finish("return", velocity);
          return;
        }
        finish((current.releaseMode ?? "rest") === "commit" ? "commit" : "rest", velocity);
      };

      const handleCancel = (native: PointerEvent) => {
        const gesture = active.current;
        if (!gesture || native.pointerId !== gesture.pointerId) return;
        finish("abort", 0);
      };

      // Window listeners rather than element handlers: a finger that leaves the
      // row mid-drag must keep driving it, and a pointerup outside the element
      // must still end the gesture rather than stranding it mid-travel.
      const view = target.ownerDocument.defaultView ?? window;
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
