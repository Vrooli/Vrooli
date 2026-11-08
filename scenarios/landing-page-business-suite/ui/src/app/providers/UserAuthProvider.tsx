import { useState, useEffect, useCallback, useRef, ReactNode } from 'react';
import { getUserMe, userLogout, refreshSessionOnce, isApiError, type UserAuthUser } from '../../shared/api';
import { UserAuthContext } from './UserAuthContext';

/** Cross-tab channel: another tab's sign-out clears this tab's private state. */
const SESSION_EVENT_KEY = 'lpbs.auth.session-event';

function hasSessionHint(): boolean {
  return document.cookie.split(';').some((cookie) => {
    const name = cookie.trim().split('=', 1)[0];
    return name === 'lpbs_session_hint' || name === '__Host-lpbs_session_hint';
  });
}

/** A read that a newer read or a sign-out made irrelevant. */
const SUPERSEDED = Symbol('superseded');

export function UserAuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<UserAuthUser | null>(null);
  const [isSessionLoading, setIsSessionLoading] = useState(hasSessionHint);
  // Every session read and every sign-out takes a new generation. A read only
  // writes state while it is still the latest generation, so a late response
  // can never restore an account after sign-out or overwrite a newer read.
  const generation = useRef(0);
  const authenticatedRef = useRef(false);

  const clearLocalSession = useCallback(() => {
    generation.current += 1;
    authenticatedRef.current = false;
    setIsAuthenticated(false);
    setUser(null);
    setIsSessionLoading(false);
  }, []);

  const checkSession = useCallback(async (): Promise<UserAuthUser | null | typeof SUPERSEDED> => {
    const read = ++generation.current;
    const current = () => read === generation.current;
    setIsSessionLoading(true);
    let resolved: UserAuthUser | null = null;
    try {
      resolved = (await getUserMe()).user;
    } catch (err) {
      if (isApiError(err, 'unauthorized')) {
        try {
          await refreshSessionOnce();
          resolved = (await getUserMe()).user;
        } catch {
          // Refresh failed; the session is not authorized.
        }
      }
    }
    if (!current()) return SUPERSEDED;
    authenticatedRef.current = resolved !== null;
    setIsAuthenticated(resolved !== null);
    setUser(resolved);
    setIsSessionLoading(false);
    return resolved;
  }, []);

  // Only check on mount when the non-authoritative presence hint exists, so a
  // signed-out visitor does not trigger an unauthenticated request.
  useEffect(() => {
    if (hasSessionHint()) {
      void checkSession();
    }
  }, [checkSession]);

  useEffect(() => {
    // Returning to the tab re-asks the server whether a shown session is still
    // authorized. A tab that is signed out locally stays signed out.
    const onFocus = () => {
      if (authenticatedRef.current) void checkSession();
    };
    // Another tab signed out: drop private state without treating the event
    // as proof of any session.
    const onStorage = (event: StorageEvent) => {
      if (event.key !== SESSION_EVENT_KEY || !event.newValue) return;
      try {
        if ((JSON.parse(event.newValue) as { kind?: string }).kind === 'signed-out') clearLocalSession();
      } catch {
        // A malformed event is ignored.
      }
    };
    window.addEventListener('focus', onFocus);
    window.addEventListener('storage', onStorage);
    return () => {
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('storage', onStorage);
    };
  }, [checkSession, clearLocalSession]);

  const logout = useCallback(async () => {
    // Clear first so an in-flight read cannot land after sign-out.
    clearLocalSession();
    try {
      window.localStorage.setItem(SESSION_EVENT_KEY, JSON.stringify({ kind: 'signed-out', nonce: `${Date.now()}-${Math.random()}` }));
    } catch {
      // Storage may be unavailable; this tab is still signed out.
    }
    try {
      await userLogout();
    } catch (err) {
      console.error('Logout failed:', err);
    }
  }, [clearLocalSession]);

  const refreshSession = useCallback(async (options?: { requireAuthenticated?: boolean; expectedEmail?: string }) => {
    const authenticated = await checkSession();
    if (authenticated === SUPERSEDED) {
      // A newer read or a sign-out owns the session state now.
      if (options?.requireAuthenticated) throw new Error('We couldn’t confirm your signed-in session. Please try again.');
      return;
    }
    const wrongAccount = authenticated && options?.expectedEmail && authenticated.email.toLowerCase() !== options.expectedEmail.toLowerCase();
    if (wrongAccount) {
      clearLocalSession();
    }
    if (options?.requireAuthenticated && (!authenticated || wrongAccount)) {
      throw new Error('We couldn’t confirm your signed-in session. Please try again.');
    }
  }, [checkSession, clearLocalSession]);

  return (
    <UserAuthContext.Provider
      value={{
        isAuthenticated,
        isSessionLoading,
        user,
        logout,
        refreshSession,
      }}
    >
      {children}
    </UserAuthContext.Provider>
  );
}
