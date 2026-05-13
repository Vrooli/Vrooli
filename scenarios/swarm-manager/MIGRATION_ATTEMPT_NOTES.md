# Reverted big-bang template migration — 2026-05-13

A prior attempt to migrate swarm-manager onto the new
`templates/scenarios/react-vite/` (ReactVeat 1.0.0) layout in a single
plan-driven pass was **reverted** on 2026-05-13.

## Why
Audit found the migration was hard to verify and left swarm-manager unable to
restart:

- Proto/Connect-RPC migration only ~5% applied (only `settings` wired through
  `MountConnect`); 13 of 14 API handlers still REST/gorilla-mux.
- UI `surfaces/graph/` → `features/graph/` restructure left ~96K LOC of
  deletions plus likely dangling imports.
- CLI domains (~18) misaligned with API/UI domains (~14).
- Required template artifacts missing
  (`.vrooli/service.json::generation` block, `docs/internal/SEAMS.md`,
  `PROBLEMS.md`, flow models, RESTException docs).
- Test files for the graph feature were deleted with no replacement.

## Where the reverted work lives
Full snapshot at:
**`~/swarm-manager-bigbang-archive-2026-05-13/`**

Contents:
- `tracked.patch` — `git diff HEAD --binary` over the 945 tracked paths
  (modifications + deletions). Reversibly applicable with `git apply`.
- `working-tree.tar.gz` — on-disk state of all 180 untracked paths
  (105 swarm-manager + 75 proto), 1298 files total.
- `NOTES.md` — full restore recipe and audit summary pointer.
- `paths-tracked.txt`, `paths-untracked.txt`, `scope-*.txt` — path lists.

HEAD SHA at snapshot time: `9559ed2f4039671084027d81f1b6caa10e549e4f`

## Recommended path forward
Redo the migration in **web-console style** — one domain at a time, each
committed and verified working before moving on. Phases:

1. Land proto + Connect-RPC infrastructure (`MountConnect`, generated
   clients) and migrate a single pilot domain end-to-end
   (`settings` is already done; pick `backlog` or `agents` next).
   Commit. Verify `vrooli scenario restart swarm-manager`.
2. Migrate remaining domains one at a time, each as its own commit with
   matching CLI + UI changes. Run the `screaming-architecture-audit` skill
   between domains.
3. UI graph `surfaces/` → `features/` restructure as its own dedicated
   commit, after Connect-RPC is stable across all domains.
4. Add the `generation` block to `.vrooli/service.json` and write
   `docs/internal/SEAMS.md` (REST exceptions) and `PROBLEMS.md` as part of
   Phase 1.

This file itself is uncommitted — delete it once the new migration is
underway or once the archive has been mined for whatever was useful.
