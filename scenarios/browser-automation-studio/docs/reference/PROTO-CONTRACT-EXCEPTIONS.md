# Proto Contract Exceptions

This register records the remaining `proto-health` findings that require a versioned contract migration or a validator capability rather than a local cosmetic edit.

The machine-readable companion, [`.vrooli/proto-health-exceptions.json`](../../.vrooli/proto-health-exceptions.json), names each temporarily retained V1 shared message exactly. Proto Health accepts only those explicit `proto.shared_type_misplaced` exceptions; new or unlisted shared-type debt remains a validation error.

| Finding | Scope | Decision | Owner / exit criterion |
| --- | --- | --- | --- |
| `proto.package_mismatch` | Existing `browser_automation_studio.v1` packages and generated Go/TypeScript/Python clients | Retain the established package prefix. Renaming to the validator's preferred prefix changes generated symbols and every consumer import; it must be a separately planned, versioned V2+ wire migration. | BAS contract owner; migrate only with consumer inventory, compatibility plan, regenerated clients, and cross-scenario validation. |
| `proto.shared_type_misplaced` / `proto.cross_domain_import` | Action, workflow, geometry, timeline, and evidence models | Retain domain-owned models for V1. Moving them changes proto type identities and consumers; new cross-domain primitives belong in `v1/shared`, while existing V1 types move only during the same versioned migration. | BAS contract owner; no new shared types outside `v1/shared`. |
| `proto.domain_mismatch` | Schema domains without one-to-one HTTP handler packages | The proto layout describes product contract domains, while API handlers are organized by execution modules. A directory-name mirror is not an ownership invariant. | BAS architecture owner; revisit if Connect service modules are introduced. |
| `proto.missing_health_proto` | HTTP lifecycle `/health` endpoint | Health is lifecycle-owned HTTP infrastructure, not an application RPC. Do not add an unused public proto merely to satisfy this advisory. | BAS operations owner; add a proto only when a genuine programmatic health consumer needs one. |
| `proto.possibly_unused` | Driver/UI/CLI payloads not reachable from served Connect RPCs | These schemas are intentionally consumed by the API-driver protocol and generated SDKs, which the current reachability check cannot see. | Proto-health/code-facts integration; clear when it can prove non-RPC consumers. |
| `proto.code_facts_unavailable` | Source adoption proof | A provider timeout is external evidence degradation, not a BAS contract defect. | code-facts service owner; rerun when healthy. |

## Stability policy

Contract files whose declared services are not discovered as implemented Connect services are marked `beta`, not `stable`. Stable is reserved for a proven public implementation with a compatibility commitment.

## Generated-artifact policy

Every schema change uses the repository proto generator and must leave the generation manifest fresh. The Playwright driver distribution is rebuilt from its source as part of its own release checks.
