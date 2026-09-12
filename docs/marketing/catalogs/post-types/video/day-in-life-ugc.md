# Post Type: Day-in-Life UGC

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-day-in-life-ugc` *(planned)*
**Primary lane/member:** `producer` — usually subscription lane
**Craft observation topic:** `marketing-craft-observation/day-in-life-ugc/<slug>`

## Purpose

A day-in-life UGC video is a **persona-actor moving through a routine**, with Vrooli appearing as one element in the routine rather than as the focus. The reader is not being sold to — they're observing a lifestyle that happens to use Vrooli. The product placement is the conversion mechanism; the routine is the cover.

Distinct from `narrative-talking-head.md`: that's persona speaking *to* camera; this is persona living *in front of* camera.

## Audience

Primary: lifestyle-bundle audience on TikTok and Reels. Persona-actor account.

Specific personas: "busy parent's morning routine", "homelab tinkerer's evening", "indie maker's writing session", "older person's cleaning routine".

## Conversion goal

- **Bio-tap rate** is the proximate signal. Day-in-life formats convert by readers tapping the persona's profile to learn more, then tapping link-in-bio.
- **Save / share** as secondary signal — a routine that captures lifestyle aspiration gets saved.
- Click-through and sign-up are downstream.

## Structure

- **Cold open (0-2s).** Persona mid-action. No intro card. The reader is dropped into the moment.
- **Routine sequence (2-30s).** Persona moves through ~3-5 routine moments — making coffee, prepping the workspace, opening tools, completing a task, winding down. Vrooli appears in one of these moments naturally (using the scenario on the laptop, looking at the schedule on the phone).
- **Closer (30-40s).** Routine ends; persona looks satisfied/calm/in-control. Caption + link-in-bio carry the explicit CTA.

Length: 30-45s typical.

## Asset requirements

- **Persona character + scene JSON** at [`../../rich-media/characters/`](../../rich-media/characters/) and [`../../rich-media/scenes/`](../../rich-media/scenes/) — multi-shot consistency across the routine sequence is the key production challenge. Character must remain recognizably the same across cuts; scene transitions need to flow.
- **Product placement asset** — Vrooli scenario screen (UI, mobile app, terminal) appears in one shot. Assets in [`../../rich-media/products/`](../../rich-media/products/) define on-screen content rules.
- **Vertical 9:16.**
- **Captions burned in.**
- **Disclosure label.**

## Voice

Often near-silent — ambient sound, captions for context. When voice present, persona voice rules apply (see `ai-ugc-personas.md`).

## Contrarian failure modes

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Asset-style-drift-across-shots** | Character or scene visually drifts across cuts. | mode 9 specialization. |
| **Product-placement-over-rendering** | Vrooli appears in too many shots; reads as ad rather than routine. | mode 1 (hype-drift). |
| **Aspirational-routine-implausibility** | Routine is so stylized it reads as fake-influencer content rather than relatable life. | mode 2 (voice drift) — applied to lifestyle-tone drift. |
| **Persona-disclosure-violation** | Substantial AI-persona content unlabeled. | mode 13. |
| **Real-person-impersonation** | Persona resembles a specific identifiable real person. | mode 15. |
| **Fabricated-real-customer-testimonial** | Persona claims to be a real Vrooli customer in caption / closer. | mode 16. |

Honesty flags: same as `narrative-talking-head.md`.

## Cross-references

- [`../../../strategy/patterns/ai-ugc-personas.md`](../../../strategy/patterns/ai-ugc-personas.md).
- [`../../rich-media/`](../../rich-media/).
- Sibling: [`narrative-talking-head.md`](narrative-talking-head.md), [`problem-agitate-solve.md`](problem-agitate-solve.md).
