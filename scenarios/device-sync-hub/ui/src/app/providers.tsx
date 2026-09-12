import type { ReactNode } from "react";

import { ThemeProvider } from "../theme/ThemeProvider";
import { SessionProvider } from "../features/session/SessionProvider";
import { RealtimeProvider } from "../features/realtime/RealtimeProvider";

/**
 * Composes the cross-cutting providers a real `<App />` mount needs.
 *
 * QueryClient and i18n are wired in `main.tsx` (they own setup that has to run
 * before React mounts and during HMR). Session + Realtime live here, INSIDE the
 * QueryClient from main.tsx (realtime invalidates queries) and outside routing
 * so both the join screen and the paired shell share one session + SSE stream.
 * SessionProvider wraps RealtimeProvider because the stream is keyed to the
 * session's device token.
 */
export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <SessionProvider>
        <RealtimeProvider>{children}</RealtimeProvider>
      </SessionProvider>
    </ThemeProvider>
  );
}
