import { createContext, useContext } from "react";

import type { SessionState } from "./store";

export interface SessionContextValue {
  session: SessionState;
  /** True once a locally minted owner session is present. */
  isOwner: boolean;
  /** Owner email for display, when known. */
  ownerEmail: string | null;
  /** Restoration gates sign-in until saved local identity is checked. */
  restoreStatus: "checking" | "ready" | "unavailable";
  /** Retry reading the existing local enrollment without contacting identity. */
  retryRestore: () => Promise<void>;
  /** Store / replace the ephemeral local session (and optional email). */
  setOwnerToken: (ownerToken: string, ownerEmail?: string | null) => void;
  /** Drop the local session; durable enrollment remains available. */
  clearOwnerToken: () => void;
}

export const SessionContext = createContext<SessionContextValue | null>(null);

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSession must be used within a <SessionProvider>");
  }
  return ctx;
}
