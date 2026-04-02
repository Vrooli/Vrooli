# Research Conclusion: Define The Release Record And History Model

## Research Question
What should "version history" and "release records" mean as a unified concept across deployment-manager profile versions, build provenance, LPBS artifacts, update channels, manifests, and customer-visible release history? What are the canonical IDs, references, and schema shape that later implementation work should use?

## Summary
The release record is a new first-class entity owned by deployment-manager (DM). Each release is identified by a UUID minted at orchestration time, correlating a git commit, profile version snapshot, semantic version, channel, and per-platform artifact references into a single auditable event. LPBS stores the `release_id` on artifacts and assets as a foreign correlation key but does not own the record. Channels follow a defined ordering (nightly → beta → stable) with explicit promotion semantics. Release notes live on the release record; no separate changelog API is needed.

## Methodology
- **Round 1 dependency input:** Built on the completed `desktop-release-control-plane-audit` research (7 gaps mapped).
- **Round 1 schema analysis:** Read database schemas and Go types across deployment-manager, scenario-to-desktop, and LPBS. Identified 3 incompatible version ID schemes and the absence of a "release event" record.
- **Round 2 deep dive:** Full schema review of DM tables (profiles, profile_versions, deployments, deployment_approvals, visual_validations), LPBS tables (download_apps, download_assets, download_artifacts), and S2D pipeline types (BuildProvenance, Config, DeployConfig, DeployResult). Traced the complete data flow from pipeline trigger through S2D's 4-step LPBS upload (presign → S3 → commit → apply).

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

**LPBS compatibility:** The existing `channelToVariantKey()` mapping (`stable` → `default`) continues to work. New channels (`beta`, `nightly`) map to their own variant_keys. No LPBS schema change needed for channel storage.

### Finding 7: Proposed Release Record Schema (deployment-manager)

```sql
CREATE TABLE releases (
    id              TEXT PRIMARY KEY,          -- UUID minted by orchestrator
    profile_id      TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
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
```

**Indexes:**
```sql
CREATE INDEX idx_releases_profile_channel ON releases(profile_id, channel);
CREATE INDEX idx_releases_status ON releases(status);
CREATE INDEX idx_releases_commit ON releases(git_commit_hash);
CREATE INDEX idx_release_platforms_status ON release_platforms(status);
```

### Finding 8: LPBS Needs a release_id Correlation Column

To correlate LPBS artifacts back to DM release records, LPBS needs:

```sql
ALTER TABLE download_artifacts ADD COLUMN release_id TEXT;
CREATE INDEX idx_artifacts_release_id ON download_artifacts(release_id);
```

This is a lightweight foreign correlation key (not a foreign key constraint, since it references a different database). S2D's LPBS client upload flow (presign → S3 → commit → apply) would pass `release_id` in the commit step alongside the existing `git_commit_hash` and `release_version`.

### Finding 9: S2D Pipeline Data Flow Already Supports Extension

The S2D pipeline already carries `Config.Version` → `ReleaseVersion` and `Provenance.GitCommitHash` through to LPBS upload. Adding `release_id` requires:
1. DM passes `release_id` to S2D when triggering a pipeline (via `DeployDesktopRequest` or equivalent).
2. S2D carries it through `Config` or a new field on `DeployConfig`.
3. S2D's `UploadRequest` struct gains a `ReleaseID` field.
4. The LPBS commit step includes `release_id` in the request body.

The 4-step upload flow (presign → S3 → commit → apply) doesn't need structural changes — just an additional field passed through.

### Finding 10: Go Types for the Release Record

```go
// Release represents a versioned release event in deployment-manager.
type Release struct {
    ID                    string     `json:"id"`
    ProfileID             string     `json:"profile_id"`
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

## Limitations
- The `release_id` on LPBS is a correlation key, not a foreign key constraint. Consistency depends on the orchestrator correctly propagating the ID. A reconciliation job or webhook confirmation would strengthen this.
- The channel promotion model is designed for the current 3-channel case. If channels become user-definable or per-profile, the hardcoded ordering would need to become configurable.
- Runtime behavior of the S2D → LPBS upload has not been verified end-to-end; the analysis is code-based.
- The `promoted_from_release_id` chain could grow long for releases that promote through all channels. Queries that need the full promotion chain would need recursive CTEs.

## Actions

### Action 1: Create `releases` and `release_platforms` Tables in DM
Add migration `004_add_releases.sql` with the schema from Finding 7. Add Go types, repository, and handler scaffolding.

### Action 2: Add `release_id` Column to LPBS `download_artifacts`
Add migration to LPBS for the `release_id` TEXT column and index (Finding 8).

### Action 3: Extend S2D Pipeline to Carry `release_id`
Add `ReleaseID` field to S2D's `Config`/`DeployConfig`, `UploadRequest`, and the LPBS client commit step (Finding 9).

### Action 4: Implement Channel Promotion Logic in DM
Add promotion endpoint and validation: only allow promotion up the channel ladder, create new release record with `promoted_from_release_id`, enforce approval gates for stable promotion.

### Action 5: Build Release History API in DM
Endpoints: list releases by profile (with channel/status filters), get release detail with platform statuses, get promotion chain for a release.

### Action 6: Wire Release Record into Orchestration Flow
DM orchestrator mints UUID and creates `releases` row at pipeline start, updates `release_platforms` as S2D reports per-platform results, sets status to `published` when all platforms complete (or `failed` on error).
