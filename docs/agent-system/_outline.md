# Agent-System Migration Outline

This file is the **Phase 1 migration manifest** for `path:docs/plans/agent-system-migration-implementation-plan.md`. Each row maps a canonical paragraph in an existing skill or doc to a destination PoR file in `path:docs/agent-system/`. Phase 1 executes paired atomic moves: extract content → place in PoR file → strip source of relocated section → add `Required reading: docs/agent-system/<file>` line.

## Status

- Phase 0: this manifest authored. **Operator acceptance required** before Phase 1 starts (file as a `meta-optimization` decision; reference this file).
- Conflicts to surface during review (see "Three-framings conflict" below) must be resolved before extraction.

## Conventions

- **Source line ranges** are taken from the skill files as of the date this manifest was authored (2026-05-01). They will drift; treat as locator hints, not addresses.
- **Move** = the section is extracted and the source loses it. The source skill keeps a pointer.
- **Absorb wholesale** = the entire skill file is moved; the skill is deleted in the same commit.
- **Cite-only** = the skill keeps the section but rewrites it as one or two lines pointing at PoR; no body content remains in the skill.
- **Keep in skill** = the section stays in the skill because it's procedure (not canon).

## Three-framings conflict to resolve

The promotion ladder is currently described in three places with conflicting vocabulary:

| Source | Framing | Notes |
|---|---|---|
| `skill-principles/SKILL.md` §6 | "Promotion-Retirement Lifecycle" — 4-step lifecycle (interim prose → CLI/tool contract → Action → retire prose) | Most operationally specific |
| `team-member-capability-architecture-audit/SKILL.md` §2 | "Layer Model" — 9-layer table mixed with the layer mantra | Most conceptually clean; canonical mantra source |
| `path:docs/meta-optimization/README.md` "three-tier mental model" | Hot buffer → Living notebook → Permanent structure | Frames notebook as a permanent layer (which the migration plan retires) |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` "Promotion Classifier" | "If it says X → Y" classifier (truth → PoR, decide → skill, run → Action, etc.) | Cleanest formulation, but lives in a notebook file |

**Resolution direction (proposed; needs operator confirmation):**

1. `LAYERS.md` becomes the single home for the layer mantra and the "If it says X → Y" classifier — they're complementary statements of the same rule.
2. `PROMOTION_LADDER.md` becomes the single home for the 4-step lifecycle (interim prose → CLI/tool → Action → retire), with retention/retirement criteria.
3. The "hot buffer / living notebook / permanent structure" three-tier framing is **retired**. The new model: hot buffer = per-heartbeat shared state (unchanged); inbox + synthesis = team knowledge entries under topic prefixes (already adopted by marketing-crew, monetization); permanent structure = PoR + skills + Actions + CLIs. Notebooks-as-markdown-files survive only as `drafts/` adjacent to canon, on the synthesis path.

Operator: please confirm or redirect during the Phase 0 review decision.

## Migration manifest

Rows are listed by destination PoR file. Each row's "action on source" column states what happens to that source after Phase 1 completes.

### `path:docs/agent-system/PRIMITIVES.md`

Defines the eight primitives: skill, agent, team (incl. member), plan-of-record, decision, knowledge, action, CLI. Plus three runtime concepts: notebook, capability-gap, scenario.

| Source | Section | Action on source |
|---|---|---|
| `skill-principles/SKILL.md` | §1 "What Skills Are (and Are Not)" | Move; keep one-line cite |
| `skill-principles/SKILL.md` | §2 "Scope Boundaries" | Keep in skill (per-skill scope, not framework canon) |
| `skill-principles/SKILL.md` | §3 "Choose the Right Category" + decision check | Move (this is a primitive-typology question) |
| `team-shared-docs-design/SKILL.md` | "Pattern A — Plan-of-record" + "Pattern B — Working notebook" definitions | Move (just the definitions; the four-axis discussion lives in `TEAM_DOCS_PATTERNS.md`) |
| `team-member-capability-architecture-audit/SKILL.md` | §2 "Layer Model" table — column "Belongs in" | Move (this is what each primitive holds) |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | "Promotion Classifier" (8 lines) | Move (also referenced by `LAYERS.md`; lives once here, cited from `LAYERS.md`) |

### `path:docs/agent-system/LAYERS.md`

Single canonical home for the layering rule. Cites `PROMOTION_LADDER.md` for the lifecycle and `PRIMITIVES.md` for the typology.

| Source | Section | Action on source |
|---|---|---|
| `skill-principles/SKILL.md` | §6 "Layering rule" block (the layer mantra) | Move (single canonical home) |
| `team-member-capability-architecture-audit/SKILL.md` | §2 "Keep the layers separate" 8-line block | Move (it's the same mantra rephrased) |
| `team-shared-docs-design/SKILL.md` | "Use the right permanent structure" bullet | Move (yet another rephrasing) |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | "Promotion Classifier" cross-reference | Cite-only |
| `path:docs/meta-optimization/README.md` | "three-tier mental model" diagram | Retire (replaced by the inbox-pipeline model in `INTAKE_PIPELINE.md`); see "Three-framings conflict" |

**Test gate:** after Phase 1, the canonical layer-mantra sentence (the one beginning "Truth ... in Plan of Record") appears exactly once under `path:docs/agent-system/` (in `LAYERS.md`) and zero times under `path:scenarios/prompt-manager/store/skills/`. Enforced by `path:scenarios/prompt-manager/test/agent_system_canon_test.sh`.

### `path:docs/agent-system/PROMOTION_LADDER.md`

Lifecycle of a guidance from raw observation to retired prose. Cites `LAYERS.md` and `INTAKE_PIPELINE.md`.

| Source | Section | Action on source |
|---|---|---|
| `skill-principles/SKILL.md` | §6 "Promotion-Retirement Lifecycle" 4-step block | Move (canonical home) |
| `skill-principles/SKILL.md` | §6 "Retirement criteria" + "Retention criteria" | Move |
| `skill-principles/SKILL.md` | §6 "Output requirement for meta analyses" | Move |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | "When to Attempt Conversion" (3 prerequisites) | Move |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | "Conversion Procedure" 8-step block | Move |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | "Patterns" + "Anti-Patterns" + "Conversion Log" + "Open Questions" | Retire (these were notebook-mode entries; the conversion log can be replaced by a structured `actions` index if it has continued value, otherwise it dies) |

### `path:docs/agent-system/TEAM_DOCS_PATTERNS.md`

Wholesale absorption of `team-shared-docs-design`. The skill is then deleted.

| Source | Section | Action on source |
|---|---|---|
| `team-shared-docs-design/SKILL.md` | All sections except primitive definitions (which moved to `PRIMITIVES.md`) | Move (skill deleted in same commit) |

**Skill outcome:** `team-shared-docs-design` is deleted. Tree-wide grep for `team-shared-docs-design` ID is run; every consumer's `TOOLS.md` / `HEARTBEAT.md` / `AGENTS.md` is updated to cite `path:docs/agent-system/TEAM_DOCS_PATTERNS.md` instead.

### `path:docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md`

The 9-layer member capability model. Sourced from the audit skill, but stripped of audit-procedure specifics (those stay in the skill).

| Source | Section | Action on source |
|---|---|---|
| `team-member-capability-architecture-audit/SKILL.md` | §2 "Layer Model" table (just the layer definitions, not audit guidance) | Move |
| `team-member-capability-architecture-audit/SKILL.md` | §4 "Intake, Collection, Analysis, Promotion" pipeline definitions | Move (also lives in `INTAKE_PIPELINE.md`; here only the 9-layer-relevant slice) |
| `team-member-capability-architecture-audit/SKILL.md` | §1 "When to Use" + §3 "Audit Process" Phase 1–4 + §3 architecture-smells table + §3 "Choose the Smallest Useful Fix" + §5 "Output Contract" + §6 "Worked Pattern" | Keep in skill (these are audit procedure, not canon) |

**Skill outcome:** `team-member-capability-architecture-audit` is kept. §2 + §4 are replaced by `Required reading: docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md`. Phase 5 of the migration plan additionally rewires the skill's Phase 2 scoring to consume `topics.json` programmatically.

### `path:docs/agent-system/INTAKE_PIPELINE.md`

The Intake → Collection → Analysis → Promotion pipeline. Inbox-router-drain pattern. Topic-prefix conventions. Cites `LAYERS.md` and `TEAM_MEMBER_ARCHITECTURE.md`.

| Source | Section | Action on source |
|---|---|---|
| `team-member-capability-architecture-audit/SKILL.md` | §4 entire section (the model) | Move (lives once here, `TEAM_MEMBER_ARCHITECTURE.md` cites it) |
| `marketing-research-router/SKILL.md` | "Routing Process" steps 1–5 — the **generic** parts (normalize, classify, choose smallest action, apply collection discipline, resolve inbox entry under topic-prefix retag/delete) | Move (the routing-process *pattern* is canon; the marketing-specific signal-type table and method registry stay in the skill) |
| `monetization-opportunity-router/SKILL.md` | (TBD — analogous router pattern) | Cite-only (pattern lives once in `INTAKE_PIPELINE.md`; the skill keeps monetization-specific tables) |
| `market-validation-router/SKILL.md` | (TBD — analogous) | Cite-only |
| `path:docs/marketing/research/README.md` | "Pipeline" + "Intake" + "Inbox convention" + "Routing inbox entries" + "Collection" + "Analysis Methods" + "Promotion Matrix" + "Evidence Rules" | Move the **generic** patterns; keep the marketing-specific registry in `path:docs/marketing/research/README.md`, which becomes a marketing-specific instantiation citing `path:docs/agent-system/INTAKE_PIPELINE.md` |

### `path:docs/agent-system/SKILL_AUTHORING.md`

Universal quality bars for any skill. Per-category specifics (steer, search, tools, practice, meta) stay in the per-category authoring skills.

| Source | Section | Action on source |
|---|---|---|
| `skill-principles/SKILL.md` | §4 "Universal Quality Bars" | Move |
| `skill-principles/SKILL.md` | §5 "Referencing Other Skills" | Move |
| `skill-principles/SKILL.md` | §7 "Registration and Metadata" | Move |
| `skill-principles/SKILL.md` | §8 "Avoid Skill Sprawl" | Move |
| `skill-principles/SKILL.md` | §9 "Output Expectations" | Move |
| `skill-principles/SKILL.md` | §10 "Skill Architecture Heuristics" | Move |
| `skill-authoring/SKILL.md` | §1 "Shared Mental Model Problem" | Move |
| `skill-authoring/SKILL.md` | §2 "Convergence Patterns" | Move |
| `skill-authoring/SKILL.md` | §3 "Principles Over Prescriptions" | Move |
| `skill-authoring/SKILL.md` | §4 "Clear Intent Statement" (excluding §4.1 which is Steer-specific) | Move |
| `skill-authoring/SKILL.md` | §4.1 "Steer Skills Must Target a Specific Scenario" | Keep in `skill-authoring` (Steer-specific) |
| `skill-authoring/SKILL.md` | §5 "Boundary Definition" | Move |
| `skill-authoring/SKILL.md` | §6 "Skill Structure" | Move |
| `skill-authoring/SKILL.md` | §7 "Anti-Gaming Measures" | Move |
| `skill-authoring/SKILL.md` | §8 "Agent Memory Loop" | Move |
| `skill-authoring/SKILL.md` | §9 "Protective Comments" | Move |
| `skill-authoring/SKILL.md` | §10 "Registration" | Cite-only (registration is also in `skill-principles` §7; one PoR home) |
| `skill-authoring/SKILL.md` | §11 "Maintain Skill System Constraints" | Move |
| `skill-authoring/SKILL.md` | §12 "Output Expectations" | Keep in `skill-authoring` (specific to authoring sessions, not framework-wide quality bars) |
| `skill-authoring-{meta,platform,practice,search,tools}/SKILL.md` | Common-quality content (decision-table guidance, intent-statement format, etc.) | Move; per-category specifics retained |

**Skill outcomes:**
- `skill-principles` is fully absorbed and **deleted**.
- `skill-authoring` is **kept**, slimmed to just §4.1 (Steer-specific opening pattern), §10 (registration cite), §12 (per-session output expectations); plus a `Required reading: docs/agent-system/SKILL_AUTHORING.md` line.
- `skill-authoring-{meta,platform,practice,search,tools}` are **kept**, each slimmed to category-specific content with a required-reading line pointing at `SKILL_AUTHORING.md`.

### `path:docs/agent-system/DEPRECATION_POLICY.md`

Wholesale relocation of `path:docs/meta-optimization/DEPRECATION_POLICY.md`.

| Source | Section | Action on source |
|---|---|---|
| `path:docs/meta-optimization/DEPRECATION_POLICY.md` | All sections | Move (file deleted) |

### `path:docs/agent-system/REFERENCE_SCENARIOS.md`

Wholesale relocation of `path:docs/meta-optimization/REFERENCE_SCENARIOS.md`.

| Source | Section | Action on source |
|---|---|---|
| `path:docs/meta-optimization/REFERENCE_SCENARIOS.md` | All sections | Move (file deleted) |

### `path:docs/agent-system/TOPICS_SCHEMA.md` (originally `drafts/topics-schema.md`)

Authored fresh in Phase 0 as `drafts/topics-schema.md`; promoted to canon at `TOPICS_SCHEMA.md` during the inbox-flow refactor. Paired with the Go schema at `path:scenarios/prompt-manager/api/memberflow/schema.go`. Documents the per-member topic declarations data layer (intake / output / decisions_owned / decisions_consumed / raises_capability_gaps / external_producers).

| Source | Action on source |
|---|---|
| New file (Phase 0 deliverable) | n/a |

## Skills retired in Phase 1

| Skill ID | Reason | Consumer-update plan |
|---|---|---|
| `skill-principles` | Fully absorbed into `LAYERS.md`, `PRIMITIVES.md`, `PROMOTION_LADDER.md`, `SKILL_AUTHORING.md` | Tree-wide grep for `skill-principles` in `TOOLS.md` / `HEARTBEAT.md` / `AGENTS.md`; replace with `path:docs/agent-system/<file>` cites |
| `team-shared-docs-design` | Fully absorbed into `TEAM_DOCS_PATTERNS.md` and `PRIMITIVES.md` | Same grep + replace |

## Files relocated in Phase 1

| Source path | Destination path | Note |
|---|---|---|
| `path:docs/meta-optimization/README.md` | (none — deleted) | Replaced by `path:docs/agent-system/README.md` |
| `path:docs/meta-optimization/CONVERSION_PLAYBOOK.md` | content split between `LAYERS.md` and `PROMOTION_LADDER.md`; file deleted | Notebook entries (Patterns, Anti-Patterns, Log, Open Questions) retired or moved to team knowledge entries under `meta-optimization/notebook/<slug>` |
| `path:docs/meta-optimization/DEPRECATION_POLICY.md` | `path:docs/agent-system/DEPRECATION_POLICY.md` | Wholesale move |
| `path:docs/meta-optimization/REFERENCE_SCENARIOS.md` | `path:docs/agent-system/REFERENCE_SCENARIOS.md` | Wholesale move |
| `path:docs/meta-optimization/` (folder) | n/a | Deleted at end of Phase 1 |

## Consumers to update in Phase 1

Anything in this list must be re-pointed before the source is removed. The Phase 1 atomicity rule still applies: per **piece** of canon, the same commit that creates the PoR file removes the source content and updates consumers. Bulk consumer updates may be a separate commit when the consumer change is purely a citation rewrite (no content drift).

- `path:scenarios/prompt-manager/store/skills/packs/core/*/SKILL.md` — any skill that lists `skill-principles` or `team-shared-docs-design` as required/optional reading
- `path:scenarios/prompt-manager/store/teams/*/members/*/HEARTBEAT.md` — same grep
- `path:scenarios/prompt-manager/store/teams/*/members/*/RESPONSIBILITIES.md` — same
- `path:scenarios/prompt-manager/store/agents/*/AGENTS.md` and `TOOLS.md` — same
- `path:scenarios/prompt-manager/store/teams/*/shared/TEAM.md` — any team that cites a relocated doc
- `path:docs/marketing/research/README.md` — already plan-of-record; will gain a citation line to `path:docs/agent-system/INTAKE_PIPELINE.md` and lose any duplicated framing
- Top-level `CLAUDE.md` — currently references `path:docs/meta-optimization/` indirectly via prompt-manager skill paths; check for direct mentions

## Out of scope for Phase 1

- Updating consumers in scenarios outside `prompt-manager` and `docs/`. If a stray reference exists in another scenario's docs, file a follow-up `meta-optimization` decision.
- Rewriting the audit skill's Phase 2 scoring to consume `topics.json` (deferred to Phase 5 of the migration plan).
- Authoring new draft content beyond `topics-schema.md` (drafts emerge as needed, not bulk-authored).
