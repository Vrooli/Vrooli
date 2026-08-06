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
