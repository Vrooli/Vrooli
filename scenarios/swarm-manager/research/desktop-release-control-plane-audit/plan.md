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
- Exhaustive action inventory across all surfaces (API, CLI, UI, skill)
- Prioritized recommendations for gap remediation

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
| **LPBS** | Artifact host + manifest server | Apps, assets, artifacts, storage config (PostgreSQL + S3) | REST API, CLI, UI |
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
4. Catalogs **all existing actions** by surface (API, CLI, UI, skill) — exhaustive matrix
5. Lists **gaps** where manual steps have no API/automation support
6. Provides **prioritized recommendations** for gap remediation

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

## Findings: Exhaustive Action Inventory

### deployment-manager

#### API Endpoints (86+)

| Category | Method | Path | Purpose |
|----------|--------|------|---------|
| **Health** | GET | `/health`, `/api/v1/health` | Health checks |
| **Dependencies** | GET | `/api/v1/dependencies/analyze/{scenario}` | Analyze scenario dependencies |
| **Fitness** | POST | `/api/v1/fitness/score` | Calculate fitness scores by tier |
| **Profiles** | GET | `/api/v1/profiles` | List all profiles |
| | POST | `/api/v1/profiles` | Create profile |
| | GET | `/api/v1/profiles/{id}` | Get profile |
| | PUT | `/api/v1/profiles/{id}` | Update profile |
| | DELETE | `/api/v1/profiles/{id}` | Delete profile |
| | GET | `/api/v1/profiles/{id}/versions` | Version history |
| **Deploy** | POST | `/api/v1/deploy/{profile_id}` | Deploy profile |
| | POST | `/api/v1/deploy-desktop` | Full desktop deployment orchestration |
| | GET | `/api/v1/deployments/{id}` | Deployment status |
| **Swaps** | GET | `/api/v1/swaps/suggest/{scenario}` | List recommended swaps |
| | GET | `/api/v1/swaps/analyze/{from}/{to}` | Analyze swap impact |
| | GET | `/api/v1/swaps/cascade/{from}/{to}` | Cascade impact detection |
| | POST | `/api/v1/swaps/apply` | Apply swap |
| | POST | `/api/v1/profiles/{id}/swaps` | Apply swap to profile |
| **Validation** | GET | `/api/v1/profiles/{id}/validate` | Validate profile |
| | GET | `/api/v1/profiles/{id}/cost-estimate` | Cost estimation |
| **Secrets** | GET | `/api/v1/profiles/{id}/secrets` | Identify required secrets |
| | GET | `/api/v1/profiles/{id}/secrets/template` | Secret template (env/json) |
| | POST | `/api/v1/profiles/{id}/secrets/validate` | Validate secrets |
| | POST | `/api/v1/secrets/validate` | Validate single secret |
| | GET/POST | `/api/v1/secrets/test` | Test secret functionality |
| **Bundles** | POST | `/api/v1/bundles/validate` | Validate bundle manifest |
| | POST | `/api/v1/bundles/merge-secrets` | Merge secrets into bundle |
| | POST | `/api/v1/bundles/assemble` | Assemble bundle from scenario |
| | POST | `/api/v1/bundles/export` | Export production bundle |
| | POST | `/api/v1/bundles/signing-config` | Generate signing config |
| **Telemetry** | GET | `/api/v1/telemetry` | List summaries |
| | POST | `/api/v1/telemetry/upload` | Upload events |
| **Signing** | GET | `/api/v1/profiles/{id}/signing` | Get signing config |
| | PUT | `/api/v1/profiles/{id}/signing` | Set signing config |
| | PATCH | `/api/v1/profiles/{id}/signing/{platform}` | Platform-specific signing |
| | DELETE | `/api/v1/profiles/{id}/signing` | Remove signing config |
| | DELETE | `/api/v1/profiles/{id}/signing/{platform}` | Remove platform signing |
| | POST | `/api/v1/profiles/{id}/signing/validate` | Validate signing |
| | GET | `/api/v1/signing/prerequisites` | Check signing tools |
| | GET | `/api/v1/signing/discover/{platform}` | Discover certificates |
| **Approvals** | POST | `/api/v1/profiles/{id}/approvals` | Create approval request |
| | GET | `/api/v1/profiles/{id}/approvals` | List approvals |
| | GET | `/api/v1/approvals/{id}` | Get approval |
| | POST | `/api/v1/approvals/{id}/decide` | Approve/reject |
| | GET | `/api/v1/profiles/{id}/release-gate` | Release gate check |
| | PUT | `/api/v1/profiles/{id}/required-platforms` | Set required platforms |
| | GET | `/api/v1/profiles/{id}/required-platforms` | Get required platforms |
| **Visual Validation** | POST | `/api/v1/validations` | Create validation run |
| | GET | `/api/v1/validations/{id}` | Get validation |
| | GET | `/api/v1/validations/{id}/video` | Stream validation video |
| | POST | `/api/v1/validations/{id}/review` | Submit review |
| | GET | `/api/v1/profiles/{id}/validations` | List validations |
| **Build** | POST | `/api/v1/build` | Cross-compile binaries |
| | POST | `/api/v1/build/auto` | Auto-build (async) |
| | GET | `/api/v1/build/{build_id}` | Build status |
| | GET | `/api/v1/build/auto/{build_id}` | Auto-build status |

#### CLI Commands (50+)

| Group | Command | Purpose |
|-------|---------|---------|
| `overview` | `status`, `analyze`, `fitness` | Health, dependency analysis, fitness scoring |
| `profile` | `list`, `create`, `show`, `delete`, `export`, `import`, `update`, `set`, `swap`, `versions`, `analyze`, `save`, `diff`, `rollback`, `secrets identify/template/validate` | Full profile lifecycle |
| `deploy` | `deploy`, `deployment status`, `validate`, `estimate-cost`, `build`, `deploy-desktop`, `logs` | Deployment operations |
| `swaps` | `list`, `analyze`, `cascade`, `info`, `apply` | Swap management |
| `bundle` | `assemble`, `export`, `validate` | Bundle operations |
| `signing` | `show`, `set`, `remove`, `validate`, `prerequisites`, `discover`, `help` | Code signing |
| `validations` | `run`, `status`, `video`, `review`, `list` | Visual validation |

#### UI Pages (8)

| Route | Purpose |
|-------|---------|
| `/` | Dashboard — overview and status |
| `/profiles` | Profile list |
| `/profiles/new` | Create profile |
| `/profiles/:id` | Profile detail + edit + secrets + signing |
| `/analyze` | Dependency analysis + swap suggestions |
| `/telemetry` | Telemetry summaries |
| `/deployments` | Deployment list + status |
| `/deployments/:id` | Deployment detail + logs |

---

### scenario-to-desktop

#### API Endpoints (60+)

| Category | Method | Path | Purpose |
|----------|--------|------|---------|
| **Health** | GET | `/health`, `/api/v1/health`, `/api/v1/status` | Health + system status |
| **Pipeline** | POST | `/api/v1/pipeline/run` | Start pipeline (bundle→preflight→generate→build→smoketest→deploy) |
| | GET | `/api/v1/pipeline/{id}` | Pipeline status |
| | POST | `/api/v1/pipeline/{id}/resume` | Resume pipeline |
| | POST | `/api/v1/pipeline/{id}/cancel` | Cancel pipeline |
| | GET | `/api/v1/pipelines` | List pipelines |
| | GET | `/api/v1/scenarios/{name}/pipeline/active` | Get/create active pipeline |
| **Tasks** | POST | `/api/v1/pipeline/{id}/tasks` | Spawn agent (investigate/fix) |
| | GET | `/api/v1/pipeline/{id}/tasks` | List tasks |
| | GET | `/api/v1/pipeline/{id}/tasks/{taskId}` | Get task |
| | POST | `/api/v1/pipeline/{id}/tasks/{taskId}/stop` | Stop task |
| **Templates** | GET | `/api/v1/templates` | List templates |
| | GET | `/api/v1/templates/{type}` | Get template |
| **Wine** | GET | `/api/v1/system/wine/check` | Check Wine |
| | POST | `/api/v1/system/wine/install` | Install Wine |
| **Download** | GET | `/api/v1/desktop/download/{scenario}/{platform}` | Download built package |
| **Records** | GET | `/api/v1/desktop/records` | List records |
| | POST | `/api/v1/desktop/records/{id}/move` | Move wrapper |
| | DELETE | `/api/v1/desktop/delete/{scenario}` | Delete app |
| **Telemetry** | POST | `/api/v1/deployment/telemetry` | Ingest events |
| | GET | `/api/v1/deployment/telemetry/{scenario}/summary` | Summary |
| | GET | `/api/v1/deployment/telemetry/{scenario}/insights` | AI insights |
| | GET | `/api/v1/deployment/telemetry/{scenario}/tail` | Recent entries |
| | GET | `/api/v1/deployment/telemetry/{scenario}/download` | Download file |
| | DELETE | `/api/v1/deployment/telemetry/{scenario}` | Delete telemetry |
| **Desktop Status** | GET | `/api/v1/scenarios/desktop-status` | All scenarios' desktop status |
| **State** | GET/PUT/DELETE | `/api/v1/scenarios/{scenario}/state` | Scenario state CRUD |
| | POST | `/api/v1/scenarios/{scenario}/state/check` | Staleness check |
| | POST | `/api/v1/scenarios/{scenario}/state/invalidate` | Invalidate state |
| **Signing** | GET/PUT/DELETE | `/api/v1/signing/{scenario}` | Signing config CRUD |
| | PATCH/DELETE | `/api/v1/signing/{scenario}/{platform}` | Platform signing |
| | POST | `/api/v1/signing/{scenario}/validate` | Validate signing |
| | GET | `/api/v1/signing/{scenario}/ready` | Signing readiness |
| | POST | `/api/v1/signing/{scenario}/linux/generate-key` | Generate GPG key |
| | GET | `/api/v1/signing/prerequisites` | Check tools |
| | GET | `/api/v1/signing/discover/{platform}` | Discover certs |
| **Deploy Targets** | GET | `/api/v1/deploy-targets` | List targets |
| | GET/PUT/DELETE | `/api/v1/deploy-targets/{name}` | Target CRUD |
| | POST | `/api/v1/deploy-targets/{name}/test` | Test session |
| | POST | `/api/v1/deploy-targets/{name}/doctor` | Diagnose readiness |
| **Live Desktop** | POST | `/api/v1/livedesktop/sessions` | Start VNC session |
| | GET | `/api/v1/livedesktop/sessions` | List sessions |
| | Various | `/api/v1/livedesktop/sessions/{id}/*` | Session ops (heartbeat, launch, control, metrics, files, stop, WS) |
| **Captures** | GET | `/api/v1/captures/{scenario}` | List captures |
| | Various | `/api/v1/captures/{scenario}/*` | Capture ops (summary, file, delete, download) |
| **Tools** | GET | `/api/v1/tools` | Tool manifest |
| | POST | `/api/v1/tools/execute` | Execute tool |

#### CLI Commands (40+)

| Group | Commands | Purpose |
|-------|----------|---------|
| Flat | `status`, `templates`, `template`, `records`, `records-move`, `records-delete`, `download`, `desktop-status`, `configure` | Quick operations |
| `pipeline` | `run`, `status`, `resume`, `cancel`, `list`, `active`, `create`, `reset`, `history`, `start` | Pipeline lifecycle |
| `bundle` | `clean` | Clean bundle output |
| `telemetry` | `ingest`, `summary`, `insights`, `tail`, `download`, `delete` | Telemetry ops |
| `signing` | `get`, `set`, `delete`, `validate`, `ready`, `prerequisites`, `discover`, `generate-key` | Code signing |
| `wine` | `check`, `install`, `status` | Wine management |
| `deploy-target` | `list`, `add`, `remove`, `test`, `doctor` | LPBS deploy targets |

#### UI (5 tab views)

| Tab | Purpose |
|-----|---------|
| Inventory | Discover scenarios + desktop status |
| Generate | Pipeline orchestration + build form |
| Apps | Browse/download/move generated apps |
| Signing | Code signing configuration |
| Docs | Documentation browser |

---

### LPBS (Download/Artifact Surface)

#### API Endpoints (15 download-related)

| Category | Method | Path | Purpose |
|----------|--------|------|---------|
| **User Downloads** | GET | `/api/v1/downloads` | User download with entitlement check (auth required) |
| **Update Channel** | GET | `/api/v1/updates/{app_key}/{channel}/{file}` | Electron-updater manifests + binary redirects (optional API key) |
| **Storage Admin** | GET | `/api/v1/admin/download-storage` | Get S3 config (masked) |
| | PUT | `/api/v1/admin/download-storage` | Update S3 config |
| | POST | `/api/v1/admin/download-storage/test` | Test S3 connection |
| **Apps Admin** | GET | `/api/v1/admin/download-apps` | List apps with assets |
| | POST | `/api/v1/admin/download-apps` | Create app |
| | PUT | `/api/v1/admin/download-apps/{app_key}` | Update app |
| | DELETE | `/api/v1/admin/download-apps/{app_key}` | Delete app |
| **Artifacts Admin** | GET | `/api/v1/admin/download-artifacts` | List artifacts (paginated) |
| | GET | `/api/v1/admin/download-artifacts/by-app` | List by app |
| | POST | `/api/v1/admin/download-artifacts/presign-upload` | Presign S3 PUT (admin or service auth) |
| | POST | `/api/v1/admin/download-artifacts/commit` | Register artifact after upload (admin or service auth) |
| | GET | `/api/v1/admin/download-artifacts/{id}/presign-get` | Presign S3 GET |
| **Assets Admin** | POST | `/api/v1/admin/download-assets/apply` | Link artifact to asset (admin or service auth) |
| | POST | `/api/v1/admin/download-assets/set-current` | Promote artifact as current (admin only) |

#### CLI Commands (16 download-related)

| Command | Purpose |
|---------|---------|
| `admin-download-apps-list/create/save/delete` | App CRUD |
| `admin-download-storage-get/update/test` | S3 config |
| `admin-download-artifacts-list/by-app/presign-upload/commit/presign-get` | Artifact ops |
| `admin-download-assets-apply/set-current` | Asset linking |
| `admin-downloads-upload-managed` | Upload helper workflow |
| `downloads` | User download request |

#### Database Schema (4 tables)

| Table | Key Fields | Purpose |
|-------|------------|---------|
| `download_apps` | `bundle_key`, `app_key`, `name`, `update_api_key` | App identity |
| `download_assets` | `(bundle_key, app_key, platform, variant_key)` → `release_version`, `artifact_id`, `requires_entitlement` | Release per platform/channel |
| `download_artifacts` | `sha256`, `sha512`, `size_bytes`, `bucket`, `object_key`, `platform`, `release_version` | S3 object metadata |
| `download_storage_settings` | `bundle_key` → S3 config, credentials, TTL | Storage provider config |

---

### scenario-to-cloud (Desktop-Adjacent Surface)

#### Desktop-Relevant API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/deployments/{id}/health` | Comprehensive health (DNS, TLS, process, freshness) |
| GET | `/api/v1/deployments/{id}/drift` | Configuration drift detection |
| POST | `/api/v1/deployments/{id}/execute` | Execute deployment |
| POST | `/api/v1/deployments/{id}/start/stop` | Lifecycle control |
| Various | `/api/v1/secrets/*` | Secrets management (local + VPS) |
| Various | `/api/v1/bundles/*` | Bundle build/list/cleanup |

**Freshness mechanism:** `evaluateDeploymentFreshness()` compares deployed `scenario.ref` with local `service.json` version and bundle SHA256 fingerprint.

**Desktop relevance:** scenario-to-cloud deploys LPBS itself. The `landing-page-desktop-upload` skill uses scenario-to-cloud to ensure LPBS is deployed and healthy before uploading desktop artifacts.

#### CLI Commands (Desktop-Relevant)

| Command | Purpose |
|---------|---------|
| `deployment health --domain <d> --scenario <s>` | Health check (used by upload skill) |
| `redeploy --domain <d> --scenario <s> --if-needed --preflight --wait` | Converge deployment |
| `secrets verify/set/get/delete` | Secret management (LPBS_SERVICE_SECRET) |

---

### Prompt-Manager Skills (3 Desktop Release Skills)

#### `scenario-to-desktop` skill
**Purpose:** CLI reference for building desktop apps.
**Key commands encoded:** `pipeline run`, `deploy-target add/test/doctor`, `download`, `telemetry`, `signing`
**No state** — purely instructional.

#### `landing-page-deploy-setup` skill
**Purpose:** Prepare LPBS control plane for desktop artifact deployment.
**Gates (idempotent):**
- A: Local LPBS running
- B: Remote LPBS health-checked
- C: Local admin session active
- D: S3 storage configured + reachable
- E: Download app exists (optional)
- F: Remote profile created + logged in + tested
- G: `LPBS_SERVICE_SECRET` synchronized
**Key commands encoded:** `landing-page-business-suite deploy-readiness`, `admin-login`, `admin-download-storage-*`, `remote-profiles-*`, `scenario-to-cloud secrets set`

#### `landing-page-desktop-upload` skill
**Purpose:** End-to-end desktop release orchestration.
**Stages:**
- -0: Input discovery (profile tags, app keys)
- 0: Tool health checks (scenario-to-cloud, LPBS, scenario-to-desktop)
- 0.5: Build prerequisites (fpm/Ruby for Linux)
- 1: LPBS deployment convergence (via scenario-to-cloud)
- 2: LPBS readiness (via landing-page-deploy-setup / `deploy-readiness`)
- 3: Build + deploy (via `scenario-to-desktop pipeline run --deploy-to`)
- 4: Post-release verification (curl update manifests)
**Key commands encoded:** `scenario-to-cloud deployment health`, `redeploy`, `landing-page-business-suite deploy-readiness`, `scenario-to-desktop pipeline run`

## System Contracts: Interface Boundaries

### Contract 1: deployment-manager

**Owns (source of truth):**
- Deployment profiles (tier, config, version history)
- Approval state (per profile × platform × commit)
- Release gate decisions
- Visual validation records
- Deployment execution logs
- Cross-platform build orchestration
- Code signing configuration (per profile)

**Exposes (APIs others depend on):**
- `GET /api/v1/profiles/{id}/release-gate?commit={hash}` — gate check for automation
- `POST /api/v1/deploy-desktop` — orchestrated desktop deployment
- `POST /api/v1/build` — cross-compilation
- Profile CRUD for external tooling

**Consumes (APIs it calls):**
- scenario-to-desktop: `POST /api/v1/pipeline/run` (via desktop_client.go)
- None from LPBS directly (goes through scenario-to-desktop)

**Assumes (implicit contracts):**
- scenario-to-desktop is reachable at a discoverable URL
- scenario-to-desktop's pipeline stages will call LPBS for publishing
- Build artifacts from `/api/v1/build` are compatible with scenario-to-desktop's bundle stage

---

### Contract 2: scenario-to-desktop

**Owns (source of truth):**
- Pipeline state (stage progression, build provenance, platform results)
- Deploy targets (LPBS connection configs)
- Desktop app records (generated wrappers)
- Telemetry data (deployment telemetry from Electron apps)
- Scenario state (runtime state for preflight)

**Exposes (APIs others depend on):**
- `POST /api/v1/pipeline/run` — start build+deploy pipeline (called by deployment-manager)
- `GET /api/v1/pipeline/{id}` — pipeline status (polled by deployment-manager)
- `GET /api/v1/desktop/download/{scenario}/{platform}` — download built packages

**Consumes (APIs it calls):**
- LPBS: `POST /admin/download-artifacts/presign-upload` — get S3 upload URL
- LPBS: `POST /admin/download-artifacts/commit` — register artifact
- LPBS: `POST /admin/download-assets/apply` — link artifact to asset
- LPBS: `POST /admin/download-assets/set-current` — promote to current
- LPBS: Remote profile test (session validation)
- agent-manager: Task creation for investigate/fix

**Assumes (implicit contracts):**
- LPBS service auth (`LPBS_SERVICE_SECRET`) matches between both runtimes
- Deploy target's remote profile has an active session on local LPBS
- Download app with the specified `app_key` exists on remote LPBS
- LPBS S3 storage is configured and reachable

---

### Contract 3: LPBS (Download Surface)

**Owns (source of truth):**
- App registry (`download_apps`: bundle_key, app_key, name, update_api_key)
- Release state per platform/channel (`download_assets`: version, notes, artifact link, entitlement flag)
- Artifact metadata (`download_artifacts`: SHA256/512, size, S3 location, platform)
- S3 storage configuration (`download_storage_settings`)
- Update manifests (dynamically generated from artifact metadata)
- Entitlement gating (subscription check for downloads)

**Exposes (APIs others depend on):**
- `POST /admin/download-artifacts/presign-upload` — S3 upload (service auth)
- `POST /admin/download-artifacts/commit` — register artifact (service auth)
- `POST /admin/download-assets/apply` — link artifact (service auth)
- `POST /admin/download-assets/set-current` — promote current (admin)
- `GET /api/v1/updates/{app_key}/{channel}/{file}` — electron-updater manifests (public/API key)
- `GET /api/v1/downloads` — user download with entitlement check

**Consumes:**
- S3-compatible storage (MinIO/AWS) for artifact storage
- No calls to other Vrooli scenarios

**Assumes (implicit contracts):**
- `LPBS_SERVICE_SECRET` is shared with scenario-to-desktop for service auth
- S3 storage is pre-configured before artifacts can be uploaded
- Remote profile sessions are maintained by the calling system (scenario-to-desktop or agent)

---

### Contract 4: scenario-to-cloud

**Owns (source of truth):**
- VPS deployment records (status, manifest, bundle hash, SSH identity)
- Deployment freshness (version comparison, bundle fingerprint)
- VPS health state (DNS, TLS, process, resource metrics)
- Secrets management (local + VPS-side)
- SSH key management

**Exposes (APIs desktop pipeline depends on):**
- `GET /api/v1/deployments/{id}/health` — LPBS health check (used by upload skill)
- `POST /api/v1/deployments/{id}/execute` — deploy/redeploy LPBS (used by upload skill)
- Secrets API — `LPBS_SERVICE_SECRET` management

**Consumes:**
- VPS targets via SSH
- No calls to scenario-to-desktop or LPBS directly

**Desktop role:** Ensures LPBS is deployed, healthy, and has the correct secrets before desktop artifact upload begins.

---

### Contract 5: Prompt-Manager Skills

**Own:** Nothing (stateless instruction sets)

**Encode:**
- The correct sequence of CLI commands across all 4 scenarios
- Prerequisite gates and health checks
- Error recovery heuristics
- Post-release verification steps

**Depend on:**
- All 4 scenario CLIs being installed and accessible
- `--auto-start` flag for lazy scenario startup
- `api-core/discovery` for URL resolution between scenarios

## Findings: Gaps and Implicit Contracts

### Implicit Contracts (should be made explicit)

1. **LPBS_SERVICE_SECRET synchronization** — scenario-to-desktop must have the same secret as LPBS runtime. The `landing-page-deploy-setup` skill handles this manually, but there's no API contract or validation endpoint.

2. **Deploy target configuration** — scenario-to-desktop stores deploy targets in `.vrooli/deploy-targets.json`. These reference LPBS remote profile tags but are managed independently from LPBS's remote profile registry.

3. **Version propagation** — When scenario-to-desktop publishes v1.2.3 to LPBS, deployment-manager has no callback to record "this profile's latest published version is 1.2.3."

4. **Build provenance → LPBS linkage** — scenario-to-desktop tracks git commit in `BuildProvenance`, but this is not forwarded to LPBS. The `download_artifacts` table has no git commit field.

5. **Artifact source auth split** — LPBS `presign-upload` and `commit` accept either admin or service auth, but `set-current` is admin-only. scenario-to-desktop's deploy stage uses service auth, meaning it can upload and link but cannot promote without also having admin credentials or using `apply` instead (which also accepts service auth).

### Gaps (no API/automation support)

1. **No webhook from LPBS on publish** — After an artifact is published, nothing notifies deployment-manager or other systems.

2. **No "current live version" query** — To determine what's live, you must query LPBS directly. deployment-manager tracks deployment attempts but not post-publish state.

3. **No rollback API** — To rollback a release, you must manually `set-current` a prior artifact in LPBS. No single "rollback" action exists across the pipeline.

4. **No approval → deploy automation** — Approvals and deploys are separate actions. No automation triggers deploy when all approvals pass.

5. **Entitlement ↔ approval disconnect** — LPBS has `requires_entitlement` per asset (subscription gating) independent from deployment-manager approvals (release gating). These serve different purposes but could confuse operators.

6. **No git commit in LPBS artifacts** — Cannot trace from a published artifact back to a git commit without going through scenario-to-desktop's pipeline state (which is file-based and transient).

7. **No deployment-manager → LPBS version sync** — After a successful deploy, deployment-manager records success in its `deployments` table but has no record of what version is now live on LPBS.

## Prioritized Gap Recommendations

| Priority | Gap | Recommendation | Effort | Impact |
|----------|-----|----------------|--------|--------|
| **P1** | No git commit in LPBS artifacts | Add `git_commit_hash` field to `download_artifacts` table; pass from scenario-to-desktop's `BuildProvenance` during `commit` | S | High — enables artifact→commit traceability |
| **P1** | No version sync back to deployment-manager | Add callback from scenario-to-desktop deploy stage to deployment-manager recording published version per platform | M | High — closes the feedback loop |
| **P2** | No rollback API | Add `POST /api/v1/rollback` to deployment-manager that calls LPBS `set-current` with a prior artifact + records the rollback event | M | Medium — currently requires manual multi-step process |
| **P2** | No publish webhook from LPBS | Add webhook/event on artifact publish; deployment-manager subscribes to update its state | M | Medium — enables reactive automation |
| **P3** | No approval→deploy automation | Add optional `auto_deploy_on_approval` flag to deployment profiles; when release gate passes, trigger deploy-desktop automatically | S | Medium — removes manual step |
| **P3** | LPBS_SERVICE_SECRET validation | Add `POST /api/v1/deploy-targets/{name}/verify-auth` that does a round-trip secret validation | S | Low — reduces setup debugging |
| **P4** | Entitlement ↔ approval confusion | Documentation fix: add clear callout that LPBS entitlements gate user downloads while deployment-manager approvals gate release publishing | XS | Low — documentation only |

## Implementation Strategy

This is a research item — the deliverable is this document itself, not code changes. The strategy is:

1. **Phase 1 (round 1):** Map ownership and data flow from codebase exploration ✓
2. **Phase 2 (round 2):** Build exhaustive action inventory, validate via code, draft system contracts ✓
3. **Phase 3 (round 3):** Finalize system contract, verify completeness against Definition of Done

## Testing Plan

Research validation:
- [x] Cross-reference each ownership claim against the actual database schema
- [x] Verify API endpoints exist and accept the documented parameters
- [x] Trace one complete release through code paths to confirm the data flow diagram
- [x] Validate skill instructions match current API contracts
- [ ] Peer review: have someone familiar with deployment-manager confirm approval flow accuracy
- [ ] Peer review: have someone familiar with LPBS confirm artifact publishing flow accuracy

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Codebase changes during audit | Findings become stale | Pin findings to current commit; re-validate before finalizing |
| Undocumented runtime behavior | Missing implicit contracts | Traced actual code paths, not just assumptions |
| Scope creep into implementation | Delays research completion | Strict separation: document gaps, don't fix them |
| Skill instructions drift from API | Skills encode stale commands | Cross-validated skill commands against current API routes |

## Non-goals / Prohibited Patterns

- Do NOT implement fixes for identified gaps (separate backlog items)
- Do NOT modify any scenario code as part of this research
- Do NOT create new APIs or database schemas

## Definition of Done

- [x] All 5 questions from the description are answered with evidence
- [x] Ownership map covers all state (approval, build, version, artifact, manifest)
- [x] End-to-end data flow diagram is validated
- [x] Action inventory matrix is complete (API, CLI, UI, skill per system)
- [x] Gaps and implicit contracts are cataloged
- [x] System contract is formalized with explicit interface boundaries
- [x] Prioritized recommendations for gap remediation included
- [ ] Final peer review pass
