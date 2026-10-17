# Audio and delivered-file review

## Select sound deliberately

Use music-tools for generation. Inspect the style and current compose parameters;
record structured BPM, key, duration, model/variant, seed, and the caption that
actually reached the generator. Verify whether rewrite/planning is enabled. Do
not trust a requested BPM as measured tempo. Governed provenance does not prove
musical fit or establish rights for unrelated stock assets.

Recall the operator's listening feedback and inspect prior delivery folders,
`alternates/music/`, and the music-tools pool. “Whatever we had before” means the
actual earlier candidates/styles, not a newly inferred mood. Write the current
sound brief before selection: required pace/energy, rhythm and bass character,
reference excerpt with source hash/times, opening/build allowance, and exclusions.
Preserve those choices in reuse and fallbacks. Do not carry an older genre label
forward when the operator has changed the direction.

Choose variety inside that brief. Distinct timbre, rhythm, phrase, arrangement or
take can supply variety; a different genre is optional. Reject a family change
that weakens the requested energy. Do not replace fast, hard electronic music
with slower corporate/launch music merely to make two films different. Keep
useful older candidates even when new ones are generated.

Reuse suitable provenance-complete audio first. Start with two takes for teaser
or three for flagship; expand only when audition findings justify the cost.
Retain every candidate. Record requested versus perceived/measured tempo, density,
strong cues, direction fit, chosen/rejected reason, and the audition method in
`audio-selection.md`. If audio cannot be auditioned, say so and do not mark it
reviewed. Do not silently choose the first successful job.

Compare short level-matched excerpts, including the operator-preferred anchor,
before committing the full cut. Compare the excerpt used in the edit, not only
the source song's name or generator caption. Record whether the opening reaches
the useful rhythmic phrase promptly and whether the edit matches its intended
energy. Authored BPM, measured pulse/onsets and perceived energy are different
observations. A high BPM can still feel half-time or sparse. If hearing is
unavailable, keep that choice provisional and preserve the known preference;
do not claim the new arrangement beat the reference.

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
proof of musical quality. Measure the actual delivered encode: a filter report
or normalized PCM does not establish AAC reconstruction peaks. If a boundary
transient fails, isolate PCM and encoded output with a short controlled test,
retain evidence, and verify the corrected final file. Check clipping, unexpected silence, sound/visual sync,
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

## Arrange and verify useful coverage

Generated files can contain long silent tails. Inspect useful signal span before
planning the cut. Retain rejected sources and reasons. Select explicit tracks per
product. The October operator preferred Aquila's energetic DnB over Vega's and
rejected the earlier reused bed. This is a preference reference, not a universal
DnB rule.

Use `music-tools delivery arrange --recipe <recipe.json> --out <new-directory>`
for retained source segments. Read its owner contract in
`path:scenarios/music-tools/docs/reference/cli-commands.md`. It decodes independently,
joins exact PCM samples, validates sample arithmetic, retains source/recipe/output
hashes and marks audition unreviewed. Retire the failed split/trim/acrossfade graph
from R3/R5. Keep the receipt in the production manifest. Listen to useful signal,
seams and ending; correct length cannot establish a good musical edit.

Inspect encoded audio through the ending after mastering or CTA extensions.
A 33-second container with a 23.115-second audio stream fails coverage.
`production.py inspect` checks stream timing and retains signal diagnostics.
When WebM omits stream duration, the owner probes beginning/EOF packet boundaries,
including explicit codec skip/discard samples. Those observations retain their
method in metadata; DURATION tags/container length never substitute for audio
coverage. Missing/nonfinite packet timing remains unqualified. Boundary coverage
does not prove continuous useful sound, absence of gaps, or musical fit.
Intentional fades/silence need review. The Music Tools Audition view and report
players provide an operator playback path; they do not prove agent hearing.
Record listener identity, method, actual candidate/final hash and finding.

Current operator escalation (2026-10-16): all four matched continuation samples
were rejected as not good, fast or hardcore enough, including prior174DnB
anchors and new178DnB/152hardgroove probes. Their source IDs are excluded in
examples.json; historical preferences do not override this newer feedback.
Aim for immediate, forceful hard-dance energy with heavy kick/bass and dense
driving percussion when that is the brief. A higher prompt tempo, genre label,
measured pulse or technical master is not evidence that the result meets it.
Compare useful passages at matched levels; retain actual listener rejection and
revise the sound direction and edit rather than polishing a rejected score.


## Artist references and instrumental launch scores

The later October16 clarification names Slayyyter, ARTMS and KiiiKiii as vibe references. It also explicitly requires lyric-free promotional music while retaining ordinary song generation as a separate capability. This updates the brief; it does not approve any generated take. At that clarification checkpoint the prior hardcore/dance experiments were unselected. The operator subsequently explicitly rejected all three harder-v3 takes as primitive-sounding; those source IDs are now excluded. The two later v4 takes remain paused without an inferred taste verdict.

Extract concrete original production hypotheses: a prominent instrumental hook, bass/drum impact, rhythmic density, timbral detail and short phrase changes that sustain momentum. Reference catalogs can cover many styles. Record what the operator actually identified and distinguish documented descriptions from inferred creative choices; do not claim to have heard an artist or generated result from metadata alone. Retain fast/upbeat intensity while testing hook, groove and sound design. No fixed genre or BPM band becomes a universal requirement.

For this instrumental brief, explicitly request no singing, speech or vocal chops and preserve the sent lyrics/instrumental setting in generation evidence. A prompt setting is intent, not proof of a vocal-free output; listening remains required. When ordinary songs are requested, use the music owner's explicit lyrics input and retain its provenance. If that route is missing, repair/document the owner gap rather than pretending it is supported or changing every generation to instrumental.


## Diagnose persistent primitive or generic results

If several actual takes fail the listener’s brief, pause more prompt variations and audit the real model integration. Confirm the loaded checkpoint and planner, effective sampling settings, semantic-code generation and actual conditioning input. Separate correctness defects, supported tuning choices and an unproven model-quality ceiling. The October16 audit found that the wrapper’s default upstream random-seed mode ignored recorded requested seeds; large seeds also passed through a lossy float conversion. Historical requested seeds are not actual sampler provenance. A controlled comparison needs safe effective seeds and returned resolved evidence, with all other factors recorded. A resolved audio seed does not prove that the musical planner applied it. Trace the actual RNG consumer for each stochastic stage. Require matching returned semantic-code witnesses before attributing a same-brief sampler comparison to one changed parameter; differing witnesses make it confounded. The pinned single-item PyTorch planner ignored passed seeds even after the audio-seed flag was corrected, which the first live comparison exposed.

Eight steps and guidance1 are expected for the inspected turbo model. Disabling caption rewriting does not disable its semantic-code planner. The supported timestep-shift default and a larger-model preset need real controlled comparison and listening; neither is an automatic quality pass. Count actual assembled tokenizer input before blaming truncation. Avoid new batch production until the next experiment tests a falsifiable cause. Root audit and pinned primary links: `path:/home/matthalloran8/.vrooli/plan-artifacts/efforts/video-toolchain-reliability-2026-10-16/research/music-ai-quality-audit.md`.
