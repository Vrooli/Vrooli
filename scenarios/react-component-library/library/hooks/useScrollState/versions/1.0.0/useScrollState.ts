/** @vrooliComponentSource hooks.use-scroll-state */
import { useEffect, useState } from "react";

export function useScrollState(element?: HTMLElement | null) {
  const [scrollTop, setScrollTop] = useState(0);
  useEffect(() => {
    const target = element ?? window;
    const onScroll = () => setScrollTop(element?.scrollTop ?? window.scrollY);
    target.addEventListener("scroll", onScroll, { passive: true });
    return () => target.removeEventListener("scroll", onScroll);
  }, [element]);
  return { scrollTop, atStart: scrollTop === 0 };
}
