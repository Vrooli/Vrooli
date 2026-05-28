# Storage Classes

prompt-manager writes to three distinct filesystem roots, one per **storage
class**. The classes are not interchangeable — each has a different lifecycle,
a different backup posture, and a different rule for deciding which class a
new file belongs to. The split exists so authored configuration stays in git,
runtime execution state stays out of git but is backed up, and derived caches
can be lost without consequence.

The seam is `internal/paths.Roots`. Every store/handler that needs a
filesystem path takes `Roots` (or a single narrowly-typed root) from its
constructor; no package writes a path against a global.

## The three classes

### Config — `Roots.Config`

Authored, version-controlled, intentional. Lives in
`scenarios/prompt-manager/store/`.

- Files: `skills/`, `agents/`, `actions/`, `relations/`, `schemas/`,
  `templates/`, `topics/`, `config/`, `world-{scale,seats}.json`,
  `teams/<team>/{team,roles,org}.json`,
  `teams/<team>/members/<member>/{RESPONSIBILITIES.md, HEARTBEAT.md, topics.json}`,
  `teams/<team>/shared/TEAM.md`.
- Lifecycle: edited by humans, reviewed in PRs, deployed as code.
- Backup: covered by git. No DBM coverage needed.

### RuntimeData — `Roots.RuntimeData`

Mutable execution state that the system writes during normal operation.
Lives under `~/.vrooli/data/vrooli/prompt-manager/` (api-core/storage
`ClassData`, `ProfileAuto`).

- Files: `teams/<team>/members/<member>/{heartbeat.json, last-handoff.md,
  inbox.json, logs/*.log}`, `teams/<team>/shared/{tasks.json,
  decisions.jsonl, handoff-history.jsonl, heartbeat-attempts.jsonl,
  knowledge.jsonl}`, `team-queue-<team>.json`, `heartbeat-active-runs.json`,
  `experiments/<id>/...`, `backups/<mirrored-rel-path>.backup`.
- Lifecycle: written every heartbeat tick, handoff, knowledge append, queue
  update. Never edited by humans.
- Backup: covered by data-backup-manager's `WellKnownScanner` via the
  `vrooli/data` discovery suggestion. No self-registration call.

### RuntimeCache — `Roots.RuntimeCache`

Derived state. Lives under `~/.vrooli/cache/vrooli/prompt-manager/`
(api-core/storage `ClassCache`).

- Files: `indexes/{skills,agents,teams,topics,graph}.index.json`.
- Lifecycle: regenerable from Config + RuntimeData via
  `FileStore.RegenerateIndexes(ctx)`. Safe to lose.
- Backup: none — cache class is defined as ok-to-lose.

## Decision rule for a new file

Pick the class by asking, in order:

1. **Is it durable, authored, and meaningful in a PR diff?** → Config.
2. **Does running execution mutate it, and would losing it represent data
   loss (history, decisions, queue depth)?** → RuntimeData.
3. **Is it fully reconstructable from Config + RuntimeData?** → RuntimeCache.

Backups (`.backup` artifacts) always route through `Roots.BackupFor(rel,
suffix)` and land under `RuntimeData/backups/` — never as siblings of the
original. The suffix disambiguates concurrent writes (timestamp, content
hash, etc.).

## Why a split

`store/` historically held all three classes interleaved. The result:
`git status` was permanently dirty with heartbeat/handoff/queue noise, the
scenario violated `storage-steer`'s "mutable filesystem state routes through
`package:api-core/storage`" rule, and data-backup-manager could not back up
prompt-manager runtime state because the well-known scanner only covers
`~/.vrooli/`. The split fixes all three.

The split is **not** a wholesale move like swarm-manager. The config subtree
shape is preserved verbatim; only the runtime/cache classes migrate to
api-core/storage homes.

## Testing

Tests get `paths.Roots` via two fixture functions in `internal/paths`:

- `RootsForTest(t)` — every class rooted under a fresh `t.TempDir()`.
- `RootsForRepoStoreTest(t, configDir)` — Config points at a real on-disk
  store path (e.g. `"../../store"`) while the runtime classes live under
  `t.TempDir()`. Used by canary tests that read the authored store but must
  not write runtime artifacts back into the repo.

## See also

- `internal/paths/paths.go` — the `Roots` struct + `Resolve`/test helpers.
- `api/docs/internal/STORAGE_AUDIT.md` — per-file class assignment ledger.
- `packages/api-core/storage/` — the upstream substrate behind RuntimeData
  and RuntimeCache.
