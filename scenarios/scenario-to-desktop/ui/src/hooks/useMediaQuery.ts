/**
 * Responsive breakpoint hook using window.matchMedia.
 *
 * Seam: UI ↔ viewport — all mobile/desktop branching goes through this hook
 * so tests can substitute a fixed breakpoint without touching window internals.
 *
 * DOC: docs/internal/SEAMS.md#responsive-breakpoint-seam
 */

import { useEffect, useState } from "react";

/**
 * Subscribe to a CSS media-query string and return whether it currently matches.
 *
 * @example
 * const isMobile = useMediaQuery("(max-width: 767px)");
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    const mql = window.matchMedia(query);
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches);

    // Sync in case the value changed between render and effect
    setMatches(mql.matches);

    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [query]);

  return matches;
}

/** Tailwind `md` breakpoint — screens below 768 px are considered mobile. */
export const MOBILE_QUERY = "(max-width: 767px)";

/** Convenience wrapper: true when viewport < 768 px. */
export function useIsMobile(): boolean {
  return useMediaQuery(MOBILE_QUERY);
}
