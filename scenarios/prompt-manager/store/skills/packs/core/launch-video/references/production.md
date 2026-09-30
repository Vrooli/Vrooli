# Direct one promise

Write `product-truth.md` before story work. Give each claim an ID, source revision
or read timestamp, approved meaning, required qualifiers, availability, and any
exact-copy restriction. Separate verified behavior from marketing aspirations.
Record brand inheritance and shipped assets. Do not treat source precedence as
permission to show a feature that the product cannot perform.

Write one core promise, primary viewer, problem, transformation, decisive proof
moment, and takeaway. Select evidence that proves that promise. Feature coverage
is not an objective. A scene must add something that would be lost if it were cut.

## Work table

| Checkpoint | Teaser | Flagship | Failure action |
|---|---|---|---|
| Safety | Verified instance/build/dependency/data assessment | Same | Stop unsafe capture; continue safe work |
| Truth + story | One source-linked promise and proof | Same | Narrow unsupported claims |
| Visual continuity | Rendered page reference and source-to-video mapping; reuse a current reviewed treatment | Same plus comparison across four styleframes | Restore missing identity cues; record justified departures |
| Direction | One deliberate direction | Three alternatives and scored choice | Change story/shot language, not only colors |
| Shot plan | Purpose, truth, surface, state/action/result, timing, crop, sound | Same with curated shot vocabulary | Cut redundant or infeasible shots |
| Styleframes | Optional for established treatment | Opening, hero, detail, ending; reviewed before polish | Repair type/hierarchy/framing |
| Animatic | Optional for straightforward cut | Timed rough export; review story/readability/proof | Recut before expensive animation |
| Compose | HyperFrames; local pinned assets | Same; motivated motion and pacing contrast | Fix renderer checks and inspected frames |
| Export review | Actual final MP4, sound-on and muted | Same plus retained critique/revision loop | Fix major defects and review changed export |
| Deliver | Draft, poster, sources, alternatives, decisions | Also concepts, styleframes, animatic, revision history | List missing evidence; never silently pass |

Checkpoints are agent review decisions, not automatic requests for user permission.
The user reviews the concrete draft before publication. Do not publish it.

## Direction comparison

Each direction states concept, audience fit, emotional arc, narrative structure,
shot language, density, typography role, camera behavior, transitions, audio, and
ending. Score product clarity, factual support, distinctiveness, feasibility,
source-material fit, and readability from 1–5 with reasons. Scores record judgment,
not objective quality. Choose a direction whose decisive shot can actually be made. Select from usable
captures before full polish. In a product launch, real UI occupies at least half
the film and must be large enough to inspect; a tiny decorative product inset does
not satisfy the intent. Custom scenes introduce, connect, and resolve that proof.

Evaluate against the user's requested energy and the previous accepted film, not
only the agent's own draft. “Clean,” “calm,” and “restrained” are direction choices,
not universal definitions of premium. Look for a distinctive opening silhouette,
sharp editorial contrast, one memorable interaction, motivated spatial continuity,
and a concise ending. Do not equate added diagrams or more transitions with craft.

## Landing-page visual continuity

Inspect the actual published product route after fonts and assets load. Save
representative hero, product, and closing frames in `visual-reference/`. Record
the URL, read time, viewport, theme values and relevant computed styles. Inspect
motion separately when a reduced-motion screenshot hides the page's signature.
If the page is unavailable, use a dated verified reference or Brand Manager and
record the gap. Do not silently invent an unrelated treatment.

Extend the generated `composition/DESIGN.md` with this compact mapping:

| Cue | Page observation and source | Video application or reason to depart |
|---|---|---|
| Palette | Canvas, raised surface, text, muted text, accent and CTA roles | Preserve each role; do not merely sample a dominant color |
| Type | Font, weight, line height, tracking, accent words and label hierarchy | Scale for video readability while retaining hierarchy |
| Surface and light | Border, radius, rim light, shadow and depth | Frame real captures with the same treatment |
| Motif | Product-specific geometry, texture, imagery or constellation | Adapt the product's own motif; keep it secondary to proof |
| Composition | Spacing, alignment, product framing and density | Translate into the shot's aspect ratio and focal point |
| Motion | Reveal, settle, transition and atmosphere | Compress for film without extending the hook or adding constant drift |

Keep token differences explicit. A mark's container color and a page's campaign
accent can have different roles. Brand Manager governs identity assets and explicit
mandatory restrictions; the current official landing page governs the surrounding
presentation treatment by default. Report conflicts with explicit restrictions.
Leave the source systems unchanged. Page order remains independent of film
order; visual continuity does not narrow the story to a scrolling page tour.

Before polish, compare opening, product, detail and ending frames against these
references. For an established teaser treatment, one compact comparison is enough.
Record a visual verdict and specific mismatches. Repair a missing defining cue
before full rendering, or explain a deliberate departure. Matching hex values is
not a passing visual review. This is a reviewer judgment, not an automated taste
score. Review the exported file again because renderer output can differ.

Do not use illustrative landing-page UI as evidence of shipped functionality.
Keep real captures intact. Put the page's design into surrounding composition,
type and lighting rather than repainting the product. Preserve the requested
music character when adopting the page's visual tone.

## Three visual classes

1. **Product evidence:** moving capture of a real interaction or changing result.
   Record the instance, source digest, capture date, known start state, action, and
   end state. A composition camera move does not prove product motion.
2. **Product presentation:** faithful crop/mask/reframe of that evidence. Retain the
   source and transform. Never change values, remove qualifications, or imply a
   functionality or speed the capture does not demonstrate. Label time compression.
3. **Explanatory/brand:** typography, diagrams, metaphors, generated atmospherics.
   Label simulations/concepts where a viewer could mistake them for real behavior.
   These cannot satisfy the product-proof gate. Never generate counterfeit UI.

## Shot selection and motion

Retrieve a few recipes by intent with `scripts/production.py shots <intent>`.
The compact catalog is vocabulary, not a running order. Choose no effect merely
because it exists. For each major motion, finish: “This makes the viewer notice …”.
“Cinematic” is not an answer. Plan anticipation → action → settle → hold; vary pace
around the proof. A still camera can be the strongest choice.

Record crop bounds in encoded source pixels and the maximum displayed footprint
in output pixels, including the largest camera scale. Both crop dimensions must
meet that footprint. Pixel count is necessary, not sufficient: inspect compression,
focus, and text at delivery size. Capture a detail closer when pixels are lacking.
Set both CSS width and height for captured video in HyperFrames, preserving the
source aspect ratio before masking. Version 0.8.44 injects intrinsic width/height
attributes during export; a width-only element can acquire an unintended height
and letterbox the product inside its crop. Snapshot appearance alone does not
catch this: compare the rough MP4 with its source footage before full rendering.
Do not solve weak hierarchy by adding multiple rings. Measure an overlay against
the exact capture layout, and remove it when its subject scrolls away.

## Styleframes and animatic

Compare four representative frames with the landing-page references for visual
continuity. Review brand, dominant focal point, typography,
contrast, density, authenticity, and generic-template symptoms. Keep a short
written verdict for each. Fail a frame with two competing primary reads.

The animatic uses the actual intended shot durations and reading holds. Review
hook, sequence, proof, redundancies, and ending on mute. Use temporary audio only
with its provenance and replacement status recorded. Do not add polish to rescue
an unproven story.

## Production record

Record elapsed time, render time, dimensions/fps, renderer version, generated media
jobs/model/seed, API cost if reported, attempts, manual interventions, and failures.
Unknown cost is `unknown`, not zero. Compare renderer alternatives using identical
source assets and timing. Compare orchestration separately from renderer changes.
A small Remotion test is optional; a new dependency and license obligation require
an evidenced benefit. Do not change the default renderer from documentation alone.

## Accepted reference: Vega, 2026-09-22

The operator accepted the landing-style draft: “I like this a lot” and “pretty
good for now.” The local evidence directory is
`~/.vrooli/evidence/vega-launch-video-retry-2026-09-22/video-evaluation/landing-style/`.
Start with `review.html`, `composition/DESIGN.md`, `production.json`, and
`quality-review.md`. The primary `video.mp4` SHA-256 is
`e28d11ff08df7d8c739953f97142a5d6f19de83d7de95ab922ac5a4219af5250`.
The prior product-free studies are rejected examples, not this quality reference.

Reuse the method: 24 of 28 seconds contain the actual frontend; a recorded click,
chart inspection and process scroll supply proof; custom type and geometry frame
that footage; the rendered page supplies its visual identity; original music
families remain available; the exported file receives critique. Preserve those
relationships, not Vega's copy, colors, constellation, or six-scene arrangement.
Read each new product's sources and select its own proof moment.

The example uses labeled fictional data in an isolated frontend. It does not
establish backend investigation behavior or make other products safe to capture.
Its composer is a retained production example, not a portable render service.
Discover a current installed toolchain; do not depend on its old `/tmp` location.
Operator draft acceptance is not publication permission or a fabricated agent
audio audition. Record acceptance separately from technical/listening checks.
