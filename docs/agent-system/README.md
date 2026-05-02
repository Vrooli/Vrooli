# Agent System

Plan-of-record for the prompt-manager self-improvement framework: how skills, agents, teams, plans, decisions, knowledge, notebooks, actions, and CLIs fit together.

This folder is **canon**. Edits go through approved decisions on the `meta-optimization` team. Other teams cite files here as required reading; nobody outside the meta-optimization decision flow rewrites them in place.

## Status

This is the **Phase 0 skeleton**. Files listed below either do not yet exist or contain only placeholders. They will be filled by Phase 1 of the migration plan in `docs/plans/agent-system-migration-implementation-plan.md`, where each canonical paragraph is moved out of an existing skill in a paired atomic commit.

Until Phase 1 completes, the source-of-truth for each topic remains the skill listed in `_outline.md` under the corresponding row.

## Files

| File | Status | Covers |
|---|---|---|
| `_outline.md` | authored (Phase 0) | Migration manifest mapping source-skill sections to destination PoR files |
| `PRIMITIVES.md` | TBD (Phase 1) | What skills, agents, teams, plans, decisions, knowledge, notebooks, actions, CLIs are; how they relate |
| `LAYERS.md` | TBD (Phase 1) | The layering rule (truth / judgment / execution / implementation / unbuilt / raw learning) — single home for the layer mantra |
| `PROMOTION_LADDER.md` | TBD (Phase 1) | Lifecycle of a guidance: notebook → skill → CLI → Action; retirement criteria |
| `TEAM_DOCS_PATTERNS.md` | TBD (Phase 1) | Plan-of-record vs working-notebook patterns, the four axes, both-patterns rules |
| `TEAM_MEMBER_ARCHITECTURE.md` | TBD (Phase 1) | The 9-layer member capability model (Identity, Ownership, Plan of Record, Skill Surface, Intake, Collection, Analysis Method, Promotion / Routing, Feedback Loop) |
| `INTAKE_PIPELINE.md` | TBD (Phase 1) | Intake → Collection → Analysis → Promotion pipeline; inbox-router-drain pattern; topic-prefix conventions |
| `SKILL_AUTHORING.md` | TBD (Phase 1) | Universal authoring quality bars (intent statement, boundaries, convergence patterns, output expectations, troubleshooting section, anti-gaming) |
| `DEPRECATION_POLICY.md` | TBD (Phase 1) | Staleness windows, mandatory roadmap check, archive path, who-files-what |
| `REFERENCE_SCENARIOS.md` | TBD (Phase 1) | Gold-star reference scenario registry, nomination + demotion rules, rot triage |
| `drafts/` | folder | Synthesis-in-flux content not yet promoted to canon. Subject to faster churn; reviewed by meta-optimization before promotion |
| `drafts/topics-schema.md` | authored (Phase 0) | Human-readable PoR for the `topics.json` data layer — paired with `scenarios/prompt-manager/api/memberflow/schema.go` |

## Editing rules

1. **Approval-gated.** Operator-curated via `meta-optimization` decisions. Agents propose diffs; they never edit directly.
2. **Cross-team-readable.** Any team's members may cite a file here as required reading.
3. **One concept, one file.** No double residency. The PoR coherence test (`scenarios/prompt-manager/test/agent_system_canon_test.sh`) enforces this.
4. **Skills cite, never restate.** Any skill that previously contained doctrine in this folder must drop it and add a `Required reading: docs/agent-system/<file>` line.
5. **Drafts are not canon.** `drafts/` exists for content that is being workshopped before it becomes a stable PoR file. Drafts may be cited only by the same team's draft skills, not by external consumers.

## Naming note: `topics.json` vs `api/topics/` package

The per-member data file is `topics.json` (declares intake/output topic-prefixes for the inbox-router-drain pattern). The Go implementation lives at `scenarios/prompt-manager/api/memberflow/` because the existing `scenarios/prompt-manager/api/topics/` package serves a different concern (content-taxonomy topics with parent/child relationships and attached skills). Keeping the data-file name as `topics.json` matches the inbox topic-prefix vocabulary; the package is renamed to avoid collision.

## Folder origin

Migrated from `docs/meta-optimization/` (which previously declared itself a "working notebook" but actually contained framework canon — see Phase 1 of the migration plan for the resolution).
