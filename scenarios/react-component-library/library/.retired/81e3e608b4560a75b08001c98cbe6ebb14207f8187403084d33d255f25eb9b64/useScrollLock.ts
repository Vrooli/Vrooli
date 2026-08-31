/** @vrooliComponentSource hooks.use-scroll-lock */
import { useEffect } from "react";

export function useScrollLock(locked = true) {
  useEffect(() => {
    if (!locked) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [locked]);
}
