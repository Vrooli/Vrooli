# Agent-System Migration Implementation Plan

## 1. Purpose

Consolidate the prompt-manager self-improvement framework into a single coherent plan-of-record at `docs/agent-system/`, strip duplicated canon from skills, and add a structural data layer (`topics.json` + `prompt-manager graph topics` CLI + UI dual-mode graph) so agent message-flow is declared, validated, and visualized rather than implied through prose.

This is the system eating its own dog food: doctrine that today lives scattered across 7+ skills (and partially mis-classified as "notebook" in `docs/meta-optimization/`) becomes plan-of-record once, with skills reduced to procedure-only artifacts that cite it.

---

## 2. Required Reading

Plan-authoring required reading (per `implementation-plan-authoring`):

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Domain-relevant skills for execution (discovered via `plan-skill-discovery`):

```bash
prompt-manager skill read skill-principles team-shared-docs-design team-member-capability-architecture-audit capability-extraction skill-authoring documentation-health
```

Files that are the **source material** for canon extraction (read these to understand what's moving):

- `scenarios/prompt-manager/store/skills/packs/core/skill-principles/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/team-member-capability-architecture-audit/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/capability-extraction/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/skill-authoring/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/skill-authoring-{meta,platform,practice,search,tools}/SKILL.md`
- `docs/meta-optimization/README.md`
- `docs/meta-optimization/CONVERSION_PLAYBOOK.md`
- `docs/meta-optimization/DEPRECATION_POLICY.md`
- `docs/meta-optimization/REFERENCE_SCENARIOS.md`

Worked examples of the inbox/router pattern that informed the topics data model:

- `scenarios/prompt-manager/store/skills/packs/core/marketing-research-router/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/monetization-opportunity-router/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/market-validation-router/SKILL.md`
- `docs/marketing/research/README.md`

---

## 3. Hard Constraints

### Greenfield

This is greenfield work. Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. Skills that fully migrate to PoR are deleted, not left as stubs. Doc files that move are removed from their old location, not left as redirects.

### Paired moves, not sequential phases

For every piece of canon migrated to `docs/agent-system/`, the source skill is updated **in the same commit/atomic step** to remove the relocated content and add a `Required reading: docs/agent-system/<file>` line. At no point does the same canon live in two places. This rule is enforced per piece — not per file, not per phase.

### Skills cite PoR, never restate

Any skill whose content includes definitions of "Plan of Record", "Skill", "Action", "CLI", "Notebook", the layer mantra, the promotion ladder, or the 9-layer audit model must drop those definitions and cite `docs/agent-system/`. This becomes a lint rule for `team-agent-optimizer` audits going forward.

---

## 4. Problem Statement

The prompt-manager framework — definitions of skill / agent / team / Plan of Record / Action / CLI / notebook, the promotion ladder, the 9-layer team-member architecture model, the intake/collection/analysis/promotion pipeline, skill authoring rules, and the deprecation policy — is currently scattered across 7+ skills with verbatim duplication, three competing framings of the same promotion ladder, and a `docs/meta-optimization/` folder that declares itself a notebook but actually contains canon.

Concrete duplication evidence:
- Layer mantra ("Truth lives in Plan of Record. Judgment lives in Skills. Execution lives in Actions...") appears verbatim in `skill-principles/SKILL.md` and `team-member-capability-architecture-audit/SKILL.md`, and rephrased in `team-shared-docs-design/SKILL.md` and `docs/meta-optimization/README.md`.
- Three names for one ladder: "Promotion-Retirement Lifecycle" (skill-principles), "Layer Model" (audit skill), "three-tier mental model" (meta-opt README).

Separately, agent-to-agent message flow within and across teams is declared only in prose (router skills name their input/output prefixes, but no machine-readable contract exists). This means: orphan output prefixes, dangling intake claims, conflicting drain duty, and stalled inboxes are not detectable structurally. The prompt-manager UI's team graph view, which renders edges from `managerId`, degenerates to disconnected nodes for leaderless teams (the current preferred pattern).

---

## 5. Scope

### In scope

- Author `docs/agent-system/` plan-of-record (8–10 canonical files + `drafts/` subfolder).
- Strip relocated canon from existing skills; mark fully-absorbed skills deprecated and delete them.
- Migrate `docs/meta-optimization/` content into `docs/agent-system/`; reduce or delete the meta-optim folder.
- Update every meta-optimization team member's `TOOLS.md` / `HEARTBEAT.md` to read from the new locations.
- Define and implement `topics.json` schema at `store/teams/<team>/members/<member>/topics.json`.
- Backfill `topics.json` for every member of every active team (~30+ files).
- Add API endpoints in prompt-manager API for reading/listing/updating topics.
- Add `prompt-manager graph topics` (and `graph drain-status`) CLI verbs with validation rules.
- Add UI dual-mode team graph: `Hierarchy | Topics | Both` toggle, topics-mode edges, validation overlay, side-panel queue inspection.
- Rewire `team-member-capability-architecture-audit` skill to consume topics.json structurally for the layer-scoring step.
- End-to-end cleanup + scenario restart + health verification.

### Out of scope (v1)

- System-wide cross-team graph page (per-team page only; cluster view is a follow-up).
- Drag-drop topic editing in UI (read-only graph + JSON editor side panel for v1; visual editing later).
- Drain-metrics historical graphing and trendlines.
- Auto-promotion / auto-routing of inbox entries.
- Centralized topic registry / namespace governance.
- Renaming / restructuring `docs/meta-optimization/` consumers in scenarios outside prompt-manager (only update the consumers we own).
- Adoption of the inbox/router pattern by teams that haven't already done it (marketing-crew and monetization are done; this plan does not expand the pattern).

---

## 6. Current Technical Context

### Doctrine sources (read-only; will be edited per paired-move rule)

- `scenarios/prompt-manager/store/skills/packs/core/skill-principles/` (181 lines)
- `scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/` (105 lines)
- `scenarios/prompt-manager/store/skills/packs/core/team-member-capability-architecture-audit/` (243 lines)
- `scenarios/prompt-manager/store/skills/packs/core/capability-extraction/` (305 lines)
- `scenarios/prompt-manager/store/skills/packs/core/skill-authoring{,-meta,-platform,-practice,-search,-tools}/` (six files, 79–392 lines each)
- `docs/meta-optimization/README.md`, `CONVERSION_PLAYBOOK.md`, `DEPRECATION_POLICY.md`, `REFERENCE_SCENARIOS.md`

### Topics data layer

- New file location: `scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json`
- Sibling to existing per-member files (`HEARTBEAT.md`, `RESPONSIBILITIES.md`, `last-handoff.md`)
- Loader/parser will live in prompt-manager API alongside existing graph queries (`graph health`, `graph node`, `graph skillless-agents`, `graph empty-teams`)

### Existing API/CLI surfaces to extend

- `scenarios/prompt-manager/api/` — Go service exposing graph + team + skill endpoints
- `scenarios/prompt-manager/cli/` — `prompt-manager` CLI (Cobra-style)
- `scenarios/prompt-manager/ui/src/components/editor/OrgChartPanel.tsx` (558 lines) — React Flow + dagre layout for hierarchy view
- `scenarios/prompt-manager/ui/src/types/orgChart.ts` — current types (`managerId`, `OrgEdge`)

### Teams to backfill

Active teams (from `store/teams/`):
- `director-swarm` (incl. `vision-walk-prep`)
- `marketing-crew` (researcher already drains `research-inbox/*`; backfill formalizes other members)
- `monetization` (opportunity-router and market-validator already documented)
- `meta-optimization`
- `infra-health`

Approximate member count: 30+. Schema must be stable before bulk backfill.

---

## 7. Target End State

After this plan completes:

1. `docs/agent-system/` exists as the single home for framework canon, with 8–10 short PoR files plus `drafts/`.
2. `docs/meta-optimization/` either does not exist, or contains only a stub README pointing at `docs/agent-system/` (decision in Phase 1).
3. No skill in `scenarios/prompt-manager/store/skills/packs/core/` restates layered-architecture canon. Every skill that previously did either (a) is deleted, or (b) cites `docs/agent-system/<file>` as required reading.
4. Every active team member has a `topics.json` declaring intake prefixes, output prefixes, decisions owned, and capability-gap raise capability.
5. `prompt-manager graph topics` returns a directed graph and validation report (orphan output, orphan input, conflicting drain, dangling sink, stalled drain, piling inbox). Exit code is non-zero when any team has unresolved smells.
6. The UI team graph page renders both Hierarchy and Topics modes; mode auto-defaults based on whether any `managerId` edges exist; Topics mode shows validation overlays.
7. `team-member-capability-architecture-audit` consumes `topics.json` for the structural layer-scoring step (Intake / Collection / Promotion / Routing scores derive from the file, not from prose grep).
8. prompt-manager scenario passes type-check, lint, unit tests, and restarts cleanly.

---

## 8. Implementation Strategy

Six phases. Phases 1 and 2 run **concurrently** (different surfaces, no shared files). Phase 5 is the synchronization gate where doctrine + data must both be in place. Phases 3 and 4 run sequentially after Phase 2 ships.

### Phase 0 — Schema + structure agreement (small, blocking)

Goal: settle the two contracts that everything else depends on, without writing content.

**Deliverables:**
- `docs/agent-system/README.md` skeleton — index page listing intended files, no content yet. Establishes the folder.
- `docs/agent-system/_outline.md` — table mapping each PoR file to its source skill(s) and which sections move where. Used as the migration manifest for Phase 1.
- `scenarios/prompt-manager/api/internal/topics/schema.go` (or equivalent) — the `topics.json` schema as a Go struct + JSON Schema validator. No loader yet, no endpoints yet.
- `docs/agent-system/TOPICS_SCHEMA.md` — human-readable schema doc used as PoR for the data model. (Originally drafted at `docs/agent-system/drafts/topics-schema.md` in Phase 0; promoted to canon during the inbox-flow refactor.)

**`topics.json` schema (canonical for the rest of the plan):**

```json
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "drained_by_skill": "marketing-research-router",
      "source_team": null
    }
  ],
  "output": [
    { "prefix": "audience-scan/*", "destination_kind": "knowledge", "destination_team": null },
    { "prefix": "monetization-benchmark-adjacent-record/*", "destination_kind": "knowledge", "destination_team": "monetization" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["vision-walk", "operator"]
}
```

`destination_kind` ∈ `{knowledge, decision, por_file, capability_gap, skill_proposal, backlog}`. When `por_file`, an additional `destination_path` field names the target file under `docs/`.

**Exit criteria:**
- [ ] `docs/agent-system/README.md` exists with file list
- [ ] `_outline.md` migration manifest is reviewed and accepted by an operator (this is a `meta-optimization` decision)
- [ ] `topics.json` schema validates the marketing-crew researcher's expected declaration without modification

### Phase 1 — Paired canon migration (concurrent with Phase 2)

For each row in `_outline.md`, do an atomic move: extract content from source skill → place in PoR file → strip the skill of the relocated section and add `Required reading: docs/agent-system/<file>` to its top.

**Migration manifest (final shape; subject to operator acceptance in Phase 0):**

| PoR file | Sources |
|---|---|
| `docs/agent-system/PRIMITIVES.md` | `skill-principles` §1; `team-shared-docs-design` definitions; `team-member-capability-architecture-audit` §2 layer table |
| `docs/agent-system/LAYERS.md` | The layer mantra (canonical home, all duplicates removed); `skill-principles` §6 layering rule; audit skill §2 |
| `docs/agent-system/PROMOTION_LADDER.md` | `skill-principles` §6 lifecycle; `docs/meta-optimization/CONVERSION_PLAYBOOK.md` |
| `docs/agent-system/TEAM_DOCS_PATTERNS.md` | `team-shared-docs-design` wholesale (skill is then deprecated/deleted) |
| `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md` | `team-member-capability-architecture-audit` §2 layer model + §4 pipeline (skill keeps Phase 1–4 procedure + output contract) |
| `docs/agent-system/SKILL_AUTHORING.md` | `skill-principles` universal quality bars; `skill-authoring` (and its category siblings) commonalities |
| `docs/agent-system/INTAKE_PIPELINE.md` | Intake → Collection → Analysis → Promotion model from audit skill; inbox-router-drain pattern from marketing-crew worked example |
| `docs/agent-system/DEPRECATION_POLICY.md` | `docs/meta-optimization/DEPRECATION_POLICY.md` (relocated) |
| `docs/agent-system/REFERENCE_SCENARIOS.md` | `docs/meta-optimization/REFERENCE_SCENARIOS.md` (relocated) |

**Skill outcomes after migration:**

| Skill | Outcome |
|---|---|
| `skill-principles` | Deleted. Fully absorbed into PoR. |
| `team-shared-docs-design` | Deleted. Fully absorbed. |
| `team-member-capability-architecture-audit` | Kept. §2 + §4 deleted; Phase 1–4 procedure + output contract retained; required-reading line added. |
| `capability-extraction` | Kept. Layer/ladder references replaced with PoR citation. |
| `skill-authoring` | Kept. Common-quality content removed; per-category specifics retained. |
| `skill-authoring-{meta,platform,practice,search,tools}` | Kept. Common-quality content removed; category-specific content retained. |

**`docs/meta-optimization/` decision:** delete the folder entirely. Its three content files migrate; the README's "notebook" framing was a fiction that this plan retires. If a small amount of genuinely team-specific running notebook material survives, it lives as team knowledge entries under topic prefix `meta-optimization/notebook/<slug>` (per the inbox-router-drain pattern) — not as markdown files.

**Per-piece atomicity check:** before each commit in this phase, grep the entire skills tree for the canonical sentence being relocated and confirm zero residual occurrences after the commit.

**Exit criteria:**
- [ ] All PoR files exist with content
- [ ] `grep -r "Truth lives in Plan of Record" scenarios/prompt-manager/store/skills/` returns no matches
- [ ] `grep -r "Truth lives in Plan of Record" docs/agent-system/` returns exactly one match
- [ ] `docs/meta-optimization/` is deleted
- [ ] Every consumer of a relocated skill (grep `TOOLS.md`, `HEARTBEAT.md`, `AGENTS.md` for the skill IDs) has been updated
- [ ] Tests in `scenarios/prompt-manager/api/...` and `scenarios/prompt-manager/cli/...` that load skills by ID pass after the deletions

### Phase 2 — Topics data layer (concurrent with Phase 1)

Goal: ship a working `topics.json` loader, API, and one team's backfill, before generalizing.

**Steps:**

1. **Loader + validator** (`scenarios/prompt-manager/api/internal/topics/`):
   - `schema.go` — Go struct + JSON Schema (from Phase 0)
   - `loader.go` — read all `topics.json` across `store/teams/*/members/*/`
   - `loader_test.go` — automated tests covering valid file, missing file, malformed JSON, schema violations

2. **API endpoints** (extend existing team handler):
   - `GET /teams/{team}/members/{member}/topics`
   - `PUT /teams/{team}/members/{member}/topics` (full-document write)
   - `GET /teams/{team}/topics` (all members of a team)
   - `GET /topics/graph` (full directed graph as JSON: nodes, edges, validation results)
   - `GET /topics/drain-status` (per-prefix queue depth + age + recent throughput)
   - Handler tests covering each endpoint, including 404 / malformed-payload paths

3. **Canary backfill — marketing-crew**:
   - Author `topics.json` for every member of marketing-crew (researcher, brand-manager, publisher, oss-advertiser, subscription-advertiser, marketing-contrarian)
   - Verify the loader, API, and validation rules all return clean for this team
   - This is the test bed: if the schema is wrong, fix it here before touching other teams

4. **Schema-stable gate:** operator acceptance via a `meta-optimization` decision before generalizing.

5. **Generalize backfill** to remaining teams: monetization, meta-optimization, infra-health, director-swarm. ≥30 total files.

**Exit criteria:**
- [ ] All loader/API tests pass
- [ ] Every member of every active team has a valid `topics.json`
- [ ] `GET /topics/graph` returns a non-empty graph with no malformed-data errors
- [ ] Schema-stable decision is accepted on the meta-optimization team

### Phase 3 — CLI: `graph topics` + validation rules

Depends on Phase 2 (loader + endpoints stable).

**Validation rules to implement** (each a separate function with its own test):

| Rule | Smell | Severity |
|---|---|---|
| `orphan_output` | A member's output prefix has no consumer (no router intake matches, no PoR sink declared) | error |
| `orphan_input` | A member's intake prefix has no producer (no member writes it, no external producer claims it) | error |
| `conflicting_drain` | Two members' intake prefixes overlap (both claim to drain the same topic) | error |
| `missing_drain_skill` | `drained_by_skill` references a skill ID that doesn't exist in the skills tree | error |
| `dangling_por_sink` | `destination_kind=por_file` references a `destination_path` that doesn't exist | error |
| `stalled_drain` | Intake prefix has unrouted entries older than threshold (default 7 days) | warning |
| `piling_inbox` | Intake prefix has > N unrouted entries (default 50) | warning |

**CLI surface:**

```bash
prompt-manager graph topics                   # human-readable directed graph + validation report
prompt-manager graph topics --team <name>     # restrict to one team (boundary nodes for cross-team edges)
prompt-manager graph drain-status             # per-prefix queue depth + age + recent throughput
prompt-manager graph drain-status --team <name>
```

Default output is human-readable per CLI convention (not `--json`). `--json` flag available for programmatic consumers (the UI uses the API directly, not this flag).

**Exit code semantics:**
- `0` — no smells
- `1` — any error-severity smell present
- Warnings do not affect exit code

**Tests:**
- Unit tests per validation rule with fixture `topics.json` files designed to trip each rule.
- Integration test running `prompt-manager graph topics` against a synthetic team set, asserting exit code and output content.
- Regression test for the canary marketing-crew backfill (must return zero smells).

**Exit criteria:**
- [ ] All validation rule tests pass
- [ ] `prompt-manager graph topics` returns exit 0 across all current teams (or operator-accepted exceptions are documented)
- [ ] `prompt-manager graph drain-status` output matches manual `team knowledge-list --topic-prefix=` counts

### Phase 4 — UI: dual-mode team graph

Depends on Phase 3 (API + CLI proven stable; graph data is correct).

**File-level changes:**

- `scenarios/prompt-manager/ui/src/types/orgChart.ts` — extend with topic-edge types (`TopicEdge`, `BoundaryNode`, `ValidationOverlay`)
- `scenarios/prompt-manager/ui/src/services/teamService.ts` — add `getTopicsGraph(teamId)`, `getDrainStatus(teamId)`
- `scenarios/prompt-manager/ui/src/components/editor/OrgChartPanel.tsx` — add mode toggle (`Hierarchy | Topics | Both`), conditional renderer, validation overlay
- `scenarios/prompt-manager/ui/src/components/editor/TopicEdgeDetail.tsx` — new side panel for queue inspection on edge click
- `scenarios/prompt-manager/ui/src/components/editor/TopicsValidationPanel.tsx` — new sidebar listing smells

**Behavioral spec:**

| Element | Hierarchy mode | Topics mode |
|---|---|---|
| Nodes | Members only | Members + boundary nodes (vision-walk, operator, decision queues, capability-gap registry, PoR sinks) |
| Edges | `managerId` relationships, dagre tree layout | Topic flow, force-directed layout (or dagre with explicit ranks for ingress/egress columns) |
| Edge label | None | Topic prefix (or stack icon when multiple between same pair) |
| Edge style | Default | Thickness ∝ drain rate; color = healthy/quiet/stalled/piling |
| Node decoration | Default | Red ring on validation errors; warning badge on warnings |
| Side panel on edge click | Member detail (existing) | Queue inspection: depth, oldest, recent routed |
| Validation panel | Hidden | Visible; lists smells from `graph topics` |

**Mode auto-default:** if `getTopicsGraph(teamId).edges.length === 0` AND `managerId`-based edges exist, default to Hierarchy. Else default to Topics. The "Both" toggle is opt-in only.

**Boundary-node collapse rule:** all PoR sinks collapse into one "canon" boundary node by default; click-to-expand reveals individual files. Decision queues collapse similarly. (UX fairness: don't lie about flow, don't drown the canvas.)

**Tests:**
- Component tests for each new component (`TopicEdgeDetail`, `TopicsValidationPanel`, mode toggle)
- React Flow rendering test: assert node count, edge count, edge labels for a synthetic team payload
- End-to-end (Playwright/scenario UI smoke) for: load team page, switch modes, click edge, inspect queue

**Exit criteria:**
- [ ] All component + e2e tests pass
- [ ] Mode toggle persists per-team via URL or local state (operator preference)
- [ ] Validation panel matches `prompt-manager graph topics` CLI output

### Phase 5 — Audit skill rewire (synchronization gate)

Depends on Phases 1, 2, 3 (PoR exists, topics.json exists everywhere, validation rules exist).

**Changes to `team-member-capability-architecture-audit`:**

- Phase 2 of the skill (Score the Nine Layers) for the four pipeline layers (Intake / Collection / Analysis / Promotion) now derives scores from `topics.json` instead of prose grep:
  - Intake score: 0 if no `intake` entries; 2 if at least one with `drained_by_skill`; 3 if validation passes for that prefix
  - Collection score: 2 if `external_producers` declared OR collection skill exists; 3 if collection is a CLI/Action
  - Analysis score: 2 if a method skill is referenced from `drained_by_skill`; 3 if multiple methods declared
  - Promotion score: derived from `output` declarations + `decisions_owned` + `raises_capability_gaps` boolean
- Identity, Ownership, Plan of Record, Skill Surface, and Feedback Loop layers remain prose-judgment.
- Output contract gains a "Validation report" subsection auto-populated from `prompt-manager graph topics --team <name>`.

**Lint rule (new):** the skill's audit procedure now flags any skill whose content includes verbatim canon strings (the layer mantra, the promotion-ladder ordering, etc.) as "skillless canon residue" — the dogfooding rule.

**Exit criteria:**
- [ ] Audit skill consumes `topics.json` programmatically (via prompt-manager CLI within the skill's procedure)
- [ ] Running the audit against marketing-crew/researcher produces structurally consistent layer scores

### Phase 6 — Cleanup + scenario restart + health verification

Per the user's standing requirement (`feedback_planning_guidelines`):

1. Run `go build ./...` and `go test ./...` on `scenarios/prompt-manager/api/`; fix all errors and warnings, including pre-existing.
2. Run `npx tsc --noEmit` and `eslint` on `scenarios/prompt-manager/ui/`; fix all errors and warnings in modified files, including pre-existing.
3. Run unit tests on the whole prompt-manager scenario; fix all failures.
4. `vrooli scenario restart prompt-manager`
5. Verify health: API health endpoint returns 200, UI loads, team graph page renders for one team in each mode, `prompt-manager graph topics` returns clean exit code.

**Exit criteria:**
- [ ] All builds, lints, type checks, and tests pass on prompt-manager
- [ ] Scenario restarts cleanly
- [ ] Manual UI smoke: open a leaderless team in Topics mode, confirm edges render with validation overlay

---

## 9. Contract Decisions

### `topics.json` schema

Authoritative shape defined in Phase 0. Backwards-incompatible changes after Phase 2 schema-stable gate require a new `meta-optimization` decision.

### `prompt-manager graph topics` CLI

- Default human output (per `feedback_cli_default_human_output` memory)
- `--json` available for programmatic consumers
- Exit `0` on no errors, `1` on any error-severity validation failure, warnings ignored for exit
- `--team <name>` filter

### API endpoints

All under existing `/teams/...` and new `/topics/...` namespaces. Read endpoints public to authenticated UI; write endpoints follow same auth pattern as existing team mutation endpoints.

### `docs/agent-system/` is plan-of-record

- Approval-gated (file edits go through `meta-optimization` decisions; agents propose diffs, never edit directly)
- Cross-team-readable (any team's members may cite it as required reading)
- One concept lives in exactly one file (the "no double residency" rule from `team-shared-docs-design`)

### `docs/meta-optimization/` is deleted

Existing content fully migrated. Future genuinely-transient meta-optim observations live as team knowledge entries under `meta-optimization/notebook/<slug>` topic prefix, drained by an existing or future meta-optim router skill.

---

## 10. Testing Plan

All verification is automated. No manual checklists.

| Layer | Test type | Location |
|---|---|---|
| `topics.json` schema | Unit (Go) | `scenarios/prompt-manager/api/internal/topics/schema_test.go` |
| Topics loader | Unit (Go) | `scenarios/prompt-manager/api/internal/topics/loader_test.go` |
| API endpoints | Handler tests (Go) | `scenarios/prompt-manager/api/internal/topics/handler_test.go` |
| Validation rules | Unit (Go), one fixture per rule | `scenarios/prompt-manager/api/internal/topics/validation_test.go` |
| `graph topics` CLI | Integration (Go), exit-code + output assertions | `scenarios/prompt-manager/cli/cmd_graph_test.go` (or equivalent) |
| `graph drain-status` CLI | Integration | same |
| UI types | TS compile | `npx tsc --noEmit` |
| UI components | Vitest component tests | `scenarios/prompt-manager/ui/src/components/editor/__tests__/` |
| UI e2e | Scenario UI smoke (Playwright via existing harness) | `scenarios/prompt-manager/coverage/ui-smoke/` |
| PoR coherence | grep-based regression test | `scenarios/prompt-manager/test/agent_system_canon_test.sh` (new) — asserts no canon residue in skills, no double residency |

The PoR coherence test is the dogfooding lint: it greps for the canonical layer-mantra sentence and ensures it appears in `docs/agent-system/LAYERS.md` and nowhere else under `scenarios/prompt-manager/store/skills/`. If a future skill drifts and restates canon, this test fails.

---

## 11. Rollout / Validation Checklist

Per phase, gated:

- [ ] Phase 0: schema approved, outline approved (meta-optimization decision)
- [ ] Phase 1: PoR coherence test passes; all skill consumers updated; `docs/meta-optimization/` deleted
- [ ] Phase 2: marketing-crew canary clean; schema-stable decision accepted; full backfill complete; loader + API tests pass
- [ ] Phase 3: all validation rule tests pass; `graph topics` returns 0 across all teams (or exceptions documented)
- [ ] Phase 4: UI tests pass; mode toggle works; validation panel matches CLI output
- [ ] Phase 5: audit skill consumes topics.json; structural scoring matches expected for known members
- [ ] Phase 6: lint + type + tests + restart + health all clean

---

## 12. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Schema churn during backfill (discover schema is wrong on member 12, redo 1-11) | Medium | Phase 2 canary on marketing-crew only; schema-stable decision required before generalizing |
| Three framings of the promotion ladder genuinely disagree (not just editorial drift) | Medium | Phase 0 outline review explicitly looks for semantic conflicts; surface them as `meta-optimization` decisions before authoring |
| UI complexity creep (force-directed layout + boundary nodes + validation overlay + side panels in one phase) | High | Phase 4 split: ship Topics-mode rendering first with read-only edges and basic validation; defer side-panel queue inspection if it slips |
| Skills referenced from many `TOOLS.md` files; deleting `skill-principles` breaks consumers silently | Medium | Phase 1 includes a tree-wide grep for every deleted skill ID and an update to every consumer; the PoR coherence test catches residual references |
| `docs/meta-optimization/` consumers outside prompt-manager scenario | Low-Medium | Tree-wide grep for `docs/meta-optimization/` references at start of Phase 1; update or document each |
| Operator acceptance gates (Phase 0, Phase 2 schema-stable) block progress | Low | Surface decisions early; both gates sized to be one-walk decisions during the morning vision walk |
| Phase 1 and Phase 2 concurrent execution causes merge conflicts on `team-member-capability-architecture-audit` skill (Phase 1 strips canon; Phase 5 rewires) | Low | Phase 5 explicitly waits on Phase 1 completion; Phase 1 leaves the skill's procedure intact for Phase 5 to extend |

---

## 13. Non-goals / Prohibited Patterns

- **No backwards compatibility shims.** Deleted skills are deleted, not stubbed. Moved doc files are moved, not redirected. Renamed concepts are renamed everywhere in one move.
- **No double residency of canon.** The same definition does not live in two files. The PoR coherence test enforces this.
- **No prose graph documentation as a substitute for `topics.json`.** Once the data layer ships, member message-flow is declared in JSON or it does not exist. Prose descriptions of "this member receives X and emits Y" in `RESPONSIBILITIES.md` should be deleted in favor of the structured declaration.
- **No drag-drop topic editing in v1.** Side-panel JSON editor only. Visual editing is a follow-up.
- **No system-wide cross-team graph page in v1.** Per-team page only.
- **No `// removed` comments, no `_unused` renames, no re-export shims** anywhere this plan touches.
- **Do not expand inbox/router pattern adoption.** Adoption agents already in flight handle that. This plan only consolidates doctrine + adds the structural data layer.

---

## 14. Definition of Done

All of the following must be true:

1. `docs/agent-system/` exists with all 8–10 PoR files; `drafts/` subfolder exists for synthesis-in-flux.
2. `docs/meta-optimization/` is deleted; tree-wide grep finds no references to it under `docs/`, `scenarios/prompt-manager/`.
3. PoR coherence test (`scenarios/prompt-manager/test/agent_system_canon_test.sh`) passes — layer mantra appears in exactly one file under `docs/agent-system/`, zero residue elsewhere.
4. `skill-principles` and `team-shared-docs-design` skills are deleted; remaining skills have required-reading lines pointing at PoR.
5. Every member of every active team has a `topics.json` validating against the schema.
6. `prompt-manager graph topics` exits 0 across all teams; `prompt-manager graph drain-status` runs without error.
7. UI team graph page renders Hierarchy and Topics modes; validation overlay matches CLI output.
8. `team-member-capability-architecture-audit` skill consumes `topics.json` for the four pipeline layers.
9. `vrooli scenario restart prompt-manager` succeeds; API health and UI smoke pass.
10. All Go builds, type checks, lints, and tests pass on `scenarios/prompt-manager/` — including pre-existing.
11. All TS type checks, lints, and component tests pass on `scenarios/prompt-manager/ui/`.
12. The greenfield constraint held: no shims, no `// removed`, no re-exports, no renamed `_unused` variables introduced.
