# Web Console Configuration Reference

This document describes the **control surface** for the Web Console scenario: the small set of meaningful, safe levers that shape its behavior without requiring code changes.

## Design Principles

- **Fewer, well-chosen knobs** over many obscure settings
- Every lever maps to a real tradeoff operators care about
- Sane defaults that work for common usage — tuning is optional
- Extreme values degrade gracefully rather than crash
- Values are clamped to safe ranges; invalid input falls back to defaults

---

## API Levers (Environment Variables)

Implementation: [CODE: api/config.go#LoadConfig]

Set these in your environment or `.env` file before starting the API.

### Optional Bridge Federation

The web-console exposes registered Bridge nodes as terminal-launcher targets.
The server owns the Bridge connection and projects only safe metadata to the UI
and CLI. If configuration or registry access is incomplete, the catalog reports
an explicit state (`UNCONFIGURED`, `CONFIGURED_EMPTY`, or `REGISTRY_ERROR`) and
keeps unavailable nodes visible with readiness facts and a recovery action.
Tokens are server-side only.

| Variable | Purpose |
|----------|---------|
| `VROOLI_BRIDGE_NODE_ID` | Optional registered node identity retained for a single-node Bridge deployment |
| `VROOLI_BRIDGE_API_TOKEN` | Explicit fallback Bridge credential; an enrolled local operator session is preferred when available |
| `VROOLI_BRIDGE_REAUTH_TOKEN` | Optional short-lived owner re-authentication proof |
| `VROOLI_BRIDGE_LABEL` | Optional launcher label; defaults to `Bridge fleet` |

Remote sessions currently report `survives_restart=false`: the Web Console
keeps the short-lived federation registry in memory and preserves reconnect
sequence state while it is running. A future Bridge-owned lease/session
contract is required before remote restart recovery can be promised.

### Session & Memory

| Variable | Default | Range | Impact |
|----------|---------|-------|--------|
| `WC_TERMINAL_SCROLLBACK_LINES` | `50000` | 100–100,000 | Decoded scrollback lines retained by the per-session terminal emulator and replayed via the snapshot stream on every (re)connect. **Higher** = more history restored on reconnect at the cost of memory per idle session. **Lower** = lighter idle footprint, less history. |
| `WC_MAX_SESSIONS` | `0` (unlimited) | 0–1,000 | Maximum concurrent PTY sessions. Safety guardrail for resource-constrained systems. `0` = no limit. |
| `WC_CLIENT_CHANNEL_BUFFER` | `256` | 8–1,024 | Per-client output channel capacity. **Higher** = absorbs output bursts better, uses more memory. **Lower** = less memory, may drop frames from slow WebSocket consumers. |
| `WC_INPUT_QUEUE_SIZE` | `256` | 16–4,096 | Bounded ordered input requests per session. A full queue returns a typed input acknowledgement so the client can retry. |

### Performance Tuning

| Variable | Default | Range | Impact |
|----------|---------|-------|--------|
| `WC_PTY_READ_BUFFER` | `4096` | 512–65,536 | Byte size of the PTY read buffer. **Higher** = fewer syscalls for high-throughput terminals. **Lower** = less per-session memory. Most operators never need to change this. |
| `WC_WS_BUFFER_SIZE` | `4096` | 512–65,536 | WebSocket read/write buffer size. **Higher** = better throughput for heavy output. **Lower** = less memory per connection. |

### Terminal Defaults

| Variable | Default | Range | Impact |
|----------|---------|-------|--------|
| `WC_DEFAULT_COLS` | `80` | 20–500 | Default terminal columns when the client doesn't specify. Used as fallback — the UI normally sends actual terminal dimensions on connect. |
| `WC_DEFAULT_ROWS` | `24` | 5–200 | Default terminal rows when the client doesn't specify. |
| `WC_DEFAULT_SHELL` | `$SHELL` or `/bin/sh` | any path | Shell binary to launch for new sessions. Respects the `$SHELL` environment variable by default. Override to force a specific shell for all sessions. |

### Session Backend & Policy Defaults

| Variable | Default | Values | Impact |
|----------|---------|--------|--------|
| `WC_DEFAULT_BACKEND` | `auto` | `auto`, `standard`, `persistent` | Default session backend. `auto` (default) resolves to `persistent` when tmux is available, `standard` otherwise. `persistent` uses tmux for restart-survivable sessions. |
| `WC_DEFAULT_POLICY_MODE` | `never` | `never`, `preset`, `custom` | Default expiration policy mode for new sessions. |
| `WC_DEFAULT_POLICY_DURATION` | *(empty)* | `1h`, `8h`, `24h`, or Go duration | Default expiration duration. Only used when policy mode is `preset` or `custom`. |

### Archive Retention

All archive-retention limits are opt-in. Inspect them with `web-console session archive-retention`; preview actions with `web-console session archive-prune` before using `--apply`.

| Variable | Default | Range | Impact |
|----------|---------|-------|--------|
| `WC_ARCHIVE_MESSAGELESS_AGE_DAYS` | `0` (unlimited) | 0–36,500 days | Makes explicitly archived rows with no messages eligible for permanent transcript deletion after this age. Message-bearing transcripts are retained. |
| `WC_ARCHIVE_AGENT_HOME_AGE_DAYS` | `0` (unlimited) | 0–36,500 days | Prunes exact session-owned agent history after this age. The conversation stays searchable and its restore state becomes `Read-only`. |
| `WC_ARCHIVE_MAX_BYTES` | `0` (unlimited) | 0–2^62 bytes | Soft ceiling across measured archive transcript and agent-history bytes. Size pressure prunes agent history before any eligible message-less transcript. |
| `WC_CONVERSATION_RETENTION_DAYS` | `180` | 0–36,500 days | Automatically removes conversation events older than this age. Zero disables age-based retention. The sweep is bounded per cycle. |
| `WC_CONVERSATION_MAX_EVENTS_PER_SESSION` | `5000` | 0–1,000,000 events | Retains the newest events per session up to this limit. Zero disables the per-session cap. |

---

## UI Constants

UI configuration is centralized in [CODE: ui/src/consts/config.ts]. These are compile-time constants — change them and rebuild the UI.

### Terminal Appearance

| Constant | Default | Purpose |
|----------|---------|---------|
| `TERMINAL_THEME` | Slate-950 dark theme | xterm.js color theme (background, foreground, cursor, selection) |
| `TERMINAL_FONT_SIZE` | `14` | Terminal font size in pixels. Range 8–24 recommended. |
| `TERMINAL_FONT_FAMILY` | JetBrains Mono stack | Monospace font stack. First available font is used. |

### Terminal Defaults

| Constant | Default | Purpose |
|----------|---------|---------|
| `DEFAULT_COLS` | `80` | Columns requested when creating a session via the UI |
| `DEFAULT_ROWS` | `24` | Rows requested when creating a session via the UI |

### Pane Layout

| Constant | Default | Purpose |
|----------|---------|---------|
| `PANE_MIN_WIDTH_PX` | `500` | Minimum pane width before grid wraps to a new row. Lower = more panes side-by-side. Higher = more readable per pane. |
| `PANE_MIN_HEIGHT_PX` | `300` | Minimum pane height when >2 panes are open. Prevents panes from becoming too small. |

### Connection

| Constant | Default | Purpose |
|----------|---------|---------|
| `HEALTH_RETRY_COUNT` | `3` | How many times to retry the API health check on startup |
| `HEALTH_RETRY_DELAY_MS` | `1000` | Milliseconds between health check retries |

---

## Launcher Shortcuts

The terminal launcher presents configurable shortcut entries alongside the empty shell option. Default shortcuts are defined in [CODE: ui/src/consts/shortcuts.ts] as `DEFAULT_SHORTCUTS`:

- **Claude Code**: `vrooli agent launch --runner claude --arg=--dangerously-skip-permissions`
- **Codex**: `codex --yolo`
- **Attributed Claude Code**: checks for `vrooli-agent-launcher`, then runs the
  same direct Claude command when the launcher is absent.
- **Attributed Codex**: checks for `vrooli-agent-launcher`, then runs the same
  direct Codex command when the launcher is absent.
- **Attributed OpenCode** and **Attributed Grok** use the same additive
  preflight pattern.

Ambient attribution is no longer a shell function. It is delivered by the
`coding_agent_shims` host safeguard
([CODE: internal/safeguards/coding-agent-shims]), which installs one link per
agent in `~/.vrooli/bin` pointing at `vrooli-agent-launcher`. The launcher
attributes the run and then replaces its own process image with the real agent,
so nothing of it survives into the agent's lifetime.

A shim is a real executable, so attribution reaches every shell, non-interactive
contexts, and agents started with `execve` — none of which a bash function could
cover. The shell-function version required an `~/.bashrc` edit that nothing
installed, upgraded, or removed; if a host still has that block, delete it. The
shims make it redundant, and leaving it in place only adds a hop.

The `TerminalLauncher` component accepts a `shortcuts` prop, enabling parent scenarios to inject custom shortcut lists without modifying the component. This is the extension point for OT-P1-002 (Shortcut Profile Management).

---

## Mobile Toolbar Keys

The escape sequences the toolbar can send are defined in [CODE: ui/src/lib/terminalKeys.ts] as `TOOLBAR_KEYS`. The default set covers essential terminal operations:

- Esc, Tab, Ctrl+C, Ctrl+D, Ctrl+Z, Arrow keys

The toolbar component accepts `visible` and `onInput` props. Sequences can be customized by modifying the `TOOLBAR_KEYS` array.

---

## Mobile Toolbar Layout

Which controls appear, how large they are, and how many rows they occupy are three
independent settings, not one. They live in the persisted workspace store as
`toolbarPrefs` and are **device-local**: screen size belongs to the phone, not the
account, so they are never synced (unlike shortcut profiles, which are per-user).

| Setting | Values | Owned by |
|---|---|---|
| `enabled` | per-control visibility | the user |
| `density` | `compact` 32px · `standard` 40px · `large` 44px | the user |
| `maxRows` | `1` · `2` · `3` | the user (a ceiling, not a hint) |
| `arrows` | `dpad` (two rows tall) · `inline` (one row) | the user |
| `overflow` | `strip` · `more` | the user |
| the arrangement | — | `layoutToolbar()` |

Arrangement is computed, never authored. [CODE: ui/src/lib/toolbarLayout.ts] exports
`layoutToolbar(prefs, availableWidth, options)`, a pure function that seats controls in
priority order and returns the rows to paint. It replaces flex-wrap, which has no
ceiling and let the toolbar grow to four rows on a 390px phone.

### Invariants

`layoutToolbar` guarantees three things, each covered by
[CODE: ui/src/__tests__/toolbarLayout.test.ts]:

1. **The budget is a ceiling.** `rowCount <= prefs.maxRows` at every width and density.
   A D-pad needs two rows, so a one-row budget degrades it to the inline run rather
   than spending a row the user did not authorise.
2. **Overflow is never unreachable.** `more` is *pinned*: seated before every other
   control and never allowed to overflow. It is the surface hidden and overflowed
   controls live in, so stranding it would strand everything it holds — which is why
   it is not a toggle.
3. **Priority is respected.** A control only loses its seat to a lower-priority control
   that is no wider than it. `TOOLBAR_CONTROLS` is the priority order, and the settings
   checklist renders it top to bottom so the order is user-visible.

### Adding a control

Add one entry to `TOOLBAR_CONTROLS` and one case to `renderToolbarControl` in
[CODE: ui/src/components/toolbar/toolbarControls.tsx]. Do not add a branch to a
component: the toolbar, the settings preview, and the More sheet all render through
that one function, at sizes the engine supplies. `ToolbarControlId` is an open string
type so pinned shortcuts (`shortcut:<id>`) can join without a schema change.

### Preview fidelity

The settings preview calls the same `layoutToolbar` with a simulated width instead of a
measured one, and paints it with the same `ToolbarSurface`. There is no second
implementation for it to drift from. Adding preview-only layout code would break that
property and should be treated as a defect.

### Defaults

Three presets ship — `dense`, `balanced` (the default), `essential` — plus `custom`,
which any individual edit switches to. All three keep **More**, **Snippets**, and **image upload** on
(screenshots are how coding agents are usually driven) and leave **AI suggest** off.
Densities below 44px warn about the recommended touch target but are never blocked:
trading target size for more controls per row is a legitimate preference.

---

## Audio Summarization

Web Console reads and updates TTS summarization through its own
`AudioAdminService`, which forwards to `audio-tools` `SummarizeService`.
Operators should use Settings -> Voice Output -> Summarization rather than
editing JSON by hand.

| Lever | Default | Safe range / values | Impact |
|-------|---------|---------------------|--------|
| Enabled | `true` | on/off | Enables automatic summarization before TTS for long assistant messages. |
| Character threshold | `500` | 100-10,000 in UI | Messages shorter than this are spoken as-is. Higher values reduce summarizer calls. |
| Level | `moderate` | `light`, `moderate`, `heavy` | Controls how aggressively audio-tools shortens speech text. |
| Model | `llama3.2:3b` fallback | installed Ollama model from the settings picker | Local Ollama model used by the summarize provider. The picker shows installed, missing, recommended, and reasoning models. |
| Timeout seconds | `120` | 15-300 in UI | Per-summary deadline. It must stay below web-console's 150s audio-tools call timeout and audio-tools' 180s HTTP write timeout. |

Model defaults are policy-driven, not release-hype-driven. The catalog
recommends **fast, non-reasoning** local models (small instruct/chat-tuned
models track the installed set; see `resource-ollama policy roles`) and marks
reasoning-tuned models as unsuitable for default TTS summaries because they are
slower and spend output budget on internal reasoning. Missing recommended
models show an `ollama pull <model>` command; Web Console never pulls models
automatically.

The default remains `llama3.2:3b` until a locally installed newer candidate
beats it on both latency and summary quality. If summarization fails, the UI
distinguishes audio-tools unreachable, deadline exceeded, and selected model
not installed.

---

## What Is NOT Exposed (and Why)

These internal details are intentionally kept as implementation constants:

| Value | Reason |
|-------|--------|
| WebSocket message type strings (`stdin`, `stdout`, etc.) | Protocol contract — changing these breaks client-server compatibility |
| PTY read loop goroutine behavior | Internal concurrency pattern, not a behavioral tradeoff |
| Session UUID format | Implementation detail, no operational impact |
| Drawer width (288px) | CSS layout detail, not a behavioral lever |
| Health check version string (`1.0.0`) | API metadata, not a tuning knob |
| Terminal `cursorBlink` and `allowProposedApi` | Always-on features with no meaningful off state |

---

## Recommended Configurations

### Resource-Constrained System
```bash
WC_MAX_SESSIONS=5
WC_TERMINAL_SCROLLBACK_LINES=2000
WC_CLIENT_CHANNEL_BUFFER=16
```

### High-Throughput Terminal (heavy build output)
```bash
WC_PTY_READ_BUFFER=16384
WC_WS_BUFFER_SIZE=16384
WC_TERMINAL_SCROLLBACK_LINES=50000
WC_CLIENT_CHANNEL_BUFFER=256
```

### Default (no changes needed)
All defaults are designed for the common case: a single operator running a few terminal sessions on a personal server.
