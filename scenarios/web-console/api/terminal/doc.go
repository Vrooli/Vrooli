// Package terminal owns the authoritative decoded state of a PTY's output:
// a screen grid, an alt-buffer flag, and a bounded scrollback of decoded
// lines.  It exists so callers never have to interpret raw PTY bytes.
//
// Responsibilities:
//   - Parse PTY byte streams (UTF-8 + VT/xterm escape sequences).
//   - Maintain screen state and normal-buffer scrollback.
//   - Produce self-contained ANSI snapshots that reproduce equivalent
//     state in any conforming xterm-compatible client.
//   - Reflow scrollback on resize.
//
// Non-goals:
//   - Live byte forwarding (Session.broadcast still owns that).
//   - Client-side rendering (xterm.js owns that).
//   - Any persistence to disk (out of scope; future work).
//
// See: docs/plans/terminal-emulator-replay-implementation-plan.md
package terminal
