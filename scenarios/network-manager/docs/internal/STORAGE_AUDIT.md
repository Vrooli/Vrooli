# Network Manager Storage Architecture Audit

## Last Updated

2026-06-24

## Current Pattern

- [x] Per-domain schema files for implemented storage (`api/internal/snapshot/schema.sql`, `api/internal/adapters/schema.sql`, `api/internal/resolver/schema.sql`, `api/internal/policy/schema.sql`, `api/internal/inventory/schema.sql`, `api/internal/privacy/schema.sql`, `api/internal/optimization/schema.sql`, `api/internal/homeintegration/schema.sql`, `api/internal/monitoring/schema.sql`)
- [x] Per-domain schemas for all API/CLI-backed P0 domains
- [ ] Centralized product schema
- [ ] Resource-applied schema

## Migration Strategy

- [x] Greenfield: idempotent `Schema()` providers only
- [ ] Brownfield: versioned migrations
- Current data state: dev-only/local baseline snapshots plus local adapter/resolver configuration, policy profile intent, policy preview/approval/rollback evidence for approved AdGuard global rule/protection changes, AdGuard-derived device inventory/group labels, privacy retention/visibility defaults, optimization experiment ledgers, Home Automation action/event audit records, and monitoring schedule/run/alert evidence

## Architecture Status

- [x] Snapshot domain owns `network_snapshots` and `snapshot_probe_results`
- [x] Snapshot service uses a repository interface rather than handler SQL
- [x] Snapshot schema is idempotent
- [x] Adapter domain owns `adapter_capabilities` and `adapter_platform_summaries`
- [x] Resolver domain owns `resolver_backends` and `resolver_upstreams`
- [x] Policy domain owns `policy_profiles`, `policy_change_plans`, `approval_records`, and `rollback_records`
- [x] Inventory domain owns `devices` and `device_groups`
- [x] Privacy domain owns `retention_settings`, `visibility_settings`, and `privacy_sweep_records`
- [x] Optimization domain owns `optimization_runs`, `optimization_candidates`, `optimization_approval_records`, and `optimization_rollback_records`
- [x] Home Integration domain owns `home_action_invocations` and `home_events`
- [x] Monitoring domain owns `monitoring_schedules`, `monitoring_runs`, and `monitoring_alerts`
- [x] Implemented storage-backed domains use repository interfaces rather than handler SQL

## Issues Found

1. Inventory production discovery can import AdGuard Home configured and auto client evidence from `/control/clients` through the governed resolver backend and credential-authority seam. It intentionally does not persist query-level DNS logs and keeps IP-only evidence low confidence.
2. Policy live resolver writes are supported only for global AdGuard user-defined rules and global protection pause/resume after approval and rollback-handle capture. Client/group-specific household profile enforcement remains advisory until AdGuard client identity mapping is explicitly approved for policy mutation semantics.
3. Optimization persistent applies use the AdGuard policy rollback adapter for the supported global DNS filtering candidate and store optimization approval/rollback evidence in the optimization ledger. Non-AdGuard/router/client-specific candidates still return `manual_required` until their adapters prove safe mutation and rollback.
4. Home Automation write actions intentionally return `approval_required` or `manual_required` until a governed Home Automation publisher and network adapter path can safely mutate resolver/router state.
5. Privacy retention sweep currently prunes expired non-baseline snapshots and records no-op notes for query logs and optimization ledgers until query log tables exist.
6. Monitoring is storage-backed and operator-triggered; autonomous background scheduling is deferred to a future scheduler that can consume `monitoring_schedules`.
7. `storage-manager validate scenario network-manager` has previously passed with known false-positive `DIRECT_SQL_IN_HANDLERS` warnings on endpoint descriptor lines rather than SQL execution sites. It also returned unexpected EOF/deadline errors during later Phase 7-9 validation (`knw-1782239066919260424`, `knw-1782240251178806531`) and returned `deadline_exceeded` again during the 2026-06-24 monitoring slice. Rerun storage-manager after each slice because both validator availability and warning line numbers can change.

## Cross-References

- `storage-manager validate scenario network-manager`
- `api/internal/snapshot/schema.sql`
- `api/internal/adapters/schema.sql`
- `api/internal/resolver/schema.sql`
- `api/internal/policy/schema.sql`
- `api/internal/inventory/schema.sql`
- `api/internal/privacy/schema.sql`
- `api/internal/optimization/schema.sql`
- `api/internal/homeintegration/schema.sql`
- `api/internal/monitoring/schema.sql`
- `docs/concepts/DATA.md`
