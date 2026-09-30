import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  clearSession,
  loadSession,
  saveSession,
  restoreLocalSession,
  SESSION_EXPIRED_EVENT,
  type SessionState,
} from "./store";

/**
 * Session context + `useSession()` hook. Enrollment metadata survives a
 * returning visit, while the short-lived LocalSession stays in memory. The
 * provider restores it locally on mount so React surfaces re-render when the
 * session changes (signed in vs not).
 */
export interface SessionContextValue {
  session: SessionState;
  /** True once a locally minted owner session is present. */
  isOwner: boolean;
  /** Owner email for display, when known. */
  ownerEmail: string | null;
  /** Store / replace the ephemeral local session (and optional email). */
  setOwnerToken: (ownerToken: string, ownerEmail?: string | null) => void;
  /** Drop the local session; durable enrollment remains available. */
  clearOwnerToken: () => void;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<SessionState>(() => loadSession());

  useEffect(() => {
    let cancelled = false;
    void restoreLocalSession().then((restored) => {
      if (!cancelled && restored) setSession(restored);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const setOwnerToken = useCallback((ownerToken: string, ownerEmail: string | null = null) => {
    const next: SessionState = { ownerToken, ownerEmail };
    saveSession(next);
    setSession(next);
  }, []);

  const clearOwnerToken = useCallback(() => {
    clearSession();
    setSession({ ownerToken: null, ownerEmail: null });
  }, []);

  // The transport dispatches SESSION_EXPIRED_EVENT when a token-bearing
  // request comes back 401 (expired/revoked JWT). Clearing here flips the app
  // gate back to the sign-in screen instead of stranding the owner in a shell
  // where every panel errors "please sign in".
  useEffect(() => {
    const onExpired = () => clearOwnerToken();
    window.addEventListener(SESSION_EXPIRED_EVENT, onExpired);
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, onExpired);
  }, [clearOwnerToken]);

  const value = useMemo<SessionContextValue>(
    () => ({
      session,
      isOwner: Boolean(session.ownerToken),
      ownerEmail: session.ownerEmail,
      setOwnerToken,
      clearOwnerToken,
    }),
    [session, setOwnerToken, clearOwnerToken],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSession must be used within a <SessionProvider>");
  }
  return ctx;
}
