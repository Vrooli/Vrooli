import { useEffect, useState } from "react";

const MOBILE_QUERY = "(max-width: 42rem)";

/** SSR-safe interaction breakpoint. Use this when the interaction model changes, not for styling. */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(() => typeof window !== "undefined" && window.matchMedia(MOBILE_QUERY).matches);

  useEffect(() => {
    const media = window.matchMedia(MOBILE_QUERY);
    const update = () => setIsMobile(media.matches);
    update();
    media.addEventListener?.("change", update);
    return () => media.removeEventListener?.("change", update);
  }, []);

  return isMobile;
}

export function useBreakpoint() {
  const isMobile = useIsMobile();
  return { isMobile, isDesktop: !isMobile };
}
