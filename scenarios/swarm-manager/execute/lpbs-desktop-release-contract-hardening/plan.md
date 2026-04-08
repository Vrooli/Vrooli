# Implementation Plan: Harden The LPBS Desktop Release Contract

## 1. Purpose

Make the LPBS desktop release contract deterministic, inspectable, and commercially reliable. After this work, generated desktop apps ship a non-invasive auto-update flow by default, releases are traceable end-to-end, and the update endpoint guarantees are strong enough to sell through LPBS.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer seam-discovery-and-enforcement
prompt-manager skill read scenario-to-desktop landing-page-deploy-setup landing-page-desktop-upload
```

**Research dependency (completed):**
```bash
swarm-manager backlog file-get --kind research --name release-record-contract-and-history-model --path conclusion.md
```

## 2. Problem Statement

LPBS has a functional update endpoint and artifact storage layer, but the release contract has gaps that prevent it from being a reliable commercial distribution surface:

1. **Hardcoded variant_key**: `UpsertAsset` hardcodes `'default'` (download_service.go:596), blocking multi-channel releases.
2. **No release_id correlation**: LPBS artifacts cannot be traced back to deployment-manager release records.
3. **Release notes not served**: `download_assets.release_notes` exists but `buildElectronManifest()` (update_handlers.go:58-69) ignores it.
4. **No channel discovery**: Desktop apps must hardcode channel names; no API to enumerate available channels/versions.
5. **No current-version semantics at artifact level**: Current-version is determined by `download_assets.artifact_id` JOIN, but there's no verification or discovery API for this.
6. **Update API key uses non-constant-time comparison**: `update_handlers.go` compares keys with Go string equality, not `subtle.ConstantTimeCompare`.
7. **No post-release verification endpoint**: After a release, there's no way to programmatically verify that the update endpoint serves the expected version.
8. **Update policy defaults are implicit**: No documented or configurable default policy for auto-update behavior.

## 3. Scope

### In Scope
- LPBS API changes (new endpoints, schema additions, handler updates)
- scenario-to-desktop upload flow changes (pass release_id, channel, release_notes)
- Update endpoint enhancements (release notes in manifest, channel discovery)
- Post-release verification endpoint (lightweight + optional deep S3 check)
- Update policy defaults (schema + API, seam for per-app override)
- Constant-time API key comparison fix
- Auth middleware extraction (`requireUpdateAPIKey`) for update-family endpoints
- Relevant skill updates (landing-page-deploy-setup, landing-page-desktop-upload)
- Test coverage for all new and modified endpoints

### Out of Scope
- deployment-manager release record tables (separate initiative item)
- Channel promotion logic in DM (separate initiative item)
- Release history API in DM (separate initiative item)
- DM orchestrator integration (separate initiative item)
- Delta/patch updates
- Code signing infrastructure
- Bearer token / OAuth for update endpoints (future)
- UI changes
- **is_current column migration** — already computed dynamically via LEFT JOIN in `ListArtifactsByApp()` (download_hosting.go:981) and `GetCurrentArtifactByFilename()` (download_hosting.go:1016-1039). No schema change needed.

## 4. Current Technical Context

### Key Files — LPBS API
| File | Role |
|------|------|
| `scenarios/landing-page-business-suite/api/update_handlers.go` | Update endpoint: manifest serving (lines 58-69), binary redirects, API key gating (lines 103-109), `channelToVariantKey()` (lines 50-55) |
| `scenarios/landing-page-business-suite/api/update_handlers_test.go` | 13 test cases for update flows |
| `scenarios/landing-page-business-suite/api/download_service.go` | Download app/asset CRUD, `UpsertAsset` with hardcoded variant_key (line 596), `GetAssetByVariant()` (lines 529-549) |
| `scenarios/landing-page-business-suite/api/download_service_test.go` | Asset/app service tests |
| `scenarios/landing-page-business-suite/api/download_hosting.go` | S3 artifact management, presign flows, `ListArtifactsByApp()` with computed is_current (line 981), `GetCurrentArtifactByFilename()` (lines 1016-1039) |
| `scenarios/landing-page-business-suite/api/download_hosting_test.go` | Artifact lifecycle tests |
| `scenarios/landing-page-business-suite/api/routes.go` | Route registration, `handleAdminSetArtifactAsCurrent` at POST `/api/v1/admin/download-assets/set-current` (line 126) |
| `scenarios/landing-page-business-suite/api/main.go:742-806` | Table DDL with IF NOT EXISTS pattern |

### Key Files — scenario-to-desktop
| File | Role |
|------|------|
| `scenarios/scenario-to-desktop/` | Desktop app build + deploy pipeline |

### Database Schema (current)
- `download_apps` — app registry (app_key, update_api_key, metadata)
- `download_assets` — per-platform asset binding (platform, variant_key, artifact_id, release_version, release_notes). Unique on (bundle_key, app_key, platform, variant_key).
- `download_artifacts` — S3 object references (sha256, sha512, size_bytes, original_filename, platform, release_version)
- `download_storage_settings` — S3 bucket config

### is_current Architecture
The "current" artifact for a given slot is determined by `download_assets.artifact_id` FK join, NOT by a stored boolean column. This is already implemented:
- `ListArtifactsByApp()` computes `is_current` via `CASE WHEN da.artifact_id = a.id THEN true ELSE false END`
- `GetCurrentArtifactByFilename()` joins download_artifacts with download_assets on artifact_id
- `handleAdminSetArtifactAsCurrent` updates `download_assets.artifact_id` to promote an artifact
- No schema change needed — this is application-level logic matching the research conclusion

### Research Findings (dependency: release-record-contract-and-history-model)
The completed research defines:
- UUID-based release records owned by deployment-manager
- Channel ordering: nightly → beta → stable (variant_key mapping: stable → "default")
- `release_id` correlation column for LPBS `download_artifacts`
- Parameterized variant_key in `UpsertAsset`
- Commit-scoped approvals (no channel dimension)
- Superseded status via application logic

## 5. Target End State

After implementation:
1. `UpsertAsset` accepts `variant_key` as a parameter (no hardcoded default)
2. `download_artifacts` has a `release_id` TEXT column for DM correlation (added in IF NOT EXISTS DDL)
3. `download_apps` has an `update_policy` JSONB column with defaults `{"check_interval_hours": 4, "update_mode": "optional", "allow_downgrade": false}`
4. Update manifests include `releaseNotes` as plain text passthrough when `download_assets.release_notes` is non-empty; field omitted when empty
5. `GET /api/v1/updates/{app_key}/channels` returns available channels with latest version per platform, gated by same API-key logic as update endpoint
6. `GET /api/v1/updates/{app_key}/verify?channel=X&platform=Y&expected_version=Z&deep=false` confirms update endpoint correctness — lightweight by default, optional deep S3 check with `deep=true`
7. Update API key comparison uses `crypto/subtle.ConstantTimeCompare` via extracted `requireUpdateAPIKey` middleware
8. `requireUpdateAPIKey` middleware shared across update file, channel discovery, and verification endpoints
9. `GET/PUT /api/v1/admin/download-apps/{app_key}/update-policy` for per-app policy management
10. All new/modified endpoints have test coverage
11. Skills updated to document new capabilities

## 6. Implementation Strategy

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

### Phase 1: Schema & Core Fixes (foundation)
1. **Add `release_id` TEXT column to `download_artifacts`** — Add `ALTER TABLE ... ADD COLUMN IF NOT EXISTS release_id TEXT` to the IF NOT EXISTS DDL block in main.go. Add index: `CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_id ON download_artifacts(release_id)`. Nullable, no default.
2. **Add `update_policy` JSONB column to `download_apps`** — Add `ALTER TABLE ... ADD COLUMN IF NOT EXISTS update_policy JSONB NOT NULL DEFAULT '{"check_interval_hours": 4, "update_mode": "optional", "allow_downgrade": false}'::jsonb` to DDL. This provides conservative defaults per the settled policy: 4-hour check interval, optional updates, no downgrade.
3. **Fix `UpsertAsset` hardcoded variant_key** — Add `VariantKey` field to `DownloadAsset` struct, parameterize in the INSERT/upsert statement. Callers pass the variant_key instead of relying on the hardcoded `'default'`.
4. **Extract `requireUpdateAPIKey` middleware** — Create a new middleware function that loads the app by app_key, checks if `update_api_key` is set, and if so validates the `X-Update-Key` header using `crypto/subtle.ConstantTimeCompare`. Apply this middleware to the existing update file handler, replacing the inline comparison. This middleware will also be used by the new endpoints in Phase 2.

### Phase 2: Endpoint Enhancements
5. **Add release notes to manifest** — Extend `buildElectronManifest()` to include `releaseNotes` field from `download_assets.release_notes` as plain text passthrough. Omit the field entirely when notes are empty (no `releaseNotesFormat` field needed since it's plain text).
6. **Add channel discovery endpoint** — `GET /api/v1/updates/{app_key}/channels`. Gated by `requireUpdateAPIKey` middleware (same behavior as update endpoint: public if no key set, gated if set). Returns `[{channel, platform, version, updated_at}]` aggregated from download_assets joined with download_artifacts.
7. **Add post-release verification endpoint** — `GET /api/v1/updates/{app_key}/verify?channel=X&platform=Y&expected_version=Z&deep=false`. Gated by `requireUpdateAPIKey`. Default (lightweight): checks manifest version+sha512 match expected. With `deep=true`: also HEADs the S3 object and tests presign generation. Response: `{match: bool, actual_version: string, actual_sha512: string, artifact_accessible?: bool, presign_valid?: bool}`. The `artifact_accessible` and `presign_valid` fields are only present when `deep=true`.
8. **Update policy CRUD endpoint** — `GET/PUT /api/v1/admin/download-apps/{app_key}/update-policy` (requireAdmin). GET returns the current `update_policy` JSONB for the app (defaults applied if column is at default). PUT accepts and validates `{check_interval_hours: int, update_mode: "optional"|"recommended"|"mandatory", allow_downgrade: bool}` and writes to the JSONB column.

### Phase 3: S2D Integration
9. **Extend S2D upload flow** — Pass `release_id` in artifact commit request body. Pass `channel` → `variant_key` mapping in asset apply step.
10. **Update artifact commit handler** — Accept and store `release_id` in download_artifacts when provided.

### Phase 4: Skills & Verification
11. **Update `landing-page-desktop-upload` skill** — Document release_id flow, channel handling, verification steps.
12. **Update `landing-page-deploy-setup` skill** — Document update_policy configuration gate.

### Final: Cleanup & Verification
- Run `go build ./...` and fix ALL errors, even pre-existing
- Run `gofumpt -w .` to format
- Run `golangci-lint run` and fix ALL warnings in modified files
- Run `go test ./... -timeout 300s` and fix any failures
- `vrooli scenario restart landing-page-business-suite`
- Verify health: `curl -s http://localhost:<port>/health`

## 7. Contract Decisions

### Settled (from research dependency)
- **Release ID ownership**: deployment-manager owns the UUID; LPBS stores it as a correlation key
- **Channel ordering**: nightly → beta → stable; stable maps to variant_key "default"
- **Approval model**: commit-scoped, no channel dimension
- **Superseded status**: application-level logic, not DB trigger

### Settled (from round 1)
- **Update policy defaults**: Conservative — 4h check interval, optional updates, no rollback window (d1→A)
- **Channel discovery gating**: Same as update endpoint — public if no update_api_key, gated if set (d2→A)
- **is_current strategy**: Keep existing computed JOIN approach — no schema change (d3→A, corrected in round 2)
- **Verification endpoint**: Combined lightweight + optional deep S3 check (d4→other, resolved round 2)

### Settled (from round 2)
- **Update policy JSONB schema**: Minimal — `{check_interval_hours: int, update_mode: string, allow_downgrade: bool}` (d1→A). Additional fields like `mandatory_after_days`, `rollback_window_hours`, `min_version` can be added later via JSONB without migration.
- **Release notes format**: Plain text passthrough — serve `download_assets.release_notes` as-is in the `releaseNotes` YAML field; omit when empty (d2→A). LPBS is format-agnostic; the content producer controls the format.
- **Migration strategy**: Add both columns in the existing IF NOT EXISTS DDL block in main.go (d3→A). No numbered migration system needed for 2 additive columns.
- **Auth middleware extraction**: Extract `requireUpdateAPIKey(appKey)` middleware shared across update file, channel discovery, and verification endpoints (d4→A). Single place for constant-time API key check, header name, and error response.

## 8. Testing Plan

| Area | Test Type | Coverage |
|------|-----------|----------|
| UpsertAsset variant_key parameterization | Unit + integration | Verify non-default variant_keys insert correctly |
| release_id on artifact commit | Unit + integration | Verify column stored and queryable |
| Release notes in manifest | Unit | Verify YAML output includes releaseNotes when present, omits when empty |
| Channel discovery endpoint | Unit + integration | Verify correct aggregation across platforms/variants; verify API key gating |
| Verification endpoint (lightweight) | Unit + integration | Verify match/mismatch responses for version+sha512 |
| Verification endpoint (deep) | Unit + integration | Verify S3 HEAD and presign check when deep=true |
| Update policy CRUD | Unit + integration | Verify default application, per-app override, and GET/PUT round-trip |
| API key constant-time comparison | Unit | Verify timing-safe comparison works for match/mismatch |
| requireUpdateAPIKey middleware | Unit | Verify gating behavior: no key set → allow, key set + correct → allow, key set + wrong → 401 |
| Non-default channel update flow | Integration | End-to-end: beta channel artifact → manifest serve |

## 9. Rollout/Validation Checklist

- [ ] Schema additions apply cleanly on fresh DB (IF NOT EXISTS / ADD COLUMN IF NOT EXISTS)
- [ ] Existing data is preserved (all additions are nullable or have defaults)
- [ ] Update endpoint serves manifests for default and non-default channels
- [ ] Release notes appear in manifest YAML when populated, omitted when empty
- [ ] Channel discovery returns correct data for multi-platform apps
- [ ] Verification endpoint correctly reports match/mismatch (lightweight mode)
- [ ] Verification endpoint correctly reports S3 accessibility (deep mode)
- [ ] S2D upload flow passes release_id through to LPBS
- [ ] Update policy endpoint returns defaults for unconfigured apps
- [ ] requireUpdateAPIKey middleware correctly gates all update-family endpoints
- [ ] All tests pass with `go test ./... -timeout 300s`
- [ ] Scenario restarts cleanly

## 10. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| DDL additions break existing data | High | All additions use ADD COLUMN IF NOT EXISTS, nullable columns, or columns with defaults |
| S2D upload flow breaks if LPBS expects new fields | Medium | New fields are optional; old S2D versions continue to work without release_id |
| Channel discovery performance on large datasets | Low | Index on (app_key, variant_key, platform) already exists via unique constraint |
| Update policy schema evolution | Low | JSONB column allows additive changes without migrations |
| Deep verification adds S3 latency | Low | Deep mode is opt-in via `deep=true`; default is lightweight check only |
| requireUpdateAPIKey middleware changes auth behavior | Medium | Middleware replicates exact existing inline logic with constant-time fix; comprehensive test coverage |

## 11. Non-goals / Prohibited Patterns

- **No compatibility shims**: This is greenfield work. No legacy wrappers or dead code.
- **No DM schema changes**: Release records live in DM (separate backlog items).
- **No UI changes**: API-only; UI can consume these endpoints later.
- **No delta updates**: Full installer replacement only for now.
- **No code signing**: Checksum integrity only; signing is a separate initiative.
- **No is_current column**: Current-version is already computed via JOIN — do not add a stored flag.

## 12. Definition of Done

1. DDL additions apply cleanly (release_id on artifacts, update_policy on apps) via IF NOT EXISTS pattern
2. `UpsertAsset` accepts parameterized variant_key
3. `download_artifacts` stores release_id when provided
4. Update manifests include plain text release notes when available, omit when empty
5. Channel discovery endpoint works with correct API-key gating
6. Post-release verification endpoint works (lightweight + deep modes)
7. Update policy defaults are configurable per-app via admin endpoint with minimal schema
8. API key comparison is timing-safe via extracted `requireUpdateAPIKey` middleware
9. All tests pass
10. Scenario restarts and is healthy
11. Skills updated with new capabilities
