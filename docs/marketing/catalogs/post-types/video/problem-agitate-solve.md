# Post Type: Problem-Agitate-Solve

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-problem-agitate-solve` *(planned)*
**Primary lane/member:** Usually subscription lane (`subscription-advertiser`)
**Notebook home for emerging craft patterns:** `../../../notebook/PROBLEM_AGITATE_SOLVE_CRAFT.md` (created on first observation)

## Purpose

A problem-agitate-solve (PAS) video is the classic 3-act ad structure: **identify a friction** → **escalate the friction** → **present the solution**. Aggressively conversion-shaped. Not the right format for awareness or relationship-building; right for moments when the audience knows the friction and is ready to consider a tool.

Distinct from `narrative-talking-head.md` (story-shaped) and `day-in-life-ugc.md` (observational). PAS is explicitly persuasive shape.

## Audience

Primary: lifestyle and dev-tool bundle audiences on TikTok / Reels / Shorts who recognize the friction immediately.

## Conversion goal

- **Click-through to link-in-bio** with explicit CTA.
- PAS is the format with the strongest direct-conversion intent; works only when the friction is real and recognized.

## Structure

- **Problem (0-3s).** Friction shown or stated. "Have you ever spent the whole evening figuring out what to cook?" "Tired of forgetting to follow up with leads?" Hook on a recognized pain.
- **Agitate (3-12s).** Show the cost of the friction continuing: missed deadlines, wasted time, frustration, lost opportunity. Don't moralize; show consequences.
- **Solve (12-25s).** Present the Vrooli scenario as the answer. Demonstrate (in persona-voice or brand-voice) how it eliminates the friction.
- **Closer (25-35s).** CTA. "Link in bio." One verb.

Length: 25-40s typical.

## Asset requirements

- **Multi-shot consistency** across problem → agitate → solve cuts.
- **Demonstration shot** of Vrooli scenario in use during the solve phase. Real screen recording (via BAS) preferred over static product imagery; readers detect mockups.
- **Vertical 9:16, captions, disclosure label** as standard.

## Voice

Persona voice or brand voice both work. PAS in persona voice carries higher conversion when the persona is in the audience's lived context. Brand voice works when the friction is technical and the demonstration is the focal point.

## Contrarian failure modes

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Manufactured-friction** | The "problem" is invented or marginal; audience doesn't recognize it as their pain. | mode 1 (hype-drift) variation — overclaim of audience pain. |
| **Catastrophizing** | Agitate phase exaggerates consequences beyond believability; reads as fear-marketing. | mode 2 (voice drift). |
| **Demo-theater** | Solve phase shows an idealized Vrooli demo that hides setup / failure modes. | scenario-spotlight's "demo theater" specialization (mode 1). |
| **Capability-inflation** | Solve phase claims Vrooli does things it doesn't yet do. | mode 1. |
| **Persona-disclosure-violation** | When persona-voice. | mode 13. |
| **Regulated-domain-friction** | The friction itself is in medical / financial / legal domain. | mode 18. |

Honesty flags:

- `friction_authenticity=verified-from-research | speculative` — `verified-from-research` requires the friction to appear in researcher's `audience-scans.jsonl` or in audience interviews; speculative frictions are flagged for operator review.
- `feature_claims=measured` for the solve phase.
- Plus the standard AI-UGC flags when persona-voice.

## Cross-references

- [`../../../strategy/patterns/ai-ugc-personas.md`](../../../strategy/patterns/ai-ugc-personas.md).
- [`../../rich-media/`](../../rich-media/).
- Sibling: [`narrative-talking-head.md`](narrative-talking-head.md), [`comparison-reel.md`](comparison-reel.md).
- [`../text/scenario-spotlight.md`](../text/scenario-spotlight.md) — the long-form sibling using the same conversion impulse.
