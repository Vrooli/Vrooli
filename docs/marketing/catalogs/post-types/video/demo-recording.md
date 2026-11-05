# Post Type: Demo Recording

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-demo-recording` *(planned for general demos)*. Launch-film drafts use `prompt-manager skill read launch-video`; this does not activate publishing for the post type.
**Primary lane/member:** `producer` — lane depends on demo subject. Release execution is unowned until account operations gain a scheduler-side home.
**Craft observation topic:** `marketing-craft-observation/demo-recording/<slug>`

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

- **BAS-produced screen recording** is the canonical substrate (per `../text/scenario-spotlight.md`'s asset-production protocol). Read current capture/export limits in `scenarios/browser-automation-studio/api/services/export/README.md`; verify the actual export rather than relying on historical workarounds.
- **Sanitization** per `x-dev-log` guardrails: no paths, emails, credentials, internal IDs in the recording.
- **Brand assets** per [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md).
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

- [Hook-vs-body length asymmetry](../../../methods/post-techniques/hook-vs-body-asymmetry.md).
- [No internal numbering externally](../../../methods/post-techniques/no-internal-numbering-externally.md).
- [Competitive comparison](../../../methods/post-techniques/competitive-comparison.md) when demo positions against named alternative.

## Cross-references

- [`../text/scenario-spotlight.md`](../text/scenario-spotlight.md) — text-led sibling; often consumes a demo-recording as its attached asset.
- [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md).
- Browser Automation Studio scenario at `path:scenarios/browser-automation-studio/` — asset substrate.
- Sibling video types: [`narrative-talking-head.md`](narrative-talking-head.md), [`comparison-reel.md`](comparison-reel.md).

## Directed launch-film variant

Before story selection, discover the useful customer workflows and resolve the
intended offer and deployment targets. Evaluate feature reliability, dependencies
and setup on that delivery path before ranking benefits. Retain valuable excluded
capabilities with reasons. Link selected benefits, qualified claims and real proof
through launch-video's versioned product brief. Discovery may propose overlooked
value; it does not approve claims. A phone-sized player and depicted phone use
are different decisions. Review a plain script or rough for product purpose and
customer benefit before expensive production.

Use an audience/context/promise brief, decisive genuine product proof and a usable
next action for a launch film. The demo timing above is an example sequence, not
a universal shot layout. Compare distinct timed directions before expensive
polish. Mix readable product captures, purposeful type, source details, conceptual
inserts and holds; select the footage balance from the story. Preserve shipped
marks and mandatory restrictions without imposing a permanent corner-logo/footer
layout. Keep fictional-data provenance in the report/share context and preserve
truthful availability implications. Respect the operator's preference against
persistent demo/mock/date labels.

Carry the operator's current sound and pacing preference into a recorded brief.
For this campaign, keep fast, hard, upbeat electronic music as the energy floor;
variety can come from rhythm, timbre and arrangement within it. A nominal tempo
or valid audio file does not establish perceived energy. Compare short excerpts
at matched playback level and record actual listening separately. Preserve useful
operator-rated techniques across tool changes: polished generated material plates
with motivated camera/light motion are an available medium when they explain the
story or add a deliberate visual payoff. Retain exact prompts, source dimensions
and licensing/provenance; do not imply a moving still has animated geometry.
Before full production, review a short representative sequence that combines the
chosen sound, a conceptual scene, a real interaction/result and their transition.
A technical pass cannot substitute for creative comparison with the stronger
rated reference. Distinct colors alone do not make distinct shot grammar.

Review technical correctness, first-time viewer understanding and individual-shot
craft separately against the exact final export. Separate features, sample inputs
and observed results. End with an explicit action and a full verified public URL;
when using QR, decode compressed final-video frames at intended playback sizes.
Retain raw takes, alternatives, costs, decisions, exact feedback and hashes.
`launch-video` owns this draft-production procedure and its rated references.
Operator ratings establish local quality references; future reliability and
conversion need independent-brief and audience evidence. Publication authority,
platform requirements and post-type activation remain separate.
