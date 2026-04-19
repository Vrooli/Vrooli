import { useEffect, type ReactNode } from "react";

export type ThemeKey =
  | "ground-control"
  | "bioluminescent"
  | "foundry"
  | "vault"
  | "signal-tower"
  | "cosmos";

interface ThemeProviderProps {
  themeKey: ThemeKey;
  children: ReactNode;
}

// Map themeKey -> dynamic CSS loader. Using an object literal rather than a
// string template keeps the list explicit and static-analysis friendly.
const themeLoaders: Record<ThemeKey, () => Promise<unknown>> = {
  "ground-control": () => import("../themes/ground-control.css"),
  bioluminescent: () => import("../themes/bioluminescent.css"),
  foundry: () => import("../themes/foundry.css"),
  vault: () => import("../themes/vault.css"),
  "signal-tower": () => import("../themes/signal-tower.css"),
  cosmos: () => import("../themes/cosmos.css"),
};

/**
 * ThemeProvider loads the requested theme's CSS on mount and mirrors the
 * themeKey onto the <html> element via `data-theme` so CSS variable rules
 * cascade across the whole document tree.
 */
export function ThemeProvider({ themeKey, children }: ThemeProviderProps) {
  useEffect(() => {
    // Kick off the dynamic import; swallow failures so a missing theme
    // stylesheet never breaks the page render.
    const loader = themeLoaders[themeKey];
    void loader().catch(() => {
      /* theme stylesheet failed to load — fall back to defaults */
    });

    const root = document.documentElement;
    const previous = root.getAttribute("data-theme");
    root.setAttribute("data-theme", themeKey);

    return () => {
      if (previous === null) {
        root.removeAttribute("data-theme");
      } else {
        root.setAttribute("data-theme", previous);
      }
    };
  }, [themeKey]);

  return <>{children}</>;
}
