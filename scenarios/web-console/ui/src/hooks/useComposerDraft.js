import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDraftPersistence } from "./useDraftPersistence";
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
export function useComposerDraft(sessionId) {
    const { readDraft, persistDraft, clearDraft } = useDraftPersistence(sessionId);
    // Seed the mirror from persistence exactly once.
    const initialRef = useRef(null);
    if (initialRef.current === null)
        initialRef.current = readDraft();
    const valueRef = useRef(initialRef.current);
    const selectionRef = useRef(null);
    const subscribersRef = useRef(new Set());
    const notify = useCallback((change) => {
        for (const cb of subscribersRef.current)
            cb(change);
    }, []);
    const getValue = useCallback(() => valueRef.current, []);
    const handleChange = useCallback((el) => {
        const value = el.value;
        valueRef.current = value;
        selectionRef.current = {
            start: el.selectionStart ?? value.length,
            end: el.selectionEnd ?? value.length,
        };
        persistDraft(value);
        notify({ value, caret: selectionRef.current.end, reason: "input" });
    }, [persistDraft, notify]);
    const trackSelection = useCallback((el) => {
        selectionRef.current = {
            start: el.selectionStart ?? el.value.length,
            end: el.selectionEnd ?? el.value.length,
        };
    }, []);
    const setValue = useCallback((value, caret) => {
        valueRef.current = value;
        const pos = caret ?? value.length;
        selectionRef.current = { start: pos, end: pos };
        persistDraft(value);
        notify({ value, caret: pos, reason: "set" });
    }, [persistDraft, notify]);
    const reset = useCallback(() => {
        valueRef.current = "";
        selectionRef.current = null;
        clearDraft();
        notify({ value: "", caret: 0, reason: "reset" });
    }, [clearDraft, notify]);
    const appendAtCaret = useCallback((text) => {
        const prev = valueRef.current;
        let start = prev.length;
        let end = prev.length;
        // Prefer the live caret of a focused DRAFT textarea (its value mirrors the
        // draft, which is how we know it's ours). Otherwise fall back to the
        // last-known selection; otherwise append to the end.
        const active = typeof document !== "undefined" ? document.activeElement : null;
        if (active instanceof HTMLTextAreaElement &&
            active.value === prev &&
            active.selectionStart !== null &&
            active.selectionEnd !== null) {
            start = Math.min(active.selectionStart, prev.length);
            end = Math.min(active.selectionEnd, prev.length);
        }
        else if (selectionRef.current) {
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
    const subscribe = useCallback((cb) => {
        subscribersRef.current.add(cb);
        return () => {
            subscribersRef.current.delete(cb);
        };
    }, []);
    // Reload the persisted draft when the session changes (skip initial mount,
    // which is already seeded above).
    const prevSessionRef = useRef(sessionId);
    useEffect(() => {
        if (prevSessionRef.current === sessionId)
            return;
        prevSessionRef.current = sessionId;
        const draft = readDraft();
        valueRef.current = draft;
        selectionRef.current = null;
        notify({ value: draft, caret: draft.length, reason: "set" });
    }, [sessionId, readDraft, notify]);
    const initialValue = initialRef.current ?? "";
    return useMemo(() => ({
        getValue,
        initialValue,
        handleChange,
        trackSelection,
        setValue,
        reset,
        appendAtCaret,
        subscribe,
    }), [getValue, initialValue, handleChange, trackSelection, setValue, reset, appendAtCaret, subscribe]);
}
