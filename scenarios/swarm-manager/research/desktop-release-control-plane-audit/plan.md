# Audit: Desktop Release Control Plane

## Purpose

Map the real desktop monetization release path end-to-end across deployment-manager, scenario-to-desktop, landing-page-business-suite (LPBS), scenario-to-cloud, and the prompt-manager deployment skills. Produce a concrete system contract and dependency map that later implementation items can follow.

## Required Reading

```bash
prompt-manager skill read scenario-to-desktop deployment-coordinator
prompt-manager skill read cli-steer api-steer
```

## Problem Statement

The desktop release pipeline spans 5 scenarios and 3 deployment skills, with overlapping responsibilities for version tracking, build records, approval gating, and artifact publishing. There is no single authoritative document mapping which surface owns which state, creating ambiguity for agents and developers building on or extending the pipeline.

**Key questions this audit answers:**
1. Which surface owns canonical approval state?
2. Which surface owns release/build/version records?
3. Which step actually publishes artifacts and update manifests?
4. Where is LPBS app/app-key/channel/release-note state the source of truth?
5. Which actions exist in API/UI/CLI/skills vs. are still implicit or manual?

## Scope

**In scope:**
- Mapping ownership and data flow across all 5 scenarios
- Identifying gaps, overlaps, and implicit contracts
- Producing a dependency map and system contract

**Out of scope:**
- Code changes or implementation work
- Mobile (iOS/Android) deployment paths
- SaaS/cloud deployment path (except where scenario-to-cloud touches desktop)
- Performance or security audits

## Current Technical Context

### System Inventory

| Scenario | Role | Key State | Primary Interface |
|----------|------|-----------|-------------------|
| **deployment-manager** | Orchestrator + approval gate | Profiles, deployments, approvals (PostgreSQL) | REST API, CLI, UI |
| **scenario-to-desktop** | Builder + publisher | Pipeline state, build provenance, deploy targets (file-based + memory) | REST API, CLI, UI |
| **LPBS** | Artifact host + manifest server | Apps, assets, artifacts, storage config (PostgreSQL + S3) | REST API, UI |
| **scenario-to-cloud** | VPS deployer (incl. LPBS itself) | Deployment records, bundle hashes (PostgreSQL) | REST API, CLI, UI |
| **prompt-manager skills** | Agent workflow encoding | N/A (stateless instructions) | Skill read commands |

### Key Files

| Component | Path |
|-----------|------|
| deployment-manager orchestrator | `scenarios/deployment-manager/api/deployments/orchestrator.go` |
| deployment-manager approvals | `scenarios/deployment-manager/api/deployments/approvals_handler.go` |
| deployment-manager desktop client | `scenarios/deployment-manager/api/deployments/desktop_client.go` |
| scenario-to-desktop pipeline | `scenarios/scenario-to-desktop/api/pipeline/orchestrator.go` |
| scenario-to-desktop deploy stage | `scenarios/scenario-to-desktop/api/pipeline/stage_deploy.go` |
| scenario-to-desktop LPBS client | `scenarios/scenario-to-desktop/api/deploy/lpbs_client.go` |
| LPBS update handler | `scenarios/landing-page-business-suite/api/update_handlers.go` |
| LPBS artifact hosting | `scenarios/landing-page-business-suite/api/download_hosting.go` |
| LPBS download service | `scenarios/landing-page-business-suite/api/download_service.go` |
| scenario-to-cloud freshness | `scenarios/scenario-to-cloud/api/freshness.go` |
| Skill: scenario-to-desktop | `scenarios/prompt-manager/store/skills/packs/core/scenario-to-desktop/` |
| Skill: landing-page-deploy-setup | `scenarios/prompt-manager/store/skills/packs/core/landing-page-deploy-setup/` |
| Skill: landing-page-desktop-upload | `scenarios/prompt-manager/store/skills/packs/core/landing-page-desktop-upload/` |

## Target End State

A single authoritative document (this plan, once complete) that:
1. Defines which system is the **source of truth** for each piece of state
2. Maps the **data flow** for a complete release cycle
3. Identifies **implicit contracts** that should be made explicit
4. Catalogs **all existing actions** by surface (API, CLI, UI, skill)
5. Lists **gaps** where manual steps have no API/automation support

## Findings: Ownership Map

### 1. Canonical Approval State → deployment-manager

**Source of truth:** `deployment_approvals` table in deployment-manager PostgreSQL.

| Field | Purpose |
|-------|---------|
| `profile_id + git_commit_hash + platform` | Unique approval identity |
| `status` (pending/approved/rejected/stale) | Current approval state |
| `approved_by`, `approved_at`, `notes` | Audit trail |
| `validation_id` | Link to visual validation run |

**APIs:**
- `POST /api/v1/profiles/{id}/approvals` — create approval request
- `POST /api/v1/approvals/{id}/decide` — approve/reject
- `GET /api/v1/profiles/{id}/release-gate?commit={hash}` — check all platforms approved
- `PUT /api/v1/profiles/{id}/required-platforms` — configure which platforms gate

**Stale detection:** New commit → old approvals marked stale. Release gate checks ALL required platforms approved for the specific commit.

**Gap:** No webhook/notification when approval status changes (P2 backlog item in deployment-manager).

### 2. Release/Build/Version Records → Split Across 3 Systems

| Record Type | Owner | Storage | Notes |
|-------------|-------|---------|-------|
| Deployment profile + version history | deployment-manager | `profiles` + `profile_versions` (PostgreSQL) | Auto-incremented version on profile update |
| Deployment execution log | deployment-manager | `deployments` table (PostgreSQL) | Status, artifacts JSONB, logs |
| Build provenance (git commit, branch) | scenario-to-desktop | Pipeline state (file-based JSON) | `BuildProvenance` struct |
| Platform build results | scenario-to-desktop | Pipeline state (file-based JSON) | `PlatformResult` per platform |
| Published artifact metadata | LPBS | `download_artifacts` (PostgreSQL) | SHA256/512, size, S3 location |
| Published release version | LPBS | `download_assets` (PostgreSQL) | `release_version`, `release_notes` per platform/channel |

**Key insight:** There is no single place to query "what version is live for app X on platform Y in channel Z?" You must query LPBS for the published state (`download_assets`) and deployment-manager for the approval/deployment state.

### 3. Artifact Publishing Flow → scenario-to-desktop → LPBS

The deploy stage in scenario-to-desktop's pipeline handles the actual publishing:

```
scenario-to-desktop pipeline (stage_deploy.go)
  │
  ├─ 1. TestRemoteProfile(profileTag)     → LPBS: verify session
  ├─ 2. DeriveUpdateURL(profile, appKey)  → LPBS: get auto-update URL
  ├─ 3. PresignUpload(...)                → LPBS: get S3 presigned URL
  ├─ 4. Upload artifact to S3             → S3: direct upload
  ├─ 5. CommitArtifact(...)               → LPBS: record metadata
  └─ 6. Apply/SetCurrent(...)             → LPBS: link artifact to asset, promote
```

**deployment-manager does NOT publish directly.** Its orchestrator calls scenario-to-desktop's API, which in turn calls LPBS.

### 4. Update Manifest Generation → LPBS

**Endpoint:** `GET /api/v1/updates/{app_key}/{channel}/{file}`

Manifests are generated **dynamically** from `download_artifacts` metadata:
- `latest.yml` → Windows artifact
- `latest-mac.yml` → macOS artifact
- `latest-linux.yml` → Linux artifact

Channel mapping: `"stable"` → `variant_key="default"`, others pass through.

Optional per-app API key gating via `X-Update-Key` header.

### 5. LPBS App/Channel/Release State — Source of Truth

| State | Table | Key Fields |
|-------|-------|------------|
| App identity | `download_apps` | `bundle_key`, `app_key`, `name`, `update_api_key` |
| Release per platform/channel | `download_assets` | `(bundle_key, app_key, platform, variant_key)` → `release_version`, `release_notes`, `artifact_id` |
| Binary artifact | `download_artifacts` | `sha256`, `sha512`, `size_bytes`, `bucket`, `object_key` |
| Storage config | `download_storage_settings` | S3 endpoint, bucket, credentials per bundle |

### 6. Action Inventory

<!-- TBD — will be populated with a comprehensive matrix after decisions on audit depth -->

## Findings: End-to-End Data Flow

```
Developer pushes code
       │
       ▼
deployment-manager: Create approval requests (per platform)
       │
       ▼
Reviewer: Approve/reject via deployment-manager API/UI
       │
       ▼
deployment-manager: Release gate check (all platforms approved?)
       │  ✓ all approved
       ▼
deployment-manager: POST /api/v1/deploy-desktop
       │  (orchestrator.go)
       ├─ Load profile, validate, assemble manifest
       ├─ Build binaries (cross-compile)
       ├─ Call scenario-to-desktop API
       │       │
       │       ▼
       │  scenario-to-desktop: Pipeline execution
       │       ├─ Bundle → Preflight → Generate → Build → Smoketest
       │       └─ Deploy stage → LPBS
       │               │
       │               ▼
       │          LPBS: Presign → S3 Upload → Commit → Apply
       │               │
       │               ▼
       │          LPBS: Update manifests now serve new version
       │
       ▼
deployment-manager: Record deployment result
```

**Alternative flow (skill-driven):** The `landing-page-desktop-upload` skill encodes a similar flow but executed by an agent via CLI commands rather than API orchestration. It adds:
- Stage -0: Discovery (profile tags, app keys)
- Stage 0: Health checks (all 3 tools responsive)
- Stage 0.5: Build prerequisites (fpm/Ruby for Linux)
- Stage 1: Cloud health (LPBS deployment is healthy)
- Stage 2: LPBS readiness (deploy-readiness gate)
- Stage 3: Build & deploy (scenario-to-desktop pipeline)
- Stage 4: Post-release verification (manifest endpoints live)

## Findings: Gaps and Implicit Contracts

### Implicit Contracts (should be made explicit)

1. **LPBS_SERVICE_SECRET synchronization** — scenario-to-desktop must have the same secret as LPBS runtime. The `landing-page-deploy-setup` skill handles this manually, but there's no API contract.

2. **Deploy target configuration** — scenario-to-desktop stores deploy targets in `.vrooli/deploy-targets.json`. These reference LPBS remote profile tags but are managed independently.

3. **Version propagation** — When scenario-to-desktop publishes v1.2.3 to LPBS, deployment-manager has no callback to record "this profile's latest published version is 1.2.3."

4. **Build provenance → LPBS linkage** — scenario-to-desktop tracks git commit in `BuildProvenance`, but this is not forwarded to LPBS. The `download_artifacts` table has no git commit field.

### Gaps (no API/automation support)

1. **No webhook from LPBS on publish** — After an artifact is published, nothing notifies deployment-manager or other systems.

2. **No "current live version" query** — To determine what's live, you must query LPBS directly. deployment-manager tracks deployment attempts but not post-publish state.

3. **No rollback API** — To rollback a release, you must manually `set-current` a prior artifact in LPBS. No single "rollback" action exists.

4. **No approval → deploy automation** — Approvals and deploys are separate actions. No automation triggers deploy when all approvals pass.

5. **Entitlement ↔ approval disconnect** — LPBS has `requires_entitlement` per asset (subscription gating) but this is independent from deployment-manager approvals (release gating). These serve different purposes but could confuse operators.

## Implementation Strategy

This is a research item — the deliverable is this document itself, not code changes. The strategy is:

1. **Phase 1 (this round):** Map ownership and data flow from codebase exploration ✓
2. **Phase 2 (next round):** Validate findings against actual runtime behavior, fill action inventory matrix
3. **Phase 3 (final):** Produce the system contract with explicit interface boundaries and recommended improvements

## Contract Decisions

<!-- TBD — will be formalized after workshop decisions on contract format and depth -->

## Testing Plan

Research validation:
- [ ] Cross-reference each ownership claim against the actual database schema
- [ ] Verify API endpoints exist and accept the documented parameters
- [ ] Trace one complete release through logs/state to confirm the data flow diagram
- [ ] Validate skill instructions match current API contracts

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Codebase changes during audit | Findings become stale | Pin findings to current commit; re-validate before finalizing |
| Undocumented runtime behavior | Missing implicit contracts | Trace actual release flow, not just code |
| Scope creep into implementation | Delays research completion | Strict separation: document gaps, don't fix them |

## Non-goals / Prohibited Patterns

- Do NOT implement fixes for identified gaps (separate backlog items)
- Do NOT modify any scenario code as part of this research
- Do NOT create new APIs or database schemas

## Definition of Done

- [ ] All 5 questions from the description are answered with evidence
- [ ] Ownership map covers all state (approval, build, version, artifact, manifest)
- [ ] End-to-end data flow diagram is validated
- [ ] Action inventory matrix is complete (API, CLI, UI, skill per system)
- [ ] Gaps and implicit contracts are cataloged
- [ ] System contract is formalized with explicit interface boundaries
