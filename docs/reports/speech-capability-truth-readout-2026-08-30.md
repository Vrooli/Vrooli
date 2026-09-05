# Speech capability truth readout — 2026-08-30

## Result

The capability registry now separates liveness from full readiness and gives
each checker its own bounded context. `/health`, provider health, lifecycle
listing, and the Web Console capability surface use liveness checks. Explicit
health refresh remains the path to the expensive model-work checks.

The liveness OpenRouter checker is a separate instance using the short
liveness client; the full OpenRouter checker retains the longer readiness
client.

The liveness tier is now strict: when a liveness checker is absent it returns
`unknown` and never falls back to a full/readiness checker. This prevents a
partially configured registry from silently reintroducing expensive probes.
Full and liveness checks also execute outside the registry mutex; a blocked
full refresh cannot delay a liveness or synthesis decision.

## Live evidence

- `audio-tools health show --refresh --json` completed in 122 ms and reported effective
  availability for STT, TTS, summarization, and transcode.
- A fresh TTS probe succeeded with valid 28,844-byte MP3 output in 550 ms;
  earlier post-change probes completed in 1,456 ms, 129 ms, and 108 ms.
- The STT quality fixture produced `the quick brown fox jumps` with exit code
  0; the command wall time was 6.69 s because it exercises real local model
  work, not the liveness path.
- Stopping Kokoro made only `kokoro-tts` unavailable; STT, summarization, and
  transcode remained available. Stopping Whisper made only `whisper-stt`
  unavailable; Kokoro TTS and the other capabilities remained available.
- Ollama reports `healthy` with `capacity-sync` alive (pid 414602). Its stale
  installed resource CLI was rebuilt using the canonical `cli-installer`, and
  Ollama was restarted through `vrooli resource restart ollama`.
- A degraded audio-tools snapshot carries `providers:
  speaker-verification` into `health_error` after a governed scenario restart.
  Web Console reports the same provider and its restart command. Speaker
  verification is disabled on this host because its sherpa-onnx dependency is
  unsupported; the degraded state is preserved rather than masked.
- The deferred Linux Whisper/CUDA capability gap is tracked in Scenario QA as
  `knw-1788133188776451942`; no `resources/whisper/**` changes were made.

## Validation

- `go test ./...` in `packages/capability-registry-go`: pass.
- Registry race coverage includes a blocked full-refresh/liveness concurrency
  regression and passes.
- Audio-tools targeted capability, health, provider-lifecycle, bootstrap, and
  TTS tests: pass.
- Web Console capability tests: pass.
- `go test ./internal/scenarioruntime -run 'TestHealthProbe' -count=1`: pass.
- Root `go build ./...` and `make build`: pass.
- Audio capture browser package build and Web Console TypeScript compilation:
  pass.
- Targeted Web Console Vitest files: 69 tests pass, including the visible,
  disabled microphone, recovery-command, and runtime browser-fallback
  assertions.
- The full audio-tools Go package sweep runs all functional packages
  successfully but exits on its pre-existing coverage floors
  (`internal/stt/segmenter` 73.4% < 75%; `handlers/stt` 79.9% < 80%).
- The full Web Console API Go package sweep passes.

## Test Genie queue state

Fresh quick runs were submitted for audio-tools
(`20260830-221119-a5e11b7d`) and web-console
(`20260830-223304-4603edb3`). Both remained queued for their complete wait
windows and executed zero phases. The earlier audio-tools comprehensive run
(`20260830-220322-a0fe9311`) aborted before phase execution because its cached
baseline was classified as legacy. These are infrastructure scheduling and
baseline-state limitations, not assertion failures.

A later clean audio-tools comprehensive run
(`20260830-231411-2f2a2d2e`) executed all phases and returned 14 passed, 12
failed, and 1 skipped. Its failures were scenario-wide pre-existing health
findings (for example missing CLI architecture declarations, coverage/test
surface debt, branding, and provider-conformance metrics), not failures in the
speech capability changes. The affected Go and UI tests were independently
run above and passed.

The clean Web Console comprehensive run
(`20260830-231940-2c1b188f`) also returned `FAIL` after 477.7 seconds. Its
terminal findings are broad scenario debt and unavailable browser automation
infrastructure (including a non-responding Playwright driver); the focused
speech tests, TypeScript build, and capability API tests pass independently.
