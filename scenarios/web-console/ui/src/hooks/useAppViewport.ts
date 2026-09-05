import { useEffect, useRef } from "react";
import { useViewportEnvironment } from "@vrooli/react-component-library/useViewportEnvironment/1";

/** Project shared viewport state into Web Console's app-shell contract. */
export function useAppViewport(options: { onKeyboardChange?: (open: boolean) => void } = {}): void {
  const viewport = useViewportEnvironment();
  const keyboardListenerRef = useRef(options.onKeyboardChange);
  const publishedKeyboardRef = useRef<boolean | null>(null);
  keyboardListenerRef.current = options.onKeyboardChange;

  useEffect(() => {
    const root = document.documentElement.style;
    root.setProperty("--wc-kb-height", `${String(viewport.keyboardInset)}px`);
    root.setProperty("--wc-app-height", `${String(viewport.visibleHeight)}px`);
    root.setProperty("--wc-safe-top", "env(safe-area-inset-top)");
    root.setProperty("--wc-safe-left", "env(safe-area-inset-left)");
    root.setProperty("--wc-safe-right", "env(safe-area-inset-right)");
    root.setProperty("--wc-safe-bottom", viewport.keyboardVisible ? "0px" : "env(safe-area-inset-bottom)");

    if (publishedKeyboardRef.current !== viewport.keyboardVisible) {
      publishedKeyboardRef.current = viewport.keyboardVisible;
      keyboardListenerRef.current?.(viewport.keyboardVisible);
    }

    // Correct only a demonstrated browser pan. Calling scrollTo for every
    // viewport signal creates a feedback loop that can move an idle overlay.
    if (window.scrollX !== 0 || window.scrollY !== 0 || viewport.offsetLeft !== 0 || viewport.offsetTop !== 0) {
      window.scrollTo(0, 0);
    }
  }, [viewport.keyboardInset, viewport.keyboardVisible, viewport.offsetLeft, viewport.offsetTop, viewport.visibleHeight]);

  useEffect(() => () => {
    const root = document.documentElement.style;
    root.removeProperty("--wc-kb-height");
    root.removeProperty("--wc-app-height");
    root.removeProperty("--wc-safe-top");
    root.removeProperty("--wc-safe-bottom");
    root.removeProperty("--wc-safe-left");
    root.removeProperty("--wc-safe-right");
    publishedKeyboardRef.current = null;
  }, []);
}
