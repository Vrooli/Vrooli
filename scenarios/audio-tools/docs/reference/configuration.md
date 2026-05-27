# Configuration — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Environment variables

| Variable | Default | Effect |
|---|---|---|
| `SQLITE_PATH` | resolved via `api-core/storage` | SQLite DB file path. |
| `AUDIO_AI_ENABLE_BYOK` | `true` | Enables the BYOK tier in all three provider chains. |
| `AUDIO_AI_ENABLE_VROOLI` | `false` | Enables the Vrooli/LPBS tier. Defaults off until `execute/lpbs-audio-gateway-endpoints` ships. |
| `AUDIO_AI_ENABLE_LOCAL` | `true` | Enables the Local tier. |
| `AUDIO_WHISPER_URL` | `http://localhost:8090` | Local STT resource. |
| `AUDIO_KOKORO_URL` | `http://localhost:8880` | Local TTS resource. |
| `AUDIO_OLLAMA_URL` | `http://localhost:11434` | Local summarize resource. |
| `AUDIO_LPBS_BASE_URL` | `""` | LPBS base URL (Vrooli tier). |
| `AUDIO_LPBS_APP_BUNDLE_KEY` | `audio-tools` | LPBS app-bundle key for usage attribution. |
| `AUDIO_AVAIL_TTL_BYOK` | `5m` | BYOK availability cache TTL. |
| `AUDIO_AVAIL_TTL_VROOLI` | `30s` | Vrooli availability cache TTL. |
| `AUDIO_SUMMARIZE_DEFAULT_MODEL` | `llama3.2:3b` | Local summarize default model. Empty or known reasoning defaults are coerced to the safe fallback at startup. |

## Service manifest (`.vrooli/service.json`)

Declares the resource dependencies (`whisper`, `kokoro`, `ollama`, `postgres`) — all `required: false` so audio-tools starts cleanly with zero local resources — and the LPBS scenario dependency (`required: false`, flag-off).

How this scenario is configured — env vars consumed by the binaries,
the `.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every
required variable automatically. You only need this reference when
running a binary by hand or when a scenario adds a new variable.

## Environment variables

### Required at runtime (set by the lifecycle)

| Variable | Range / format | Purpose |
|---|---|---|
| `API_PORT` | `15000-19999` | Port for the Go API server |
| `UI_PORT` | `20000-24999` | Port for the production UI server (`ui/server.js`) |

If the scenario adds WebSocket channels on the existing API or UI server, do
not add another `ports` entry. Declare an additional port only when the
scenario starts a separate listener process.

The canonical bands all sit below 32768 so Linux never hands out the
ports as outbound source ports. See the project-level
[port-allocation reference](../../../../docs/reference/port-allocation.md)
for the full policy.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/audio-tools.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `audio-tools` the prefix is the scenario id upper-cased with
hyphens replaced by underscores (so `my-scenario` → `MY_SCENARIO`).
The following are recognised, in precedence order (first-found wins);
substitute your scenario's prefix for `<PREFIX>`:

| Purpose | Variables |
|---|---|
| API base URL | `<PREFIX>_API_BASE`, `<PREFIX>_API_URL`, `VROOLI_API_BASE` |
| API port | `<PREFIX>_API_PORT` |
| API token | `<PREFIX>_API_TOKEN`, `VROOLI_API_TOKEN` |
| Config dir | `<PREFIX>_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |
| HTTP timeout | `<PREFIX>_HTTP_TIMEOUT`, `VROOLI_HTTP_TIMEOUT` |

> **Do not** set the un-prefixed `API_PORT` for a CLI invocation —
> when CLIs run inside web-console terminals it leaks across scenarios.
> Use the scenario-prefixed form or the `--api-base` flag.

## Service manifest (`.vrooli/service.json`)

Single source of truth for everything the lifecycle needs to know.

| Section | Owns |
|---|---|
| `service` | name, display name, description, version, category, maintainers, repository URL |
| `ports` | port-name → env-var + range mapping (lifecycle allocates from these) |
| `cli` | command name, install scripts (per OS), invoke shape, freshness inputs |
| `lifecycle.health` | `/health` endpoint, startup grace period, periodic checks |
| `lifecycle.setup` | build steps + idempotency conditions (binary present, UI bundle fresh) |
| `lifecycle.develop` | how to start the running scenario |
| `lifecycle.test` | which test command to invoke |
| `lifecycle.stop` | how to shut down cleanly |
| `environment` | static env vars set for every lifecycle step |
| `dependencies.resources` | shared local resources (postgres, redis, qdrant, …) |

The template ships with `dependencies.resources: {}` — SQLite is
in-process, so no resource is required. Scenarios add resources here
when they need shared infrastructure.

## Schema bootstrap

Schema is owned per-domain. `api/internal/<dom>/schema.sql` declares
each domain's tables and is embedded into the binary via `go:embed`
from `api/internal/<dom>/schema.go::Schema()`. Cross-cutting
infrastructure (postgres extensions, custom types, cross-domain views)
lives in `api/internal/database/system.sql` — empty by default in
SQLite scenarios.

The shared registry at `api/internal/modules/registry.go::AllSchemas()`
collects them in order (system first, then domains alphabetical), and
`apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)` from
`api-core/database` applies them at startup. The path is idempotent —
all DDL uses `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE … ADD COLUMN
IF NOT EXISTS`, so re-runs on every boot are no-ops.

Adding a column lands in the same diff as the Go struct field, the
repository scan, and the proto wire shape — single location, single
edit. Drops/renames in production data need the brownfield
versioned-migration helpers (`Migrate` / `MigrationProvider` in
`api-core/database`, deferred until the first scenario hits the pain).

See [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#domain-owned-schema)
for the design rationale and [`../internal/SEAMS.md`](../internal/SEAMS.md)
for the per-seam table including `notes.Schema` and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/audio-tools/config.json`
3. `~/.vrooli/config/audio-tools/config.json`
4. `~/.config/vrooli/audio-tools/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
audio-tools configure api_base http://localhost:15001/api/v1
audio-tools configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port audio-tools API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
audio-tools`").

## Streaming STT control surface

The streaming STT pipeline ([`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md))
exposes five operator-tunable levers. They are read once per
streaming session by `StrategySelector` and apply to both transports
(browser WS, Connect bidi). The defaults match the pre-extraction
web-console behavior — operators only adjust them to trade latency,
CPU, or quality.

| Lever | Type | Default | Range | Audience | Trade-off |
|---|---|---|---|---|---|
| `stt.streaming_mode` | enum: `auto`, `off` | `auto` | — | operator | `off` forces a single batch `Transcribe` at `StreamEnd` — cheapest, no partials. `auto` selects the best (strategy, provider) pair for the negotiated tier. |
| `stt.strategy_preference` | enum: `auto`, `vad`, `overlap` | `auto` | — | operator | `vad` = silence-bounded segments; lower CPU, segment-only events. `overlap` = sliding-window LocalAgreement; higher CPU on Local Whisper, live partials, lower end-of-utterance latency. Ignored for native-streaming providers (Deepgram, Azure, Google, future LPBS). `auto` picks per-provider defaults. |
| `stt.vad_silence_ms` | integer | `700` | `200–3000` | operator | Silence window that closes a VAD segment. Lower = snappier but may chop natural pauses; higher = preserves long sentences, increases end-of-segment latency. Only meaningful when `VADSegmentStrategy` is active. |
| `stt.overlap_window_ms` | integer | `2000` | `1000–5000` | operator | Sliding window size for `OverlapAgreeStrategy`. Bigger = better agreement, more CPU per partial. Only meaningful for Local Whisper + `overlap`. |
| `stt.overlap_commit_runs` | integer | `2` | `2–4` | operator | How many consecutive sliding-window runs must agree on a prefix before it commits from `Partial` → `Segment`. Higher = more stable text, longer commit latency. |

Lever rules (per [control-surface-tunable-levers-design]):

- All five have safe defaults that produce working behavior.
- All five are monotonic where applicable — increasing a value moves
  the system in one consistent direction (more CPU, more latency, more
  stable text, …).
- The selector refuses incompatible combinations via the matrix in
  [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md#strategy--provider-compatibility);
  forbidden pairs raise a typed error at session start rather than
  degrading silently.
- Setting these levers does NOT pick a provider — provider tier
  precedence remains the fixed BYOK → Vrooli → Local order defined in
  the PRD.

### Audio format: per-session declaration, not a lever

The inbound **codec is per-session client-declared, not an operator
lever**. Clients SHOULD declare it (WS `?format=` query param / Connect
`StreamStart.input_format`); when omitted, the `internal/audioformat`
substrate sniffs the first chunk's magic bytes. Declaring `pcm_s16le`
takes an ffmpeg-free fast-path. The canonical internal STT representation
(16-bit LE PCM, mono, 16 kHz) is **fixed, not tunable** — VAD RMS and
Whisper both depend on it. ffmpeg low-latency decode flags
(`-flush_packets 1`, decode-only, `-loglevel error`) are fixed-internal,
not levers, for predictable latency and a minimal untrusted-input surface.

### Whisper concurrency

Local STT calls to the Whisper resource are bounded by a semaphore
(`DefaultWhisperConcurrency = 5`, matching the resource cap in
`resources/whisper/docs/API.md`). Over-limit callers **block (queue with
backpressure), never error**; a cancelled session releases its slot. This
is the true multi-session ceiling, upstream of the format layer — raising
it means scaling the Whisper resource, not just the bound.

## Summarize model policy

The local summarize provider uses Ollama. Model selection is centralized in
`api/internal/summarize/model_policy.go` and surfaced through
`SummarizeService.ListSummarizeModels`.

Current policy:

- Default fallback: `llama3.2:3b`.
- Recommended non-reasoning candidates: `gemma3:4b`, `gemma3n:e2b`,
  `llama3.2:3b`, `llama3.2:1b`, `qwen2.5:3b`, `phi4-mini:3.8b`.
- Reasoning models such as `qwen3:*` and `deepseek-r1:*` are marked
  non-default because they are slower and often spend output budget on
  internal reasoning.
- `/api/tags` is the installed-model source. Missing recommended models are
  returned with `ollama pull <model>` commands; audio-tools does not pull
  models automatically.
- Startup/env stale-config protection coerces empty or known unsafe reasoning
  stored defaults back to `llama3.2:3b`. Explicit runtime updates are accepted
  so operators can benchmark non-default models deliberately.

## STT streaming pipeline levers

The local Whisper provider hands every VAD-bounded audio segment to Whisper as
its own batch. Two knobs trade boundary-word accuracy against perceived latency:

| Knob | Default | Bounds | Effect |
|---|---|---|---|
| `segmentSilenceMs` | `2500` | `800`–`3000` | Trailing silence before a segment closes. Longer = fewer cuts inside words. |
| `overlapBytes` | `8192` | `0`–`16384` | PCM16 carry-over (16 kHz mono) prefixed to the next segment. Higher = more cross-cut context, slightly more Whisper work. |
| `flushIntervalMs` | `500` | `100`–`5000` | Min interval between partial transcripts. |
| `minDeltaBytes` | `4096` | `512`–`32768` | Min audio delta before a partial transcript fires. |

### Profiles

Curated presets for `segmentSilenceMs` + `overlapBytes`. Selecting a profile
overlays only those two fields — raw knob edits afterwards stay in force.

| Profile | `segmentSilenceMs` | `overlapBytes` | When to pick |
|---|---|---|---|
| `latency` | 1200 | 2048 | Short utterances, demo flow, low-noise mic. |
| `balanced` | 2500 | 8192 | **Default.** Matches `DefaultConfig()`. |
| `accuracy` | 3000 | 16384 | Long dictation, noisy room, names/jargon-heavy speech. |

Higher values monotonically trade more end-of-utterance latency for fewer
mis-transcribed boundary words. Preset definitions live in
`api/internal/stt/pipeline/profile.go`; apply via `pipeline.ApplyProfile`.

The architecturally correct fix for boundary accuracy is
`OverlapAgreeStrategy` (built in `internal/stt/strategy/overlap_agree.go` but
not wired into the WS transport yet — tracked separately).

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
