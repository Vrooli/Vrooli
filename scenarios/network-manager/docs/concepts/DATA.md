# Data — Network Manager

## Purpose Of This Document

This document defines Network Manager's data ownership, storage expectations, retention, deletion, import/export, and privacy-relevant behavior.

## Storage Overview

P0 should use local scenario storage from the React/Vite template rather than requiring PostgreSQL. Mutable state should live under Vrooli runtime storage, not the scenario source tree.

Primary stored records:

- health snapshots,
- resolver backend connections and capability reports,
- device inventory records,
- policy profiles and change plans,
- optimization experiment runs,
- approval and rollback records,
- privacy/retention settings,
- exported reports.

## Data Ownership

| Data | Owner Domain | Sensitivity | Notes |
|---|---|---|---|
| Snapshot metrics | snapshot | Medium | Can reveal ISP/network quality and local topology. |
| Resolver config | resolver | Medium | May include upstream choices and backend URLs. |
| Device inventory | inventory | High | Identifies household or office devices. |
| DNS policy | policy | Medium | Reveals access-control intent. |
| DNS query visibility | privacy | High | Default retention must be minimal. |
| Experiment ledger | optimization | Medium | Contains before/after network evidence. |
| Home Automation events | homeintegration | Medium | Should avoid sensitive query details. |

## Schema Map

Implementation should add domain-owned schema files beside domain code. Expected tables or collections:

- `network_snapshots`
- `snapshot_probe_results`
- `resolver_backends`
- `adapter_capabilities`
- `devices`
- `device_groups`
- `policy_profiles`
- `policy_change_plans`
- `optimization_runs`
- `optimization_candidates`
- `approval_records`
- `rollback_records`
- `retention_settings`

## Migrations And Compatibility

Migrations must preserve local operator data. Any schema that stores device identity, query visibility, approvals, or rollback handles needs migration tests before it can be considered production-ready.

## Import / Export

P0 exports should support:

- human-readable health reports,
- before/after optimization reports,
- resolver policy summaries,
- small-office evidence packs once audit mode exists.

Imports are deferred until policy profile sharing or multi-site support exists.

## Retention And Deletion

Default retention should be minimal:

- DNS query-level visibility: disabled or short retention by default.
- Health snapshots: retained long enough for trends and optimization comparison.
- Device inventory: retained while devices remain relevant, with manual delete.
- Audit mode: longer retention only when small-office profile is explicitly selected.

Deleting a device should remove local labels and policy assignment history unless audit mode requires preserving a non-sensitive event reference.

## Privacy Notes

DNS and device metadata can reveal sensitive household or business behavior. The UI must make query visibility explicit, distinguish household and business audit defaults, and never silently expose browsing metadata to unrelated users or scenarios.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../internal/SECURITY.md`](../internal/SECURITY.md)
