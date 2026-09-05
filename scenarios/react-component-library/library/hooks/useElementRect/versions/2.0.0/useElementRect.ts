/** @vrooliComponentSource hooks.use-element-rect */
import { useEffect, useLayoutEffect, useRef, useState, type RefObject } from "react";

export type ElementRectTarget = RefObject<HTMLElement | null> | HTMLElement | null;

export interface UseElementRectOptions {
  /** Stop observing without unmounting. The last measured rect is retained. */
  disabled?: boolean;
  /**
   * Re-measure on scroll. Off by default because only consumers that compare
   * against viewport coordinates need it, and a capture-phase scroll listener
   * is the most expensive thing this hook can install.
   */
  trackScroll?: boolean;
}

// The first measurement runs before paint so consumers never render against a
// null rect. On the server there is no layout phase to run it in.
const useIsomorphicLayoutEffect = typeof window === "undefined" ? useEffect : useLayoutEffect;

function resolveElement(target: ElementRectTarget): HTMLElement | null {
  if (!target) return null;
  return "current" in target ? target.current : target;
}

function sameRect(a: DOMRectReadOnly | null, b: DOMRectReadOnly): boolean {
  return (
    a !== null && a.width === b.width && a.height === b.height && a.x === b.x && a.y === b.y
  );
}

/**
 * Border-box geometry for an element, kept current through a ResizeObserver
 * rather than a window `resize` listener — a panel collapsing, a font loading,
 * or a container reflowing all change the box without resizing the window.
 * Reads are batched into an animation frame so a burst of observer callbacks
 * costs one layout read, not one per callback.
 */
export function useElementRect(
  target: ElementRectTarget,
  options: UseElementRectOptions = {},
): DOMRectReadOnly | null {
  const { disabled = false, trackScroll = false } = options;
  const [rect, setRect] = useState<DOMRectReadOnly | null>(null);
  const lastRef = useRef<DOMRectReadOnly | null>(null);

  // A ref populated during commit does not re-render the component, so reading
  // `ref.current` during render would pin this hook to the mount-time value of
  // null and it would never observe anything. Resolving into state in the
  // layout phase picks the node up before paint, for a ref or a raw node alike.
  const [element, setElement] = useState<HTMLElement | null>(() => resolveElement(target));
  useIsomorphicLayoutEffect(() => {
    const next = resolveElement(target);
    setElement((current) => (current === next ? current : next));
  });

  useIsomorphicLayoutEffect(() => {
    if (!element || disabled) return;

    let frame = 0;
    const commit = () => {
      frame = 0;
      const next = element.getBoundingClientRect();
      if (sameRect(lastRef.current, next)) return;
      lastRef.current = next;
      setRect(next);
    };
    const schedule = () => {
      if (frame) return;
      if (typeof requestAnimationFrame !== "function") {
        commit();
        return;
      }
      frame = requestAnimationFrame(commit);
    };

    commit();

    let observer: ResizeObserver | undefined;
    if (typeof ResizeObserver !== "undefined") {
      observer = new ResizeObserver(schedule);
      observer.observe(element, { box: "border-box" });
    }
    window.addEventListener("resize", schedule);
    if (trackScroll) window.addEventListener("scroll", schedule, true);

    return () => {
      if (frame && typeof cancelAnimationFrame === "function") cancelAnimationFrame(frame);
      observer?.disconnect();
      window.removeEventListener("resize", schedule);
      if (trackScroll) window.removeEventListener("scroll", schedule, true);
    };
  }, [element, disabled, trackScroll]);

  return rect;
}
