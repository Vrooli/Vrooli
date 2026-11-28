const e=`# Long-form dictation soak phase

**ID**: \`soak\`

The phase is the executable Test Genie gate for the accelerated browser
qualification. It uses the exact \`virtual-replay\` / \`virtual-corpus\` /
\`passthrough\` / \`default\` cell and requires a complete conformance artifact.

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
  \`passed\`.

The phase is gating. Missing configuration, a failed assertion, or a
\`not_measured\` assertion fails the phase; it is never silently downgraded to a
warning or skipped result.

## What each finding means

- \`SOAK_CONFIGURATION_MISSING\` means the trusted Agent Manager or Test Genie
  receipt, BAS endpoint, owner-selected UI, or canonical WAV fixture was unavailable.
- \`SOAK_QUALIFICATION_FAILED\` means the provider ran (or attempted to run) the
  product path but did not produce a qualified artifact. The finding message
  includes the provider-owned failure detail and artifact reference when one
  exists.

## The canonical fix

Issue the request through an active Test Genie demand lease with trusted Agent Manager identity and browser-grant receipts. Audio Tools verifies the lease and resolves its configured BAS owner, UI, and canonical WAV fixture; callers cannot select a browser driver, UI, or local fixture. Then inspect the persisted conformance artifact and fix the first failed assertion before rerunning the phase. The lease and browser grant are request-scoped and are never persisted in evidence.

## How to verify

\`\`\`bash
test-genie execute audio-tools --phases soak
test-genie runs findings audio-tools
\`\`\`

The provider also persists the one conformance document under
\`scenarios/audio-tools/coverage/\`. The phase response carries its run ID,
provider cell, assertion count, and artifact reference as opaque native detail
for provider-owned diagnostics.
`;export{e as default};
