# Idea Pipeline

> **Status:** v1. Authored 2026-04-28 (walk #5 divergence #5).
>
> **Purpose:** the **operator-curated, capacity-deferred staging area** for project-wide ideas — scenarios, features, team-member additions, platform upgrades, capability proposals — that aren't ready to enter any team's active pipeline yet, but shouldn't be lost.
>
> **What this isn't:**
> - opportunity-scout's raw pool (that's `monetization` team knowledge entries under `topic[example]:monetization/opportunity/<slug>` and `topic[example]:opportunity-inbox/<signal-type>/<slug>` topics)
> - the SKU index (that's [`docs/monetization/catalogs/CATALOG.md`](../../monetization/catalogs/CATALOG.md))
> - engineering-pipeline staging (that's `path:scenarios/swarm-manager/ideas/`)
> - settled monetization reference material (that's [`path:docs/monetization/catalogs/revenue-lines/`](../../monetization/catalogs/revenue-lines/))
> - team-internal craft observations (that's the `notebook/` subdirectories under each team-shared docs folder)

## Write rule (load-bearing)

**Only the operator writes here. Agents propose additions via decisions; agents NEVER write directly.**

This is the single rule that keeps idea-pipeline structurally distinct from team-pipeline surfaces. If any team's heartbeat starts writing here directly, the surface collapses into another team's queue and breaks the **capacity-deferred** property — which is the entire reason this surface exists. Surface this rule prominently when reviewing or proposing changes. Violating it is the failure mode that retires this whole convention.

The convention applies even to the operator's own automated tooling: a script or skill that auto-populates idea-pipeline entries from any agent surface (monetization opportunity pool, marketing observations, etc.) violates the spirit of this rule. The operator may copy *content* from those sources by hand; the operator is the curation gate.

## Why this surface exists

Project-wide ideas come from many sources. They aren't all the same shape:

- A SKU-shaped idea fits opportunity-scout's flow → `topic[example]:monetization/opportunity/<slug>` knowledge entry → CATALOG candidate.
- A scenario-shaped idea ready to build fits swarm-manager/ideas/ → workshop → backlog.
- An engineering insight that's too immature for either of the above, but valuable enough that losing it would be a real cost — *that's where idea-pipeline lives.*

The 53 archived ideas in `path:scenarios/swarm-manager/ideas/` are partial evidence of the prior gap: many were archived because there was no other place for "captured but not ready" — they had to use swarm-manager's archive workflow. With idea-pipeline in place, future capacity-deferred ideas land here instead of in a team-pipeline's archive.

## Sources of ingestion

Ideas can arrive from any of these sources. None of them write directly to idea-pipeline; the operator promotes from each:

| Source | What lives there | Promotion to idea-pipeline |
|---|---|---|
| Vision-walk **chore phase** (Phase 6) | Operator narrating outside-Vrooli activities; surfaces capability gaps as friction observations | Operator decides during Phase 8 of the walk; idea-pipeline is one valid "note for later" target |
| Vision-walk **big-picture phase** (Phase 7) | Operator + agent ideating from chore-audit raw material + bundle-roadmap context | Same as above |
| **Social-media alpha extraction** | External viral posts / tools / strategies the operator surfaces during a walk; structural patterns get extracted to marketing canon, but residual scenario-shaped ideas may need their own home | Operator promotes during Phase 8; cite the source URL in the entry |
| **opportunity-scout heartbeat** | SKU-shaped monetization candidates (knowledge entries under `topic[example]:monetization/opportunity/<slug>`) — usually destined for CATALOG promotion | When an opportunity is **broader-than-SKU** (a scenario hypothesis the team will eventually act on, not a SKU-shaped offering) OR **not-yet-ready-for-active-tracking**, opportunity-scout proposes idea-pipeline as the promotion target via a `catalog-promotion`-class decision instead of (or in addition to) a CATALOG candidate file |
| **Marketing-crew researcher** | Audience scans, competitive observations | When a research observation surfaces a *capability gap* that maps to a project-wide idea (vs. an audience or marketing-craft observation), the researcher proposes promotion via decision |
| **BIH ideation-extraction** *(future)* | Bookmark-intelligence-hub-derived idea candidates from operator's bookmark stream | When BIH ships, its extraction agent proposes promotions via decision |
| **Operator notes** | Direct operator capture during ad-hoc thinking | Operator writes the entry directly |
| **Team-member suggestions** | Any team's heartbeat may surface a scenario-shaped idea outside its primary scope | Team raises a decision proposing the entry; operator approves |

## Decision tree: where does an idea go?

Ask the questions in order. The first match wins.

1. **Is the idea a Tier-4 hardware proposal?** → Operator-only territory. Out of scope here. Capture in `path:docs/strategy/long-term-capability-flags.md` or a dedicated strategy doc.
2. **Is the idea a marketing-craft pattern (post technique, hook style, voice rule)?** → marketing canon (`path:docs/marketing/methods/post-techniques/` or `path:docs/marketing/catalogs/post-types/`).
3. **Is the idea a settled monetization reference (revenue-line architecture, pricing benchmark)?** → `path:docs/monetization/catalogs/revenue-lines/<line>.md` reference section.
4. **Is the idea a SKU-shaped offering (bundle / add-on / services line)?** AND **does it need active tracking now?** → `path:docs/monetization/catalogs/CATALOG.md` candidate (via opportunity-scout → catalog-promotion decision).
5. **Is the idea a SKU-shaped offering** but **revisit-trigger-deferred?** → opportunity-scout's pool (knowledge entry under `topic[example]:monetization/opportunity/<slug>`).
6. **Is the idea a scenario / feature / team-member / platform-upgrade ready to be worked on now?** → `path:scenarios/swarm-manager/ideas/<slug>/` as a standard backlog item.
7. **Is the idea a scenario / feature / team-member / platform-upgrade** that's well-shaped but **not capacity-available**? → **idea-pipeline** (this folder).
8. **Is the idea half-formed / single-signal / no clear shape?** → don't capture. Cheap-to-recover ideas don't pay the capture cost.

The goal is to keep idea-pipeline narrow: well-shaped ideas, capacity-deferred only.

## Information-tier discipline

Two tiers. Don't conflate.

### Tier 1 (capture — required, ~5 minutes)

Every entry has these in its `README.md` frontmatter or top section:

- **name** — slug-style, kebab-case, matches folder name.
- **summary** — one sentence. What this idea is.
- **source** — citation: where this idea came from. Vision-walk-N / social-media URL / opportunity-scout-id / operator-notes / etc.
- **sourced_at** — date the idea entered the pipeline.
- **status** — one of `raw | staged | evaluating | promoted | retired`.

That's it for capture. If filling in more would be premature, leave the optional sections empty. **The discipline is "honestly empty beats performatively filled."**

### Tier 2 (deepen on evaluation — optional, ~30 minutes per section)

When the operator (or a delegated team member via decision) decides to evaluate an idea for promotion, the README's optional sections are filled in:

- **Monetization framing** — how this would generate or support revenue. Reference `revenue-lines/` and `CATALOG.md` SKUs as appropriate. Note "not directly monetizable but enables X" if it's a capability multiplier.
- **Marketing framing** — how this would be marketed. Reference `path:docs/marketing/catalogs/post-types/` and `path:docs/marketing/methods/post-techniques/` as appropriate. Note audience persona and conversion goal.
- **Capability multipliers** — what does this unlock for other scenarios? What does it require? What scenarios are downstream consumers?
- **Goal alignment** — how this serves project goals. Reference `path:docs/strategy/` framing docs (`context.md`, `roadmap.md`, `business-solutions.md`) explicitly.

Sections may grow long. **When a section grows past ~500 words, it splits into its own file** (`monetization-framing.md`, etc.) inside the idea's folder. Until then, all sections live in the single `README.md`. Splits happen lazily when content demands; never preemptively.

## Lifecycle

```
raw ──▶ staged ──▶ evaluating ──▶ promoted ──▶ (closed)
                                       │
                                       └──▶ retired (with reason)
```

| Status | Meaning |
|---|---|
| `raw` | Just captured. Tier 1 only. Operator hasn't decided whether to keep curating. Default for fresh entries. |
| `staged` | Operator confirmed it's worth keeping in the pipeline. Tier 2 sections may begin to fill in. |
| `evaluating` | Operator (or a delegated team) is actively assessing for promotion to swarm-manager / CATALOG / elsewhere. Tier 2 sections fill in here in earnest. |
| `promoted` | Idea graduated to its target home (swarm-manager backlog item, CATALOG candidate, narrative doc, etc.). The idea-pipeline entry remains as breadcrumb pointing to the new home. After ~6 months, can be pruned. |
| `retired` | Idea is no longer worth pursuing. Reason captured in the entry. Stays in-folder for ~6 months for audit, then optionally moves to a `_retired/` subdirectory. |

Status transitions are operator decisions, not agent decisions.

## Promotion paths (idea-pipeline → elsewhere)

When an idea graduates from idea-pipeline, **information must transfer** to the new home. The idea-pipeline entry's content is the seed for the destination's content.

| Destination | When | Information transfer |
|---|---|---|
| `path:scenarios/swarm-manager/ideas/<slug>/` | Capacity opens up; idea is well-shaped and ready for engineering pipeline | Tier 1 fields seed `spec.json` (name, title, description). Tier 2 sections seed `PRD.md`, `README.md`, and inform workshop rounds. The idea-pipeline entry is updated to `status: promoted` with the destination path. |
| `path:docs/monetization/catalogs/CATALOG.md` candidate | SKU-shaped + revisit trigger fires (or is judged imminent) | Tier 1 + Monetization framing seed the candidate file under `path:docs/monetization/catalogs/skus/base/` or `path:docs/monetization/catalogs/skus/addons/`. opportunity-scout typically does this via `catalog-promotion` decision. |
| `path:docs/monetization/catalogs/revenue-lines/<slug>.md` reference | Idea matures into settled reference material (a playbook, an architecture, a competitive analysis) | The idea-pipeline content moves into the revenue-line file as a reference section. Idea-pipeline entry is set to `status: promoted` with destination. |
| `path:docs/strategy/<file>.md` | Idea is a strategic framing rather than an executable thing (e.g., a positioning principle, a long-term capability flag) | The relevant strategy doc absorbs the framing. |

The promotion is a one-way move; idea-pipeline doesn't dual-track with the destination. Once promoted, the destination owns the idea.

## Retirement path (with explicit cadence)

Retired entries stay in-folder with `status: retired` + retirement reason + `retired_at` date. They serve as audit trail — "this idea was considered and explicitly declined for these reasons."

**Cadence:** every **4 vision walks**, the operator scans the active idea-pipeline entries during a walk's Phase 6 or Phase 8 for stale ones — entries that have sat in `raw` or `staged` status for ≥4 walks without movement. Each stale entry gets a decision: graduate, retire, or explicitly defer with a fresh review trigger. **No silent decay.** Retired entries older than ~6 months may be moved to a `_retired/` subdirectory or pruned, operator's call.

## Responsible-agent table

| Stage | Who proposes | Who decides | Who writes |
|---|---|---|---|
| Capture (creation of new entry) | Any source listed above | Operator | Operator |
| Tier 2 deepening | Operator OR a delegated team member via decision | Operator | Whoever is delegated; operator final review |
| Promotion to swarm-manager / CATALOG | Any relevant team via decision | Operator | Operator (or delegated) |
| Status changes | Operator OR via decision | Operator | Operator |
| Retirement | Any source via decision | Operator | Operator |

The pattern: agents propose via decisions; operator decides; operator (or delegated) writes. This preserves the operator-curated property.

## Freshness re-validation

This README's decision tree and source list **age out** as the project's surfaces evolve (new ingestion sources emerge; team members are added/removed; CATALOG conventions change). Without re-validation, idea-pipeline routing decays.

**Re-validation hook:** during each vision walk's Phase 9 wrap-up, if any *new ingestion source* surfaced during the walk (e.g., a new team's heartbeat, a new external content stream), the operator notes it as a Phase 8 follow-on action: review and update this README's source table accordingly. One-line addition to the wrap-up checklist; cheap insurance against the routing convention rotting.

## Relationship to existing surfaces

Idea-pipeline does not replace any existing idea-storage surface. It *complements* them. To prevent confusion:

| Surface | Owner | Shape it serves | Why it's not idea-pipeline |
|---|---|---|---|
| `monetization` knowledge entries under `topic[example]:monetization/opportunity/<slug>` | opportunity-scout (agent) | SKU-shaped monetization candidates with rich framing (TAM, effort, revisit triggers, acquisition + retention hypotheses) | Agent-side raw pool, not operator-curated; SKU-shaped only; capacity-deferral handled internally via revisit triggers. **Parallel pool for SKU-shaped ideas, not competing.** |
| `path:docs/monetization/catalogs/CATALOG.md` + `path:docs/monetization/catalogs/skus/base/*.md` + `path:docs/monetization/catalogs/skus/addons/*.md` | catalog-strategist proposes → operator approves | SKU lifecycle (`idea → candidate → trigger-met → active → shipped → retired`) | SKU-scope, not scenario-scope. Idea-pipeline can reference CATALOG SKUs in its monetization framing. |
| `path:docs/monetization/catalogs/revenue-lines/*.md` | monetization team | Revenue-stream registry with productization targets, legal surfaces, candidate playbooks | Settled reference material, not idea-staging. Reference-architectures can be cited from idea-pipeline entries. |
| `path:scenarios/swarm-manager/ideas/<slug>/` | swarm-manager (engineering pipeline) | Scenario hypotheses ready for workshop loop | Pulls into development pipeline / director-team prioritization. Idea-pipeline graduates here when capacity opens. |
| `path:docs/strategy/*.md` (other files) | operator-curated | Long-form durable strategic framing (context, decisions, risks, roadmap, business-solutions, capability-flags) | Strategic positioning, not idea-staging. Idea-pipeline entries reference these for goal-alignment context. |
| `*/notebook/*.md` (per-team) | Team members append-anyone | Team-internal craft observations, pre-promotion to plan-of-record | Team-internal working memory, not project-wide ideas. Notebook entries promote to *that team's* plan-of-record, not to idea-pipeline. |
| Vision-walk `last-handoff.md` Phase-8 deferred actions | walk artifact | Ephemeral list of post-walk actions | Walk-internal; cleaned at Phase 9. Idea-pipeline entries created during a walk are listed here only as the action that creates them. |

## Folder convention

Each idea is a folder, slug-named (kebab-case, matches `name` frontmatter):

```
docs/strategy/idea-pipeline/
├── README.md                                           # this file
├── _template.md                                        # starter shape for new entries
├── <slug-1>/
│   └── README.md                                       # Tier 1 + (optional) Tier 2 sections inline
├── <slug-2>/
│   ├── README.md                                       # Tier 1 + (optional) Tier 2 sections
│   └── monetization-framing.md                         # split when Monetization framing > ~500 words
└── ...
```

A single `README.md` per idea is enough until a Tier 2 section grows past ~500 words. At that point, the operator may split the section into its own file. Splits happen lazily when content demands; never preemptively.

## Cross-references

- [`../README.md`](../README.md) — strategy folder index.
- [`../context.md`](../context.md), [`../roadmap.md`](../roadmap.md), [`../business-solutions.md`](../business-solutions.md) — durable framing docs that idea-pipeline entries reference for goal alignment.
- [`../../monetization/catalogs/CATALOG.md`](../../monetization/catalogs/CATALOG.md) — SKU index; cited from monetization framing sections.
- [`../../monetization/catalogs/revenue-lines/`](../../monetization/catalogs/revenue-lines/) — settled revenue-stream reference material.
- `path:scenarios/swarm-manager/ideas/` — engineering pipeline staging; idea-pipeline graduates here.
- `monetization` team knowledge entries under `topic[example]:monetization/opportunity/<slug>` — opportunity-scout's parallel SKU-shaped raw pool. List with `prompt-manager team knowledge-list monetization --topic-prefix=monetization/opportunity/`.

## Changelog

- **2026-04-28** — Initial v1. Authored during walk #5 divergence #5 after first-principles audit of the project's six existing idea-storage surfaces surfaced a real gap for operator-curated, capacity-deferred, project-wide ideas.
