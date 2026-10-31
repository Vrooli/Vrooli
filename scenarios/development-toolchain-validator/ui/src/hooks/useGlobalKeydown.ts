import { useEffect, useRef } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/**
 * Single central keyboard shortcut hook.
 *
 * Per `ui-health`: one shortcut manager per app shell. All
 * `document.addEventListener("keydown", …)` calls outside this hook are a
 * regression — route shortcut handling through here instead.
 *
 * Handlers receive the active sequence (the buffer of keys pressed within
 * the chord window) and the triggering event. Return `true` to consume the
 * event (calls `preventDefault`); return `false` to let it pass.
 *
 * Suppressed when focus is inside an editable element (`input`, `textarea`,
 * `select`, `contenteditable`) so typing in a field doesn't fire shortcuts.
 */
const CHORD_WINDOW_MS = 800;

const isEditableTarget = (target: EventTarget | null): boolean => {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName.toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return true;
  if (target.isContentEditable) return true;
  return false;
};

const normalizeKey = (e: KeyboardEvent): string => {
  // Lowercase letters, preserve digits and named keys.
  const key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
  const mods: string[] = [];
  if (e.ctrlKey) mods.push("ctrl");
  if (e.metaKey) mods.push("meta");
  if (e.altKey) mods.push("alt");
  if (e.shiftKey && key.length > 1) mods.push("shift");
  return mods.length ? `${mods.join("+")}+${key}` : key;
};

export type ShortcutHandler = (sequence: string, event: KeyboardEvent) => boolean;

export function useGlobalKeydown(handler: ShortcutHandler): void {
  const handlerRef = useRef<ShortcutHandler>(handler);
  handlerRef.current = handler;

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    let buffer: string[] = [];
    let timer: ReturnType<typeof setTimeout> | null = null;

    const reset = () => {
      buffer = [];
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
    };

    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented) return;
      if (isEditableTarget(e.target)) return;
      const key = normalizeKey(e);
      buffer.push(key);
      if (timer) clearTimeout(timer);
      timer = setTimeout(reset, CHORD_WINDOW_MS);
      const seq = buffer.join(" ");
      const consumed = handlerRef.current(seq, e);
      if (consumed) {
        e.preventDefault();
        reset();
      } else {
        // Local-first, relay-on-noop: when no local handler claims the chord
        // and we're embedded in an iframe host, surface the intent so the
        // host can offer its own shortcut (e.g. app-monitor's global
        // switcher on ⌘K). emitShortcutIntent is a no-op outside iframes.
        emitShortcutIntent({
          action: "scenario.unhandled",
          outcome: "noop",
          chord: seq,
          source: "keyboard",
        });
      }
    };

    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      reset();
    };
  }, []);
}
