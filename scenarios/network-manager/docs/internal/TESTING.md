# Testing — Network Manager

## TL;DR — the canonical examples

Implemented examples now exist for health snapshots with fake probes (`[REQ:NM-P0-001]`), adapter capability reports (`[REQ:NM-P0-006]`), AdGuard Home resolver status/configuration with a fake client (`[REQ:NM-P0-002]`), DNS policy preview/apply/rollback state with fake policy adapters and a fake AdGuard control API (`[REQ:NM-P0-003]`), device identity reconciliation plus AdGuard client import with fake discovery observations/control API (`[REQ:NM-P0-004]`), safe optimization experiments with fake capabilities/applier (`[REQ:NM-P0-005]`), Home Automation action/event publishing with a capturing fake (`[REQ:NM-P0-007]`), privacy retention defaults with fake-clock sweeps (`[REQ:NM-P0-008]`), operator UI workflow coverage for dashboard/snapshot/resolver-policy/devices/optimization/settings surfaces (`[REQ:NM-P0-009]`), advisory household policy profile/schedule coverage (`[REQ:NM-P1-001]`, `[REQ:NM-P1-002]`), guidance-only IPv6/encrypted-DNS bypass plus endpoint/browser DoH coverage (`[REQ:NM-P1-004]`, `[REQ:NM-P1-008]`), and advisory continuous monitoring with fake snapshot comparisons plus SQLite persistence (`[REQ:NM-P1-007]`).

## API testing

API tests should cover domain services first, then handlers. Required P0 test themes:

- Snapshot report shape, partial probe results, confidence flags.
- Resolver adapter status, dry-run behavior, secret-reference persistence, and failure mapping.
- Policy preview/apply/rollback records, approval gating, unsupported adapter behavior, AdGuard global user-rule/protection apply with rollback, and rollback failure handling.
- Household policy profile persistence, validation, filtering strength defaults, and schedule evaluation without live mutation.
- IPv6/encrypted-DNS bypass and endpoint/browser DoH guidance reports that avoid fake enforcement, TLS interception, and hidden monitoring.
- Continuous monitoring schedule validation, baseline anchoring, disabled-schedule advisory behavior, regression detection, and persisted alerts.
- Device identity reconciliation, ambiguity notes, randomized MAC handling, stale-hostname evidence, group updates, and AdGuard `/control/clients` import without query-level DNS log exposure.
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
- Resolver status and rollout labels that separate AdGuard resource protection, this host DNS configuration, observed AdGuard client metadata evidence, still-unverified router/client-wide enforcement, IPv6 DNS/RDNSS handling, router settings, and resolver warning rendering.
- Household profile save/list and schedule evaluation controls.
- IPv6/encrypted-DNS bypass and endpoint/browser DoH guidance controls.
- Device empty/table states, inventory refresh, and group update behavior.
- Optimization safe-run timeline, candidate scoring, and approval action.
- Settings privacy retention/visibility defaults.
- Dashboard continuous monitoring schedule creation, run trigger, and open alert visibility.

`pnpm test:coverage` currently passes with global coverage above the UI thresholds (97.7% statements, 88.42% branches, 87.14% functions during the encrypted-DNS guidance slice).

Latest direct validation for the AdGuard UI parity/live-validation slice passed resource CLI tests/build, Network Manager API/CLI tests/build, UI tests/lint/build, `vrooli resource validate adguard-home`, `vrooli resource schema validate`, `cli-health validate scenario network-manager`, `ui-health validate scenario network-manager`, `scenario-dependency-analyzer health network-manager`, `structure-health validate scenario network-manager`, `prd-control-tower requirements validate network-manager --json`, and `unit-health validate scenario network-manager --execution`. Live smoke confirmed `resource-adguard-home api-health --json` and `network-manager resolver health --json` are healthy with filtering enabled and query logging disabled, `network-manager devices refresh --json` imports two low-confidence AdGuard observations without query-level DNS logs, an inert global block rule can be previewed/applied/rolled back, and post-rollback snapshot `d50f5265-1cb0-4c59-a0fa-b9accb23c4ec` stayed at 9 healthy, 0 degraded, 2 unavailable, and 0 failed probe results. UI lint still reports two preexisting fast-refresh warnings in `routes.tsx` and `ThemeProvider.tsx`; Structure Health and Unit Health pass with existing warning debt. Direct `storage-health validate scenario network-manager` did not return in this run and was terminated, matching the existing external storage validator timeout/EOF class.

Latest full scenario validation for the AdGuard UI parity/live-validation slice (`vrooli scenario test network-manager`) completed 15/17 phases. The failed embedded phases were contracts (`cli-health` RPC unexpected EOF) and storage (`storage-health` RPC unexpected EOF), matching the known external validator instability while direct validators passed as noted above.

Latest validation for the AdGuard rollout affordance slice passed Network
Manager API/CLI tests and builds, AdGuard resource CLI tests/build,
Network Manager UI tests/lint/build, proto lint, `vrooli resource validate
adguard-home`, `vrooli resource schema validate`, direct `cli-health validate
scenario network-manager`, `ui-health validate scenario network-manager`,
`scenario-dependency-analyzer health network-manager`, `structure-health
validate scenario network-manager`, `security-health validate scenario
network-manager`, `prd-control-tower requirements validate network-manager
--json`, and `unit-health validate scenario network-manager --execution`.
Live smoke confirmed `network-manager resolver rollout --json` reports
`host_protected_router_manual`, AdGuard resource protection and this-host DNS
checks pass, router DHCP remains `manual_required`, and default host DNS still
blocks `doubleclick.net` while resolving `example.com`. Full `vrooli scenario
test network-manager` again completed 15/17, failing only the known embedded
contracts/storage EOF phases.

Latest full scenario validation for the continuous monitoring slice (`vrooli scenario test network-manager`, run `20260624-145028-9d371586`) completed 15/17 phases: structure, UI health, standards, architecture, dependencies, quality, docs, performance, unit, playbooks, business, tidiness, security, measures, and proto passed. The failed embedded phases were contracts (`cli-health` RPC unexpected EOF) and storage (`storage-health` RPC unexpected EOF). Direct `cli-health validate scenario network-manager` passed immediately after; direct `storage-health validate scenario network-manager` reproduced the known external validator timeout as `deadline_exceeded`.

Requirement traceability was rechecked with `prd-control-tower requirements validate network-manager --json` and `vrooli scenario requirements sync network-manager`. Sync completed with 0 files and 0 statuses changed because the canonical comprehensive suite still exits failed on embedded validator EOFs.

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
- Fake resolver policy adapters and fake AdGuard control API servers.
- Fake device discovery sources.
- Fake AdGuard clients API servers.
- Fake adapter capability reports.
- Fake optimization appliers.
- Capturing Home Automation publishers.
- Fake clock for retention and schedules.
- Fake snapshot service for monitoring baseline/current comparisons.
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
