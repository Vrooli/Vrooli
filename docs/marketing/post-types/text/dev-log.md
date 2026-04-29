# Post Type: Dev Log

**Status:** Extracted from `STRATEGY.md`'s Content-type principles + Dev-log narrative principles on 2026-04-28 (walk #5 divergence #3, Action B). Original Dev-log narrative principles added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

**Paired skill:** [`x-dev-log`](../../../../scenarios/prompt-manager/store/skills/packs/core/x-dev-log/)
**Primary author:** `oss-advertiser`
**Notebook home for emerging craft patterns:** [`docs/marketing/notebook/DEV_LOG_CRAFT.md`](../../notebook/DEV_LOG_CRAFT.md) — already populated; promotion target is `x-dev-log` skill edits.

## Purpose

A dev log is **project-wide progress narrative in builder voice**. It tells the audience what's been built since the last post, why each change matters, what failed, and what's coming next. The protagonist is the operator-and-agents *building Vrooli*; the protagonist is not any specific scenario as a product (that's `scenario-spotlight`) and not Vrooli as a developer platform (that's `oss-framework`, planned).

It is **not**:
- A scenario-spotlight (those pitch one scenario as an end-user tool — see [`scenario-spotlight.md`](scenario-spotlight.md)).
- An OSS-framework post (those pitch Vrooli as a developer platform — see `oss-framework.md` when authored).
- A feature announcement (those are shipped-features-only with imminent-release dates — see `STRATEGY.md`'s Content-type principles pointer).

## Audience

Primary: the **OSS / contributor** persona (see [`AUDIENCES.md`](../../AUDIENCES.md) for the full definition). Specifically:
- Developers and curious followers interested in *how* Vrooli is built — the agents-as-builders story is the unique signal.
- Existing contributors and would-be contributors orienting on the project's current state.
- Other AI-builder communities watching how a multi-agent system actually evolves week-to-week.

Secondary: the **subscription buyer** persona, when the dev log surfaces an end-user-relevant capability shipping. The framing stays builder-voice; subscription audience reads as a quiet validation rather than a pitch.

## Conversion goal

- Awareness — the audience knows the project moved this week.
- Follower retention — the audience returns next week.
- Contributor pipeline — over time, some readers move from follower → forker → contributor.

Dev logs are **not** a sign-up funnel. Treat each post's success as "did the audience read it, and would they read the next one." Sign-up is `scenario-spotlight`'s job.

## Structure

Apply the cross-cutting [essay-shape technique](../../post-techniques/essay-shape.md) — hook → introduction → body → conclusion. With the cross-cutting [hook-vs-body-asymmetry technique](../../post-techniques/hook-vs-body-asymmetry.md) on length distribution.

Specific to dev-logs:
- **Range:** 3-7 tweets per X thread is a typical range, *not* a structural cap. Body tweets may exceed 280 chars per the asymmetry technique. Blog version: 500-2000 words.
- **Work-in-progress is explicitly labeled.** "This is shipping — here's what works and what's still rough" is honest builder voice; "we're proud to launch X" is voice drift.
- **Inter-post linkage** ([technique](../../post-techniques/inter-post-linkage.md)): each post after the first cites the prior post's URL via `previous_post_url` in `marketing-crew/shared/publish-log.jsonl`.

## Voice (dev-log-specific application)

The general voice canon lives in [`STRATEGY.md`](../../STRATEGY.md). Dev-logs apply it with an additional discipline:

**Personal voice grounded in builder identity.** Voice canon prescribes "first person, conversational, technically credible." Embodiment matters — a sentence that names an agent in third person is *not yet* personal voice. The first post of any series must especially feel like a software engineer talking about what they built — real grounded excitement tied to real work. The "I" can be the operator's or an agent's; pick deliberately, but pick.

**Agents as protagonists** (from voice canon): name the specific agent (`run-introspector`, `oss-advertiser`, `toolchain-validator`, etc.), let it be the subject of the sentence, and credit the work to it. "team-agent-optimizer noticed the false-positive pattern and proposed the gate" lands harder than "I noticed it and added the gate." Operator authorship still happens — for vision, strategic decisions, and direct critique — but agent authorship is the unique signal no SaaS competitor can copy.

## What → why framing

Every change shown carries its own reason-to-care.
- **What:** "`resolve.go` is 52 lines + 202 test lines."
- **Why:** "`resolve.go` lets an initiative dispatch work to whichever agent the scenario expects, instead of hard-coding the wire — so adding a new scenario no longer means editing the dispatcher."

The why connects to broader narrative: last post's setup, next post's payoff, the project's vision. **Show only changes that actually matter; do not narrate every commit.**

The mechanism: use `marketing-crew/shared/published-improvements-log.jsonl` to track what's already been narrated per scenario, so the next post advances the story rather than repeating it.

The corresponding contrarian failure mode is **what-without-why** (mode 12 in `STRATEGY.md`'s Anti-patterns).

## Asset requirements

Dev-logs are **text-led**, not asset-led (in contrast to `scenario-spotlight.md`'s asset-led shape). When an asset appears, it is supporting evidence for the narrative — a diff fragment, a screenshot of a graph view, an ASCII tree of the new package layout. Brand consistency with [`ASSETS.md`](../../ASSETS.md) and [`IMAGE_STYLE.md`](../../IMAGE_STYLE.md) still applies; sanitization per [`x-dev-log`](../../../../scenarios/prompt-manager/store/skills/packs/core/x-dev-log/) guardrails is non-negotiable.

When a dev-log post would benefit from a longer asset (e.g., a complete demo of a shipped feature), consider whether the right surface is actually a paired `scenario-spotlight` post linked from the dev-log conclusion, rather than embedding the full asset in the dev-log itself.

## Contrarian failure modes (dev-log-specific specializations)

The `marketing-contrarian` member skill ingests this section as type-level review rules. Each failure mode below maps to a framework-level mode in `STRATEGY.md`'s Anti-patterns; type-level modes are *specializations*, not new framework modes (per [`marketing-contrarian/RESPONSIBILITIES.md`](../../../../scenarios/prompt-manager/store/teams/marketing-crew/members/marketing-contrarian/RESPONSIBILITIES.md)).

| Type-level failure mode | What it looks like in a dev-log | Framework-level mode |
|---|---|---|
| **Narrative-flatness** | The post reads as a changelog or atomic-tweet list rather than essay-shape (hook → intro → body → conclusion). | mode 9 (narrative-flatness) |
| **What-without-why** | The post lists changes / line counts / commit refs without why-it-mattered framing tied to broader narrative. | mode 12 (what-without-why) |
| **Missing-introduction-on-first-mention** | The post refers to a scenario / agent / named file by name with no prior mention in `published-scenario-mentions.jsonl` for the target audience AND no introduction in the draft itself. | mode 11 |
| **Internal-vocabulary leakage** | Published copy uses internal artifact names (e.g., `p8`, `round-002`, batch ids) without translation. | mode 10 |
| **Hype drift in feature claims** | Overpromising shipped scope; "soon" without a committed date; claims not verifiable in `docs/monetization/catalog/base/*.md` or in shipped commits. | mode 1 (hype drift) |
| **Voice drift to corporate-marketer language** | "We're excited to announce," "supercharge," "transform," "unlock" replacing builder voice. | mode 2 (voice drift) |
| **Engagement-metric hallucination** | Numbers without honesty flags (`measured / estimate / aspirational / pending-telemetry`). | mode 3 (hallucinated engagement metrics) |
| **Capability-workaround without gap** | The post narrates a manual workaround the operator is doing, with no matching `capability-gap` decision and no notebook entry. | mode 8 |

Honesty flags the publisher / oss-advertiser member must attach to a dev-log draft:

- `feature_claims=measured | overclaimed | uncertain` — measured = every claim cross-checked against shipped commits and PRDs.
- `engagement=measured | pending-telemetry | aspirational | estimate` — for any numeric claim about engagement / growth / usage.
- `data_source=complete | incomplete-data:<source-unavailable>` — when a draft was authored against a degraded data source (e.g., agent-manager unavailable), flag which leg of the data-source matrix was missing. See `notebook/DEV_LOG_CRAFT.md` for the canonical example.

## Cross-cutting techniques this type uses

(All have canonical homes under [`../../post-techniques/`](../../post-techniques/).)

- [Essay-shape per post](../../post-techniques/essay-shape.md)
- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md)
- [Intro-on-first-mention](../../post-techniques/intro-on-first-mention.md) (with `published-scenario-mentions.jsonl` lookup before assuming familiarity; subject-familiarity corollary applies to hook calibration)
- [Inter-post linkage](../../post-techniques/inter-post-linkage.md)
- [No internal numbering externally](../../post-techniques/no-internal-numbering-externally.md)

## Where this fits in the marketing flow

```
                    ┌──────────────────────────────────────┐
                    │ oss-advertiser produces              │
                    │ dev-log draft via x-dev-log skill,   │
                    │ mining git-control-tower +           │
                    │ agent-manager + swarm-manager +      │
                    │ app-issue-tracker for the period     │
                    └────────────────┬─────────────────────┘
                                     │
                     ┌───────────────▼──────────────────┐
                     │ marketing-contrarian reviews     │
                     │ against framework-level + type-  │
                     │ level failure modes (this file)  │
                     └───────────────┬──────────────────┘
                                     │
                                     ▼
                          ┌────────────────────────┐
                          │ publisher proposes     │
                          │ content-publish-       │
                          │ proposal decision      │
                          └──────────┬─────────────┘
                                     │
                                     ▼
                            operator decides at vision walk
                                     │
                                     ▼
                            operator manually posts;
                            operator pastes URL back into
                            publish-log (until social-
                            media-scheduler ships)
```

## Promotion path for craft observations

Patterns observed during production runs of `x-dev-log` are appended to [`docs/marketing/notebook/DEV_LOG_CRAFT.md`](../../notebook/DEV_LOG_CRAFT.md). Promotion targets per [team-shared-docs-design](../../../../scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md):

- **Skill edit** — most observations land as edits to the `x-dev-log` skill (mining-strategy rubric, interestingness-scoring weights, output-contract refinements). Primary promotion target.
- **This file** — observations that change strategic canon (audience model, conversion goal, new failure mode, type-specific voice rule).
- **A `post-techniques/` file** — observations that turn out cross-cutting and apply to other post types.

`brand-manager` curates the promotion path via `notebook-promotion` decisions.

## Cross-references

- Paired skill: [`x-dev-log`](../../../../scenarios/prompt-manager/store/skills/packs/core/x-dev-log/) (in prompt-manager core skills).
- Plan-of-record neighbors: [`../../STRATEGY.md`](../../STRATEGY.md) (voice canon, voice samples — Sample 5 specifically illustrates the first-post intro burden), [`../../AUDIENCES.md`](../../AUDIENCES.md), [`../../CHANNELS.md`](../../CHANNELS.md), [`../../ASSETS.md`](../../ASSETS.md).
- Notebook: [`../../notebook/DEV_LOG_CRAFT.md`](../../notebook/DEV_LOG_CRAFT.md) — populated; promotes mostly to `x-dev-log` skill.
- Sibling post type: [`scenario-spotlight.md`](scenario-spotlight.md) — for contrast (asset-led, product-focused, conversion-funnel).
- Persistence surfaces: `marketing-crew/shared/publish-log.jsonl` (URL paste-back), `marketing-crew/shared/published-scenario-mentions.jsonl` (familiarity tracking), `marketing-crew/shared/published-improvements-log.jsonl` (what-→-why de-duplication).

## Changelog

- **2026-04-28** — Extracted from `STRATEGY.md` during walk #5 divergence #3 (Action B). Move-not-rewrite; content preserved verbatim with light adaptation for the per-entity-file shape. Original Dev-log narrative principles dated 2026-04-27 (framework-update `dec-1777300532504756717`).
