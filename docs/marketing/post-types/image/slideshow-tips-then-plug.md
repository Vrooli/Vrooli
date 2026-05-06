# Post Type: Slideshow Tips-Then-Plug

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-slideshow-tips-then-plug` *(planned)*
**Primary lane/member:** Usually subscription lane (`subscription-advertiser`)
**Notebook home for emerging craft patterns:** `../../notebook/SLIDESHOW_TIPS_THEN_PLUG_CRAFT.md` (created on first observation)

## Purpose

A tips-then-plug slideshow opens with **3-5 generic, broadly-useful tips** in the topic area, then **closes with a Vrooli scenario as the practical next step**. The arc is: lean into a need the audience already has → deliver enough useful content that the swipe was worth it → frame the Vrooli scenario as the natural amplifier of the tips just shown.

Example shape: "5 ways I keep my house organized" — tips 1-4 are generic (label drawers, weekly reset, capsule wardrobe, etc.), tip 5 is "I plan the whole cleaning schedule in [Vrooli scenario] so I don't have to think about it." The audience leaves with both the tips AND a concrete tool to act on them.

This is the **canonical AI-UGC persona-account format** for the lifestyle bundle: the persona shares useful general-purpose tips, the closer frame plugs Vrooli without claiming to be a real customer.

Distinct from `slideshow-listicle.md`: that's a symmetric list. This is asymmetric — the early tips are setup; the closer is payoff.

## Audience

Primary: lifestyle-bundle audience on TikTok, Instagram Reels, Instagram Feed carousels, LinkedIn carousels. Persona-actor account is the typical voice surface.

Specific persona-actor candidates: "homelab tinkerer" (tech-lifestyle bundle), "busy parent" (lifestyle bundle), "older person giving advice" (life-organization bundle), "indie maker" (dev-tool bundle).

## Conversion goal

- **Click-through to link-in-bio** is the conversion goal. The closer frame names the Vrooli scenario; the link-in-bio targets the appropriate landing page or scenario surface.
- **Save / share** as a secondary signal — strong save/share rates indicate the tips were genuinely useful, which compounds reach.
- **Sign-up** is downstream of click-through; not a one-frame ask.

## Structure

- **Cover frame:** hook. Format: "[N] ways to ___" or "[N] tips for ___" or "How I stopped ___". Sets the topic and signals "useful content ahead."
- **Tips frames 1 through N-1:** each frame delivers one generic, useful tip. **No Vrooli mention in these frames** — the tips have to stand on their own merit. If they're not useful without Vrooli, the slideshow fails the contrarian's `padded-tips-pre-plug` check.
- **Closer frame (the plug):** the scenario gets named here. Persona introduces it as how they personally keep up with the tips ("I plan my whole cleaning schedule in [scenario] so I don't have to remember"). Persona is a *recommendation* in their own labeled-AI voice — not a fabricated real customer.
- **Frame count:** 5-7 typical. Below 4 reads as a thin pretext; above 7 loses the swipe momentum.

## Asset requirements

- **Multi-frame visual consistency** — same as `slideshow-listicle.md`; the rich-media schema at [`../../rich-media/`](../../rich-media/) is the substrate.
- **Persona-actor character JSON.** Required when a persona is depicted across frames; the JSON in [`../../rich-media/characters/`](../../rich-media/characters/) keeps face/wardrobe/scene consistent.
- **Brand assets in the closer frame.** Logo, scenario name, tier-aligned CTA per [`../../ASSETS.md`](../../ASSETS.md).
- **Disclosure label** — substantial AI-generated persona content carries the platform's native AI-content label per [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md).

## Voice

Persona voice is the default. Persona voice is **not the brand voice** — see [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) for the rules. Persona voice can sound like a normal person on TikTok, including informal language banned in brand voice. But:

- Claims about Vrooli features in the closer frame must pass `feature_claims=measured`.
- Persona must not claim a credential (no "as a doctor", "as a financial advisor"). See AI-UGC mode 14 (`credential-claim-by-persona`).
- Persona must not give regulated-domain advice (medical, financial, legal). See mode 18.
- Persona is *recommending* the scenario in its own voice; not claiming to be a real Vrooli customer (mode 16).

## Contrarian failure modes (slideshow-tips-then-plug-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Padded-tips-pre-plug** | Early tips are weak filler so the closer frame can plug Vrooli. The audience feels deceived when they realize the tips were a vehicle. | mode 12 (what-without-why) — tips without genuine utility. |
| **Plug-too-early** | Vrooli is mentioned before the closer frame; reads as ad rather than as value-then-recommendation. | mode 1 (hype-drift) variation. |
| **Asset-style-drift-across-frames** | Visual identity drifts across frames. | mode 9 specialization. |
| **Persona-disclosure-violation** | Substantial persona content without AI-content label. | mode 13. |
| **Credential-claim-by-persona** | Persona implies professional credential. | mode 14. |
| **Fabricated-real-customer-testimonial** | Persona claims to be a real, named Vrooli customer with a specific story. (Distinct from the allowed pattern: persona recommending in their own labeled voice.) | mode 16. |
| **Regulated-domain-advice-by-persona** | Tips touch medical / financial / legal advice. | mode 18. |
| **Closer-frame-overclaim** | Plug overpromises what Vrooli does. | mode 1. |

Honesty flags:

- `tips_useful_without_vrooli=YES | NO` — must be YES. The contrarian validates by checking whether tips 1 through N-1 stand alone.
- `persona_disclosure=labeled`.
- `credential_claims=[]` — empty list required.
- `regulated_domain_check=clear | flagged:<medical|financial|legal>`.
- `feature_claims=measured` for the closer frame.
- `real_person_check=verified-no-likeness`.
- `persona_actor_id=<id>` — references the persona-actor entry.

## Cross-cutting techniques this type uses

- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md) — cover hook is short; tips frames are the body.
- [Recommendation framing](../../post-techniques/recommendation-framing.md) — *with the AI-UGC scope clarification* — persona voice is allowed when persona is labeled-AI; not a third-party-attribution fabrication.

## Cross-references

- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — persona-actor account discipline, disclosure protocol, persona voice rules.
- [`../../rich-media/characters/`](../../rich-media/characters/) — persona-actor character JSON.
- [`../../rich-media/`](../../rich-media/) — multi-frame visual consistency substrate.
- [`slideshow-listicle.md`](slideshow-listicle.md) — symmetric-list sibling format.
- [`../video/slideshow-voiceover.md`](../video/slideshow-voiceover.md) — same arc rendered as video with TTS narration.
- [`../../strategies/hook-library.md`](../../strategies/hook-library.md) — tips-then-plug cover hooks.
