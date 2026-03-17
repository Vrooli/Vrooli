# Auto-TTS Troubleshooting

## "Auto-TTS is enabled but nothing plays"

Work through these checks in order. Each step addresses the most common failure point at that stage of the pipeline.

### Check 1: Is Kokoro running?

Open Settings -> Voice Output (TTS) and press `Refresh`. The runtime checks block now shows the Kokoro status message returned by `/api/v1/tts/status`.

From the API:
```bash
curl http://localhost:<port>/api/v1/capabilities/liveness
```
Look for `kokoro-tts` with `status: "available"`. If unavailable, start the Kokoro resource:
```bash
vrooli resource start kokoro
```

Note: liveness is cached client-side for up to 30 seconds. Use the Settings `Refresh` button to force a fresh decision.

### Check 2: Is the Claude hook registered?

During `make start`, you should see:
```
tts-hook: registered Stop hook -> localhost:<port>
```

The hook is now reconciled by the `claude-code` resource and written to the project-level Claude file at `.claude/settings.json` in the repository root. `tts-hooks.sh` no longer writes the file directly; it delegates to the resource-owned reconciliation seam.

If Settings shows `Claude hook: Not registered`, `hook_missing`, or `hook_stale`, or you saw `hook token not available after 5 attempts`, fix:
```bash
source lib/tts-hooks.sh && wc::register_tts_hook
```

To inspect the project-level Claude settings file directly:
```bash
cat /path/to/repo/.claude/settings.json
```

You should see a `hooks.Stop[].hooks[]` entry with `_id: "web-console-tts"` and `type: "http"`.

Important:
- The canonical project hook file is repo-root `.claude/settings.json`
- `~/.claude/settings.json` is global, not project-level
- Starting web-console through the lifecycle now auto-heals the project hook via the `claude-code` resource

### Check 3: Is auto-TTS enabled?

In the Settings modal, the "Auto-speak AI responses" toggle must be ON.

From the API:
```bash
curl http://localhost:<port>/api/v1/tts/config
```
Verify `autoEnabled: true`. The server-backed settings also include `backend`, `kokoroVoice`, and `kokoroSpeed`.

### Check 3.5: What does "Active backend" mean?

- `Preference` is what you asked for.
- `Active backend` is what the current tab is actually using.
- `auto` means "prefer Kokoro, otherwise browser".
- `kokoro` means strict Kokoro only. If Kokoro is unavailable, active backend becomes `Unavailable` with an explanation instead of silently switching to browser.

### Check 4: Text correlation

`deliverTTS` validates that the AI response text appears in the target terminal's recent output buffer. This prevents stale or spoofed text from being spoken.

If the terminal was cleared, the wrong pane/session was targeted, or the text was never displayed, the correlation check fails. The Settings panel now shows the last Claude-hook delivery result and the last Codex-tailer delivery result separately from `/api/v1/tts/status`.

### Check 5: Browser audio policy

Browsers require a user interaction (click, keypress) before audio can play. Settings now shows `Browser audio: Blocked until you interact with the page`. Click or press a key in the page, then run `Test`.

### Check 6: Backend fallback

If `backend=auto` and Kokoro fails at runtime, the frontend attempts a browser fallback and updates the backend reason accordingly. If both fail, a transient amber error banner appears in the terminal pane for 5 seconds.

### Check 7: Use the built-in Test button

Settings -> Voice Output (TTS) -> `Test`

This plays a short sample through the current runtime backend decision and is the fastest way to isolate:
- bad Kokoro availability
- browser audio lockout
- strict `kokoro` mode with Kokoro down
- browser-only playback issues

## "TTS plays but sounds wrong"

- **Speed**: Adjust in Settings > Kokoro Speed (0.5-4.0x)
- **Voice**: Change in Settings > Kokoro Voice dropdown
- **Choppy playback**: Text is split on double-newlines into paragraphs and spoken sequentially. Very long responses may have noticeable gaps between paragraphs.
