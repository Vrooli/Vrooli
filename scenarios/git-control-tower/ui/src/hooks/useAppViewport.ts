import { useCallback, useEffect, useRef } from "react";
import { useViewportEnvironment } from "@vrooli/react-component-library/useViewportEnvironment/1";

/** Share the shell height and effective safe area with library navigation/overlays. */
export function useAppViewport(): void {
  const viewport = useViewportEnvironment();
  const latestViewport = useRef(viewport);
  latestViewport.current = viewport;
  const frame = useRef(0);
  const recover = useCallback(() => {
    frame.current = 0;
    const current = latestViewport.current;
    // Native zoom can arrive before the shared environment publishes it.
    const scale = window.visualViewport?.scale ?? current.scale;
    if (Math.abs(scale - 1) >= 0.01) return;

    // The app owns inner scrolling, including focus reveal. A real document pan
    // moves its fixed shell offscreen; a visual offset alone is not that proof.
    if (window.scrollX !== 0 || window.scrollY !== 0) {
      window.scrollTo(0, 0);
      // WebKit can acknowledge asynchronously. Its scroll event schedules the
      // next measurement; never reveal using the still-displaced coordinates.
      if (window.scrollX !== 0 || window.scrollY !== 0) return;
    }
    const focused = document.activeElement;
    if (!(focused instanceof HTMLElement) || !(focused.isContentEditable || focused.matches(
      "input:not([type='button']):not([type='checkbox']):not([type='radio']):not([type='range']):not([type='submit']):not([type='reset']), textarea, select",
    ))) return;
    const owner = focused.closest<HTMLElement>("[data-gct-scroll-owner]");
    if (!owner || owner.clientHeight <= 0) return;
    const clip = owner.getBoundingClientRect();
    const editor = focused.getBoundingClientRect();
    const top = Math.max(0, clip.top) + 8;
    const bottom = Math.min(clip.bottom, window.visualViewport?.height ?? current.visibleHeight) - 8;
    if (bottom <= top || editor.height <= 0) return;
    let delta = 0;
    if (editor.height > bottom - top) {
      if (editor.top < top || editor.top >= bottom) delta = editor.top - top;
    } else if (editor.bottom > bottom) {
      delta = editor.bottom - bottom;
    } else if (editor.top < top) {
      delta = editor.top - top;
    }
    if (delta !== 0) owner.scrollTop = Math.max(0, Math.min(owner.scrollHeight - owner.clientHeight, owner.scrollTop + delta));
  }, []);
  const scheduleRecovery = useCallback(() => {
    if (!frame.current) frame.current = window.requestAnimationFrame(recover);
  }, [recover]);

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

    scheduleRecovery();
  }, [
    viewport.keyboardVisible,
    viewport.visibleHeight,
    viewport.layoutHeight,
    viewport.frameHeight,
    viewport.reachesScreenBottom,
    viewport.offsetLeft,
    viewport.offsetTop,
    viewport.scale,
    scheduleRecovery,
  ]);

  useEffect(() => {
    const onScroll = (event: Event) => {
      if (event.target === event.currentTarget || event.target === document) scheduleRecovery();
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    document.addEventListener("focusin", scheduleRecovery, true);
    return () => {
      window.removeEventListener("scroll", onScroll);
      document.removeEventListener("focusin", scheduleRecovery, true);
      window.cancelAnimationFrame(frame.current);
      frame.current = 0;
    };
  }, [scheduleRecovery]);

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
