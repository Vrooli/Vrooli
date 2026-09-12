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
| `WC_BRIDGE_URL` | Optional server-side Bridge address; empty resolves `vrooli-bridge` on this machine. A non-empty value must be reachable from the Web Console host. |
| `VROOLI_BRIDGE_API_TOKEN` | Explicit fallback Bridge credential; an enrolled local operator session is preferred when available |
| `VROOLI_BRIDGE_REAUTH_TOKEN` | Optional short-lived owner re-authentication proof |
| `VROOLI_BRIDGE_LABEL` | Optional launcher label; defaults to `Bridge fleet` |

When `WC_BRIDGE_URL` points to another machine, the Web Console sends the
server-held owner credential and remote shell traffic over that connection.
Use `https://` (and the resulting `wss://` session transport) for non-loopback
addresses unless an explicitly trusted deployment overrides that policy.

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

The terminal launcher shows one card per coding agent the selected machine
reports, plus the empty shell and the editor card. Three different facts meet in
that grid and they come from different owners:

| Fact | Owner | Where |
|------|-------|-------|
| Which agents exist, and their state and version | the machine's capability probe | [CODE: packages/capabilityprobe] via the target catalog |
| Their display names | the same probe, forwarded intact | `capability:<id>` readiness facts |
| Their order, and the command each one runs | the operator's shortcut profile | `shortcut_profiles`, served by `GetEffective` |

The rule is one sentence: **membership from the probe, order and command from
the profile, built-ins only where the profile is silent.** The built-in verbs
live in `FALLBACK_COMMANDS` ([CODE: ui/src/components/launcher/agentGrid.ts])
and apply only to an agent the profile has never mentioned — they never
override a command the operator saved.

`DEFAULT_SHORTCUTS` in [CODE: ui/src/consts/shortcuts.ts] is a **fallback for a
failed `GetEffective` call, not a source of truth.** The server owns the
defaults ([CODE: api/shortcut_profiles.go]); keep the two identical, and prefer
deleting an entry from the client copy over adding one. The two drifted once —
the client kept an eight-entry list with `(attributed)` duplicates long after
the server dropped them, and the launcher grew a module whose only job was to
fold duplicates that only the client still produced.

### Agent identity

Each shortcut entry carries an optional `agent_id` naming one catalogued agent,
or nothing for a plain operator command. The server resolves it
([CODE: api/shortcut_agent_identity.go]) and backfills entries stored before the
field existed; no consumer pattern-matches command text to guess which agent an
entry launches.

### Reordering

The launcher's **Reorder** control rewrites the effective profile's entry order,
which is why `GetEffective` returns the profile's id, scope, and name alongside
the list: a client that edits the effective list has to write it back to the
profile it actually came from, without re-deriving scope priority for itself.
Dragging a grip and pressing `Alt` with the arrow keys go through the same move,
so the two can never disagree.

### Installing an agent

A card for a missing agent offers **Install**, which runs the governed resource
installer — locally through `LifecycleActionService`, or on a Bridge node
through the relay ([CODE: api/remote_capability_install.go]).

**An install is reported as done only when the machine itself reports the
agent.** An installer's exit code says a command ran; it does not say the binary
landed somewhere the machine looks. The two are routinely different, and the
outcome is therefore one of three values, defined in
[CODE: api/internal/capabilities/actions.go]:

| Status | Means | Card shows |
|--------|-------|------------|
| `installed` | the target now reports the capability | Installed, transiently |
| `unconfirmed` | the installer completed and the target has not reported it | Not confirmed · Retry |
| `failed` | the installer itself did not succeed | Install failed · Retry |

Confirmation is a re-read of the target's own capability report, not of the
installer's output: the local host answers immediately from its probe, and a
Bridge node answers through its next heartbeat inventory, so a remote install
waits across several heartbeats before reporting `unconfirmed`. Expiring that
window is not a failure — it is the honest answer that nothing is yet known,
and its remedy is to look again rather than to install again.

Both halves happen inside one HTTP response, so `capabilityInstallBudget`
(relay + confirmation) is sized under the server's `httpWriteTimeout`; a budget
larger than the write timeout cuts the response off mid-flight and the operator
learns nothing. `TestCapabilityInstallFitsInsideTheServerWriteTimeout` pins the
relationship. The relay's previous 180s ceiling was already over that budget on
its own.

`Def.OperatorCommand` in the capability registry serves two masters: it is the
argv that runs *and* the line rendered to an operator to paste. It therefore
names the CLI, and `operatorCommandArgs` drops exactly one leading token when it
does. Passing the whole string through as arguments produced
`vrooli vrooli resource install codex --json`, which is why local coding-agent
installs never worked.

### Attribution

Ambient attribution is not a shell function. It is delivered by the
`coding_agent_shims` host safeguard
([CODE: internal/safeguards/coding-agent-shims]), which installs one link per
agent in `~/.vrooli/shims` pointing at `vrooli-agent-launcher`. The launcher
attributes the run, materializes the session-scoped agent home, and then
replaces its own process image with the real agent, so nothing of it survives
into the agent's lifetime.

A shim is a real executable, so attribution reaches every shell, non-interactive
contexts, and agents started with `execve` — none of which a bash function could
cover.

**A bare agent command depends on that shim resolving first on `PATH`.** For
Codex and Grok that resolution is what creates the session-scoped home the
message-capture tailers read, so a command like `codex --yolo` records a
conversation only if the shim directory really is ahead of the real binary in
the shell the console spawned. The launch-command editor states this per entry
and offers the governed rewrite; the classification lives in
[CODE: ui/src/lib/captureSafety.ts].

The `TerminalLauncher` component accepts a `shortcuts` prop, enabling parent
scenarios to inject custom shortcut lists without modifying the component. A
caller that supplies the list owns it, so the dialog offers no reorder control
in that mode.

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
# Machine configuration through Bridge

The Machines surface is the desktop entry point for target-aware onboarding.
It sends the selected machine's generated `setup/v1.Selection` through Bridge,
renders the schema-backed question set, and reattaches to durable apply status
by run id. Secrets use the authenticated sealed credential-grant path; they are
not placed in ordinary configuration answers or command arguments.

Machine detail shows readiness, desired and applied policy evidence, typed
drift, outstanding questions, credential receipts, audit events, and terminal
apply outcomes. Re-apply uses the same durable status surface.
