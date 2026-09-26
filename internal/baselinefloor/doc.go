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
//     would corrupt the snapshot). The exclude list is aligned to the repo
//     .gitignore so build artifacts (node_modules/dist/coverage/.git/.vrooli/
//     generated/.build-fingerprint.json) are skipped — the single biggest size
//     and speed win, and safe because excluded == git-ignored == regenerable.
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
