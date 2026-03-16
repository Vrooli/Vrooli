import { useEffect } from "react";

/**
 * Sets the CSS custom property `--wc-kb-height` on the document root
 * to the current virtual-keyboard height (px).  Layout is driven
 * entirely through CSS so that no React re-render is needed — this
 * avoids focus-loss issues in xterm when the container resizes.
 *
 * Also scrolls the window back to (0,0) on every viewport change to
 * prevent the browser from pushing the page up behind the keyboard.
 *
 * ## Why focusin/focusout polling is needed
 *
 * The `visualViewport` resize event is the primary signal, but some
 * mobile browsers don't fire it reliably for all focus targets. In
 * particular, focusing the MobileToolbar textarea (at the bottom of
 * the layout) may not trigger a resize event on certain devices/browsers,
 * even though the virtual keyboard opens and the viewport shrinks.
 *
 * To handle this, we also listen for `focusin`/`focusout` on the
 * document and poll the viewport dimensions a few times during the
 * ~300ms keyboard open/close animation. This ensures the layout
 * adjusts regardless of which element receives focus.
 */
export function useVirtualKeyboard(): void {
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;

    const update = () => {
      const kbHeight = Math.max(0, window.innerHeight - vv.height - vv.offsetTop);
      document.documentElement.style.setProperty("--wc-kb-height", `${kbHeight}px`);
      // Prevent the browser from scrolling the page up to keep the
      // focused input visible — we handle layout via CSS.
      window.scrollTo(0, 0);
    };

    // Poll update() several times over the keyboard animation period.
    // This catches cases where visualViewport resize events don't fire
    // (or fire too early) for certain focus targets.
    let pollTimers: ReturnType<typeof setTimeout>[] = [];
    const pollDuringAnimation = () => {
      // Clear any pending polls from a previous focus change
      for (const t of pollTimers) clearTimeout(t);
      pollTimers = [];
      update();
      // Poll at 50ms intervals over 500ms to cover the keyboard animation
      for (const delay of [50, 100, 150, 200, 300, 500]) {
        pollTimers.push(setTimeout(update, delay));
      }
    };

    const isInputElement = (el: EventTarget | null): boolean => {
      if (!(el instanceof HTMLElement)) return false;
      const tag = el.tagName;
      return tag === "INPUT" || tag === "TEXTAREA" || el.isContentEditable;
    };

    const onFocusIn = (e: FocusEvent) => {
      if (isInputElement(e.target)) pollDuringAnimation();
    };
    const onFocusOut = (e: FocusEvent) => {
      if (isInputElement(e.target)) pollDuringAnimation();
    };

    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    document.addEventListener("focusin", onFocusIn);
    document.addEventListener("focusout", onFocusOut);
    // Set initial value
    update();
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
      document.removeEventListener("focusin", onFocusIn);
      document.removeEventListener("focusout", onFocusOut);
      for (const t of pollTimers) clearTimeout(t);
      document.documentElement.style.removeProperty("--wc-kb-height");
    };
  }, []);
}
