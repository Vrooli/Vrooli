# Visited Tracker Seams

## Last Updated
2026-02-04

## Key Seams

### 1. Visit/Exclude Target Resolution
**Purpose:** Normalize request targets (files + per-file notes), expand glob patterns, and expose unmatched patterns.

**Code:**
- `scenarios/visited-tracker/api/targets.go`

**Seam Mechanism:**
- `PathMatcher` interface allows substitution of glob behavior.
- `resolveTargetsWithMatcher` is the seam for tests and future adapters.

**Why it matters:**
- Centralizes glob expansion rules for visit/exclude.
- Enables unit tests to inject deterministic matcher results without filesystem setup.

### 2. Campaign File Normalization
**Purpose:** Consistent resolution of relative vs absolute file paths per campaign.

**Code:**
- `scenarios/visited-tracker/api/sync.go` (`getCampaignBaseDir`, `normalizeFilePath`)

**Why it matters:**
- Prevents path drift between stored file paths and incoming request paths.

### 3. Storage Boundary
**Purpose:** File-based persistence of campaigns and visits.

**Code:**
- `scenarios/visited-tracker/api/storage.go`
- `scenarios/visited-tracker/api/main.go` (health wiring)

**Why it matters:**
- Allows future swap to a database by keeping persistence calls centralized.
- Exposes `storageHealthCheck` as the health seam for dependency reporting without leaking filesystem logic into handlers.

## Notes
- Avoid bypassing `resolveTargets` in handlers; it is the canonical seam for visit/exclude target expansion.
