/**
 * useMediaQuery — SSR-safe responsive hook.
 *
 * Returns `false` during SSR / when matchMedia is unavailable, then resolves
 * to the live value once a window is present. Listens via addEventListener
 * (modern). Adjacent helpers (`useIsMobile`/`useIsTablet`/`useIsDesktop`)
 * encode the breakpoints declared in DESIGN.md so screen-size branching
 * lives in one place.
 */
import { useEffect, useState } from "react";

import { APP_MEDIA_QUERIES } from "../styles/breakpoints";

function readMatch(query: string): boolean {
  if (typeof window === "undefined") return false;
  if (typeof window.matchMedia !== "function") return false;
  return window.matchMedia(query).matches;
}

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => readMatch(query));

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (typeof window.matchMedia !== "function") return;

    const mql = window.matchMedia(query);
    setMatches(mql.matches);

    const handler = (event: MediaQueryListEvent) => setMatches(event.matches);
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [query]);

  return matches;
}

// Breakpoints align with DESIGN.md responsive transformations.
export function useIsMobile(): boolean {
  return useMediaQuery(APP_MEDIA_QUERIES.mobile);
}

export function useIsTablet(): boolean {
  return useMediaQuery(APP_MEDIA_QUERIES.tablet);
}

export function useIsDesktop(): boolean {
  return useMediaQuery(APP_MEDIA_QUERIES.desktop);
}
