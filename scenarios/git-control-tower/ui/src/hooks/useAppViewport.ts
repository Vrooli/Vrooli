import { useEffect } from "react";
import { useViewportEnvironment } from "@vrooli/react-component-library/useViewportEnvironment/1";

/** Share the shell height and effective safe area with library navigation/overlays. */
export function useAppViewport(): void {
  const viewport = useViewportEnvironment();

  useEffect(() => {
    const root = document.documentElement.style;
    // visualViewport can stay keyboard-shrunken for a frame (and on some iOS
    // paths until the next interaction) after the keyboard is dismissed. The
    // layout viewport is the stable app height; only adopt the smaller visible
    // height once the shared detector has positively established a keyboard.
    const appHeight = viewport.keyboardVisible ? viewport.visibleHeight : viewport.frameHeight ?? viewport.layoutHeight;
    document.documentElement.dataset.gctKeyboardVisible = String(viewport.keyboardVisible);
    root.setProperty("--gct-app-height", `${String(appHeight)}px`);
    root.setProperty("--rcl-safe-bottom", viewport.reachesScreenBottom ? "env(safe-area-inset-bottom, 0px)" : "0px");

    // Correct only a demonstrated browser pan; scrolling for every signal creates
    // a feedback loop that can drift an idle overlay.
    if (window.scrollX !== 0 || window.scrollY !== 0 || viewport.offsetLeft !== 0 || viewport.offsetTop !== 0) {
      window.scrollTo(0, 0);
    }
  }, [
    viewport.keyboardVisible,
    viewport.visibleHeight,
    viewport.layoutHeight,
    viewport.frameHeight,
    viewport.reachesScreenBottom,
    viewport.offsetLeft,
    viewport.offsetTop,
  ]);

  useEffect(
    () => () => {
      const root = document.documentElement.style;
      delete document.documentElement.dataset.gctKeyboardVisible;
      root.removeProperty("--gct-app-height");
      root.removeProperty("--rcl-safe-bottom");
    },
    []
  );
}
