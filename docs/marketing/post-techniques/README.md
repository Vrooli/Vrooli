# Marketing Post Techniques — Plan of Record

This folder is the **strategic canon** for cross-cutting voice and structure techniques shared across multiple post types. One file per technique.

These docs answer: *what is this technique, when does it apply, when does it backfire, what's the failure mode the marketing-contrarian watches for?*

They do **not** answer: *how does any specific post type apply this technique end-to-end.* That belongs in the per-type file under [`../post-types/`](../post-types/) which references the technique here.

## Why a separate folder for techniques

Many marketing patterns are not specific to one post type. *Hook-vs-body asymmetry* applies to dev-logs, scenario-spotlights, and oss-framework posts equally. *Intro-on-first-mention* applies any time a post references a not-yet-introduced internal name. Putting these in a single post type's file misrepresents their scope; duplicating across post-type files invites drift. The technique gets one canonical home and is referenced from each post-type file that uses it.

This mirrors the [`team-shared-docs-design`](../../../scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md) per-entity guidance: each technique is its own entity.

## Files in this folder

| File | Status | Applies to |
|------|--------|-----------|
| [`essay-shape.md`](essay-shape.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28) | dev-log, scenario-spotlight, oss-framework, blog posts |
| [`hook-vs-body-asymmetry.md`](hook-vs-body-asymmetry.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28) | All post types |
| [`intro-on-first-mention.md`](intro-on-first-mention.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28) | All post types referencing internal names |
| [`inter-post-linkage.md`](inter-post-linkage.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28; cross-platform amplification subsection added 2026-04-28 walk #5 divergence #4) | All series-shaped post types; cross-platform amplification applies to any post with paired surfaces |
| [`no-internal-numbering-externally.md`](no-internal-numbering-externally.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28) | All post types |
| [`recommendation-framing.md`](recommendation-framing.md) | v1 (authored 2026-04-28 walk #5 divergence #4 from observed Post 2 / MapiLeads-style framing + prior `scenario-spotlight.md` Recommendation variant) | All post types where a genuine third-party basis exists |
| [`competitive-comparison.md`](competitive-comparison.md) | v1 (authored 2026-04-28 walk #5 divergence #4 from observed Post 3 / jcode-vs-Claude-Code framing) | `oss-framework` (planned), `scenario-spotlight`, occasional `dev-log` |

## Write rules

Same as the rest of `docs/marketing/`: agents never write directly; operator-curated via approved decisions. Use `brand-guideline-update` for edits unless the technique is platform-specific.

## Cross-references

- [`../post-types/`](../post-types/) — per-post-type strategic canon. Each post-type file lists which techniques here it depends on.
- [`../STRATEGY.md`](../STRATEGY.md) — voice canon. Several techniques currently still live as subsections there; will migrate via Action B.
- [`../notebook/AUDIENCE_OBSERVATIONS.md`](../notebook/AUDIENCE_OBSERVATIONS.md) — where new technique observations are appended before promotion to this folder.
