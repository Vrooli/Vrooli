import { useEffect } from "react";

/**
 * Writes the current usable viewport height to `--app-height` (in px) on
 * `document.documentElement`. Mobile browsers shrink/restore the URL bar
 * dynamically, so `100vh` overflows; consumers should reach for
 * `var(--app-height)` (or the Tailwind `h-[var(--app-height)]` form) on
 * any full-height container.
 *
 * Ported from `scenarios/web-console/ui/src/hooks/useAppViewport.ts`.
 * Feature-detects `visualViewport`; falls back to `window.innerHeight`.
 */
export function useAppViewport(): void {
  useEffect(() => {
    if (typeof window === "undefined") return undefined;

    const setVar = () => {
      const visual = window.visualViewport;
      const height = visual ? visual.height : window.innerHeight;
      document.documentElement.style.setProperty("--app-height", `${height}px`);
    };

    setVar();

    if (window.visualViewport) {
      window.visualViewport.addEventListener("resize", setVar);
      window.visualViewport.addEventListener("scroll", setVar);
    }
    window.addEventListener("resize", setVar);
    window.addEventListener("orientationchange", setVar);

    return () => {
      if (window.visualViewport) {
        window.visualViewport.removeEventListener("resize", setVar);
        window.visualViewport.removeEventListener("scroll", setVar);
      }
      window.removeEventListener("resize", setVar);
      window.removeEventListener("orientationchange", setVar);
    };
  }, []);
}
