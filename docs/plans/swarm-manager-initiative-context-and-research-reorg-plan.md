# Swarm Manager: Initiative-Aware Context + Research-Driven Backlog Reorganization

## 1. Purpose

Close the gap between three interacting weaknesses in the swarm-manager workflow:

1. **Cascade is asymmetric** — item deletes clean `depends_on` references on other items, but *nothing* keeps `initiative.items[]` in sync when items are deleted, moved between initiatives, or when the initiative itself is deleted. The system relies on agents to maintain referential integrity, which is the opposite of the user's stated principle ("I shouldn't need to worry about cascading … minimize the chance of invalid state").
2. **Research conclusions can only grow the backlog** — the `research-conclusion-authoring` action vocabulary is additive-only (`Create backlog item`, `Update document`, `No further action`). There is no way to express *"delete the obsolete idea,"* *"reprioritize the sibling fix,"* or *"update the enclosing initiative's depends_on,"* so agents never do these things even when research findings demand them.
3. **Skills work in a single-item tunnel** — every workshop / process skill loads the current item and maybe its initiative's description, but never sibling items, never upstream or downstream initiatives. An agent doing research on `idea/notification-hub` has no idea that `idea/email-digest` in the same initiative has been superseded by its findings.

All three must be fixed together: cascading must become a server invariant, a single endpoint must serve the full "initiative context" in one call, and research skills must speak an action vocabulary that matches what the API and cascade layer now allow.

## 2. Required Reading

Execution agent runs these before touching code:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring skill-principles skill-validation
prompt-manager skill read swarm-manager-meta-orchestrator
prompt-manager skill read swarm-manager-backlog-tools
prompt-manager skill read swarm-manager-workshop swarm-manager-workshop-research
prompt-manager skill read swarm-manager-workshop-research-finalize swarm-manager-process-research
prompt-manager skill read research-conclusion-authoring
```

Also re-read these files (the execution surface):

- `scenarios/swarm-manager/api/internal/backlog/handler.go` (single-item delete, update_patch, and batch entry points)
- `scenarios/swarm-manager/api/internal/backlog/update_patch.go` (patch handler that silently changes `initiative` without cascade)
- `scenarios/swarm-manager/api/internal/backlog/store.go` (`RemoveDependencyRef`, `ValidateDependencies`)
- `scenarios/swarm-manager/api/internal/backlog/batch_handler.go`, `batch_queue.go` (batch validation — the only place initiative-dep validation exists today)
- `scenarios/swarm-manager/api/internal/initiatives/service.go` (`Create`, `Update`, `Delete`, `AddItems`, `RemoveItems`)
- `scenarios/swarm-manager/api/internal/initiatives/handler.go` (HTTP surface for the new `context` endpoint)
- `scenarios/swarm-manager/api/internal/initiatives/store.go`
- `scenarios/swarm-manager/api/internal/initiatives/adapters.go`
- `scenarios/swarm-manager/cli/cmd/backlog_*.go`, `cmd/initiatives_*.go` (CLI exposure for the new endpoint)
- `scenarios/prompt-manager/store/skills/packs/core/research-conclusion-authoring/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-process-research/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-research/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-research-finalize/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-finalize/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-clarify/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-backlog-tools/SKILL.md`
- `scenarios/swarm-manager/docs/reference/cli-commands.md`

## 3. Problem Statement

### 3.1 Cascade is silently broken

Confirmed by direct code audit:

- `scenarios/swarm-manager/api/internal/backlog/handler.go:412-418` — item delete calls `store.RemoveDependencyRef(ref)` which sweeps every other item's `depends_on` array. **But it does not touch `initiative.items[]` on the enclosing initiative.** Deleting `idea/foo` that belongs to `initiative/bar` leaves `bar.items[]` still containing `"idea/foo"`.
- `scenarios/swarm-manager/api/internal/backlog/update_patch.go:218-220` — patching an item's `initiative` field just overwrites the scalar. **Neither the old nor the new initiative's `items[]` list is touched.** Moving an item between initiatives silently desynchronizes both of them.
- `scenarios/swarm-manager/api/internal/initiatives/service.go:234-247` — initiative delete removes the file and emits an analytics event. **It does not clear the `initiative` field on member items, and does not remove the deleted initiative's name from other initiatives' `depends_on` arrays.**
- `scenarios/swarm-manager/api/internal/initiatives/service.go:320-353` — `AddItems` updates `initiative.items[]` but does not touch the item file's `initiative` field. Called symmetrically today (meta-orchestrator sets `item.initiative` and then calls `AddItems`), but the asymmetry is dangerous because agent-driven reorganization could easily miss one side.

The design doesn't reject invalid inputs either: you can `PATCH` an item's `initiative` field to point at an initiative that doesn't exist (the single-item path doesn't call the batch's `validateInitiativeDepRefs`), and you can delete an initiative even if items reference it.

### 3.2 Research conclusions can only grow the backlog

`research-conclusion-authoring` enumerates three action types:

- `Create backlog item`
- `Update document`
- `No further action required`

There is literally no way to say "delete the obsolete item," "reprioritize the sibling fix," or "move this item to a different initiative" in the conclusion's Actions section. Even though the underlying CLI supports `backlog delete`, `backlog update`, `initiatives update`, and `initiatives remove-items`, agents following the skill never reach for them. `swarm-manager-process-research` compounds the problem: its worked example only ever calls `backlog batch-create`.

### 3.3 Skills are initiative-blind

- `swarm-manager-workshop-research` tells the agent to "read any files present" in the initiative directory, but never to load sibling items, upstream initiatives (the current initiative's `depends_on`), or downstream initiatives (other initiatives that depend_on this one).
- `swarm-manager-workshop` (idea/fix/execute/chore) is the same.
- There is no shared "initiative context loader" skill or API endpoint that other skills reference. Every skill reinvents ad-hoc context loading.
- The overview endpoint (`/api/v1/overview`) is global — agents would have to fetch everything and filter client-side, which is the wrong tool for the job.

### 3.4 Concrete user-observed symptom

In practice: the meta-orchestrator creates a research item alongside a handful of non-research items. Research completes, its conclusion writes new items, and the pre-research items (which may now be wrong, redundant, or mis-sequenced) are never touched. The user has never seen the research flow delete or reprioritize anything, because the skill layer never teaches it to.

## 4. Scope

**In scope**

- **API cascade invariants** — item delete, item patch (initiative move), initiative delete, initiative patch (items list update) all maintain referential integrity automatically. Server enforces; agents stop tracking cascade.
- **New context endpoint** — `GET /api/v1/initiatives/{name}/context` returning the initiative, its member items with current status, upstream initiatives (full objects), and downstream initiatives (full objects). Exposed via CLI.
- **Stronger single-item validation** — `PATCH /api/v1/backlog/{kind}/{name}` rejects `initiative` changes to non-existent initiatives (matching batch-create's existing check). Single-item item-create gets the same check.
- **Research action vocabulary expansion** — `research-conclusion-authoring` gains `Update backlog item`, `Delete backlog item`, and `Update initiative`. Process-research gains matching CLI examples.
- **Reuse-before-create heuristic** — research workshop skill instructs the agent to enumerate siblings before proposing a new item, and to prefer updating an existing one when the intent overlaps.
- **Initiative-aware context loading for all workshop skills** — research *and* non-research, via a shared reference to the new endpoint.
- **Shared skill** `swarm-manager-initiative-context` that encapsulates the loading pattern and is read-in by every workshop/process skill that needs it.
- **Docs + CLI reference** updates.
- **Automated tests** across all the above.

**Out of scope**

- UI changes in swarm-manager web console (initiative context viewer is a separate backlog item if wanted).
- Batch mutation API — no `batch-update` or `batch-delete` endpoint in this plan. Multi-item reorganization uses N CLI calls; batch becomes pressing only if conclusion reorgs routinely exceed ~5 items. Flagged as a future item.
- New action types for non-research kinds. Non-research workshops will continue to produce `plan.md` (guidance) rather than backlog mutations. Only research produces conclusion actions.
- Retroactive migration / repair of already-orphaned references in existing data. New invariants apply going forward. (Optional one-shot repair script mentioned in risks but not required for DoD.)
- Changes to the meta-orchestrator's *planning* flow beyond pointing it at the context endpoint.

## 5. Current Technical Context

### 5.1 API package structure

```
scenarios/swarm-manager/api/
├── internal/backlog/
│   ├── handler.go                  # single-item CRUD HTTP handlers (including delete cascade call)
│   ├── update_patch.go             # PATCH handler — changes initiative field with no cascade
│   ├── batch_handler.go            # atomic multi-item create; validates initiative deps
│   ├── batch_queue.go              # resolveInitiativePlans, orderedInitiativePlans (topo sort)
│   ├── store.go                    # FileStore — RemoveDependencyRef, ValidateDependencies
│   └── ...
├── internal/initiatives/
│   ├── service.go                  # Create/Update/Delete/AddItems/RemoveItems — NO cascade
│   ├── handler.go                  # HTTP handlers (new context endpoint lands here)
│   ├── store.go                    # FileStore for initiatives
│   ├── adapters.go                 # backlogAssignerAdapter (the bridge batch_handler uses)
│   └── model.go                    # Initiative, CreateRequest, UpdateRequest
└── routes.go                       # route registration
```

### 5.2 Reused primitives (do NOT duplicate)

- `backlog.FileStore.RemoveDependencyRef(ref string) (int, error)` — already sweeps other items' `depends_on`. Reuse for item-delete cascade on the depends_on axis. Do not reinvent.
- `backlog.FileStore.ValidateDependencies([]string)` — forward-ref validation for item depends_on.
- `initiatives.Service.AddItems`, `RemoveItems` — both already exist; extend them to also update the item-side `initiative` field if the symmetric cascade design demands it (see §8.3).
- `initiatives.Service.Exists(name string) bool` — use from the backlog package via the existing `InitiativeAssigner` adapter to validate single-item `initiative` field changes.
- `internal/depgraph` — already used for topological sort of initiatives and items. Reuse if any new traversal is needed.

### 5.3 Skill files that change

- Modified:
  - `research-conclusion-authoring/SKILL.md`
  - `swarm-manager-process-research/SKILL.md`
  - `swarm-manager-workshop-research/SKILL.md`
  - `swarm-manager-workshop-research-finalize/SKILL.md`
  - `swarm-manager-workshop/SKILL.md`
  - `swarm-manager-workshop-finalize/SKILL.md`
  - `swarm-manager-workshop-clarify/SKILL.md`
  - `swarm-manager-meta-orchestrator/SKILL.md`
  - `swarm-manager-backlog-tools/SKILL.md`
  - Each skill's `skill.json` gets a `revision` bump and `updatedAt` set to the execution date.
- New:
  - `swarm-manager-initiative-context/SKILL.md` (+ `skill.json`) — shared reference skill describing how to load and use initiative context from the new endpoint.

### 5.4 Docs files that change

- `scenarios/swarm-manager/docs/reference/cli-commands.md` — document the new `initiatives context` command, cascade semantics, and updated delete behavior.

## 6. Target End State

**API invariants (all must hold)**

- Deleting an item `kind/name` removes every `depends_on` occurrence of `"kind/name"` in other items *and* removes `"kind/name"` from its enclosing initiative's `items[]` (if any). Atomic from the caller's perspective; rollback-safe on partial failure.
- Patching an item so its `initiative` field changes from `A` to `B` removes `"kind/name"` from `A.items[]` and adds it to `B.items[]` (dedup). Setting the field to empty removes it from `A.items[]` without adding to anything. Setting it to a non-existent initiative is rejected with a clear error.
- Creating an item with an `initiative` field sets it and adds `"kind/name"` to that initiative's `items[]` (dedup). Rejected if the initiative doesn't exist.
- Deleting an initiative clears the `initiative` field on every member item (leaves the items alive and orphaned) *and* removes the deleted initiative's name from every other initiative's `depends_on` array. Atomic-and-rollback.
- Adding/removing items via `POST /api/v1/initiatives/{name}/add-items` and `POST /api/v1/initiatives/{name}/remove-items` also updates the items' `initiative` field symmetrically.

**New read endpoint**

- `GET /api/v1/initiatives/{name}/context` returns a single JSON document:

  ```json
  {
    "initiative": { ...full Initiative object with rollup... },
    "items": [ { "kind": "...", "name": "...", "title": "...", "status": "...", "priority": 0, "depends_on": [...] }, ... ],
    "upstream_initiatives": [ { ...full Initiative for each depends_on target... } ],
    "downstream_initiatives": [ { ...full Initiative for each one that depends_on this... } ]
  }
  ```

- CLI exposure: `swarm-manager initiatives context --name <name>` prints human-friendly output by default, JSON with `--json`. (Skill examples use the human default per standing user feedback — `--json` only in automated scripts or when the agent explicitly needs to parse.)

**Skill behavior**

- Research workshop and research-finalize skills load context via `swarm-manager initiatives context` as part of their "gather context" step and use it to drive the Decisions list (e.g., "Does Finding X supersede `idea/email-digest`?").
- Research conclusion templates accept and show examples for `Create`, `Update backlog item`, `Delete backlog item`, `Update initiative`, `Update document`, `No further action`.
- `swarm-manager-process-research` executor demonstrates CLI calls for all five action types.
- Non-research workshop skills (`swarm-manager-workshop`, `-finalize`, `-clarify`) load initiative context via the same endpoint but continue to produce `plan.md` — their scope does not expand to backlog mutations.
- Meta-orchestrator and backlog-tools skills document the new endpoint.
- Cascade is **never discussed in skills** — the server handles it. Agents may freely delete/move items without worrying about dangling references; the API either succeeds atomically or returns a clear error.

**Test and gate coverage**

- Unit + integration tests for each cascade invariant (see §9).
- Unit + integration tests for the context endpoint.
- `go build ./...`, `gofumpt -l` (scoped to edited files), `golangci-lint run ./...`, `go test ./... -race -count=1 -timeout 600s` all pass in `scenarios/swarm-manager/api`.
- Skill JSON validates and renders successfully.

## 7. Implementation Strategy (phased)

### Phase 1 — Cascade invariants in the API

1. **Item delete cascade completion** (`internal/backlog/handler.go`, `store.go`, `internal/initiatives/service.go`):
   - After `RemoveDependencyRef`, if the deleted item has an `Initiative` field set, call a new `initiatives.Service.RemoveItems(initiativeName, []string{"kind/name"})` path *without* also touching member items' `initiative` field (the item is already gone). Add a narrower helper if needed — e.g., `initiatives.Service.ForgetItem(initiativeName, ref)` — that only rewrites `initiative.items[]`.
   - Wrap the two side effects (depends_on sweep + initiative items[] rewrite) so that if either fails, the whole delete surfaces an error and the caller sees an atomic failure. The item file itself is the last thing to be removed; if removing it fails, the earlier cleanups must roll back. Use the same file-based rollback pattern batch_handler already uses.

2. **Item patch cascade** (`internal/backlog/update_patch.go`):
   - When `req.Initiative` is set and differs from `item.Initiative`:
     - Validate the target initiative exists (via the existing `InitiativeAssigner.Get` path).
     - Remove the item ref from the old initiative's `items[]`.
     - Add it to the new initiative's `items[]`.
     - Update the item file's `Initiative` field last, so if anything upstream fails the item is unchanged.
   - When `req.Initiative` is set to "" and differs from `item.Initiative`:
     - Remove the item ref from the old initiative's `items[]`.
     - Clear the item's `Initiative` field.

3. **Item create cascade** (`internal/backlog/handler.go` create path):
   - When the create request includes a non-empty `initiative` field, validate it exists, then after the item is written, add the ref to that initiative's `items[]`.
   - Batch-create already handles this correctly via `groupItemRefsByInitiative` + `AddItems`; extend to the single-item path.

4. **Initiative delete cascade** (`internal/initiatives/service.go`, `handler.go`):
   - Before deleting the initiative, enumerate all member items and clear their `Initiative` field (items are preserved, just orphaned). Also scan all other initiatives and remove the deleted name from their `depends_on` arrays.
   - Rollback-safe: if any side effect fails partway, restore the cleared item fields and the modified initiatives' depends_on, then surface the error. Use snapshot-then-write like `rollbackBatchCreate`.
   - Analytics event (`EmitInitiativeArchived`) fires only after all cascades succeed.

5. **Initiative AddItems / RemoveItems symmetry** (`internal/initiatives/service.go`):
   - Change `AddItems(name, items)` to also set each item's `Initiative = name` if it's currently empty or differs. If an item already belongs to a different initiative, either (a) reject with a clear error ("item already belongs to X; move explicitly via PATCH") or (b) do the move. Decision: **reject**, because implicit cross-initiative moves are surprising. The agent must use PATCH for moves; AddItems is for attaching orphans.
   - Change `RemoveItems(name, items)` to also clear each item's `Initiative` field if it currently equals `name`. If it differs, log a warning (the two sides were desynchronized — skill-layer bug somewhere).

6. **Validator: single-item initiative ref** (`internal/backlog/update_patch.go`, handler.go create path):
   - Share a helper `validateInitiativeExists(name string) error` that returns a typed "initiative not found" error. Wire into create and patch.

### Phase 2 — Initiative context endpoint

1. **Service method** (`internal/initiatives/service.go`):
   - `GetContext(name string) (*InitiativeContext, error)` returning the initiative + rollup, its `depends_on` targets (loaded as full `Initiative`s), its downstream initiatives (computed by scanning all initiatives for ones whose `depends_on` includes this name), and its member items resolved via the `BacklogLoader` interface.
   - `InitiativeContext` is a new struct in `model.go`. Items within it use a compact view (kind, name, title, status, priority, depends_on, archivedAt) — not full item bodies (keep payload bounded).

2. **HTTP handler** (`internal/initiatives/handler.go`):
   - `GET /api/v1/initiatives/{name}/context` → `service.GetContext` → JSON.
   - 404 if the named initiative doesn't exist.

3. **Route registration** (`routes.go`): add the new path alongside existing initiative routes.

4. **CLI** (`scenarios/swarm-manager/cli/cmd/`):
   - Add `initiatives context --name <name>` with `--json` flag. Default output is human-readable (headings: "Initiative", "Members", "Upstream", "Downstream"). Shell-complete + help text matches existing subcommands.

### Phase 3 — Shared initiative-context skill + workshop wiring

1. **New skill**: `swarm-manager-initiative-context` (SKILL.md + skill.json).
   - Short, practical. Describes:
     - When to load context (any time the agent is reasoning about whether to create, update, or delete backlog items).
     - The one-line CLI command (`swarm-manager initiatives context --name <name>`).
     - The semantic interpretation (members are the "plan within this initiative"; upstream are what this blocks on; downstream are what this unblocks).
     - A 2-step heuristic for "reuse before create": before proposing a new item, enumerate members and siblings; only create if no existing item can be updated to cover the same intent.
   - Not a stand-alone workflow — it exists to be read-in by other skills via "Required reading."

2. **Update research workshop skills** to reference and use the context skill:
   - `swarm-manager-workshop-research/SKILL.md`: add `swarm-manager-initiative-context` to required reading; add a "Load context" step before the Decisions phase; add guidance that Decisions must explicitly consider sibling items ("Does any finding supersede a sibling? Does priority reorder change?").
   - `swarm-manager-workshop-research-finalize/SKILL.md`: same required-reading addition; re-scoring readiness now includes "are sibling/initiative impacts captured in the Actions list?"

3. **Update non-research workshop skills**:
   - `swarm-manager-workshop`, `swarm-manager-workshop-finalize`, `swarm-manager-workshop-clarify`: add `swarm-manager-initiative-context` to required reading; add a "load context" step; add guidance to call out (in `plan.md`) any sibling items or initiative-level concerns that surface but can't be acted on directly (non-research workflows still don't mutate the backlog — they surface intent for the orchestrator).

### Phase 4 — Research conclusion action vocabulary

1. **`research-conclusion-authoring/SKILL.md`**:
   - Keep the three existing action types (Create, Update document, No further action).
   - Add three new action types with full examples:
     - `Update backlog item` — fields: kind, name, changes (priority/title/depends_on/initiative), reason.
     - `Delete backlog item` — fields: kind, name, reason (no "check dependents first" — cascade handles it).
     - `Update initiative` — fields: initiative name, changes (priority/depends_on/description/title), reason. Marked as rare but supported.
   - Add a "reuse before create" guardrail referring to `swarm-manager-initiative-context`.
   - Add a note that the server maintains referential integrity automatically; agents do not need to patch siblings or the initiative items list themselves after a delete.

2. **`swarm-manager-process-research/SKILL.md`**:
   - Expand the executable-action examples to cover all five action types (one CLI example each: batch-create, backlog update, backlog delete, initiatives update, document write).
   - Remove the implicit "additive-only" framing; the preamble becomes "Execute each action exactly as written — including deletions and updates."
   - Keep the "preserve initiative references on new items" guidance but reframe it as "when creating, set `initiative` explicitly; the server will keep membership in sync."

### Phase 5 — Meta-orchestrator + backlog-tools + docs alignment

1. **`swarm-manager-meta-orchestrator/SKILL.md`**:
   - Add a sentence in the Canonical Contract section: research-driven reorganization (updates/deletes) happens *after* research completes; meta-orchestrator plans the initial shape, not the reorg.
   - Add a one-line pointer to `swarm-manager-initiative-context` for any sub-step that needs to understand what's already in an initiative.

2. **`swarm-manager-backlog-tools/SKILL.md`**:
   - Document the new `initiatives context` CLI command with a short example (human output, not `--json`).
   - Add a "Referential integrity" subsection stating server-side guarantees (no agent-side cascade bookkeeping).

3. **`scenarios/swarm-manager/docs/reference/cli-commands.md`**:
   - Document `initiatives context`.
   - Document the updated delete semantics for items and initiatives (what cascade guarantees).

### Phase 6 — Test coverage

All new tests live alongside the code they cover:

- `internal/backlog/handler_test.go` (or a new `handler_cascade_test.go`):
  - `TestDeleteItem_RemovesDependsOnRefsAndInitiativeMembership` — creates items A, B with B.depends_on=A, both in initiative X; deletes A; asserts B.depends_on empty *and* X.items[] no longer contains A.
  - `TestDeleteItem_AtomicOnPartialFailure` — inject failure in `ForgetItem`; assert A still exists and B.depends_on untouched.

- `internal/backlog/update_patch_test.go`:
  - `TestPatchItem_MoveInitiative_SyncsBothSides` — move item from X to Y; assert X.items[] no longer contains it, Y.items[] does.
  - `TestPatchItem_MoveInitiative_UnknownTarget_Rejected`.
  - `TestPatchItem_ClearInitiative_RemovesFromOldItems`.

- `internal/initiatives/service_test.go`:
  - `TestDeleteInitiative_ClearsMemberInitiativeField`.
  - `TestDeleteInitiative_RemovesFromOtherInitiativesDependsOn`.
  - `TestDeleteInitiative_AtomicOnPartialFailure`.
  - `TestAddItems_RejectsItemsAlreadyInDifferentInitiative`.
  - `TestRemoveItems_ClearsItemInitiativeField`.

- `internal/initiatives/handler_test.go` (new):
  - `TestGetContext_ReturnsUpstreamDownstreamAndMembers`.
  - `TestGetContext_404OnUnknown`.
  - `TestGetContext_EmptyArraysWhenIsolated`.

- CLI tests under `scenarios/swarm-manager/cli/cmd/` cover `initiatives context` happy path and `--json` flag.

Run all from `scenarios/swarm-manager/api`:

```bash
go build ./...
gofumpt -l internal/backlog internal/initiatives   # scoped to files touched
golangci-lint run ./...
go test ./... -race -count=1 -timeout 600s
```

## 8. Contract Decisions

### 8.1 Cascade semantics: clear, don't delete

When an initiative is deleted, member items are **orphaned (Initiative field cleared), not deleted**. The items outlive their initiative. Justification: deletion of work is a much bigger decision than deletion of a container; agents should make per-item archive/delete decisions explicitly. If the user wants a one-shot "delete initiative and all its items," that's a separate CLI flag or orchestrator workflow — not the default cascade.

### 8.2 Initiative-move is explicit; AddItems is for orphans only

`POST /api/v1/initiatives/{name}/add-items` rejects items that already belong to a different initiative with a typed error. Callers wanting to move an item use `PATCH /api/v1/backlog/{kind}/{name}` with a new `initiative`. This keeps "move" in one place and prevents implicit cross-initiative silent moves.

### 8.3 Symmetric sync on AddItems / RemoveItems

Even though AddItems now rejects already-attached items, it does set `item.Initiative = name` on orphans it accepts. `RemoveItems` clears the field if it matches. This means `initiative.items[]` and `item.initiative` are always symmetric after any operation.

### 8.4 Cascade is atomic from the caller's view

Either the whole operation succeeds and all side effects are applied, or nothing is changed and the caller sees an error. Partial state is never visible. Rollback follows the file-based pattern `batch_handler.go:rollbackBatchCreate` already uses: capture originals before mutation; restore on failure.

### 8.5 Context endpoint shape

One endpoint, one call. The agent should not have to compose multiple `/initiatives/{n}` calls. `upstream_initiatives` is a flat list (we don't recurse transitive upstream — that's too much). `downstream_initiatives` is direct dependents only, same reason. Items within the context use a compact view — full item bodies would bloat the response.

### 8.6 Research action vocabulary stays opinionated

`Update backlog item` accepts these fields only: `priority`, `title`, `depends_on`, `initiative`. Not `body`, `description`, or `status` — those are what workshop/execution is for. `Update initiative` similarly limited to `priority`, `depends_on`, `title`, `description`. Status stays `active|completed`; `archived` is already rejected by validator.

### 8.7 No cascade language in skills

Skills do not teach agents how to maintain cross-reference integrity. They describe the operations (create / update / delete) and their effects at a single-item level. The server is the source of truth for invariants.

### 8.8 No batch mutation endpoint

Multi-item reorg = N CLI calls via the standard delete/update endpoints, executed sequentially by `swarm-manager-process-research`. If profiling shows this bottlenecks conclusions with 5+ changes, a future plan adds `POST /api/v1/backlog/batch/update` and `POST /api/v1/backlog/batch/delete`. Out of scope here.

### 8.9 Greenfield

Standing feedback: no compatibility shims, no migration flags, no "v2 endpoint." The existing single-item delete becomes the cascading delete; older callers that happened to work on isolated items see no observable change.

### 8.10 No agent-initiated scenario restart

User restarts `swarm-manager` and `prompt-manager` manually after code and skill edits land. The execution agent writes files and runs tests; it does not restart scenarios.

## 9. Testing Plan

All verification is automated. No manual checklist.

### 9.1 Test cases (added in Phase 6)

Numbered for Definition of Done tracking:

| # | Test | File |
|---|------|------|
| T1 | `TestDeleteItem_RemovesDependsOnRefsAndInitiativeMembership` | `internal/backlog/handler_cascade_test.go` |
| T2 | `TestDeleteItem_AtomicOnPartialFailure` | same |
| T3 | `TestPatchItem_MoveInitiative_SyncsBothSides` | `internal/backlog/update_patch_test.go` |
| T4 | `TestPatchItem_MoveInitiative_UnknownTarget_Rejected` | same |
| T5 | `TestPatchItem_ClearInitiative_RemovesFromOldItems` | same |
| T6 | `TestCreateItem_AttachesToInitiativeMembership` | `internal/backlog/handler_test.go` |
| T7 | `TestCreateItem_UnknownInitiative_Rejected` | same |
| T8 | `TestDeleteInitiative_ClearsMemberInitiativeField` | `internal/initiatives/service_test.go` |
| T9 | `TestDeleteInitiative_RemovesFromOtherInitiativesDependsOn` | same |
| T10 | `TestDeleteInitiative_AtomicOnPartialFailure` | same |
| T11 | `TestAddItems_RejectsItemsInDifferentInitiative` | same |
| T12 | `TestAddItems_SetsItemInitiativeField` | same |
| T13 | `TestRemoveItems_ClearsItemInitiativeField` | same |
| T14 | `TestGetContext_ReturnsFullShape` | `internal/initiatives/handler_test.go` |
| T15 | `TestGetContext_EmptyArraysWhenIsolated` | same |
| T16 | `TestGetContext_404OnUnknown` | same |
| T17 | CLI: `initiatives context --name <n>` renders human output | `cli/cmd/initiatives_context_test.go` |
| T18 | CLI: `initiatives context --name <n> --json` emits valid JSON matching the contract | same |
| T19 | Meta/regression: existing batch-create tests still pass unchanged | existing tests |

### 9.2 Gate commands (run at end of implementation, copy-paste)

```bash
cd scenarios/swarm-manager/api && go build ./...
cd scenarios/swarm-manager/api && gofumpt -l internal/backlog internal/initiatives | grep -v -F -x -f /dev/null || true   # only edited files must be clean; see §11 for tool divergence note
cd scenarios/swarm-manager/api && golangci-lint run ./...
cd scenarios/swarm-manager/api && go test ./... -race -count=1 -timeout 600s
cd scenarios/swarm-manager/cli && go build ./...
cd scenarios/swarm-manager/cli && go test ./... -count=1 -timeout 300s

jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-context/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/research-conclusion-authoring/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-process-research/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-research/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-research-finalize/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-finalize/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop-clarify/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/skill.json >/dev/null
jq . scenarios/prompt-manager/store/skills/packs/core/swarm-manager-backlog-tools/skill.json >/dev/null
```

### 9.3 Skill smoke (after user restarts prompt-manager)

```bash
prompt-manager skill read swarm-manager-initiative-context >/dev/null
prompt-manager skill read research-conclusion-authoring >/dev/null
prompt-manager skill read swarm-manager-process-research >/dev/null
prompt-manager skill read swarm-manager-workshop-research >/dev/null
```

### 9.4 Negative regression asserts

- A test that creates A→B→A and asserts item cycle detection still fires on create (guards that we didn't break existing validation).
- A test that previously asserted alphabetical initiative-apply order is updated to assert topological order (continuation of prior plan).
- The existing "unknown field rejected on batch-create initiative" test must still pass.

## 10. Rollout / Validation Checklist

Run in order; all must succeed:

1. All Phase 6 tests exist and pass locally (`go test ./... -race -count=1 -timeout 600s` in `scenarios/swarm-manager/api`).
2. `go build ./...` clean in both `api` and `cli`.
3. `golangci-lint run ./...` clean in `api`.
4. `gofumpt -l` on the set of edited files returns empty. (Scoped, not repo-wide — repo has pre-existing non-compliant files out of scope.)
5. Every edited `skill.json` has `revision` bumped exactly by 1 and `updatedAt` set to the execution date.
6. The new `swarm-manager-initiative-context/skill.json` is valid JSON and the skill renders via `prompt-manager skill read` (after user restarts prompt-manager).
7. `scenarios/swarm-manager/docs/reference/cli-commands.md` documents `initiatives context` and the cascade semantics.
8. `grep -RIn "cascade" scenarios/prompt-manager/store/skills/packs/core/` returns zero results outside `swarm-manager-backlog-tools` (skills don't lecture agents about cascade — only the reference docs note server-side guarantees).
9. The execution agent asks the user to run `vrooli scenario restart swarm-manager && vrooli scenario restart prompt-manager`. Agent does not restart.
10. After user restart, agent verifies one end-to-end: `swarm-manager initiatives context --name <real-name>` prints expected shape; `swarm-manager backlog delete --kind <k> --name <n>` for a test item cleans both depends_on and initiative.items[] in one call.

## 11. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Cascade change silently alters existing test expectations (e.g., a test that asserted initiative.items[] still contained a deleted item's ref because "the test authored the silently-broken state"). | Medium | Grep for `items[]`-related assertions in existing tests before editing; rewrite any that depend on the broken behavior. The change is correct; tests that asserted the bug are the tests to fix. |
| `AddItems` rejecting cross-initiative items breaks a caller that was implicitly relying on it as a move operation. | Medium | Grep for `AddItems` call sites; meta-orchestrator and batch_handler are the known callers. Confirm neither calls it for cross-initiative moves. Document the rejection clearly in `adapters.go` and the backlog-tools skill. |
| Initiative delete cascade runs many writes (one per member item + one per dependent initiative) with no batch primitive, and on large initiatives the write fan-out is slow. | Low-Medium | Out of scope for optimization; note the cost in the docs and flag a future batch-cascade optimization if profiling shows pain. |
| `golangci-lint` bundled gofumpt differs from standalone version (observed on prior plan). | Medium | Run `golangci-lint run --disable-all --enable gofumpt --fix ./internal/backlog/... ./internal/initiatives/...` after standalone `gofumpt -w`; treat golangci's bundled version as authoritative. |
| Skill revision bumps across ~10 skills mean large git-noise surface. | Low | Bumps are mechanical; a single commit covers skill updates, another covers code. |
| A conclusion written BEFORE this plan lands uses the old 3-action vocabulary; after the plan lands, the process-research executor is backward-compatible (the old types still work, new ones are additive). | Low | Confirmed additive: the new action types parse in addition to, not instead of, the old. |
| Context endpoint is large when an initiative has many downstream dependents. | Low-Medium | Contract limits to *direct* upstream/downstream, not transitive. Items use compact view. Document the max expected size. |
| Agents ignore the new `Update backlog item` action type and keep creating near-duplicates. | Medium (habit) | Reinforce in `research-conclusion-authoring` guardrails ("before proposing Create, enumerate members and siblings; prefer Update if intent overlaps"). The shared context skill is the primary lever. |
| `swarm-manager-workshop-clarify` is updated but its scope is unclear; changes may be unnecessary. | Low | Before editing, read its current SKILL.md; if its scope is purely "ask clarifying questions" and never touches initiative context, skip and note in §12. |
| Orphaned items after initiative delete pile up and pollute lists. | Low | Orphaning is by design (see §8.1); the overview and initiative list views can filter on `initiative` presence. Future UI can surface orphans as a bucket. Not blocking for this plan. |

## 12. Non-Goals / Prohibited Patterns

- **No cascade responsibility in skill text.** Skills describe operations; the server enforces invariants. Any text that says "remember to update X when you delete Y" is a bug.
- **No compatibility shim.** No feature flag, no v2 endpoint, no "old-style delete" fallback. Greenfield per standing user feedback.
- **No batch-mutation API in this plan.** Multi-item reorg uses N CLI calls. Future work if profiling demands it.
- **No retroactive data repair.** Existing orphaned or dangling refs are not scanned or fixed by this plan. If the user wants a one-shot repair, that's a separate, bounded task.
- **No UI work.** Console surfacing of initiative context, orphaned items, or cascade events is out of scope.
- **No `--json` in agent-facing skill examples** (standing user feedback). Skill examples use human default output. `--json` appears only in automated tests or explicitly-parsing contexts.
- **No restart from the agent.** User runs `vrooli scenario restart ...` after code and skill edits.
- **No expansion of non-research workshop action vocabulary.** Non-research workshops continue to produce `plan.md`, not backlog mutations. They gain context loading, not mutation authority.
- **No new action types beyond the five listed in §8.6.** Do not add `Archive backlog item`, `Split backlog item`, or similar; those are future proposals if demanded.
- **No inference of "deletion intent" from vague conclusion language.** The action must be literally written. Process-research does not heuristically delete items from prose.
- **No cascade that also deletes items.** Initiative delete *orphans* items, never deletes them (§8.1).
- **Do not duplicate** `internal/initiatives` logic in the backlog package. The adapter is the one bridge. Cascade functions live where their data lives.

## 13. Definition of Done

All of the following are objectively true:

- [ ] API cascade invariants (§6 "API invariants") hold; all 13 cascade + context tests (T1-T16) pass.
- [ ] `GET /api/v1/initiatives/{name}/context` returns the specified shape; `initiatives context` CLI command exists with human and `--json` outputs; T14-T18 pass.
- [ ] `PATCH /api/v1/backlog/{kind}/{name}` and `POST /api/v1/backlog/{kind}` reject `initiative` fields pointing at non-existent initiatives (matching batch-create semantics).
- [ ] `swarm-manager-initiative-context` skill exists with valid `skill.json` and a SKILL.md that describes the endpoint, when to load, and the "reuse before create" heuristic.
- [ ] `research-conclusion-authoring/SKILL.md` defines six action types total: Create, Update backlog item, Delete backlog item, Update initiative, Update document, No further action. Revision bumped.
- [ ] `swarm-manager-process-research/SKILL.md` demonstrates CLI calls for all six action types, with the preamble framing the action list as non-additive. Revision bumped.
- [ ] Research workshop and research-finalize skills read in `swarm-manager-initiative-context` and include an explicit "load initiative context" step. Revisions bumped.
- [ ] Non-research workshop skills (`swarm-manager-workshop`, `-finalize`, `-clarify` where applicable) read in `swarm-manager-initiative-context` and include a "load context" step. Revisions bumped.
- [ ] Meta-orchestrator and backlog-tools skills document the new endpoint and the server-side cascade guarantees (one short section each). Revisions bumped.
- [ ] `scenarios/swarm-manager/docs/reference/cli-commands.md` documents `initiatives context` and the cascade semantics.
- [ ] `go build ./...`, `gofumpt -l` (scoped), `golangci-lint run ./...`, `go test ./... -race -count=1 -timeout 600s` all pass in `scenarios/swarm-manager/api`.
- [ ] `go build ./...` and `go test ./... -count=1 -timeout 300s` pass in `scenarios/swarm-manager/cli`.
- [ ] Skill JSON files all validate with `jq`.
- [ ] No skill text instructs agents on cascade bookkeeping. Grep-check: `grep -RIn "depends_on" scenarios/prompt-manager/store/skills/packs/core/swarm-manager-workshop*` shows no instructions to "remember to clean up depends_on."
- [ ] No compatibility shim, no feature flag, no batch-mutation endpoint, no `--json` in skill examples, no agent-initiated scenario restart.
- [ ] Plan-scope pre-existing issues resolved: (a) item patch validating initiative ref (was silent), (b) initiative delete handling the cascade (was silent), (c) AddItems/RemoveItems symmetry (was asymmetric).
