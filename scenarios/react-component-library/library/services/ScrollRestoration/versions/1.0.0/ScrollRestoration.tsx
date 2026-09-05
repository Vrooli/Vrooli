/** @vrooliComponentSource navigation.scroll-restoration */
import { useEffect } from "react";

export type ScrollIntent = "restore" | "reset" | "preserve";

export interface ScrollRestorationOptions {
  intent?: ScrollIntent;
  storage?: Storage;
}

export function useScrollRestoration(
  key: string,
  { intent = "restore", storage }: ScrollRestorationOptions = {},
) {
  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    const stateStorage =
      storage ??
      (typeof sessionStorage === "undefined" ? undefined : sessionStorage);
    const storageKey = `scroll:${key}`;
    const previousRestoration = window.history.scrollRestoration;
    window.history.scrollRestoration = "manual";
    let frame = 0;

    if (intent === "reset") {
      frame = window.requestAnimationFrame(() =>
        window.scrollTo({ top: 0, behavior: "auto" }),
      );
    } else if (intent === "restore") {
      const saved = stateStorage?.getItem(storageKey);
      const top = saved === null ? 0 : Number(saved);
      if (Number.isFinite(top)) {
        frame = window.requestAnimationFrame(() =>
          window.scrollTo({ top, behavior: "auto" }),
        );
      }
    }

    return () => {
      if (frame) window.cancelAnimationFrame(frame);
      if (stateStorage && intent !== "reset")
        stateStorage.setItem(storageKey, String(window.scrollY));
      window.history.scrollRestoration = previousRestoration;
    };
  }, [intent, key, storage]);
}
