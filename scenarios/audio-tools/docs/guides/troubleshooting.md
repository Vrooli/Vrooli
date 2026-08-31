# Troubleshooting — Audio Tools

Common issues that surface across any scenario built from the
`react-vite` template. Scenario-specific issues belong in
[`internal/PROBLEMS.md`](../internal/PROBLEMS.md), not here.

## Lifecycle and ports

### "Port already in use" or "address already in use"

A previous scenario instance still holds the port. The lifecycle
allocates from declared ranges (`API_PORT 15000-19999`,
`UI_PORT 20000-24999`), so collisions usually mean a previous run
did not stop cleanly.

```bash
make restart
# or, if that doesn't recover:
make stop
make start
```

If a process is genuinely orphaned, let the control plane reconcile its owned state:

```bash
vrooli scenario status audio-tools
make stop
vrooli scenario start audio-tools --clean-stale
```

**Don't** use `make stop && make start` on autopilot — `make restart`
is the canonical command and gives the lifecycle a chance to clean
state in order.

### `vrooli: command not found`

The Vrooli CLI isn't on your `PATH`. Run the workspace-root setup:

```bash
cd ../../..   # to the Vrooli repo root
make setup
```

Then re-source your shell or open a new terminal.

### Scenario won't open in browser

Confirm the UI server is running:

```bash
make status
vrooli scenario port audio-tools UI_PORT
```

Then open `http://localhost:<UI_PORT>` directly. If the URL works
but `make open` doesn't, the issue is your default-browser handler,
not the scenario.

## API and CLI

### CLI: "API not available at http://localhost:..."

The API isn't running, or it's running on a different port than the
CLI is looking at. Resolution order is documented in
[`../reference/configuration.md`](../reference/configuration.md#api-base-resolution-precedence).

Quick fixes:

```bash
# Start the API if it's not running
make start

# Or let the CLI auto-start it
audio-tools status --auto-start

# Or override the API base for this invocation
audio-tools status --api-base "http://localhost:$(vrooli scenario port audio-tools API_PORT)/api/v1"
```

### CLI behaves like an old version after editing source

cli-core auto-rebuilds the binary when sources change, but only
before commands marked `NeedsAPI: true`. If your edit landed in a
non-API command, force a rebuild:

```bash
make setup   # rebuilds CLI from sources
```

### `audio-tools configure` doesn't persist

Check which config-file path resolved (precedence in
[`../reference/configuration.md`](../reference/configuration.md#cli-config-file)).
Most commonly `~/.vrooli/config/audio-tools/config.json`. The
parent directory is created on first write — if that fails, your
home directory is read-only or `XDG_CONFIG_HOME` is set to an
unwritable path.

### API returns `400 invalid_request` on multipart upload

Likely cause: the upload request is not valid `multipart/form-data` or
is missing the `file` part. Proto-typed operations use Connect-RPC and
will surface Connect codes such as `invalid_argument`; `invalid_request`
is reserved for REST exceptions like file upload.

## Build and dependencies

### `go build` fails with `cgo` errors

The template requires `CGO_ENABLED=0`. SQLite is via `modernc.org/sqlite`
(pure-Go), and the CI gate proves no C dependency has snuck in. If
your build fails with cgo errors, a recently added dependency wants C:

```bash
cd api && go list -deps -f '{{`{{.ImportPath}}: {{.CgoFiles}}`}}' ./... | grep -v ': $'
```

Replace the offending dependency with a pure-Go alternative.

### `pnpm install` fails or installs the wrong tree

The UI deliberately installs **outside** the workspace:

```bash
cd ui
pnpm install --ignore-workspace
```

`make setup` does this automatically. If you ran a plain `pnpm install`
from `ui/` and got the workspace tree, delete `ui/node_modules` and
re-run with `--ignore-workspace`.

### UI build is slow (5–10 minutes)

`vite build` processes 4400+ modules in the production bundle. This
is expected. Use `pnpm dev` (via `make start`) for the fast iteration
loop and reserve `pnpm build` for verification.

### `pnpm strings:check` fails in CI

The codegen is out of sync with `en.json`. Regenerate locally:

```bash
cd ui
pnpm strings:gen
git add src/consts/strings.generated.ts
```

Commit `en.json`, the locale files, **and** `strings.generated.ts`
together — never one without the others.

## Tests

### Vitest test fails with `t('app.title')` returning the literal key

That's by design. The test runner sets i18next to `cimode` so
component tests are copy-independent. Assert against the typed
`strings.x.y` registry, not real translations. If your test needs
real-locale behaviour, opt back in with `await setLocale("en")` in
its own `beforeEach` (see `App.test.tsx` for an example).

### Go test fails with `dial tcp: connection refused` on `httpx.NewLiveServer`

The harness binds to `127.0.0.1:0` and lets the OS assign a port. If
this fails, you have a system-level limit (`ulimit -n`, IPv4 disabled).
Increase open-file limits or check that loopback is reachable.

### API E2E test (`go test -tags=e2e`) hangs

The E2E harness boots the actual binary and waits for `/health`. If
schema bootstrap fails (corrupt SQLite file, unwritable data
directory), `/health` never returns ready and the test times out.
Wipe the test data dir and retry. The default lives under
`${XDG_DATA_HOME:-~/.local/share}/vrooli/audio-tools/`.

### Coverage gate fails (`API coverage 71.4% < 75%`)

Coverage dropped below the floor. The fix is to add tests to the
file the report names — never to lower the threshold. Floors live in
[`../internal/TESTING.md`](../internal/TESTING.md#coverage-thresholds).

## Storage

### The resolved database directory is not writable

The default route is `${SCENARIO_DATA_DIR}/audio-tools.db` via
`api-core/storage`. If your filesystem is unusual (read-only home,
strict sandboxing), override:

```bash
# Redirect the whole storage tree, not one database file. The root is
# scenario-agnostic, so every scenario beneath it still resolves to its own
# separate path.
export VROOLI_STORAGE_ROOT=/tmp/vrooli-storage
make start
```

The schema is embedded and idempotent, so a fresh path is always
safe.

### "database is locked"

SQLite single-writer behaviour. If two processes (e.g., a stale API
plus a new one) hold the file, find and kill the older one:

```bash
fuser "${SCENARIO_DATA_DIR}/audio-tools.db"
```

`make stop` followed by `make start` is usually sufficient.

## Proto codegen

### "type not found" after editing a `.proto`

You haven't regenerated. From the workspace root:

```bash
make generate
```

The generator runs entirely on local plugins (no BSR network calls) and
writes to language-specific output paths: Go under
`packages/proto/gen/go/audio-tools/v1/`, TypeScript under
`packages/proto/gen/typescript/audio-tools/v1/`, and Python under
`packages/proto/gen/python/audio_tools/v1/`.

### Codegen ran but Go imports still fail

The generated package paths follow `packages/proto/gen/go/audio-tools/v1/<domain>/`.
After a rename, run `go mod tidy` from the affected module (`api/`
and `cli/`) so the import paths resolve.

## Audio capture and providers

These entries are specific to audio-tools (mic, browser audio formats,
streaming, BYOK). For deeper architectural background, follow the
cross-reference into `internal/PROBLEMS.md`.

### Mic permission denied

**Symptom:** The voice panel will not start recording. The
`MicReadinessIndicator` component renders with `data-state="denied"`
and the label "Microphone denied". The browser may also show a struck-
through microphone icon in the address bar. In the JavaScript console
you will see a `NotAllowedError` from `navigator.mediaDevices.getUserMedia`.

**Diagnosis:** The browser is gating mic access. Common causes:

- The user clicked **Block** when first prompted (the choice is sticky
  per origin).
- The page is being served over plain HTTP from a non-localhost origin
  (browsers only expose `getUserMedia` on secure contexts).
- An OS-level permission (Windows, macOS, GNOME, KDE) is denying the
  browser itself from accessing the mic.

`MicReadinessIndicator` reports the four `PermissionState` values
returned by the Permissions API: `granted`, `denied`, `prompt`, and
`unknown` (the API is unavailable or threw). If you see `prompt`, the
browser will ask the next time the user clicks record; if you see
`denied`, the user must reset the site permission manually.

**Fix:**

- **Chrome / Edge:** click the lock or "tune" icon in the address bar,
  open **Site settings**, set **Microphone** to **Allow**, then
  reload.
- **Firefox:** click the lock icon, expand **Connection secure**, then
  use **Clear permissions and reset** for the site (or open
  `about:preferences#privacy` → **Permissions** → **Microphone** →
  **Settings**).
- **Safari:** open **Safari → Settings → Websites → Microphone**,
  switch the entry for this origin to **Allow**, then reload.
- **OS check:** confirm the browser itself has mic access in the OS
  privacy panel (System Settings on macOS, Settings → Privacy on
  Windows, `gnome-control-center sound` on Linux).

After a successful grant, the indicator transitions to `granted` /
"Microphone ready" without a page reload.

### Browser audio format mismatch (WebM/Opus vs PCM)

**Symptom:** The browser successfully captures audio and uploads it,
but transcription either fails with a decode error or returns garbage
text. In DevTools → **Network**, the request to
`/api/v1/voice/stream` (WebSocket) or the streaming Connect RPC shows
binary frames whose first bytes are `1A 45 DF A3` (the EBML / WebM
magic) instead of a raw PCM payload.

**Diagnosis:** The audio-tools streaming endpoint expects **raw 16-bit
PCM at 16 kHz**. The `MediaRecorder` API in browsers emits
**WebM/Opus** by default. Sending WebM into the streaming path makes
mid-stream slices undecodable by the strategy layer, because the
strategy never sees the WebM init segment. This is the same root cause
documented in `internal/PROBLEMS.md` under
"Browser WebM partial-decoding regression after HandleStreamWS deletion".

To confirm in DevTools:

1. Open the **Network** tab, filter for `voice/stream` or the
   streaming RPC.
2. Click the request, open the **Messages** tab (WS) or **Payload**
   tab (Connect).
3. Inspect the first outbound binary frame. If it begins with
   `1A 45 DF A3`, the client is sending WebM. PCM frames have no
   magic — they look like raw little-endian int16 samples.

**Fix:** the multipart upload path decodes WebM/Opus correctly because
the server has the full container before it starts. Until the embed
emits PCM directly, use the unary transcription endpoint
(multipart upload of the captured `Blob`) instead of the streaming
path. From the CLI:

```bash
audio-tools stt transcribe --file ./capture.webm
```

From the UI, prefer the "Upload audio" affordance over "Live
transcription". The buffered path returns a correct final transcript
in every case.

### Zero-partial streaming (no live transcripts)

**Symptom:** A streaming transcription session connects and completes
successfully, but only the **final** transcript ever arrives. No
interim "partial" events render in the UI. The unary / buffered path
returns the same final text correctly.

**Diagnosis:** Streaming providers are declared on the wire but not
implemented yet. `chain.Stream` falls back to the buffered unary path
in every case, so the `DoneEvent.FellBackToUnary` flag is set to
`true` and no partials are emitted. See `internal/PROBLEMS.md` entry
**"Streaming providers declared but not implemented"** for the full
explanation and the planned fix.

**Fix:** use the unary path explicitly until streaming partials land.

- **UI:** select **Upload audio** (or any non-streaming variant) in
  the voice panel.
- **CLI:** use `audio-tools stt transcribe --file "<file>"` — it targets
  the unary endpoint.
- **Embed integrators:** call `transcribe()` (unary) instead of
  `transcribeStream()` until the embed signals streaming readiness.

There is no functional gap — every consumer still sees the final
transcript — only the latency benefit of live partials is missing.

### WebSocket provider routing

The browser WebSocket path now uses the same `sttchain` provider chain as
the Connect transport. A configured server-side BYOK credential is applied
when the browser cannot attach custom handshake headers; explicit request
credentials still take precedence. Provider identity and terminal routing
metadata are emitted on the stream, so the browser path and Connect path
have the same provider-selection semantics.

If the stream reports `provider_failure`, inspect the provider identity and
the capability projection first. Configure or repair the named provider from
**Settings → Providers**, or use the unary upload path when a provider does
not support native streaming.

### Voice works but sounds different

**Symptom:** Voice output works, but the voice, pronunciation, or timing is
different from the local Kokoro result.

**Diagnosis:** The selected provider is the declared browser speech tier or a
BYOK provider. Browser speech synthesis is last-resort, browser-dependent
output and does not guarantee Kokoro voice quality. Check the response tier
and the capability feature status before treating this as an audio failure.

**Fix:** Start Kokoro for local output, or configure a BYOK TTS credential in
**Settings → Providers**. If browser output is intentional, select a browser
voice explicitly and treat its quality as host-dependent.

### Voice is disabled and the engines are healthy

**Diagnosis:** Inspect the consumer feature projection and the audio provider
rollup. A healthy scenario can still have one unavailable feature, while a
stopped scenario has no feature projection:

```bash
curl -s -X POST localhost:$WEB_CONSOLE_PORT/vrooli.web_console.v1.capabilities.CapabilitiesService/Liveness \
  -H 'Content-Type: application/json' -d '{}' | jq '.capabilities[] | select(.id=="audio-tools")'
curl -s -X POST localhost:$AUDIO_TOOLS_PORT/vrooli.audio_tools.v1.health_status.HealthStatusService/GetProviderHealth \
  -H 'Content-Type: application/json' -d '{}' | jq '.capabilities'
```

Gate on the needed `featureStatus` entry. If it is unavailable, follow its
`featureReason` and `featureOperatorCommand`; for an unsupported speaker
provider, inspect `vrooli resource status sherpa-onnx --json` rather than
trying to start it.

### Provider BYOK missing or invalid key

**Symptom:** STT or TTS calls that should use a BYOK provider fall back
to the local provider (or to Vrooli tier, depending on configuration)
and the response metadata shows the requested provider was skipped.
Server logs contain a "byok envelope incomplete" or "byok credential
not found" message. From a browser tab, the request to the Connect
RPC is missing the `X-Audio-BYOK-Provider` and `X-Audio-BYOK-Key`
headers; or they are present but the key is rejected upstream
(`401`/`403` from the third-party API surfaces as a chain fallback).

**Diagnosis:** The BYOK envelope is built from four headers that
travel together on every Connect, multipart, and bidi request:

| Header | Purpose |
|---|---|
| `X-Audio-BYOK-Provider` | Which adapter to use (e.g. `openai`, `deepgram`, `elevenlabs`). |
| `X-Audio-BYOK-Key` | The user's API key for that provider. |
| `X-Audio-LPBS-Token` | Optional LPBS-issued token for tiered routing. |
| `X-Audio-User-Identity` | Caller identity for per-user accounting. |

If `Provider` or `Key` is missing or empty, the envelope is treated as
absent and the chain falls back to the next provider in its preference
order. If the key is present but the upstream provider rejects it,
you will see a `401`/`403` in the server log followed by a chain
fallback.

**Fix:** set the keys through the configuration UI rather than
patching headers by hand.

1. Open the audio-tools UI and navigate to **Settings → Providers**
   (the Voice Settings panel in the embed exposes the same surface).
2. Pick the provider (OpenAI, Deepgram, ElevenLabs, OpenRouter, etc.)
   and paste the API key. The UI calls
   `SettingsService.UpsertBYOKCredential`, which persists the
   credential and starts attaching the envelope headers to subsequent
   requests automatically.
3. Re-issue the transcription or synthesis request. Confirm in
   DevTools → **Network** that the request now carries
   `X-Audio-BYOK-Provider` and `X-Audio-BYOK-Key`. The response
   metadata should name the BYOK provider instead of the local
   fallback.

To verify from the CLI:

```bash
audio-tools settings byok-list
audio-tools settings provider
```

A stored credential that still produces a fallback usually means the
upstream key has been revoked or is missing required scopes — rotate
the key in the third-party dashboard and re-upsert it.

## When to add a new entry here

Add to this guide if:

- The issue can occur in **any** scenario from this template
- The root cause is non-obvious from the error message
- The fix is a stable, repeatable command

Add to [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) if:

- The issue is specific to your scenario
- It's tech debt or a known workaround pending a real fix
- It needs scenario-specific context to act on

## Cross-references

- [`../QUICKSTART.md`](../QUICKSTART.md) — first-touch setup
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and config precedence
- [`../reference/cli-commands.md`](../reference/cli-commands.md) — CLI command reference
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns and coverage gates
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — scenario-specific issues
