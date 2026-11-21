# Audio Transformations

This document is the canonical architecture reference for the
ffmpeg-backed audio operations exposed by `AudioProcessingService`:
transcode, trim, fade, volume, normalize, split, merge, and
extract-metadata. It explains how every operation flows from the
Connect-RPC edge into a single `Runner` seam, where temp files appear,
and how failure modes map onto Connect codes.

Read this first when:

- adding a new ffmpeg-backed transformation,
- swapping the ffmpeg invocation strategy (e.g., adopting libav
  bindings, switching to a long-lived ffmpeg server),
- changing an `AudioProcessingService` method,
- debugging "why does my merge crash with a tempfile error?" or "why
  do I get `FailedPrecondition` when ffmpeg is installed?".

The audio domain is intentionally provider-free — there is no BYOK,
Vrooli, or Local tier here. Every operation is local CPU work bounded
by ffmpeg. Cross-tier abstractions belong in `ttschain`/`sttchain`/
`summarizechain`, not here.

## Purpose

`internal/audio` (`api/internal/audio/ops.go`,
`api/internal/audio/transcode.go`, `api/internal/audio/formats.go`)
owns the ffmpeg shell. It is the single home for:

- the `Runner` seam — every ffmpeg / ffprobe invocation routes through it,
- presence detection (`hasFfmpeg`, `hasFfprobe`),
- the `ErrFFmpegMissing`, `ErrFFmpegExec`, and `ErrInvalidArgument` sentinels,
- the format→muxer/demuxer translation (`aac` → `adts`, `opus` → `ogg`),
- temp-file lifecycle for operations that need seekable input
  (`runFfprobeJSON`, `writeTempInputs`).

Handlers (`api/handlers/audio/*.go`) are thin translation shims: copy
request bytes in, call one `intaudio` function, copy bytes out, and map
errors via `mapAudioErr` (`api/handlers/audio/errors.go`).

The same eight operations are reachable through the CLI (`audio-tools
audio …`, declared in `cli/manifest.json`) and the Diagnostics UI
(`AudioOpsPanel`, generated from `api/internal/operationcatalog`).
The transport is Connect-RPC; there is no multipart REST mount.

## Inputs

Every Connect method accepts raw audio bytes plus operation-specific
parameters. The shared guard `requireBytes`
(`api/handlers/audio/errors.go`) rejects empty payloads with
`CodeInvalidArgument` before any ffmpeg call.

| Operation | Handler | Inputs |
|---|---|---|
| `Transcode` | `api/handlers/audio/transcode.go` | `audio`, `input_format` (optional), `output_format`, `sample_rate`, `channels`, `bitrate` |
| `Trim` | `api/handlers/audio/trim.go` | `audio`, `format`, `start_seconds`, `end_seconds` |
| `Fade` | `api/handlers/audio/fade.go` | `audio`, `format`, `fade_in_seconds`, `fade_out_seconds`, `output_format` |
| `Volume` | `api/handlers/audio/volume.go` | `audio`, `format`, `gain_db`, `output_format` |
| `Normalize` | `api/handlers/audio/normalize.go` | `audio`, `format`, `method` (`peak`/`rms`/default EBU R128), `target_lufs`, `output_format` |
| `Split` | `api/handlers/audio/split.go` | `audio`, `format`, `chunk_seconds` OR `boundaries_seconds`, `output_format` |
| `Merge` | `api/handlers/audio/merge.go` | `sources[]` (each `{audio, format}`), `output_format`, `crossfade_seconds` |
| `ExtractMetadata` | `api/handlers/audio/metadata.go` | `audio`, `format` |

Time and size windows are validated before ffmpeg is invoked: finite,
non-negative, and bounded; `trim` requires `end > start` when an end is
set; `split` requires finite, strictly increasing, positive boundaries
and a bounded chunk count.

## Outputs

Every transformation returns transformed bytes plus a `content_type`
string derived from `contentTypeFor` (`api/internal/audio/ops.go`):

| Format | Muxer | `content_type` |
|---|---|---|
| `wav` (default) | `wav` | `audio/wav` |
| `mp3` | `mp3` | `audio/mpeg` |
| `flac` | `flac` | `audio/flac` |
| `aac` | `adts` (there is no `aac` muxer) | `audio/aac` |
| `ogg` / `opus` | `ogg` | `audio/ogg` |
| `webm` | `webm` | `audio/webm` |
| `pcm_s16le` | `s16le` | `application/octet-stream` |

`ExtractMetadata` returns a typed `AudioMetadata` message rather than
bytes: duration, sample rate, channels, bitrate, codec, format, and
container tags. `Split` returns one typed chunk per cut with its
`start_seconds` and `duration_seconds`.

## Internal Chain

Every operation follows a single shape: handler validates inputs,
calls a stateless `intaudio.Xxx(ctx, audio, ...)` function which
formats ffmpeg argv and pipes audio through stdin → stdout. The
browser, CLI, and other consumers all reach the same typed Connect
method.

```
Connect request (typed audio bytes + params)
        │
        ▼
requireBytes / typed validators
        │
        ▼
intaudio.Transcode / Trim / Fade / ...
        │
        ▼
runFfmpeg (ctx, audio, args...)
        │
        ▼
DefaultRunner.Run(ctx, "ffmpeg", stdin, baseArgs + args...)
        │
        ▼
exec.CommandContext("ffmpeg", "-nostdin", "-hide_banner", "-y",
                    "-loglevel", "error", ...)
        │
        ▼
stdout (transformed bytes)   |   stderr (joined into error message)
```

Two operations break the pure pipe-in/pipe-out shape:

1. **`Probe` and `ExtractMetadata`** — ffprobe needs seekable input,
   so the implementation writes to a temp file (`runFfprobeJSON`) and
   invokes ffprobe on the path. The temp file is removed on every exit
   path; `-select_streams a:0` keeps the result audio-only when a
   container also carries video.
2. **`Merge`** — concat / crossfade filter graphs need seekable
   per-source input, so every source is written to a temp file via
   `writeTempInputs`. Paths are recorded immediately after creation and
   cleaned up through `defer`, so a write or close failure cannot leak
   an earlier file.

Operation semantics that are easy to get wrong:

- **Trim** uses input `-ss` plus an output `-t (end-start)` *duration*.
  An output `-to` is measured on the post-seek timeline and would
  extend the clip instead of ending it at `end_seconds`.
- **Fade** anchors a fade-out at `duration - fadeOut` using the probed
  duration. When duration is unavailable (ffprobe missing or no
  duration metadata) it falls back to the dependency-free
  `areverse,afade=t=in,areverse` chain rather than misplacing the fade
  at `t=0`.
- **Split** with `chunk_seconds` probes duration once and issues N
  `Trim` calls sequentially. Explicit `boundaries_seconds` skip the
  probe; boundaries past EOF produce an empty final chunk rather than
  an out-of-range cut.
- **Normalize** currently reports the requested target in
  `measured_lufs`; a true measured value requires parsing
  `loudnorm`/`ebur128` output and is tracked in
  [PROBLEMS.md](../../internal/PROBLEMS.md).

`Transcode` has two entry points: the canonical
`Transcode(ctx, audio)` hard-codes 16 kHz mono WAV (the STT-pipeline
default), and `TranscodeWithInput` honors caller-supplied input
format, sample rate, channels, bitrate, and output format. The Connect
handler calls `TranscodeWithInput` so clients can specialise; a
declared input format forces the matching demuxer (for example raw
ADTS AAC) when ffmpeg cannot reliably probe it from a pipe.

### ffmpeg invocation contract

Every ffmpeg call prepends the same base flags
(`ffmpegBaseArgs`):

- `-nostdin` — never compete with `-i pipe:0` for the process's
  standard input; required for reliable pipe use on headless hosts and
  Windows.
- `-hide_banner` — keep CI/headless logs and the captured stderr tail
  readable.
- `-y` — never block on an overwrite prompt (defensive for future
  temp-file outputs).
- `-loglevel error` — keep stderr terse.

ffprobe rejects `-nostdin` (a hard error), so `runFfprobeJSON` passes
`-hide_banner` only; it reads a seekable file and leaves stdin nil.
Stderr is captured and joined into the returned error so callers see
the actual ffmpeg complaint, not just "exit 1". A canceled context is
surfaced as the context error, not as an ffmpeg rejection.

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| ffmpeg / ffprobe runner | `audio.Runner` | `execRunner` invoking `os/exec.CommandContext` | `DefaultRunner` is package-level mutable so tests can swap in a fake recording argv and returning canned stdout (`api/internal/audio/ops_test.go`) |
| ffmpeg / ffprobe presence | `hasFfmpeg` / `hasFfprobe` | `exec.LookPath` cached via `sync.Once` | `SetFfmpegAvailableForTest` seeds the cache and returns a restorer |
| temp-file creation | `createTempFile` | `os.CreateTemp("", …)` | Tests override it to force create/write/close failures and assert cleanup |

The `Runner` seam is the load-bearing one: it lets every test exercise
the argv-construction logic without depending on a real ffmpeg binary.
The presence check is intentionally not a general seam in production;
`SetFfmpegAvailableForTest` is the documented test hook.

## Failure Modes

| Cause | Detected by | Wire mapping (`mapAudioErr`) |
|---|---|---|
| Empty `audio` bytes | `requireBytes` | `CodeInvalidArgument` |
| Empty merge source | `Merge` / handler | `CodeInvalidArgument` |
| Negative / non-finite time window | op-level validation | `CodeInvalidArgument` |
| `end_seconds <= start_seconds` | `Trim` | `CodeInvalidArgument` |
| Unsorted / duplicate / non-finite split boundary | `Split` | `CodeInvalidArgument` |
| Unknown output format (ffmpeg muxer error) | `runFfmpeg` | `CodeInvalidArgument` |
| ffmpeg not on PATH | `hasFfmpeg() == false` | `CodeFailedPrecondition` |
| ffprobe not on PATH | `Probe` | `CodeFailedPrecondition` |
| Context canceled or deadline exceeded | `runFfmpeg` / `runFfprobeJSON` | `CodeCanceled` / `CodeDeadlineExceeded` |
| Temp-file creation failure (disk full, perms) | `writeTempInputs` / `runFfprobeJSON` | `CodeInternal` |
| Unexpected ffmpeg error | stderr-joined error | `CodeInternal` |

## Capacity Notes

Every operation forks an ffmpeg subprocess. Concurrency is bounded by
the host's CPU and ffmpeg's own threading; there is no in-process
queue or rate limit. Operators expecting bursty traffic should front
the API with a request-concurrency limit upstream rather than adding
one inside the chain.

Pipe-based operations (transcode, trim, fade, volume, normalize)
allocate roughly `2× audio bytes` (input buffer + ffmpeg stdout
capture) and one subprocess per request. The temp-file paths (probe,
metadata, merge, split) additionally write to `os.TempDir()`; large
inputs hit the disk twice (write, then read by ffmpeg/ffprobe).
Cleanup is deferred so a panic between create and defer registration
does not leak; paths are recorded immediately after creation.

`Split` is N+1 invocations of ffmpeg (one `Probe`, N `Trim` calls).
Splitting a long file into many chunks is therefore O(N) in
subprocess overhead; a future optimization would use ffmpeg
`-segment_time` to emit all chunks in one pass. The current shape
favors implementation simplicity over per-call latency, and the chunk
count is capped so a bad request cannot spawn an unbounded process
fan-out.

`Merge` writes every source to a temp file before invoking ffmpeg
with `-i path` for each. Merging many sources opens many temp files
concurrently; the host's `ulimit -n` is the practical ceiling. Sources
are expected to share a compatible sample rate/layout; mismatches
fail at ffmpeg rather than being silently resampled.

The Connect path has no explicit upload cap of its own and inherits
whatever limit the upstream gateway enforces. The old multipart
`ParseMultipartForm` cap no longer applies because the REST mount was
retired.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry (Runner)
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — durable decisions
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — current drift
- [`../../reference/configuration.md`](../../reference/configuration.md) — operator-tunable levers
- [`../../reference/models.md`](../../reference/models.md) — model/voice registry
- `packages/proto/schemas/audio-tools/v1/audio/audio.proto` — wire shape
- [`../../internal/MATURATION-PROGRAM.md`](../../internal/MATURATION-PROGRAM.md) — declaration/catalog program
- [`../../reference/api-endpoints.md`](../../reference/api-endpoints.md) — endpoint catalogue
