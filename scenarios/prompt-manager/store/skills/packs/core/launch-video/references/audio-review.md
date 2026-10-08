# Audio and delivered-file review

## Select sound deliberately

Use music-tools for generation. Inspect the style and current compose parameters;
record structured BPM, key, duration, model/variant, seed, and the caption that
actually reached the generator. Verify whether rewrite/planning is enabled. Do
not trust a requested BPM as measured tempo. Governed provenance does not prove
musical fit or establish rights for unrelated stock assets.

Recall the operator's listening feedback and inspect prior delivery folders,
`alternates/music/`, and the music-tools pool. “Whatever we had before” means the
actual earlier candidates/styles, not a newly inferred mood. Preserve a requested
upbeat, catchy, futuristic trap/drill/hip-hop character in the fallback too.
Variety means distinct style families; different seeds of one style are take
variety only. For flagship exploration, compare the requested style families
before spending on many seeds. Keep useful older candidates even when new ones
are generated.

Reuse suitable provenance-complete audio first. Start with two takes for teaser
or three for flagship; expand only when audition findings justify the cost.
Retain every candidate. Record requested versus perceived/measured tempo, density,
strong cues, direction fit, chosen/rejected reason, and the audition method in
`audio-selection.md`. If audio cannot be auditioned, say so and do not mark it
reviewed. Do not silently choose the first successful job.

Brief rhythm, instruments, bass, texture, build, break, and final resolve. Use
silence when it directs attention. Align a few accents to visible actions; do not
sonify every tween. Readability outranks a beat grid. If generation would threaten
shared capacity, use a licensed existing take or a disclosed silent timing rough. Original
oscillator/noise music is not a substitute for a requested produced soundtrack;
it requires an explicit original-score brief. Small authored SFX are separate. Preserve provider/model/license and local recipe data.
Never install a model or stop a shared service from this skill.

Narration is opt-in. Use governed audio-tools roles; do not adopt an upstream cloud
provider automatically. Write for the ear, measure the actual audio, align proof to
spoken words, duck music, and provide readable captions. Do not read visible text
word for word. External generated inserts need provider/model, price, commercial
terms, retention/training policy, provenance, and an authorized data-egress decision.

## Critique the file that ships

Run `scripts/production.py inspect <video.mp4> --out <new-review-directory>` after
all encode, audio, and poster operations. The helper records ffprobe metadata,
full decode status, black/silence diagnostics, loudness, SHA-256, and samples from
the exported file. These are observations, not a creative verdict. Black or silent
intervals can be intentional; review them in context.

Inspect frames at settled beats and on both sides of cuts. Inspect motion through
short dense sequences or actual playback; a contact sheet alone cannot prove
continuity. Listen to the exported audio. Review once muted for comprehension.
Use one reviewer pass that starts from the delivered media and truth pack rather
than trusting the composition source. If a separate critic is available and
explicitly authorized, give it the same bounded review task; otherwise retain a
separate self-critique pass and disclose that limitation.

`quality-review.json` contains the asset digest, reviewer identity/method, visual
and audio observations, and defects. Each defect contains timestamp, severity
(`blocking`, `major`, `minor`), category, observation, consequence, recommendation,
and disposition. Categories: story, product clarity, composition, typography,
motion, transition, audio, branding, technical, claim risk. The critic reports
specific defects, not an automatic whole-film rewrite.

Preserve v1. Fix high-value defects, export v2, and inspect it again. Record the
before/after files and what changed in `revision-notes.md`. Bind each review to its
actual asset digest; an earlier review never approves a later encode. A flagship
normally includes one revision; if the first export has no material defects,
record the exception and a fresh second review rather than manufacturing a defect.

Measure delivery loudness and true peak against the recorded channel target
(default music-led web draft: -16 LUFS ±2, ≤-1 dBTP). Do not interpret levels as
proof of musical quality. Check clipping, unexpected silence, sound/visual sync,
encode dimensions/fps, stale media, black frames, readable holds, and the ending.
Keep all output as drafts. Claim/factual risk blocks delivery readiness; publication
requires explicit operator review even after every check passes.

Final reviews declare `audio_status`: `auditioned-pass`, `unreviewed`, `fail`, or
`not-applicable` (silent/still). Music/voice delivery requires `auditioned-pass`.
Measured loudness and technical decode cannot satisfy listening review.

## Listening is a capability, not a phrase in a report

When the agent cannot hear audio, it must not claim an audition or emotional-fit
verdict. Reuse operator-preferred material when possible, identify any provisional
choice, and supply a compact local player with the actual candidates. Technical
analysis may eliminate silence/clipping or expose tempo drift, but cannot pick
the “catchiest” track. Retain a listening-review gap without replacing the track
with easier-to-measure synthetic sound. Final-file gate remains unpassed until an
authorized listener reviews the exact exported audio.
