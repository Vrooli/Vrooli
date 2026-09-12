# Audio Transformations

This document is the canonical architecture reference for the
ffmpeg-backed audio operations exposed by `AudioProcessingService`:
transcode, trim, fade, volume, normalize, split, merge, and
extract-metadata. It explains how every operation flows from the
Connect-RPC edge into a single `Runner` seam, where temp files appear,
and how the failure modes map onto Connect codes.

Read this first when:

- adding a new ffmpeg-backed transformation,
- swapping the ffmpeg invocation strategy (e.g., adopting libav
  bindings, switching to a long-lived ffmpeg server),
- changing the multipart REST exception (`POST /api/v1/audio/transcode`),
- debugging "why does my merge crash with a tempfile error?" or "why
  do I get `FailedPrecondition` when ffmpeg is installed?".

The audio domain is intentionally provider-free — there is no BYOK,
Vrooli, or Local tier here. Every operation is local CPU work bounded
by ffmpeg. Cross-tier abstractions belong in `ttschain`/`sttchain`/
`summarizechain`, not here.

## Purpose

`internal/audio` (`api/internal/audio/ops.go`,
`api/internal/audio/transcode.go`) owns the ffmpeg shell. It is the
single home for:

- the `Runner` seam (`api/internal/audio/transcode.go:45`) — every
  ffmpeg / ffprobe invocation routes through it,
- presence detection (`hasFfmpeg`, `hasFfprobe` —
  `api/internal/audio/transcode.go:26`),
- the `ErrFFmpegMissing` sentinel
  (`api/internal/audio/transcode.go:17`),
- temp-file lifecycle for operations that need seekable input
  (`runFfprobeJSON`, `writeTempInputs` —
  `api/internal/audio/ffprobe.go`).

Handlers (`api/handlers/audio/*.go`) are thin translation shims: copy
request bytes in, call one `intaudio` function, copy bytes out, map
errors via `mapAudioErr` (`api/handlers/audio/errors.go:22`).

## Inputs

Every Connect method accepts raw audio bytes plus operation-specific
parameters. The shared guard `requireBytes`
(`api/handlers/audio/errors.go:12`) rejects empty payloads with
`CodeInvalidArgument` before any ffmpeg call.

| Operation | Handler | Inputs |
|---|---|---|
| `Transcode` | `api/handlers/audio/transcode.go:16` | `audio`, `output_format`, `sample_rate`, `channels`, `bitrate` (all optional except audio) |
| `Trim` | `api/handlers/audio/trim.go:13` | `audio`, `format`, `start_seconds`, `end_seconds` |
| `Fade` | `api/handlers/audio/fade.go` | `audio`, `format`, `fade_in_seconds`, `fade_out_seconds`, `output_format` |
| `Volume` | `api/handlers/audio/volume.go` | `audio`, `format`, `gain_db`, `output_format` |
| `Normalize` | `api/handlers/audio/normalize.go` | `audio`, `format`, `method` (`peak`/`rms`/default EBU R128), `target_lufs`, `output_format` |
| `Split` | `api/handlers/audio/split.go:13` | `audio`, `format`, `chunk_seconds` OR `boundaries_seconds`, `output_format` |
| `Merge` | `api/handlers/audio/merge.go:14` | `sources[]` (each `{audio, format}`), `output_format`, `crossfade_seconds` |
| `ExtractMetadata` | `api/handlers/audio/metadata.go:13` | `audio` |

The REST multipart fallback `POST /api/v1/audio/transcode`
(`api/handlers/audio/transcode.go:35`) is registered as a deliberate
REST exception in the endpoints descriptor
(`api/handlers/audio/connect_handler.go:33`) — clients that cannot
form a Connect request (browsers uploading a `<input type=file>`) get
a multipart form with `audio` part plus optional
`output_format`/`sample_rate`/`channels`/`bitrate` fields.

## Outputs

Every transformation returns transformed bytes plus a `content_type`
string derived from `contentTypeFor`
(`api/internal/audio/ops.go:267`):

| Format | `content_type` |
|---|---|
| `wav` (default) | `audio/wav` |
| `mp3` | `audio/mpeg` |
| `flac` | `audio/flac` |
| `aac` | `audio/aac` |
| `ogg` | `audio/ogg` |
| anything else | `application/octet-stream` |

`Split` returns `SplitResponse.chunks[]` where each chunk carries
its own bytes, content-type, `start_seconds`, and `duration_seconds`
(`api/handlers/audio/split.go:22`). The last chunk has `duration=0`
as an EOF sentinel.

`ExtractMetadata` returns a typed `AudioMetadata` populated from
`ffprobe -of json` output
(`api/internal/audio/ops.go:116`): `duration_seconds`, `sample_rate`,
`channels`, `bitrate`, `codec`, `format`, plus a `tags` map mirroring
the container's tag set.

## Internal Chain

Most operations follow a single shape: handler validates inputs,
calls a stateless `intaudio.Xxx(ctx, audio, ...)` function which
formats ffmpeg argv and pipes audio through stdin → stdout.

```
ConnectRPC request                                     multipart POST
        │                                                    │
        ▼                                                    ▼
requireBytes / typed validators            r.ParseMultipartForm + FormFile read
        │                                                    │
        ▼                                                    │
intaudio.Transcode / Trim / Fade / ...                       │
        │                                                    │
        ▼                                                    ▼
runFfmpeg (api/internal/audio/transcode.go:73)               │
        │                                                    │
        ▼                                                    │
DefaultRunner.Run("ffmpeg", stdin, args...)  ◄───────────────┘
        │
        ▼
exec.CommandContext("ffmpeg", "-y", "-loglevel", "error", ...)
        │
        ▼
stdout (transformed bytes)   |   stderr (joined into error message)
```

Two operations break the pure pipe-in/pipe-out shape:

1. **`Probe` and `ExtractMetadata`** — ffprobe needs seekable input,
   so the implementation writes to a temp file
   (`api/internal/audio/ffprobe.go:12`) and invokes ffprobe on the
   path. Temp file is unconditionally removed via `defer os.Remove`.
2. **`Merge`** — concat / crossfade filter graphs need seekable
   per-source input, so every source is written to a temp file via
   `writeTempInputs` (`api/internal/audio/ffprobe.go:44`). Cleanup is
   driven by `tempInputs.cleanup()` invoked through `defer`
   (`api/internal/audio/ops.go:239`).

`Split` is a composite: when `chunk_seconds > 0` it calls `Probe` to
discover duration, derives cut points, then issues N `Trim` calls
sequentially (`api/internal/audio/ops.go:194`). When explicit
`boundaries_seconds` are supplied it skips the probe.

`Transcode` has two entry points: the canonical `Transcode(ctx, audio)`
hard-codes 16 kHz mono WAV (the STT-pipeline default,
`api/internal/audio/transcode.go:83`), and `TranscodeOpts` honors
caller-supplied sample-rate/channels/bitrate/format
(`api/internal/audio/transcode.go:94`). The Connect handler calls
`TranscodeOpts` so clients can specialise; the canonical entrypoint
exists for upstream packages that just need STT-ready PCM.

### ffmpeg invocation contract

Every ffmpeg call prepends `-y -loglevel error`
(`api/internal/audio/transcode.go:77`). The intent:

- `-y` — silently overwrite (we are writing to `pipe:1` so this is
  mostly a safety net for future temp-file paths).
- `-loglevel error` — keep stderr terse so the wrapped error message
  is human-readable when ffmpeg fails.

Stderr is captured and joined into the returned error
(`api/internal/audio/transcode.go:60`) so callers see the actual
ffmpeg complaint, not just "exit 1".

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| ffmpeg / ffprobe runner | `audio.Runner` (`api/internal/audio/transcode.go:45`) | `execRunner` invoking `os/exec.CommandContext` | `DefaultRunner` is package-level mutable so tests can swap in a fake recording argv and returning canned stdout (`api/internal/audio/ops_test.go`) |
| ffmpeg presence | `hasFfmpeg` (`api/internal/audio/transcode.go:26`) | `exec.LookPath("ffmpeg")` cached via `sync.Once` | Currently process-global; tests that need to simulate absence must run on a system without ffmpeg |
| Temp-file paths | `writeTempInputs` (`api/internal/audio/ffprobe.go:44`) | `os.CreateTemp` | Not seamed; production uses real filesystem in tests too |

The `Runner` seam is the load-bearing one: it lets every test exercise
the argv-construction logic without depending on a real ffmpeg
binary. The presence check is intentionally not seamed because
production behavior (return `ErrFFmpegMissing` → handler returns
`CodeFailedPrecondition`) is exactly what we want under test on a
ffmpeg-less host.

## Failure Modes

| Cause | Detected by | Wire mapping (`mapAudioErr`, `api/handlers/audio/errors.go:22`) |
|---|---|---|
| Empty `audio` bytes | `requireBytes` | `CodeInvalidArgument` |
| ffmpeg not on PATH | `hasFfmpeg() == false`; `runFfmpeg` returns `ErrFFmpegMissing` | `CodeFailedPrecondition` |
| ffprobe not on PATH | `Probe` returns `ErrFFmpegMissing` | `CodeFailedPrecondition` |
| ffmpeg non-zero exit | Stderr-joined error from `execRunner.Run` | `CodeInternal` |
| `Trim` with negative `start_seconds` | Explicit guard (`api/internal/audio/ops.go:27`) | `CodeInternal` (current — could tighten to `InvalidArgument`) |
| `Merge` with zero sources | `Merge` returns `"merge requires at least one source"` | Handler also pre-rejects with `CodeInvalidArgument` (`api/handlers/audio/merge.go:15`) |
| Temp-file creation failure (disk full, perms) | `writeTempInputs` / `runFfprobeJSON` | `CodeInternal` |
| Unknown output format | ffmpeg muxer error | `CodeInternal` (ffmpeg stderr surfaced) |
| Multipart endpoint, missing `audio` part | `r.FormFile("audio")` error | HTTP `400 Bad Request` (this path bypasses `mapAudioErr`) |
| Multipart endpoint, ffmpeg missing | Error path branches on `ErrFFmpegMissing` | HTTP `424 Failed Dependency` (`api/handlers/audio/transcode.go:69`) |
| Multipart endpoint, other ffmpeg error | Default path | HTTP `502 Bad Gateway` |

The multipart endpoint uses HTTP status codes directly rather than
Connect's mapping because the REST exception exists precisely for
clients that aren't speaking Connect.

## Capacity Notes

Every operation forks a ffmpeg subprocess. Concurrency is bounded by
the host's CPU and ffmpeg's own threading; there is no in-process
queue or rate limit. Operators expecting bursty traffic should
front the API with a request-concurrency limit upstream rather than
adding one inside the chain.

Pipe-based operations (transcode, trim, fade, volume, normalize)
allocate roughly `2× audio bytes` (input buffer + ffmpeg stdout
capture) and one subprocess per request. The temp-file paths
(`probe`, `metadata`, `merge`, `split`) additionally write to
`os.TempDir()`; large inputs hit the disk twice (write, then read by
ffmpeg/ffprobe). Cleanup is deferred so a panic between create and
defer registration would leak; the current code structure avoids any
non-trivial logic in that window.

`Split` is N+1 invocations of ffmpeg (one `Probe`, N `Trim` calls).
Splitting a long file into many chunks is therefore O(N) in
subprocess overhead; a future optimization would use `ffmpeg
-segment_time` to emit all chunks in one pass, but that change has
not been made — the current shape favors implementation simplicity
over per-call latency.

`Merge` writes every source to a temp file before invoking ffmpeg
with `-i path` for each. Merging 100 sources opens 100 temp files
concurrently; the host's `ulimit -n` is the practical ceiling.

Multipart upload size is capped at 64 MiB by `ParseMultipartForm`
(`api/handlers/audio/transcode.go:37`); larger uploads return
`400 Bad Request`. The Connect path has no explicit cap and inherits
whatever limit the upstream gateway enforces.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry (Runner)
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — durable decisions
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — current drift
- [`../../reference/configuration.md`](../../reference/configuration.md) — operator-tunable levers
- `packages/proto/schemas/audio-tools/v1/audio/audio.proto` — wire shape
- [`../../reference/api-endpoints.md`](../../reference/api-endpoints.md) — endpoint catalogue (Connect + REST exception)
