# Testing — Network Manager

## TL;DR — the canonical examples

The first implementation slice should test a health snapshot with fake probes and tag it `[REQ:NM-P0-001]`. The second should test AdGuard Home adapter status with a fake client and tag it `[REQ:NM-P0-002]`.

## API testing

API tests should cover domain services first, then handlers. Required P0 test themes:

- Snapshot report shape, partial probe results, confidence flags.
- Resolver adapter status and failure mapping.
- Policy preview/apply/rollback records.
- Device identity reconciliation and ambiguity notes.
- Optimization run state transitions and approval gating.
- Capability reports for supported and unsupported actions.
- Privacy retention defaults.

## UI testing

UI tests should cover:

- Loading, empty, error, and partial-result states.
- Accessible tables and status summaries.
- Confirmation flows for risky changes.
- Before/after comparison readability.
- Non-color-only status indicators.

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
- Fake adapter capability reports.
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
