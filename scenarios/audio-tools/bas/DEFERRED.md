# BAS Coverage — Partial activation (2026-06-29)

The `bas/` directory was scaffolded-empty from 2026-05-16. As of 2026-06-29 the
non-media diagnostics surface is now covered by **authored observer flows** (see
`flows/`). The functional **playbook-case** suite (requirement-bound, gated by the
`playbooks` phase) remains deferred for the reasons below — but the registry is no
longer a deliberate empty-pass: real flows exist and exercise the live surface.

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
runnable via `performance-health` / visual capture. They are **not yet** bound as
functional `cases/` playbooks — see below.

## What is still deferred (and why)

1. **Requirement-bound playbook cases.** A functional playbook in
   `cases/<group>/` must bind to a requirement ID via
   `labels.requirements_json` (see command-center). audio-tools' requirements are
   still the starter stub — the only requirement is `FOUND-001` ("Replace starter
   requirements with PRD-specific operational targets"). Binding the diagnostics
   flows to real requirements is blocked on first fleshing out the requirements
   set; binding to `FOUND-001` would be semantically wrong. **Next step:** add
   PRD-specific requirements (e.g. `AUDIO-DIAG-001` self-test runs 4/4), then move
   these two flows into `cases/01-foundation/` with `requirements_json` set, and
   `test-genie registry build`.
2. **Harness validation.** The two flows are authored against real selectors
   (`suite-run`, `suite-last-run`) but have not been executed in the BAS Playwright
   harness in this change; the operator/CI should run the `playbooks` (or
   browser-automation-studio) harness once to confirm green before binding them as
   gating cases — so a flaky selector cannot flip the `playbooks` phase red.
3. **Media / streaming flows.** Microphone capture, audio playback, and WebSocket
   streaming still cannot be driven deterministically without mic-permission grants
   + deterministic media-stream fixtures + fake audio device routing in the shared
   BAS toolchain. When those land, add `flows/stt-streaming.json` and a
   Dictation-Studio real record→save→eval case. The scripted-bench flow covers
   the non-mic UI path only. **Physical mic capture is not faked here** —
   explicitly deferred.

## Activation criteria (unchanged triggers for more coverage)

1. A non-media UI flow lands with no `ui/src/**/*.test.tsx` coverage.
2. A user-reported diagnostics/configuration regression slips past unit tests →
   land it as a `cases/<surface>/` case to prevent recurrence.
3. A consumer adopts `@audio-tools/embed` → `flows/embed-<consumer>-smoke.json`.
4. Mic-permission + fake-audio-stream fixtures arrive → `flows/stt-streaming.json`.
