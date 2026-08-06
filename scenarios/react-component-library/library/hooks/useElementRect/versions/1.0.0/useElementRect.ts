/** @vrooliComponentSource hooks.use-element-rect */
import { useEffect, useState } from "react";

export function useElementRect(element: HTMLElement | null) {
  const [rect, setRect] = useState<DOMRectReadOnly | null>(null);
  useEffect(() => {
    if (!element) return;
    const measure = () => setRect(element.getBoundingClientRect());
    measure();
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  }, [element]);
  return rect;
}
