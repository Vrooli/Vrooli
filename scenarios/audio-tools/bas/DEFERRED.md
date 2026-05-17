# BAS Coverage — Deferred

The `bas/` directory is scaffolded but intentionally empty as of 2026-05-16.

## Status

- `actions/`, `cases/`, `flows/`, `seeds/`: empty.
- `registry.json`: empty `playbooks: []`.

## Why deferred

Audio-tools' user-facing surfaces are dominated by media flows (microphone
capture, audio playback, WebSocket streaming) that the current BAS Playwright
harness cannot drive deterministically without significant additional
fixtures (mic permission grants, deterministic media streams, fake audio
device routing). The non-media diagnostics surface is covered by API +
component tests today; adding BAS coverage on top would duplicate signal
without reducing risk.

## Activation criteria

Populate this directory when **any** of the following becomes true:

1. **A non-media UI flow lands** that has no automated coverage in
   `ui/src/**/*.test.tsx` — e.g., a new admin/configuration wizard, a
   multi-step onboarding flow, or a docs portal change requiring
   navigation assertion.
2. **A user-reported regression** in the diagnostics or configuration UI
   slips past unit/component tests; the regression should land as a BAS
   case under `cases/<surface>/` to prevent recurrence.
3. **A consumer integration scenario** (web-console, swarm-manager,
   phone-agent) adopts `@audio-tools/embed` and needs end-to-end smoke
   coverage of the embed surface; that smoke flow belongs here, named
   `flows/embed-<consumer>-smoke.json`.
4. **Mic-permission and fake-audio-stream fixtures** become available in
   the shared BAS toolchain; at that point the streaming STT path
   becomes BAS-testable and should get a `flows/stt-streaming.json`.

## First flow to add when activated

When the first activation criterion fires, the recommended seed flow is:

- `flows/audio-tools-health-probe.json` — open `/`, assert the health
  card renders, assert capability matrix loads. Pairs with
  `actions/health-probe.json` fixture (mocked capability registry).

Until then, leave the directory empty and let `test-genie registry build`
produce the empty `playbooks: []` so other agents see the deliberate
absence rather than a missing harness.
