/** @vrooliComponentSource hooks.use-resize-observer */
import { useCallback, useEffect, useState } from "react";

export interface UseResizeObserverOptions {
  /**
   * Which box to observe. Defaults to the border box, which is what a caller
   * clamping a panel against a container is actually measuring; `contentRect`
   * excludes padding and border and silently under-reports by that much.
   */
  box?: ResizeObserverBoxOptions;
}

export interface UseResizeObserverResult<T extends HTMLElement = HTMLElement> {
  ref: (node: T | null) => void;
  rect: DOMRectReadOnly | null;
}

/**
 * Observed border-box geometry for whatever the returned ref is attached to.
 *
 * The observer lives in an effect keyed on the attached node. A callback ref
 * cannot own it: React only honours a cleanup function returned from a callback
 * ref as of React 19, and this package supports React 18, where the return is
 * discarded and `disconnect()` never runs.
 */
export function useResizeObserver<T extends HTMLElement = HTMLElement>(
  options: UseResizeObserverOptions = {},
): UseResizeObserverResult<T> {
  const { box = "border-box" } = options;
  const [node, setNode] = useState<T | null>(null);
  const [rect, setRect] = useState<DOMRectReadOnly | null>(null);
  const ref = useCallback((next: T | null) => setNode(next), []);

  useEffect(() => {
    if (!node || typeof ResizeObserver === "undefined") return;

    let frame = 0;
    const commit = () => {
      frame = 0;
      setRect(node.getBoundingClientRect());
    };
    const schedule = () => {
      if (frame) return;
      if (typeof requestAnimationFrame !== "function") {
        commit();
        return;
      }
      frame = requestAnimationFrame(commit);
    };

    const observer = new ResizeObserver(schedule);
    observer.observe(node, { box });
    commit();

    return () => {
      if (frame && typeof cancelAnimationFrame === "function") cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, [node, box]);

  return { ref, rect };
}
