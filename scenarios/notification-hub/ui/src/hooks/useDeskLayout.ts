import { useEffect, useState } from "react";

/**
 * The width at which the desk can hold the queue beside a decision without
 * squeezing either. Below it the queue keeps its single-column phone/tablet
 * composition. This is a window width, not a CSS query, so a test viewport and
 * a resized browser take the same branch.
 */
export const DESK_MIN_WIDTH = 1200;

export function useDeskLayout(): boolean {
  const [desk, setDesk] = useState(() => typeof window !== "undefined" && window.innerWidth >= DESK_MIN_WIDTH);
  useEffect(() => {
    const update = () => setDesk(window.innerWidth >= DESK_MIN_WIDTH);
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);
  return desk;
}
