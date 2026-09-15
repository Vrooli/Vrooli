# CLI Commands — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Command map

| Command | Description | API method |
|---|---|---|
| `audio-tools status` | Health probe (provided by cli-core's StandardScenarioApp) | `GET /health` |
| `audio-tools stt transcribe --file PATH [--language EN] [--format FMT]` | Transcribe an audio file | `STTService.Transcribe` |
| `audio-tools stt transcribe-stream --file PATH [--language EN] [--chunk-bytes N]` | Stream-transcribe a file (one event per line) | `STTService.TranscribeStream` |
| `audio-tools stt stream-config` | Show the resolved streaming STT levers | `STTService.GetStreamConfig` |
| `audio-tools stt stream-config-set [--streaming-mode auto\|off] [--strategy-preference …] [--vad-silence-ms N] [--overlap-window-ms N] [--overlap-commit-runs N] [--overlap-max-stall-rejects N]` | Mutate streaming STT levers (incl. the overlap stall-fallback; `0` disables it) | `STTService.UpdateStreamConfig` |
| `audio-tools corpus list [--tag SUBSTR] [--limit N]` | List speech-eval corpus clips (newest first) | `CorpusService.ListClips` |
| `audio-tools corpus import AUDIO_FILE --reference TEXT [--reference-file PATH] [--tags a,b] [--format pcm_s16le] [--sample-rate 16000] [--scripted]` | Import a PCM clip + ground-truth transcript into the corpus | `CorpusService.CreateClip` |
| `audio-tools corpus get ID` | Show one clip's metadata | `CorpusService.GetClip` |
| `audio-tools corpus delete ID` | Delete a clip (row + audio blob) | `CorpusService.DeleteClip` |
| `audio-tools experiment start [--name LABEL] [--strategies …] [--clip-ids …] [--realtime-repeats N] [--latency-tail-seconds N] [--chunk-ms N] [--dropped-span-threshold N] [--overlap-max-window-ms N] [--seed N] [--long-form true] [--target-duration-seconds N] [--sweep-durations 30,60,120] [--gap-ms N] [--tag-contains SUBSTR] [--noise-types white,fan] [--snr-db 18,12,6] [--competing-voices af_bella] [--competing-text TEXT] [--target-profile-id ID] [--speaker-extraction true] [--speaker-verification true] [--speaker-mode filter] [--speaker-threshold 0.5] [--speaker-fallback false] [--speaker-ablation true] [--estimated-seconds N]` | Enqueue a persisted async STT experiment and return its id immediately; long-form mode concatenates selected clips with deterministic silence gaps, `--sweep-durations` materializes one input per requested duration for backend-owned scaling analysis, augmentation mixes generated noise or Kokoro voices at the SNR grid, speaker flags bind per-run extraction/verification without mutating live speaker config, and the report stores safety gates/length curves/scaling warnings (`--dropped-span-threshold`, default 4) with condition notes | `ExperimentService.StartExperiment` |
| `audio-tools experiment get ID` | Show one experiment's persisted lifecycle state and run cells | `ExperimentService.GetExperiment` |
| `audio-tools experiment wait ID` | Block once until the experiment reaches a terminal state; caller cancellation does not abort the run | `ExperimentService.WaitExperiment` |
| `audio-tools experiment list [--status queued\|running\|succeeded\|failed\|canceled] [--limit N] [--offset N]` | List persisted experiments newest first | `ExperimentService.ListExperiments` |
| `audio-tools experiment cancel ID` | Cancel a queued or running experiment | `ExperimentService.CancelExperiment` |
| `audio-tools experiment watch ID` | Stream progress events for an active experiment | `ExperimentService.StreamExperimentEvents` |
| `audio-tools experiment report ID` | Fetch the stored experiment report artifact and run-cell condition rows, including compact backend-owned scaling classifications when a duration sweep produced enough points | `ExperimentService.GetExperimentReport` |
| `audio-tools experiment compare ID1,ID2[,ID3…]` | Compare two or more experiments; human output includes compact scaling verdicts when present, and `--json` carries the full recipes/reports/scaling points | `ExperimentService.CompareExperiments` |
| `audio-tools tts synthesize --text TEXT [--voice ID] [--speed N] [--format FMT] [--out PATH]` | Synthesize speech audio | `TTSService.Synthesize` |
| `audio-tools tts synthesize-stream --text TEXT [--voice ID] [--speed N] [--format FMT] [--out PATH]` | Stream-synthesize speech (writes frames to --out as they arrive) | `TTSService.SynthesizeStream` |
| `audio-tools tts voices` | List canonical voices | `TTSService.ListVoices` |
| `audio-tools summarize text --text TEXT [--level light\|moderate\|heavy]` | Summarize text | `SummarizeService.Summarize` |
| `audio-tools audio transcode --input PATH --output PATH` | Transcode to WAV | `AudioProcessingService.Transcode` |
| `audio-tools settings provider` | Show the current provider-routing config | `SettingsService.GetProviderConfig` |
| `audio-tools settings providers` | Provider routing + TTS-tier availability matrix (folded from former `diagnose providers` on 2026-05-17) | composite: `SettingsService.GetProviderConfig` + `TTSService.GetStatus` |
| `audio-tools settings byok-list` | List stored BYOK credentials (redacted) | `SettingsService.ListBYOKCredentials` |
| `audio-tools settings byok-upsert --provider ID --capability stt\|tts\|summarize --key SECRET` | Add or replace a BYOK credential | `SettingsService.UpsertBYOKCredential` |
| `audio-tools settings byok-delete --provider ID --capability stt\|tts\|summarize` | Delete a BYOK credential | `SettingsService.DeleteBYOKCredential` |

Output defaults to human-readable rendering; pass `--json` for machine output (Connect-RPC wire shape).

`audio-tools stt transcribe-stream` uses Connect gRPC over HTTP/2. Against
the local scenario API it uses h2c, matching the production handler wrapper,
so the command can stream against the default cleartext development URL.

The former synchronous eval command was retired on 2026-07-01. Use
`audio-tools experiment start` plus `experiment wait`, `experiment report`,
and `experiment compare` for all STT evaluation runs.

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/audio-tools`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Domain layout convention

Every CLI domain directory under `cli/domains/<name>/` has exactly two files:

- `handlers.go` — `type handlers struct { core *cliapp.ScenarioApp; client <ServiceClient> }` plus a `newHandlers(core)` constructor that lazily builds the Connect client(s) once, plus one method per subcommand (`func (h *handlers) <subcommand>(ctx cliapp.RunContext) error`).
- `register.go` — `package <name>`, a `Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup` function that calls `newHandlers(core)` once, then declares the `SubcommandGroup` with `Name`, `Description`, `NeedsAPI`, and `Subcommands` (each `RunCtx: h.<subcommand>`).

This shape is the single source of truth — do not inline `RunCtx` closures inside `register.go`, do not rebuild the Connect client per command, and do not split a domain across more than two files. Any new domain (or any cleanup pass on an existing one) must match this layout.

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start audio-tools` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `audio-tools status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
audio-tools status
audio-tools status --json
```

### `audio-tools configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
audio-tools configure api_base http://localhost:15001/api/v1
audio-tools configure token "<token>"
```

Read values back without an argument:

```bash
audio-tools configure api_base
```

## Diagnostics — two-layer STT signal

`audio-tools diagnostics run` exercises STT/TTS/Summarize/Transcode against
bundled fixtures. The STT step reports **two** layers:

- **Readiness** — provider reachability (`diagnostic_scope=asr_readiness`).
- **Quality smoke** — no-speech safety + clean-speech WER against the shared
  egress policy. The human output shows `quality=<pass|warn|fail>` with a
  per-fixture breakdown; when a quality leak flips the step, the row also
  shows `readiness=pass` so reachability stays distinct from the fault.

```bash
# Run all capabilities; STT tile shows the quality breakdown.
audio-tools diagnostics run

# Machine-readable: quality fields live under steps[stt].details
#   quality_assessed / quality_status / quality_hallucination_detected /
#   quality_fixtures (JSON array of per-fixture verdicts).
audio-tools diagnostics run --capability stt --json
```

A `quality_smoke_failed` STT step means a no-speech fixture leaked a
surviving transcript — a hallucination-filter regression. Clean-speech WER
drift warns but keeps the step green. For corpus-grade quality, use Dictation
Studio experiments (see `docs/reference/eval-harness.md`).

## Output contracts

Every scenario command should render through one of three human
contracts. Proto-backed commands should use `cliapp.RenderProtoList`
or `cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `health`, `audit`, `validate`, `doctor` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a
non-proto report, use the `RunContext` render helpers directly
(`ctx.RenderList`, `ctx.RenderMutation`, or the operational report
helpers).

## Adding a new command

For a new command, add a manifest-bound RPC command or a documented
manifest exception when no single RPC can own the behavior.

For a command inside an existing domain:

1. If the command needs a new API endpoint, add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint).
2. Add the command to `cli/domains/<domain>/register.go`.
3. Implement its handler in `cli/domains/<domain>/handlers.go` or a
   focused sibling file.
4. The handler should:
   - Declare flags and positionals in `cliapp.ArgSchema`; cli-core
     uses the schema for parsing and help output.
   - Implement `RunCtx func(ctx cliapp.RunContext) error`, then read
     values with `ctx.Flag(...)`, `ctx.Positional(...)`, and
     `ctx.JSON()`.
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions.
   - Mark the command with `NeedsAPI: true` so stale-checking,
     token validation, and `--auto-start` preflight all stay
     connected automatically
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
5. Add endpoint metadata in the API handler module and add a matching
   row to `api/cmd/gen-endpoints/cli_commands_seed.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
6. Add a row to this document.
7. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)).

## Command structure principles

- **Subcommand groups** (`stt transcribe`, `experiment start`) over flat
  verbs. Discoverability via `--help` is the goal.
- **Positionals for required identifiers, flags for optional inputs.**
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start audio-tools`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
