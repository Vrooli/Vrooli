import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDraftPersistence } from "./useDraftPersistence";

/**
 * Why a change carries a `reason`:
 *   - "input" comes from a bound textarea's own onChange. The surface that
 *     fired it already holds the value, so subscribers only reseed OTHER
 *     surfaces (peer-sync) — never clobber the caret of the textarea being
 *     typed in.
 *   - "set" / "reset" are programmatic (voice insertion, clear, session
 *     reload). Every subscriber reseeds its DOM and restores the caret.
 */
export type ComposerDraftChangeReason = "input" | "set" | "reset";

export interface ComposerDraftChange {
  value: string;
  /** Caret position to restore when known (programmatic set/reset). */
  caret: number | null;
  reason: ComposerDraftChangeReason;
}

export interface ComposerDraft {
  /** Read the current live draft value (the mirror is the single source of truth). */
  getValue(): string;
  /** Value at mount, for seeding an uncontrolled textarea's `defaultValue`. */
  readonly initialValue: string;
  /** onChange handler for a bound uncontrolled textarea. */
  handleChange(el: HTMLTextAreaElement): void;
  /** onSelect/onBlur handler that records the caret for later caret-aware inserts. */
  trackSelection(el: HTMLTextAreaElement): void;
  /** Programmatically set the value (voice/session reload); `caret` defaults to end. */
  setValue(value: string, caret?: number): void;
  /**
   * Clear the draft and its persistence. Defaults to the session the draft is
   * currently bound to. Pass a session (from `getSessionId()` at the time the
   * work started) for a clear that may land late — a send whose ack settles
   * after the user switched sessions must clear the session it was sent from,
   * never whatever session is now on screen.
   */
  reset(session?: string | null): void;
  /** The session this draft is bound to right now. */
  getSessionId(): string | null;
  /** Insert text at the caret (focused draft textarea, else last-known selection). */
  appendAtCaret(text: string): void;
  /** Subscribe to value changes. Returns an unsubscribe fn. */
  subscribe(cb: (change: ComposerDraftChange) => void): () => void;
}

/**
 * useComposerDraft — the single owner of a session's text draft.
 *
 * The draft was previously private to MobileToolbar. Hoisting it into a shared
 * hook lets the collapsed toolbar input and the full-screen composer read and
 * write ONE value that cannot diverge: whichever surface is typed into updates
 * the mirror + persistence and notifies the others to reseed.
 *
 * Like useDraftPersistence, this is deliberately imperative: the value lives in
 * a ref (not React state) so typing into an uncontrolled textarea never
 * re-renders — that controlled round-trip was the historical cause of mobile
 * typing lag. Consumers that MUST re-render on the draft (e.g. the AI-suggest
 * bar) opt in via `subscribe`.
 */
export function useComposerDraft(sessionId?: string | null): ComposerDraft {
  const { readDraft, persistDraft, flushDraft, clearDraft } = useDraftPersistence(sessionId);

  // The session bound right now, readable from async callbacks that may fire
  // after a switch. Tracked during render so it is never a render behind.
  const sessionRef = useRef(sessionId);
  sessionRef.current = sessionId;

  // Seed the mirror from persistence exactly once.
  const initialRef = useRef<string | null>(null);
  if (initialRef.current === null) initialRef.current = readDraft();
  const valueRef = useRef(initialRef.current);
  const selectionRef = useRef<{ start: number; end: number } | null>(null);
  const subscribersRef = useRef(new Set<(change: ComposerDraftChange) => void>());

  const notify = useCallback((change: ComposerDraftChange) => {
    for (const cb of subscribersRef.current) cb(change);
  }, []);

  const getValue = useCallback(() => valueRef.current, []);

  const handleChange = useCallback((el: HTMLTextAreaElement) => {
    const value = el.value;
    valueRef.current = value;
    selectionRef.current = {
      start: el.selectionStart ?? value.length,
      end: el.selectionEnd ?? value.length,
    };
    persistDraft(value);
    notify({ value, caret: selectionRef.current.end, reason: "input" });
  }, [persistDraft, notify]);

  const trackSelection = useCallback((el: HTMLTextAreaElement) => {
    selectionRef.current = {
      start: el.selectionStart ?? el.value.length,
      end: el.selectionEnd ?? el.value.length,
    };
  }, []);

  const setValue = useCallback((value: string, caret?: number) => {
    valueRef.current = value;
    const pos = caret ?? value.length;
    selectionRef.current = { start: pos, end: pos };
    persistDraft(value);
    notify({ value, caret: pos, reason: "set" });
  }, [persistDraft, notify]);

  const getSessionId = useCallback(() => sessionRef.current ?? null, []);

  const reset = useCallback((session?: string | null) => {
    const target = session === undefined ? sessionRef.current : session;
    if (target !== sessionRef.current) {
      // A late clear for a session we have since left (a send ack that settled
      // after the switch). Drop only that session's stored draft — touching the
      // live value here would wipe the draft of the session now on screen.
      clearDraft(target);
      return;
    }
    valueRef.current = "";
    selectionRef.current = null;
    clearDraft(target);
    notify({ value: "", caret: 0, reason: "reset" });
  }, [clearDraft, notify]);

  const appendAtCaret = useCallback((text: string) => {
    const prev = valueRef.current;
    let start = prev.length;
    let end = prev.length;
    // Prefer the live caret of a focused DRAFT textarea (its value mirrors the
    // draft, which is how we know it's ours). Otherwise fall back to the
    // last-known selection; otherwise append to the end.
    const active = typeof document !== "undefined" ? document.activeElement : null;
    if (
      active instanceof HTMLTextAreaElement &&
      active.value === prev &&
      active.selectionStart !== null &&
      active.selectionEnd !== null
    ) {
      start = Math.min(active.selectionStart, prev.length);
      end = Math.min(active.selectionEnd, prev.length);
    } else if (selectionRef.current) {
      start = Math.min(selectionRef.current.start, prev.length);
      end = Math.min(selectionRef.current.end, prev.length);
    }
    const before = prev.slice(0, start);
    const after = prev.slice(end);
    const lead = before.length > 0 && !/\s$/.test(before) && !/^\s/.test(text) ? " " : "";
    const trail = after.length > 0 && !/^\s/.test(after) && !/\s$/.test(text) ? " " : "";
    const insertEnd = before.length + lead.length + text.length;
    setValue(before + lead + text + trail + after, insertEnd);
  }, [setValue]);

  const subscribe = useCallback((cb: (change: ComposerDraftChange) => void) => {
    subscribersRef.current.add(cb);
    return () => {
      subscribersRef.current.delete(cb);
    };
  }, []);

  // Reload the persisted draft when the session changes (skip initial mount,
  // which is already seeded above).
  const prevSessionRef = useRef(sessionId);
  useEffect(() => {
    if (prevSessionRef.current === sessionId) return;
    // Make the outgoing session's draft durable BEFORE loading the new one.
    // Its last keystrokes may still be sitting in the debounce, and the user
    // typing (or sending) in the session they just switched to would otherwise
    // race it — which is how switching sessions lost the draft you left behind.
    flushDraft(prevSessionRef.current, valueRef.current);
    prevSessionRef.current = sessionId;
    const draft = readDraft();
    valueRef.current = draft;
    selectionRef.current = null;
    notify({ value: draft, caret: draft.length, reason: "set" });
  }, [sessionId, readDraft, notify, flushDraft]);

  const initialValue = initialRef.current ?? "";
  return useMemo(
    () => ({
      getValue,
      initialValue,
      handleChange,
      trackSelection,
      setValue,
      reset,
      getSessionId,
      appendAtCaret,
      subscribe,
    }),
    [getValue, initialValue, handleChange, trackSelection, setValue, reset, getSessionId, appendAtCaret, subscribe],
  );
}
