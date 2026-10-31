import { useEffect, useState } from "react";

const MOBILE_QUERY = "(max-width: 42rem)";
const SHORT_LANDSCAPE_QUERY = "(max-width: 60rem) and (max-height: 42rem)";

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => typeof window !== "undefined" && window.matchMedia(query).matches);

  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [query]);

  return matches;
}

/** SSR-safe interaction breakpoint. Use this when the interaction model changes, not for styling. */
export function useIsMobile(): boolean {
  return useMediaQuery(MOBILE_QUERY);
}

export function useBreakpoint() {
  const isMobile = useIsMobile();
  const isShortLandscape = useMediaQuery(SHORT_LANDSCAPE_QUERY);
  return { isMobile, isDesktop: !isMobile, isShortLandscape };
}
