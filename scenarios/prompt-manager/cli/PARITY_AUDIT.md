# Prompt-Manager CLI ↔ API Parity Audit

**Principle:** Every prompt-manager API endpoint must have a corresponding CLI subcommand. The CLI is a thin, fully-featured wrapper over the API. Endpoints that are intentionally not exposed on the CLI must be marked as such with a justification.

**Enforcement:** The Go test in `scenarios/prompt-manager/cli/parity/` reads the source-of-truth coverage map (`parity/coverage.json`) and the API source (`api/main.go`), then fails CI if any v1 route is unmapped or if a coverage entry refers to a route that no longer exists.

---

## Current State (2026-04-30)

| Status | Count |
|---|---:|
| `covered` | 113 |
| `intentionally-absent` | 5 |
| `audit-pending` | 50 |
| **Total v1 routes** | **168** |

`audit-pending` entries are not gaps in the strict CI-failing sense — they have been classified just enough to satisfy the guard, but every one needs a follow-up pass to either confirm `covered` (with the precise CLI invocation) or `intentionally-absent` (with a real reason). They are the work surface for follow-up backlog items.

---

## How to extend

### Adding a new API endpoint
1. Add the `v1.HandleFunc(...)` in `api/main.go`.
2. Add the matching `case "<sub>":` entry to the appropriate CLI domain (or create a new domain package).
3. Add an entry to `parity/coverage.json` with status `covered` and the precise CLI invocation in the `cli` field.
4. Run `go test ./parity/` from the cli directory — it should pass.

### Removing an API endpoint
1. Delete the `v1.HandleFunc(...)` registration.
2. Delete the corresponding CLI dispatch case and `cmd*` function (greenfield — no shims).
3. Delete the `coverage.json` entry. The `TestParityNoStaleCoverageEntries` test will flag it for you if you forget.

### Marking an endpoint as intentionally CLI-absent
Add a coverage entry with:
```json
{
  "status": "intentionally-absent",
  "reason": "<why operators never invoke this — must be specific>"
}
```
Generic reasons ("internal API") are not acceptable; the justification must explain *who* calls it and *why* a human operator never would.

### Draining the audit-pending queue
Change a `audit-pending` entry to `covered` once you have:
1. Verified the precise CLI invocation that wraps the route (test it locally).
2. Updated the `cli` field to the canonical command form.
3. Removed the `reason` field (not required for `covered`).

Or change it to `intentionally-absent` with a real reason.

---

## Coverage by area

The following groups summarize where covered/audit-pending entries cluster. See `parity/coverage.json` for the per-route detail.

### ✅ Fully covered (or near-fully)

- **Unified work** — team findings and operator dispositions are owned by Swarm Manager; prompt-manager does not expose a parallel decision CLI.
- **Knowledge** — `team knowledge-{add,list,update,delete}` (all 4 endpoints)
- **Tasks (team-scoped)** — `team task-{list,add,update,delete}` (all 4 endpoints)
- **Skills core CRUD** — `skill {list,show,add,read,update,delete,sync,use,rate,versions,revert,variants,add-variant,rm-variant}`
- **Actions CRUD/validation** — `action {list,show,create,update,delete,validate}`; execution is intentionally deferred until runtime governance lands
- **Agents core CRUD** — `agent {list,show,create,update,delete,soul,search}`
- **Experiments core** — `experiment {list,show,create,delete,start,conclude,outcomes}`
- **Graph reads** — `graph {show,regenerate,orphans,skillless,empty-teams,unaffiliated,popular,cycles,health,node}`
- **Topics** — `topic {list,show,create,update,delete,skills}`
- **Tags** — `tag {list,create}`
- **Teams core CRUD** — `team {list,show,create,update,delete,add-member,update-member,remove-member,roles,org-list,org-set,org-remove,message-*,heartbeat-list,heartbeat,heartbeat-enable,heartbeat-disable,heartbeat-trigger,heartbeat-logs,trigger,prompt-preview,prompt-preview-structured,prompt-matrix,member-context,handoff-latest,handoff-history,responsibilities,heartbeat-instructions,import-cc,export-cc,retention,prune}`

### ⚠️ Gap clusters (audit-pending — follow-up work)

- **Agent file management** (7 routes) — no CLI subcommands for `agent files {list,get,set,create,delete,rename}` or `agent teams`
- **Team shared files** (6 routes) — no `team shared-files-*` subcommands
- **Runs** (8 routes) — `POST /runs`, investigation runs, retry/continue, run events; no top-level `run` command
- **Search teams** (3 routes) — no `team search` or AI-team-search command
- **Graph health-config** (2 routes) — no get/put for the health-config endpoint
- **Discover-filter / budget config** (4 routes) — no get/put for `/config/budgets` or `/config/discover-filters`
- **Heartbeat lifecycle edges** (5 routes) — no delete-heartbeat, no global running list, no team-wide log list, no single-log show, no execution-status
- **Handoff clearing** (2 routes) — no DELETE wrappers for handoff or handoff-history
- **Org chart full PUT** (1 route) — `team org-set` covers single edges; bulk `PUT /teams/{id}/org` not exposed
- **Topic match** (1 route) — `POST /topics/match` may already be wrapped by `topic search`; verify
- **Variant update** (1 route) — `PUT /skills/{id}/variants/{vid}` — `add-variant` may overwrite, or this is a real gap
- **Experiment update** (1 route) — `PUT /experiments/{eid}` — no explicit `experiment update` case
- **Roles set** (1 route) — `team roles` only lists; PUT requires a separate command or a `--set` flag
- **Single template list** (1 route) — `GET /agent-file-templates` — likely UI-only, candidate for `intentionally-absent` after verification
- **Exclusive members** (1 route) — `GET /teams/{id}/exclusive-members` — likely UI-only, candidate for `intentionally-absent` after verification
- **Available CC teams** (1 route) — `GET /teams/import/claude-code/available` — `import-cc` may bundle this; verify

### 🚫 Intentionally absent (2)

- `GET /health` — liveness probe
- `GET /og-metadata` — server-side landing-page metadata fetch

---

## Follow-up work

The audit-pending queue is the gap inventory. Recommended decomposition (one swarm-manager backlog item per cluster, each kind=execute, priority 3):

1. `prompt-manager-cli-agent-files` — Add `agent files {list,get,set,create,delete,rename}` and `agent teams` (7 routes)
2. `prompt-manager-cli-team-shared-files` — Add `team shared-file-*` subcommands (6 routes)
3. `prompt-manager-cli-runs` — Add a top-level `run` command package (8 routes; design choice: which routes are operator-relevant vs UI-only)
4. `prompt-manager-cli-heartbeat-lifecycle-gaps` — Close delete-heartbeat, global running list, team-wide logs, single-log show, execution-status (5 routes)
5. `prompt-manager-cli-config-and-budgets` — Add `discover budget {get,set}` and `discover filters {get,set}` (4 routes)
6. `prompt-manager-cli-misc-parity-fixes` — Topic-match, variant-update, experiment-update, roles-set, org-bulk-put, handoff-clearing (~7 routes)

Each follow-up should: add the missing subcommand, add a focused unit test, and update its `coverage.json` entry from `audit-pending` to `covered`.

When all 50 audit-pending entries are drained, the CLI achieves true thin-wrapper parity over the API and the principle stated at the top of this document is enforced end-to-end.
