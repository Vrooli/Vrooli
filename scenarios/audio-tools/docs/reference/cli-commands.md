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
| `audio-tools eval run [--strategies batch,vad_segment,overlap_agree] [--clip-ids …] [--realtime-repeats N] [--chunk-ms N] [--overlap-max-window-ms N] [--output FILE] [--format table\|json]` | Replay the corpus through the strategies and print the WER/compute/latency comparison table (see [`eval-harness.md`](eval-harness.md)) | `EvalService.RunEval` |
| `audio-tools experiment start [--name LABEL] [--strategies …] [--clip-ids …] [--realtime-repeats N] [--chunk-ms N] [--dropped-span-threshold N] [--overlap-max-window-ms N] [--seed N] [--long-form true] [--target-duration-seconds N] [--gap-ms N] [--tag-contains SUBSTR] [--noise-types white,fan] [--snr-db 18,12,6] [--competing-voices af_bella] [--competing-text TEXT] [--target-profile-id ID] [--speaker-extraction true] [--speaker-verification true] [--speaker-mode filter] [--speaker-threshold 0.5] [--speaker-fallback false] [--speaker-ablation true] [--estimated-seconds N]` | Enqueue a persisted async STT experiment and return its id immediately; long-form mode concatenates selected clips with deterministic silence gaps, augmentation mixes generated noise or Kokoro voices at the SNR grid, speaker flags bind per-run extraction/verification without mutating live speaker config, and the report stores safety gates/length curves (`--dropped-span-threshold`, default 4) with condition notes | `ExperimentService.StartExperiment` |
| `audio-tools experiment get ID` | Show one experiment's persisted lifecycle state and run cells | `ExperimentService.GetExperiment` |
| `audio-tools experiment wait ID` | Block once until the experiment reaches a terminal state; caller cancellation does not abort the run | `ExperimentService.WaitExperiment` |
| `audio-tools experiment list [--status queued\|running\|succeeded\|failed\|canceled] [--limit N] [--offset N]` | List persisted experiments newest first | `ExperimentService.ListExperiments` |
| `audio-tools experiment cancel ID` | Cancel a queued or running experiment | `ExperimentService.CancelExperiment` |
| `audio-tools experiment watch ID` | Stream progress events for an active experiment | `ExperimentService.StreamExperimentEvents` |
| `audio-tools experiment report ID` | Fetch the stored experiment report artifact and run-cell metrics | `ExperimentService.GetExperimentReport` |
| `audio-tools experiment compare ID1,ID2[,ID3…]` | Compare two or more experiments; pass `--json` for full recipes and reports | `ExperimentService.CompareExperiments` |
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
audio-tools configure token <token>
```

Read values back without an argument:

```bash
audio-tools configure api_base
```

## Scenario commands — `notes` (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layout
when adding the first non-trivial domain to your scenario.

### `audio-tools notes list`

List notes, newest-first. Calls the generated Connect-RPC
`Notes/List` method. Uses the
**data-retrieval contract**: `Summary → Results → Retrieval Hints`.

```bash
audio-tools notes list
audio-tools notes list --json
```

### `audio-tools notes create --title <title> [--body <body>]`

Create a note. Calls the generated Connect-RPC `Notes/Create` method. Uses the **mutation
contract**: `Result → What Changed → Next Command`.

```bash
audio-tools notes create --title "First note" --body "Hello world"
```

`--title` is required. `--body` is optional. Validation lives in the
API service, so an empty title surfaces as an `invalid_argument`
Connect error rather than a CLI-side check.

### `audio-tools notes get <id>`

Fetch a note by id. Calls the generated Connect-RPC `Notes/Get` method.

```bash
audio-tools notes get abc123
```

A non-existent id surfaces as `not_found`; the CLI translates the
typed Connect code to an actionable error message.

### `audio-tools notes attach <id> --file <path>`

Attach a file to a note. This is the documented REST multipart
exception because the request body contains opaque bytes. The response
is proto-typed attachment metadata.

```bash
audio-tools notes attach abc123 --file ./example.png
```

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

For a new domain, copy the notes command group first, then replace it
once your real domain is green.

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

- **Subcommand groups** (`notes list`, `notes create`) over flat
  verbs (`list-notes`, `create-note`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `notes get <id>`
  not `notes get --id <id>`.
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
