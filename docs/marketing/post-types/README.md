# Marketing Post Types — Plan of Record

This folder is the **strategic canon** for each kind of marketing post Vrooli produces. One file per post type.

These docs answer: *what is this type for, who is it aimed at, what is the conversion goal, what does a good one look like, what failure modes does the marketing-contrarian watch for?*

They do **not** answer: *here is the prompt, here is the round structure, here are the data sources to mine.* That is the role of the per-type **skill** under [`scenarios/prompt-manager/store/skills/packs/core/x-<type>/`](../../../scenarios/prompt-manager/store/skills/packs/core/) (e.g., `x-dev-log`, `x-scenario-spotlight`). The skill is the *executable spec* an agent runs; the file in this folder is the *strategic canon* the operator reads to make decisions and the agent reads as required-reading context.

## Why per-entity files (not subsections of STRATEGY.md)

Per the [`team-shared-docs-design`](../../../scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md) skill: monolithic plan-of-record docs invite accidental damage when adding/retiring one entity. Per-entity files keep edits scoped and let each post type evolve at its own pace. Cross-cutting structural rules (essay-shape, hook-vs-body asymmetry, intro-on-first-mention, inter-post linkage) live in [`../post-techniques/`](../post-techniques/) and are referenced from each post-type file rather than duplicated.

> **Migration note:** dev-log content was originally part of `STRATEGY.md` and was extracted into `post-types/dev-log.md` on 2026-04-28 (walk #5 divergence #3, Action B). `STRATEGY.md` retains voice canon, voice samples, dual-audience framing, anti-patterns, and cross-references; per-content-type and cross-cutting-technique content now lives here and in `../post-techniques/` respectively.

## Decision tree — which post type when?

Ask the questions in order. The first "yes" picks the type.

1. **Is the operator trying to show project-wide progress / building-in-public narrative?** → `dev-log`
2. **Is the operator trying to get a specific person to use, sign up for, or buy a single Vrooli scenario as an app/product?** → `scenario-spotlight`
3. **Is the operator trying to get developers to adopt Vrooli as an OSS framework for building agentic apps / app ecosystems?** → `oss-framework` *(planned; not yet authored)*
4. **None of the above** → likely doesn't need a structured post type. Capture as a one-off and let the marketing-crew researcher / publisher decide.

The dimensions that drive the choice:

| Dimension | dev-log | scenario-spotlight | oss-framework |
|-----------|---------|-------------------|---------------|
| Subject | The project as a whole | One scenario as an app/product | Vrooli as an OSS platform |
| Primary audience | OSS contributors + curious followers | Subscription buyers / end-users | Developers building on Vrooli |
| Conversion goal | Awareness, follower retention, contributor pipeline | Sign-up / subscription / app adoption | GitHub stars, contributor signups, builds-on-Vrooli |
| Asset weight | Light (text-first, occasional code/diff) | Heavy (screen recordings, screenshots, demo) | Medium (architectural diagrams, code snippets) |
| Voice | Personal builder narrative | Product-focused, demo-led | Technical, framework-oriented |
| Primary skill | `x-dev-log` | `x-scenario-spotlight` | `x-oss-framework` *(future)* |
| Primary member | `oss-advertiser` | `subscription-advertiser` | `oss-advertiser` *(may split out)* |

## Cross-cutting techniques (shared across types)

Each post-type file references the techniques it depends on. The canonical home for each technique is [`../post-techniques/<name>.md`](../post-techniques/). Examples (some currently still inside `STRATEGY.md`, pending Action B extraction):

- Essay-shape per post (hook → intro → body → conclusion)
- Hook-vs-body length asymmetry
- Intro-on-first-mention (with `shared/published-scenario-mentions.jsonl` lookup)
- Inter-post linkage
- No internal numbering externally
- Recommendation framing *(third-party voice, applies when genuine third-party basis exists)*

## Files in this folder

| File | Status | Primary skill | Primary member |
|------|--------|---------------|----------------|
| [`dev-log.md`](dev-log.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28) | `x-dev-log` | `oss-advertiser` |
| [`scenario-spotlight.md`](scenario-spotlight.md) | v1 | `x-scenario-spotlight` | `subscription-advertiser` |
| `oss-framework.md` | *future — pending the third reference post* | `x-oss-framework` *(future)* | `oss-advertiser` |

## Write rules

Same as the rest of `docs/marketing/`: agents never write directly; operator-curated via approved decisions (`brand-guideline-update` is the canonical context for edits to per-type files; type-specific tweaks may use `channel-update` if the change is platform-specific rather than type-strategic).

When proposing a new post-type file, the proposal must include:

1. The strategic canon content (purpose, audience, conversion goal, asset requirements, contrarian failure modes).
2. The paired `x-<type>` skill stub (the doc must have a real consumer or it's a stale shrine).
3. The member-skill required-reading wiring (which marketing-crew member's skill picks this up).

## Cross-references

- [`../STRATEGY.md`](../STRATEGY.md) — voice canon, dual-audience framing, anti-patterns. Currently also holds dev-log narrative principles and content-type principles; both will migrate here in Action B.
- [`../post-techniques/`](../post-techniques/) — cross-cutting voice and structure techniques shared by multiple post types.
- [`../notebook/`](../notebook/) — working notebook for patterns observed in production. Each post-type file's promotion target may be either *this folder* (strategic refinements) or its paired skill (executable refinements).
