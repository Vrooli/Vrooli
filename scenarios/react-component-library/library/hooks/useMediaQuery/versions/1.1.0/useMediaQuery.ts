/**
 * @libraryId react-component-library:useMediaQuery
 * @displayName useMediaQuery
 * @version 1.1.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-media-query */
import { useSyncExternalStore } from "react";

export function useMediaQuery(query: string) {
  return useSyncExternalStore(
    (onChange) => {
      const media = window.matchMedia(query);
      media.addEventListener("change", onChange);
      return () => media.removeEventListener("change", onChange);
    },
    () => typeof window !== "undefined" && window.matchMedia(query).matches,
    () => false,
  );
}

export type BreakpointName = "md" | (string & {});

/** Resolves a published breakpoint token and subscribes to its media query. */
export function useBreakpoint(name: BreakpointName) {
  const query =
    typeof document === "undefined"
      ? "(min-width: 48rem)"
      : `(min-width: ${getComputedStyle(document.documentElement).getPropertyValue(`--breakpoint-${name}`).trim() || "48rem"})`;
  return useMediaQuery(query);
}
