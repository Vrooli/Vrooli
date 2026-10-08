# Narrative — Project Identity Canon

This folder is the **plan-of-record** for Vrooli's project identity: the canonical pitch, story, FAQ, press kit, and pitch-deck outline that every team consumes. Unlike `path:docs/marketing/` (which holds *marketing canon* — voice, audiences, channels, campaigns), the docs in here are *project-identity artifacts* — the canonical answers to "what is Vrooli, and why?"

Cross-team consumers:
- `marketing-crew` advertisers pull from PITCH and NARRATIVE before drafting.
- `monetization` team references PITCH and NARRATIVE when authoring landing-page copy.
- `director-swarm` reads NARRATIVE deepest-layer for vision-arc alignment in vision walks.
- `landing-page-business-suite` (and other public-facing scenarios) embed PITCH and NARRATIVE content directly.
- The operator references all of these when explaining the project externally — to family, partners, journalists, customers.

## Start here for agents

Choose the spoke by the communication question:

| Question | Start with |
|---|---|
| What is the shortest approved pitch? | [`PITCH.md`](PITCH.md) |
| What is the deeper story arc? | [`NARRATIVE.md`](NARRATIVE.md) |
| How should a common objection be answered? | [`FAQ.md`](FAQ.md) |
| What should a journalist or publication receive? | [`PRESS_KIT.md`](PRESS_KIT.md) |
| What is the investor-style slide sequence? | [`PITCH_DECK.md`](PITCH_DECK.md) |

## Files

| File | Purpose |
|------|---------|
| [`PITCH.md`](PITCH.md) | Slogan, motto, taglines, elevator pitches at multiple lengths, audience-tailored leads, key positioning lines, what-Vrooli-is-NOT framing. |
| [`NARRATIVE.md`](NARRATIVE.md) | The story arc — project description at 4 depths (1-line, 1-paragraph, 1-page, deep-vision bracketed for vision-aligned audiences only). Pointer to `VISION.md` for the operator-authored manifesto. |
| [`FAQ.md`](FAQ.md) | Canonical Q&A. Common audience questions with approved answers. Cited by advertisers, used in family-explainer / customer / journalist conversations. |
| [`PRESS_KIT.md`](PRESS_KIT.md) | Composition skeleton for journalists / external publications. Pulls boilerplate from PITCH + NARRATIVE + ASSETS + VISION. |
| [`PITCH_DECK.md`](PITCH_DECK.md) | Slide-by-slide markdown outline. Operator authors actual slide content; the outline keeps shape consistent across deck variations. |

## Posture

- **Operator-curated.** Agents propose updates via `brand-guideline-update` decisions on `marketing-crew`. They do not edit directly.
- **`VISION.md` (root)** is the operator-authored manifesto. Owned by `director-swarm` for drift detection (via `vision-update` context); operator authors substantive changes directly.
- **`docs/concepts/ARCHITECTURE.md`** is the canonical technical reference for "how Vrooli actually works." Owned by `director-swarm`; operator authors substantive expansion.
- **Visual identity** (logos, fonts, image-style) lives in [`docs/marketing/strategy/ASSETS.md`](../marketing/strategy/ASSETS.md) and [`docs/marketing/strategy/IMAGE_STYLE.md`](../marketing/strategy/IMAGE_STYLE.md). Eventually subsumed by the `brand-manager` scenario when it ships.

## Why this layer exists

Until 2026-04-27, project-identity content was scattered: implicit in `README.md`, philosophical in `VISION.md`, fragmentary in `docs/marketing/strategy/STRATEGY.md`. Every advertiser had to reconstruct the elevator pitch from primary sources every time, producing slightly different versions across drafts. This folder centralizes those answers so all external touchpoints stay coherent.

Created during vision walk #4's third divergence (2026-04-27). See `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md` for context.
