// terminalConfig.ts: Shared client/server constants for the terminal pane.

// TERMINAL_SCROLLBACK_LINES is the number of lines xterm.js retains in its
// own scrollback. Must match (or exceed) the server's emulator scrollback
// (Config.TerminalScrollbackLines, default ten-thousand) so a snapshot replay
// fits without truncation on the client.
export const TERMINAL_SCROLLBACK_LINES = 10 * 1000;

/** Pending input older than this is held until the operator explicitly sends it. */
export const PENDING_INPUT_HOLD_MS = 10 * 60 * 1000;

/** Adjacent typing events inside this window represent one user action. */
export const PENDING_INPUT_COALESCE_MS = 250;
