# Glossary: git-control-tower

Domain vocabulary used across the API, CLI, UI, and docs.

## Git domain terms

- **Working tree** — files in the repo directory, modified or not.
  git-control-tower operates on a single working tree (the local Vrooli
  repo); multi-repo support is deferred.
- **Stage / index** — git's "staging area" between working tree and
  commit history. Files are *staged* (added to index) before being
  *committed*.
- **Hunk** — a contiguous range of changed lines within a file's diff.
  The diff viewer can stage / discard at file granularity (hunk-level
  is not yet exposed).
- **Untracked** — file that exists in the working tree but is not yet
  tracked by git (no `git add` ever run for it).
- **Conflict** — both working-tree and index have diverging changes for
  the same file region after a merge / rebase. See [REQ: OT-P1-003].
- **Behind / ahead** — local branch's commit count relative to its
  upstream. *Behind* means the upstream has commits the local branch
  doesn't; *ahead* is the inverse.
- **Blame** — line-by-line attribution of which commit introduced each
  line. The UI's "blame mode" filters history to commits touching one
  file.
- **Hotspot** — a file that's been modified across many recent commits;
  surfaced via [CODE: api/branch_service.go] hotspot calculations.

## Scenario-specific terms

- **Scope filter** — UI history filter by Vrooli scope (scenario or
  resource). Implemented in
  [CODE: ui/src/components/GitHistory.tsx#GitHistoryImpl].
- **Change group** — a read-only bucket of changed files shown in the Changes
  list. Groups resolve from manual rules first, declared contract targets
  second, and `Other` last.
- **Contract group** — a change group derived at read time from one target in
  `.vrooli/repo-contract.json`; it is never stored in grouping rules.
- **Target index** — the cached path-to-target lookup supplied by
  `packages/repo-contract-go`; it uses longest-root matching and a bounded
  ten-second freshness window.
- **Working set** — files currently changed in the working tree; used
  by the "working-set only" history filter.
- **Commit group** — a prefix-based grouping of commits sharing a
  conventional-commits scope (e.g. `feat(api):`); parsed by
  [CODE: ui/src/lib/commitGroup.ts].
- **Audited operation** — any mutating API call (stage, unstage,
  commit, discard, etc.) that gets persisted to the SQLite audit log
  via [CODE: api/audit_logger.go].
- **Scenario review** — the auditor pipeline that produces a structured
  quality report for a scenario directory. Surface:
  [CODE: api/auditor_client.go] + UI's `ScenarioReviewPanel`.

## React / UI terms

- **Profiler boundary** — a `<React.Profiler>` wrapper emitting
  user-timing entries when the perf-build is active. See
  [CODE: ui/src/lib/profiler.ts]; pattern documented in
  [docs/perf/2026-05-03-history-markdown-resize.md](../perf/2026-05-03-history-markdown-resize.md).
- **Imperative resize** — panel-resize implementation that writes to
  DOM `style` directly during drag, committing to React state only on
  mouseup. Avoids per-mousemove tree re-renders.
- **Variable-height virtualization** — list rendering with
  `@tanstack/react-virtual`'s `measureElement`, used in
  [CODE: ui/src/components/GitHistory.tsx#rowVirtualizer].
