---
name: "music-generation"
description: "Choose a governed local music model for the sound brief, preserve generation provenance, and compare actual audio before selection. Covers instrumental launch scores and ordinary songs with explicit lyrics."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "audio", "music", "video"]
  status: "active"
  revision: 8
  updatedAt: "2026-10-17T00:00:00Z"
  origin:
    kind: "authored"
---

## Tools focus: Music Generation

Write the current sound brief before choosing an engine: instrumental or vocals,
pace and perceived energy, hook, rhythm, bass, timbre, arrangement and intended use.
Preserve the operator's actual feedback; a default backend or high authored BPM
cannot establish musical fit.

For this operator's harder, upbeat instrumental launch music, pinned **Stable
Audio 3 Medium** is the current preferred local starting point. The October 17
pilot's matched excerpt pair received 8.5/10, with harder B preferred; the operator
then approved the full 48-second B source. That evidence
is specific to those audio files, not a universal model ranking or approval of
future tracks. Retrieve `stable-audio-3-harder-instrumental` through launch-video's example
catalog for exact audio, request, generation receipt and feedback hashes. Preserve
the harder production character while creating distinct original tracks per brief.

Use Music Tools' maintained styles → compose → jobs → takes → delivery route.
For fresh scores for different products, write a distinct original caption/style
for each product first, compile each, then generate its bounded candidate batch.
Two seeds of the same default preset are alternatives, not evidence of distinct
product direction. Explicit freshness overrides reuse-first guidance: approved
existing music is the listening anchor or a disclosed fallback.

The default `launch-club` style uses the qualified harder instrumental brief:

```sh
music-tools compose --style launch-club --takes 2
music-tools jobs wait JOB_ID
music-tools takes list --job JOB_ID
music-tools takes receipt TAKE_ID --out NEW_RECEIPT.json
music-tools takes audio TAKE_ID --out NEW_SOURCE.wav
```

For a distinct original, import a new style with a brief-specific caption, blank
lyrics (instrumental), and `variant: "stable-audio-3-medium"`; use 48 seconds,
8 inference steps and guidance 1 as the qualified starting settings. Stable Audio
accepts tempo/key intentions in the caption; leave structured BPM at 0 and key
empty. It rejects unsupported controls rather than ignoring them. Read
`path:scenarios/music-tools/docs/reference/cli-commands.md` for portable style JSON
and complete command syntax. Do not edit a built-in style or copy the same caption
into every product campaign.

Lyrics are part of the common Style input, not a separate model-specific user
workflow. Use the preferred model when its declared capability supports authored
sung lyrics. Select another capable model only for an actual unsupported request,
and retain the reason and actual model in provenance. Inspect model capabilities
and the compiled style before submitting; never drop lyrics or reinterpret a
recorded job when changing defaults.

The current Stable Audio 3 Medium checkpoint is a genuine exception: its pinned
upstream [prompt guide](https://github.com/Stability-AI/stable-audio-3/blob/77e0398c26f7d22d801ec1bb036307e8b231f741/docs/guides/prompting.md) says it does not produce intelligible vocals. Non-lexical
vocal textures are not lyric support. New instrumentals therefore default to
Stable Audio 3, while lyric requests without an explicit variant use a supported ACE-Step model. Explicit MiniMax Music3 NF4 and HeartMuLa variants use their declared lyric capability. This
is a recorded capability limit, not a permanent rule that only ACE can sing.
Explicit choices must satisfy the requested capability; unsupported choices fail
with their reason instead of silently dropping text or switching engines. A permissive-only deployment refuses the restricted Stable Audio
model explicitly; its Community/Gemma terms are not an MIT/permissive model lane.

For explicit local song alternatives, use the same Style/compose/job/take route.
Select the profile by its declared variant:

| Brief or constraint | Variant and starting controls |
|---|---|
| Preferred harder instrumental launch score | `stable-audio-3-medium`, 8 steps, guidance 1 |
| Current default authored song | A capable ACE-Step variant; inspect its controls |
| Alternate song realism | `minimax-music3-nf4`, 30 steps, guidance 1.7 |
| Alternate permissive song model | `heartmula-3b`, steps 0, guidance 1.5 |

For the two `songgen` variants, put tempo/key intent in the caption.
Leave structured BPM 0 and key empty.
Install Songgen's runtime through `vrooli resource install songgen`.
Acquire the selected weights through
`vrooli resource acquisition data songgen --group PROFILE`.
Use `minimax-music3-nf4` or `heartmula-3b` as PROFILE.
Start Songgen through `vrooli resource start songgen` after the group is acquired.
Ordinary install/start skips missing on-demand groups.
Acquire a retained profile only when its weights are missing or need verification.
Read `path:resources/songgen/README.md` for lifecycle and model-cache operations.
Use `music-tools models list` and `music-tools styles compile STYLE` before admission.
A registry declaration reports `runtime_unverified` until actual runtime evidence exists.
Do not treat it as an installation probe.

The October 17 listener preferred ACE take 2 for the requested pop style and
MiniMax output-head NF4 seed 170104 for song realism. The MiniMax trial requested
UK garage at 128 BPM, so its different vibe does not prove a hard model limit.
Write a new brief for the intended style before comparing again.
Neither integrated song profile accepts a reference voice.
Do not claim native voice matching from a prompt or from the operator's
style comparison.

Write a song arrangement before its lyrics.
Specify short verses, a repeating chorus hook, a contrast section, and instrumental
space in the arrangement.
Use section tags accepted by the selected model.
Fit the words to the intended section durations.
Distinguish singer roles in the brief when a feature is requested.
Treat role and section instructions as soft conditioning until heard in the output.
Retain the exact authored lyrics with the style and receipt.
Review lyric delivery, section changes, instrumental space, genre fit, realism,
and the final ending separately.
A clean decode or high BPM does not measure those qualities.


Resources own model setup, access conditions and GPU lifecycle. Use normal Vrooli
resource management for an unavailable engine, with current capacity and operator
authority; never install packages, create a private loader or evict another tenant
from a generation task. There is no silent ACE or cloud fallback. The direct
`resource-stable-audio-3 content compose` export remains a lower-level diagnostic
surface, documented in `path:resources/stable-audio-3/docs/API.md`.

Retain original source audio, Music job/take identities and the exact resource
receipt with model revision, seed, sampler, runtime and source hash. Stable Audio
takes now have real Music provenance; do not invent ACE headers or use an old
standalone export's operation ID as a Music job ID. Music delivery owns edits.

Compare bounded, level-matched excerpts against the current preferred anchor.
Retain every candidate and exact listener findings. Agent hearing, operator
feedback and technical decode/loudness checks are different evidence. A prompt
requesting no vocals is not proof of vocal absence. Review the full source and
final arrangement before claiming final-film fit. Launch-specific scoring and
review guidance lives in `path:scenarios/prompt-manager/store/skills/packs/core/launch-video/references/audio-review.md`.

## Troubleshooting & Edge Cases

| Symptom | First check and recovery |
|---|---|
| Another music resource occupies the group | Check its active work. Finish it, then stop it through `vrooli resource stop NAME` before starting the selected member. Never stop coding agents. |
| Song resource unavailable or files missing | Inspect `vrooli resource status songgen`. Acquire the exact missing group through the control plane, then start the resource. Do not run a private loader. |
| Identity or receipt mismatch | Keep the failed job evidence. Check declared catalog/artifact/model revisions. Do not switch the queued job to another model. |
| Model disk use grows | Preview the owner's model cache. The 30-day idle policy retains recent downloads; a pin retains a long-lived model. Never delete shards by modification time. |
| Lyrics, voice or style fit is weak | Keep the source and exact listener findings. Revise the brief or audition another declared profile. Do not label an audio file accepted without listening evidence. |

Songgen exits its GPU worker after every generation.
The lightweight supervisor can stay up without a loaded model.
New jobs sweep expired unpinned peer bundles under the acquisition fence.
Resource lifecycle exclusivity and cache ownership are code-enforced primitives;
keep their implementation out of this skill.
Use JSON for saved cleanup previews because the owner checks the exact snapshot.
Model choice and listening judgments remain in this skill because they require
a sound brief and human feedback.
