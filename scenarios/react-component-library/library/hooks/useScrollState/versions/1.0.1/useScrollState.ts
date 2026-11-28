/**
 * @libraryId react-component-library:useScrollState
 * @displayName useScrollState
 * @description A scroll primitive reporting position, direction, velocity, boundary proximity, and recent activity without re-rendering on every raw scroll event.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useScrollState
 * @vrooliComponentSourceSlot hooks.use-scroll-state */
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
