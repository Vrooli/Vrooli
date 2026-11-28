import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  clearSession,
  loadInitialSession,
  saveSession,
  restoreLocalSession,
  SESSION_EXPIRED_EVENT,
  type SessionState,
} from "./store";
import { SessionContext, type SessionContextValue } from "./sessionContext";

/**
 * Session context + `useSession()` hook. Enrollment metadata survives a
 * returning visit, while the short-lived LocalSession stays in memory. The
 * provider restores it locally on mount so React surfaces re-render when the
 * session changes (signed in vs not).
 */
export function SessionProvider({ children }: { children: ReactNode }) {
  const [initial] = useState(loadInitialSession);
  const [session, setSession] = useState<SessionState>(initial.session);
  const [restoreStatus, setRestoreStatus] = useState<SessionContextValue["restoreStatus"]>(initial.restoreStatus);

  const restore = useCallback(async (isCurrent: () => boolean, markChecking = true) => {
    if (markChecking) setRestoreStatus("checking");
    try {
      const restored = await restoreLocalSession();
      if (!isCurrent()) return;
      if (restored) setSession(restored);
      setRestoreStatus("ready");
    } catch {
      if (isCurrent()) setRestoreStatus("unavailable");
    }
  }, []);
  const retryRestore = useCallback(() => restore(() => true), [restore]);

  useEffect(() => {
    if (initial.restoreStatus !== "checking") return;
    let cancelled = false;
    void restore(() => !cancelled, false);
    return () => {
      cancelled = true;
    };
  }, [initial.restoreStatus, restore]);

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
      restoreStatus,
      retryRestore,
      setOwnerToken,
      clearOwnerToken,
    }),
    [session, restoreStatus, retryRestore, setOwnerToken, clearOwnerToken],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}
