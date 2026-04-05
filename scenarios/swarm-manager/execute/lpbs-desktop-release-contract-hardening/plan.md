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
3. **Release notes not served**: `download_assets.release_notes` exists but the manifest endpoint ignores it.
4. **No channel discovery**: Desktop apps must hardcode channel names; no API to enumerate available channels/versions.
5. **No current-version semantics**: All linked artifacts are equally valid; no explicit "current" vs "superseded" distinction at the LPBS level.
6. **Update API key uses non-constant-time comparison**: `update_handlers.go` compares keys with Go string equality, not `subtle.ConstantTimeCompare`.
7. **No post-release verification endpoint**: After a release, there's no way to programmatically verify that the update endpoint serves the expected version.
8. **Update policy defaults are implicit**: No documented or configurable default policy for auto-update behavior (check interval, mandatory vs optional, rollback window).

## 3. Scope

### In Scope
- LPBS API changes (new endpoints, schema migrations, handler updates)
- scenario-to-desktop upload flow changes (pass release_id, channel, release_notes)
- Update endpoint enhancements (release notes in manifest, channel discovery)
- Post-release verification endpoint
- Update policy defaults (schema + API, seam for per-app override)
- Relevant skill updates (landing-page-deploy-setup, landing-page-desktop-upload)
- Test coverage for all new and modified endpoints

### Out of Scope
- deployment-manager release record tables (separate backlog item: Action 1 from research)
- Channel promotion logic in DM (separate backlog item: Action 5)
- Release history API in DM (separate backlog item: Action 6)
- DM orchestrator integration (separate backlog item: Action 7)
- Delta/patch updates
- Code signing infrastructure
- Bearer token / OAuth for update endpoints (future)
- UI changes

## 4. Current Technical Context

### Key Files — LPBS API
| File | Role |
|------|------|
| `scenarios/landing-page-business-suite/api/update_handlers.go` | Update endpoint: manifest serving, binary redirects, API key gating |
| `scenarios/landing-page-business-suite/api/update_handlers_test.go` | 13 test cases for update flows |
| `scenarios/landing-page-business-suite/api/download_service.go` | Download app/asset CRUD, `UpsertAsset` with hardcoded variant_key |
| `scenarios/landing-page-business-suite/api/download_service_test.go` | Asset/app service tests |
| `scenarios/landing-page-business-suite/api/download_hosting.go` | S3 artifact management, presign flows |
| `scenarios/landing-page-business-suite/api/download_hosting_test.go` | Artifact lifecycle tests |
| `scenarios/landing-page-business-suite/api/routes.go` | Route registration |
| `scenarios/landing-page-business-suite/api/main.go:742-806` | Table DDL (download_apps, download_assets, download_artifacts) |

### Key Files — scenario-to-desktop
| File | Role |
|------|------|
| `scenarios/scenario-to-desktop/` | Desktop app build + deploy pipeline |

### Database Schema (current)
- `download_apps` — app registry (app_key, update_api_key, metadata)
- `download_assets` — per-platform asset binding (platform, variant_key, artifact_id, release_version, release_notes)
- `download_artifacts` — S3 object references (sha256, sha512, size_bytes, original_filename)
- `download_storage_settings` — S3 bucket config

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
2. `download_artifacts` has a `release_id` TEXT column for DM correlation
3. Update manifests include `releaseNotes` when available
4. `GET /api/v1/updates/{app_key}/channels` returns available channels with latest version per platform
5. `GET /api/v1/updates/{app_key}/verify` confirms the update endpoint serves an expected version+sha512
6. `download_apps` has an `update_policy` JSONB column with configurable defaults (check_interval, mandatory, rollback_window)
7. Update API key comparison uses `crypto/subtle.ConstantTimeCompare`
8. `download_assets` has an `is_current` BOOLEAN column (default true) for current-version selection
9. All new/modified endpoints have test coverage
10. Skills updated to document new capabilities

## 6. Implementation Strategy

### Phase 1: Schema & Core Fixes (foundation)
1. **Migration: Add `release_id` to `download_artifacts`** — TEXT column + index
2. **Migration: Add `is_current` to `download_assets`** — BOOLEAN DEFAULT true + index
3. **Migration: Add `update_policy` to `download_apps`** — JSONB column with sensible defaults
4. **Fix `UpsertAsset` hardcoded variant_key** — parameterize from caller
5. **Fix API key timing attack** — use `crypto/subtle.ConstantTimeCompare`

### Phase 2: Endpoint Enhancements
6. **Add release notes to manifest** — extend `buildElectronManifest` to include `releaseNotes` field from `download_assets.release_notes`
7. **Add channel discovery endpoint** — `GET /api/v1/updates/{app_key}/channels` returning `[{channel, platform, version, updated_at}]`
8. **Add post-release verification endpoint** — `GET /api/v1/updates/{app_key}/verify?channel=stable&platform=linux&expected_version=1.2.3`
9. **Update policy defaults endpoint** — `GET/PUT /api/v1/admin/download-apps/{app_key}/update-policy`

### Phase 3: S2D Integration
10. **Extend S2D upload flow** — pass `release_id` in artifact commit, pass `channel` → `variant_key` in asset apply
11. **Update artifact commit handler** — accept and store `release_id`

### Phase 4: Skills & Verification
12. **Update `landing-page-desktop-upload` skill** — document release_id flow, channel handling, verification steps
13. **Update `landing-page-deploy-setup` skill** — document update_policy configuration gate

### Final: Cleanup & Verification
- Run `go build ./...` and fix ALL errors, even pre-existing
- Run `golangci-lint run` and fix ALL warnings in modified files
- Run `go test ./... -timeout 300s` and fix any failures
- `vrooli scenario restart landing-page-business-suite`
- Verify health: `curl -s http://localhost:<port>/health`

## 7. Contract Decisions

<!-- Pending workshop decisions — will be populated from round answers -->

### Settled (from research dependency)
- **Release ID ownership**: deployment-manager owns the UUID; LPBS stores it as a correlation key
- **Channel ordering**: nightly → beta → stable; stable maps to variant_key "default"
- **Approval model**: commit-scoped, no channel dimension
- **Superseded status**: application-level logic, not DB trigger

### Pending
- Update policy default values (check interval, mandatory flag, rollback window)
- Channel discovery endpoint: public or gated?
- Verification endpoint: admin-only or public?
- is_current management: who sets it and when?
- Manifest format extensions beyond releaseNotes

## 8. Testing Plan

| Area | Test Type | Coverage |
|------|-----------|----------|
| UpsertAsset variant_key parameterization | Unit + integration | Verify non-default variant_keys insert correctly |
| release_id on artifact commit | Unit + integration | Verify column stored and queryable |
| is_current flag management | Integration | Verify only one asset per (app_key, platform, variant_key) is current |
| Release notes in manifest | Unit | Verify YAML output includes releaseNotes |
| Channel discovery endpoint | Unit + integration | Verify correct aggregation across platforms/variants |
| Verification endpoint | Unit + integration | Verify match/mismatch responses |
| Update policy CRUD | Unit + integration | Verify default application and per-app override |
| API key constant-time comparison | Unit | Verify timing-safe comparison works for match/mismatch |
| Non-default channel update flow | Integration | End-to-end: beta channel artifact → manifest serve |

## 9. Rollout/Validation Checklist

- [ ] All migrations apply cleanly on fresh DB
- [ ] Existing data is preserved (migrations are additive)
- [ ] Update endpoint serves manifests for default and non-default channels
- [ ] Release notes appear in manifest YAML when populated
- [ ] Channel discovery returns correct data for multi-platform apps
- [ ] Verification endpoint correctly reports match/mismatch
- [ ] S2D upload flow passes release_id through to LPBS
- [ ] All tests pass with `go test ./... -timeout 300s`
- [ ] Scenario restarts cleanly

## 10. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Migration breaks existing data | High | All migrations are ALTER TABLE ADD COLUMN with defaults; no destructive changes |
| S2D upload flow breaks if LPBS expects new fields | Medium | New fields are optional; old S2D versions continue to work without release_id |
| Channel discovery performance on large datasets | Low | Index on (app_key, variant_key, platform) already exists |
| Update policy schema evolution | Low | JSONB column allows additive changes without migrations |

## 11. Non-goals / Prohibited Patterns

- **No compatibility shims**: This is greenfield work. No legacy wrappers or dead code.
- **No DM schema changes**: Release records live in DM (separate backlog items).
- **No UI changes**: API-only; UI can consume these endpoints later.
- **No delta updates**: Full installer replacement only for now.
- **No code signing**: Checksum integrity only; signing is a separate initiative.

## 12. Definition of Done

1. All schema migrations apply cleanly
2. `UpsertAsset` accepts parameterized variant_key
3. `download_artifacts` stores release_id
4. Update manifests include release notes
5. Channel discovery endpoint works
6. Post-release verification endpoint works
7. Update policy defaults are configurable per-app
8. API key comparison is timing-safe
9. All tests pass
10. Scenario restarts and is healthy
11. Skills updated with new capabilities
