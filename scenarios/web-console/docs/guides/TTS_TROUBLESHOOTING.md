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

If Settings shows `Claude hook: Not registered`, or you saw `hook token not available after 5 attempts`, the API had not created the token file before hook registration ran. Fix:
```bash
source lib/tts-hooks.sh && wc::register_tts_hook
```

Important: the hook is only auto-registered by the scenario `Makefile` path today. If you started the scenario another way, confirm registration manually.

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

If the terminal was cleared, the wrong pane/session was targeted, or the text was never displayed, the correlation check fails. The Settings panel now shows the last delivery result and reason from `/api/v1/tts/status`.

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
