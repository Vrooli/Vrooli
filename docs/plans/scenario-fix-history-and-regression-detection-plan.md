# Scenario Fix History & Regression Detection — Implementation Plan

## 1. Purpose

Make it cheap and reliable to answer two questions when a bug is reported against
a scenario:

1. **"Has this exact scenario had a related fix before?"** — answered from
   structured per-scenario fix history surfaced in API, CLI, and UI.
2. **"Has any scenario solved a similar-looking bug before?"** — answered via
   AI semantic search over archived + active fixes, callable from the
   `scientific-debugging` skill.

The plan also bakes those two questions into the scientific-debugging skill as a
mandatory pre-hypothesis pass so future debugging sessions can't skip it.

A separate already-shipped change (mobile `ScenarioCoverageSection` lift in
`scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.tsx`) is the
prerequisite that made the existing coverage section visible on mobile; this
plan does not redo it but treats it as the baseline.

---

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read scientific-debugging
```

Also re-read in-repo before starting any phase:

- `scenarios/swarm-manager/api/internal/scenarios/context.go`
- `scenarios/swarm-manager/api/internal/aisearch/{models.go,index.go,handlers.go}`
- `scenarios/swarm-manager/api/internal/backlog/types.go` (esp. `KindFix`,
  `AcceptanceAllow`, `ArchivedAt`)
- `scenarios/swarm-manager/cli/cmd_scenarios.go` (esp. `cmdScenariosGet`)
- `scenarios/swarm-manager/cli/domains/{scenarios,backlog,aisearch}/register.go`
- `scenarios/swarm-manager/ui/src/components/scenarios/ScenarioCoverageSection.tsx`
- `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md`

---

## 3. Problem Statement

Current state, verified against the code:

- `GET /scenarios/{name}/context` returns initiatives + orphan items + a rollup,
  but **does not separate fixes from other kinds**, does not surface per-fix
  metadata useful for triage (final review verdict, completion date,
  conclusion path), and does not make active vs archived addressable as a
  filter — only as a derived counter.
- The `swarm-manager scenarios get` CLI command renders metadata only. There is
  **no CLI surface for fix history** at all.
- The UI `ScenarioCoverageSection` lists initiatives and orphan items, but
  fixes are not grouped, not toggleable between active/archived, and not
  searchable.
- `aisearch.SearchFilters` already supports `Kind` and `IncludeArchived`, but
  `buildBacklogPayload` (`scenarios/swarm-manager/api/internal/aisearch/index.go:152`)
  **does not index `AcceptanceAllow`**, so an AI search cannot be constrained to
  "fixes that affected scenario X". Cross-scenario semantic recall is possible;
  per-scenario semantic recall is not.
- The `scientific-debugging` skill has no Phase 0 prior-art step. Recurrences
  are detected only by accident.

Net effect: when the user reviews a fix and archives it, the institutional
memory is buried in the filesystem. Recurrences become "new" bugs, and prior
investigation transcripts are not pulled in to inform the new one.

---

## 4. Scope

### In scope

- Extend the scenario context API with a structured `fixes` block (active +
  archived).
- Add a CLI surface for fix history: extend `scenarios get` and add
  `scenarios fixes` with active/archived/all and free-text filtering.
- Add a Fix History UI section under coverage with active/archived/all toggle
  and a client-side title+description filter.
- Extend `aisearch` payload + filter to support a `targetScenario` constraint,
  so cross-scenario semantic search can be narrowed to one scenario.
- Update `scientific-debugging` SKILL.md with a mandatory Phase 0 prior-art
  pass that calls the new CLI surfaces.

### Out of scope

- Reopening archived items automatically.
- New backlog kinds; `KindFix` already exists.
- A standalone "fixes" page outside scenario detail.
- LLM summarization of fix conclusions; the plan exposes raw conclusion paths
  and final review verdicts only.
- Changes to initiative-level fix views; the scenario detail page is the
  surface.
- Backfilling missing `acceptance_allow` on historical archived fixes —
  documented as a known gap, not addressed here.

---

## 5. Greenfield Constraint (Hard Rule)

Per project memory (`feedback_planning_guidelines.md`,
`feedback_no_git_mutations.md`):

- No backwards-compat shims, no deprecated-but-kept fields, no parallel
  endpoints. New shape replaces old shape in the same response.
- No legacy aliases on the CLI; new flags ship, removed flags are deleted.
- All affected tests are updated in the same change; do not leave a
  "compat layer" to be cleaned up later.
- Every phase must end with the scenario restarted (`vrooli scenario restart
  swarm-manager`) before claiming done — per
  `feedback_use_vrooli_scenario_restart.md`.

This constraint is repeated in §13 Definition of Done.

---

## 6. Current Technical Context

| Area | File | Today |
|---|---|---|
| API context handler | `scenarios/swarm-manager/api/internal/scenarios/context.go` | Returns `Initiatives`, `OrphanItems`, `Rollup`. Iterates initiatives and orphan items but does not group by kind. |
| Backlog model | `scenarios/swarm-manager/api/internal/backlog/types.go:81` | `KindFix = "fix"` already exists. `BacklogItem` has `AcceptanceAllow []string` and `ArchivedAt *string`. |
| AI search filters | `scenarios/swarm-manager/api/internal/aisearch/models.go:28` | `Kind`, `Initiative`, `IncludeArchived`, `Status`. **No `TargetScenario`.** |
| AI search payload | `scenarios/swarm-manager/api/internal/aisearch/index.go:152` | `kind, name, title, status, priority, tags, initiative, effort, archived`. **No `target_scenarios`.** |
| Scenario CLI | `scenarios/swarm-manager/cli/cmd_scenarios.go:94` (`cmdScenariosGet`) | Prints metadata only. |
| Scenario CLI registry | `scenarios/swarm-manager/cli/domains/scenarios/register.go` | No `fixes` subcommand. |
| Coverage UI | `scenarios/swarm-manager/ui/src/components/scenarios/ScenarioCoverageSection.tsx` | Renders rollup + initiatives + orphan items. No kind grouping, no archive toggle, no search. |
| Scenario detail page | `scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.tsx` | Coverage section now lifted out of `lg:block` and shared by mobile + desktop (already shipped). |
| Scientific debugging skill | `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md` | Phases: Observe, Hypothesize, Test, Analyze, Fix, Verify. No prior-art phase. |

Assumption marked as such: the final review verdict for a completed fix is
already persisted somewhere reachable from a `BacklogItem` (e.g., review log
file under the item's directory). **Verify in Phase 1** by running:

```bash
ls scenarios/swarm-manager/fix/<some-completed-fix>/
```

If no structured verdict exists, the plan exposes "last review classification
or empty" as a tolerated null and the deeper review-storage redesign is filed
as a separate item — it is not absorbed into this plan.

---

## 7. Target End State

Concretely, when this plan is done:

- `GET /scenarios/{name}/context` includes a `fixes` object with `active` and
  `archived` arrays, each entry carrying `name, title, status, priority,
  initiative, archivedAt, completedAt, lastReviewClassification,
  conclusionPath`.
- `swarm-manager scenarios get --name X` prints a "Fix history" section showing
  active count + most recent 5 archived; `--json` includes the new block.
- `swarm-manager scenarios fixes --name X [--archived|--active|--all] [--search Q] [--limit N]`
  exists, defaults to `--all`, prints a tabular human view, supports `--json`.
- `aisearch` payload includes `target_scenarios` (string array). `SearchFilters`
  has `TargetScenario string`. A reindex is required and is wired into the
  existing reindex path; the plan documents the operator command.
- `ScenarioCoverageSection` renders a "Fix history" subsection with an
  Active/Archived/All segmented control and a search input.
- `scientific-debugging/SKILL.md` has a "Phase 0: Prior-art check" with two
  required passes (scenario-local CLI search + cross-scenario AI search) and
  three required outputs (no prior art / related prior art / likely
  recurrence).

---

## 8. Implementation Strategy (Phased)

Phases are independently mergeable. Order matters because later phases consume
earlier ones.

### Phase 1 — API: structured fix history in scenario context

Files: `scenarios/swarm-manager/api/internal/scenarios/{context.go,context_test.go,types.go}`.

Steps:

1. Verify the review-verdict storage assumption (see §6). Pick one of
   `lastReviewClassification` (already on `Scenario` for the scenario itself)
   or a derived value from the per-item review log; document the choice in
   the handler.
2. Add `ScenarioFix` struct: `Kind, Name, Title, Status, Priority, Initiative,
   ArchivedAt, CompletedAt, LastReviewClassification, ConclusionPath`.
3. Add `Fixes struct { Active []ScenarioFix; Archived []ScenarioFix }` to
   `ScenarioContext`.
4. In `GetContext`, single-pass over the same `backlogLister.LoadAll(nil)` set
   (already loaded for orphan computation) and emit `ScenarioFix` for every
   `KindFix` item targeting the scenario, regardless of initiative
   membership. Include archived ones — `LoadAll` already returns them.
5. Sort `Active` by priority desc then updated desc; sort `Archived` by
   `archivedAt` desc.
6. Update `context_test.go` table: a fixture with one active fix in an
   initiative, one archived orphan fix, one active orphan fix; assert all
   three appear in `Fixes` and counts match.

Done when: `go test ./scenarios/swarm-manager/api/internal/scenarios/...`
passes and the JSON shape matches §9.

### Phase 2 — CLI: surface fix history

Files: `scenarios/swarm-manager/cli/cmd_scenarios.go`,
`scenarios/swarm-manager/cli/cmd_scenarios_test.go` (new if needed),
`scenarios/swarm-manager/cli/domains/scenarios/register.go`,
`scenarios/swarm-manager/cli/internal/support/dependencies.go`,
`scenarios/swarm-manager/cli/app.go`.

Steps:

1. Extend `ScenarioResponse` (or add a sibling `ScenarioContextResponse`) so
   `cmdScenariosGet` can reuse a single fetch path. Prefer one extra round
   trip to `/scenarios/{name}/context` rather than overloading
   `/scenarios/{name}` — keeps responsibilities clean.
2. In `cmdScenariosGet`, after the existing details print, fetch context and
   render a "Fix History" section: active count + bullet list of up to 5 most
   recent archived (`name`, status, archivedAt). Per
   `feedback_cli_default_human_output.md`, default human output; `--json`
   merges the context payload into the existing JSON.
3. Add `cmdScenariosFixes(args []string)`:
   - Flags: `--name NAME` (required), `--active`, `--archived`, `--all`
     (default `--all`, mutually exclusive trio), `--search Q` (substring
     match on title and name, case-insensitive), `--limit N` (default 50),
     `--json`.
   - Calls `/scenarios/{name}/context`, slices the `fixes` block per flags,
     applies search client-side.
   - Prints a tabular human view (`Status | Priority | Title | Archived`),
     followed by a `cliCommand` "Next steps" block pointing to `aisearch`
     for cross-scenario recurrence.
4. Register `fixes` in `domains/scenarios/register.go`.
5. Add a test in `cmd_scenarios_test.go` verifying flag parsing,
   active/archived split, and `--search` filtering against a fake API.

Done when: `cd scenarios/swarm-manager/cli && go build ./... && go test ./...`
passes.

### Phase 3 — UI: Fix History section

Files: `scenarios/swarm-manager/ui/src/components/scenarios/ScenarioCoverageSection.tsx`
(extend) or new `ScenarioFixHistorySection.tsx` consumed by it,
`scenarios/swarm-manager/ui/src/services/scenarios-service.ts` (type update),
`scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.test.tsx`.

Steps:

1. Update `ScenarioContext` TS type to include the new `fixes` block.
2. Add `ScenarioFixHistorySection` rendering a `DetailSection` titled "Fix
   History" with:
   - Segmented control: `Active | Archived | All` (default `Active`).
   - Search input filtering by title or name.
   - Item rows linking to the backlog item via existing `EntityLink`.
   - Empty state copy for each filter.
3. Mount inside `ScenarioCoverageSection` below the existing orphan list, so
   one network call covers both. Single test-id `scenario-fix-history` to
   keep `getByTestId` unique across mobile+desktop layouts (mirrors the lift
   pattern just applied to coverage).
4. Add Vitest cases under "associated initiatives & backlog coverage" group:
   active-only filter, archived-only filter, search narrows results.

Done when: `pnpm vitest run` in `scenarios/swarm-manager/ui` is green and the
new section is visible on both mobile and desktop layouts (verified by
restarting the scenario and loading the page).

### Phase 4 — AI search: per-scenario filter

Files: `scenarios/swarm-manager/api/internal/aisearch/{models.go,index.go,service.go,handlers.go,*_test.go}`.

Steps:

1. Add `TargetScenarios []string` to `BacklogPayload`. Update
   `buildBacklogPayload` to populate from `item.AcceptanceAllow` (only entries
   whose tokens look like scenario names — keep as-is for now and document the
   convention; over-filtering can come later).
2. Add `TargetScenario string` to `SearchFilters` (single value: callers want
   "fixes that touched X").
3. In the Qdrant query path, translate `TargetScenario` to a payload-array
   contains-filter.
4. Add `IncludeArchived` default behaviour: keep current default (`false`),
   but when `Kind == ["fix"]`, treat as `true` unless caller explicitly sets
   it. Encodes the "fix history wants archived by default" rule in one place.
5. Add unit tests in `service_test.go` for the new filter + the
   archived-default-for-fixes rule.
6. Document the operator reindex: `swarm-manager aisearch reindex` (existing
   command) — the new payload field requires a full reindex to populate.

Done when: API tests green; running reindex against a dev environment fills
the new payload field.

### Phase 5 — Skill: `scientific-debugging` Phase 0

File: `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md`.

Steps:

1. Insert a new section between "When to Use" and "The Process": **Phase 0:
   Prior-art check.**
2. Phase 0 mandatory actions:
   - Determine the affected scenario name.
   - Run scenario-local search:
     ```bash
     swarm-manager scenarios fixes --name <scenario> --all --search "<symptom keywords>"
     ```
   - Run cross-scenario semantic search:
     ```bash
     swarm-manager aisearch search --query "<one-sentence symptom>" --kind fix --include-archived
     ```
     (Also pass `--target-scenario <scenario>` once Phase 4 ships, for a
     focused first pass.)
3. Phase 0 required output: one of
   - **No prior art** — proceed to Hypothesize.
   - **Related prior art** — link the prior fix(es) and summarize how each was
     resolved before forming new hypotheses.
   - **Likely recurrence** — link the prior fix item, recommend reopening or
     spawning a follow-up fix linked via `spawned_from` rather than starting
     fresh investigation.
4. Update the diagram and the "When to Use" entry rules to reference Phase 0.
5. Add a brief reminder that the CLI is the contract, never raw HTTP, per
   `feedback_skills_use_cli_never_api.md`.

Done when: `prompt-manager skill read scientific-debugging` returns the new
phase verbatim.

---

## 9. Contract Decisions

### API response shape (additive within `ScenarioContext`)

```jsonc
{
  "scenarioName": "web-console",
  "rollup": { "...": "unchanged" },
  "initiatives": [ /* unchanged */ ],
  "orphanItems": [ /* unchanged */ ],
  "fixes": {
    "active":   [ScenarioFix, ...],
    "archived": [ScenarioFix, ...]
  }
}

// ScenarioFix
{
  "kind": "fix",
  "name": "fix-foo-bar",
  "title": "Foo crashes when bar is empty",
  "status": "completed",
  "priority": 2,
  "initiative": "",                       // empty if orphan
  "archivedAt": "2026-04-01T12:00:00Z",   // null if active
  "completedAt": "2026-03-30T10:00:00Z",  // null if not completed
  "lastReviewClassification": "approved", // "" if unreviewed
  "conclusionPath": "fix/fix-foo-bar/conclusion.md" // "" if absent
}
```

Greenfield: this is the only shape. No `legacy_fixes` parallel field.

### CLI shape

- `swarm-manager scenarios get --name X` — adds a "Fix History" section to
  human output; merges `fixes` into JSON output.
- `swarm-manager scenarios fixes --name X [--active|--archived|--all]
  [--search Q] [--limit N] [--json]` — defaults: `--all`, `--limit 50`, no
  search.
- Exit code: 0 even when no fixes; non-empty stderr only on real errors. Empty
  fix history is a normal state.

### AI search

- `BacklogPayload` gains `target_scenarios []string`. JSON key:
  `target_scenarios`.
- `SearchFilters` gains `target_scenario string`. JSON key: `target_scenario`.
- Default behaviour change: when `kind == ["fix"]` and `include_archived` is
  unset, treat as `true`.

### Skill

- Phase 0 is mandatory before Hypothesize.
- All commands invoked via the scenario CLI; no raw HTTP.

---

## 10. Testing Plan

Per project preference (`feedback_testing_over_manual.md`), validation is
automated, not a manual checklist.

| Phase | Test command | Asserts |
|---|---|---|
| 1 | `cd scenarios/swarm-manager/api && go test ./internal/scenarios/... -timeout 120s` | `fixes.active` and `fixes.archived` populated correctly across initiative-member, orphan, and archived inputs |
| 2 | `cd scenarios/swarm-manager/cli && go test ./... -timeout 120s` | `cmdScenariosFixes` flag parsing, `--active`/`--archived`/`--all` partitioning, `--search` substring filter, `--json` shape |
| 3 | `cd scenarios/swarm-manager/ui && pnpm vitest run` | Active default render, Archived toggle render, search narrows results, single test-id, no double-render across mobile/desktop |
| 4 | `cd scenarios/swarm-manager/api && go test ./internal/aisearch/... -timeout 120s` | New payload field populated; `target_scenario` filter narrows results; archived-by-default-for-fixes rule |
| 5 | `prompt-manager skill read scientific-debugging \| grep -c "Phase 0"` ≥ 1 | Skill update is published |

Cross-cutting smoke after Phase 4 ships:

```bash
swarm-manager aisearch reindex
swarm-manager aisearch search --query "smoke" --kind fix --include-archived --target-scenario web-console
```

---

## 11. Rollout / Validation Checklist

1. Phase 1 merged → API tests green → restart: `vrooli scenario restart swarm-manager`.
2. Phase 2 merged → CLI tests green → re-install CLI: `cd scenarios/swarm-manager/cli && ./install.sh`.
3. Phase 3 merged → UI tests green → restart and load `/scenarios/<name>` on
   both mobile width and desktop width; confirm "Fix History" section renders
   exactly once per layout.
4. Phase 4 merged → API tests green → restart → run `aisearch reindex`,
   confirm `aisearch status` reports indexed counts > 0, then run a sample
   `--target-scenario` query and confirm results constrain.
5. Phase 5 merged → `prompt-manager skill read scientific-debugging` returns
   Phase 0 content.

After every phase: scenario restart is mandatory before claiming done
(per `feedback_use_vrooli_scenario_restart.md`).

---

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Review verdict not stored in a structured place | Med | Phase 1 produces "" for `lastReviewClassification` for older items | Document the gap; tolerate empty; do not block plan on a separate review-storage redesign |
| `acceptance_allow` historically inconsistent (free text vs scenario names) | Med | Some fixes won't be discoverable by `target_scenario` until rewritten | Document; do not backfill in this plan; only filter on exact match |
| Reindex required after Phase 4 | High | One-time operator action | Document in §11; surface in `aisearch status` if payload is missing field |
| UI test-id duplication regressions across mobile/desktop | Med | Vitest `getByTestId` failures | Single shared section above the `lg:block` wrapper, mirroring the coverage-lift pattern |
| Skill update lands before CLI ships | Low | Phase 0 commands fail | Sequence Phase 5 strictly after Phase 2, and Phase 4's `--target-scenario` after Phase 4 |

---

## 13. Non-goals / Prohibited Patterns

- No legacy/parallel `fixes_v1` fields in API responses (greenfield).
- No raw HTTP in the skill — CLI only (`feedback_skills_use_cli_never_api.md`).
- No git mutations as part of validation (`feedback_no_git_mutations.md`).
- No automatic reopening of archived fixes; recurrence detection surfaces
  evidence, the human/agent decides.
- No new backlog kind; reuse `KindFix`.
- No backfill scripts for historical `acceptance_allow` data.
- No mass-update scripts to retrofit existing fix items
  (CLAUDE.md "Common Pitfalls").

---

## 14. Definition of Done

All of the following are true:

- [ ] `GET /scenarios/{name}/context` returns the new `fixes` block with the
      shape in §9, verified by Go tests.
- [ ] `swarm-manager scenarios get --name X` and `swarm-manager scenarios fixes
      --name X` both work end-to-end against a live scenario, with
      `--active`, `--archived`, `--all`, `--search`, `--limit`, `--json`.
- [ ] `ScenarioCoverageSection` renders a Fix History subsection with
      Active/Archived/All toggle and search, on both mobile and desktop, with
      a single test-id (no duplication).
- [ ] `aisearch` payload includes `target_scenarios`; `SearchFilters` accepts
      `target_scenario`; reindex documented and verified populating the
      field; archived-by-default-for-fix rule covered by a unit test.
- [ ] `scientific-debugging` SKILL.md has a Phase 0 prior-art section that
      mandates both the scenario-local CLI search and the cross-scenario
      semantic search, with the three required outputs.
- [ ] Greenfield: no parallel `legacy_*` fields, no compat shims, no removed
      flags retained as aliases.
- [ ] After every phase merge: `vrooli scenario restart swarm-manager`
      run and the relevant page/CLI re-validated.
- [ ] All affected tests updated in the same PR as the change; no "TODO:
      fix tests" left behind.
