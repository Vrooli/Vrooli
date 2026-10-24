import { useEffect, useState } from "react";

/**
 * Subscribe to a CSS media query. Returns the current match state and
 * updates on change. SSR-safe: returns `false` when `window` is missing.
 */
export function useMediaQuery(query: string): boolean {
  const getInitial = () => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return false;
    }
    return window.matchMedia(query).matches;
  };

  const [matches, setMatches] = useState<boolean>(getInitial);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return undefined;
    }
    const mql = window.matchMedia(query);
    const onChange = (e: MediaQueryListEvent) => setMatches(e.matches);
    // Update once on mount in case the query changed between render and effect.
    setMatches(mql.matches);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [query]);

  return matches;
}

/** Convenience wrapper for the canonical mobile breakpoint (< md). */
export function useIsMobile(): boolean {
  return useMediaQuery("(max-width: 767px)");
}
