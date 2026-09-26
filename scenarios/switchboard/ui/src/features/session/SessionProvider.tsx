import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { ConsoleApiError } from "../../api/console";
import { session as sessionStore, type Session } from "../../api/session";
import { SignInDialog } from "./SignInDialog";

interface SessionContextValue {
  session: Session | null;
  /** Opens the sign-in dialog and resolves true once a session exists. */
  requireSession: () => Promise<boolean>;
  signOut: () => void;
  /**
   * Runs a write. On 401 it asks for a sign-in and retries once, so every
   * owner action shares one path and no page hand-rolls the prompt.
   */
  withSession: <T>(action: () => Promise<T>) => Promise<T>;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [current, setCurrent] = useState<Session | null>(() => sessionStore.get());
  const [open, setOpen] = useState(false);
  const waiters = useRef<Array<(ok: boolean) => void>>([]);

  useEffect(() => sessionStore.subscribe(setCurrent), []);

  const settle = useCallback((ok: boolean) => {
    const pending = waiters.current;
    waiters.current = [];
    pending.forEach((resolve) => resolve(ok));
  }, []);

  const requireSession = useCallback((): Promise<boolean> => {
    if (sessionStore.get()) return Promise.resolve(true);
    setOpen(true);
    return new Promise<boolean>((resolve) => waiters.current.push(resolve));
  }, []);

  const withSession = useCallback(
    async <T,>(action: () => Promise<T>): Promise<T> => {
      try {
        return await action();
      } catch (error) {
        if (error instanceof ConsoleApiError && error.status === 401) {
          if (current) sessionStore.clear();
          const ok = await requireSession();
          if (ok) return action();
        }
        throw error;
      }
    },
    [current, requireSession],
  );

  const signOut = useCallback(() => sessionStore.clear(), []);

  const value = useMemo<SessionContextValue>(() => ({ session: current, requireSession, signOut, withSession }), [current, requireSession, signOut, withSession]);

  return (
    <SessionContext.Provider value={value}>
      {children}
      {open ? (
        <SignInDialog
          onSignedIn={() => {
            setOpen(false);
            settle(true);
          }}
          onClose={() => {
            setOpen(false);
            settle(false);
          }}
        />
      ) : null}
    </SessionContext.Provider>
  );
}

const DETACHED: SessionContextValue = {
  session: null,
  requireSession: () => Promise.resolve(false),
  signOut: () => undefined,
  withSession: (action) => action(),
};

/** Outside a provider (isolated component tests) writes run without a sign-in prompt. */
export function useSession(): SessionContextValue {
  return useContext(SessionContext) ?? DETACHED;
}
