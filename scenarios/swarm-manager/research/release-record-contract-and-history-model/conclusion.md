# Research Conclusion: Define The Release Record And History Model

## Research Question
What should "version history" and "release records" mean as a unified concept across deployment-manager profile versions, build provenance, LPBS artifacts, update channels, manifests, and customer-visible release history? What are the canonical IDs, references, and schema shape that later implementation work should use?

## Summary
<!-- TBD — will be refined as workshop rounds progress -->

## Methodology
- **Dependency input:** Built on the completed `desktop-release-control-plane-audit` research, which mapped ownership, data flow, system contracts, and 7 gaps across the 4-system pipeline.
- **Schema analysis:** Read database schemas and Go types across deployment-manager (`profiles`, `profile_versions`, `deployment_approvals`), scenario-to-desktop (`BuildProvenance`, `Config`, `DeployResult`), and LPBS (`download_apps`, `download_assets`, `download_artifacts`).
- **Cross-system tracing:** Followed version identifiers through the full release flow to identify where IDs are created, transformed, or lost.

## Findings

### Finding 1: Three Incompatible Version ID Schemes Coexist

| System | Version ID | Type | Scope |
|--------|-----------|------|-------|
| deployment-manager | `profiles.version` | Auto-incrementing integer | Internal config versioning (profile snapshots) |
| scenario-to-desktop | `BuildProvenance.GitCommitHash` + `Config.Version` | SHA-1 hash + semver string | Build-time provenance |
| LPBS | `download_artifacts.release_version` + `download_assets.release_version` | Semver string (VARCHAR 50) | Customer-facing release identifier |

No system stores a composite "release record ID" that links all three. The git commit hash is the closest candidate for a cross-system correlation key (deployment-manager stores it in `deployment_approvals`, scenario-to-desktop captures it in `BuildProvenance`, and LPBS now has it in `download_artifacts.git_commit_hash`), but it is not used as a primary key anywhere.

### Finding 2: "Profile Version" and "Release Version" Are Fundamentally Different Concepts

- **Profile version** (deployment-manager): An integer counter tracking config changes — tiers, swaps, secrets, settings. Increments on every profile edit. Has nothing to do with customer-facing releases.
- **Release version** (LPBS): A semver string (e.g., "1.2.3") attached to a published artifact. This is what customers see in update manifests and download pages.
- **Build version** (scenario-to-desktop): The semver from `.vrooli/service.json`, stamped into `BuildProvenance`. Becomes the `release_version` when uploaded to LPBS.

These are three distinct lifecycle tracks that happen to use the word "version." A unified model must not conflate them.

### Finding 3: Release History Is Implicitly Stored but Not Explicitly Queryable

LPBS stores multiple artifacts per `(bundle_key, app_key, platform)` with `is_current` flags, so historical artifacts exist in the database. However:
- There is no API to list all versions ever published for an app (only `ListArtifactsByApp` which returns artifacts with `is_current` flags)
- deployment-manager's `profile_versions` tracks config history, not release history
- scenario-to-desktop's pipeline state is file-based and transient — once a pipeline completes, provenance is only preserved if captured by LPBS

### Finding 4: The Channel/Variant Model Is Underspecified

- LPBS has `variant_key` (default: `"default"`) on `download_assets`, and the update manifest URL uses `{channel}` where `"stable"` maps to `variant_key="default"`
- deployment-manager has no channel concept — approval is per `(profile, commit, platform)`
- scenario-to-desktop passes `channel` through to LPBS but doesn't model it internally
- There's no definition of what channels should exist, how they relate to each other (e.g., can a "beta" release promote to "stable"?), or who controls channel promotion

### Finding 5: No "Release Event" Record Exists Anywhere

None of the three systems record the event of "we released version X.Y.Z to channel C for platform P at time T, from commit H, approved by user U." The closest is:
- deployment-manager: `deployments` table records execution logs but not the resulting published version
- scenario-to-desktop: Pipeline status has `FinalArtifacts` but is transient
- LPBS: `download_artifacts.created_at` records when an artifact was uploaded, and `download_assets.updated_at` records when a release pointer changed

A proper release event record would unify these into a single auditable entry.

## Limitations
- This is round 1; the schema shape and canonical IDs are not yet defined — they depend on decisions about where the release record should live and what it should contain.
- The analysis is code-based; runtime behavior around version propagation has not been verified.
- The `git_commit_hash` column in LPBS `download_artifacts` was added via migration but its population path through the pipeline has not been verified end-to-end.

## Actions
<!-- TBD — actions will be defined once the canonical schema and ownership decisions are settled through workshop rounds -->
