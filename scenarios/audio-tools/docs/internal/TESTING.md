# Testing — Audio Tools

Run the scenario suite through the server-owned test runner:

```bash
vrooli scenario test audio-tools
```

Wait once with the command returned by the runner. Do not poll or start API
binaries directly.

## API tests

Place domain behavior tests beside the owning package under `api/internal/`.
Place transport tests beside `api/handlers/<domain>/`. Use generated proto
types and real SQLite test handles where persistence behavior matters. Keep
handlers thin and test business rules below the transport boundary.

### Shared test infrastructure

Audio fixtures belong under `api/internal/stt/segmenter/testaudio`. Use the
named duration fixtures (`SpeechTonePauseTone3s`, `Silence1s`) or the
sample-rate-aware builders there instead of repeating PCM byte arithmetic in a
test. Provider tests should use the shared fakes in
`api/internal/ai/sttchain/mocks`; builders return fresh instances so call
counts and scripted results cannot leak between tests.

Browser audio tests use the shared `ui/src/audio-integration/test-support`
fixtures for `MediaStream` shapes. Browser APIs, clocks, transport clients and
server endpoints must be injected or mocked at their boundary; tests must not
wait on scheduler timing to establish a transport state. Prefer a readiness
channel or an explicit seam over `time.Sleep`.

Run focused checks during development:

```bash
cd api && GOWORK=off go test ./...
```

## CLI tests

Place command tests under `cli/domains/<domain>/`. Every runtime command must
be declared in `cli/manifest.json` or listed in `exceptions` with a valid
special-case reason. Run:

```bash
cd cli && GOWORK=off go test ./...
```

## UI tests

Place feature tests beside the component or hook. Use generated clients and
assert user-visible behavior, including failure and accessibility paths. Run:

```bash
pnpm test:coverage
```

The coverage gate is a regression boundary. Add behavior-focused tests before
raising its threshold. Do not weaken the configured threshold to pass a run.

## Streaming checks

After changes to STT selection or stream events, run diagnostics and a smoke
transcription. The stream must produce partial output and a final segment.

```bash
audio-tools diagnostics run
audio-tools stt transcribe-stream --file "<smoke-fixture>"
```

For the complete scenario contract, see `.vrooli/testing.json`,
[`SEAMS.md`](SEAMS.md), and [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md).
