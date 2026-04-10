# Research Conclusion: Define The Release Record And History Model

## Research Question
What should "version history" and "release records" mean as a unified concept across deployment-manager profile versions, build provenance, LPBS artifacts, update channels, manifests, and customer-visible release history? What are the canonical IDs, references, and schema shape that later implementation work should use?

## Summary
The release record is a new first-class entity owned by deployment-manager (DM). Each release is identified by a UUID minted at orchestration time, correlating a git commit, profile version snapshot, semantic version, channel, and per-platform artifact references into a single auditable event. LPBS stores the `release_id` on artifacts and assets as a foreign correlation key but does not own the record. Channels follow a defined ordering (nightly → beta → stable) with explicit promotion semantics. Release notes live on the release record; no separate changelog API is needed. Each release links back to the deployment that triggered it. Approvals are commit-scoped — one approval per (profile, commit, platform) covers all channels. Only publishing deployments create release records; dry runs and validation-only runs do not. Previous releases are marked "superseded" by application-level logic when a new release publishes to the same profile and channel.

## Methodology
- **Round 1 dependency input:** Built on the completed `desktop-release-control-plane-audit` research (7 gaps mapped).
- **Round 1 schema analysis:** Read database schemas and Go types across deployment-manager, scenario-to-desktop, and LPBS. Identified 3 incompatible version ID schemes and the absence of a "release event" record.
- **Round 2 deep dive:** Full schema review of DM tables (profiles, profile_versions, deployments, deployment_approvals, visual_validations), LPBS tables (download_apps, download_assets, download_artifacts), and S2D pipeline types (BuildProvenance, Config, DeployConfig, DeployResult). Traced the complete data flow from pipeline trigger through S2D's 4-step LPBS upload (presign → S3 → commit → apply).
- **Round 3 verification:** Cross-referenced the proposed schema against actual codebase. Verified DM migration numbering (next: 004), confirmed `DeployDesktopRequest` struct fields (orchestrator.go:24-55), validated `CheckReleaseGate` logic (approvals_repository.go:183-224), and confirmed LPBS `UpsertAsset` hardcoded variant_key (download_service.go:596). Identified schema drift, pipeline extension points, and approval/channel interaction gaps. Resolved three remaining decisions: commit-scoped approvals, dry-run exclusion, and application-level superseded status.

## Findings

### Finding 1: Three Incompatible Version ID Schemes Coexist

| System | Version ID | Type | Scope |
|--------|-----------|------|-------|
| deployment-manager | `profiles.version` | Auto-incrementing integer | Internal config versioning (profile snapshots) |
| scenario-to-desktop | `BuildProvenance.GitCommitHash` + `Config.Version` | SHA-1 hash + semver string | Build-time provenance |
| LPBS | `download_artifacts.release_version` + `download_assets.release_version` | Semver string (VARCHAR 50) | Customer-facing release identifier |

**Resolution (decided round 1, d2→B):** A UUID minted at release time by the DM orchestrator becomes the canonical cross-system release identifier. All three existing ID schemes remain in their systems but are correlated through this UUID.

### Finding 2: "Profile Version" and "Release Version" Are Fundamentally Different Concepts

- **Profile version** (deployment-manager): Integer counter tracking config changes. Increments on every profile edit.
- **Release version** (LPBS): Semver string (e.g., "1.2.3") attached to published artifacts. Customer-facing.
- **Build version** (scenario-to-desktop): Semver from `.vrooli/service.json`, stamped into `BuildProvenance`. Becomes `release_version` when uploaded to LPBS.

These are three distinct lifecycle tracks. The release record links them without conflating them.

### Finding 3: Release History Is Implicitly Stored but Not Explicitly Queryable

LPBS stores historical artifacts (multiple per app/platform with `is_current` flags) but has no API to list all versions ever published. DM's `profile_versions` tracks config history, not release history. S2D's pipeline state is transient.

**Resolution:** The DM `releases` table becomes the canonical release history. It records each release event with timestamp, actor, channel, version, and per-platform status.

### Finding 4: The Channel/Variant Model Is Underspecified

LPBS maps channels to `variant_key` via a simple pass-through (`stable`/empty → `default`, others → as-is). No ordering, promotion rules, or channel relationship exists.

**Resolution (decided round 1, d3→A):** Channels get a formal ordering model. See Finding 6 for the proposed design.

### Finding 5: No "Release Event" Record Exists Anywhere

None of the three systems record the full event: "version X.Y.Z released to channel C for platform P at time T, from commit H, approved by user U."

**Resolution (decided round 1, d1→A):** DM owns the release record. See proposed schema below.

### Finding 6: Proposed Channel Model

**Channel ordering** (lowest → highest stability):

| Channel | variant_key (LPBS) | Stability | Promotion Target |
|---------|-------------------|-----------|-----------------|
| `nightly` | `nightly` | Lowest — automated builds, no approval required | `beta` |
| `beta` | `beta` | Middle — requires approval, used for testing | `stable` |
| `stable` | `default` | Highest — full approval gate, customer-facing | (terminal) |

**Promotion semantics:**
- A release can only promote **up** the stability ladder (nightly → beta → stable).
- Promotion creates a new release record with `promoted_from_release_id` pointing to the source release.
- The promoted release inherits the same git commit, artifacts, and version but targets a higher channel.
- Promotion to `stable` requires all `profile_required_platforms` to be approved (existing DM gate).
- Each channel can have at most one "current" release per (profile, platform) at a time.

**LPBS compatibility:** The existing `channelToVariantKey()` mapping (`stable` → `default`) continues to work. New channels (`beta`, `nightly`) map to their own variant_keys. No LPBS schema change needed for channel storage — but `UpsertAsset` must stop hardcoding `variant_key = 'default'` and accept it as a parameter (see Finding 11).

### Finding 7: Proposed Release Record Schema (deployment-manager)

```sql
-- Migration: 004_add_releases.sql

CREATE TABLE releases (
    id              TEXT PRIMARY KEY,          -- UUID minted by orchestrator
    profile_id      TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    deployment_id   TEXT REFERENCES deployments(id), -- links to the triggering deployment
    profile_version INTEGER NOT NULL,          -- snapshot reference (profiles.version at release time)
    git_commit_hash TEXT NOT NULL,
    release_version TEXT NOT NULL,             -- semver (e.g., "1.2.3")
    channel         TEXT NOT NULL DEFAULT 'stable',  -- nightly, beta, stable
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, publishing, published, failed, superseded
    release_notes   TEXT,
    released_by     TEXT,                      -- who triggered the release
    promoted_from_release_id TEXT REFERENCES releases(id), -- NULL if original, set if promoted
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ,              -- when status became 'published'
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, git_commit_hash, channel)
);

CREATE TABLE release_platforms (
    release_id  TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    platform    TEXT NOT NULL,                -- windows, mac, linux
    status      TEXT NOT NULL DEFAULT 'pending', -- pending, building, uploading, published, failed
    approval_id TEXT REFERENCES deployment_approvals(id),
    lpbs_artifact_id INTEGER,                 -- references LPBS download_artifacts.id (cross-system)
    published_at TIMESTAMPTZ,
    error       TEXT,
    PRIMARY KEY (release_id, platform)
);

CREATE INDEX idx_releases_profile_channel ON releases(profile_id, channel);
CREATE INDEX idx_releases_status ON releases(status);
CREATE INDEX idx_releases_commit ON releases(git_commit_hash);
CREATE INDEX idx_releases_deployment ON releases(deployment_id);
CREATE INDEX idx_release_platforms_status ON release_platforms(status);
```

**Key design notes:**
- `deployment_id` links each release to the triggering deployment (round 2, d1→A). The deployment holds execution details (logs, timing, error); the release holds semantic details (version, channel, per-platform status). Promoted releases may have `deployment_id = NULL` if promotion doesn't trigger a new pipeline run.
- Only publishing deployments (not dry-run, not skip-packaging) create release records (round 3, d2→A). The orchestrator checks `DryRun` and `SkipPackaging` flags before minting a release UUID.

### Finding 8: LPBS Needs a release_id Correlation Column

To correlate LPBS artifacts back to DM release records, LPBS needs:

```sql
ALTER TABLE download_artifacts ADD COLUMN release_id TEXT;
CREATE INDEX idx_artifacts_release_id ON download_artifacts(release_id);
```

This is a lightweight foreign correlation key (not a foreign key constraint, since it references a different database). S2D's LPBS client upload flow (presign → S3 → commit → apply) would pass `release_id` in the commit step alongside the existing `git_commit_hash` and `release_version`.

### Finding 9: S2D Pipeline Data Flow Already Supports Extension

The S2D pipeline already carries `Config.Version` → `ReleaseVersion` and `Provenance.GitCommitHash` through to LPBS upload. Adding `release_id` requires:
1. DM passes `release_id` and `channel` to S2D when triggering a pipeline (via `DeployDesktopRequest` — needs new `ReleaseID` and `Channel` fields added to the struct at orchestrator.go:24-55).
2. S2D carries them through `Config` or a new field on `DeployConfig`.
3. S2D's `UploadRequest` struct gains `ReleaseID` and `Channel` fields.
4. The LPBS commit step includes `release_id` in the request body, and uses `channelToVariantKey(channel)` to set the correct variant_key.

The 4-step upload flow (presign → S3 → commit → apply) doesn't need structural changes — just additional fields passed through.

### Finding 10: Go Types for the Release Record

```go
// Release represents a versioned release event in deployment-manager.
type Release struct {
    ID                    string     `json:"id"`
    ProfileID             string     `json:"profile_id"`
    DeploymentID          string     `json:"deployment_id,omitempty"`
    ProfileVersion        int        `json:"profile_version"`
    GitCommitHash         string     `json:"git_commit_hash"`
    ReleaseVersion        string     `json:"release_version"`
    Channel               string     `json:"channel"`
    Status                string     `json:"status"`
    ReleaseNotes          string     `json:"release_notes,omitempty"`
    ReleasedBy            string     `json:"released_by,omitempty"`
    PromotedFromReleaseID string     `json:"promoted_from_release_id,omitempty"`
    CreatedAt             time.Time  `json:"created_at"`
    PublishedAt           *time.Time `json:"published_at,omitempty"`
    UpdatedAt             time.Time  `json:"updated_at"`
    Platforms             []ReleasePlatform `json:"platforms,omitempty"`
}

type ReleasePlatform struct {
    ReleaseID      string     `json:"release_id"`
    Platform       string     `json:"platform"`
    Status         string     `json:"status"`
    ApprovalID     string     `json:"approval_id,omitempty"`
    LPBSArtifactID int64      `json:"lpbs_artifact_id,omitempty"`
    PublishedAt    *time.Time `json:"published_at,omitempty"`
    Error          string     `json:"error,omitempty"`
}

// Channel constants
const (
    ChannelNightly = "nightly"
    ChannelBeta    = "beta"
    ChannelStable  = "stable"
)

// ChannelOrder defines promotion ordering. Lower index = less stable.
var ChannelOrder = []string{ChannelNightly, ChannelBeta, ChannelStable}

// Release status constants
const (
    ReleaseStatusPending    = "pending"
    ReleaseStatusPublishing = "publishing"
    ReleaseStatusPublished  = "published"
    ReleaseStatusFailed     = "failed"
    ReleaseStatusSuperseded = "superseded"
)
```

### Finding 11: LPBS UpsertAsset Hardcodes variant_key (Verified Round 3)

`UpsertAsset` at download_service.go:596 hardcodes `'default'` in the INSERT VALUES clause. For multi-channel support:
- Add `VariantKey string` field to the `DownloadAsset` struct.
- Replace the hardcoded `'default'` with the parameterized value (`$12` instead of literal).
- S2D's upload flow passes `channelToVariantKey(channel)` when constructing the asset.
- The `channelToVariantKey()` function (update_handlers.go:50-55) already handles the mapping correctly and can be reused or moved to a shared location.

### Finding 12: DeployDesktopRequest Needs Release Context Fields (Verified Round 3)

The `DeployDesktopRequest` struct (orchestrator.go:24-55) currently has 15 fields but no release or channel awareness. Two new fields are needed:

```go
// ReleaseID is the UUID of the release record created for this deployment.
// Empty for dry runs and non-publishing deployments.
ReleaseID string `json:"release_id,omitempty"`
// Channel is the target release channel (nightly, beta, stable).
// Defaults to "stable" if empty.
Channel string `json:"channel,omitempty"`
```

These flow through the pipeline into S2D and ultimately into the LPBS upload step.

### Finding 13: Approval Model Is Commit-Scoped, Not Channel-Scoped

The `deployment_approvals` table has a unique constraint on `(profile_id, git_commit_hash, platform)` — no channel dimension. The `CheckReleaseGate` function (approvals_repository.go:183-224) checks that ALL required platforms have status "approved" without considering channels.

**Resolution (decided round 3, d1→A):** Approvals remain commit-scoped. A single approval per (profile, commit, platform) covers all channels. This means:
- Once a commit is approved for a platform, it can be promoted to any channel without re-approval.
- The `deployment_approvals` table needs no schema changes.
- `CheckReleaseGate` needs no channel awareness — it continues to check that all required platforms are approved.
- The nightly gate bypass (round 2, d3→A) is implemented at the orchestrator level: the orchestrator simply skips the `CheckReleaseGate` call for nightly releases.
- Stricter per-channel review policies, if ever needed, would require a future schema extension.

### Finding 14: Verification Paths Identified (Round 3)

Four concrete verification strategies for validating the release record implementation:

1. **End-to-end data flow test:** Trace a mock release from DM orchestrator → S2D pipeline → LPBS upload, asserting `release_id` appears at each hop (DM releases table, S2D request, LPBS download_artifacts).
2. **Schema integration test:** Using the same testcontainers pattern as LPBS (postgres:15-alpine + `setupTestDB(t)`), verify migrations 004 apply cleanly, FK constraints work, and unique constraints enforce correctly.
3. **Promotion chain test:** Create nightly → beta → stable releases and verify `promoted_from_release_id` links form a valid chain. Assert that promotion only goes up the stability ladder.
4. **Channel-gate interaction test:** Verify that `CheckReleaseGate` is skipped for nightly, required for beta (at least one approval), and requires all-platform approval for stable.

### Finding 15: Only Publishing Deployments Create Release Records

**Resolution (decided round 3, d2→A):** The orchestrator checks `DryRun` and `SkipPackaging` flags before creating a release record. Only deployments that intend to publish (both flags false) mint a release UUID and insert into the `releases` table. This keeps release history clean — dry runs and validation-only runs are tracked solely in the `deployments` table. The `releases.deployment_id` FK ensures traceability back to the deployment that triggered any given release.

### Finding 16: Superseded Status Is Set by Application Logic

**Resolution (decided round 3, d3→A):** When a new release reaches `published` status for a given (profile, channel), the orchestrator sets the previous `published` release for that same (profile, channel) to `superseded`. This is application-level logic in the orchestrator, not a database trigger. The pattern matches LPBS's existing `is_current` flag management. The supersede step is:
1. Query for the currently `published` release on (profile_id, channel) where `id != new_release_id`.
2. If found, update its status to `superseded` and set `updated_at`.
3. This runs in the same transaction as the new release's status update to `published`, ensuring atomicity.

## Limitations
- The `release_id` on LPBS is a correlation key, not a foreign key constraint. Consistency depends on the orchestrator correctly propagating the ID. A reconciliation job or webhook confirmation would strengthen this.
- The channel promotion model is designed for the current 3-channel case. If channels become user-definable or per-profile, the hardcoded ordering would need to become configurable.
- Runtime behavior of the S2D → LPBS upload has not been verified end-to-end; the analysis is code-based.
- The `promoted_from_release_id` chain could grow long for releases that promote through all channels. Queries that need the full promotion chain would need recursive CTEs.
- Approvals are commit-scoped with no channel dimension. This simplifies the model but means there is no mechanism for requiring stricter review specifically for stable promotions. If that need arises, the approval model would need a channel column or a separate promotion-approval concept.
- The superseded status transition depends on correct orchestrator logic running in a transaction. If the orchestrator crashes between publishing the new release and superseding the old one, both could briefly show as `published`. Recovery logic or a periodic consistency check should be considered during implementation.

## Actions

### Action 1: Create backlog item — Add `releases` and `release_platforms` tables to DM
- **Kind**: execute
- **Title**: Add releases and release_platforms tables to deployment-manager
- **Description**: Create migration `004_add_releases.sql` with the schema from Finding 7. Add Go types (Finding 10), repository (CRUD + promotion + supersede logic), and handler scaffolding for release history API endpoints. The supersede step (Finding 16) must run in the same transaction as the publish status update.
- **Initiative**: desktop-release-governance
- **Priority**: high
- **Effort**: L

### Action 2: Create backlog item — Add `release_id` column to LPBS `download_artifacts`
- **Kind**: execute
- **Title**: Add release_id correlation column to LPBS download_artifacts
- **Description**: Add migration for the `release_id` TEXT column and index on `download_artifacts` (Finding 8). Update the artifact commit handler to accept and store `release_id` from S2D uploads.
- **Initiative**: desktop-release-governance
- **Priority**: high
- **Effort**: S

### Action 3: Create backlog item — Extend S2D pipeline to carry `release_id` and `channel`
- **Kind**: execute
- **Title**: Add release_id and channel to S2D pipeline data flow
- **Description**: Add `ReleaseID` and `Channel` fields to S2D's `Config`/`DeployConfig` and `UploadRequest` structs (Finding 9). Pass `release_id` in the LPBS commit step. Use `channelToVariantKey(channel)` to set the correct variant_key on upload.
- **Initiative**: desktop-release-governance
- **Priority**: high
- **Effort**: M

### Action 4: Create backlog item — Parameterize LPBS UpsertAsset variant_key
- **Kind**: fix
- **Title**: Remove hardcoded variant_key in LPBS UpsertAsset
- **Description**: Replace the hardcoded `'default'` at download_service.go:596 with a `VariantKey` field on `DownloadAsset` (Finding 11). This unblocks multi-channel releases.
- **Initiative**: desktop-release-governance
- **Priority**: high
- **Effort**: S

### Action 5: Create backlog item — Implement channel promotion logic in DM
- **Kind**: execute
- **Title**: Add channel promotion endpoint and validation to deployment-manager
- **Description**: Implement promotion endpoint: only allow promotion up the channel ladder, create new release record with `promoted_from_release_id`, enforce approval gates per channel policy (Finding 6). Nightly skips the gate entirely; beta and stable call `CheckReleaseGate`. Approvals are commit-scoped (Finding 13) — no re-approval needed when promoting, only verification that required approvals exist.
- **Initiative**: desktop-release-governance
- **Priority**: medium
- **Effort**: M

### Action 6: Create backlog item — Build release history API in DM
- **Kind**: execute
- **Title**: Add release history API endpoints to deployment-manager
- **Description**: Endpoints: list releases by profile (with channel/status filters), get release detail with platform statuses, get promotion chain for a release. Uses the schema from Finding 7.
- **Initiative**: desktop-release-governance
- **Priority**: medium
- **Effort**: M

### Action 7: Create backlog item — Wire release record into DM orchestration flow
- **Kind**: execute
- **Title**: Integrate release record creation into DM orchestrator pipeline
- **Description**: Orchestrator mints UUID and creates `releases` row at pipeline start — only for non-dry-run, non-skip-packaging deployments (Finding 15). Updates `release_platforms` as S2D reports per-platform results. Sets status to `published` when all platforms complete (or `failed` on error), and atomically supersedes the previous release for the same profile+channel (Finding 16). Adds `ReleaseID` and `Channel` to `DeployDesktopRequest` (Finding 12). Skips `CheckReleaseGate` for nightly releases (Finding 13).
- **Initiative**: desktop-release-governance
- **Priority**: high
- **Effort**: L
