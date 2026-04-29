# Post Type: Narrative Talking Head

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-narrative-talking-head` *(planned)*
**Primary author:** `subscription-advertiser`
**Notebook home for emerging craft patterns:** `../../notebook/NARRATIVE_TALKING_HEAD_CRAFT.md` (created on first observation)

## Purpose

A narrative talking-head video is a **persona-actor speaking to camera, telling a short story or giving general advice that lands at a Vrooli scenario as the practical answer**. The format is the dominant short-video advertising shape on TikTok, Reels, and Shorts in 2026. The persona's age, demeanor, and lifestyle imply expertise; the persona never claims a credential.

Canonical example shape: persona in their 60s, casual setting, telling a short story about how they used to forget appointments / lose track of chores / waste evenings, leading into "I started using [Vrooli scenario] and now it just tells me what to do." The story is generic and broadly believable; the plug is the closer.

This is the canonical AI-UGC format for the lifestyle bundle.

## Audience

Primary: lifestyle-bundle audience on TikTok, Reels, Shorts. Persona-actor account is the typical voice surface; not the brand account.

Specific persona-actor candidates: *older person giving life advice* (life-organization bundle), *busy parent sharing routine* (lifestyle bundle), *homelab tinkerer telling setup story* (tech-lifestyle bundle).

## Conversion goal

- **Click-through to link-in-bio** is the conversion goal. Video closer names the scenario; bio link targets the appropriate landing page.
- **Watch-completion** is the algorithmic compounding signal — the format works when viewers watch to the end. First 1.5 seconds determine completion.
- Sign-up downstream of click-through.

## Structure

Apply the standard 5-part video formula (see [`../../rich-media/templates/video-prompt.template.json`](../../rich-media/templates/video-prompt.template.json)): **[Cinematography] + [Subject] + [Action] + [Context] + [Style & Ambiance]**.

Narrative arc:

- **Hook (0-1.5s).** Persona starts mid-sentence with a friction-recognition opener — "I used to ___" / "Let me tell you about ___" / "Here's something I learned ___". No intro card; faces tap straight into story.
- **Setup (1.5-10s).** Persona names the friction or moment of recognition. Concrete and broadly relatable. Audience leans in because the friction is theirs.
- **Turn (10-25s).** Persona describes the change — "and then I started using ___". Names the Vrooli scenario in the persona's own labeled-AI voice (not as a real customer testimonial).
- **Closer (25-40s).** What the persona does now; CTA to link-in-bio. Often: "Link's in my bio if you want to try it."

Length: 30-45 seconds typical. Below 25s rushes the arc; above 60s loses retention.

## Asset requirements

- **Persona-actor character JSON** at [`../../rich-media/characters/<persona-slug>.json`](../../rich-media/characters/) defines face/wardrobe/voice — kept consistent across every video the persona produces.
- **Scene JSON** at [`../../rich-media/scenes/<scene-slug>.json`](../../rich-media/scenes/) defines lighting, ambiance, environment.
- **Voice profile** in the character JSON drives the TTS or voice-model voice. Consistent voice across persona's content slate.
- **Vertical aspect ratio** (9:16) for TikTok / Reels / Shorts; cross-posts to landscape surfaces re-frame.
- **Mandatory captions burned in.** Most viewers watch with sound off; captions drive completion.
- **Disclosure label** — substantial AI-generated persona content carries the platform's native AI-content label per [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md).

## Voice

Persona voice — see `ai-ugc-personas.md` for rules. Persona is allowed informal language banned in brand voice. Persona must not claim credentials, must not give regulated-domain advice, must not impersonate real people.

## Contrarian failure modes (narrative-talking-head-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Persona-disclosure-violation** | Substantial persona video without platform AI-content label. | mode 13. |
| **Credential-claim-by-persona** | Persona claims professional credential ("as a doctor", "as a therapist") explicitly or implicitly (uniform, props). | mode 14. |
| **Real-person-impersonation** | Persona resembles a specific identifiable real person. | mode 15. |
| **Fabricated-real-customer-testimonial** | Persona claims to be a real Vrooli customer with a specific story; first-person testimonial framing. | mode 16. |
| **Regulated-domain-advice** | Story or advice touches medical / financial / legal domains. | mode 18. |
| **Uncanny-character-rendering** | Generated face has uncanny-valley artifacts (hand drift, eye drift, mouth-sync mismatch). | mode 9 specialization at the visual-rendering layer. |
| **Voice-actor-mismatch-with-character** | TTS or voice-model voice doesn't match the persona's apparent age/demeanor. | mode 9 specialization. |
| **Closer-frame-overclaim** | Plug at the end overpromises what Vrooli does. | mode 1. |

Honesty flags:

- `persona_disclosure=labeled`.
- `credential_claims=[]` (must be empty).
- `real_person_check=verified-no-likeness`.
- `regulated_domain_check=clear | flagged:<medical|financial|legal>`.
- `feature_claims=measured` for the closer.
- `persona_actor_id=<id>`.
- `voice_profile_match=verified` — voice matches character JSON's voice profile.

## Cross-cutting techniques this type uses

- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md) — first 1.5s carries massive weight; rest is body.
- [Recommendation framing](../../post-techniques/recommendation-framing.md) (with AI-UGC scope per `ai-ugc-personas.md`).

## Cross-references

- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — persona discipline, disclosure protocol.
- [`../../rich-media/characters/`](../../rich-media/characters/), [`../../rich-media/scenes/`](../../rich-media/scenes/), [`../../rich-media/templates/video-prompt.template.json`](../../rich-media/templates/video-prompt.template.json).
- Sibling: [`day-in-life-ugc.md`](day-in-life-ugc.md), [`problem-agitate-solve.md`](problem-agitate-solve.md), [`comparison-reel.md`](comparison-reel.md), [`slideshow-voiceover.md`](slideshow-voiceover.md).
- [`../../strategies/hook-library.md`](../../strategies/hook-library.md) — narrative talking-head openers.
