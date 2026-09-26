/**
 * @libraryId react-component-library:useReducedMotion
 * @displayName useReducedMotion
 * @description A motion-policy primitive resolving user preference and application policy into full, reduced, or disabled animation behavior for the calling surface.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useReducedMotion
 * @vrooliComponentSourceSlot hooks.use-reduced-motion */
import { useSyncExternalStore } from "react";

export function useMediaQuery(query: string) {
  return useSyncExternalStore(
    (onChange) => {
      if (typeof window === "undefined" || typeof window.matchMedia !== "function") return () => {};
      const media = window.matchMedia(query);
      const listener = () => onChange();
      media.addEventListener("change", listener);
      return () => {
        media.removeEventListener("change", listener);
      };
    },
    () => typeof window !== "undefined" && window.matchMedia(query).matches,
    () => false,
  );
}

export function useReducedMotion() {
  return useMediaQuery("(prefers-reduced-motion: reduce)");
}
