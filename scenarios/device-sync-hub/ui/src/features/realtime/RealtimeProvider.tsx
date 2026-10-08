import { createContext, useContext, type ReactNode } from "react";

import { useSession } from "../session/SessionProvider";
import { useRealtime, type UseRealtimeResult } from "./useRealtime";

/**
 * App-level realtime context so multiple surfaces (the top-bar connection
 * indicator, the devices presence dots, the pairing banner) share ONE SSE
 * connection rather than each opening its own. The stream is keyed to the
 * paired device token; when unpaired it's simply closed.
 */
const RealtimeContext = createContext<UseRealtimeResult | null>(null);

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { session } = useSession();
  const realtime = useRealtime(session.deviceToken);
  return <RealtimeContext.Provider value={realtime}>{children}</RealtimeContext.Provider>;
}

export function useRealtimeContext(): UseRealtimeResult {
  const ctx = useContext(RealtimeContext);
  if (!ctx) {
    throw new Error("useRealtimeContext must be used within a <RealtimeProvider>");
  }
  return ctx;
}
