import { useEffect, useRef } from "react";

/** Runs once for the first pointer, touch, or keyboard user activation. */
export function useFirstUserGesture(onGesture: () => void): void {
  const callback = useRef(onGesture);
  const fired = useRef(false);
  callback.current = onGesture;

  useEffect(() => {
    const handler = () => {
      if (fired.current) return;
      fired.current = true;
      callback.current();
    };
    window.addEventListener("pointerdown", handler, { passive: true, once: true });
    window.addEventListener("keydown", handler, { passive: true, once: true });
    window.addEventListener("touchstart", handler, { passive: true, once: true });
    return () => {
      window.removeEventListener("pointerdown", handler);
      window.removeEventListener("keydown", handler);
      window.removeEventListener("touchstart", handler);
    };
  }, []);
}
