# Testing — Network Manager

## TL;DR — the canonical examples

Implemented examples now exist for health snapshots with fake probes (`[REQ:NM-P0-001]`), adapter capability reports (`[REQ:NM-P0-006]`), conservative AdGuard Home resolver status/configuration with a fake client (`[REQ:NM-P0-002]`), conservative DNS policy preview/apply/rollback state with a fake policy adapter (`[REQ:NM-P0-003]`), device identity reconciliation with fake discovery observations (`[REQ:NM-P0-004]`), safe optimization experiments with fake capabilities/applier (`[REQ:NM-P0-005]`), Home Automation action/event publishing with a capturing fake (`[REQ:NM-P0-007]`), privacy retention defaults with fake-clock sweeps (`[REQ:NM-P0-008]`), and operator UI workflow coverage for dashboard/snapshot/resolver-policy/devices/optimization/settings surfaces (`[REQ:NM-P0-009]`).

## API testing

API tests should cover domain services first, then handlers. Required P0 test themes:

- Snapshot report shape, partial probe results, confidence flags.
- Resolver adapter status, dry-run behavior, secret-reference persistence, and failure mapping.
- Policy preview/apply/rollback records, approval gating, unsupported adapter behavior, and rollback failure handling.
- Device identity reconciliation, ambiguity notes, randomized MAC handling, stale-hostname evidence, and group updates.
- Optimization baseline requirement, candidate evidence capture, reliability-first scoring, approval gating, apply failure handling, rollback failure handling, and manual-required candidates.
- Capability reports for supported and unsupported actions, including platform profiles.
- Home Automation action listing, approval-gated invocation, manual-required unsupported write actions, redacted event summaries, and publisher failure recording.
- Optimization retention interaction with before/after experiment ledgers.

## UI testing

UI tests should cover:

- Loading, empty, error, and partial-result states.
- Accessible tables and status summaries.
- Confirmation flows for risky changes.
- Before/after comparison readability.
- Non-color-only status indicators.

Current UI coverage includes:

- API-backed dashboard loading, empty, error, populated snapshot/resolver/capability/privacy states.
- Snapshot run/export actions with report output.
- Resolver upstream dry-run preview and policy preview/apply/rollback confirmation flow.
- Device empty/table states, inventory refresh, and group update behavior.
- Optimization safe-run timeline, candidate scoring, and approval action.
- Settings privacy retention/visibility defaults.

`pnpm test:coverage` currently passes with global coverage above the UI thresholds (97.4% statements, 88.2% branches, 92.2% functions during the Phase 9 slice).

Latest full scenario validation for the Phase 9/10 slice (`vrooli scenario test network-manager`, run `20260623-193107-6a372e44`) completed 16/17 phases: structure, contracts, UI health, standards, architecture, dependencies, quality, docs, performance, unit, playbooks, business, tidiness, security, measures, and proto passed. The remaining failure was the known storage-health RPC EOF tracked in `PROBLEMS.md`.

Requirement traceability was rechecked with `prd-control-tower requirements validate network-manager --json` and `vrooli scenario requirements sync network-manager`. Sync completed with 0 files and 0 statuses changed because the canonical comprehensive suite still exits failed on storage-health EOF. `vrooli scenario orient network-manager` reported 6/8 required steps complete; scaffold-health is blocked by the spawned comprehensive suite. The dependency-decisions checklist had flagged a missing `.vrooli/service.json:dependencies.resources` path, so the manifest now declares explicit empty resource and scenario dependency maps.

## CLI testing

CLI tests should verify every command calls the API and renders the correct output contract. Do not duplicate business decisions in CLI handlers.

## How to add a new proto

1. Add schema under `packages/proto/schemas/network-manager/v1/<domain>/`.
2. Run proto generation.
3. Use generated Go and TypeScript clients.
4. Add handler and CLI coverage.
5. Tag tests with matching `[REQ:ID]`.

## E2E binary smoke gate

The scenario should retain lifecycle smoke coverage: API starts, UI serves, CLI can reach status, and basic docs/manifest checks pass.

## Coverage thresholds

P0 domains need meaningful unit tests before status can move from `planned`. Adapter behavior must use deterministic fakes so tests do not depend on the operator's real network.

## Common patterns and anti-patterns

Patterns:

- Fake network probes.
- Fake resolver clients.
- Fake resolver policy adapters.
- Fake device discovery sources.
- Fake adapter capability reports.
- Fake optimization appliers.
- Capturing Home Automation publishers.
- Fake clock for retention and schedules.
- Golden-ish report assertions for export shape.

Anti-patterns:

- Running live speed tests in unit tests.
- Depending on the operator's router model.
- Marking unsupported probes as failures.
- Applying real DNS/router changes in tests.
- Hand-flipping requirement status without passing evidence.

## Cross-references

- [`../concepts/FLOWS.md`](../concepts/FLOWS.md)
- [`SEAMS.md`](SEAMS.md)
- [`../../requirements/README.md`](../../requirements/README.md)
