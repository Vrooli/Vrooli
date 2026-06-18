import { useCallback, useEffect } from "react";

export interface ShortcutHandlers {
  /** Ctrl/Cmd+Z — return true if the undo was handled. */
  onUndo?: () => boolean;
  /** Ctrl/Cmd+Shift+Z or Ctrl+Y — return true if the redo was handled. */
  onRedo?: () => boolean;
}

function isEditableTarget(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) {
    return false;
  }
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.tagName === "SELECT" ||
    el.isContentEditable
  );
}

/**
 * The single app-level keydown owner (vrooli-ui-interop slot [G]). Components
 * must not register their own window keydown listeners for app shortcuts; a
 * dialog-local Escape handler is fine. Shortcuts are suppressed while focus is
 * in a text field so typing values is never hijacked.
 *
 * Stage 0b wires only the local undo/redo chords. When a host-relevant chord
 * (e.g. the global switcher) is added, relay unhandled chords here via
 * `emitShortcutIntent` from `@vrooli/iframe-bridge`.
 */
export function useKeyboardShortcuts(handlers: ShortcutHandlers): void {
  const onKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (isEditableTarget(event.target)) {
        return;
      }
      const mod = event.metaKey || event.ctrlKey;
      if (!mod) {
        return;
      }
      const key = event.key.toLowerCase();

      if (key === "z" && !event.shiftKey) {
        event.preventDefault();
        handlers.onUndo?.();
        return;
      }
      if ((key === "z" && event.shiftKey) || key === "y") {
        event.preventDefault();
        handlers.onRedo?.();
      }
    },
    [handlers],
  );

  useEffect(() => {
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onKeyDown]);
}
