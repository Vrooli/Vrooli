# Workbench source-line ledger — 2026-11-21

This is a point-in-time inventory of the shared worktree, not a certification
that the strict net-negative gate passes. The canonical program remains
`WORKBENCH-PROGRAM.md`; this companion avoids replacing its delivery record.

## Measurement

Scope: `scenarios/react-component-library`; source extensions `.go`, `.ts`,
`.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.css`, `.scss`, `.sql`, and `.proto`.
Tracked changes are measured against `HEAD` with `git diff --numstat`; untracked
files are counted with `git ls-files --others --exclude-standard` and physical
newline counts. Generated code and tests are included. Binary assets, docs,
JSON, and build output are excluded. Because the shared worktree contains
concurrent unrelated changes, this is deliberately not represented as a
workbench-only attributable total.

| Current-tree measure | Lines |
|---|---:|
| Tracked additions | 15,004 |
| Tracked deletions | 25,614 |
| Tracked net | -10,610 |
| Changed or untracked source files | 568 |
| Untracked source files | 275 files / 37,269 lines |
| Combined net versus HEAD, including untracked files | +26,659 |

The last row is a gross source inventory, not a valid attribution of the
workbench change: untracked files include tests, generated sources, and
concurrent contributions. It does establish that the broad strict gate cannot
be claimed from tracked deletions alone. Conversely, the 11,204 tracked net
decrease is not proof of a pass because it nets unrelated additions and
deletions and does not account for untracked source.

## Production-only cross-check

A second inventory excludes test files and generated paths/files, and includes
tracked diff lines plus untracked current production lines. The latest shared
worktree measurement reports 291 changed or untracked production source files:
7,592 tracked additions plus 20,062 untracked lines, and 19,600 tracked
deletions, for a net **+8,054 production lines**. The scope includes
`scenarios/react-component-library` and `packages/react-component-library`.
This is not a strict-gate pass: concurrent shared-worktree edits remain mixed
in, and the additions/deletions cannot be attributed to the commissioned
workbench change. Tests, generated files, and non-source files are excluded.
The refreshed inventory is stored at
`~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-eight-screen-workspace-20261121/data/source-line-inventory-followup-2026-11-21.json`.

### Current shared-worktree snapshot — 2026-11-22

Using the same source scope and production-only exclusions, the refreshed
snapshot contains 211 changed tracked production source files: 9,115 tracked
additions and 22,257 tracked deletions. It also contains 159 untracked
production source files totaling 25,276 lines, for a combined **+12,134 lines
versus `HEAD`**. This is a current inventory of the shared worktree; concurrent
changes remain mixed in, so it does not establish an attributable RCL net
increase or decrease and does not close the strict reduction gate. The result
is recorded at
`~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-eight-screen-workspace-20261121/data/source-line-inventory-followup-2026-11-22.json`.

## Completed retirement ledger

The following deleted tracked owners have these `HEAD` line counts (deleted
file bodies measured with `git show HEAD:<path>`):

| Retired owner or slice | Removed tracked lines | Evidence / residual |
|---|---:|---|
| `api/internal/designs` | 1,455 | Transactional import into workspace aggregates and source-table retirement are covered by the workspace migration tests; see `WORKBENCH-LEGACY-OWNERS-AUDIT.md`. |
| `api/internal/templates` | 737 | Custom template rows migrate to stable workspace IDs; schema and source table retire in the same successful transaction. |
| Legacy `useStudio` hook and test | 1,176 | Workspace studio is mounted; legacy editor route is removed. |
| Browser-local template shelf implementation and tests | 950 | Shelf CRUD/sync removed. The read-only `readLegacyTemplate` bookmark bridge remains intentionally until the compatibility window and privacy-safe usage evidence allow retirement. |
| `ui/src/api/versionHistory.ts` | 55 | Unreferenced join helper and stale coverage exclusion removed, as recorded in canonical program. |
| Other deleted tracked source | 10,067 | Includes other migration/cleanup slices; not attributed here without a file-by-file owner and consumer audit. |

These rows document removals, not the corresponding replacement cost. A
strict owner-by-owner net ledger still needs a reviewed mapping from each
retired slice to its replacement additions, tests, generated contracts, and
remaining consumers. Do not compare only the 4,373 lines above against total
untracked files and call it a net-negative migration.

## Legacy owner audit outcome

The dedicated legacy-owner audit finds no clear, safe unused owner left to
delete in this pass. Remaining lookalikes have active obligations:

- `readLegacyTemplate` preserves unsynced browser drafts reachable from old
  component-context bookmarks; there is no usage telemetry yet.
- Sketch wire messages remain shared across the migrated Experience,
  WorkspaceCapture, Review, WorkspaceAssist, and WorkspaceCandidate contracts;
  only the old service/client owner was retired.
- Reference-asset server routes remain mounted; dead UI-client evidence does
  not prove those published endpoints are unused or retire their compatibility
  obligation.
- Catalog health, gate, readiness, and capture procedures remain live owners;
  the audit does not identify them as dead code.

Evidence and detailed consumer maps: `WORKBENCH-LEGACY-OWNERS-AUDIT.md`. The
remaining work is to produce a complete reviewed attribution ledger and close
the compatibility window with usage evidence, not to delete active code to
force a lower raw count.
