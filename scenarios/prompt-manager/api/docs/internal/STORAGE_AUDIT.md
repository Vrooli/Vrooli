# STORAGE_AUDIT

Audit of every filesystem read/write in `scenarios/prompt-manager/api/` against
the three-class storage model defined in
[`docs/concepts/STORAGE_CLASSES.md`](../concepts/STORAGE_CLASSES.md):

- **Config** — authored, git-tracked, lives under repo `store/`.
- **RuntimeData** — mutable application state, `api-core/storage` `ClassData`,
  lives under `~/.vrooli/data/vrooli/prompt-manager/`.
- **RuntimeCache** — rebuildable cache, `api-core/storage` `ClassCache`, lives
  under `~/.vrooli/cache/vrooli/prompt-manager/`.

Authored under `storage-steer` §11. The checklist below covers every call site
returned by:

```
grep -rn 'heartbeat\.json\|last-handoff\|handoff-history\|heartbeat-attempts\|knowledge\.jsonl\|decisions\.jsonl\|tasks\.json\|team-queue\|heartbeat-active-runs\|index\.json\|\.backup\|inbox\.json' scenarios/prompt-manager/api/ --include='*.go'
grep -rn 'storeDir\|absStoreDir' scenarios/prompt-manager/api/ --include='*.go'
grep -rn 'experimentsDir\|"experiments"\|"indexes"\|"logs"' scenarios/prompt-manager/api/ --include='*.go'
```

## Class assignments

### Cache → `Roots.RuntimeCache`

| File pattern | Touched at | Move (Phase) |
|---|---|---|
| `indexes/skills.index.json` | `store/indexer.go:78,142` | 3a ✅ |
| `indexes/agents.index.json` | `store/indexer.go:103,154` | 3a ✅ |
| `indexes/teams.index.json` | `store/indexer.go:136,166` | 3a ✅ |
| `indexes/topics.index.json` | `store/indexer.go:199,205` | 3a ✅ |
| `indexes/graph.index.json` | `graph/index.go:42` | 3a ✅ |
| `indexes/` mkdir bootstrap | `store/store.go:138` | 3a ✅ |
| `indexesDir()` helper | `store/indexer.go:34` | 3a ✅ |

### Runtime data, top-level → `Roots.RuntimeData/<file>`

| File pattern | Touched at | Move (Phase) |
|---|---|---|
| `team-queue-<team>.json` (kept verbatim per CD-2 path-shape preservation) | `heartbeat/team_execution.go:418` + helpers `team_execution_helpers.go:23,28,29` | 3d ✅ |
| `heartbeat-active-runs.json` | `heartbeat/run_registry.go:34` | 3d ✅ |
| `experiments/<id>/...` | `store/experiment_store.go:25-37`, `store/store.go:132` mkdir | 3b ✅ |

### Runtime data, per-team → `Roots.RuntimeData/teams/<team>/`

#### `members/<member>/` writes

| File | Get / Set / Delete / List | Touched at | Move (Phase) |
|---|---|---|---|
| `heartbeat.json` | Get / Set / Delete / List | `store/team_store.go:427,454,460` + `ListHeartbeatConfigs` | 3c ✅ |
| `last-handoff.md` | Set / Get / GetOrEmpty | `store/team_store.go:837,849,971` | 3c ✅ |
| `inbox.json` | Get / Set | `store/team_store.go:255,285` | 3c ✅ |
| `logs/<ts>.log` | Path / List | `store/team_store.go:614,619` + `EnsureMemberDir:358` mkdir | 3c ✅ |

`EnsureMemberDir` and `DeleteMemberData` must operate on both `configMemberDir`
and `runtimeMemberDir` simultaneously (`store/team_store.go:351,334`).

#### `shared/` writes

| File | Operation | Touched at | Move (Phase) |
|---|---|---|---|
| `heartbeat-attempts.jsonl` | Append / List | `store/team_store.go:494,513` | 3c ✅ |
| `handoff-history.jsonl` | Append / List / Clear | `store/team_store.go:859,879,921` | 3c ✅ |
| `tasks.json` | Set / Get | `store/team_store.go:982,995` | 3c ✅ |
| `decisions.jsonl` | Append / List | `store/team_store.go:1067,1130` | 3c ✅ |
| `knowledge.jsonl` | Append / List | `store/team_store.go:1216,1268` | 3c ✅ |
| `knowledge.jsonl` | Read (attribution scan) | `memberflow/runtime_attribution.go:129` | 3c ✅ |

`ListSharedFiles` (`store/team_store.go:638…`) currently enumerates the entire
config-tree `shared/` directory. After cutover the directory contains only
`TEAM.md` under Config; runtime jsonl files live under RuntimeData. Operation
behavior preserved by merging the two listings, with file paths reported
relative to the merged virtual `shared/` tree (Phase 3c).

#### Prompt-section source-path references

`heartbeat/prompt_builder.go:185-196` constructs **display** `SourcePath`
strings (`teams/<t>/members/<m>/last-handoff.md`, `teams/<t>/shared/knowledge.jsonl`)
for prompt sections. These are relative-path labels, not file reads — they stay
shape-identical because RuntimeData mirrors the Config tree shape (CD-2). No
change required at the writer site; the reader in `runtime_attribution.go` (and
any future reader) resolves the relative path against `Roots.RuntimeData`.

### Stays Config (no change)

These remain under repo `store/` and are written by authoring flows, never by
agent execution:

- `teams/<team>/team.json` (lines 92, 184, 291)
- `teams/<team>/roles.json` (lines 200, 229)
- `teams/<team>/org.json` (lines 206, 244)
- `teams/<team>/members/<m>/RESPONSIBILITIES.md` (line 375)
- `teams/<team>/members/<m>/HEARTBEAT.md` (in-tree near line 395)
- `teams/<team>/members/<m>/topics.json`
- `teams/<team>/shared/TEAM.md`
- All of `skills/`, `agents/`, `schemas/`, `templates/`, `topics/`,
  `relations/`, `actions/`, `config/`, `world-*.json`

### `.backup` writers

Production scenario code has **zero** `.backup` writers; the only emitter is
`cmd/migrate-knowledge-attribution/migrate.go:326`, a one-shot historical
migration tool that today writes `<path>.backup` next to the original under
`store/teams/<team>/shared/`. Phase 3e routes that tool through
`paths.Roots.BackupFor(...)` so any future re-run (or any new `.backup` emitter)
lands under `RuntimeData/backups/` (CD-3). Pre-existing `.backup` artifacts
already committed under `store/` are removed during the Phase 5 cutover.

## `storeDir` / `absStoreDir` god-variable spread

226 hits across 31 files (Phase 3f sweep). The threading flows from
`main.go`'s `absStoreDir` into ~30 constructors. After Phase 3, each
constructor takes either `paths.Roots` (when it needs multiple classes — only
`FileStore`, the index store, and a handful of handlers) or a single
root-typed string (when it only needs one class). The bare `storeDir string`
parameter shape is forbidden post-Phase-3f.

Files touched:

```
actions/resolver.go
agents/handlers.go
aisearch/budget_config.go
aisearch/discover_filter_config.go
graph/health_config.go
graph/index.go
heartbeat/inbox_flow.go
heartbeat/prompt_builder.go
heartbeat/topic_contract.go
main.go
memberflow/handlers.go
memberflow/loader.go
memberflow/operating_graph_runtime.go
memberflow/prose_scan.go
memberflow/runtime_attribution.go
memberflow/team_contracts.go
memberflow/validation.go
skills/handlers.go
store/action_store.go
store/agent_store.go
store/experiment_store.go
store/indexer.go
store/relation_store.go
store/skill_store.go
store/store.go
store/team_store.go
store/topic_store.go
teamcontract/contract.go
templates/store.go
worldscale/handlers.go
worldseats/handlers.go
```

## Cutover migration list (Phase 5)

Live data to move from `store/` to the new roots, computed from the audit
above. Concrete `rsync` commands enumerated in the phase-5 runbook. The
authoritative tracked list is:

```
git ls-files scenarios/prompt-manager/store/ | grep -E '(heartbeat\.json|last-handoff\.md|.*\.jsonl|tasks\.json|team-queue-.*\.json|heartbeat-active-runs\.json|inbox\.json|/logs/|/indexes/|\.backup|/experiments/)$'
```

Untracked live runtime files (heartbeats etc. written since the last commit)
are mirrored to the new root before the corresponding tracked path is
`git rm`ed; the operator verifies each mirror before removal.
