import { useEffect, useState } from "react";

function getNeedsTouchControls(): boolean {
  if (typeof window === "undefined") return false;
  const nav = window.navigator;
  const hasTouchPoints =
    typeof nav.maxTouchPoints === "number" && nav.maxTouchPoints > 0;
  const coarsePointer =
    typeof window.matchMedia === "function" &&
    (window.matchMedia("(pointer: coarse)").matches ||
      window.matchMedia("(hover: none)").matches);
  return hasTouchPoints || coarsePointer;
}

/**
 * True when the current device needs on-screen terminal controls even if its
 * CSS viewport is wide enough for the desktop layout.
 */
export function useTouchControls(): boolean {
  const [needsTouchControls, setNeedsTouchControls] = useState(getNeedsTouchControls);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }

    const pointerQuery = window.matchMedia("(pointer: coarse)");
    const hoverQuery = window.matchMedia("(hover: none)");
    const update = () => setNeedsTouchControls(getNeedsTouchControls());

    update();
    pointerQuery.addEventListener("change", update);
    hoverQuery.addEventListener("change", update);
    window.addEventListener("resize", update);
    return () => {
      pointerQuery.removeEventListener("change", update);
      hoverQuery.removeEventListener("change", update);
      window.removeEventListener("resize", update);
    };
  }, []);

  return needsTouchControls;
}
