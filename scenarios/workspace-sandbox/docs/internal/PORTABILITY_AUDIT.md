# Workspace Sandbox Cross-Platform Readiness Audit

## Last Updated
2026-04-28

## Target Tiers
- [x] Tier 1 Local Stack — works (default).
- [x] Tier 2 Desktop (Electron-style bundles) — feasible as far as
  storage is concerned. Overlayfs is still Linux-only; on macOS/Windows
  the copy fallback driver remains the path of least resistance.
- [x] Tier 4 Cloud/SaaS — embedded SQLite is appropriate for a
  single-host scenario; cloud deploys would scale by running multiple
  instances each owning their DB file (or by mounting a persistent
  volume per pod).
- [ ] Tier 3 Mobile / Tier 5 Enterprise — out of scope for this scenario
  given its overlayfs + bwrap requirements.

## Environment Variable Status
| Variable | Usage | Fallback? | Desktop-Ready? |
|----------|-------|-----------|----------------|
| `SQLITE_PATH` | DB file override | Yes — derived from `api-core/storage` `ClassData` when unset. | Yes |
| `VROOLI_ROOT` | Source-tree resolution for some helpers | Yes — falls back to `VROOLI_SOURCE_ROOT` or cwd-based detection | Yes |
| `API_PORT`, `UI_PORT`, `WS_PORT` | Lifecycle-injected; defaulted in code | Yes | Yes |
| `WORKSPACE_SANDBOX_*` | Local feature flags / driver knobs | Yes — defaulted | Yes |

No `POSTGRES_*` or `DATABASE_URL` references remain.

## Resource Dependencies
| Resource | Fitness (Desktop) | Strategy | Alternative | Reasoning |
|----------|-------------------|----------|-------------|-----------|
| SQLite (embedded, modernc) | 0.95 | Embedded | n/a | Pure-Go driver; CGO_ENABLED=0 builds clean. |
| `postgres` | n/a | Removed | SQLite | Greenfield cutover; no compatibility shims. |
| `redis` | n/a | Not used | n/a | |
| Overlayfs / fuse-overlayfs | 0.6 | Conditional + copy fallback | copy driver | The driver layer auto-selects by host capability. |

## Build Status
- [x] `go build ./...` succeeds.
- [x] `CGO_ENABLED=0 go build ./...` succeeds (modernc.org/sqlite is
  pure Go).
- [x] No `github.com/mattn/go-sqlite3` import; no CGO directives.

## Network Status
- [x] CORS configurable (`WORKSPACE_SANDBOX_CORS_ORIGINS`).
- [x] Ports configurable via `API_PORT`/`UI_PORT`/`WS_PORT`.
- [x] No fixed `:8080`/`:5432` references.

## Issues Found
- None outstanding for storage portability. Driver-layer portability
  beyond Linux remains a separate concern owned by
  `api/internal/driver`.

## Required Changes for Tier 2 (Desktop)
- The driver layer must continue to fall back to the copy driver on
  non-Linux hosts (already implemented).

## Required Changes for Tier 3 (Mobile)
- N/A. The scenario depends on host-level filesystem mounts that mobile
  sandboxes do not provide. Mobile deployment is not a target.
