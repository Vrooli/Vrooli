/**
 * @libraryId react-component-library:useIconMorph
 * @displayName useIconMorph
 * @description Drives an icon swap, choosing path morphing or a crossfade from measured geometry compatibility.
 * @version 1.1.0
 * @tags ["hook","motion","icons","reduced-motion"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-icon-morph */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  MORPH_SCORE_THRESHOLD,
  alignGeometry,
  geometryFromElement,
  interpolateAligned,
  morphCompatibility,
  type IconGeometry,
} from "@vrooli/react-component-library/IconGeometry/1.0.0";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";

/**
 * How a swap should be animated.
 *
 * - `auto` — always animate; upgrade to a path morph only when the two shapes
 *   measure compatible. This is the default because it is the only setting that
 *   is safe to apply without knowing which icons a caller will pass.
 * - `morph` — force path interpolation whenever geometry can be read at all.
 * - `crossfade` — never morph.
 * - `none` — swap instantly.
 */
export type IconMorphMode = "auto" | "morph" | "crossfade" | "none";

export type IconMorphTechnique = "morph" | "crossfade" | "none";

export interface UseIconMorphOptions {
  /** Identity of the icon currently requested. A change starts a transition. */
  iconKey: string;
  /**
   * A stable identity for the *control*, so an icon change still animates when
   * the control remounts.
   *
   * A component instance normally remembers what it was showing in a ref, which
   * is destroyed on unmount. That is the right default — a control appearing for
   * the first time should not animate. But a control can legitimately move
   * between parents while remaining the same control to the user: a responsive
   * layout that renders it floating in one mode and inline in another, a portal,
   * a reordered list. React cannot preserve state across that move, so the
   * remounted control believes it has always shown its current icon and the swap
   * is silently skipped.
   *
   * Supplying a `swapIdentity` moves the record of "what was last showing" from
   * per-instance state into a registry keyed by this string, so the move is
   * invisible to the animation. Give it a value that is stable for the control
   * and unique among controls.
   */
  swapIdentity?: string;
  mode?: IconMorphMode;
  /** Milliseconds. Ignored under `prefers-reduced-motion`. */
  duration?: number;
}

export interface IconMorphState {
  /** Which technique this swap settled on. */
  technique: IconMorphTechnique;
  /** 0..1 across the current transition; 1 when idle. */
  progress: number;
  /** True while a transition is running. */
  active: boolean;
  /** Interpolated geometry for `technique === "morph"`, else null. */
  geometry: IconGeometry | null;
  /**
   * The key of the icon being animated away from. Rendered underneath the
   * incoming icon during a crossfade; null when idle.
   */
  previousKey: string | null;
  /** Attach to the element wrapping the *current* icon. */
  currentRef: (node: Element | null) => void;
  /** Attach to the element wrapping the *outgoing* icon during a crossfade. */
  previousRef: (node: Element | null) => void;
}

const DEFAULT_DURATION = 320;

/**
 * Geometry is expensive relative to a render but cheap relative to a session,
 * and it is a pure function of the icon's markup. Caching by icon key means a
 * given icon is measured once no matter how many times it is swapped in, and a
 * button whose icon never changes measures nothing at all.
 *
 * A module-level `Map` is correct here rather than per-instance state: two
 * buttons showing the same icon should share the measurement, and icon
 * vocabularies are bounded by the application's own imports.
 */
const geometryCache = new Map<string, IconGeometry | null>();

/**
 * The last settled icon key per `swapIdentity`. This is deliberately outside
 * React state: its whole purpose is to outlive the component instance.
 */
const settledByIdentity = new Map<string, string>();

/** Alignment is O(points²) for closed loops, so cache per ordered pair. */
const alignmentCache = new Map<string, ReturnType<typeof alignGeometry>>();
const compatibilityCache = new Map<string, number>();

/** Escape hatch for tests and for hosts that hot-swap an icon set. */
export function clearIconMorphCache() {
  geometryCache.clear();
  alignmentCache.clear();
  compatibilityCache.clear();
  settledByIdentity.clear();
}

function readGeometry(key: string, node: Element | null): IconGeometry | null {
  if (geometryCache.has(key)) return geometryCache.get(key) ?? null;
  const geometry = geometryFromElement(node);
  // Only cache a successful read. A miss usually means the element had not
  // mounted yet, and caching that would poison the icon for the session.
  if (geometry) geometryCache.set(key, geometry);
  return geometry;
}

/**
 * Ease-out cubic. Icon swaps are short, and a decelerating curve puts most of
 * the visible motion in the first half where the eye is still tracking it.
 */
function ease(t: number): number {
  return 1 - (1 - t) ** 3;
}

/**
 * Drive a single icon swap.
 *
 * The hook owns three things a component should not re-implement: the
 * measurement lifecycle (icons can only be measured once mounted), the
 * technique decision, and the animation frame loop. It deliberately does not
 * render anything — `MorphingIcon` owns that — so the same decision logic can
 * back a different renderer later without forking.
 */
export function useIconMorph({
  iconKey,
  swapIdentity,
  mode = "auto",
  duration = DEFAULT_DURATION,
}: UseIconMorphOptions): IconMorphState {
  const reducedMotion = useReducedMotion();
  const currentNode = useRef<Element | null>(null);
  const previousNode = useRef<Element | null>(null);
  const frame = useRef<number | null>(null);
  const pairs = useRef<ReturnType<typeof alignGeometry> | null>(null);
  const viewBox = useRef("0 0 24 24");

  // The key rendered on the previous commit, so a change can be detected
  // without the caller telling us one happened. With a `swapIdentity`, the
  // starting value comes from the registry instead, so a control that remounted
  // still knows what it was showing before the move. An identity seen for the
  // first time falls back to the current key, which correctly means "do not
  // animate on first appearance".
  const settledKey = useRef(
    swapIdentity ? (settledByIdentity.get(swapIdentity) ?? iconKey) : iconKey,
  );

  const [state, setState] = useState<{
    technique: IconMorphTechnique;
    progress: number;
    active: boolean;
    geometry: IconGeometry | null;
    previousKey: string | null;
  }>({ technique: "none", progress: 1, active: false, geometry: null, previousKey: null });

  const currentRef = useCallback((node: Element | null) => {
    currentNode.current = node;
  }, []);
  const previousRef = useCallback((node: Element | null) => {
    previousNode.current = node;
  }, []);

  const stop = useCallback(() => {
    if (frame.current !== null) {
      cancelAnimationFrame(frame.current);
      frame.current = null;
    }
  }, []);

  useEffect(() => {
    // Measure whatever is on screen now, so the geometry is available as the
    // *outgoing* icon on the next swap even if this icon never animated in.
    readGeometry(iconKey, currentNode.current);
    // Record the settled key even when no transition ran, so the next mount
    // under this identity compares against what was actually last shown.
    if (swapIdentity) settledByIdentity.set(swapIdentity, iconKey);
  });

  useEffect(() => {
    const from = settledKey.current;
    if (from === iconKey) return undefined;
    settledKey.current = iconKey;
    if (swapIdentity) settledByIdentity.set(swapIdentity, iconKey);

    const finish = () => {
      stop();
      pairs.current = null;
      setState({
        technique: "none", progress: 1, active: false, geometry: null, previousKey: null,
      });
    };

    if (mode === "none" || reducedMotion || duration <= 0) {
      finish();
      return undefined;
    }

    const fromGeometry = geometryCache.get(from) ?? null;
    // The incoming icon is already committed to the DOM by the time this effect
    // runs, so it can be measured directly rather than guessed at.
    const toGeometry = readGeometry(iconKey, currentNode.current);

    let technique: IconMorphTechnique = "crossfade";
    if (mode !== "crossfade" && fromGeometry && toGeometry) {
      if (mode === "morph") {
        technique = "morph";
      } else {
        const cacheKey = `${from} ${iconKey}`;
        let score = compatibilityCache.get(cacheKey);
        if (score === undefined) {
          score = morphCompatibility(fromGeometry, toGeometry).score;
          compatibilityCache.set(cacheKey, score);
        }
        if (score > MORPH_SCORE_THRESHOLD) technique = "morph";
      }
    }

    if (technique === "morph" && fromGeometry && toGeometry) {
      const cacheKey = `${from} ${iconKey}`;
      let aligned = alignmentCache.get(cacheKey);
      if (!aligned) {
        aligned = alignGeometry(fromGeometry, toGeometry);
        alignmentCache.set(cacheKey, aligned);
      }
      pairs.current = aligned;
      viewBox.current = toGeometry.viewBox || fromGeometry.viewBox;
    } else {
      pairs.current = null;
    }

    stop();
    const started = performance.now();
    setState({
      technique,
      progress: 0,
      active: true,
      geometry: pairs.current ? interpolateAligned(pairs.current, 0, viewBox.current) : null,
      previousKey: technique === "crossfade" ? from : null,
    });

    const tick = (now: number) => {
      const linear = Math.min(1, (now - started) / duration);
      const progress = ease(linear);
      if (linear >= 1) {
        finish();
        return;
      }
      setState({
        technique,
        progress,
        active: true,
        geometry: pairs.current
          ? interpolateAligned(pairs.current, progress, viewBox.current)
          : null,
        previousKey: technique === "crossfade" ? from : null,
      });
      frame.current = requestAnimationFrame(tick);
    };
    frame.current = requestAnimationFrame(tick);

    return stop;
  }, [iconKey, swapIdentity, mode, duration, reducedMotion, stop]);

  useEffect(() => stop, [stop]);

  return {
    technique: state.technique,
    progress: state.progress,
    active: state.active,
    geometry: state.geometry,
    previousKey: state.previousKey,
    currentRef,
    previousRef,
  };
}
