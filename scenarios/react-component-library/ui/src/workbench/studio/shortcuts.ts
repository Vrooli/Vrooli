/** The two history commands a canvas honours from the keyboard. */
export type HistoryShortcut = "undo" | "redo";

export interface HistoryShortcutInput {
  key: string;
  meta: boolean;
  ctrl: boolean;
  shift: boolean;
  /**
   * True while focus is in a field. The browser's own text undo must win
   * there; stealing Ctrl+Z from a prompt would erase typing, not design work.
   */
  editable: boolean;
}

/**
 * Cmd/Ctrl+Z undoes, Cmd/Ctrl+Shift+Z and Ctrl+Y redo. Kept pure so the key
 * policy is tested once instead of being re-derived in an event listener.
 */
export function historyShortcut(input: HistoryShortcutInput): HistoryShortcut | null {
  if (input.editable || !(input.meta || input.ctrl)) return null;
  const key = input.key.toLowerCase();
  if (key === "y" && !input.meta && !input.shift) return "redo";
  if (key !== "z") return null;
  return input.shift ? "redo" : "undo";
}
