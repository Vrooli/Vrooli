// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
/**
 * Key combo definitions for the mobile toolbar combo picker.
 *
 * ── VOLATILE: This file is an extension point. ──
 * When adding new key combos, changes should land HERE
 * and not in the KeyComboPicker component.
 */
/** Category display order. */
export const CATEGORY_ORDER = [
    "Process Control",
    "Navigation",
    "History",
    "Editing",
    "Terminal",
];
/** Prepopulated key combos. */
export const KEY_COMBOS = [
    // ── Process Control ──
    { id: "ctrl-c", label: "Interrupt / Cancel", keys: "Ctrl+C", sequence: [{ data: "\x03" }], category: "Process Control", searchTerms: ["sigint", "stop", "cancel", "kill"] },
    { id: "ctrl-c-x2", label: "Force Quit", keys: "Ctrl+C ×2", sequence: [{ data: "\x03" }, { data: "\x03", delayMs: 80 }], category: "Process Control", searchTerms: ["exit", "kill", "terminate", "claude"] },
    { id: "ctrl-d", label: "EOF / Exit", keys: "Ctrl+D", sequence: [{ data: "\x04" }], category: "Process Control", searchTerms: ["end", "logout", "close"] },
    { id: "ctrl-z", label: "Suspend", keys: "Ctrl+Z", sequence: [{ data: "\x1a" }], category: "Process Control", searchTerms: ["background", "pause", "sigtstp"] },
    { id: "ctrl-backslash", label: "SIGQUIT", keys: "Ctrl+\\", sequence: [{ data: "\x1c" }], category: "Process Control", searchTerms: ["quit", "core dump", "abort"] },
    // ── Navigation ──
    // Home and End send standard CSI escape sequences that most shells and TUI
    // apps recognise.  They duplicate the Ctrl+A / Ctrl+E readline shortcuts
    // below, but are far more discoverable for users who think in terms of
    // physical keyboard keys rather than Emacs-style bindings.
    { id: "home", label: "Home (Start of Line)", keys: "Home", sequence: [{ data: "\x1b[H" }], category: "Navigation", searchTerms: ["beginning", "start", "bol", "ctrl+a"] },
    { id: "end", label: "End (End of Line)", keys: "End", sequence: [{ data: "\x1b[F" }], category: "Navigation", searchTerms: ["eol", "ctrl+e"] },
    { id: "ctrl-l", label: "Clear Screen", keys: "Ctrl+L", sequence: [{ data: "\x0c" }], category: "Navigation", searchTerms: ["clear", "cls", "refresh"] },
    { id: "ctrl-a", label: "Start of Line", keys: "Ctrl+A", sequence: [{ data: "\x01" }], category: "Navigation", searchTerms: ["home", "beginning"] },
    { id: "ctrl-e", label: "End of Line", keys: "Ctrl+E", sequence: [{ data: "\x05" }], category: "Navigation", searchTerms: ["end"] },
    { id: "ctrl-u", label: "Kill Line Before", keys: "Ctrl+U", sequence: [{ data: "\x15" }], category: "Navigation", searchTerms: ["delete", "erase", "clear line"] },
    { id: "ctrl-k", label: "Kill Line After", keys: "Ctrl+K", sequence: [{ data: "\x0b" }], category: "Navigation", searchTerms: ["delete", "erase", "cut"] },
    // ── History ──
    { id: "ctrl-r", label: "Reverse Search", keys: "Ctrl+R", sequence: [{ data: "\x12" }], category: "History", searchTerms: ["search", "find", "history"] },
    { id: "ctrl-p", label: "Previous Command", keys: "Ctrl+P", sequence: [{ data: "\x10" }], category: "History", searchTerms: ["up", "last"] },
    { id: "ctrl-n", label: "Next Command", keys: "Ctrl+N", sequence: [{ data: "\x0e" }], category: "History", searchTerms: ["down"] },
    // ── Editing ──
    { id: "ctrl-w", label: "Delete Word", keys: "Ctrl+W", sequence: [{ data: "\x17" }], category: "Editing", searchTerms: ["backspace", "erase word"] },
    { id: "ctrl-y", label: "Yank / Paste", keys: "Ctrl+Y", sequence: [{ data: "\x19" }], category: "Editing", searchTerms: ["paste", "put"] },
    { id: "ctrl-t", label: "Transpose Chars", keys: "Ctrl+T", sequence: [{ data: "\x14" }], category: "Editing", searchTerms: ["swap", "transpose"] },
    // ── Terminal ──
    { id: "ctrl-s", label: "Pause Output", keys: "Ctrl+S", sequence: [{ data: "\x13" }], category: "Terminal", searchTerms: ["freeze", "xoff", "hold"] },
    { id: "ctrl-q", label: "Resume Output", keys: "Ctrl+Q", sequence: [{ data: "\x11" }], category: "Terminal", searchTerms: ["unfreeze", "xon", "continue"] },
];
/**
 * Filter combos by a search query. Case-insensitive match against
 * label, keys, category, and searchTerms.
 */
export function filterCombos(combos, query) {
    const q = query.trim().toLowerCase();
    if (!q)
        return combos;
    return combos.filter((c) => {
        if (c.label.toLowerCase().includes(q))
            return true;
        if (c.keys.toLowerCase().includes(q))
            return true;
        if (c.category.toLowerCase().includes(q))
            return true;
        if (c.searchTerms?.some((t) => t.toLowerCase().includes(q)))
            return true;
        return false;
    });
}
