# E7 — Scenario App Deployment and Browser Automation Studio Device Testing

## Outcome
Make a scenario-built app easy to install on an exact supported device for a bounded test or durable use, then reliably exercise it through Browser Automation Studio (BAS) with inspectable evidence.

## Owners and starting points

- `scenarios/scenario-to-android/docs/reference/cli-commands.md`: `android run`, artifact generation, matrices and retained review recordings. Trace current behavior before adding a path.
- `scenarios/device-control/docs/reference/capabilities.md`: immutable producer refs and lease-owned `install-artifact`. These are verified building blocks, not yet proof of an integrated test/deployment lifecycle.
- `scenarios/browser-automation-studio/docs/START-HERE.md`, `docs/PROBLEMS.md`, its canonical skill and existing `smoke-flow`, `find-flows`, and `author-flow` programs own BAS execution and reliability. Reuse them.
- Scenario-to-Android owns build/signing/artifact meaning; Device Control owns exact identity, lease/install lifecycle; BAS owns browser workflows/assertions/evidence; Test Genie owns scenario suites.

## Work

1. Trace scenario/source revision → Android build/immutable ref → Device Control exact-ref install → app readiness/launch → BAS workflow → terminal assertion/evidence. Record support by target class, missing seams and owner contracts. Use Android owner for Android, and identify the correct producer for any other platform in scope.
2. Define **temporary test install** with exact artifact/device identity, pre-test app state, an explicit isolated cleanup/restore plan, post-cleanup verification and retained evidence. Never silently overwrite/remove user apps or data. If restoration/isolation cannot be proven, refuse cleanup or use a more isolated target and report the residual state.
3. Define **durable install/update** with immutable ref, digest, package, signing/version identity, install audit, upgrade/data semantics, and post-install verification. Reject identity/signing mismatches before actuation.
4. Provide one discoverable operator workflow for each intent; avoid requiring manual APK-path handoffs. Compose existing scenario-to-android, Device Control and BAS owners rather than duplicating build, device or browser workflow engines.
5. Run BAS against installed scenario apps on real supported Android targets where possible. Cover clean install/cold start, warm relaunch, readiness/selectors, transient timing/network conditions, exact terminal assertions, available screenshots/logs/recordings, retries, and cleanup after failure. Preserve first-attempt reliability separately from retry-assisted completion.
6. Repair diagnosable failures in the owning layer and retain artifact/device/workflow/run identity, failed chapter, bounded logs, lifecycle state, and retry/cleanup outcome. Keep target capability and unavailable evidence explicit.

## Proof

- Producer and Device Control tests prove immutable identity, lease binding, mismatch refusal before actuation, install and post-install state.
- A temporary test install proves safe cleanup/restore, or a safe refusal with no destructive side effect; a durable install/update remains installed after fresh inventory. State exactly how pre-existing app data is protected.
- BAS terminal runs exercise scenario-built apps on each supported target class, with cold and warm starts and exact assertions. Repeat enough runs to expose flakiness; report first-pass rate, retries and latency.
- Owner Test Genie runs reach terminal results. Physical evidence identifies device, artifact and workflow revision. Build success alone does not satisfy E7. Unsupported targets and human prompts remain explicit.
