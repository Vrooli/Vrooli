// terminalConfig.ts: Shared client/server constants for the terminal pane.

// TERMINAL_SCROLLBACK_LINES is the number of lines xterm.js retains in its
// own scrollback. Must match (or exceed) the server's emulator scrollback
// (Config.TerminalScrollbackLines, default 10_000) so a snapshot replay
// fits without truncation on the client.
export const TERMINAL_SCROLLBACK_LINES = 10_000;
