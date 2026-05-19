// Viewport detection via matchMedia. Maps the browser width to one of
// the three viewport buckets declared in ui/flow/navigation.json
// (mobile, tablet, desktop). The thresholds mirror Tailwind's md/lg
// breakpoints so layout decisions stay aligned with utility classes.
import { useEffect, useState } from "react";

export type Viewport = "mobile" | "tablet" | "desktop";

const MOBILE_MAX = 767;
const TABLET_MAX = 1023;

function classify(width: number): Viewport {
  if (width <= MOBILE_MAX) return "mobile";
  if (width <= TABLET_MAX) return "tablet";
  return "desktop";
}

export function useViewport(): Viewport {
  const [vp, setVp] = useState<Viewport>(() => {
    if (typeof window === "undefined") return "desktop";
    return classify(window.innerWidth);
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    const onResize = () => setVp(classify(window.innerWidth));
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  return vp;
}
