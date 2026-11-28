import { useCallback, useEffect, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent, PointerEvent as ReactPointerEvent } from "react";
import { GestureTokens } from "@vrooli/react-component-library/GestureTokens";

/**
 * Tap to step once; hold to scrub, faster the longer the hold lasts — the
 * fast-forward of a video player, for anything that steps through a list.
 *
 * The rate is a function of time, not of frames: each frame moves however
 * many items the elapsed time owes. A slow render therefore skips items
 * instead of slowing the scrub, so a 16× hold feels 16× on a phone that
 * renders each item in 80 ms as on one that renders it in 8.
 *
 * Release is watched on the window, not the button. Reaching an end disables
 * the button under the finger, and a disabled control stops receiving its own
 * events — the release must still land.
 */

export interface ScrubTier {
  /** Milliseconds of scrubbing before this tier engages. */
  readonly after: number;
  /** Items stepped per second while it is engaged. */
  readonly perSecond: number;
  /** The multiplier shown to the reader. */
  readonly speed: number;
}

/** 2× → 16× over four and a half seconds; the first tier is a readable crawl. */
export const SCRUB_TIERS: readonly ScrubTier[] = [
  { after: 0, perSecond: 3, speed: 2 },
  { after: 1500, perSecond: 6, speed: 4 },
  { after: 3000, perSecond: 12, speed: 8 },
  { after: 4500, perSecond: 24, speed: 16 },
];

export type ScrubPhase = "idle" | "pressed" | "scrubbing";

export interface ScrubState {
  /** `pressed` is a hold that has not yet become a scrub. */
  readonly phase: ScrubPhase;
  /** Index into the tiers; meaningful while scrubbing. */
  readonly tier: number;
  /** The scrub ran out of items. It stays engaged until release. */
  readonly atEdge: boolean;
}

const IDLE: ScrubState = { phase: "idle", tier: 0, atEdge: false };

/** A click arriving this soon after a scrub ended belongs to that scrub. */
const CLICK_AFTER_SCRUB_MS = 600;

export interface UseHoldToScrubOptions {
  /** Move `count` items; returns how many it actually moved (fewer at an end). */
  readonly step: (count: number) => number;
  /** Whether a step is possible at all; a press is ignored otherwise. */
  readonly canStep: boolean;
  readonly holdDelay?: number;
  readonly tiers?: readonly ScrubTier[];
  /** A scrub ended (released, cancelled, or the page lost focus), with where it stood. */
  readonly onScrubEnd?: (final: ScrubState) => void;
}

interface Gesture {
  source: "pointer" | "key";
  phase: "pressed" | "scrubbing";
  pressedAt: number;
  scrubbedAt: number;
  lastFrame: number;
  owed: number;
  tier: number;
  atEdge: boolean;
}

function buzz(ms: number): void {
  // Android only; iOS exposes no web haptics, and the visual cue carries it there.
  if (typeof navigator === "undefined" || !("vibrate" in navigator)) return;
  try { navigator.vibrate(ms); } catch { /* blocked without a user gesture */ }
}

function tierAt(tiers: readonly ScrubTier[], elapsed: number): number {
  let index = 0;
  tiers.forEach((tier, candidate) => { if (elapsed >= tier.after) index = candidate; });
  return index;
}

export function useHoldToScrub({
  step,
  canStep,
  holdDelay = GestureTokens.longPressDelay,
  tiers = SCRUB_TIERS,
  onScrubEnd,
}: UseHoldToScrubOptions) {
  const [state, setState] = useState<ScrubState>(IDLE);
  const gestureRef = useRef<Gesture | null>(null);
  const frameRef = useRef<number | null>(null);
  const detachRef = useRef<(() => void) | null>(null);
  const swallowClickUntil = useRef(0);
  // The loop runs across renders; it always calls the latest of these.
  const latest = useRef({ step, canStep, holdDelay, tiers, onScrubEnd });
  latest.current = { step, canStep, holdDelay, tiers, onScrubEnd };

  const publish = useCallback((gesture: Gesture) => {
    setState((previous) => {
      const next: ScrubState = { phase: gesture.phase, tier: gesture.tier, atEdge: gesture.atEdge };
      return previous.phase === next.phase && previous.tier === next.tier && previous.atEdge === next.atEdge ? previous : next;
    });
  }, []);

  const end = useCallback((commitTap: boolean) => {
    const gesture = gestureRef.current;
    if (!gesture) return;
    gestureRef.current = null;
    if (frameRef.current != null) cancelAnimationFrame(frameRef.current);
    frameRef.current = null;
    detachRef.current?.();
    detachRef.current = null;
    setState(IDLE);
    if (gesture.phase === "scrubbing") {
      swallowClickUntil.current = performance.now() + CLICK_AFTER_SCRUB_MS;
      latest.current.onScrubEnd?.({ phase: "scrubbing", tier: gesture.tier, atEdge: gesture.atEdge });
    } else if (commitTap && gesture.source === "key") {
      // A pointer's tap arrives as its click; a key's never does, because the
      // press suppressed the button's own activation to own the hold.
      latest.current.step(1);
    }
  }, []);

  const frame = useCallback(() => {
    const gesture = gestureRef.current;
    if (!gesture) return;
    const { step: stepNow, holdDelay: delay, tiers: ladder } = latest.current;
    const now = performance.now();
    if (gesture.phase === "pressed") {
      if (now - gesture.pressedAt < delay) {
        frameRef.current = requestAnimationFrame(frame);
        return;
      }
      // The hold engages with a step, so it answers the moment it starts.
      gesture.phase = "scrubbing";
      gesture.scrubbedAt = now;
      gesture.lastFrame = now;
      gesture.atEdge = stepNow(1) < 1;
      buzz(gesture.atEdge ? 20 : 8);
    } else {
      const tier = tierAt(ladder, now - gesture.scrubbedAt);
      if (tier !== gesture.tier) {
        gesture.tier = tier;
        buzz(8);
      }
      gesture.owed += ((now - gesture.lastFrame) / 1000) * (ladder[tier]?.perSecond ?? 0);
      gesture.lastFrame = now;
      const whole = Math.floor(gesture.owed);
      if (whole > 0) {
        gesture.owed -= whole;
        if (stepNow(whole) < whole) {
          gesture.atEdge = true;
          buzz(20);
        }
      }
    }
    publish(gesture);
    if (!gesture.atEdge) frameRef.current = requestAnimationFrame(frame);
  }, [publish]);

  const begin = useCallback((source: Gesture["source"]) => {
    if (gestureRef.current || !latest.current.canStep) return;
    const now = performance.now();
    const gesture: Gesture = { source, phase: "pressed", pressedAt: now, scrubbedAt: now, lastFrame: now, owed: 0, tier: 0, atEdge: false };
    gestureRef.current = gesture;
    swallowClickUntil.current = 0;
    publish(gesture);
    const release = (event: Event) => {
      if (source === "key") {
        const key = (event as KeyboardEvent).key;
        if (key !== "Enter" && key !== " ") return;
        // Space activates a button on keyup; this hold already decided.
        event.preventDefault();
      }
      end(true);
    };
    const abandon = () => { end(false); };
    const onVisibility = () => { if (document.hidden) end(false); };
    const releaseEvents = source === "key" ? ["keyup"] : ["pointerup"];
    for (const name of releaseEvents) window.addEventListener(name, release);
    window.addEventListener("pointercancel", abandon);
    window.addEventListener("blur", abandon);
    document.addEventListener("visibilitychange", onVisibility);
    detachRef.current = () => {
      for (const name of releaseEvents) window.removeEventListener(name, release);
      window.removeEventListener("pointercancel", abandon);
      window.removeEventListener("blur", abandon);
      document.removeEventListener("visibilitychange", onVisibility);
    };
    frameRef.current = requestAnimationFrame(frame);
  }, [end, frame, publish]);

  useEffect(() => () => { end(false); }, [end]);

  const pressProps = {
    onPointerDown: (event: ReactPointerEvent<HTMLElement>) => {
      if (!event.isPrimary || event.button !== 0) return;
      begin("pointer");
    },
    onKeyDown: (event: ReactKeyboardEvent<HTMLElement>) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      // The button's own activation would step once per key repeat; the hold owns it.
      event.preventDefault();
      if (!event.repeat) begin("key");
    },
    onClick: (event: ReactMouseEvent<HTMLElement>) => {
      if (performance.now() < swallowClickUntil.current) {
        swallowClickUntil.current = 0;
        event.preventDefault();
        return;
      }
      latest.current.step(1);
    },
    // Android raises a context menu on a long press; the hold is the long press.
    onContextMenu: (event: ReactMouseEvent<HTMLElement>) => { event.preventDefault(); },
  };

  return { state, pressProps, holdDelay, tiers };
}
