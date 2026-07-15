/**
 * Persisted session shape. The bridge console has exactly one credential: the
 * owner JWT (there is no per-device token — a browser is the owner's console,
 * not a fleet node). The owner token authorizes every owner-gated fleet RPC
 * (registry, dispatch, onboard, provision, …); `api/client.ts` reads it fresh
 * per request and attaches it as `Authorization: Bearer`.
 */
export interface SessionState {
  ownerToken: string | null;
  /** Owner email for display ("signed in as …"); best-effort, may be null. */
  ownerEmail: string | null;
}

const STORAGE_KEY = "vrooli-bridge.session";

interface StoredSession {
  ownerToken?: string | null;
  ownerEmail?: string | null;
}

export const emptySession: SessionState = {
  ownerToken: null,
  ownerEmail: null,
};

const safeStorage = (): Storage | null => {
  try {
    return typeof window !== "undefined" ? window.localStorage : null;
  } catch {
    // Access can throw in privacy modes / sandboxed iframes.
    return null;
  }
};

/**
 * Read the full persisted session. Tolerant of malformed/legacy payloads:
 * anything unparseable resolves to the empty session rather than throwing, so a
 * corrupt entry can never wedge the app at boot.
 */
export function loadSession(): SessionState {
  const storage = safeStorage();
  if (!storage) return emptySession;
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return emptySession;
  try {
    const parsed = JSON.parse(raw) as StoredSession;
    return {
      ownerToken: parsed.ownerToken ?? null,
      ownerEmail: parsed.ownerEmail ?? null,
    };
  } catch {
    return emptySession;
  }
}

export function saveSession(state: SessionState): void {
  const storage = safeStorage();
  if (!storage) return;
  const stored: StoredSession = {
    ownerToken: state.ownerToken,
    ownerEmail: state.ownerEmail,
  };
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(stored));
  } catch {
    // Quota / disabled storage — non-fatal; the in-memory context still holds.
  }
}

export function clearSession(): void {
  const storage = safeStorage();
  if (!storage) return;
  try {
    storage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}

/**
 * Read just the owner token, fresh from storage. `authedFetch` calls this on
 * every request so a sign-in mid-session is picked up without rebuilding the
 * Connect transport.
 */
export function readOwnerToken(): string | null {
  return loadSession().ownerToken;
}

/**
 * Window event dispatched when a request that carried the owner token came
 * back 401 — the token is expired or revoked. The transport (`api/client.ts`)
 * dispatches it; `SessionProvider` listens and clears the session, so the app
 * gate returns the user to the sign-in screen instead of stranding them in a
 * shell where every panel errors.
 */
export const SESSION_EXPIRED_EVENT = "vrooli-bridge:session-expired";

export function notifySessionExpired(): void {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT));
}
