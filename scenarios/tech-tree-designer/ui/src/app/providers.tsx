import type { ReactNode } from "react";

import { ThemeProvider } from "../theme/ThemeProvider";

/**
 * Composes the cross-cutting providers a real `<App />` mount needs.
 *
 * QueryClient and i18n are wired in `main.tsx` (they own setup that has to run
 * before React mounts and during HMR). Anything that only needs context goes
 * here so tests and Storybook-style harnesses can re-use the same composition.
 */
export function Providers({ children }: { children: ReactNode }) {
  return <ThemeProvider>{children}</ThemeProvider>;
}
