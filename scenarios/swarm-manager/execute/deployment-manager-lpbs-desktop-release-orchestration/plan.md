# Implementation Plan: Deployment Manager Owns LPBS Desktop Release Orchestration

## 1. Purpose

Make `deployment-manager` the authoritative control plane for LPBS desktop releases. Today the real release flow (cloud health → LPBS readiness → build → LPBS publish → update-endpoint verification) is only encoded in the `landing-page-desktop-upload` prompt-manager skill. After this work, an operator drives the entire release from deployment-manager (API + CLI + UI). The skill is kept as an automation client that calls deployment-manager, not as the sequencer of the four-system relay.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer seam-discovery-and-enforcement interoperability-steer decision-boundary-extraction boundary-of-responsibility-enforcement test
prompt-manager skill read scenario-to-desktop landing-page-deploy-setup landing-page-desktop-upload
```

Research dependencies (completed):
```bash
swarm-manager backlog file-get --kind research --name desktop-release-control-plane-audit --path conclusion.md
swarm-manager backlog file-get --kind research --name release-record-contract-and-history-model --path conclusion.md
```

Completed sibling plans (context for conventions and what exists already):
```bash
swarm-manager backlog file-get --kind execute --name deployment-manager-approval-gate-surfaces --path plan.md
swarm-manager backlog file-get --kind execute --name deployment-manager-visual-validation-approval-flow --path plan.md
swarm-manager backlog file-get --kind execute --name lpbs-desktop-release-contract-hardening --path plan.md
```

## 3. Problem Statement

Four systems participate in every LPBS desktop release:

- `deployment-manager` — owns profiles, approvals, published-version records, and runs the `/api/v1/deploy-desktop` orchestrator.
- `scenario-to-desktop` — owns the build-and-publish pipeline; its `deploy` stage uploads artifacts to LPBS via `lpbs_client.go`.
- `landing-page-business-suite` — owns app registry, artifacts, update manifests (now with `release_id`, `variant_key`, `update_policy`, channel discovery, and verify endpoints after contract hardening).
- `scenario-to-cloud` — deploys and health-checks the LPBS runtime itself.

Four gaps keep `deployment-manager` from being the authoritative surface:

1. **LPBS release coordinates are not part of the profile.** The `deploy-desktop` orchestrator never passes `app_key`, `remote_profile`, `channel`, `update_url`, or a `release_id` to `scenario-to-desktop`. `orchestrator_helpers.go:30-44` sends an empty `PublishPipelineRequest` with only `ScenarioName`, `Platforms`, `Publish`, and stage hints. The deploy stage therefore relies on scenario-to-desktop's saved deploy target, which is configured out of band by the skill.
2. **LPBS readiness and cloud health are not checked by DM.** Stages 1–2 of `landing-page-desktop-upload` (cloud-deploy convergence + `deploy-readiness` gate for the app key) run only when a human or skill invokes them. DM assumes the operator did this.
3. **Post-release verification is not owned by DM.** Stage 4 verification (GET `/api/v1/updates/{app_key}/verify?...`) runs only when the skill executes it. DM's `PublishedVersion` record is written without proof that the update endpoint actually serves the new version.
4. **Operator surface is split across systems.** An operator must look at DM for approvals and deployments, LPBS admin for app/asset state, and a running skill for post-release verification. The "is this release live?" answer is not in one place.

Result: prompt-manager skills are load-bearing for production releases. Skill drift or human error breaks the pipeline. There is no single operator-facing surface that shows end-to-end release status.

## 4. Scope

### In scope
- `scenarios/deployment-manager/**` — profile schema additions for LPBS release config, orchestrator stage additions (cloud health, LPBS readiness, post-release verification), release-record persistence, new client(s) to `scenario-to-cloud` and LPBS, new API endpoints for release lifecycle and status, CLI commands, UI page/component updates.
- `scenarios/scenario-to-desktop/**` — extend `DesktopPackagerClient` request/response surface in DM so `app_key`, `remote_profile`, `channel`, `release_id`, and `update_url` reach the deploy stage; wire through to existing `lpbs_client.go` `ReleaseID` and `variant_key` plumbing.
- `scenarios/landing-page-business-suite/**` — only what DM needs to call directly (e.g., verify endpoint, channels endpoint, `deploy-readiness` equivalent) and only if DM cannot get the same signal via `scenario-to-cloud` or S2D.
- `scenarios/prompt-manager/store/skills/packs/core/landing-page-desktop-upload/**` — retarget to call DM as the single orchestrator; remove stage-by-stage CLI sequencing that DM now owns.
- `scenarios/prompt-manager/store/skills/packs/core/scenario-to-desktop/**` — reference doc updates only (new DM flags / release-id pathway).
- Automated tests for every new DM surface (unit for repositories + handlers, integration for the DM → S2D → LPBS path using the existing `DesktopPackagerClient` test pattern).

### Out of scope
- Changes to LPBS release contract not delivered by `lpbs-desktop-release-contract-hardening` (already completed). If a hole appears, create a follow-up backlog item rather than widening this scope.
- Rollback API (Action 3 in research — separate item).
- Auto-deploy after approval (Action 5 in research — separate item).
- Webhook/event notifications from LPBS (Action 4 — separate item).
- Entitlement UX changes in LPBS.
- Anything under `scenarios/scenario-to-cloud/**` other than consuming its existing endpoints via a new DM client.
- Promote API/CLI/UI for moving a release between channels (D9=B). Schema reserves `promoted_from_release_id` and the orchestrator accepts arbitrary `--channel` values, but the promote handler/CLI/UI is a follow-up backlog item.

## 5. Current Technical Context

### deployment-manager (Go, `scenarios/deployment-manager/api/**`)
| File | Role |
|------|------|
| `api/deployments/orchestrator.go` | `DeployDesktop` handler + phased pipeline (`deployLoadProfile`, `deployValidateAndSign`, `deployAssembleManifest`, `deployBuildBinaries`, `deployPackageAndInstall`, `deployFinalizeAndPublish`). Steps produced via `startStep` / `successStep` / `failStep`. Release gate check is inside `deployLoadProfile` (`orchestrator.go:219-239`). |
| `api/deployments/orchestrator_helpers.go` | `publishToLPBS` builds a minimal `PublishPipelineRequest` with no LPBS coords and records `PublishedVersion` rows (lines 30-100). |
| `api/deployments/desktop_client_pipeline.go` | `PublishPipelineRequest` + `PublishDeployConfig` DTOs (lines 176-201). `PublishDeployConfig` already has fields for `TargetName`, `ScenarioName`, `RemoteProfile`, `AppKey`, `UpdateURL` but they are never populated by the orchestrator. |
| `api/deployments/published_versions.go` | `PublishedVersion` model + repo (profile_id, platform, version, git_commit_hash, artifact_id, deployment_id). |
| `api/deployments/approvals_*.go` | Approval gate — authoritative source of "is this commit releasable". |
| `api/profiles/**` | Profile CRUD + `profile_versions`. Profile does not currently carry LPBS release config. |
| `api/server/routes.go` | All HTTP routes. Approval routes conditional on `ApprovalsHandler != nil`; PublishedVersions likewise. |
| `cli/deployment-manager/**` | `deploy-desktop`, `approvals *`, `deployments *`, etc. No release/LPBS verb today. |
| `ui/src/pages/Deployments.tsx`, `ProfileDetail.tsx`, `Approvals.tsx` | Existing operator surfaces. No LPBS release config or post-release status shown. |

### scenario-to-desktop (Go, `scenarios/scenario-to-desktop/api/**`)
| File | Role |
|------|------|
| `api/pipeline/types.go:244-260` | `DeployConfig` already has `ReleaseID` and channel-related plumbing. |
| `api/pipeline/stage_deploy.go:220-230` | Deploy stage forwards `ReleaseID` into `lpbs_client.go`. |
| `api/deploy/lpbs_client.go:84, 321-322, 354` | `ReleaseID` added to commit payload; `variant_key` added to apply payload. |
| `api/deploy/dm_client.go` | Already can call deployment-manager (used for approval gate lookups). |

Note: per round 2 i1, S2D already carries `Config.Version → ReleaseVersion` and `Provenance.GitCommitHash` end-to-end. The S2D-side change in Phase 5 reduces to threading **two new fields** (`ReleaseID`, `Channel`) through existing structs.

### landing-page-business-suite (Go, `scenarios/landing-page-business-suite/api/**`)
| File | Role |
|------|------|
| `api/update_handlers.go` | `/updates/{app_key}/{channel}/{file}`, `/updates/{app_key}/channels`, `/updates/{app_key}/verify` (lightweight and deep). `requireUpdateAPIKey` middleware. |
| `api/download_service.go` | `UpsertAsset` now accepts `variant_key`; `GetCurrentArtifactByFilename` resolves current. |
| `api/download_hosting.go` | S3 artifact lifecycle. |
| CLI `landing-page-business-suite deploy-readiness --profile-tag … --app-key …` | Single-contract readiness gate used by the skill today. |
| `api/server.go` (`requireAdminOrService`) | Existing peer-service auth pattern (LPBS_SERVICE_SECRET) used by presign/commit/apply — the auth model the new `/deploy-readiness` endpoint will adopt (D8=B). |

### prompt-manager skills
| File | Role |
|------|------|
| `store/skills/packs/core/landing-page-desktop-upload/SKILL.md` | 6-stage orchestration (-0 Input discovery → 0 health → 0.5 build prereqs → 1 cloud converge → 2 LPBS readiness → 3 pipeline run → 4 verify). This is the flow to migrate. |
| `store/skills/packs/core/landing-page-deploy-setup/SKILL.md` | 7 idempotent gates (A–G) prepping LPBS control plane. Largely invoked through `deploy-readiness` today. |
| `store/skills/packs/core/scenario-to-desktop/SKILL.md` | CLI reference only. |

### Key contracts already in place (post contract hardening)
- `download_artifacts.release_id TEXT` — LPBS stores DM-owned UUID for correlation.
- `download_apps.update_policy` JSONB default `{"check_interval_hours": 4, "update_mode": "optional", "allow_downgrade": false}`.
- `GET /api/v1/updates/{app_key}/channels` — channel enumeration.
- `GET /api/v1/updates/{app_key}/verify?channel=&platform=&expected_version=&deep=` — post-release verification.
- `scenario-to-desktop` deploy stage accepts `--release-id` and `--channel` on the pipeline run command and passes them to LPBS.

## 6. Target End State

After implementation, a DM operator can run a full desktop release end to end from DM. The externally visible end state is:

1. **Profile carries release coordinates.** A new 1:1 child table `profile_lpbs_release_config` (per D1=B) holds `profile_id` (FK), `lpbs_domain`, `lpbs_remote_profile`, `lpbs_app_key`, `default_channel`, `update_url` (derivable if empty), timestamps. CRUD'd through DM API + UI.
2. **`POST /api/v1/profiles/{id}/releases/start` is the operator-intent entrypoint (per D4=B).** It allocates a release record, then drives the same underlying pipeline. `POST /api/v1/deploy-desktop` remains as the lower-level build-and-bundle entrypoint used by the release-start handler internally. The orchestrator adds steps: *Check release gate* (existing) → *Check LPBS deployment health* (new — via scenario-to-cloud client) → *Check LPBS upload readiness* (new — via a new LPBS `POST /api/v1/deploy-readiness` endpoint called from DM with `requireAdminOrService` auth using `LPBS_SERVICE_SECRET`; per D2=A and D8=B) → *Acquire profile-scoped advisory lock + allocate release_id + persist release row* (new — DM owns it; lock per D10=B) → existing build/package/install → *Publish to LPBS* (extended to pass `AppKey`, `RemoteProfile`, `Channel`, `ReleaseID`, `UpdateURL`) → *Verify update endpoints* (new — calls LPBS `/updates/{app_key}/verify` per platform; hard-fails the release on mismatch per D3=A).
3. **DM persists canonical release records.** Two new tables per the `release-record-contract-and-history-model` research (Finding 7), adopted verbatim per D7=A:
   - `releases` — `id TEXT PK` (UUID), `profile_id`, `deployment_id`, `profile_version`, `git_commit_hash`, `release_version`, `channel` (default `stable`), `status` (`pending|publishing|published|failed|superseded|verify_failed`), `release_notes`, `released_by`, `promoted_from_release_id`, `created_at`, `published_at`, `updated_at`, `UNIQUE(profile_id, git_commit_hash, channel)`.
   - `release_platforms` — `release_id` (FK), `platform`, `status`, `approval_id`, `lpbs_artifact_id`, `published_at`, `error`, `PRIMARY KEY (release_id, platform)`.
   - Migration file numbered `004_add_releases.sql` (next available per research round 3 verification).
   - Records are allocated only by publishing deployments — dry-run and skip-packaging runs skip release-record creation.
   - The orchestrator accepts arbitrary `channel` values (per D9=B); the promote API/CLI/UI is a follow-up item but the schema and orchestrator already support `--channel beta` etc. today.
4. **Operator surfaces.**
   - API: `GET /api/v1/profiles/{id}/releases`, `GET /api/v1/releases/{release_id}`, `POST /api/v1/releases/{release_id}/verify`, `POST /api/v1/profiles/{id}/releases/start`.
   - CLI: `deployment-manager releases list|get|verify|start`. Pattern mirrors the existing `approvals` CLI group.
   - UI (per D5=B): release summary embedded on `ProfileDetail` only — no dedicated `/releases` page in this item. Shows latest release + last N entries, per-platform verification status, channel, `release_id`, re-verify action.
5. **Skill retargeting (per D6=A).** `landing-page-desktop-upload` becomes a thin DM automation wrapper: one `deployment-manager releases start` call + poll + result render. The 6-stage CLI relay is deleted from the skill body; a minimal fallback note documents the DM-unavailable scenario.
6. **Tests.** Go tests cover: new repository methods, new handlers (including error paths), orchestrator step additions via the existing `orchestrator_test.go` pattern, a DM→S2D→LPBS integration test that exercises the extended `PublishDeployConfig`, advisory-lock concurrency guard on simultaneous release starts, dry-run exclusion from release-record creation, and `requireAdminOrService` auth on the new LPBS endpoint. UI tests cover the new release surfaces.
7. **Build + scenario restart rules.** All three scenarios rebuild cleanly (`go build ./...`), lint clean (`golangci-lint run`), formatted (`gofumpt -w .`). User restarts scenarios; plan does not auto-restart.

### Greenfield Constraint
This is greenfield for the orchestration seams. Do NOT add compatibility shims for "call DM directly vs call S2D directly." Old flow is deleted once the new surfaces land — no dual code paths.

## 7. Implementation Strategy

Phased because components span four scenarios and involve schema, HTTP contracts, UI, CLI, and skill retargeting. Phases are executed in order; later phases depend on the earlier ones.

### Phase 1 — Schema + profile-level LPBS release config (deployment-manager)
- **Migration 004_add_releases.sql** (deployment-manager) creates:
  - `profile_lpbs_release_config(profile_id PK FK → profiles, lpbs_domain, lpbs_remote_profile, lpbs_app_key, default_channel, update_url, created_at, updated_at)` — new 1:1 child of `profiles`.
  - `releases(id PK, profile_id FK, deployment_id FK, profile_version INT, git_commit_hash, release_version, channel DEFAULT 'stable', status, release_notes, released_by, promoted_from_release_id, created_at, published_at, updated_at, UNIQUE(profile_id, git_commit_hash, channel))` — per research Finding 7 schema, adopted verbatim (D7=A).
  - `release_platforms(release_id FK, platform, status, approval_id, lpbs_artifact_id, published_at, error, PRIMARY KEY(release_id, platform))`.
  - Indexes: `idx_releases_profile_channel`, `idx_releases_status`, `idx_releases_commit`, `idx_releases_deployment`, `idx_release_platforms_status`.
  - `channel` is a free-text TEXT column with `DEFAULT 'stable'`; no CHECK constraint enumerating channel values, so the orchestrator can accept arbitrary channels per D9=B without a schema change for promote.
- **Migration 005** (LPBS) only if needed — the new `POST /api/v1/deploy-readiness` endpoint is read-through of existing tables (apps, remote profiles, storage config), so no new LPBS schema is anticipated. Confirm during implementation.
- **Repositories**: `api/profiles/lpbs_config_repo.go` (CRUD of config rows) + `api/releases/repo.go` (insert/get/list/update-status, load platforms, mark superseded). Mirror the `approvals_repository.go` style.
- **Handlers**: extend `api/profiles/handlers.go` with LPBS-config routes; new `api/releases/handlers.go` for release routes (wired in Phase 3).
- **Unit tests** table-driven using `testcontainers` postgres pattern already used in DM tests.

### Phase 2 — Orchestrator extensions (deployment-manager)
- New client `api/deployments/cloud_client.go` talks to `scenario-to-cloud`. Expose a `CloudClient` interface for test fakes. Step: *Check LPBS deployment health* — called after release-gate, before `deployValidateAndSign`; failure aborts orchestration with `failStep("cloud-health", err)`.
- New client `api/deployments/lpbs_client.go` calls `POST /api/v1/deploy-readiness` (per D2=A). **Auth (D8=B): `requireAdminOrService` using `LPBS_SERVICE_SECRET`** — DM reads the secret from its config (env var `LPBS_SERVICE_SECRET`, same pattern S2D already uses for presign/commit/apply) and sends it as the service header LPBS expects. Step: *Check LPBS upload readiness* — follows cloud-health.
- New step: *Allocate release_id* — only when request is publishing (not `DryRun`, not `SkipPackaging`). Acquires a **profile-scoped Postgres advisory lock** via `pg_try_advisory_xact_lock(hashtext('release:' || profile_id))` (D10=B); on lock contention returns HTTP 409 with `{"error":"release_in_flight","profile_id":...}` and a clear message. With the lock held, mints a UUID, inserts `releases` row (status `pending`), inserts one `release_platforms` row per target platform (status `pending`), echoes `release_id` into orchestration response. The advisory lock is held for the duration of the orchestration transaction and releases automatically on transaction end or connection drop.
- Extend `publishToLPBS` (orchestrator_helpers.go:30-100) to set `PublishDeployConfig.AppKey`, `RemoteProfile`, `UpdateURL` from the new config, and add `ReleaseID` + `Channel` fields to `PublishPipelineRequest` (forwarded to S2D — Phase 5 plumbs the rest). On publish success, update `releases.status = 'publishing'`, `release_platforms.status = 'uploading' → 'published'` per platform, `release_platforms.lpbs_artifact_id` from S2D response.
- New step: *Verify update endpoints* — for each platform, call LPBS `GET /updates/{app_key}/verify?channel=&platform=&expected_version=&deep=false`. On match, flip `release_platforms.status = 'published'`. On mismatch or error on any platform (per D3=A): `releases.status = 'verify_failed'`, the handler returns HTTP 502, orchestration step `failStep("verify")`. Include verification evidence (expected vs observed version, SHA512 match, timestamp) as JSON in an `error` / `notes` column or a dedicated `verification_evidence` JSONB.
- `PublishedVersion` rows (existing table) also get `release_id` so legacy views keep working.
- Extend `orchestrator_test.go` to assert step ordering, that LPBS coords reach the outbound request, that dry-run / skip-packaging bypasses release-record allocation, and that the advisory-lock concurrency guard returns 409 on the second of two parallel starts for the same profile.

### Phase 3 — Release lifecycle API + CLI (deployment-manager)
- New routes registered in `server/routes.go` (conditional on repo non-nil, matching the approvals/published-versions pattern):
  - `GET /api/v1/profiles/{id}/releases` — list for a profile.
  - `GET /api/v1/releases/{release_id}` — detail including per-platform rows.
  - `POST /api/v1/releases/{release_id}/verify` — re-run verification on demand; returns fresh evidence.
  - `POST /api/v1/profiles/{id}/releases/start` — per D4=B: operator-intent entrypoint. Body: `{channel, git_commit_hash, release_version, release_notes, platforms[]}`. `channel` is free-text and defaults to the profile's `default_channel` if omitted (D9=B). Handler drives the same underlying orchestrator used by `POST /deploy-desktop`. Internally generates the release record and invokes the existing phased pipeline.
- CLI verbs: `deployment-manager releases list|get|verify|start`. Uses existing `approvals` CLI package as a pattern (same flag shape, `--json` support, `--auto-start`). `releases start` accepts `--channel <name>` (free-text per D9=B); promote-as-a-verb is **not** added in this item.
- Tests per handler + CLI subcommand following `approvals_handler_test.go` and `approvals_cli_test.go` patterns.

### Phase 4 — UI surfaces (deployment-manager)
- `ProfileDetail` gets two new sections (per D5=B — no dedicated `/releases` page):
  - **LPBS Release Config card**: editable form for `lpbs_domain`, `lpbs_remote_profile`, `lpbs_app_key`, `default_channel`, `update_url`. Tied to new React-Query hook `useProfileLPBSConfig(profileId)`.
  - **Releases panel**: latest release + last 10 historical entries, per-platform verify badges (`published` / `verify_failed` / `pending`), release channel, release_id, and a re-verify action that calls `POST /releases/{id}/verify`. Hook `useProfileReleases(profileId, {limit: 10})`.
- New `api.ts` types: `LPBSReleaseConfig`, `Release`, `ReleasePlatform`, `VerificationEvidence`.
- Vitest coverage for: config card (render/edit/save/error), release panel (loading/empty/populated/mid-flight/verify-failed states), re-verify action (success + 409 for in-flight).

### Phase 5 — scenario-to-desktop wiring
- Per round 2 i1: S2D already carries `Config.Version → ReleaseVersion` and `Provenance.GitCommitHash` end-to-end. The S2D-side change reduces to threading **two new fields** through existing structs:
  - Add `ReleaseID` and `Channel` to `pipeline/PublishPipelineRequest`.
  - Surface them on `DeployConfig`, then have `stage_deploy.go` map `Channel` → `variantKey` via the existing `channelToVariantKey` helper and pass `ReleaseID` through to the existing `lpbs_client.go` plumbing (which already accepts `ReleaseID` on the commit payload and `variant_key` on apply).
  - `AppKey`, `RemoteProfile`, `UpdateURL` already exist on `PublishDeployConfig`; DM populates them in Phase 2 and they flow through unchanged.
- Tests in `scenario-to-desktop/api/pipeline` assert passthrough of `ReleaseID` and `Channel` from request to `lpbs_client.go` call sites.
- Update `scenario-to-desktop` CLI reference skill (doc only) to show the new flags.

### Phase 6 — Skill retargeting (prompt-manager)
- Rewrite `landing-page-desktop-upload/SKILL.md` per D6=A: single operational block calls `deployment-manager --auto-start releases start --profile <id> --channel <channel>`, polls with `deployment-manager releases get <id>` until a terminal status, renders the final status + per-platform verification table.
- Delete the release-time stages (1 cloud converge, 2 LPBS readiness, 3 pipeline run, 4 verify) from the skill body — they are now DM responsibilities.
- Keep setup-time gates (A–G in `landing-page-deploy-setup`) that configure LPBS; drop the release-time gates that DM now owns.
- Add a short "Fallback (DM unavailable)" section pointing at the pre-DM CLI relay; flag it as emergency-only.

### Phase 7 — Cleanup, scenario restart, validation
- `go build ./...`, `gofumpt -w .`, `golangci-lint run` in each scenario.
- `go test ./... -timeout 600s` in each scenario (tests can take 15+ minutes total in worst case).
- UI `npm run build` + `npx vitest run` in deployment-manager.
- Document "user must run `vrooli scenario restart deployment-manager landing-page-business-suite scenario-to-desktop`" — do NOT restart these in the plan runner; write to disk and let the user restart.

## 8. Contract Decisions

### Settled (from dependencies and research)
- DM owns canonical approval state; the gate check is commit-scoped, no channel dimension.
- DM owns the release_id (UUID). LPBS stores it on `download_artifacts.release_id` for correlation.
- Channel → LPBS `variant_key` mapping: `stable` → `default`; others pass through verbatim.
- `scenario-to-desktop` deploy stage already forwards `ReleaseID` and `variant_key` to `lpbs_client.go`.
- Update manifests include `releaseNotes` passthrough; `update_policy` defaults: 4h check interval, optional mode, no downgrade.
- Release record schema (from `release-record-contract-and-history-model` research): `releases` + `release_platforms` two-table design; migration numbered `004_add_releases.sql`.
- Dry-run and skip-packaging orchestrator runs do NOT create release records.

### Settled in round 1 workshop
- **D1=B** — LPBS release config lives in a new 1:1 child table `profile_lpbs_release_config` (not on `profiles`, not supplied at deploy time).
- **D2=A** — DM calls a new LPBS `POST /api/v1/deploy-readiness` HTTP endpoint via a typed client. LPBS reuses its existing CLI gate logic behind the new handler.
- **D3=A** — Verification failure is a hard release failure (`releases.status='verify_failed'`, HTTP 502 from the release-start handler). No "published_unverified" state.
- **D4=B** — New `POST /api/v1/profiles/{id}/releases/start` is the operator-intent entrypoint. `/deploy-desktop` remains as the underlying build-and-bundle endpoint invoked by the release-start handler.
- **D5=B** — UI release surface is embedded on `ProfileDetail` only. No dedicated `/releases` page in this item.
- **D6=A** — `landing-page-desktop-upload` skill becomes a thin DM automation wrapper; the 6-stage CLI relay is removed from the skill body.

### Settled in round 2 workshop
- **D7=A** — Release record schema adopts the research's two-table design verbatim (`releases` + `release_platforms`). Per-platform status updates, FK to `deployment_approvals`, normalized indexes; `promoted_from_release_id` reserved on `releases` for future promote without schema change.
- **D8=B** — LPBS's new `POST /api/v1/deploy-readiness` endpoint uses `requireAdminOrService` (LPBS_SERVICE_SECRET), the same peer-service auth model already used for presign/commit/apply. DM holds the secret in env (`LPBS_SERVICE_SECRET`); no new secret type or per-app-key reuse.
- **D9=B** — Promote API/CLI/UI is **deferred** to a follow-up backlog item. This item ensures the schema (`channel` is free-text TEXT, `promoted_from_release_id` reserved) and the orchestrator (accepts `--channel <name>` for any value) handle arbitrary channels today, so an operator can release to `beta` or `nightly` directly. Promotion (creating a promoted copy that points back via `promoted_from_release_id`) is not implemented here.
- **D10=B** — Concurrency guard is a Postgres advisory lock at profile scope: `pg_try_advisory_xact_lock(hashtext('release:' || profile_id))` for the duration of the release-orchestration transaction. Lock contention returns HTTP 409 with a clear `release_in_flight` payload. Rejected: app-level mutex (single-node only) and "rely on UNIQUE only" (doesn't serialize different commits).

## 9. Testing Plan

### Unit
- `profile_lpbs_release_config` repo CRUD — table-driven against testcontainers postgres.
- `releases` + `release_platforms` repos: insert, load (with platforms join), status transitions, superseded marking, UNIQUE-constraint rejection on `(profile_id, git_commit_hash, channel)`.
- Orchestrator step additions: extend `orchestrator_test.go` with fakes for `CloudClient` and `LPBSClient`; assert step sequence (release-gate → cloud-health → LPBS readiness → allocate release_id → build/package → publish → verify), state transitions on each hop, and response payload includes `release_id` + verification evidence.
- **Dry-run / skip-packaging exclusion**: orchestrator test asserts that a request with `DryRun=true` or `SkipPackaging=true` does NOT insert into `releases` or `release_platforms`.
- **Concurrency guard (D10=B)**: two parallel `releases/start` requests for the same profile — first acquires the advisory lock and proceeds, second receives HTTP 409 with `release_in_flight` payload. Test also asserts the lock releases on transaction end so a third request after the first completes succeeds.
- `publishToLPBS` now sets all five LPBS coords on `PublishDeployConfig` — guard via fake `DesktopPackagerClient` that captures the outbound request.
- **Verification hard-fail path**: fake LPBS `/updates/.../verify` returning `match=false`; test asserts `releases.status='verify_failed'`, handler returns 502, and verification evidence is persisted.
- **Arbitrary channel acceptance (D9=B)**: orchestrator test asserts `--channel beta` is accepted end-to-end and recorded on `releases.channel` without schema rejection.
- New CLI subcommands (`releases list|get|verify|start`) mirror the approvals CLI pattern with fake `APIClient`.

### Integration
- DM → S2D contract: one test that spins up a fake S2D server (httptest) and asserts `PublishPipelineRequest` carries `AppKey`, `RemoteProfile`, `Channel`, `ReleaseID`, `UpdateURL`.
- DM → LPBS verify contract: fake LPBS server returning match/no-match; ensure DM records evidence and transitions status per D3.
- DM → LPBS deploy-readiness contract (D8=B): fake LPBS server; assert DM sends the `LPBS_SERVICE_SECRET` header expected by `requireAdminOrService`, and that LPBS's own handler test rejects requests without the secret (401/403) and accepts requests with it (200 / 4xx with structured per-gate error).
- LPBS `/deploy-readiness` handler test: uses existing gate logic, asserts 200 on ready, 400/409 with structured error on unready per-gate, and `requireAdminOrService` middleware coverage.

### UI
- Vitest + React Testing Library for `ProfileDetail` LPBS config card (render, edit, save, error path).
- Releases panel (loading/empty/populated/mid-flight/verify-failed states; re-verify action success + 409 for in-flight).
- Hook tests (`useProfileLPBSConfig`, `useProfileReleases`) via Mock Service Worker.

### Manual (documented, not required to pass)
- `deployment-manager releases start --profile <id> --channel stable` end-to-end against local LPBS and a test scenario; confirm update manifest serves new version.

### Golden-path CLI smoke (automated where possible)
- `bats` test that runs `deployment-manager releases list --json` and asserts schema shape.
- Skip runtime LPBS calls in CI; mock via envvars.

## 10. Rollout / Validation Checklist

- [ ] Phase 1 merged: migrations apply cleanly on an empty DB and on a DB with existing deployments rows.
- [ ] Phase 2 merged: `go test ./api/deployments/...` passes including orchestrator sequence tests, advisory-lock concurrency test, and dry-run exclusion test.
- [ ] Phase 3 merged: CLI `releases` subcommands listed in `deployment-manager --help`; `releases list` returns JSON on `--json`; `releases start --channel beta` accepted end-to-end.
- [ ] Phase 4 merged: UI Profile page shows LPBS config card and latest release summary; `npm run build` succeeds; Vitest green.
- [ ] Phase 5 merged: `scenario-to-desktop/api/pipeline` tests assert passthrough of `release_id` and `channel`.
- [ ] Phase 6 merged: `prompt-manager skill read landing-page-desktop-upload` shows new DM-driven flow; markdown lint passes.
- [ ] Phase 7: all three scenarios build + lint clean.
- [ ] User-run verification: `vrooli scenario restart ...` then exercise `releases start`, confirm DB rows and verify endpoint evidence.

## 11. Risks + Mitigations

| Risk | Mitigation |
|------|------------|
| `scenario-to-cloud` client surface drifts from its current CLI flags (`deployment health --scenario ...`) | Pin to a stable API endpoint if one exists; otherwise abstract in a `CloudClient` interface with a faked test double so we can adapt without touching the orchestrator. |
| LPBS `deploy-readiness` endpoint behavior diverges from the existing CLI gate logic | New LPBS handler calls the same internal gate-check functions as the CLI command; integration test asserts parity on a shared fixture. |
| Two release entry points (`deploy-desktop` and `releases start`) create drift | D4=B bounds this: `releases/start` is the operator surface; `deploy-desktop` is the lower-level primitive it calls. Shared orchestrator means shared behavior. |
| Migration lands on a DB with in-flight deployments | New tables are additive; `releases` and `release_platforms` start empty. No destructive schema changes. |
| Skill retargeting breaks existing automation using the old stage sequence | Replace atomically with the DM entrypoint; keep a fallback section documenting the old relay for DM-unavailable cases. Skill callers who pinned the sub-skill names will need a single update. |
| Test flakiness from real LPBS / cloud calls | All new clients interface-driven + tested with httptest fakes. No CI dependency on live services. |
| UI surface creeps beyond plan scope | D5=B explicitly scopes UI to `ProfileDetail` embedding only. |
| Verification endpoint occasionally slow (S3 deep check) | Default verification is lightweight; deep only on explicit re-verify. |
| Concurrent `releases/start` requests for the same profile race on release-record insert | Resolved by D10=B: profile-scoped Postgres advisory lock serializes orchestration; second caller fast-fails with 409 `release_in_flight`. Test covers both parallel-start outcomes and lock release on transaction end. |
| Orphaned release records (DM crash mid-flight leaves rows in `pending`/`publishing`) | Release records are recoverable read-only state; a sweep job or manual re-verify transitions stuck rows. The advisory lock auto-releases on connection drop, so a crashed run does not block subsequent attempts. The `POST /releases/{id}/verify` endpoint lets an operator re-run verification on any `published`/`verify_failed` row. |
| Auth mismatch between DM and LPBS on the new `/deploy-readiness` endpoint | D8=B pins `requireAdminOrService` (LPBS_SERVICE_SECRET). Integration test asserts DM sends the secret header and LPBS rejects unauthorized callers. DM startup logs a warning if `LPBS_SERVICE_SECRET` is unset, so the misconfiguration is visible early. |
| Future need to promote releases across channels not anticipated in this schema | Per D9=B, schema includes `promoted_from_release_id` and `channel` is free-text TEXT (no CHECK constraint), so a follow-up promote API can land without another migration. The orchestrator already accepts arbitrary channel values today. |

## 12. Non-goals / Prohibited Patterns

- Do NOT add compatibility shims so DM can talk to S2D with or without `release_id` / `channel` — always pass them.
- Do NOT duplicate LPBS app/artifact state in DM; only correlation keys.
- Do NOT call LPBS directly from S2D AND from DM for the same action; choose the owner per action (publish via S2D; verify + readiness via DM).
- Do NOT implement rollback, auto-deploy, or webhook notifications in this item.
- Do NOT implement a promote-release API, CLI verb, or UI action in this item (D9=B). Schema reserves `promoted_from_release_id` and orchestrator accepts arbitrary channels, but the promote *flow* (creating a promoted copy) is a follow-up.
- Do NOT add a dedicated `/releases` UI page. Release status lives on `ProfileDetail` only (D5).
- Do NOT introduce a "published_unverified" status — verification is a hard gate (D3).
- Do NOT use per-app `X-Update-Key` or admin-only auth on `/deploy-readiness` — D8=B pins it to `requireAdminOrService`.
- Do NOT use an in-memory mutex for release concurrency — D10=B pins it to a Postgres advisory lock so the guard survives multi-node deploys.
- Do NOT skip tests; every new handler, repo method, CLI subcommand, and UI component needs coverage before being declared done.
- Do NOT use `--no-verify`, `gofmt --amend`, or `git reset --hard` to silence lint/test failures.
- Do NOT restart scenarios from within the plan; user restarts manually.

## 13. Definition of Done

1. All 7 phases merged under a single feature branch and tested together.
2. `go build ./...`, `gofumpt -w .`, `golangci-lint run`, and `go test ./... -timeout 600s` pass in `deployment-manager`, `scenario-to-desktop`, and `landing-page-business-suite`.
3. UI in deployment-manager builds and Vitest is green.
4. `deployment-manager releases list --json` returns rows for a test profile after a completed release.
5. `GET /api/v1/releases/{release_id}` returns a record with: profile_id, git_commit_hash, channel, per-platform version+sha512+verify status, deployment_id, status.
6. `landing-page-desktop-upload` skill is a thin DM wrapper; old multi-stage CLI relay is removed from the skill body (fallback documented separately).
7. Research actions 1 and 2 (git commit hash on LPBS artifacts + version sync callback) are either delivered by this item or re-filed as follow-ups with explicit links from this plan.
8. Every workshop decision in section 8 has a `selected` value incorporated into the relevant plan sections; no `<!-- TBD -->` remains in section 7 or 8. All ten decisions (D1–D10) are resolved.
