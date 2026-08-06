/** @vrooliComponentSource hooks.use-resize-observer */
import { useCallback, useState } from "react";

export function useResizeObserver() {
  const [rect, setRect] = useState<DOMRectReadOnly | null>(null);
  const ref = useCallback((node: HTMLElement | null) => {
    if (!node || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(([entry]) =>
      setRect(entry?.contentRect ?? null),
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);
  return { ref, rect };
}
