# Terminal Emulator Replay — Implementation Plan

## 1. Purpose

Replace the web-console's raw-PTY-byte history buffer with a **server-side
terminal emulator** that maintains decoded screen state and produces a
self-contained ANSI snapshot on every (re)connect. This permanently fixes the
scrollback duplication / "only-the-last-section" bug observed in persistent
sessions running TUI apps (Claude Code, vim, etc.), removes three coupled
sources of subtle bugs (byte-offset resume math, ANSI clean-boundary trim,
client cache-state drift), and converges terminal persistence onto a single
authoritative abstraction that other features (search, export, AI side-channel)
can build on.

This plan is the canonical execution artifact. A future agent must be able to
execute it end-to-end with no chat history.

---

## 2. Required Reading (run first)

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read scientific-debugging   # the bug investigation that produced this plan
```

In-repo references the executor must read before touching code:

```bash
# Canonical scenario architecture & the existing history-cache contract being deleted.
sed -n '1,200p' scenarios/web-console/docs/concepts/ARCHITECTURE.md
grep -n "terminal-history-caching" scenarios/web-console/docs/concepts/ARCHITECTURE.md

# The current PTY/history/WS code paths being replaced.
wc -l scenarios/web-console/api/{history_store,session,pty,pty_tmux,terminal_ws,terminal_ws_input}.go
wc -l scenarios/web-console/ui/src/hooks/terminal/{useTerminalSession,useTerminalTransport}.ts
wc -l scenarios/web-console/ui/src/components/TerminalPane.tsx
wc -l scenarios/web-console/ui/src/lib/terminalCache.ts

# The seams doc (must be updated when adding new emulator seams).
test -f scenarios/web-console/docs/internal/SEAMS.md && cat scenarios/web-console/docs/internal/SEAMS.md
```

---

## 3. GREENFIELD HARD RULE (non-negotiable)

This is a **greenfield refactor**. The user has explicitly forbidden compatibility
shims, dead code, deprecation bridges, or feature flags.

- **No** `history_offset` query parameter, in any form, anywhere — server, client,
  tests, docs.
- **No** `outputHistory []byte`, `appendHistory`, `historyStart`,
  `totalOutputBytes`, `offlineBufferMax`, or `OfflineBufferMax` config knob.
- **No** `WC_OFFLINE_BUFFER_MAX` env var. Delete from `api/config.go` and any
  docs that mention it.
- **No** `totalBytesRef`, `historyBufferRef`, `replayingHistoryRef`,
  `hasCachedState`, `hasCachedStateForConnectionRef`, or "stale-cache reset"
  branch in `useTerminalSession`.
- **No** `terminalCache.ts` byte-offset persistence layer (delete file).
- **No** `terminal-scrollback-dedup.test.ts` (the bug class no longer exists; delete file).
- **No** `total_bytes` or `resumed` fields on `history_end` (delete from message
  schema).
- **No** transitional "support both old and new clients" code. The UI and API
  ship together.
- **No** `// TODO: remove old path once …` comments. If it's not the new path,
  delete it.

This rule is repeated in **§13 Definition of Done** and is a CI-checkable
acceptance criterion (§9).

---

## 4. Problem Statement

Persistent-mode terminals (and standard mode when a TUI is launched mid-session)
exhibit **broken scrollback**: scrolling up shows only the last visible screen,
appearing to repeat. Direct measurement confirmed:

- Server `outputHistory` *is* populated (147 KB – 829 KB across six live sessions)
  — original "empty buffer" hypothesis falsified.
- The replay byte stream contains an **unmatched alt-buffer enter** (`\x1b[?1049h`
  with no matching `\x1b[?1049l`) and 41–332 cursor-home redraw cycles.
- xterm.js processes the replay → switches to alt-buffer → all subsequent bytes
  land in the alt-buffer where, by VT spec, **scrollback is disabled**.
- Result: client xterm has effectively zero scrollback after replay, regardless
  of how many bytes the server sent.

Investigation evidence captured in this conversation (verbatim probe outputs)
should be linked into the fix's PR description.

The architectural defect is **using raw PTY bytes as the durable history
representation**. Bytes are not replay-safe across mode boundaries (alt-buffer,
charset switches, mouse-tracking), are not resize-safe, are not UTF-8-safe at
trim boundaries, and grow without semantic bound. The fix is to replace the
durable representation with a **decoded terminal state** maintained by an
emulator.

---

## 5. Scope

### In scope
- New Go package `scenarios/web-console/api/terminal/` containing:
  - `Emulator` (VT parser + screen + scrollback + alt-buffer flag).
  - `Snapshot` (self-contained ANSI replay payload generator).
  - `Resize` semantics with scrollback reflow.
- Wire-in at `Session.readLoop` (both `pty.go` and `pty_tmux.go`).
- `Subscribe()` rewrite: returns a `Snapshot` instead of a byte slice.
- `terminal_ws.go` rewrite: emits snapshot bytes between connect and
  `history_end`; drops `history_offset` parsing entirely.
- UI: `useTerminalSession.ts` rewrite to the snapshot-only protocol; deletion of
  `terminalCache.ts` and its test.
- Removal of all touchpoints listed in §3 (greenfield).
- Test coverage: emulator unit tests, snapshot golden tests, WS integration
  test that asserts scrollback survives alt-buffer transitions.
- Docs: `docs/concepts/ARCHITECTURE.md` "Terminal History Caching" section
  rewritten as "Terminal Snapshot Replay"; `docs/internal/SEAMS.md` updated.

### Out of scope
- Server-side text search across scrollback (unlocked by this work; a separate plan).
- Structured asciicast/HTML export (unlocked; separate plan).
- Per-pane independent scrollback streams (today's session model already maps
  one PTY per session; unchanged).
- Voice / TTS / AI side-channel changes (decoupled; consume snapshot decoded
  text via a future helper, but that helper is not built here).
- Standard-mode vs. persistent-mode protocol divergence — both modes will use
  the identical snapshot protocol after this plan.

---

## 6. Current Technical Context

### 6.1 Existing architecture (to be deleted)

| File | Role today | Disposition |
|---|---|---|
| `api/history_store.go` (140 LoC) | Raw byte ring + ANSI clean-boundary trim | **Delete** |
| `api/history_store_test.go` | Tests the deleted ring | **Delete** |
| `api/session.go` lines 89–106, 219–225, 766–785, 995–1015, 1161 | `outputHistory`/`offlineBufferMax` plumbing on `Session` | **Edit** — replace with `emulator *terminal.Emulator` |
| `api/pty.go` | Standard-mode `readLoop`; calls `appendHistory` then `broadcast` | **Edit** — calls `emulator.Feed` then `broadcast` |
| `api/pty_tmux.go` | Persistent-mode `readLoop`; same shape | **Edit** — same change |
| `api/terminal_ws.go` lines 184–227 | `history_offset` query parsing, `historyEndMsg` with `TotalBytes`/`Resumed` | **Edit** — drop offset parsing; emit snapshot frames |
| `api/config.go` lines 15–19, 106, 128 | `OfflineBufferMax` config | **Delete those lines** |
| `ui/src/hooks/terminal/useTerminalSession.ts` lines 250–298, 340–380 | `historyBufferRef`, `totalBytesRef`, `hasCachedState`, stale-cache reset | **Rewrite** §6.3 |
| `ui/src/lib/terminalCache.ts` | localStorage byte-offset cache | **Delete** |
| `ui/src/__tests__/terminalCache.test.ts` | Tests the deleted cache | **Delete** |
| `ui/src/__tests__/terminal-scrollback-dedup.test.ts` | Tests the obsolete bug class | **Delete** |
| `docs/concepts/ARCHITECTURE.md` "Terminal History Caching" section | Documents the deleted protocol | **Rewrite** as "Terminal Snapshot Replay" |

### 6.2 Live-evidence baseline (captured during investigation)

```
Server WS replay vs. tmux capture-pane (six live persistent sessions):
  session 145a3157  tmux=16,740B  server=147,563B
  session 19f6a69f  tmux=13,832B  server=160,618B
  session 391d0d35  tmux=65,709B  server=251,518B
  session a50aa5e4  tmux=71,954B  server=307,412B
  session ec292a55  tmux=77,175B  server=829,291B
  session f4c9c192  tmux=71,092B  server=160,585B

ANSI sequence inventory in server replay (per session):
  \x1b[?1049h  enter alt-buffer:  1
  \x1b[?1049l  exit  alt-buffer:  0   ← unmatched (root cause)
  \x1b[2J      erase display:     1
  \x1b[H       cursor home:       41–332
```

These numbers are the **before** baseline. The acceptance test in §9 must
demonstrate the **after** state on the same kind of input.

### 6.3 Existing UI seams to preserve
- `useTerminalTransport` (WS framing, ping/pong, reconnect with backoff) —
  unchanged.
- `useStdinAck` (input write-barrier) — unchanged.
- xterm `@xterm/addon-fit`, `@xterm/addon-web-links`, `@xterm/addon-serialize`
  — unchanged. (`addon-serialize` stays available for future client-side
  features but is **not** used in the protocol.)

---

## 7. Target End State (Screaming Architecture)

### 7.1 New API package layout

```
scenarios/web-console/api/terminal/
├── doc.go                       # package banner: responsibilities + non-goals
├── emulator.go                  # Emulator type, Feed, Resize, Snapshot
├── emulator_test.go             # unit tests
├── screen.go                    # Cell, Line, Screen grid
├── scrollback.go                # bounded ring of decoded Lines
├── snapshot.go                  # ANSI snapshot serializer
├── snapshot_test.go             # golden-file tests
├── parser.go                    # thin adapter over the chosen VT library
├── testdata/
│   ├── recordings/              # *.bin captures from real sessions (anonymized)
│   ├── snapshots/               # golden snapshot bytes
│   └── README.md                # how to add a fixture
└── README.md                    # consumer-facing API + invariants
```

`api/terminal/doc.go` opens with the standard "banner of responsibilities":

```go
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
package terminal
```

### 7.2 Session integration

```go
// api/session.go
type Session struct {
    // ... unchanged fields ...
    emu *terminal.Emulator   // authoritative state; replaces outputHistory
    // (deleted: outputHistory, totalOutputBytes, offlineBufferMax)
}
```

`Session.Subscribe()` returns:

```go
type Subscription struct {
    OutputCh chan []byte    // live frames after snapshot (unchanged contract)
    NotifyCh chan int       // coalesce notifications (unchanged)
    Snapshot []byte         // self-contained ANSI replay; written before live
}
```

(Deleted from `Subscription`: `TotalBytes`, `Resumed`, `HadData`,
`InitialAltBuffer` — all subsumed by the snapshot bytes themselves.)

### 7.3 WS protocol (single source of truth: `api/terminal_ws.go` + `ui/src/types/terminal.ts`)

Connect handshake:
```
Client → GET /api/v1/sessions/{id}/ws            (no query parameters)
Server → stdout {data: <snapshot bytes, possibly chunked>}…
Server → history_end {}                          (no fields)
Server → live stdout / stdin_ack / etc.          (unchanged)
```

`history_end` becomes a pure delimiter. `pty_state` is folded into the
snapshot (the snapshot already encodes alt-buffer state through
`\x1b[?1049h/l`). Delete `MsgTypePTYState` and the post-history `pty_state`
write at `terminal_ws.go:245, 279`.

### 7.4 UI flow

```
useTerminalTransport.onOpen  ──▶  terminal.reset()
                                  ▼
                                  for each {type:"stdout"} frame: terminal.write(data)
                                  ▼
                                  on {type:"history_end"}: enter live mode
                                  ▼
                                  for each subsequent stdout: terminal.write(data)
```

No buffering. No offset accounting. No cache. The xterm instance is a pure
display; the server is the source of truth.

---

## 8. Implementation Strategy (phased, each phase shippable & green)

Phases are sequenced for **safe execution by an autonomous agent**. Each phase
ends with a green test suite and a working scenario restart. **No phase leaves
the system half-working.**

### Phase 0 — Spike: pick the VT library

**Owner artifact:** `scenarios/web-console/docs/plans/terminal-emulator-vt-spike.md`
(short eval doc, deleted at plan close).

**Contenders:**
1. `github.com/hinshun/vt10x` — most widely used Go VT terminal; stable cell grid.
2. `github.com/charmbracelet/x/exp/term` — newer, less battle-tested.
3. Embed Node sidecar with `xterm-headless` + `SerializeAddon` — strictly
   matches xterm.js semantics.

**Decision criteria (must be evaluated, recorded, defended):**
- Handles `\x1b[?1049h/l` correctly (alt-buffer separation from scrollback).
- Handles `\x1b[?1047h/l`, `\x1b[?47h/l` (older alt-buffer variants).
- Tracks SGR (color/bold/underline) per cell.
- Exposes scrollback when in alt-buffer (so we can snapshot normal-buffer
  history while a TUI is active).
- Pure Go (no cgo, no Node runtime). Sidecar option is rejected unless 1 & 2
  both fail.
- Test fixtures: feed `testdata/recordings/*.bin` and assert expected screen
  text.

**Output:** decision doc + `go get` line added to `api/go.mod`.

**Default if equal:** `vt10x`. Spike must justify departure.

### Phase 1 — Build the emulator package, isolated

No `Session` integration yet. Pure new code under `api/terminal/`.

**Tasks**
1. Add chosen VT library to `api/go.mod` + `go.sum`.
2. Implement `Emulator` with this contract:
   ```go
   type Emulator struct { /* ... */ }
   func New(opts Options) *Emulator
   type Options struct {
       Cols, Rows         int
       ScrollbackLines    int   // default 10_000
   }
   func (e *Emulator) Feed(p []byte) (n int, err error)        // io.Writer
   func (e *Emulator) Resize(cols, rows int)
   func (e *Emulator) Snapshot() []byte                         // self-contained ANSI replay
   func (e *Emulator) InAltBuffer() bool                        // for diagnostics
   func (e *Emulator) ScrollbackLineCount() int                 // for diagnostics
   ```
3. Implement `snapshot.go`:
   - Emit `\x1bc` (full reset).
   - Emit each scrollback line with run-length-coalesced SGR + `\r\n`.
   - If source is in alt-buffer: emit `\x1b[?1049h`, then current screen
     contents at correct cursor positions, then `\x1b[?25h/l` matching cursor
     visibility.
   - If source is in normal buffer: emit current screen at correct cursor.
   - Always end with cursor at the source's cursor position.
4. Concurrency: `Emulator` is **not** safe for concurrent use. Document on the
   type. Caller (Session) holds an existing mutex.
5. Banner of responsibilities at top of every file under `api/terminal/`.

**Tests** (new, all required to pass before Phase 2):
- `emulator_test.go`:
  - Plain ASCII streams produce expected scrollback.
  - `\x1b[2J` clears screen but **preserves scrollback above**.
  - `\x1b[?1049h` enters alt-buffer; scrollback is **frozen** (no new entries
    while in alt-buffer); `\x1b[?1049l` exits and screen is restored.
  - UTF-8 split across two `Feed` calls is decoded correctly.
  - SGR runs preserved on cells.
  - Resize from 80×24 → 120×40 reflows scrollback.
- `snapshot_test.go` (golden):
  - Three real-session captures (`testdata/recordings/{plain.bin, vim.bin,
    claude.bin}`) — record once with the live probe in
    `scenarios/web-console/docs/internal/probes/ws_capture.py` (move /tmp script
    into the repo as part of this phase; see §10).
  - For each: feed → snapshot → write snapshot to a fresh `Emulator` → assert
    final screen + scrollback match the original's final screen + scrollback.
  - The `claude.bin` case (alt-buffer-active TUI) is the **regression test for
    the bug**: scrollback before alt-buffer entry must be present in the
    snapshot.

**Phase exit:** `go test ./api/terminal/...` green.

### Phase 2 — Wire `Emulator` into `Session`

**Tasks**
1. Add `emu *terminal.Emulator` field on `Session`. Construct in
   `NewSession`, `RecoverSession`, and the persistent-recovery path
   (`session.go:766–785, 995–1015, 1161`). Use cols/rows from session config;
   scrollback lines from a new config knob `WC_TERMINAL_SCROLLBACK_LINES`
   (default 10_000, range 100–100_000).
2. In `pty.go` and `pty_tmux.go` `readLoop`, replace `s.appendHistory(data)`
   with `s.emu.Feed(data)` (no error handling — `Feed` cannot fail; document
   why on the method).
3. On `Session.Resize`, call `emu.Resize`.
4. **Delete**: `history_store.go`, `history_store_test.go`,
   `Session.outputHistory`, `Session.totalOutputBytes`,
   `Session.offlineBufferMax`, `Config.OfflineBufferMax`,
   `WC_OFFLINE_BUFFER_MAX` env reading. Delete tests that exercise these.
5. Rewrite `Session.Subscribe()`:
   ```go
   func (s *Session) Subscribe() *Subscription {
       s.mu.Lock(); defer s.mu.Unlock()
       snap := s.emu.Snapshot()
       sub := &Subscription{
           Snapshot: snap,
           OutputCh: make(chan []byte, outputChannelBuffer),
           NotifyCh: make(chan int, 1),
       }
       s.subscribers = append(s.subscribers, sub)
       return sub
   }
   ```
   No offset parameter. Snapshot is generated under the same lock as subscriber
   registration so no live frame can sneak between snapshot and channel.

**Tests**
- `session_test.go`: rewrite `Subscribe` tests to assert snapshot is
  non-empty and no live frames are lost between snapshot and `OutputCh`.
- New `session_emulator_integration_test.go`: spawn a fake PTY (`io.Pipe`
  backed), write a TUI-style sequence, subscribe twice, assert both
  subscribers receive equivalent state.

**Phase exit:** `go test ./api/...` green. Scenario restarts and existing UI
*may not yet work* because the WS protocol still references deleted fields —
Phase 3 closes that immediately.

### Phase 3 — Rewrite WS protocol on the server

**Tasks**
1. `terminal_ws.go`:
   - Delete `history_offset` query parsing (lines 184–189).
   - Delete `TotalBytes`, `Resumed`, `HadData`, `InitialAltBuffer` references.
   - Delete `MsgTypePTYState` definition and its writes.
   - On WS open: write `sub.Snapshot` as one or more `stdout` frames
     (chunked at e.g. 64 KB to keep individual JSON messages small), then
     write `{type:"history_end"}` with no fields.
2. `api/terminal_ws.go` message types: prune the `TerminalMessage` struct of
   the deleted fields. Don't add new fields.
3. Update `terminal_ws_test.go` and `session_test.go` to the new protocol.
   Delete tests that asserted the old fields.
4. Update `docs/concepts/ARCHITECTURE.md`: replace "Terminal History Caching"
   section with "Terminal Snapshot Replay" — diagram, invariants, failure
   modes.

**Phase exit:** `go test ./api/...` green. Server protocol is the new shape.
UI is still on the old shape, so end-to-end is broken — Phase 4 closes it.

### Phase 4 — Rewrite UI client

**Tasks**
1. `ui/src/hooks/terminal/useTerminalSession.ts`:
   - Delete `historyBufferRef`, `historyTimeoutIdRef`, `replayingHistoryRef`,
     `totalBytesRef`, `hasCachedStateForConnectionRef`,
     `flushHistoryBuffer`, `HISTORY_FLUSH_TIMEOUT_MS`.
   - Rewrite `onTransportOpen`: call `terminal.reset()` unconditionally on
     every open (fresh OR reconnect), then enter snapshot-mode (a simple
     boolean `inSnapshotRef`). Write all snapshot `stdout` frames directly
     to xterm. On `history_end`, flip to live mode.
   - Delete the `historyOffset` parameter on every signature that took it.
2. **Delete** `ui/src/lib/terminalCache.ts`.
3. **Delete** `ui/src/__tests__/terminalCache.test.ts`.
4. **Delete** `ui/src/__tests__/terminal-scrollback-dedup.test.ts`.
5. `ui/src/types/terminal.ts`: prune deleted message fields.
6. `ui/src/components/TerminalPane.tsx`: ensure xterm `Terminal({ scrollback:
   10_000, ... })` matches the server's emulator scrollback. Centralize the
   number in a constant exported from a new `ui/src/lib/terminalConfig.ts`.

**Tests**
- New `ui/src/__tests__/terminal-snapshot-replay.test.tsx`: mock WS, send a
  fixture snapshot containing scrollback + alt-buffer enter, assert xterm's
  `buffer.normal.length` after `history_end` matches expected scrollback line
  count, and `buffer.active === buffer.alternate` reflects alt-buffer state.
- Update `terminal-pane-*.test.tsx` to the new transport contract.

**Phase exit:** `cd ui && npm run lint && npm run test && npm run build` green.

### Phase 5 — End-to-end verification on real persistent sessions

**Tasks**
1. Add a repo-versioned diagnostic helper at
   `scenarios/web-console/docs/internal/probes/ws_capture.py` (move from `/tmp`,
   document usage). Used both for capturing fixtures (Phase 1) and for the
   Phase 6 acceptance evidence.
2. Add a Go integration test
   `api/terminal_ws_e2e_test.go` (build tag `//go:build e2e`) that:
   - Creates an HTTP test server.
   - Creates a session.
   - Feeds the session's PTY a recorded TUI byte stream
     (`testdata/recordings/claude.bin`).
   - Subscribes via the real WS handler.
   - Asserts that the client side, after applying all `stdout` frames into a
     fresh `Emulator`, has **non-zero scrollback** (the bug's fingerprint).
3. Restart the scenario and validate:
   ```bash
   vrooli scenario restart web-console
   vrooli scenario status web-console     # expect: running, healthy
   ```
4. Run the diagnostic against three live sessions (one TUI, one shell, one
   freshly created) and attach output to the PR.

**Phase exit:** all of the above green.

### Phase 6 — Documentation & seams

**Tasks**
1. Update `scenarios/web-console/docs/concepts/ARCHITECTURE.md`:
   - Delete the entire "Terminal History Caching" section.
   - Add new "Terminal Snapshot Replay" section: diagram, invariants,
     failure modes, scrollback-line cap rationale.
2. Update `scenarios/web-console/docs/internal/SEAMS.md`:
   - Document `Emulator` as a seam (test boundary).
   - Document the WS snapshot protocol as a seam.
   - Remove any references to `OfflineBufferMax`, `historyOffset`.
3. Update `scenarios/web-console/README.md` if it mentions history caching.
4. Sweep `grep -rn "OFFLINE_BUFFER\|history_offset\|outputHistory\|terminalCache"`
   across the repo. Result must be **zero matches**.
5. Sweep `rg "AI_CHECK:"` for stale terminal-history task IDs and update.
6. Delete the Phase-0 spike doc.

**Phase exit:** `git status` clean of stray TODO/FIXME, doc grep is clean.

---

## 9. Contract Decisions (frozen during the refactor)

These are the public contracts. Changing them is a separate plan.

### 9.1 `terminal.Emulator` invariants
- `Feed` is total: it never errors and consumes every byte. Malformed escapes
  are logged at debug level and dropped.
- `Snapshot()` is **idempotent under no input**: snapshotting twice in a row
  returns byte-equal output.
- `Snapshot()` is **complete**: applying the snapshot to a fresh `Emulator`
  configured with the same `Options` produces the same `(screen, alt-buffer,
  scrollback)` triple.
- `Resize` preserves scrollback content. Lines that reflow may change in count.

### 9.2 WS protocol
- Server emits a sequence of `stdout` frames followed by exactly one
  `history_end` frame, in that order, before any other frame type that
  isn't a control frame (ping/pong/error).
- `history_end` carries no fields.
- Client must `reset()` xterm on every WS open and enter snapshot-mode until
  `history_end`.
- Live `stdout` frames are byte-identical to PTY output (no server-side
  emulator filtering); the server emulator only **observes** the live path.

### 9.3 Configuration surface
- New env var: `WC_TERMINAL_SCROLLBACK_LINES` (int, default 10_000, range
  100–100_000). Documented in `api/config.go` comment block.
- **Deleted** env vars: `WC_OFFLINE_BUFFER_MAX`. Setting it is silently
  ignored — but not a compatibility shim, just the natural consequence of
  the field being gone.

### 9.4 CLI parity
The `web-console` CLI currently has no commands that surface history bytes.
No CLI changes are required by this plan. (Verify with
`web-console session --help`; if any subcommand mentions `history-offset` or
`offline-buffer`, delete it.)

---

## 10. Testing Plan

Tests are **automated** (per durable user preference: prefer automated tests
over manual checklists). Manual probes are diagnostic-only.

### 10.1 Test pyramid

| Layer | Location | What it asserts |
|---|---|---|
| Emulator unit | `api/terminal/emulator_test.go` | Feed/Resize/Snapshot invariants in §9.1 |
| Snapshot golden | `api/terminal/snapshot_test.go` | Real-recording → snapshot → re-emulate → byte-equal final state |
| Session integration | `api/session_emulator_integration_test.go` | Subscribe contract, no live-frame loss |
| WS protocol | `api/terminal_ws_test.go` | Snapshot frames precede `history_end`; no deleted fields appear |
| WS end-to-end | `api/terminal_ws_e2e_test.go` (`//go:build e2e`) | Full HTTP server + WS client; **scrollback survives alt-buffer** |
| UI hook | `ui/src/__tests__/terminal-snapshot-replay.test.tsx` | xterm scrollback non-empty after replay of TUI fixture |

### 10.2 The regression test for the original bug

In `api/terminal_ws_e2e_test.go`:

```go
// Regression: persistent-mode TUI sessions must preserve normal-buffer
// scrollback through the snapshot, even when the captured byte stream
// contains an unmatched \x1b[?1049h.
//
// See: docs/plans/terminal-emulator-replay-implementation-plan.md §4
//      Investigation evidence: 6 live sessions, 0 1049l vs 1 1049h.
func TestSnapshotPreservesScrollbackAcrossAltBuffer(t *testing.T) {
    rec := mustReadFile(t, "terminal/testdata/recordings/claude.bin")
    sess := newTestSession(t)
    sess.feedPTY(rec)

    // Subscribe via the real WS handler.
    client := dialWS(t, sess.ID)
    snap := client.readUntilHistoryEnd(t)

    // Apply the snapshot to a fresh emulator (modeling the client xterm).
    e := terminal.New(terminal.Options{Cols: 80, Rows: 24, ScrollbackLines: 10_000})
    _, _ = e.Feed(snap)

    if got := e.ScrollbackLineCount(); got == 0 {
        t.Fatalf("scrollback empty after snapshot replay (the bug); want > 0")
    }
}
```

### 10.3 CI-checkable greenfield assertion

Add to `ui/src/__tests__/greenfield-assertions.test.ts` (existing file) a new
case:

```ts
it("greenfield: no references to deleted history-cache symbols", async () => {
  const forbidden = [
    "history_offset", "outputHistory", "appendHistory",
    "OfflineBufferMax", "WC_OFFLINE_BUFFER_MAX",
    "totalBytesRef", "hasCachedState", "terminalCache",
  ];
  // walk repo (excluding node_modules, this test, and the plan doc)
  // assert zero matches
});
```

This codifies §3 as a CI gate. A future refactor cannot accidentally
reintroduce the deleted shape.

### 10.4 Test data captures

Procedure to generate fixtures under `api/terminal/testdata/recordings/`:

```bash
# Spin up a session, run a representative workload, capture the WS replay bytes:
python3 scenarios/web-console/docs/internal/probes/ws_capture.py \
    <session-id> <api-port> api/terminal/testdata/recordings/<name>.bin
```

Three fixtures required: `plain.bin` (bash + ls + cat), `vim.bin` (vim
session, edit file, `:q`), `claude.bin` (Claude Code interactive). Anonymize
any path/name leaks before commit.

---

## 11. Rollout / Validation Checklist

Run in order. Every box must be checked before merge.

- [ ] Phase 0 spike doc committed; VT library decision recorded.
- [ ] `go test ./api/terminal/... -race` green.
- [ ] `go test ./api/...` green (full API suite).
- [ ] `go test -tags=e2e ./api/...` green (the regression test fires).
- [ ] `cd ui && npm run lint` green.
- [ ] `cd ui && npm run test` green.
- [ ] `cd ui && npm run build` green.
- [ ] `vrooli scenario restart web-console` succeeds.
- [ ] `vrooli scenario status web-console` reports running + healthy.
- [ ] `gofumpt -l api/` produces no output (formatter clean).
- [ ] `golangci-lint run ./api/...` clean.
- [ ] `grep -rn "history_offset\|outputHistory\|appendHistory\|OFFLINE_BUFFER\|terminalCache\|hasCachedState" .` returns **zero hits** outside this plan doc.
- [ ] Manual probe captured against three live sessions, attached to PR:
      one TUI, one plain shell, one freshly opened. Each must show
      scrollback line count > 0 in the client xterm after replay.
- [ ] `docs/concepts/ARCHITECTURE.md` rewritten section reviewed.
- [ ] `docs/internal/SEAMS.md` updated.
- [ ] PR description links the investigation transcript and lists deleted symbols.

---

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Chosen VT library mishandles `\x1b[?1049h/l` (frequent vendor difference). | Medium | High — defeats the fix | Phase 0 spike has explicit acceptance test for this. Falls back to alternative library if chosen one fails. |
| Snapshot for very long sessions becomes large (multi-MB), latency on connect. | Medium | Medium — slow first paint | Bound by `ScrollbackLines` (default 10K lines ≈ 1–2 MB). Chunk over 64 KB stdout frames. Profile in Phase 5. |
| Server CPU rises (parsing every PTY byte). | Medium | Low–Medium | Benchmark in `emulator_test.go` (`-bench`). Acceptable budget: < 5% of one core for typical interactive load. If exceeded, profile parser. |
| SGR fidelity loss on snapshot (colors look different after replay). | Medium | Low (cosmetic) | Golden test asserts cell-by-cell SGR equivalence after re-emulation. |
| Resize during disconnect produces a snapshot with mismatched cols/rows. | Low | Low | Snapshot encodes cursor at server cols/rows; first client `resize` after `history_end` reconciles. Document in §9.1. |
| A non-greenfield touchpoint is missed and silently keeps working. | Medium | Medium — code rot | §10.3 CI assertion plus §11 grep gate plus PR description checklist. |
| `vt10x` is unmaintained. | Low | Low | Spike re-evaluates; if stale, prefer `charmbracelet/x` or pin a fork. |
| Test fixtures contain user-identifiable paths. | Medium | Low | §10.4 anonymization step + reviewer call-out. |

---

## 13. Non-Goals / Prohibited Patterns

- **No** "dual-mode" code that supports both old and new protocols. Rip it out.
- **No** persistence of emulator state to disk in this plan (would complicate
  recovery semantics; addressed in a future plan if needed).
- **No** new public CLI surface in this plan.
- **No** UI behavior changes beyond replay correctness — visual styling,
  context menu, font, etc., stay identical.
- **No** observation hooks / pub-sub on the emulator (encourages misuse;
  callers should subscribe to live frames as they do today).
- **No** TODO/FIXME comments left in delivered code. If something is unfinished,
  it's not in this PR.
- **No** comments narrating "the old code did X, now we do Y." Greenfield.

---

## 14. Definition of Done

Every item below must be true at merge time. This restates and tightens §11.

1. **Greenfield rule (§3) holds.** `grep` gate in §11 returns zero. CI test
   in §10.3 enforces it on every future commit.
2. **Bug is fixed.** The regression test in §10.2 passes. A live
   persistent-mode session running a TUI shows non-empty scrollback after
   reconnect, demonstrated by the manual probe in §11.
3. **All tests pass.** Full Go suite (including `-race` and `-tags=e2e`),
   full UI suite (lint + test + build).
4. **Scenario is healthy.** `vrooli scenario restart web-console` followed by
   `vrooli scenario status web-console` reports running + healthy; this is
   a non-negotiable durable user-feedback rule.
5. **Architecture screams.** `api/terminal/` exists with the layout in §7.1,
   each file opens with a banner of responsibilities, and `doc.go` matches
   §7.1 verbatim in spirit.
6. **Docs match code.** `docs/concepts/ARCHITECTURE.md` and
   `docs/internal/SEAMS.md` describe the new model; no stale references.
7. **No legacy detritus.** `outputHistory`, `appendHistory`, `history_offset`,
   `OfflineBufferMax`, `WC_OFFLINE_BUFFER_MAX`, `totalBytesRef`,
   `hasCachedState`, `terminalCache.ts`, `terminal-scrollback-dedup.test.ts`
   are all deleted, with no replacement under a new name.
8. **PR description** links: this plan, the investigation evidence (live
   probe table), the chosen VT library, and the regression test.

---

## 15. Execution Notes for the Implementing Agent

- This plan is large. Execute one phase per commit; don't bundle. Each commit
  message references the phase number and the symbols added/deleted.
- Restart the web-console scenario via `vrooli scenario restart web-console`
  after Phases 2, 3, 4, and 5 — durable user preference. Do not use
  `make stop && make start`.
- Do **not** create commits unless explicitly asked by the user (durable
  preference: no git mutations).
- If a phase's tests reveal a new fact that contradicts this plan (e.g.,
  `vt10x` doesn't track alt-buffer separately), update the plan in a new
  commit before proceeding — don't silently diverge.
- The investigation transcript (live probe outputs) is the empirical baseline.
  Keep it linked from the PR; it's the "before" picture.
