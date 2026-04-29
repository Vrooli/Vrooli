# Post Type: Demo Recording

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-demo-recording` *(planned)*
**Primary author:** `subscription-advertiser` or `oss-advertiser`
**Notebook home for emerging craft patterns:** `../../notebook/DEMO_RECORDING_CRAFT.md` (created on first observation)

## Purpose

A demo recording is **a screen recording of a Vrooli scenario in actual use, with optional voiceover narration**. Ranges from 30-second cut-downs (TikTok/Reels/Shorts cross-post) to 5-15 minute architecture walks (YouTube long-form, blog embed). Brand voice; not a persona format.

Distinct from `../text/scenario-spotlight.md`: that's a *text post with attached video*; this is a *standalone video post*. The text-led spotlight may use a demo recording produced via this type as its asset.

## Audience

Primary: subscription buyer for short-form demos; dev-tool buyer for long-form architecture walks; OSS contributor for technical deep-dives.

## Conversion goal

- **Click-through to landing page / sign-up** for short-form demos.
- **Watch-time / channel-subscribe** for long-form (compounds the channel's algorithmic reach).
- **Star / follow / share** for OSS-aimed technical demos.

## Structure

Short-form (30-60s):

- **Hook (0-2s):** the workflow's outcome shown first, in reverse chronology. ("Here's the schedule it built me. Here's how.")
- **Demo (2-50s):** screen recording showing the workflow in actual use.
- **Closer (50-60s):** CTA + link.

Long-form (2-15min):

- **Cold open with the outcome** shown briefly.
- **Walk through the workflow** with voiceover.
- **Show edge cases / honest about limits.**
- **Closer with CTA + further-reading links.**

## Asset requirements

- **BAS-produced screen recording** is the canonical substrate (per `../text/scenario-spotlight.md`'s asset-production protocol). Known constraint: BAS recordVideo gray-bar issue with CDP workaround documented in BAS scenario docs.
- **Sanitization** per `x-dev-log` guardrails: no paths, emails, credentials, internal IDs in the recording.
- **Brand assets** per [`../../ASSETS.md`](../../ASSETS.md), [`../../IMAGE_STYLE.md`](../../IMAGE_STYLE.md).
- **Captions burned in** for short-form (sound-off viewing). Captions optional but recommended for long-form.
- **AI disclosure** required only if AI-generated voiceover; raw screen recording from real Vrooli usage doesn't require AI-content label (per platform rules — verify per scan).

## Voice

Brand voice. Voiceover narration follows `STRATEGY.md` voice canon — builder, not marketer. Voiceover is *not* persona voice; demos are honest first-person operator narration or AI-narrator-clearly-labeled.

## Contrarian failure modes (demo-recording-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Demo-theater** | Recording shows idealized happy path; hides retries, failures, setup steps. | mode 1 (hype-drift) — scenario-spotlight specialization extended here. |
| **Capability-inflation** | Demo claims the scenario does X, but X is partial / WIP. | mode 1. |
| **Sanitization-leak** | Recording contains paths, emails, credentials, or internal IDs. | mode 10 (internal-vocabulary-leakage). |
| **Brand-asset drift** | Off-palette overlays, outdated logo. | brand-canon violation. |
| **Voiceover-voice-drift** | Voiceover slips into corporate-marketer language. | mode 2. |
| **Pricing-tier confusion** | Demo'd feature is gated behind a tier the audience's CTA doesn't grant. | mode 1 specialization. |

Honesty flags (mirrors scenario-spotlight):

- `feature_claims=measured | overclaimed | uncertain`.
- `demo_authenticity=replicable | operator-only | mixed`.
- `tier_alignment=verified | not-yet-checked`.
- `sanitization_pass=verified` — required.

## Cross-cutting techniques

- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md).
- [No internal numbering externally](../../post-techniques/no-internal-numbering-externally.md).
- [Competitive comparison](../../post-techniques/competitive-comparison.md) when demo positions against named alternative.

## Cross-references

- [`../text/scenario-spotlight.md`](../text/scenario-spotlight.md) — text-led sibling; often consumes a demo-recording as its attached asset.
- [`../../ASSETS.md`](../../ASSETS.md), [`../../IMAGE_STYLE.md`](../../IMAGE_STYLE.md).
- Browser Automation Studio scenario at `scenarios/bas/` — asset substrate.
- Sibling video types: [`narrative-talking-head.md`](narrative-talking-head.md), [`comparison-reel.md`](comparison-reel.md).
