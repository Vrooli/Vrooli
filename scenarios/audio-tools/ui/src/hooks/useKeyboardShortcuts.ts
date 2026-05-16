import * as React from "react";

export interface Shortcut {
  /** Lowercase key from KeyboardEvent.key. */
  key: string;
  ctrlOrMeta?: boolean;
  shift?: boolean;
  alt?: boolean;
  handler: (event: KeyboardEvent) => void;
  /** When true, fires even if focus is inside a form field. */
  allowInInputs?: boolean;
}

const FORM_TAGS = new Set(["INPUT", "TEXTAREA", "SELECT"]);

function isFormTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (FORM_TAGS.has(target.tagName)) return true;
  return target.isContentEditable;
}

/**
 * Register keyboard shortcuts. Always cleans up on unmount. Shortcuts whose
 * `ctrlOrMeta` is set fire on both Ctrl (Win/Linux) and Cmd (macOS) so the
 * binding is portable.
 */
export function useKeyboardShortcuts(shortcuts: Shortcut[]): void {
  const ref = React.useRef(shortcuts);
  ref.current = shortcuts;

  React.useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      for (const sc of ref.current) {
        if (event.key.toLowerCase() !== sc.key.toLowerCase()) continue;
        const ctrlOk = sc.ctrlOrMeta ? event.ctrlKey || event.metaKey : !event.ctrlKey && !event.metaKey;
        if (!ctrlOk) continue;
        if (Boolean(sc.shift) !== event.shiftKey) continue;
        if (Boolean(sc.alt) !== event.altKey) continue;
        if (!sc.allowInInputs && isFormTarget(event.target)) continue;
        event.preventDefault();
        sc.handler(event);
        return;
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);
}
