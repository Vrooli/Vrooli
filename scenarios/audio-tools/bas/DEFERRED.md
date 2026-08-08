# BAS Coverage — Active deterministic qualification with explicit remaining gates (2026-07-12)

The `bas/` directory was scaffolded-empty from 2026-05-16. It now has both
observer flows and three requirement-bound deterministic-media playbook cases.
The cases have executed successfully against the real browser capture and
WebSocket product path; they are not a substitute for the remaining provider
fault matrix or the required physical-device qualification.

## What is activated

- `flows/audio-tools-health-probe.json` — boots the UI, navigates to
  `/diagnostics`, asserts the capability self-test control (`suite-run`) renders.
  Non-media; deterministic. Smoke guard against the app shell / Diagnostics page
  failing to mount.
- `flows/audio-tools-diagnostics-suite.json` — clicks **Run full diagnostics**,
  which drives STT + TTS + Summarize + Transcode end-to-end against the live
  providers, then waits for and asserts the last-run summary. This is the
  regression guard for "Summarize broke" / "Transcode broke": clicking Run here
  surfaces it instead of a user. Non-media (text + embedded fixtures only).
- `flows/audio-tools-dictation-studio-scripted-bench.json` — opens
  `/dictation-studio`, switches to scripted mode, selects a built-in script,
  verifies prompt/tag prefill plus recorder affordances, then checks the Corpus
  and Eval tab surfaces. Deterministic; it does not synthesize microphone input
  or save a physical recording.

Both are `execution_mode: observer` flows in `flows/` (perf/visual-capture home),
runnable via `performance-health` / visual capture. They complement, rather than
replace, the following functional cases:

- `cases/01-foundation/01-dictation/deterministic-microphone-smoke.json` —
  requirement `ATD-P0-006`; execution
  `92e2f7e6-a005-411b-a395-bcf299743401` proved real `getUserMedia`, PCM-v2
  delivery, terminal `done`, and durable processed acknowledgement. This rerun
  explicitly checks the lifecycle attributes rather than merely selector presence.
- `cases/01-foundation/01-dictation/deterministic-provider-busy.json` —
  requirement `ATD-P0-005`; execution
  `35820ee3-2145-4f83-bac7-d62a56e0b461` proved visible typed busy recovery
  using the WAV-backed fake microphone and an explicit text assertion.
- `cases/01-foundation/01-dictation/deterministic-incomplete-coverage.json` —
  requirement `ATD-P0-004`; execution
  `ed1006b3-8aff-4c65-8855-2f941a2b7c22` proved that an intentionally withheld
  terminal acknowledgement is surfaced as a product-visible error and terminal
  `incomplete_coverage` diagnostic after actual PCM capture.
- `../web-console/bas/cases/02-messages/02-voice/deterministic-incomplete-coverage.json` —
  cross-consumer product-path execution
  `06696480-3902-47e3-8db6-8e7971ef42e0` proved a WAV-backed real microphone
  frame crosses Web Console's same-origin proxy, then a bounded post-capture
  interruption surfaces `incomplete_coverage` and the metadata-only diagnostic
  export. This catches the client-side class where a plain close otherwise
  looks like a harmless empty transcript.

## What is still deferred (and why)

1. **Broader requirement-bound product paths.** The three shipped cases cover
   browser PCM capture, typed busy recovery, and visible unacknowledged terminal
   coverage. The plan still requires stable
   cases for Kyutai queue-then-ready, reconnect/resume, retained-audio recovery,
   record-to-corpus-to-provider-comparison, diagnostics export, and experimental
   engine blocking, plus the corresponding Web Console paths.
2. **Fault matrix.** BAS now supports a per-execution fake-device
   path: a workflow declares `settings.fake_media.microphone_wav` (resolved
   against the execution's `project_root`; see
   `bas/fixtures/dictation-reference.wav`), and the driver pools a dedicated
   Chromium instance with the fake-capture flags plus context-level microphone
   permission — no specially booted driver required (validated by execution
   `3676aa0c-5fce-4159-bf93-17319d2e5d30`, 2026-07-12). The legacy
   `BAS_FAKE_MICROPHONE_FILE` env knob remains only as the default-browser
   fallback for dedicated qualification drivers. Audio Tools now has double-gated deterministic WebSocket
   faults (`provider_busy`, `close_after_chunk:N`, `close_after_chunk_recoverable:N`, `close_after_commit:N`,
   `pause_reads_after_chunk:N:MS`, `delay_processed_ack_ms:MS`, and
   `suppress_processed_ack`), and BAS now has authored requirement-bound
   `deterministic-provider-busy`, `deterministic-incomplete-coverage`, and
   `deterministic-reconnect-recovery` cases. The reconnect case is a bounded
   one-shot browser replay proof; it is not restart or full provider-resource
   qualification.
   Browser page-profile headers do not reach a page-created WebSocket handshake,
   so the case instead uses explicit test URL parameters guarded by the active
   server-owned routed-isolation lease. BAS execution
   `35820ee3-2145-4f83-bac7-d62a56e0b461` (2026-07-12) passed with a visible
   typed provider-busy recovery message, and execution
   `ed1006b3-8aff-4c65-8855-2f941a2b7c22` passed the missing-acknowledgement
   recovery path. Normal deployments ignore those URL parameters because they
   never have an active routed-isolation lease. The remaining named
   fault controls are delayed-ready, slow reader, missing acknowledgement,
   close-before-done, backend restart, muted/ended track, journal quota,
   verifier/extractor outage, and page interruption. Each needs a product-path
   assertion and persisted seed/trigger evidence before it can count toward
   promotion.
3. **Physical-device qualification.** Chromium fake audio cannot reproduce
   iOS audio-session behavior or establish the required iOS Safari, installed
   PWA, Android Chrome, and desktop-device matrix. Keep this as an explicit
   manual gate and record only observed results via `business-health manual-log`.

## Remaining-coverage triggers

1. A non-media UI flow lands with no `ui/src/**/*.test.tsx` coverage.
2. A user-reported diagnostics/configuration regression slips past unit tests →
   land it as a `cases/<surface>/` case to prevent recurrence.
3. A consumer adopts the shared browser-capture package → land its requirement-bound
   product-path case, not only `flows/embed-<consumer>-smoke.json`.
4. A newly added fault seam → land a deterministic case with coverage, terminal,
   recovery, and diagnostic assertions.
