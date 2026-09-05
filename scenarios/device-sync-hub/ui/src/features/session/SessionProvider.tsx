import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { Device } from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";
import { TrustState } from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

import {
  clearSession,
  loadSession,
  saveSession,
  emptySession,
  type SessionState,
} from "./store";

/**
 * Session context + `useSession()` hook. Backed by `localStorage` (via
 * `./store`) so a returning visit stays paired. The transport's `authedFetch`
 * reads credentials straight from storage; this context exists so React surfaces
 * re-render when the session changes (paired vs not, owner signed in vs not).
 */
export interface SessionContextValue {
  session: SessionState;
  /** True once this browser holds a device token (it has joined the trust group). */
  isPaired: boolean;
  /** True when this browser has requested access but is not trusted yet. */
  isPendingApproval: boolean;
  /** True once an owner JWT is present (device-management RPCs are reachable). */
  isOwner: boolean;
  /** Owner email for display, when known (login captures it; token-paste may not). */
  ownerEmail: string | null;
  /** Store the device token + device returned by a successful pairing. */
  setDeviceCredentials: (deviceToken: string, device: Device | null) => void;
  /** Store / replace the owner JWT (and optional email) for owner-gated device management. */
  setOwnerToken: (ownerToken: string, ownerEmail?: string | null) => void;
  /** Drop the owner JWT but keep the device paired. */
  clearOwnerToken: () => void;
  /** Forget everything (device + owner): this browser leaves the trust group locally. */
  signOut: () => void;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<SessionState>(() => loadSession());

  const setDeviceCredentials = useCallback(
    (deviceToken: string, device: Device | null) => {
      setSession((prev) => {
        const next = { ...prev, deviceToken, device };
        saveSession(next);
        return next;
      });
    },
    [],
  );

  const setOwnerToken = useCallback((ownerToken: string, ownerEmail: string | null = null) => {
    setSession((prev) => {
      const next = { ...prev, ownerToken, ownerEmail };
      saveSession(next);
      return next;
    });
  }, []);

  const clearOwnerToken = useCallback(() => {
    setSession((prev) => {
      const next = { ...prev, ownerToken: null, ownerEmail: null };
      saveSession(next);
      return next;
    });
  }, []);

  const signOut = useCallback(() => {
    clearSession();
    setSession(emptySession);
  }, []);

  const value = useMemo<SessionContextValue>(
    () => ({
      session,
      // A request-pairing response deliberately includes an inert token.  It is
      // not a session and must never unlock transfer UI before owner approval.
      isPaired: Boolean(
        session.deviceToken && session.device?.trustState === TrustState.TRUSTED,
      ),
      isPendingApproval: Boolean(
        session.deviceToken && session.device?.trustState === TrustState.PENDING,
      ),
      isOwner: Boolean(session.ownerToken),
      ownerEmail: session.ownerEmail,
      setDeviceCredentials,
      setOwnerToken,
      clearOwnerToken,
      signOut,
    }),
    [session, setDeviceCredentials, setOwnerToken, clearOwnerToken, signOut],
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
