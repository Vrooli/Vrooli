import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Track an element's content-box width.
 *
 * Returns `null` until the element is measured, so callers can distinguish
 * "not measured yet" from "zero wide" and avoid laying out against a width
 * that was never real. Falls back to a one-shot read where ResizeObserver is
 * unavailable (older Safari, jsdom).
 */
export function useElementWidth(): [(node: HTMLElement | null) => void, number | null] {
  const [width, setWidth] = useState<number | null>(null);
  const nodeRef = useRef<HTMLElement | null>(null);
  const observerRef = useRef<ResizeObserver | null>(null);

  const ref = useCallback((node: HTMLElement | null) => {
    observerRef.current?.disconnect();
    observerRef.current = null;
    nodeRef.current = node;
    if (!node) return;

    const measure = () => {
      // Round down: laying out against a fractional width can push the last
      // control a sub-pixel past the edge and trigger a scrollbar.
      const measured = Math.floor(node.getBoundingClientRect().width);
      // A zero width means "no layout yet" — a detached node, a hidden
      // ancestor, or an environment without layout at all (jsdom). Reporting
      // it as a real measurement would lay the caller out against nothing.
      const next = measured > 0 ? measured : null;
      setWidth((current) => (current === next ? current : next));
    };

    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(node);
    observerRef.current = observer;
  }, []);

  useEffect(() => () => observerRef.current?.disconnect(), []);

  return [ref, width];
}
