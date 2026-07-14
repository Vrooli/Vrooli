/**
 * Composer-draft persistence shared by the session detail page (restore +
 * autosave) and the attach-to-session sheet (prefilling a chosen starter
 * prompt into a session that was just drafted).
 */

export function sessionDraftStorageKey(sessionId: string): string {
  return `swarm-session-draft:${sessionId}`;
}

export function readSessionDraft(sessionId: string): string {
  try {
    return localStorage.getItem(sessionDraftStorageKey(sessionId)) ?? "";
  } catch {
    return "";
  }
}

/** Empty text clears the stored draft. */
export function writeSessionDraft(sessionId: string, text: string): void {
  try {
    if (text) {
      localStorage.setItem(sessionDraftStorageKey(sessionId), text);
    } else {
      localStorage.removeItem(sessionDraftStorageKey(sessionId));
    }
  } catch {
    // Storage loss should not block the operator.
  }
}
