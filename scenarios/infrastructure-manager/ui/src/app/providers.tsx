import type { ReactNode } from "react";

/**
 * Composes the cross-cutting providers a real `<App />` mount needs.
 *
 * QueryClient and i18n are wired in `main.tsx` (they own setup that has to run
 * before React mounts and during HMR). Anything that only needs context goes
 * here so tests and Storybook-style harnesses can re-use the same composition.
 *
 * There is no theme provider: this scenario commits to one visual world, so
 * there is no theme state to hold. See `DESIGN.md`.
 */
export function Providers({ children }: { children: ReactNode }) {
  return <>{children}</>;
}
