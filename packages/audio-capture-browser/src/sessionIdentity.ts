export interface RecoverableSessionIdentity {
  sessionId: string;
  resumeToken: string;
}

const STORAGE_KEY = "vrooli.audio-capture-browser.unfinished-session.v1";

/** Persist only opaque recovery identity; journal bytes stay in IndexedDB. */
export function rememberUnfinishedSession(identity: RecoverableSessionIdentity): void {
  try {
    globalThis.sessionStorage?.setItem(STORAGE_KEY, JSON.stringify(identity));
  } catch {
    // Persistence unavailability is surfaced by the journal's durability
    // status. Identity loss means a reload cannot resume, never silent reuse.
  }
}

export function loadUnfinishedSession(): RecoverableSessionIdentity | null {
  try {
    const raw = globalThis.sessionStorage?.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) return null;
    const value = parsed as Partial<RecoverableSessionIdentity>;
    return typeof value.sessionId === "string" && value.sessionId !== "" && typeof value.resumeToken === "string" && value.resumeToken !== ""
      ? { sessionId: value.sessionId, resumeToken: value.resumeToken }
      : null;
  } catch {
    return null;
  }
}

export function forgetUnfinishedSession(): void {
  try {
    globalThis.sessionStorage?.removeItem(STORAGE_KEY);
  } catch {
    // Best effort; an empty/terminal journal remains harmless if this fails.
  }
}
