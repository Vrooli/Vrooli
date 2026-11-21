// Package baselinefloor is the platform recovery floor for Baseline Modes — the
// trusted base that lets Vrooli develop and validate any scenario while its
// last-known-good version keeps running, and recover it when an edit goes wrong.
//
// It is deliberately small, dependency-light, and lives in platform internal/ so
// it can recover even the scenarios that power self-improvement (git-control-tower,
// test-genie, ...). No scenario imports it; the engagement workflow
// (git-control-tower baseline start/check/promote/abandon) shells out to the
// vrooli commands that wrap it. See the Baseline Modes plan, P2 ("Control-plane
// layering" / "Engagement-state persistence (the floor owns it, NOT GCT)").
//
// This package provides the trusted-base primitives:
//
//   - The restore-point copy ladder (CopyTree, Capture, Restore): a
//     cross-platform, git-free code-level undo. It prefers a copy-on-write
//     reflink fast path where the filesystem supports it and falls back to a
//     native-Go deep copy everywhere else; it NEVER hardlinks (in-place edits
//     would corrupt the snapshot). A restore point holds source and
//     configuration only. Build artifacts and outputs (node_modules, dist,
//     coverage, .git, generated, binaries) and runtime data (databases,
//     logs, caches) are excluded: the IgnoreMatcher applies the repository's
//     .gitignore files without git, plus the scenario's declared build
//     outputs and data directories and the floor patterns. The scenario's
//     own .vrooli configuration (service.json and similar) is kept because
//     builds read it; .vrooli runtime state (secrets, locks, local
//     overrides, state/, runtime/) is not. Data rollback belongs to
//     data-backup-manager, never to a restore point.
//
//   - The capture record and shared-source snapshot (CaptureRecord,
//     CaptureSharedSources): each capture gets a capture ID, and the repo
//     sources the scenario build reaches outside its own directory (Go
//     replace closure, pnpm file: dependencies, bundler aliases) are copied
//     with their generated outputs into the engagement, so an engaged live
//     instance builds against the shared packages it was accepted with.
//
//   - The serving tree (MaterializeServing): a generation-addressed,
//     repo-shaped tree around the frozen sources that an engaged live
//     instance builds and runs from. Frozen roots are real copies; every
//     other repository entry is a symlink into the repository.
//
//   - The floor-owned engagement manifest (Manifest, Store): the
//     recovery-critical engagement intent (scenario, variant, mode, restore-point
//     path, TTL, ...) persisted as an engagement.json file co-located with the
//     restore point under ~/.cache/vrooli/<scenario>/baseline-<slug>/. It is
//     floor-owned (not inside git-control-tower) precisely so the floor can roll
//     back a broken git-control-tower; status is a glob over sibling manifests,
//     so there is no central index to corrupt.
//
//   - The schema migration runner (LoadScripts, RunMigrations): the promote
//     step's transactional schema runner. Ordered *.sql scripts authored into an
//     engagement's managed migrations/ folder are validated against a throwaway
//     copy of the current database and then applied to live in a single
//     transaction, tracked so a re-run is a no-op and an edit-after-apply is
//     rejected. "No scripts authored" is the shape-unchanged fast path (DB
//     handling skipped). v1 implements SQLite only; a non-SQLite engine is a
//     surfaced error so promote bounces to live mode rather than risk corruption.
//
// The running-process truth (ports/owner/heartbeat) stays in the scenarioruntime
// registry (P1). The two stores are distinct: registry == running-process truth,
// manifest == engagement intent.
package baselinefloor
