// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-key-combos-p0-007
/**
 * Key combo definitions for the mobile toolbar combo picker.
 *
 * ── VOLATILE: This file is an extension point. ──
 * When adding new key combos, changes should land HERE
 * and not in the KeyComboPicker component.
 */

import {
  CTRL_A, CTRL_BACKSLASH, CTRL_C, CTRL_D, CTRL_E, CTRL_K, CTRL_L, CTRL_N,
  CTRL_P, CTRL_Q, CTRL_R, CTRL_S, CTRL_T, CTRL_U, CTRL_W, CTRL_Y, CTRL_Z,
  CSI_END, CSI_HOME,
} from "../lib/terminalKeys";

/** A single step in a key combo sequence. */
export interface KeyComboStep {
  /** Raw terminal data to send (escape sequence or control character). */
  data: string;
  /** Delay in ms before sending this step (0 or omitted = immediate). */
  delayMs?: number;
}

/** A key combo that can be sent to the terminal with a single tap. */
export interface KeyCombo {
  id: string;
  /** Human-readable name, e.g. "Interrupt / Cancel". */
  label: string;
  /** Display string, e.g. "Ctrl+C". */
  keys: string;
  /** Ordered steps to send. */
  sequence: KeyComboStep[];
  /** Grouping category. */
  category: KeyComboCategory;
  /** Additional search keywords for filtering. */
  searchTerms?: string[];
}

export type KeyComboCategory =
  | "Process Control"
  | "Navigation"
  | "History"
  | "Editing"
  | "Terminal";

/** Category display order. */
export const CATEGORY_ORDER: KeyComboCategory[] = [
  "Process Control",
  "Navigation",
  "History",
  "Editing",
  "Terminal",
];

/** Prepopulated key combos. */
export const KEY_COMBOS: KeyCombo[] = [
  // ── Process Control ──
  { id: "ctrl-c", label: "Interrupt / Cancel", keys: "Ctrl+C", sequence: [{ data: CTRL_C }], category: "Process Control", searchTerms: ["sigint", "stop", "cancel", "kill"] },
  { id: "ctrl-c-x2", label: "Force Quit", keys: "Ctrl+C ×2", sequence: [{ data: CTRL_C }, { data: CTRL_C, delayMs: 80 }], category: "Process Control", searchTerms: ["exit", "kill", "terminate", "claude"] },
  { id: "ctrl-d", label: "EOF / Exit", keys: "Ctrl+D", sequence: [{ data: CTRL_D }], category: "Process Control", searchTerms: ["end", "logout", "close"] },
  { id: "ctrl-z", label: "Suspend", keys: "Ctrl+Z", sequence: [{ data: CTRL_Z }], category: "Process Control", searchTerms: ["background", "pause", "sigtstp"] },
  { id: "ctrl-backslash", label: "SIGQUIT", keys: "Ctrl+\\", sequence: [{ data: CTRL_BACKSLASH }], category: "Process Control", searchTerms: ["quit", "core dump", "abort"] },

  // ── Navigation ──
  // Home and End send standard CSI escape sequences that most shells and TUI
  // apps recognise.  They duplicate the Ctrl+A / Ctrl+E readline shortcuts
  // below, but are far more discoverable for users who think in terms of
  // physical keyboard keys rather than Emacs-style bindings.
  { id: "home", label: "Home (Start of Line)", keys: "Home", sequence: [{ data: CSI_HOME }], category: "Navigation", searchTerms: ["beginning", "start", "bol", "ctrl+a"] },
  { id: "end", label: "End (End of Line)", keys: "End", sequence: [{ data: CSI_END }], category: "Navigation", searchTerms: ["eol", "ctrl+e"] },
  { id: "ctrl-l", label: "Clear Screen", keys: "Ctrl+L", sequence: [{ data: CTRL_L }], category: "Navigation", searchTerms: ["clear", "cls", "refresh"] },
  { id: "ctrl-a", label: "Start of Line", keys: "Ctrl+A", sequence: [{ data: CTRL_A }], category: "Navigation", searchTerms: ["home", "beginning"] },
  { id: "ctrl-e", label: "End of Line", keys: "Ctrl+E", sequence: [{ data: CTRL_E }], category: "Navigation", searchTerms: ["end"] },
  { id: "ctrl-u", label: "Kill Line Before", keys: "Ctrl+U", sequence: [{ data: CTRL_U }], category: "Navigation", searchTerms: ["delete", "erase", "clear line"] },
  { id: "ctrl-k", label: "Kill Line After", keys: "Ctrl+K", sequence: [{ data: CTRL_K }], category: "Navigation", searchTerms: ["delete", "erase", "cut"] },

  // ── History ──
  { id: "ctrl-r", label: "Reverse Search", keys: "Ctrl+R", sequence: [{ data: CTRL_R }], category: "History", searchTerms: ["search", "find", "history"] },
  { id: "ctrl-p", label: "Previous Command", keys: "Ctrl+P", sequence: [{ data: CTRL_P }], category: "History", searchTerms: ["up", "last"] },
  { id: "ctrl-n", label: "Next Command", keys: "Ctrl+N", sequence: [{ data: CTRL_N }], category: "History", searchTerms: ["down"] },

  // ── Editing ──
  { id: "ctrl-w", label: "Delete Word", keys: "Ctrl+W", sequence: [{ data: CTRL_W }], category: "Editing", searchTerms: ["backspace", "erase word"] },
  { id: "ctrl-y", label: "Yank / Paste", keys: "Ctrl+Y", sequence: [{ data: CTRL_Y }], category: "Editing", searchTerms: ["paste", "put"] },
  { id: "ctrl-t", label: "Transpose Chars", keys: "Ctrl+T", sequence: [{ data: CTRL_T }], category: "Editing", searchTerms: ["swap", "transpose"] },

  // ── Terminal ──
  { id: "ctrl-s", label: "Pause Output", keys: "Ctrl+S", sequence: [{ data: CTRL_S }], category: "Terminal", searchTerms: ["freeze", "xoff", "hold"] },
  { id: "ctrl-q", label: "Resume Output", keys: "Ctrl+Q", sequence: [{ data: CTRL_Q }], category: "Terminal", searchTerms: ["unfreeze", "xon", "continue"] },
];

/**
 * Filter combos by a search query. Case-insensitive match against
 * label, keys, category, and searchTerms.
 */
export function filterCombos(combos: KeyCombo[], query: string): KeyCombo[] {
  const q = query.trim().toLowerCase();
  if (!q) return combos;
  return combos.filter((c) => {
    if (c.label.toLowerCase().includes(q)) return true;
    if (c.keys.toLowerCase().includes(q)) return true;
    if (c.category.toLowerCase().includes(q)) return true;
    if (c.searchTerms?.some((t) => t.toLowerCase().includes(q))) return true;
    return false;
  });
}
