# Platform Audit

Rolling artifact owned by `platform-code-auditor`.

Use this file for internal Vrooli platform-code audit snapshots.

## Audit Log

| Date | Slice | Dimension grades | Top finding | Status |
|---|---|---|---|---|
| 2026-08-29 | cli-core | architecture: C; security: B; test coverage: C; documentation: C; cross-platform readiness: B; signal/feedback: C | Manifest drift validation accepts manually supplied nested paths without proving handler reachability. `vroolicli.RegisteredLeafPaths` manually appends legacy paths, and `rootcli.RegisteredLeafPaths` checks only the root (plus `scenario`'s immediate child) before materializing leaves. | capability-work-item filed; measured tests/checker pass |
| 2026-08-30 | cli-scenario-lifecycle | architecture: B; security: B; test coverage: C; documentation: D; cross-platform readiness: B; signal/feedback: C | `CommandSpecs()` omits argument schemas for executable `scenario run`, `scenario test`, and `scenario logs` commands. Generated `vrooli scenario --help` hides their required names/options, while handlers and docs expose separate contracts. | capability-work-item filed as `chore/fix-scenario-cli-help-schemas`; measured focused tests pass |
| 2026-08-31 | lifecycle-internals | architecture: B; security: B; test coverage: C; documentation: C; cross-platform readiness: B; signal/feedback: D | `startTrackedProcess` can report an immediately exiting background command as successfully started: the focused regression test fails 10/10 because the initial PID-liveness observation wins before the exit is observed, so the exit cause/log tail is not surfaced. Secondary harness friction: merged-sandbox registry tests need `env -u VROOLI_SANDBOX_MERGED` because that marker forces fresh SQLite stores read-only. | fix work-item proposed; measured focused test fails 10/10; scenario-runtime and runtime-supervisor packages pass |
