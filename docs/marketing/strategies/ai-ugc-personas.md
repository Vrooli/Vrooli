# Strategy: AI-UGC Personas

**Status:** v1. Authored 2026-04-28 in response to operator stance: AI-generated user-generated content is permitted under specific guardrails; this doc is the canonical home for those guardrails.

## What this is

AI-UGC ("AI user-generated content") is content posted from accounts that **look like users** rather than the brand — a persona-actor talking to camera, a slideshow with a stylized voiceover, a day-in-life vignette — produced with AI image/video tools rather than filmed with a real person.

The format is the dominant short-video advertising shape on TikTok, Instagram Reels, and YouTube Shorts in 2026. Pretending we won't use it cedes the surface to competitors. Using it without discipline produces the slimy fake-influencer content that erodes brand trust.

This doc draws the line: what shapes of AI-UGC are honest marketing embellishment, and what shapes are deception that this team will not produce.

## The line

Marketing is built on implication. A fit, attractive person in a fitness ad doesn't say "this product made me look like this," but the implication is there and that is *normal advertising*. Persona-actors work the same way: an older-looking persona implies life experience, a busy-looking persona implies the product fits a busy life, a homelab-tinkerer persona implies the tool is for hobbyists. Implication via persona traits is permitted.

The line is **specific false claims**, not implication:

- A persona that *looks* knowledgeable and gives generic life advice — fine. Voice, age, and demeanor imply experience without claiming a credential.
- A persona that *says* "as a licensed therapist, I've found…" or "as a doctor, I recommend…" — banned. That's a specific false credential claim and it's deception, not embellishment.

## What's allowed

- ✅ **Persona-actors implying traits via age, look, lifestyle, or demeanor.** "Older person giving general life advice", "busy parent showing a routine", "homelab tinkerer talking through their setup". Implication via persona traits is normal marketing — same shape as any ad with attractive models.
- ✅ **Generic advice in the persona's voice** — life advice, organization tips, productivity habits, lifestyle hacks. The content can lead the reader/viewer toward a Vrooli scenario as the answer at the end. Not a credential claim.
- ✅ **Recommendation/observation framing.** "I've been trying X" / "I noticed this thing helps". Not a sworn first-person testimonial; a recommendation in the persona's voice.
- ✅ **AI-generated label per platform requirements.** TikTok native AI-generated label, Instagram AI-content tag, YouTube AI-generated content disclosure, FTC `#ad` + AI-disclosure phrasing for any sponsorship-shaped content.
- ✅ **Brand-account content cross-posted with persona voiceover** — when the brand voice and persona voice are clearly distinguished in the asset.

## What's banned

- ❌ **Professional-credential claims by the persona.** "As a doctor", "as a therapist", "as a financial advisor", "as a lawyer", "as a pharmacist", "as a nurse practitioner". The persona may *imply* expertise via traits but cannot *claim* a credential it doesn't have.
- ❌ **Medical, financial, or legal advice in the persona's voice.** Even without an explicit credential claim, advice in these regulated domains is unsafe; the persona must not give it. (Generic *life* advice that doesn't touch these domains is fine; "drink more water" is not medical advice but "this supplement treats X" is.)
- ❌ **Real-person impersonation.** Persona must not look or sound like a specific identifiable real person — celebrity, public figure, named competitor founder, or anyone whose likeness is recognizable. AI generation makes accidental likeness easier; the contrarian validates.
- ❌ **Fabricated real-customer testimonials.** A persona presenting itself as a real, named Vrooli customer with a specific story ("I've been using Vrooli for six months and it changed my workflow") is deception. Persona-actors don't claim to be specific real users; they recommend or observe in their own persona voice.
- ❌ **Undisclosed AI generation** on platforms or in contexts where disclosure is required by FTC 2026 rules or platform policy. Substantial AI-generated content gets the platform's native AI-generated label; sponsorship content additionally carries `#ad` per FTC.
- ❌ **Persona-actor accounts that operate as undisclosed sock-puppets.** Persona accounts must be transparently AI-generated (per platform labeling); they are not pseudonymous "real users" pretending to be human. The line is: persona is allowed, *fake real person* is not.

## Disclosure protocol per platform

| Platform | Native AI-content label | FTC `#ad` for sponsorship | Notes |
|---|---|---|---|
| TikTok | Required for substantial AI content | Required if sponsorship | TikTok's AI-content disclosure is platform-enforced; non-compliance risks account-level penalties. |
| Instagram (Feed/Reels/Stories) | Required for substantial AI content | Required if sponsorship | Meta's AI-disclosure tag covers both. |
| YouTube (Shorts and long-form) | Required for substantial AI content | Required if sponsorship | YouTube AI-disclosure rolled out 2024-2025; mandatory for AI-altered realistic content. |
| X / Twitter | Recommended (no platform-enforced label as of 2026); use `#ai` or in-post disclosure | Required if sponsorship | X has no native label; in-post disclosure is the operator's responsibility. |
| Threads | Same as Instagram | Required if sponsorship | Inherits Meta's AI-disclosure regime. |
| Bluesky | Recommended; use in-post disclosure | Required if sponsorship | No platform-native label as of 2026. |
| Reddit | Required by some subreddits; check subreddit rules | Required if sponsorship | Subreddit-by-subreddit; many ban AI content outright. |
| HackerNews | Disclosure expected; follow community norms | n/a (Show HN doesn't take ads) | HN is hostile to AI marketing; AI-UGC is generally not a fit. |
| ProductHunt | In-post disclosure | Required if sponsorship | PH is a launch surface, not a UGC surface; AI-UGC is rarely the right format. |

**Substantial AI content = AI-generated voice, AI-generated face/body, AI-generated motion video.** Color-grading, light retouching, and basic editing do not require disclosure (per FTC and platform rules as of 2026, but verify per scan — researcher's `marketing-craft` scope includes platform-rule changes).

**When in doubt, disclose.** The contrarian's bias is toward *over-disclosure*; under-disclosure has long-term brand and legal cost.

## Persona-actor account discipline

Persona-actor accounts are first-class entities tracked in [`marketing-crew/shared/personas/`](../../../scenarios/prompt-manager/store/teams/marketing-crew/shared/) (folder created per-persona on activation). Each persona-actor account has:

- **`profile.json`** — name (clearly fictional or AI-persona; never a real person's name), voice, niche, age range or demeanor, target audience.
- **`accounts.json`** — platform handles (this file lives outside Git per `CHANNELS.md` secrets rule).
- **`slate.json`** — content cadence, post-type slate, do-not-pair-with restrictions (e.g., persona-A and persona-B must not appear in each other's content; cross-account cannibalization).
- **`link-in-bio.json`** — current bio link target (may be a campaign-specific landing page or a default scenario).
- **`tied-skus.json`** — which Vrooli SKUs this persona advertises (for attribution and conversion-lift measurement).

The character JSON for a persona-actor lives in [`docs/marketing/rich-media/characters/<persona-slug>.json`](../rich-media/characters/) per the rich-media schema. The persona's `profile.json` (in marketing-crew shared/) references that character.

**Account hygiene:**

- A persona-actor account never poses as a real Vrooli customer.
- A persona-actor account never claims professional credentials.
- A persona-actor account does not appear in another persona-actor account's content (cross-cannibalization risk).
- Brand-voice content does not appear on persona-actor accounts; persona voice does not appear on the brand account.
- Disclosure label is the same across every cross-post of the same content.

## Honesty flags for AI-UGC drafts

Drafts in persona voice carry, in addition to the standard honesty-flag schema, these AI-UGC-specific flags:

- `persona_disclosure=labeled | unlabeled-by-platform-rule | exempt-minor-edit-only` — `labeled` is the only acceptable value for substantial AI content; `unlabeled-by-platform-rule` reads as a violation; `exempt-minor-edit-only` requires evidence the edit was minor (color/crop only).
- `credential_claims=[]` — empty list required. Any claim of professional credential by the persona is an automatic reject.
- `persona_actor_id=<id>` — references the persona-actor entry in `marketing-crew/shared/personas/`.
- `real_person_check=verified-no-likeness` — contrarian must verify the persona does not resemble a specific identifiable real person.
- `regulated_domain_check=clear | flagged:<medical|financial|legal>` — flag any content touching medical, financial, or legal domains. Flagged drafts require operator review even if other fields are clean.

## Contrarian failure modes (AI-UGC-specific)

The marketing-contrarian member ingests these as type-level review rules layered on top of the framework-level twelve modes (see [`STRATEGY.md`](../STRATEGY.md#anti-patterns)). Each is a checkable claim against a draft.

| Failure mode | What it looks like | Why it backfires |
|---|---|---|
| **Persona-disclosure-violation** | Substantial AI content posted without the platform's native AI-generated label, or without `#ad` on sponsorship. | FTC and platform-policy violation; account-level penalty risk. Long-run brand-trust damage when surfaced. |
| **Undisclosed-AI-content** | Same as above, but the violation is structural (the workflow doesn't include the disclosure step) rather than per-draft. | Recurring violations from systemic gap; the contrarian raises a `framework-update` rather than per-draft rejection. |
| **Credential-claim-by-persona** | Persona says or implies a professional credential — "as a doctor", "as a therapist", "as a financial advisor". Includes implicit credentials via professional-uniform visuals (e.g., persona wearing a stethoscope and giving health advice). | Direct deception; FTC and platform-policy violation; lawsuit risk in regulated domains. Hard reject. |
| **Real-person-impersonation** | Persona looks or sounds like a specific identifiable real person. Includes accidental likeness from generation drift. | Likeness-rights violation; platform-policy violation; reputational risk if the real person notices. Contrarian validates against a do-not-resemble registry per persona. |
| **Fabricated-real-customer-testimonial** | Persona claims to be a real Vrooli customer with a specific story. | Direct deception. Distinct from `recommendation-framing-without-basis` (which is about *attribution*); this is about *fabricating* a customer where none exists. |
| **Regulated-domain-advice-by-persona** | Persona gives advice in medical, financial, or legal domains, with or without explicit credential claim. | Regulatory and lawsuit risk; even if the disclaimer says "not medical advice", persona-voice content touching these domains is hazardous. Hard reject; redirect to brand-voice content or different topic. |

## Voice constraints

Persona voices are not the brand voice. The same builder-not-marketer rule from `STRATEGY.md` does not apply to persona voice — a persona is allowed to sound like a normal person on TikTok, including using language ("amazing", "game-changing", "this just works") that's banned in brand voice.

But:

- The **claims** the persona makes are still subject to the same honesty discipline as brand content (no overclaim, no fabricated metrics, no credential claims).
- The **product portrayal** must align with what the scenario actually does — `feature_claims=measured` discipline applies regardless of voice.
- The **call-to-action** must match the conversion goal of the campaign and lead to a real, working surface.

## Cross-references

- [`../post-techniques/recommendation-framing.md`](../post-techniques/recommendation-framing.md) — distinguishes labeled AI-persona-actor (allowed under this doc's rules) from fabricated-real-customer-testimonial (banned).
- [`../STRATEGY.md`](../STRATEGY.md) — voice canon and the framework-level anti-patterns this doc layers on top of.
- [`../CHANNELS.md`](../CHANNELS.md) — per-platform AI-disclosure expectations.
- [`../post-types/video/`](../post-types/video/) — short-video formats where AI-UGC is the dominant production mode.
- [`../rich-media/characters/`](../rich-media/characters/) — schema for the character JSON each persona-actor depends on.
- [`marketing-crew/shared/personas/`](../../../scenarios/prompt-manager/store/teams/marketing-crew/shared/) — per-persona profile / accounts / slate / link-in-bio / tied-SKUs (folder created on first persona activation).
