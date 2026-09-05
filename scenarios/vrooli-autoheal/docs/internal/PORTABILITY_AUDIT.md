# vrooli-autoheal Cross-Platform Readiness Audit

## Last Updated
2026-08-13

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
| Linux | Full v1 | Parses apt/dpkg logs plus journal boot/kernel signals when available; dnf/rpm/pacman adapters report their own source status. |
| Windows | Build-verified partial | `platform-go` invokes `Get-WinEvent`, parses typed event records, and the collector persists normalized events with explicit source status. Native Windows execution and permissions remain unqualified here. |
| macOS | Build-verified partial | `platform-go` invokes `log show --style ndjson`, parses typed unified-log records, and the collector persists normalized events with explicit source status. Native macOS execution and permissions remain unqualified here. |
| Other | Unsupported | Source status is explicit and non-fatal. |

## Storage Status
- Runtime database remains SQLite via `api-core/database` and `modernc.org/sqlite`.
- The active database, WAL, SHM, deployment report, and retention receipt are declared through resolver-selected `data`/`state` classes; no new platform-specific path was introduced.
- Offline retention refuses a foreign ambient `VROOLI_STORAGE_NAMESPACE`, so an installed CLI launched from another scenario cannot inspect or prune that scenario's database. Autoheal live and shadow namespaces remain isolated.
- The active database has a 1 GiB working-set budget backed by per-table scheduled age and byte enforcement; the 2026-08-29 compacted measurement was 253,382,656 bytes.
- System events are stored in `system_events` with fingerprint dedupe.
- Source health is stored in `system_event_sources`.
- Default event retention is 30 days.

## Build Status
- Platform-go host-log parsers and the Go loop/API cross-builds pass for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64.
- Platform-go fixture tests cover journald JSONL, macOS NDJSON, and Windows event JSON; the API collector test covers normalized portable ingestion.
- The authoritative Test Genie run is recorded separately; its remaining failures are external provider availability, not parser or collector assertions.

## Required Follow-Ups
1. Record native Windows event-log and macOS unified-log execution evidence on each supported host tier.
2. Add package/software-update history adapters when the corresponding host APIs are available and their ownership contract is approved.
3. Add end-to-end scenario smoke coverage for `/api/v1/system-events` once test-genie has a stable fixture for host log evidence.
