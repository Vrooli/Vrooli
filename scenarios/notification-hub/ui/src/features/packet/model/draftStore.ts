import type { AnswerPayload, ResponsePayload } from "./answer";
import { emptyDraft, normalizeDraft, type AttachmentNoteDraft, type PacketDraft } from "./draft";

/**
 * The payload of an answer the operator has sent (or tried to send). It is
 * immutable and scoped to the signed-in owner who submitted it: after an
 * uncertain response the page only ever reads the decision back, it never
 * replays this payload.
 */
export interface AskAttempt extends AnswerPayload {
  subject: string;
}

interface AskState {
  draft: PacketDraft;
  attempt?: AskAttempt;
}

const storagePrefix = "notification-hub:ask:v1:";
const states = new Map<string, AskState>();
const keyOf = (subject: string, askId: string) => `${subject}\u0000${askId}`;
const storageKeyOf = (subject: string, askId: string) =>
  `${storagePrefix}${encodeURIComponent(subject)}:${encodeURIComponent(askId)}`;

function storage(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.sessionStorage;
  } catch {
    // Private browsing policies can make sessionStorage unavailable. The
    // in-memory cache still protects route changes for the current page.
    return null;
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

function normalizeAttempt(value: unknown): AskAttempt | undefined {
  if (!isRecord(value) || typeof value.answer !== "string" || typeof value.note !== "string" || typeof value.subject !== "string") {
    return undefined;
  }
  const responses = Array.isArray(value.responses) ? (value.responses as ResponsePayload[]) : [];
  const attachmentNotes = Array.isArray(value.attachmentNotes) ? (value.attachmentNotes as AttachmentNoteDraft[]) : [];
  return { answer: value.answer, note: value.note, subject: value.subject, responses, attachmentNotes };
}

function normalizeState(value: unknown): AskState | null {
  if (!isRecord(value)) return null;
  const draft = normalizeDraft(value.draft);
  if (!draft) return null;
  return { draft, attempt: normalizeAttempt(value.attempt) };
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
        const state = normalizeState(JSON.parse(raw) as unknown);
        if (state) {
          states.set(key, state);
          return state;
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

export function readAskDraft(subject: string, askId: string): PacketDraft {
  return structuredClone(readState(subject, askId).draft);
}

export function writeAskDraft(subject: string, askId: string, draft: PacketDraft): void {
  const state = readState(subject, askId);
  writeState(subject, askId, { ...state, draft: structuredClone(draft) });
}

export function readAskAttempt(subject: string, askId: string): AskAttempt | null {
  const attempt = readState(subject, askId).attempt;
  return attempt?.subject === subject ? structuredClone(attempt) : null;
}

export function writeAskAttempt(subject: string, askId: string, attempt: AskAttempt): void {
  const state = readState(subject, askId);
  writeState(subject, askId, { ...state, attempt: structuredClone(attempt) });
}

/** Forget a sent answer but keep the draft, when the server confirms nothing was stored. */
export function clearAskAttempt(subject: string, askId: string): void {
  const state = readState(subject, askId);
  writeState(subject, askId, { draft: state.draft });
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
