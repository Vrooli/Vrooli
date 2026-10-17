/**
 * Browser-side session state. A browser never persists an authenticator JWT:
 * it keeps only enrollment metadata and an Ed25519 private key, then mints a
 * short-lived Bridge LocalSession in memory. The key is the browser analogue
 * of the CLI's per-user local enrollment store.
 */
import { loadBrowserEnrollment, mintBrowserSession } from "./browser_session";

export interface SessionState {
  /** Ephemeral OS1 LocalSession; never written to browser storage. */
  ownerToken: string | null;
  /** Owner email for display (best-effort, non-secret). */
  ownerEmail: string | null;
}

const DISPLAY_STORAGE_KEY = "vrooli-bridge.session";

interface StoredEnrollmentOwner {
  ownerEmail?: string | null;
}

export const emptySession: SessionState = {
  ownerToken: null,
  ownerEmail: null,
};

let volatileToken: string | null = null;
let bootstrapToken: string | null = null;

const safeStorage = (): Storage | null => {
  try {
    return typeof window !== "undefined" ? window.localStorage : null;
  } catch {
    return null;
  }
};

function storedOwnerEmail(): string | null {
  const storage = safeStorage();
  if (!storage) return null;
  const raw = storage.getItem(DISPLAY_STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as StoredEnrollmentOwner;
    return parsed.ownerEmail ?? null;
  } catch {
    return null;
  }
}

/** Read the in-memory session. Persistent enrollment is restored explicitly. */
export function loadSession(): SessionState {
  const target = safeStorage();
  if (target && !target.getItem(DISPLAY_STORAGE_KEY) && !target.getItem("vrooli-bridge.operator-session")) {
    volatileToken = null;
  }
  return { ownerToken: volatileToken, ownerEmail: storedOwnerEmail() };
}

/**
 * Test and runtime seam for replacing only the ephemeral credential. The
 * owner token is deliberately not serialized.
 */
export function saveSession(state: SessionState): void {
  volatileToken = state.ownerToken;
  const storage = safeStorage();
  if (!storage) return;
  try {
    const previous = storage.getItem(DISPLAY_STORAGE_KEY);
    const ownerEmail = state.ownerEmail ?? (previous ? storedOwnerEmail() : null);
    storage.setItem(DISPLAY_STORAGE_KEY, JSON.stringify({ ownerEmail } satisfies StoredEnrollmentOwner));
  } catch {
    // Quota / disabled storage — the in-memory session remains usable.
  }
}

/** Clear the ephemeral session but keep the durable enrollment for re-entry. */
export function clearSession(): void {
  volatileToken = null;
  bootstrapToken = null;
  const target = safeStorage();
  if (target) {
    try {
      target.removeItem(DISPLAY_STORAGE_KEY);
    } catch {
      // ignore
    }
  }
}

/** Read a credential for the fetch wrapper; bootstrap is one-RPC enrollment only. */
export function readOwnerToken(): string | null {
  return bootstrapToken ?? volatileToken;
}

export function setEnrollmentBootstrapToken(token: string): void {
  bootstrapToken = token;
}

export function clearEnrollmentBootstrapToken(): void {
  bootstrapToken = null;
}

/** Restore and mint a local session without contacting the identity provider. */
export async function restoreLocalSession(): Promise<SessionState | null> {
  if (volatileToken) return null;
  const enrollment = await loadBrowserEnrollment();
  if (!enrollment) return null;
  try {
    volatileToken = await mintBrowserSession(enrollment);
  } catch {
    volatileToken = null;
  }
  return loadSession();
}

/**
 * Window event dispatched when a local credential is rejected. This clears
 * only the ephemeral token; a later request can mint again while enrollment
 * remains valid, and explicit revocation can remove the enrollment record.
 */
export const SESSION_EXPIRED_EVENT = "vrooli-bridge:session-expired";

export function notifySessionExpired(): void {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT));
}
