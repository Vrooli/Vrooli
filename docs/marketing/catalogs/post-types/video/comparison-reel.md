# Post Type: Comparison Reel

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-comparison-reel` *(planned)*
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for subscription bundles; OSS lane (`oss-advertiser`) for OSS framework comparisons.
**Craft observation topic:** `marketing-craft-observation/comparison-reel/<slug>`

## Purpose

A comparison reel is **a short video framing a Vrooli scenario or capability against a named alternative the audience already knows**, leading with a concrete differentiator. Two common shapes: side-by-side (split-screen showing both at once) and before/after (life-with-old-tool vs life-with-Vrooli). Comparison-shaped video is high-conversion when the differentiator is genuine and high-friction when it isn't.

The video application of [`../../../methods/post-techniques/competitive-comparison.md`](../../../methods/post-techniques/competitive-comparison.md) — that technique's rules apply throughout.

## Audience

Primary: dev-tool audience comparing Vrooli scenarios against specific named tools; subscription-bundle audience comparing Vrooli against general-purpose alternatives.

## Conversion goal

- **Click-through** to landing page or scenario.
- **Save / share** when the comparison gives the audience a concrete argument they can use.

## Structure

- **Hook (0-2s).** Names the alternative + the axis of comparison. "Here's why I switched from [Named Alternative] to [Vrooli scenario] for [specific use]." No mystery; the comparison is the entire point.
- **Demo (2-25s).** Side-by-side or before/after. Show the workflow in both. Be specific about conditions — same input, same task, real measurement.
- **Acknowledgment (25-30s).** What the alternative still does well. (Required per `competitive-comparison.md` — comparisons that paint the alternative as wholly inferior read as hostile.)
- **Closer (30-35s).** Differentiator restated; CTA.

Length: 30-45s typical.

## Asset requirements

- **Real demos in both halves of the comparison.** Mockups read as theater and the contrarian rejects (`unfair-comparison`).
- **Cited benchmark** when a multiplier is claimed (e.g., "20x faster"). One-click-reachable benchmark methodology, conditions, version pinning. Per `competitive-comparison.md`'s hyperbolic-but-verifiable-multipliers sub-pattern.
- **Vertical 9:16, captions burned in.**
- **Disclosure label** if any persona or AI-generated face appears.

## Voice

Brand voice typically. Persona voice possible but rare — comparison voice carries authority better in brand-as-builder mode.

## Contrarian failure modes (comparison-reel-specific specializations)

Inherits all the failure modes from [`../../../methods/post-techniques/competitive-comparison.md`](../../../methods/post-techniques/competitive-comparison.md) — uncited multiplier, unfair comparison, comparison without genuine differentiator, strawman alternative, hostile tone, stale comparison. Reapplied at video specifically:

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Visual-strawman** | Side-by-side renders the alternative in a deliberately unflattering visual setup (laggy mock UI, deliberately bad fonts). | `competitive-comparison.md` strawman + visual-rendering layer. |
| **Cherry-picked-condition** | Demo conditions favor Vrooli (best-case path) vs alternative (worst-case path). | unfair comparison. |
| **Stale-version** | Demo of the alternative uses an old version. | stale comparison. |
| **Caption-overclaim-where-demo-doesn't-support** | Caption says "20x faster" but the on-screen demo doesn't show a benchmark to support it. | uncited multiplier + mode 1. |
| **Persona-disclosure-violation** | If persona-voiced. | mode 13. |

Honesty flags (extending `competitive-comparison.md`'s):

- `comparison_basis=measured | partial | aspirational`.
- `alternative_version_pinned=YES | NO` — must be YES with date.
- `feature_claims=measured`.
- `multiplier_cited=YES | NO | n/a` — when the post leads with a multiplier.

## Cross-cutting techniques

- [Competitive comparison](../../../methods/post-techniques/competitive-comparison.md) — the canonical home; this video format applies its rules.
- [Hook-vs-body length asymmetry](../../../methods/post-techniques/hook-vs-body-asymmetry.md).
- [Recommendation framing](../../../methods/post-techniques/recommendation-framing.md) — when comparison is third-party-attributed.

## Cross-references

- [`../../../methods/post-techniques/competitive-comparison.md`](../../../methods/post-techniques/competitive-comparison.md) — required reading.
- [`demo-recording.md`](demo-recording.md) — sibling brand-voice format without comparison framing.
- [`../text/scenario-spotlight.md`](../text/scenario-spotlight.md) — text-led sibling supports a Comparison variant in its conversion-rate-friendly variants list.
