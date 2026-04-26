# `terminal` — Decoded PTY State + Snapshot Replay

Authoritative decoded state of one PTY's output, plus a self-contained
ANSI snapshot serializer used to replay state to a (re)connecting client.

## Why

Web-console used to keep a raw-PTY-byte ring as durable history. That
representation is not replay-safe across alt-buffer transitions: a TUI's
unmatched `\x1b[?1049h` left the client xterm in alt-buffer mode where
scrollback is disabled, so reconnect lost every line of history. See
`docs/plans/terminal-emulator-replay-implementation-plan.md` §4 for the
investigation.

This package replaces the byte ring with a decoded `Emulator` that
maintains a screen grid and a bounded scrollback ring. `Snapshot()`
emits an ANSI byte stream that recreates the emulator's `(screen,
alt-buffer, scrollback)` triple in any conforming xterm-compatible
client.

## API

```go
type Options struct {
    Cols, Rows      int
    ScrollbackLines int
}

type Emulator struct{ /* opaque */ }

func New(opts Options) *Emulator
func (*Emulator) Feed(p []byte) (int, error) // total — never errors
func (*Emulator) Resize(cols, rows int)
func (*Emulator) Snapshot() []byte
func (*Emulator) InAltBuffer() bool
func (*Emulator) ScrollbackLineCount() int
func (*Emulator) Cols() int
func (*Emulator) Rows() int
```

## Invariants (frozen)

- **Feed is total.** Always consumes every byte; malformed escapes are
  dropped without surfacing an error.
- **Snapshot is idempotent under no input.** Two consecutive snapshots
  return byte-equal results.
- **Snapshot is complete.** Feeding the snapshot bytes back into a
  fresh `Emulator` with the same `Options` produces an equivalent
  `(screen, alt-buffer, scrollback)` triple.
- **Alt-buffer is opaque to scrollback.** Lines written while
  `InAltBuffer()` returns true never enter the scrollback ring.
- **Resize preserves scrollback line count.** Cell content may be
  truncated to the new column count.

## Concurrency

`Emulator` is **not** safe for concurrent use. Callers (Session) hold a
mutex around `Feed` / `Snapshot` / `Resize`.

## What's deliberately NOT modeled

- Mouse-tracking modes, application-cursor mode, bracketed-paste —
  these are session-state, not scrollback-state. xterm.js is the
  authoritative renderer; the emulator only needs enough fidelity to
  produce a replayable byte stream.
- Charset switches (`ESC ( B`, etc.) are consumed and dropped.
- DCS / OSC / SOS / PM / APC sequences are consumed and dropped.

If a snapshot misrenders some specific scenario in xterm.js, the fix is
to teach `Emulator.onCSI` / `onESC` the missing case and add a test —
not to push more work onto the snapshot decoder.
