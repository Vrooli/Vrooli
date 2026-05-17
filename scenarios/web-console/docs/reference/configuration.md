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

### Session & Memory

| Variable | Default | Range | Impact |
|----------|---------|-------|--------|
| `WC_TERMINAL_SCROLLBACK_LINES` | `10000` | 100–100,000 | Decoded scrollback lines retained by the per-session terminal emulator and replayed via the snapshot stream on every (re)connect. **Higher** = more history restored on reconnect at the cost of memory per idle session. **Lower** = lighter idle footprint, less history. |
| `WC_MAX_SESSIONS` | `0` (unlimited) | 0–1,000 | Maximum concurrent PTY sessions. Safety guardrail for resource-constrained systems. `0` = no limit. |
| `WC_CLIENT_CHANNEL_BUFFER` | `64` | 8–1,024 | Per-client output channel capacity. **Higher** = absorbs output bursts better, uses more memory. **Lower** = less memory, may drop frames from slow WebSocket consumers. |

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

- **Claude Code**: `claude --dangerously-skip-permissions`
- **Codex**: `codex --yolo`

The `TerminalLauncher` component accepts a `shortcuts` prop, enabling parent scenarios to inject custom shortcut lists without modifying the component. This is the extension point for OT-P1-002 (Shortcut Profile Management).

---

## Mobile Toolbar Keys

The mobile floating toolbar keys are defined in [CODE: ui/src/components/MobileToolbar.tsx] as `TOOLBAR_KEYS`. The default set covers essential terminal operations:

- Esc, Tab, Ctrl+C, Ctrl+D, Ctrl+Z, Arrow keys

The toolbar component accepts `visible` and `onInput` props. Keys can be customized by modifying the `TOOLBAR_KEYS` array.

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
currently recommends fast non-reasoning local candidates (`gemma3:4b`,
`gemma3n:e2b`, `llama3.2:3b`, `llama3.2:1b`, `qwen2.5:3b`,
`phi4-mini:3.8b`) and marks reasoning models (`qwen3:*`,
`deepseek-r1:*`) as unsuitable for default TTS summaries. Missing recommended
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
