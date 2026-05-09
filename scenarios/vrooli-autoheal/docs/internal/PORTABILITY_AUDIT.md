# vrooli-autoheal Cross-Platform Readiness Audit

## Last Updated
2026-05-09

## Target Tiers
- [x] Tier 1 Local Stack
- [~] Tier 2 Desktop
- [~] Tier 3 Mobile
- [~] Tier 4 Cloud/SaaS
- [~] Tier 5 Enterprise

## System Event Timeline Posture

The system-event timeline is persisted in SQLite and uses injected collection
seams. It does not add CGO or external package dependencies.

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | Full v1 | Parses apt/dpkg logs plus journal boot/kernel signals when available. |
| Windows | Partial | API reports unsupported source status for event-log ingestion in this build. |
| macOS | Partial | API reports unsupported source status for unified-log/software-update ingestion in this build. |
| Other | Unsupported | Source status is explicit and non-fatal. |

## Storage Status
- Runtime database remains SQLite via `api-core/database` and `modernc.org/sqlite`.
- System events are stored in `system_events` with fingerprint dedupe.
- Source health is stored in `system_event_sources`.
- Default event retention is 30 days.

## Build Status
- `CGO_ENABLED=0 go test ./internal/systemevents ./internal/persistence ./internal/handlers` passes.
- CLI command tests pass with the new top-level `timeline` command.
- UI type-check and targeted Timeline/App tests pass.

## Required Follow-Ups
1. Implement Windows event-log ingestion with `wevtutil` behind the existing `systemevents.Collector` seam.
2. Implement macOS `softwareupdate --history` / `log show` ingestion behind the same seam.
3. Add end-to-end scenario smoke coverage for `/api/v1/system-events` once test-genie has a stable fixture for host log evidence.

