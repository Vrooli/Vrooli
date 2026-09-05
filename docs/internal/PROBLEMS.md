# Documentation Problems

This file tracks the highest-value remaining documentation debt after the main project-level rewrite.

## Current Known Problems

- Some specialized leaf docs are still intentionally thin and may need deeper rewrites later.
- The docs will drift again unless command surfaces and canonical pages are updated together.
- Scenario-level and resource-level docs can still accumulate overlap with project-level canon if not curated.
- Plan documents remain numerous and need discipline so they do not get mistaken for current truth.

## Highest-Value Remaining Follow-Up

- RCL still uses the flat proto layout for its ledger domains. The planned
  screaming-architecture move to `packages/proto/schemas/react-component-library/v1/domain/`
  is intentionally deferred; the manifest records the transitional fallback.

- continue curating specialized leaves under `path:docs/deployment/`, `path:docs/strategy/`, and other focused sections
- keep `docs/manifest.json` aligned with the actual canonical tree
- keep project-level command reference pages aligned with `go run ./cmd/vrooli ... --help`
- keep scenario and resource system docs small, cross-cutting, and clearly distinct from owning subsystem docs

- Rung: W3 / R0 — version-model and preview closure re-measurement
  - Evidence: Catalog type-check, focused component/preview/gate/version-ledger tests, and governed cold-source materialization all pass. The materialization path now restores byte-identical retired archives and avoids duplicate indexing when a restored version remains declared evicted. `versions doctor --json` reports no issues, and the orphaned `MorphingIcon@3.1.3-draft.1` row was removed by governed source indexing.
  - Outstanding: the current matrix still has 192 attributable documentation, surface-discipline, i18n, and composition findings; scenario-wide UI lint remains red on inherited copy-stability and legacy lint debt (reported to scenario-qa as `knw-1788190879155112230`); and the broad experience suite remains an inherited failure surface documented by the server-owned run.
- Measured: 2026-08-31.

- Rung: W3 / R0 — locale-registry closure follow-up
  - Evidence: stale `ar` and `ja` keys were removed to match the curated
    English registry. Locale parity passes 5/5, the generated registry is in
    sync, catalog conformance has zero errors, and the UI production build
    passes.
  - Outstanding: the broad scenario lint and full server-owned run remain
    inherited failures documented in the DOD audit; this focused cleanup does
    not claim those gates.
  - Measured: 2026-08-31.

- Rung: W3 / R0 — component-test public-identifier seam
  - Evidence: catalog ids are now resolved to canonical component ids before
    component-test reports persist. Focused Go tests pass, and the live
    lifecycle-restarted API returned HTTP 200 and a passed 15-result report
    for `controls.slider` / `react-component-library:Slider@1.1.2`.
  - Outstanding: the full server-owned suite still has unrelated inherited
    unit, storage, experience, tidiness, and provider findings. The targeted
    component-tests phase is now green after the governed template-manager
    local-replace repair.
  - Measured: 2026-08-31.

- Rung: W3 / R0 — current UI coverage baseline
  - Evidence: `pnpm run test:coverage` currently reports 511/536 tests passing
    and 19 failed files; the failures are concentrated in stale preview-kit,
    adoption, shell, and copy/geometry assertions. Coverage still measures
    97.96% statements and 88.54% branches, but the command exits non-zero.
  - Outstanding: this inherited UI test debt keeps the scenario-wide unit and
    lint/check surfaces red; no version-model gate is being weakened to hide it.
  - Measured: 2026-08-31.

- Rung: W1 / R1 — multi-mount recovery targets
  - Storage Manager currently computes a target for the reported partition.
    Recovery across several mounts needs an explicit allocation policy and
    must not treat free bytes on one device as relief for another.
  - Next step: add a multi-mount target contract and fixture-backed controller
    tests before enabling cross-device recovery.
  - Measured: 2026-09-02.

- Rung: W1 / R1 — live macOS and Windows defense verification
  - The watchdog and log-bound safeguards have build and declared-platform
    evidence. Live hosts for launchd and Task Scheduler were not available for
    this implementation pass.
  - Next step: run the platform-specific setup and floor checks on native hosts
    and record the terminal verdicts.
  - Measured: 2026-09-02.
