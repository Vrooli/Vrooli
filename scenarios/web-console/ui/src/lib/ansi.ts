// DOC: docs/reference/configuration.md
/**
 * ANSI escape sequences for rendering colored status messages
 * inside xterm.js terminal instances.
 *
 * Used by useTerminalSession to display connection/error/exit messages
 * inline in the terminal. Extracted here so any future terminal-rendering
 * code can reuse the same palette without duplication.
 */
export const ANSI = {
  gray: "\x1b[90m",
  red: "\x1b[31m",
  yellow: "\x1b[33m",
  reset: "\x1b[0m",
} as const;
