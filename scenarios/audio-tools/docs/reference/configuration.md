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
exposes the operator-tunable levers below (strategy levers plus
the egress-gate levers in the next subsection). They are read once per
streaming session by `StrategySelector` and apply to both transports
(browser WS, Connect bidi). The defaults match the pre-extraction
web-console behavior — operators only adjust them to trade latency,
CPU, or quality.

| Lever | Type | Default | Range | Audience | Trade-off |
|---|---|---|---|---|---|
| `stt.engine_id` | string (manifest id) | `whisper-local` | from `ListEngines` | operator | Active STT engine. `whisper-local` = faster-whisper (batch; VAD/overlap/buffered). `kyutai` = native-streaming (Passthrough only, ~0.5s delay, GPU). Only the Local tier honors it; BYOK/Vrooli stream natively. Read valid ids from `ListEngines`/`audio-tools stt engines` — never hardcode. Switching consults `GetEngineSwitchImpact` (shared-resource awareness). |
| `stt.streaming_mode` | enum: `auto`, `off` | `auto` | — | operator | `off` forces a single batch `Transcribe` at `StreamEnd` — cheapest, no partials. `auto` selects the best (strategy, provider) pair for the negotiated tier. |
| `stt.strategy_preference` | enum: `auto`, `vad`, `overlap` | `auto` | — | operator | `vad` = silence-bounded segments; one Segment per utterance, lower CPU. `overlap` = growing-buffer LocalAgreement-N with VAD-anchored triggering, word-aligned cursor advance, and bounded agreement window; incremental Segments mid-utterance. Ignored for native-streaming providers (Deepgram, Azure, Google, future LPBS). `auto` picks per-provider defaults: **Local Whisper → `vad`** (the seamless one-segment-per-utterance default), native-stream providers → `passthrough`. Pick `overlap` explicitly if you want mid-utterance incremental commits; it is no longer the auto default while its low-latency UX is being honed (2026-05-29). See PROBLEMS.md "OverlapAgree commit gap" for the algorithm history. |
| `stt.vad_silence_ms` | integer | `700` | `200–3000` | operator | Silence window that closes a VAD segment. Lower = snappier but may chop natural pauses; higher = preserves long sentences, increases end-of-segment latency. Only meaningful when `VADSegmentStrategy` is active. |
| `stt.overlap_window_ms` | integer | `2000` | `1000–5000` | operator | Sliding window size for `OverlapAgreeStrategy`. Bigger = better agreement, more CPU per partial. Only meaningful for Local Whisper + `overlap`. |
| `stt.overlap_commit_runs` | integer | `2` | `2–4` | operator | How many consecutive sliding-window runs must agree on a prefix before it commits from `Partial` → `Segment`. Higher = more stable text, longer commit latency. |
| `experiment.overlap_max_window_ms` | integer | `25000` | per-run experiment override | agent/operator | Hermetic eval/experiment-only ceiling for overlap-agree's uncommitted tail before force-commit. Lower = bounds worst-case tail latency and repeated Whisper work earlier; too low can force-commit less-agreed text. It is snapshotted in the experiment recipe and does not mutate live `stt_stream_config`. |
| `experiment.augmentation.snr_db` | repeated number | `12` when augmentation is enabled without an explicit grid | per-run experiment override | agent/operator | SNR grid for generated-noise and competing-voice overlays. Lower values make the interferer louder relative to the target. Stored in the experiment recipe; no live STT or speaker config row is mutated. |
| `experiment.speaker.target_profile_id` | string | empty | per-run experiment override | agent/operator | Enrolled speaker profile used as the target identity for experiment-only extraction and verification. Required when a speaker condition enables either stage. |
| `experiment.speaker.extraction_enabled` | bool | `false` | per-run experiment override | agent/operator | Binds target-speaker extraction into the eval `Segmenter` for this experiment only. Does not mutate `speaker.extraction_enabled`. |
| `experiment.speaker.verification_enabled` | bool | `false` | per-run experiment override | agent/operator | Binds egress speaker verification into the eval `Segmenter` for this experiment only. Does not mutate `speaker.enabled` or `speaker.mode`. |
| `experiment.speaker.ablation_enabled` | bool | `false` | per-run experiment override | agent/operator | Evaluates extraction/verification off/on combinations over the same realized input so reports can show recovery/dropped-word deltas. Phase 6 exposes these as condition-suffixed strategy rows; Phase 7 adds first-class attribution. |
| `stt.overlap_max_stall_rejects` | integer | `3` | `0–10` (`0`=disabled) | operator | OverlapAgree **stall-fallback**. On hard audio / jittery word timestamps, LocalAgreement can keep producing hypotheses that diverge from the committed prefix; the divergence-reject path then commits nothing and the uncommitted tail grows toward the 25s `max_window_ms` net, re-transcribing an ever-larger window each settle and saturating the 5-wide Whisper semaphore — so streaming *finalizes slower than batch*. After this many **consecutive** divergence-rejects the strategy force-commits the freshest hypothesis tail and advances its cursor, bounding tail growth well before the 25s net. Lower = snappier recovery on stalls but more risk of committing un-agreed (possibly wrong) text; higher = waits longer for natural agreement. `0` disables the fallback (only the 25s net applies — the pre-2026-06 behavior). It never silently drops audio — it commits a best guess. Only meaningful for Local Whisper + `overlap`. |

Overlap-agree compares candidate prefixes by lowercasing each whitespace token
and stripping Unicode punctuation/symbols for comparison only. The emitted
segment text remains the first agreeing hypothesis verbatim, so punctuation
normalization cannot rewrite committed user-visible text. Token-boundary
changes remain strict: `D.C.` and `DC` agree, but `well-known` and
`well known` do not become the same token sequence.

Lever rules (per [control-surface-tunable-levers-design]):

- All levers have safe defaults that produce working behavior.
- Levers are monotonic where applicable — increasing a value moves
  the system in one consistent direction (more CPU, more latency, more
  stable text, …).
- The selector refuses incompatible combinations via the matrix in
  [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md#strategy--provider-compatibility);
  forbidden pairs raise a typed error at session start rather than
  degrading silently.
- Setting these levers does NOT pick a provider — provider tier
  precedence remains the fixed BYOK → Vrooli → Local order defined in
  the PRD.

### Egress gate (post-recognition quality)

Every user-facing STT transcript passes through the shared
**post-recognition egress policy** (`internal/stt/quality/`) before it
leaves audio-tools. Streaming still runs the underlying `egress.Gate`
per segment; unary Connect `Transcribe`, multipart
`/api/v1/voice/transcribe`, buffered fallback output, and diagnostics
readiness previews apply the same policy to the final chain result. A
stage may `Drop` (suppress entirely, excluded from final text), `Reject`
(suppress text, emit a speaker-rejection event on streaming paths), or
`Emit`.

| Lever | Type | Default | Range | Audience | Trade-off |
|---|---|---|---|---|---|
| `stt.hallucination_filter_enabled` | bool | `true` | — | operator | Text-domain stage: drops segments whose text matches Whisper's known silence-hallucination phrases ("thank you for watching", "please subscribe", …). Off = raw text passes through (debugging). |
| `stt.vad_filter_enabled` | bool | `true` | — | operator | Enables faster-whisper's built-in voice-activity filter on each `/asr` request (`vad_filter=true`). Strips silence **before** decode — the source-level hallucination fix. Off = Whisper decodes silence and may narrate it. |
| `stt.no_speech_threshold` | double | `0.6` | `(0, 1]` | operator | Signal-domain stage: a segment drops when its mean `no_speech_prob` **exceeds** this **and** `avg_logprob` is below `logprob_threshold`. Higher = more permissive (fewer drops). |
| `stt.logprob_threshold` | double | `-1.0` | `[-10, 0)` | operator | Paired with `no_speech_threshold`. A segment must clear **both** conditions to be dropped, so a confidently-decoded segment containing a pause is kept. Lower (more negative) = more permissive. |

Stage applicability is capability-derived, not engine-name-branched:
the signal-domain stage only fires for engines whose manifest declares
`provides.confidenceSignals` (Whisper does; a native-streaming engine
that reports none has the stage skipped gracefully). The engine
manifest is the source of truth for which stages a given engine runs;
these levers remain the operator tunables those stages read.

Unary `TranscribeResponse` includes `filtered`, `filter_reason`, and
`policy_details`. When `filtered=true`, `text` is intentionally empty
because the configured egress policy suppressed a known silence
hallucination, low-confidence silence/noise, or equivalent non-user-facing
output. Diagnostics reports the same decision as readiness metadata:
`diagnostic_scope=asr_readiness` still means the provider accepted and
processed audio; `transcript_filtered=true` means the smoke transcript
was not displayed because the quality policy filtered it.

### Speaker isolation ("only my voice") — audio-domain stage

The audio-domain egress stage rejects voices that are not the enrolled
speaker (e.g. background music, a second person). It is configured via a
**separate** admin surface (`SpeakerConfig`, `Get/UpdateSpeakerConfig` +
`audio-tools` speaker commands), not the `StreamConfig` levers above,
because it carries enrolled-profile bindings. It applies only to segments
that carry audio (the Whisper PCM path); Passthrough engines bypass it.
Enrollment + verification are backed by the `speaker-verification`
resource (SpeechBrain ECAPA-TDNN), which owns the embeddings. A profile is one
identity holding **N labeled enrollment clips** (different devices / speaking
styles); the resource trims each clip to its voiced span before embedding (Silero
VAD by default, automatic fallback to energy VAD when the model load fails) and
verifies via **max-cosine across all clips in the profile** (the v0.4 scoring
model — centroid aggregation was dropped because spectrally divergent enrollment
clips pulled the mean toward neutral and depressed genuine scores). The egress decision
is **session-stateful**: it accumulates per-segment scores (EMA) and never
rejects until `min_decision_seconds` of voiced audio has accrued (warm-up), so a
short first utterance is not falsely dropped.

| Lever | Type | Default | Audience | Trade-off |
|---|---|---|---|---|
| `speaker.enabled` | bool | `false` | operator | Master switch. Off omits the audio-domain stage entirely (zero overhead). |
| `speaker.mode` | enum: `off`, `filter`, `advisory` | `filter` (when enabled) | operator | `filter` rejects non-matching segments; `advisory` scores only (never blocks); `off` skips the stage. |
| `speaker.threshold` | double | `0.5` | operator | Match threshold on the smoothed session score. Higher = stricter (more false rejections); lower = more permissive (other voices leak through). **Calibrate against real voices — see below.** |
| `speaker.profile_ids` | string[] | `[]` | operator | Enrolled profiles to match against. At least one required when enabled. |
| `speaker.reject_behavior` | enum: `drop`, `show-muted` | `drop` | operator | What the consumer does with a rejected segment. |
| `speaker.fallback_without_verification` | bool | `false` | operator | When the resource is down or no profile is bound: `true` lets segments through (flagged unverified on the rejection event), `false` rejects. |
| `speaker.min_decision_seconds` | double | `3.0` | operator | Warm-up window: voiced seconds that must accrue before the session verifier may reject. `0` uses the default. |
| `speaker.score_smoothing` | double | `0.4` | operator | EMA alpha `(0,1]` on per-segment scores so the session decision stops swinging mid-utterance. `0` uses the default. |

**Canonical default threshold = 0.5** across all layers (engine manifest,
`DefaultSpeakerConfig`, the persisted-config default, and the resource
`/v1/verify` form). This is a *starting* value: the right cutoff depends on the
ECAPA embedding distribution for your enrolled voices and room. **Calibrate it
live** — enroll 3–5 real clips (normal + whisper + a second device), then drive a
real-mic session and a second person; confirm the genuine session score lands in
a confident band above the cutoff and the impostor clearly below, and adjust
`speaker.threshold`. Record the chosen number here when set.

Resource-side embedding knobs (env vars on the `speaker-verification` resource,
not per-session levers): `SPEAKER_VAD` (`silero`|`energy`|`none`, default
`silero` — falls back to `energy` automatically when the Silero weights are not
loadable), `SPEAKER_MIN_ENROLL_VOICED_SECONDS` (`3.0`),
`SPEAKER_MIN_VERIFY_VOICED_SECONDS` (`1.0`), `SPEAKER_SELF_CONSISTENCY_THRESHOLD`
(`0.5`, see below), and `SPEAKER_EMBED_DENOISE` (default off — spectral denoise
before embedding; off because it can distort timbre and hurt ECAPA, and VAD trim
already removes the dominant silence diluter). `SPEAKER_SCORE_AGG` is no longer
read — scoring is unconditionally max-over-clips.

**Self-consistency at enrollment.** Every enrolled clip is scored against the
strongest existing clip in the profile (max cosine). When that score is below
`SPEAKER_SELF_CONSISTENCY_THRESHOLD`, the enroll response sets
`self_consistency_warning: true` and the resource surfaces
`self_consistency_score` and the matched existing clip. The new clip is **stored
either way** — the warning is informational. The intent: tell the operator a
clip recorded in substantially different conditions (a different mic, heavy
background noise, a different room) may not help recognition; re-record in
conditions that resemble verification time. The first clip in a fresh profile
has no self-consistency to check; its score is reported as `-1.0` and the
warning is `false`. The verify response surfaces `best_clip_id`,
`best_clip_label`, `best_clip_score`, `n_clips`, and `vad_model` so an operator
can see *which* enrollment matched and under *which* VAD.

The active egress isolation **method** (`verification`) is selected by the
engine manifest's `speakerIsolation.active` field — swapping it is a one-field
manifest edit, not an operator lever. (The egress gate filters *text*; isolating
the *audio* of the enrolled speaker is the separate ingress stage below.)

### Target-speaker extraction ("pull my voice out") — ingress stage

Where verification (above) DROPS a finished segment's text when it doesn't match
the enrolled speaker, **extraction removes the interfering voice from the audio
itself, BEFORE recognition** — so the recognizer only ever hears the enrolled
speaker. It is a pre-recognition ingress stage (source separation + ECAPA
target-selection in the `speaker-verification` resource), gated by config:

| Lever | Type | Default | Audience | Trade-off |
|---|---|---|---|---|
| `speaker.extraction_enabled` | bool | `false` | operator | Isolate the enrolled speaker's voice before recognition. Requires `speaker.enabled=true`, a bound profile, and the extraction-capable resource. Adds per-window latency (SepFormer); GPU recommended for interactive use. |

```bash
# Enable (after enrolling + binding a profile, as above):
audio-tools stt speaker-config --extraction-enabled true
```

Like denoise, extraction is gated on config + resource availability (no manifest
flag), and currently runs on the Whisper PCM path; if the resource is down or no
target is found in the mixture, it degrades to passing the original audio
through rather than dropping it. Extraction (isolates audio) and verification
(filters text) are complementary and may both be enabled.

**Default posture: OFF.** Verification ships disabled (`speaker.enabled=false`,
effectively `mode=off`) so the gate adds zero overhead until an operator opts
in. Turning it on is a single CLI call once a profile is enrolled:

```bash
# 1. Enroll your voice (Web Console → Voice Input, or the CLI), then confirm it:
audio-tools stt speaker-status            # shows profiles + their enrollment seconds

# 2. Bind the profile and switch on filter mode:
audio-tools stt speaker-config --enabled true --mode filter \
    --profile-ids <profile-id> --threshold 0.35

# 3. Validate live: speak yourself (text should pass), then have a second
#    person / play music over you (their text should be dropped). advisory
#    mode scores without blocking if you want to tune the threshold first:
audio-tools stt speaker-config --mode advisory   # observe scores, then flip to filter
```

> **Prerequisite — enrollment fidelity.** Reliable matching depends on the
> enrollment audio being preprocessed the same way the verify path is. The
> enroll handler normalizes uploaded audio to canonical PCM (16 kHz mono
> s16le) and WAV-wraps it before embedding — identical to the per-segment
> verify path — so the enrollment and verification embeddings are comparable.
> After enrolling, re-confirm the threshold (`0.35` default) against your own
> voice in `advisory` mode before committing to `filter`.

> **Scope — Whisper VAD path only.** The gate runs over each segment's
> canonical PCM, which only the VAD/overlap (Whisper) strategies produce.
> Native-streaming engines (Kyutai `passthrough`) emit no per-segment audio,
> so verification cannot run on them and segments pass through unverified.
> Audio-domain *target extraction* (isolating your voice before recognition)
> is the engine-agnostic answer and is tracked separately.

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

To inspect the live capability matrix, run `audio-tools stt formats`
(accepted ingress codecs + whether the local ffmpeg decode backend is
present + the canonical PCM target) and `audio-tools tts formats`
(producible egress containers). Both print human-readable output and are
backed by the `STTService.GetSupportedFormats` / `TTSService.GetSupportedFormats`
RPCs, which read the `internal/audioformat` substrate's vocabulary. When
ffmpeg is absent, batch STT still decodes containers via Whisper's own
decoder; only live non-PCM streaming degrades to buffered whole-file decode.

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

- Default fallback: a small general chat model (currently `llama3.2:3b`).
- The selector recommends **small, non-reasoning** local models (roughly ≤4B,
  instruct/chat-tuned) — they summarize quickly without spending the output
  budget on internal reasoning. The concrete picks track whatever such models
  are installed; see `resource-ollama policy roles` for the current
  small/general role targets.
- Reasoning-tuned models are marked non-default because they are slower and
  often spend the output budget on internal reasoning instead of the summary.
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
