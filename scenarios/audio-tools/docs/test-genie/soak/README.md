# Long-form dictation soak phase

**ID**: `soak`

The phase is the executable Test Genie gate for the accelerated browser
qualification. It uses the exact `virtual-replay` / `virtual-corpus` /
`passthrough` / `default` cell and requires a complete conformance artifact.

## North Star

The browser product path, capture package, WebSocket ledger, server runtime,
and provider complete the declared accelerated qualification with every
required assertion measured and passing. The phase proves accelerated
invariants and accounting; it does not claim realtime latency or human-device
behavior.

## The rungs and their gates

- **L0 — Unavailable:** the browser driver/UI inputs are missing, the provider
  cannot run, or the artifact is incomplete.
- **L1 — Measured:** the registered provider produced a complete conformance
  artifact for the exact provider cell.
- **L2 — Qualified:** every required accelerated assertion in that artifact is
  `passed`.

The phase is gating. Missing configuration, a failed assertion, or a
`not_measured` assertion fails the phase; it is never silently downgraded to a
warning or skipped result.

## What each finding means

- `SOAK_CONFIGURATION_MISSING` means the BAS driver, UI URL, or canonical WAV
  fixture was unavailable to the provider process.
- `SOAK_QUALIFICATION_FAILED` means the provider ran (or attempted to run) the
  product path but did not produce a qualified artifact. The finding message
  includes the provider-owned failure detail and artifact reference when one
  exists.

## The canonical fix

Start the API in the explicit qualification environment
`VROOLI_AUDIO_SOAK_REPLAY=1`, provide a live BAS Playwright driver through
`SOAK_DRIVER_URL` or `PLAYWRIGHT_DRIVER_URL`, a running Dictation Studio UI
through `SOAK_UI_URL` or `AUDIO_TOOLS_UI_URL`, and the canonical fixture. Then
inspect the persisted conformance artifact and fix the first failed assertion
before rerunning the phase. The replay flag is deliberate: this provider is
absent from the production engine manifest and must never become a user-facing
fallback.

## How to verify

```bash
test-genie execute audio-tools --phases soak
test-genie runs findings --scenario audio-tools
```

The provider also persists the one conformance document under
`scenarios/audio-tools/coverage/`. The phase response carries its run ID,
provider cell, assertion count, and artifact reference as opaque native detail
for provider-owned diagnostics.
