export interface AskDraft {
  answer: string;
  note: string;
}

export interface AskAttempt extends AskDraft {
  /** The immutable payload is scoped to the owner who submitted it. */
  subject: string;
}

interface AskState {
  draft: AskDraft;
  attempt?: AskAttempt;
}

const storagePrefix = "notification-hub:ask:v1:";
const states = new Map<string, AskState>();
const keyOf = (subject: string, askId: string) => `${subject}\u0000${askId}`;
const storageKeyOf = (subject: string, askId: string) =>
  `${storagePrefix}${encodeURIComponent(subject)}:${encodeURIComponent(askId)}`;
const emptyDraft = (): AskDraft => ({ answer: "", note: "" });

function storage(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.sessionStorage;
  } catch {
    // Private browsing policies can make sessionStorage unavailable. The
    // in-memory cache still protects route changes for the current page.
    return null;
  }
}

function validState(value: unknown): value is AskState {
  if (!value || typeof value !== "object") return false;
  const state = value as Partial<AskState>;
  return Boolean(
    state.draft &&
      typeof state.draft.answer === "string" &&
      typeof state.draft.note === "string" &&
      (!state.attempt ||
        (typeof state.attempt.answer === "string" &&
          typeof state.attempt.note === "string" &&
          typeof state.attempt.subject === "string")),
  );
}

function readState(subject: string, askId: string): AskState {
  if (!subject || !askId) return { draft: emptyDraft() };
  const key = keyOf(subject, askId);
  const cached = states.get(key);
  if (cached) return cached;
  const store = storage();
  if (store) {
    try {
      const raw = store.getItem(storageKeyOf(subject, askId));
      if (raw) {
        const value: unknown = JSON.parse(raw);
        if (validState(value)) {
          states.set(key, value);
          return value;
        }
        store.removeItem(storageKeyOf(subject, askId));
      }
    } catch {
      // A corrupt or blocked local draft must not prevent reading the ask.
    }
  }
  const empty = { draft: emptyDraft() };
  states.set(key, empty);
  return empty;
}

function writeState(subject: string, askId: string, state: AskState): void {
  if (!subject || !askId) return;
  states.set(keyOf(subject, askId), state);
  const store = storage();
  if (!store) return;
  try {
    store.setItem(storageKeyOf(subject, askId), JSON.stringify(state));
  } catch {
    // Keep route continuity in memory when browser storage is unavailable.
  }
}

export function readAskDraft(subject: string, askId: string): AskDraft {
  return { ...readState(subject, askId).draft };
}

export function writeAskDraft(
  subject: string,
  askId: string,
  draft: AskDraft,
): void {
  const state = readState(subject, askId);
  writeState(subject, askId, { ...state, draft: { ...draft } });
}

export function readAskAttempt(
  subject: string,
  askId: string,
): AskAttempt | null {
  const attempt = readState(subject, askId).attempt;
  return attempt?.subject === subject ? { ...attempt } : null;
}

export function writeAskAttempt(
  subject: string,
  askId: string,
  attempt: AskAttempt,
): void {
  const state = readState(subject, askId);
  writeState(subject, askId, { ...state, attempt: { ...attempt } });
}

export function clearAskDraft(subject: string, askId: string): void {
  if (!subject || !askId) return;
  states.delete(keyOf(subject, askId));
  try {
    storage()?.removeItem(storageKeyOf(subject, askId));
  } catch {
    // Clearing one owner's draft is best effort if browser storage is blocked.
  }
}

export function clearAskDrafts(): void {
  states.clear();
  const store = storage();
  if (!store) return;
  try {
    for (let index = store.length - 1; index >= 0; index -= 1) {
      const key = store.key(index);
      if (key?.startsWith(storagePrefix)) store.removeItem(key);
    }
  } catch {
    // This helper is also used by tests; a blocked store is already empty
    // from the current page's point of view.
  }
}
