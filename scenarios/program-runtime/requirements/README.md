# Requirements

Requirement modules live here, **one folder per owning domain**, ordered by
the domain read graph in [`../docs/concepts/DOMAINS.md`](../docs/concepts/DOMAINS.md)
so a domain's progress is readable in one place. Every requirement links back
to a PRD operational target via `prd_ref` and carries at least one validation
entry pointing at its proof.

| Module | Domain | Reads |
|---|---|---|
| `01-bindings` | `bindings` | nothing — build first |
| `02-actspace` | `actspace` | `bindings` only; parallel with `sessions` |
| `03-sessions` | `sessions` | `bindings` |
| `04-programs` | `programs` | `bindings`, `sessions` |
| `05-telemetry` | `telemetry` | observes `programs` |
| `06-platform` | — | cross-domain obligations no single domain owns |

Requirement IDs stay `PRT-P<priority>-<n>` and mirror the PRD's `OT-` targets
1:1. They deliberately encode priority rather than domain: the IDs are cited
from outside this scenario — `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`
and [`../docs/spaces/act-space.md`](../docs/spaces/act-space.md) both name
`PRT-P0-007` and `PRT-P0-008` — so they must stay stable when a requirement
moves between modules.

- Statuses are earned, not asserted: auto-sync updates them from
  `[REQ:ID]`-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a `ref` to the test file plus the `[REQ:ID]` tag) as behavior lands.
- Validate with `business-health validate scenario <scenario>`; inspect
  traceability with `business-health matrix show <scenario>`.
