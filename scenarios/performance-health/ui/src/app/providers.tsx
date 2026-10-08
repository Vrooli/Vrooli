import type { ReactNode } from "react";

import { ThemeProvider } from "../theme/ThemeProvider";
import { ScenarioProvider } from "../features/perf/ScenarioContext";

/**
 * Composes the cross-cutting providers a real `<App />` mount needs.
 *
 * QueryClient and i18n are wired in `main.tsx` (they own setup that has to run
 * before React mounts and during HMR). Anything that only needs context goes
 * here so tests and Storybook-style harnesses can re-use the same composition.
 *
 * `ScenarioProvider` owns the "current scenario" the per-scenario workflows act
 * on; it reads the picker list from ScanFleet, so it must sit inside the
 * QueryClientProvider that `main.tsx` mounts (it does — `<App />` is a child).
 */
export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ScenarioProvider>{children}</ScenarioProvider>
    </ThemeProvider>
  );
}
