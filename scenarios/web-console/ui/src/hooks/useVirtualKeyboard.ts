import { useEffect } from "react";

/**
 * Sets the CSS custom property `--wc-kb-height` on the document root
 * to the current virtual-keyboard height (px).  Layout is driven
 * entirely through CSS so that no React re-render is needed — this
 * avoids focus-loss issues in xterm when the container resizes.
 *
 * Also scrolls the window back to (0,0) on every viewport change to
 * prevent the browser from pushing the page up behind the keyboard.
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

    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    // Set initial value
    update();
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
      document.documentElement.style.removeProperty("--wc-kb-height");
    };
  }, []);
}
