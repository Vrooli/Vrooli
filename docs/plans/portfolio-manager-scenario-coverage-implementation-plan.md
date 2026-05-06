# Portfolio-Manager Scenario-Coverage Visibility — Implementation Plan

## 1. Purpose

Give the swarm-manager's `portfolio-manager` agent (and human operators via the UI) a reliable way to enumerate **"all initiatives and backlog items targeting scenario X"** before proposing new containers, so readiness-style initiatives are proposed as **umbrellas that depend on existing coverage** rather than as parallel scatter. Also harden the morning-vision-walk divergence machinery that produced this plan so checkpoints survive overnight prep-agent regeneration.

## 2. Required Reading

Before executing:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read skill-principles skill-validation
```

Spec files to read for context:

```bash
cat scenarios/swarm-manager/cli/cmd_backlog.go            # existing --scenario precedent (lines 19-42)
cat scenarios/swarm-manager/cli/cmd_initiatives.go        # target file for new flag + command (lines 26, 152)
cat scenarios/swarm-manager/api/internal/initiatives/handler.go
cat scenarios/swarm-manager/api/internal/pathutil/root.go # ScenariosFromGlobs (lines 49-71)
cat scenarios/swarm-manager/api/internal/backlog/types.go # acceptance_allow field (lines 96-117)
cat scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx   # BacklogScenariosPanel usage (line 469)
cat scenarios/swarm-manager/ui/src/components/backlog/backlog-scenarios-panel.tsx  # reusable chip
cat scenarios/prompt-manager/store/agents/portfolio-manager/TOOLS.md
cat scenarios/prompt-manager/store/agents/portfolio-manager/AGENTS.md
cat scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md
cat scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/SKILL.md   # for reference only — edits already landed
```

## 3. Problem Statement

**Observed 2026-04-24 vision walk**: `portfolio-manager` proposed creating a `web-console-readiness` initiative (`dec-1776982737575948642`) claiming web-console had *no* dedicated readiness coverage. This was factually wrong — the scenario has **7 active backlog items** under the `continuous-audio-platform` initiative targeting `path:scenarios/web-console/**` globs, plus one fresh item (`literal:execute/web-console-tts-code-tick-handling`). The agent did not enumerate existing coverage before proposing.

**Root cause (layered):**

1. **Agent-skill gap (primary).** The scenario-filtering CLI `swarm-manager backlog list --scenario <name>` already exists (`cmd_backlog.go:24, 40–42`) and the API already filters (`handler_query.go:44, 291–310`). But portfolio-manager's `TOOLS.md` and `AGENTS.md` do not document it, and the agent workflow has no "detect existing coverage" step.
2. **CLI/API gap (secondary).** `swarm-manager initiatives list` has no `--scenario` filter; nor does any command return "all initiatives containing items targeting scenario X" in a single call. Portfolio-manager (and operators) must fetch-then-filter client-side.
3. **UI gap (tertiary).** `ScenarioDetailsPage.tsx` has no "Associated initiatives / backlog items" section. `InitiativeDetailsPage.tsx` has no "Targeted scenarios" section (a reusable chip panel already exists on `BacklogDetailsPage.tsx` — `BacklogScenariosPanel`).
4. **Walk-state gap.** The morning-vision-walk divergence that produced this plan writes a `## Walk Checkpoint` section into `last-handoff.md`, but the `vision-walk-prep` agent regenerates that file at 5:00 AM and currently would overwrite the checkpoint. The skill edits landed (2026-04-24) rely on the prep agent preserving it — that preservation logic does not yet exist.

**Desired future behavior**: when portfolio-manager proposes a readiness initiative for scenario X, it has already (a) enumerated every initiative and backlog item targeting X, (b) proposed the readiness initiative as an **umbrella** with `depends_on` edges to the existing coverage, (c) added only net-new verification/testing items on top. Completion of the umbrella == production-ready.

## 4. Scope

**In scope**
- `portfolio-manager` agent-skill updates (TOOLS.md + AGENTS.md).
- `swarm-manager` CLI: add `--scenario` flag to `initiatives list`; add `initiatives context --scenario <name>` command (new mode of existing command).
- `swarm-manager` API: add scenario-filter support to the `GET /initiatives` list endpoint; extend `GET /initiatives/<name>/context` OR add `GET /scenarios/<name>/context` for the scenario-scoped rollup.
- `swarm-manager` UI: `ScenarioDetailsPage` — associated initiatives + backlog items section; `InitiativeDetailsPage` — targeted scenarios chip section (reuse `BacklogScenariosPanel`).
- `vision-walk-prep` agent HEARTBEAT.md update: preserve `## Walk Checkpoint` section across daily regenerations.

**Out of scope**
- Storing `target_scenarios` as a first-class field on initiatives (keep derivation from `acceptance_allow` globs via `pathutil.ScenariosFromGlobs`).
- Graph topology lens changes (already works; not touching it).
- Retroactive re-tagging of existing web-console items under a new initiative (that's a *portfolio-manager* decision for its next heartbeat, not a plan task here).
- Resolving `dec-1776982737575948642` itself (will re-surface naturally after portfolio-manager's next heartbeat; operator decides then).
- Refactoring `path:swarm-manager/cli/` flat `cmd_*.go` files into cli-steer's preferred domain-package layout (per cli-steer Section 4: "`cmd_<domain>.go` may exist in legacy CLIs" — don't refactor just for this change).

## 5. Hard Constraints (Greenfield)

**Greenfield, not migration.** No compatibility shims, no dual-path code, no "old vs new" flag gates:

- `GET /initiatives?scenario=X` is an additive query param on the existing endpoint. It either works correctly or the endpoint is broken; no feature flag, no fallback path.
- The CLI flag is additive; no deprecation window needed for anything.
- The UI sections are additive; no fallback rendering for "old" API responses that don't return the new data (the API and UI ship together).
- Agent-skill docs are rewritten in place; no "see also: old workflow" notes.

If a task surfaces an existing bug (e.g., `ScenariosFromGlobs` misparsing a glob, or `acceptance_allow` validation missing a case), **fix it as part of this plan** rather than bypassing it. Do not land code that works-around a pre-existing issue.

## 6. Current Technical Context

### 6.1 Data model (how scenario targeting is stored)

- Backlog items: `acceptance_allow []string` field in `path:scenarios/swarm-manager/api/internal/backlog/types.go:96-117`. Glob patterns like `path:scenarios/web-console/**`.
- Initiatives: `Items []string` (kind/name refs) in `path:scenarios/swarm-manager/api/internal/initiatives/model.go:7-19`. Initiatives have no direct scenario-targeting field — their "targeted scenarios" are **derived** by union-ing their member items' `acceptance_allow` globs.
- Derivation helper: `pathutil.ScenariosFromGlobs()` in `path:scenarios/swarm-manager/api/internal/pathutil/root.go:49-71`.

### 6.2 Existing scenario-filter precedent (DO emulate)

- `swarm-manager backlog list --scenario <csv>` (CLI: `cmd_backlog.go:19-85`, API handler: `handler_query.go:44` plus `filterByScenario()` at lines 291-310). The flag accepts comma-separated scenario names; API parses `?scenario=a,b,c` and filters items whose `acceptance_allow` globs resolve (via `ScenariosFromGlobs`) to any of the requested scenarios.

### 6.3 Existing scenario-less commands to extend

- `swarm-manager initiatives list` — `cmd_initiatives.go:26-86`. Currently no filters.
- `swarm-manager initiatives context --name <initiative>` — `cmd_initiatives.go:152-236`. Returns one initiative's context (rollup + members + upstream/downstream). Takes `--name`; we'll add `--scenario` as a mutually-exclusive alternative mode.

### 6.4 UI surfaces

- `path:scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.tsx` — page exists; needs new section.
- `path:scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx` — page exists; needs new section.
- `path:scenarios/swarm-manager/ui/src/components/backlog/backlog-scenarios-panel.tsx` — reusable chip panel. Used at `BacklogDetailsPage.tsx:469, 548` via `<BacklogScenariosPanel targetScenarios={targetScenarios} />`.

### 6.5 Agent files

- `path:scenarios/prompt-manager/store/agents/portfolio-manager/TOOLS.md` — toolkit surface.
- `path:scenarios/prompt-manager/store/agents/portfolio-manager/AGENTS.md` — workflow (step 4 proposes new initiatives; no pre-check step exists).

### 6.6 Walk-prep agent

- `path:scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md` — instructions for the 5 AM `last-handoff.md` regeneration. Must be updated to preserve `## Walk Checkpoint` sections.

## 7. Target End State

A future agent running against master after this plan ships can:

1. Run `swarm-manager backlog list --scenario web-console` and get every item targeting web-console. *(Works today — just needs docs.)*
2. Run `swarm-manager initiatives list --scenario web-console` and get every initiative containing items targeting web-console, with per-initiative completion rollups. *(New.)*
3. Run `swarm-manager initiatives context --scenario web-console` and get a single-call rollup: every initiative + every item + combined completion stats + "orphan" items (items targeting the scenario but not in any initiative). *(New.)*
4. Open the scenario details page in the UI and see, inline, all associated initiatives and backlog items with their status — no graph-topology-lens hunting needed.
5. Open any initiative details page and see clickable scenario chips for everything its member items target.
6. `portfolio-manager`, at the start of every heartbeat, runs the scenario-coverage enumeration for any scenario it is considering proposing a new initiative for, and frames proposals as umbrellas with `depends_on` edges to existing coverage.
7. Vision-walk divergence checkpoints survive overnight prep-agent regeneration. If a walk diverges on day N and the operator resumes on day N+1, the checkpoint is still in `last-handoff.md`.

## 8. Implementation Strategy (Phased)

Phases are ordered so each phase is independently mergeable and incrementally improves the surface. Phases 1, 2, 3 have no mutual dependencies (can be parallel). Phase 4 depends on Phase 2. Phase 5 depends on Phase 2 + 3. Phase 6 is independent.

### Phase 1 — Agent-skill updates (small; unblocks today's behavior)

**Files**
- `path:scenarios/prompt-manager/store/agents/portfolio-manager/TOOLS.md`
- `path:scenarios/prompt-manager/store/agents/portfolio-manager/AGENTS.md`

**Changes**
1. `TOOLS.md`: add a "Scenario coverage enumeration" section listing:
   - `swarm-manager backlog list --scenario <name>` — all backlog items targeting scenario
   - `swarm-manager initiatives list --scenario <name>` — all initiatives whose items target scenario *(ships in Phase 2; reference by name)*
   - `swarm-manager initiatives context --scenario <name>` — single-call rollup *(ships in Phase 3; reference by name)*
2. `AGENTS.md`: add a new step 4 (renumbering downstream steps): **"Detect existing coverage"** — before proposing any new initiative that has a scenario-specific name (`<scenario>-readiness`, `<scenario>-launch-prep`, etc.), run the three coverage commands above for that scenario. If ≥1 existing initiative or ≥1 existing backlog item targets the scenario, frame the proposal as an **umbrella** with explicit `depends_on: [<existing-initiatives>]` and scope its new items to the *gaps* in coverage.

**Merge order note:** The `TOOLS.md` entries for commands that don't exist yet (Phase 2, 3) must be gated with a "ships in <plan-id> Phase N" note until those phases land. After Phase 2 and 3 land, remove the gate notes.

### Phase 2 — `swarm-manager initiatives list --scenario` (API + CLI)

**Files**
- API handler: `path:scenarios/swarm-manager/api/internal/initiatives/handler.go` (add scenario filter to list endpoint).
- API query logic: reuse/extend `pathutil.ScenariosFromGlobs` to resolve each initiative's aggregated targeted scenarios (union of member items' `acceptance_allow`).
- CLI: `path:scenarios/swarm-manager/cli/cmd_initiatives.go:26` — add `scenarioFlag := fs.String("scenario", "", "Comma-separated scenario names to filter by")`; pass as `?scenario=...` query param.

**Behavior**
- `GET /initiatives?scenario=web-console,command-center` returns initiatives where at least one member item's `acceptance_allow` globs, when resolved by `ScenariosFromGlobs`, include any of the requested scenarios.
- CLI echoes the filter in the "Summary" section (mirror `cmd_backlog.go:74-76`).
- Empty result: human-readable "No initiatives found targeting scenario(s): X" with a Next-Steps block suggesting `swarm-manager backlog list --scenario X` for items-without-an-initiative.

### Phase 3 — `swarm-manager initiatives context --scenario <name>` (API + CLI)

**Files**
- API: new endpoint `GET /scenarios/<name>/context` (preferred — matches REST resource shape per api-steer §3.1) in `path:scenarios/swarm-manager/api/internal/initiatives/handler.go` or a new `handler_scenario_context.go`. Alternative: overload `GET /initiatives/<name>/context` with a `?mode=scenario` param. **Recommended: new endpoint** for clarity.
- CLI: `path:scenarios/swarm-manager/cli/cmd_initiatives.go:152` — rework `cmdInitiativesContext` so `--name` and `--scenario` are mutually exclusive; when `--scenario` is set, call the new endpoint.

**Response shape** (proto-first; add message to `path:packages/proto/.../initiatives_service.proto` or equivalent):

```proto
message ScenarioContextResponse {
  string scenario_name = 1;
  repeated InitiativeSummary initiatives = 2;   // every initiative with ≥1 item targeting this scenario
  repeated BacklogItemSummary orphan_items = 3; // items targeting this scenario but in no initiative
  RollupStats rollup = 4;                        // union of all items: total/completed/in_progress/failed/pending
}
```

**Behavior**
- Returns the full coverage picture for a scenario in one call.
- Orphan-items section is critical — this is the primary signal that a readiness initiative is needed (items exist but no container).
- CLI output: follows Data Retrieval contract (cli-steer §9.3) — Summary → Initiatives list → Orphan items → Rollup → Next Steps.

### Phase 4 — UI: Initiative details page — "Targeted scenarios" section

**Files**
- `path:scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx`
- `path:scenarios/swarm-manager/ui/src/components/backlog/backlog-scenarios-panel.tsx` — reuse as-is; do not duplicate.

**Changes**
- Compute `targetScenarios` as union of member items' derived scenarios (API should return this as part of initiative context — Phase 2 ships this, confirm the `InitiativeContextResponse` includes per-item scenario lists or a top-level `target_scenarios` aggregation).
- Render `<BacklogScenariosPanel targetScenarios={targetScenarios} />` in a new section between the existing rollup and members sections.
- Tests: extend `InitiativeDetailsPage.test.tsx` — assert the panel renders the expected chips when API returns items with multiple `acceptance_allow` globs.

**If the API does not yet return target_scenarios on the initiative context endpoint**, extend Phase 2 to include it (additive proto field). Do not compute it in the UI — keep derivation server-side to avoid drift.

### Phase 5 — UI: Scenario details page — "Associated initiatives & backlog items" section

**Files**
- `path:scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.tsx`

**Changes**
- Call the new `GET /scenarios/<name>/context` endpoint from Phase 3.
- Render three subsections:
  1. **Initiatives** — one card per initiative, showing name, status, priority, completion rollup, link to initiative details.
  2. **Orphan backlog items** — one row per item (targets this scenario but not in any initiative), with link to backlog details. This is the section that most obviously signals "a readiness initiative would help."
  3. **Rollup** — combined totals.
- Tests: extend `ScenarioDetailsPage.test.tsx` — assert empty-state messaging, assert each subsection renders correctly with mocked API response.

### Phase 6 — `vision-walk-prep` agent: preserve `## Walk Checkpoint`

**Files**
- `path:scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md`

**Changes**
- Before generating the new `last-handoff.md`, read the existing `last-handoff.md` (if present).
- If it contains a `## Walk Checkpoint (<timestamp>)` section (exact heading match, any timestamp), extract it verbatim.
- After generating the new handoff content, append the preserved checkpoint section verbatim at the end of the new file.
- The checkpoint is removed by the `morning-vision-walk` skill at Phase 9 of a resumed walk (already specified in the skill). The prep agent does *not* decide when to remove it — it only preserves until the skill wipes it.
- **Edge case**: if multiple `## Walk Checkpoint` sections somehow accumulate, preserve all of them (indicates multiple un-resumed divergences — operator will see them all on resume).

## 9. Contract Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Is scenario-targeting on initiatives a stored field? | **No.** Derived from member items' `acceptance_allow`. | Single source of truth; avoids drift; matches existing model. |
| API shape for scenario rollup | **New endpoint `GET /scenarios/<name>/context`** | api-steer §3.1: new bounded context → new domain module/endpoint. Overloading `/initiatives/<name>/context` muddles the resource. |
| CLI command shape | `initiatives context --scenario <name>` | Reuses existing command surface; no new top-level command; matches cli-steer §5.2 mapping. Alternative `scenario context --name` rejected because no `scenario` top-level CLI group exists yet (introducing one is out of scope). |
| `--scenario` flag accepts | CSV (e.g., `--scenario web-console,command-center`) | Matches `backlog list --scenario` precedent (`cmd_backlog.go:24`). |
| Empty-result rendering | Human-readable "no X found" + Next-Steps block | cli-steer §9.3 Data Retrieval contract. |
| Proto compatibility | **Additive only.** New fields get new field numbers; no renumbering, no semantic changes. | api-steer §9.1. |
| UI data source | Server-derived `target_scenarios` | No client-side glob parsing; API owns derivation. |
| Portfolio-manager enforcement | Soft (doc-based); agent reads AGENTS.md and self-enforces | Hard enforcement via API would require the agent to declare intent to the swarm-manager upfront — out of scope; the skill-principles-driven agent-skill pattern is the right surface. |
| Checkpoint-section heading format | Exactly `## Walk Checkpoint (<ISO-timestamp>)` | Deterministic preservation rule in Phase 6; skill already writes in this format. |

## 10. Testing Plan

**Automated tests are the primary verification. Manual tests are supplementary only.**

### 10.1 API tests (Go, testcontainers where applicable)

- `path:scenarios/swarm-manager/api/internal/initiatives/handler_test.go`
  - `TestListInitiatives_ScenarioFilter_ReturnsMatchingInitiatives` — seeds two initiatives, one with items targeting `web-console`, one not; asserts filter returns only the former.
  - `TestListInitiatives_ScenarioFilter_MultipleValues_CSV` — seeds three; `?scenario=a,b`; asserts returns initiatives targeting a OR b.
  - `TestListInitiatives_ScenarioFilter_NoMatches_EmptyList` — asserts graceful empty list, not 404.
  - `TestListInitiatives_ScenarioFilter_InvalidScenarioName` — asserts 400 with structured error (per api-steer §6.1).
- New `handler_scenario_context_test.go`:
  - `TestScenarioContext_ReturnsInitiatives_OrphanItems_Rollup` — seeds mixed data; asserts the three response sections match expected.
  - `TestScenarioContext_NonexistentScenario_Empty` — scenario with zero items/initiatives returns empty-but-well-formed response with rollup all zeros.
  - `TestScenarioContext_OrphanItemsExcludedFromInitiativeCount` — an item in two initiatives counts once in rollup; an orphan item appears only in `orphan_items`.

### 10.2 CLI tests (Go)

- `path:scenarios/swarm-manager/cli/cmd_initiatives_test.go` (extend) — new cases:
  - `cmdInitiativesList` with `--scenario` flag → asserts `?scenario=...` query string is sent (table-driven test over CSV forms).
  - `cmdInitiativesContext` with `--scenario` → asserts call hits `/scenarios/<name>/context`, not `/initiatives/<name>/context`.
  - Mutual exclusion: `--name` and `--scenario` together → asserts error message before any HTTP call.

### 10.3 UI tests (React Testing Library)

- `path:scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.test.tsx` (extend):
  - `renders BacklogScenariosPanel with aggregated target scenarios` — mock API returns initiative context with 3 items targeting 2 distinct scenarios; assert both chips present.
  - `empty target scenarios shows no panel` — mock empty; assert section not rendered (or rendered with empty-state copy).
- `path:scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.test.tsx` (extend):
  - `renders associated initiatives section with rollup` — mock scenario context returning 2 initiatives; assert both cards, status badges, rollup numbers.
  - `renders orphan items section` — mock with 3 orphan items; assert they appear in the orphan section, not under any initiative.
  - `empty state renders helpful guidance` — mock empty response; assert section shows "no coverage yet" copy with a "create initiative" CTA.

### 10.4 Agent-skill validation

- Run `prompt-manager skill read portfolio-manager/TOOLS portfolio-manager/AGENTS` and confirm outputs render.
- Run `prompt-manager agent validate portfolio-manager` (if such a command exists — otherwise skip).
- Dry-run: manually simulate a portfolio-manager heartbeat against a test fixture where web-console has existing coverage; assert the agent's proposal-drafting would call the enumeration commands. *(This is a manual check because the agent runs under agent-manager; a fully automated agent-behavior test is out of scope.)*

### 10.5 Prep-agent checkpoint preservation

- Add a test to `path:scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/` (or a parent test harness if one exists):
  - Pre-populate `last-handoff.md` with a known `## Walk Checkpoint (2026-04-24T10:40-04:00)` section.
  - Invoke the prep regeneration.
  - Assert the new file contains the checkpoint section verbatim (byte-for-byte) and appears after the generated content.
- If no test harness exists for prep-agents, document this as a manual validation step in Section 11 and file a follow-up backlog item: "Add automated test harness for director-swarm agent-file regeneration."

### 10.6 End-to-end smoke

**Do not restart the swarm-manager scenario as part of testing** — operator must restart manually per the no-restart-active-scenario rule. After operator restarts:

```bash
# Verify CLI commands resolve
swarm-manager initiatives list --scenario web-console
swarm-manager initiatives context --scenario web-console
swarm-manager backlog list --scenario web-console

# Verify UI pages load and render new sections
# (operator does this in browser)
```

## 11. Rollout / Validation Checklist

Execute in order. Each `✅` gate must pass before proceeding.

- [ ] **Pre-flight**: Read all files listed in §2. Confirm no other in-flight branch is modifying `cmd_initiatives.go`, `InitiativeDetailsPage.tsx`, or `ScenarioDetailsPage.tsx`. *(If there is, coordinate with the other branch owner before editing.)*
- [ ] **Phase 1 lands**: `portfolio-manager` TOOLS.md + AGENTS.md updated. `prompt-manager skill read` still renders cleanly. Agent validation (if available) passes.
- [ ] **Phase 2 lands**: `go build ./...` and `go test ./scenarios/swarm-manager/...` pass. CLI help shows new flag: `swarm-manager initiatives list --help | grep scenario`.
- [ ] **Phase 3 lands**: Same. `swarm-manager initiatives context --help` shows `--scenario` as an alternative to `--name`. Proto regen script run and generated files committed.
- [ ] **Phase 4 lands**: `cd scenarios/swarm-manager/ui && npm test -- InitiativeDetailsPage` passes. Build: `npm run build` succeeds.
- [ ] **Phase 5 lands**: `cd scenarios/swarm-manager/ui && npm test -- ScenarioDetailsPage` passes. Build: `npm run build` succeeds.
- [ ] **Phase 6 lands**: vision-walk-prep HEARTBEAT updated. Checkpoint-preservation test (automated if possible, manual otherwise) passes against a fixture.
- [ ] **Operator restart**: Operator restarts swarm-manager (`vrooli scenario restart swarm-manager`). API health check passes.
- [ ] **Smoke test**: Operator runs `swarm-manager backlog list --scenario web-console` and sees the 7+ known items. Operator runs `swarm-manager initiatives list --scenario web-console` and sees at least `continuous-audio-platform`. Operator runs `swarm-manager initiatives context --scenario web-console` and sees the combined rollup + orphans.
- [ ] **UI smoke test**: Operator opens Scenario Details for web-console and Initiative Details for continuous-audio-platform; both new sections render.
- [ ] **Agent smoke test**: On next `portfolio-manager` heartbeat, inspect its log / proposal output; confirm it now calls the scenario-coverage commands and references existing coverage in any new proposals. (First real proof will be when it re-proposes `web-console-readiness` as an umbrella.)
- [ ] **Remove checkpoint**: When the operator resumes the divergent vision walk, the skill's Phase 9 removes the `## Walk Checkpoint` section. Confirm the file no longer contains it.
- [ ] **Re-evaluate dec-1776982737575948642**: Either (a) portfolio-manager re-proposes automatically with umbrella framing → operator accepts via `prompt-manager team decision-accept`; or (b) the decision is superseded manually via `decision-accept --selected=<key> --notes="..."`.

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `ScenariosFromGlobs` has a parsing bug that surfaces under new call sites | Low | Medium | Phase 2/3 tests include glob edge cases (`path:scenarios/*/**`, bare `path:scenarios/x`, globs with `!` negation). Fix bugs in place per §5 constraint. |
| Scenario list response grows large (many items/initiatives) | Low now, growing | Medium | Pagination per api-steer §6.2. Default `page_size=50`. Defer only if Phase 3 response exceeds threshold in practice — document decision inline. |
| Portfolio-manager ignores new AGENTS.md step | Medium | High (behavior unchanged) | Phase 1 step 2 must be *imperative* and placed at the top of the propose-initiative workflow, not appended. Add an explicit failure mode in the step: "If you did not run the enumeration, your proposal is invalid." |
| Vision-walk-prep regeneration runs while a divergent walk is active and removes the checkpoint before Phase 6 lands | Medium (24h window) | High (lose walk state) | Checkpoint was written to `last-handoff.md` on 2026-04-24 at 10:40 EDT. Prep runs at 5 AM. Operator should complete Phase 6 (prep-agent change) **before the next 5 AM regeneration** or manually re-inject the checkpoint if it's wiped. Add this to Phase 6 acceptance. |
| UI `BacklogScenariosPanel` has backlog-specific styling that looks wrong on the initiative page | Low | Low | Visual check during Phase 4. If styling issues, generalize the component (rename to `ScenarioChipsPanel`?) — but per `duplicate-before-extracting` memory, copy-first is acceptable; only generalize when a third caller arrives. |
| Proto regen produces unexpected diffs in unrelated generated files | Medium | Low | Run proto regen in a dedicated commit; review diffs; revert any unrelated changes. |
| CLI `--scenario` flag conflicts with auto-start or other global flag parsing | Low | Low | cli-core uses `cliutil.ParseInterspersed` (per cli-steer §8); flag interleaving is already handled. |

## 13. Non-Goals / Prohibited Patterns

**Do not:**
- Add a `target_scenarios` field to the initiative data model as *stored* state. Keep it derived.
- Rename or remove the existing `backlog list --scenario` flag.
- Restart the swarm-manager scenario from this agent — operator restarts manually.
- Create a new top-level `scenario` CLI command group in this plan (e.g., `swarm-manager scenario context ...`) — out of scope; route via `initiatives context --scenario` for consistency with existing surface.
- Commit proto regen alongside logic changes in the same commit — keep them separate for reviewability.
- Introduce feature flags, A/B gates, or migration windows for any of the new behavior. This is greenfield per §5.
- Reuse `acceptance_allow` globs for anything other than their current purpose (path filter during execution). The derivation to scenario names is read-only.
- Hard-enforce agent behavior from the swarm-manager side (e.g., "reject `initiative create` calls from portfolio-manager if it didn't first call `path:scenarios/<x>/context`"). Agent compliance is a soft/doc contract.
- Update `portfolio-manager` to *also* reason about orphan items that aren't scenario-specific (e.g., cross-scenario audit items). Its proposal workflow change is scoped to scenario-named initiatives only; anything broader is out of scope.

## 14. Definition of Done

**Hard gates (all must be true):**

1. All six phases have landed and their automated tests pass on master.
2. Greenfield constraint honored — no compat shims, no dual paths, no feature flags introduced by this plan.
3. `swarm-manager backlog list --scenario web-console` returns ≥7 items.
4. `swarm-manager initiatives list --scenario web-console` returns ≥1 initiative (at minimum `continuous-audio-platform`).
5. `swarm-manager initiatives context --scenario web-console` returns a single-call response with `initiatives`, `orphan_items`, and `rollup` populated.
6. Scenario details UI page for `web-console` renders: Initiatives section, Orphan items section, Rollup.
7. Initiative details UI page for `continuous-audio-platform` renders: Targeted scenarios section with `web-console` chip.
8. `portfolio-manager`'s AGENTS.md step 4 says "Detect existing coverage" as an imperative pre-check with the three enumeration commands.
9. `vision-walk-prep` regeneration preserves `## Walk Checkpoint` sections across runs (verified via test fixture or manual validation against a known checkpoint).
10. Operator has restarted swarm-manager and smoke tests (§11) pass.

**Soft gates (should be true; investigate if not):**

- `portfolio-manager`'s next heartbeat output references `swarm-manager backlog list --scenario ...` invocation in its reasoning trail (evidence that the workflow change propagated).
- `dec-1776982737575948642` is either re-proposed as an umbrella or manually superseded within 7 days of plan completion.

---

## Appendix A: Operator Handoff Note

This plan was authored during a morning-vision-walk divergence on 2026-04-24 (walk date 2026-04-24, plan authored ~11:00 EDT). The divergence was captured in `path:scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md` under `## Walk Checkpoint (2026-04-24T10:40-04:00)`. When the operator resumes the walk:

1. Re-read the checkpoint.
2. Skip Phases 1–3 (covered).
3. Pick up at Phase 4 (Strategist Decisions — likely a quick "not yet active" pass).
4. Proceed through Phases 5, 5.3, 5.5, 6, 7, 8, 9.
5. At Phase 8, **do not re-file this plan's items as backlog entries** — the plan itself is the artifact. Do confirm the plan path and any merged PRs in the Actions summary.
6. At Phase 9, remove the `## Walk Checkpoint` section from `last-handoff.md`.

If this plan is picked up by an implementation agent with no chat context: start at §2 (required reading), then execute phases in §8. All file paths are absolute-from-repo-root.
