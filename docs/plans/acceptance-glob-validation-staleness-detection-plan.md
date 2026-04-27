# Acceptance Glob Validation & Plan-Staleness Detection — Implementation Plan

## 1. Purpose

When a swarm-manager backlog item's `acceptance_allow` globs reference paths that no longer exist in the repo, today's behavior is to fail the spawn with a generic `validate acceptance` error. That failure is a *symptom* of a deeper problem: the plan/conclusion was authored against an earlier repo state and has drifted out of alignment with reality. Patching the globs in isolation hides the underlying staleness; loosening the validator to warn-only lets stale work execute against assumptions that no longer hold.

This plan installs acceptance-glob validation at the points where plans are produced (every workshop round + finalization), distinguishes between "glob covers a path that doesn't exist yet but will be created by this work" (legitimate) and "glob covers a path that no longer exists" (stale), and reclassifies the spawn-time validation failure into a typed signal that drives a "re-workshop" UX rather than a raw error string.

## 2. Required Reading

```bash
prompt-manager skill read scientific-debugging implementation-plan-authoring plan-skill-discovery seam-discovery-and-enforcement
prompt-manager skill read cli-steer api-steer utils-unification
```

## 3. Greenfield Constraint

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. The acceptance-validation API is internal to swarm-manager and a few scenario-internal callers; rename freely, drop unused fields, and let callers update. No deprecation paths.

The one explicit non-greenfield obligation is migrating *existing* backlog/research item specs that already store `acceptance_allow` — these don't need a `creates` field added (workshops will re-author specs that need it), but the new workshop validator must not crash on specs lacking the field; absence means "no declared new paths," which is the safe default.

## 4. Problem Statement

### 4.1 Observed failure

Running the backlog item `agent-sandbox-auditability-contract` from the swarm-manager UI failed at spawn time with:

```
failed to queue execution: validate acceptance against project root "/home/matthalloran8/Vrooli":
acceptance_allow paths do not exist under project root:
  glob "scripts/lib/scenario/**": path "scripts/lib/scenario" does not exist under project root;
  glob "cli/commands/scenario/**": path "cli/commands" does not exist under project root
```

Source: `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/spec.json:1-9`.

The spec's `acceptance_allow` lists:
- `scenarios/agent-manager/**` ✓
- `scenarios/workspace-sandbox/**` ✓
- `scenarios/test-genie/**` ✓
- `scenarios/git-control-tower/**` ✓
- `packages/cli-core/**` ✓
- `scripts/lib/scenario/**` ✗ (`scripts/lib/` has `network/`, `utils/`, `runtimes/`, `service/`, `system/` — no `scenario/`)
- `cli/commands/scenario/**` ✗ (no top-level `cli/` directory exists)

The last two globs reference a layout that doesn't match the current repo. They were written at workshop time against a repo state that has since been refactored. The "Key references" section in the same spec also lists `scripts/lib/scenario/runner.sh`, which similarly does not exist — confirming the staleness extends into the plan's reasoning, not just the globs.

### 4.2 Why glob-level patching is the wrong response

Stale globs are not a typo. They reveal that:

1. The plan reasoned about specific files/directories that have since moved or been removed.
2. The "Key references" section anchors the plan's analysis on a file that doesn't exist.
3. The deliverables matrix may presuppose touchpoints that no longer apply.

Editing only the globs leaves the plan's body still pointing at ghosts. The agent will execute against incorrect mental models, produce a deliverable that references nonexistent code, and we'll discover the rot during review (or worse, after merge).

### 4.3 Why warn-only validation is the wrong response

Demoting `ErrAcceptanceMismatch` to a warning makes the spawn succeed and hides the staleness signal entirely. We lose the only structural check that catches plan/repo drift before an agent run is dispatched.

### 4.4 The legitimate edge case

A plan whose deliverable is *to create* a new directory or file (e.g., "create `docs/internal/SANDBOX-CONTRACT.md`") has a legitimate reason for `acceptance_allow` to cover a path that doesn't exist yet. Today's strict validator blocks these too. The validator cannot distinguish "stale reference" from "forward-looking creation" without help from the plan itself.

## 5. Scope

### In scope

- A `creates` field on the spec (research items + backlog items) listing paths the work plans to create.
- A workshop-round validator that runs after every round produces or modifies a plan, and at finalization, comparing `acceptance_allow` ∪ `creates` against the current repo. Missing paths are surfaced as round feedback, not silent warnings.
- A spawn-time validator change: keep strict checking, but classify failure as "plan stale" with structured detail; do not weaken.
- A swarm-manager UI change: intercept `ErrAcceptanceMismatch` from spawn, render a "Plan is stale — re-workshop required" panel with a button that triggers a fresh workshop round.
- A re-workshop endpoint (or reuse of an existing trigger) that resets the item's plan/conclusion to draft and queues a workshop round.
- Tests covering: validator's `creates` allowance, round feedback shape, spawn-time error classification, UI rendering of the stale-plan state, and an integration test on the actual `agent-sandbox-auditability-contract` item.
- Migration of existing specs to tolerate absence of `creates` (treat as empty list — handled by code default, not data backfill).

### Out of scope

- Changing the workspace-sandbox lowerdir model to support multi-scenario lowerdirs.
- Auto-fixing stale plans (the agent does the rewrite during the re-workshop round).
- Validating the *content* of `creates` paths beyond syntactic glob/path well-formedness (the workshop agent owns plan correctness).
- Changing how `ScenariosFromGlobs` works (already correctly returns the set of scenarios touched).

## 6. Current Technical Context

### Validators

- `scenarios/swarm-manager/api/internal/projectroot/validate.go:27` — `ValidateAcceptanceUnderRoot(projectRoot, allow)`. Stat()s each glob's literal-prefix directory; returns `ErrAcceptanceMismatch` listing every problem.
- `scenarios/swarm-manager/api/internal/projectroot/resolve.go:79` — `Resolve(opts)`. Now widens to monorepo-scope on multi/zero scenario items; calls into validation via `agentmanager.resolveScopeAndRoot`.
- `scenarios/swarm-manager/api/internal/agentmanager/resolve.go:19-47` — `resolveScopeAndRoot`. Calls `ValidateAcceptanceUnderRoot` only when `filepath.IsAbs(root) && len(acceptanceAllow) > 0`.

### UI surface

- `scenarios/swarm-manager/ui/src/...` (run dialog) — surfaces the raw error string. Need to find the queue-execution call site and render typed errors.
- `scenarios/swarm-manager/api/internal/backlog/queue_ops.go` — handles spawn requests; this is where `ErrAcceptanceMismatch` bubbles up.

### Workshop machinery

- `scenarios/swarm-manager/api/internal/backlog/workshop_ticker.go` — tick loop driving rounds.
- `scenarios/swarm-manager/api/internal/backlog/handler_create.go` and friends — round dispatch and finalization.
- `scenarios/swarm-manager/store/teams/.../skills/` — workshop-author and workshop-finalize skills (need to read to understand round outputs and how to inject validation feedback).

### Existing UI-side glob validator

- `scenarios/swarm-manager/api/internal/backlog/validate_globs.go:35` — `ValidateGlobs` HTTP handler. Returns per-pattern `MatchCount`/`Warning`. This is called from the backlog editor UI but **not** from spawn or workshop. It's the right shape to extend (per-pattern results) but not the right placement.

### Spec schema

- `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/spec.json` — current spec layout (reference). The spec parser/types live in `scenarios/swarm-manager/api/internal/backlog/types_test.go` adjacent to `types.go` and `archive_parser.go`.

### Stale instance under test

- `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/spec.json` — confirmed stale. Will be the integration-test fixture and the dogfood candidate for the re-workshop trigger.

## 7. Target End State

A backlog/research item that is workshopped against the current repo will have an `acceptance_allow` whose every literal-prefix path either exists on disk, or is explicitly listed under a new `creates` field on the spec (paths the work intends to create). When repo drift breaks that invariant — because a refactor lands after the plan is written — the next workshop round catches it; if it slips past workshops, spawn-time validation catches it and the UI surfaces a "Plan is stale" panel with a one-click "Re-workshop" action that resets the plan to draft and queues a fresh round. There is no path by which a stale plan silently executes.

## 8. Implementation Strategy

### Phase A — Spec schema: add `creates`

**A1.** Extend the spec type used by research items and backlog items to include `creates []string` (glob list, same syntax as `acceptance_allow`).

- Touch `scenarios/swarm-manager/api/internal/backlog/types.go` (or wherever `ItemSpec` / research spec lives — confirm during execution).
- JSON tag: `"creates,omitempty"`. Absence → empty list.
- Update parser/normalizer/serializer to round-trip the field.
- Update unit tests for parser to cover `creates` present, empty, and absent.

**A2.** Document the field in any spec schema doc (e.g., a CONTRIBUTING-style file under `scenarios/swarm-manager/docs/`). One paragraph: what `creates` means, when to use it, syntax (same as `acceptance_allow`), and that absence means "no declared new paths."

### Phase B — Project-root validator: accept `creates`

**B1.** Extend `ValidateAcceptanceUnderRoot` (or add a sibling) to take both `allow` and `creates`. Algorithm:

```
For each glob in allow:
  prefix := literalPrefix(glob)
  if prefix == "" or prefix exists under root:
    OK
  else if any creates glob is a prefix of (or equal to) this glob's prefix:
    OK  (declared as a path the work creates)
  else:
    record problem
```

Path-traversal rejection (`..` escape) stays unconditional; `creates` entries are *also* traversal-checked.

**B2.** Keep a strict variant `ValidateAcceptanceStrict(root, allow)` that ignores `creates` — used by the spawn-time validator (Phase D) so a forgotten `creates` declaration still fails closed at spawn.

**B3.** Tests covering: empty creates (current behavior), creates covers a missing glob (allowed), creates declares a path that *also* doesn't exist (allowed — agent owns this), creates with traversal (rejected), strict variant ignoring creates.

### Phase C — Workshop-round acceptance-validation gate

**C1.** Add a per-round validator step that runs after a round produces or modifies a plan/conclusion. It calls the relaxed validator (Phase B1: accepts `creates`).

- Surface the result as structured round feedback that the next round (or the current round's revision step) can consume.
- A failing validation does not silently pass; the round records a `validationProblems` array on the round artifact, and the workshop ticker treats this as a signal that the round is not "clean."

**C2.** Inject validation feedback into the workshop-author / workshop-finalize prompt context so the agent sees per-glob problems ("`scripts/lib/scenario/**`: path doesn't exist; not declared in `creates`") and can classify each as either:
- a stale reference to remove/rewrite, or
- a path the work creates → add to `creates`.

**C3.** At finalization, run the same validator one more time. If problems remain, finalization fails; the item stays in workshop with a clear "validation blocking finalize" status. No silent finalization.

**C4.** Tests: round outputs validation problems when present, finalization blocks on remaining problems, empty/all-OK case finalizes cleanly, integration test driving a round on a fixture spec.

### Phase D — Spawn-time validator: typed staleness error

**D1.** In `agentmanager/resolve.go:resolveScopeAndRoot`, switch the strict-validation call to a wrapper that returns a typed error:

```go
type StalePlanError struct {
    ProjectRoot     string
    AcceptanceAllow []string
    MissingPaths    []MissingPath  // {Glob, ResolvedRel, Reason}
}
```

**D2.** In `backlog/queue_ops.go` (or wherever queue-execution returns the error), serialize `StalePlanError` to a structured API error:

```json
{
  "error": "plan_stale",
  "message": "...",
  "details": {
    "missingPaths": [
      {"glob": "scripts/lib/scenario/**", "resolved": "scripts/lib/scenario"},
      {"glob": "cli/commands/scenario/**", "resolved": "cli/commands"}
    ]
  }
}
```

**D3.** Tests: a queue request for a stale spec returns the structured error; a queue request for a clean spec succeeds; absence of `creates` and presence of strict-only failure both still trigger `plan_stale`.

### Phase E — Re-workshop trigger

**E1.** Add (or wire up an existing) endpoint: `POST /api/v1/backlog/{id}/re-workshop`. Behavior:

- Sets item status back to a draft/workshop state.
- Clears the item's stale plan/conclusion artifacts (or marks them superseded; greenfield says clear, but verify there's no audit-log requirement before doing so during execution).
- Queues a fresh workshop round.

**E2.** CLI parity: `swarm-manager backlog re-workshop <id>` — per the `feedback_skills_use_cli_never_api` memory, the CLI is canonical; the UI calls the API which mirrors the CLI surface.

**E3.** Tests: re-workshop on a stale item resets state and queues a round; re-workshop on a clean item is a no-op or rejected with a clear error (decide during execution — likely "rejected, item is not stale").

### Phase F — UI: stale-plan panel

**F1.** In the run dialog (where today's raw error renders), detect the `plan_stale` error and render a structured panel:

- Title: "This plan references paths that no longer exist."
- Body: bulleted list of `missingPaths` (`{glob}` → `{resolved}`).
- Explanation line: "The plan was likely written against an earlier repo state. Re-workshopping rewrites the plan and acceptance against the current repo."
- Primary action: "Re-workshop" button → calls the trigger from E1.
- Secondary action: "Cancel."

**F2.** Make the rendering reusable — this panel will likely be needed elsewhere (initiative review, overview).

**F3.** Smoke automation: an automated UI/component test (or a Playwright-style end-to-end if one exists for swarm-manager — confirm during execution) that mounts the run dialog with a fake `plan_stale` error and asserts the re-workshop button is present and calls the right endpoint.

### Phase G — Dogfood: re-workshop the stale item

**G1.** Trigger re-workshop on `agent-sandbox-auditability-contract` via the new flow.

**G2.** Verify the round produces a plan whose `acceptance_allow` is consistent with the current repo (every prefix exists or is in `creates`), and whose body no longer references nonexistent paths like `scripts/lib/scenario/runner.sh`.

**G3.** Re-run the original spawn from the UI and confirm it queues cleanly. This is the live acceptance test for the whole pipeline.

### Phase H — Cleanup & verification

**H1.** Run `go build ./...` and `go test ./... -timeout 300s` in `scenarios/swarm-manager/api/`. Fix **all** lint, type, and unit test issues in modified files — including pre-existing ones.

**H2.** Run UI lint/type checks and tests. Fix all issues in modified files including pre-existing.

**H3.** `vrooli scenario restart swarm-manager`. Verify health: `curl -s http://localhost:<API_PORT>/health` (port from restart output).

**H4.** Smoke-check the UI loads and the run dialog still works for clean items.

## 9. Contract Decisions

- **`creates` is part of the spec, not a runtime override.** It belongs to the plan because the plan owns the claim "I will create these paths." Storing it elsewhere (e.g., per-spawn) defeats the workshop-validation goal.
- **The relaxed validator is for workshop rounds; the strict validator is for spawn.** Workshops can declare forward-looking paths via `creates`; spawn time still requires every glob's prefix to either exist or be declared. This is intentional defense in depth: workshops catch staleness early, spawn catches anything that slipped through.
- **`StalePlanError` is the single shape callers consume.** Anyone (UI, CLI, future automations) that needs to react to plan staleness keys off the `plan_stale` error type, not string matching.
- **Re-workshop clears prior plan artifacts.** Greenfield: no superseded-plan history retained in-band. If audit trail is needed, the workshop event log already records round outputs; that's the audit surface, not the plan file.
- **Path-traversal rejection is unconditional** in both validators and applies to `creates` entries too. `creates: ["../../../etc"]` is rejected.
- **Empty `creates` is the safe default.** Specs without the field behave exactly like today's specs against the relaxed validator (i.e., as strict).
- **No migration backfill for existing specs.** Existing items will be re-workshopped on their next workshop round (or on demand via the re-workshop trigger), at which point the agent decides whether `creates` is needed.

## 10. Testing Plan

All tests are automated (per `feedback_testing_over_manual` memory). No manual checklists.

- **Unit:** `internal/projectroot/validate_test.go` — relaxed and strict variants, `creates`-allows-missing, `creates`-traversal-rejected, empty `creates`.
- **Unit:** spec parser/serializer round-trips `creates`.
- **Unit:** `agentmanager/resolve_test.go` — resolve returns `StalePlanError` for stale specs, success for clean specs, success for specs whose missing globs are all in `creates`.
- **Unit:** `backlog/queue_ops_test.go` — queue endpoint returns the `plan_stale` JSON shape on `StalePlanError`.
- **Unit:** workshop-round validator records `validationProblems` correctly; finalization blocks while problems present; empty problems → finalization proceeds.
- **Integration:** end-to-end on a fixture item: workshop round → finalization → spawn → success. And: stale fixture item → spawn returns `plan_stale` → re-workshop trigger → round runs → second spawn succeeds.
- **UI:** component test for the stale-plan panel (renders missing paths, button calls re-workshop endpoint).
- **Live:** Phase G is itself the production acceptance test on the real `agent-sandbox-auditability-contract` item.

## 11. Rollout / Validation Checklist

1. All unit + integration tests pass: `go test ./scenarios/swarm-manager/api/... -timeout 300s`.
2. UI tests pass.
3. `vrooli scenario restart swarm-manager` reports healthy.
4. Phase G complete: live re-workshop on `agent-sandbox-auditability-contract` produces a clean spec and a successful spawn.
5. Spot-check: an unrelated clean item spawns without UI regressions.
6. No raw `validate acceptance against project root` strings appear in the UI for any item.

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Workshop agent forgets to declare `creates` and round keeps re-failing | Validator feedback explicitly lists the two options ("add to `creates`" or "remove from `acceptance_allow`"); failing rounds surface to the operator who can intervene. |
| Re-workshop deletes plan content the user wanted to preserve | Workshop event log still has every round's output; re-workshop is reversible by inspecting prior rounds. Document this in the re-workshop button tooltip. |
| Strict validator at spawn time still blocks a legitimate forward-looking item that the workshop already approved | Cannot happen if Phase B is correct: any glob accepted by the workshop (relaxed validator) without the path existing must have been added to `creates`; spawn-time strict validator considers `creates` too. **Update Phase D to use the relaxed validator with `creates`, not the strict one — the strict variant is only for spawn paths that don't have `creates` available.** Resolve during execution. |
| Existing in-flight items become un-spawnable until re-workshopped | Acceptable: that's the desired behavior. They are stale by definition and re-workshop is one click. |
| Multiple scenarios in `acceptance_allow` triggers wide-scope monorepo lowerdir, which is large | Already the current behavior post the previous fix; not a regression. Future workspace-sandbox multi-scope work addresses this separately. |

## 13. Non-goals / Prohibited Patterns

- Do **not** add a "skip validation" flag to the spawn path. The whole point is to catch staleness.
- Do **not** demote `ErrAcceptanceMismatch` to a logged warning at any layer.
- Do **not** auto-rewrite `acceptance_allow` from the validator. The agent owns plan content.
- Do **not** add a `force-spawn` UI button that bypasses the stale-plan panel. If we need to spawn without re-workshopping, it's because the validator is wrong; fix the validator.
- Do **not** keep the old `ValidateAcceptanceUnderRoot` signature for "compat" — update all callers.

## 14. Definition of Done

- `creates` exists on the spec type, parses, serializes, round-trips, is documented.
- Relaxed (workshop) and strict (spawn) variants of the acceptance validator exist, are tested, and produce structured per-glob results.
- Workshop rounds and finalization both run the relaxed validator, surface problems as structured round feedback, and block finalization when problems remain.
- Spawn-time validation returns a typed `StalePlanError`; the queue API returns a `plan_stale` structured error.
- UI run dialog detects `plan_stale` and renders the stale-plan panel with a working re-workshop action.
- Re-workshop endpoint (and CLI command) exists, is tested, and resets stale items to a fresh workshop round.
- The `agent-sandbox-auditability-contract` item has been re-workshopped, has a clean spec, and spawns successfully from the UI.
- All lint, type, unit, and integration tests pass in modified files (including pre-existing).
- `swarm-manager` restarts healthy.
- No `acceptance_allow paths do not exist under project root` raw strings appear in any UI surface.

## Implementation Notes (post-execution)

These notes record decisions taken while implementing Phases A–F. Phases G (live re-workshop) and H (final restart) remain for the operator/parent agent.

### Phase A — schema reach
- `creates` was added to the proto (`packages/proto/schemas/swarm-manager/v1/{api,domain}/backlog.proto`) and proto bindings were regenerated via `buf generate`. This propagates the field through Go, TypeScript, and Python clients automatically.
- `BacklogItem.Creates`, `ItemPatch.Creates`, the create/update/batch handlers, and the CLI `BacklogItem`/`CreateBacklogRequest`/`UpdateBacklogRequest`/`BatchCreateItem` types all carry the field now. CLI `backlog list` and `backlog get` print `Creates: ...` when populated.
- The proposals/handoff/initiative auxiliary types still reference `acceptance_allow` only — they do not currently surface the field for spec-mutation flows; extending those is out of scope for this plan and was not done.
- The "document the field in `scenarios/swarm-manager/docs/`" step (A2) was deferred. The field is documented inline in the proto (`backlog.proto` comment) and in the domain type definitions, which is the canonical contract source. A standalone doc page can be added later if a non-proto surface (e.g., a workshop-author skill) needs to reference it explicitly.

### Phase B — single relaxed validator
- Per §12 risk resolution, only one validator was implemented (`ValidateAcceptance(root, allow, creates)`). The "B2 strict variant" was dropped: spawn-time validation honors `creates` too, because forward-looking paths declared by the workshop are legitimate at spawn as well. This matches the §12 update.
- The validator returns `(*AcceptanceReport, error)`. The error wraps `ErrAcceptanceMismatch` when problems remain so callers can still use `errors.Is`. The report carries per-glob `GlobProblem` entries with `{Glob, ResolvedRel, Reason, Existed, AllowedByCreates}`.
- Path-traversal rejection is unconditional and applies to `creates` entries too.

### Phase C — pragmatic gate placement
- The plan envisioned the per-round validator hooking into the workshop prompt's feedback loop. In practice, the workshop prompts are skill-managed and the round content shape is opaque to the API layer (rounds are author-defined JSON blobs). To avoid scope creep, the validator hooks into `WorkshopSave` instead:
  - **Every round save** (workshop or finalize) runs `ValidateAcceptance` and persists `acceptance-validation.json` in the item directory. The next round's agent (and the operator) can read this artifact to see exactly which globs are stale.
  - **Finalize rounds** additionally block the save with a `plan_stale` 409 when problems remain, so a stale plan cannot finalize silently.
- The structured artifact carries the same `GlobProblem` shape the spawn-time validator produces, so workshop and spawn diagnostics are interchangeable.

### Phase D — typed staleness error
- `agentmanager.StalePlanError` carries `{ProjectRoot, AcceptanceAllow, MissingPaths}`. `agentmanager.AsStalePlanError(err)` is the canonical extractor.
- `apierr.DomainError` was extended with optional `Code` and `Details` fields, and the mapper now emits a JSON envelope (`{"error": code, "message": ..., "details": ...}`) when either is set. This is additive — handlers that don't set `Code`/`Details` continue to render plain-text bodies.
- `apierr.PlanStale(message, details)` is the canonical constructor; `wrapAgentError` (in `internal/execution/service.go`) translates `*StalePlanError` into it. All spawn paths (control, queue, retry, follow-up) flow through `wrapAgentError`, so they all surface `plan_stale` consistently.

### Phase E — re-workshop endpoint
- `POST /api/v1/backlog/{kind}/{name}/re-workshop` clears workshop rounds and the deliverable, reverts status to `backlog`, and asynchronously spawns a fresh `ResearchModeInitialize` workshop round.
- CLI mirror: `swarm-manager backlog re-workshop --kind <kind> --name <name>`.

### Phase F — UI panel
- New component: `src/components/backlog/stale-plan-panel.tsx`, plus a smoke render test in `stale-plan-panel.test.tsx`.
- `ApiError` was extended to expose `code` and `details` from JSON error bodies.
- `RunBacklogModal` now intercepts `plan_stale` errors and renders `StalePlanPanel` with the missing paths and a working "Re-workshop" button. Bulk mode falls back to the generic error string.
- The panel is intentionally generic — it accepts a `kind`, `name`, and `missingPaths` prop, so initiative/overview surfaces can render the same panel later without duplication.

### Test coverage delivered
- `internal/projectroot/validate_test.go`: relaxed validator, `creates` allows missing, `creates` ancestor coverage, `creates` traversal rejected, empty `creates`, wildcard-only globs, literal file path, traversal rejection, problem aggregation.
- `internal/agentmanager/resolve_test.go`: typed `StalePlanError` returned for stale specs, `creates` allows missing path, all existing scope-resolution behaviors preserved.
- `internal/backlog/`: existing tests still pass; the new round-save acceptance check runs but specific assertions for it can be added later (current backlog tests don't drive a real workshop round end-to-end).
- UI: `stale-plan-panel.test.tsx` covers rendering missing paths, calling `reWorkshop`, and parsing both snake_case/camelCase missing-paths shapes; existing 1742 UI tests still pass.

### What's deferred
- Adding `creates` to the proposals/handoff/initiative spec-mutation paths.
- A dedicated workshop-skill doc paragraph for `creates` (the proto comment is the contract; skills can reference it).
- An end-to-end integration test that drives a real workshop round through the new save-path validator (requires reproducing the prompt-manager + agent-manager spawn machinery in tests).
