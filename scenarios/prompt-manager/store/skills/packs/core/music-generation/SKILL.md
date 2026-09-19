---
name: "music-generation"
description: "Generate original, permissively licensed instrumental music locally with ACE-Step 1.5, as a batch of takes to select from, for launch videos and other marketing assets that need a soundtrack with no licence to resolve."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "audio", "music", "video"]
  icon: "music"
  status: "active"
  revision: 1
  createdAt: "2026-09-18T20:00:00Z"
  updatedAt: "2026-09-18T20:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill read", "vrooli capacity policy get"]
  origin:
    kind: "authored"
---
## Tools focus: Music Generation

Generate original instrumental music locally, from permissively licensed weights, as a **batch of takes to choose from**. The output has no licence to resolve, no attribution owed, and no revenue ceiling — which is the whole reason to prefer it over a stock library.

> **This skill is a waypoint, not a destination.** It encodes a throwaway setup that belongs in the `music-tools` scenario as the `ace-step` managed resource its `docs/internal/DECISIONS.md` already specifies. A core-pack skill is the right home *for now*, while the approach is still being learned. When `music-tools` can run a composition operation, retire this skill into it and leave a pointer — do not let a scratchpad recipe become permanent infrastructure.

---

### 1. Boundaries

| In scope | Out of scope |
|---|---|
| Generating instrumental music for a marketing asset | Generating vocals, lyrics, or anything imitating an identifiable artist |
| Producing several takes and selecting among them | Declaring a take final — selection is the operator's |
| Recording model, licence, seeds and prompts as provenance | Publishing or distributing the audio |
| Reporting where the approach hit host limits | Stopping another tenant's service to free the GPU |

**Never** generate music in the style of a named living artist, and never use a copyrighted reference track to condition generation without the operator explicitly deciding that. The licence that makes this approach worth using is the one on the *weights*; conditioning on someone else's recording reintroduces exactly the ambiguity you are here to avoid.

---

### 2. Setup

Everything lives in the session scratchpad; the weights live outside it so they survive.

```bash
SP="<scratchpad>/acestep"; mkdir -p "$SP"; cd "$SP"
uv venv --python 3.12 .venv

# The correct repository is ACE-Step-1.5. `ace-step/ACE-Step` (no suffix) is
# v1 and defaults to a different, older model — installing it is a silent
# 10 GB detour that produces a working pipeline for the wrong thing.
GIT_SSH_COMMAND='ssh -i ~/.ssh/github_ed25519 -o IdentitiesOnly=yes' \
  git clone --depth 1 https://github.com/ace-step/ACE-Step-1.5.git repo

# flash-attn does not declare torch as a build dependency and fails under
# build isolation. It is an optional speedup; sdpa is the fallback.
grep -v '^flash-attn' repo/requirements.txt > req.txt
UV_HTTP_TIMEOUT=300 VIRTUAL_ENV="$SP/.venv" uv pip install -r req.txt

# Weights outside the scratchpad: ~11 GB, and re-downloading them per session
# is the single most wasteful thing this skill can do.
mkdir -p ~/.cache/acestep-spike/checkpoints
ln -sfn ~/.cache/acestep-spike/checkpoints "$SP/repo/checkpoints"
```

Traps, each of which cost a full cycle to find:

- The repo's global git config rewrites `https://github.com/` to SSH, so a plain clone fails with `Permission denied (publickey)`. Pass `GIT_SSH_COMMAND` with the key.
- `uv`'s default 30 s HTTP timeout is not enough for the CUDA wheels.
- **The main bundle ships the 1.7B planner, not the 0.6B.** The planner that fits a ~16 GB card shared with other tenants must be fetched separately:
  ```bash
  .venv/bin/python -c "from huggingface_hub import snapshot_download; \
    snapshot_download('ACE-Step/acestep-5Hz-lm-0.6B', \
      local_dir='$HOME/.cache/acestep-spike/checkpoints/acestep-5Hz-lm-0.6B')"
  ```
- The repo's own `cli.py` is an interactive wizard and its smoke test hardcodes the Apple Silicon MLX backend. Neither is usable headlessly on CUDA. Drive `generate_music()` directly, as §3 does.

**Licence.** `ACE-Step/Ace-Step1.5` is MIT (`verified`, model card front matter). The publisher additionally asserts that output may be used commercially and that training data was licensed, royalty-free, or synthetic — that assertion is `vendor` and has not been independently audited. Record both, at those confidences, and do not upgrade the second one.

---

### 3. The batch runner

Write this to `$SP/batch.py`. It loads the model once and generates many takes, because a model load costs about as much as a take.

```python
#!/usr/bin/env python3
"""usage: batch.py <briefs.json> [variant]"""
import json, os, shutil, sys, time
for p in ("http_proxy","https_proxy","HTTP_PROXY","HTTPS_PROXY","ALL_PROXY"):
    os.environ.pop(p, None)
REPO = os.path.join(os.path.dirname(os.path.abspath(__file__)), "repo")
sys.path.insert(0, REPO)
from loguru import logger
from acestep.handler import AceStepHandler
from acestep.llm_inference import LLMHandler
from acestep.inference import GenerationParams, GenerationConfig, generate_music

briefs = json.load(open(sys.argv[1]))
variant = sys.argv[2] if len(sys.argv) > 2 else "acestep-v15-turbo"
SAVE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", variant)
os.makedirs(SAVE, exist_ok=True)
is_turbo = "turbo" in variant
steps, guidance = (8, 1.0) if is_turbo else (50, 7.0)

dit = AceStepHandler()
msg, ok = dit.initialize_service(project_root=REPO, config_path=variant,
    device="cuda", offload_to_cpu=True, offload_dit_to_cpu=True)
if not ok: sys.exit(f"DiT init failed: {msg}")
llm = LLMHandler()
msg, ok = llm.initialize(checkpoint_dir=os.path.join(REPO, "checkpoints"),
    lm_model_path="acestep-5Hz-lm-0.6B", backend="pt",
    device="cuda", offload_to_cpu=True)
if not ok: sys.exit(f"LLM init failed: {msg}")

for b in briefs:
    out = os.path.join(SAVE, f"{b['label']}.wav")
    if os.path.exists(out):
        continue
    params = GenerationParams(
        task_type="text2music", thinking=True,
        caption=b["caption"], lyrics="[instrumental]",
        duration=float(b.get("duration", 45)), bpm=float(b["bpm"]),
        keyscale=b.get("keyscale", ""),
        inference_steps=steps, guidance_scale=guidance,
        seed=int(b["seed"]), vocal_language="en",
        use_cot_caption=False, use_cot_metas=False,   # see §4 — load-bearing
    )
    t0 = time.time()
    try:
        r = generate_music(dit, llm, params=params,
                           config=GenerationConfig(batch_size=1, audio_format="wav"),
                           save_dir=SAVE)
    except Exception as exc:
        logger.error(f"{b['label']} raised {type(exc).__name__}: {exc}"); continue
    if not r.success:
        logger.error(f"{b['label']} FAILED: {r.status_message}"); continue
    src = r.audios[0].get("path")
    if src and os.path.exists(src): shutil.move(src, out)
    logger.info(f"{b['label']} OK in {time.time()-t0:.1f}s")
```

`briefs.json` is a list of `{label, seed, bpm, keyscale, caption}`. Run it as:

```bash
PYTORCH_ALLOC_CONF=expandable_segments:True .venv/bin/python batch.py briefs.json
```

---

### 4. The parameters that actually decide the result

**`use_cot_caption=False` and `use_cot_metas=False` are the most important lines in this skill.** Left at their defaults (`True`), a planner LM rewrites the caption and invents the metadata *before* the generating model sees anything. Measured at 0.6B, it replaced `bpm: 144` with `bpm: 65`, and a brief reading "cold detuned metallic lead, menacing and restless" with "atmospheric… shimmering synth pads… melancholic arpeggio… ethereal pads for reflection… a surprising melodic piano-like figure". The authored brief never reached the model, and the output was slow and bland for exactly that reason.

The general rule, which outlives this model: **when a generator disappoints, read the log for the prompt that actually reached it before concluding the model is the limit.** Prompt-rewriting front-ends are common in music and image stacks, and a small rewriter restates everything toward the bland middle.

- **Pass structured values in their own fields.** `bpm`, `keyscale`, `duration` are fields. BPM written into the caption text is discarded first by any rewriter and is the reason tempo requests appear to be ignored.
- **`turbo` has no CFG.** `acestep-v15-turbo` is distilled to 8 steps and carries no classifier-free guidance, so `guidance_scale` is inert there and the prompt is a suggestion. `acestep-v15-sft` keeps CFG at 50 steps, where `guidance_scale≈7` is the lever that forces prompt adherence. Start on turbo for speed; move to sft when takes drift from the brief.
- **Seeds are the unit of variation.** Fix the brief, vary the seed, and you get takes of one idea. Vary the brief and you get different ideas. Do both, deliberately.

---

### 5. Writing the brief

**Describe the sound, not the category.** "Modern tech product launch, polished, uplifting, confident" is the *definition* of stock music and will generate exactly that. Those are market positioning words, not sounds.

Name rhythm, bass character and timbre instead: *"booming distorted 808 sub bass with sharp pitch glides, rapid triplet hi-hat rolls, cracking rimshot snare on the three, half-time groove"*. Nouns and patterns beat adjectives and moods.

- **Mine the operator's references for tags, not vibes.** If they supply example tracks, the hosting library's own genre / mood / movement labels are a better prompt vocabulary than anything invented from scratch. Five references sharing **Restless** and **Chasing** tell you more than a paragraph of description.
- **Say what you do not want.** "no piano, no strings" measurably helps; the model reaches for them unprompted.
- **Generate more than you think you need.** A published stock track was selected from a large pool. Judging one generation against one published track is not a fair comparison — it is a comparison of a draft against a winner. Ten takes is a reasonable floor.

---

### 6. Sharing a GPU

Check before assuming you can have the card:

```bash
vrooli capacity policy get      # enforce / preempt_enabled
vrooli capacity reconcile       # who holds what
```

On a host where `enforce = advisory` and `preempt_enabled = false`, the broker **records claims but never actuates them**. A claim larger than the unreserved remainder returns `verdict: queue — waiting for N reclaimable claim(s) to idle`, and that wait never ends, because nothing reclaims. Residency is first-come, first-served regardless of what the ledger says.

Consequences to design around:

- **Shrink yourself before displacing anyone.** `offload_dit_to_cpu=True` fits under ~6 GiB, measured at 3.1× the time (17.9 s → 55.2 s for 45 s of audio). Pay it.
- **Poll for headroom and proceed when it appears.** Other tenants release on their own schedule.
- **Never stop another tenant's service to free VRAM on your own initiative.** A model service holding memory is usually serving something. Ask the operator; it is their host, and the answer is often "wait".
- Register an advisory claim anyway, so `capacity reconcile` does not report you as an unclaimed consumer. Claims need heartbeats or they expire — an expired claim is not a failure, just an absent record.

Record the real footprint you observed. Under-declaring is the most common defect in this system, and it is worth not adding to it.

---

### 7. Delivering

Put the candidates **next to the asset they are for**, not in the scratchpad:

```
<asset folder>/music-candidates/
  t01-<name>.mp3 … t10-<name>.mp3     every take, not just the chosen one
  PROVENANCE.md
```

`PROVENANCE.md` records, at minimum: model and variant, licence and its confidence label, planner LM, every brief with its seed and BPM, measured generation time, and anything that failed or could not be run and why.

Ship **all** the takes, not just the selected one. The operator's next instruction is very often "use a different one" or "more like number 7", and that instruction is free to satisfy if the alternatives are already there and expensive if they are not. Name the one you chose and say why in a sentence; do not hide the rejects.

Report measurements as approximate. Tempo estimated by a single autocorrelation over an onset envelope is subject to half/double-time ambiguity — a 144 BPM half-time trap beat reads as ~72–86. Say so rather than presenting the number as fact, and tell the operator to trust their ears over your table.

---

### 8. Troubleshooting

| Symptom | Likely cause | Check | Fix |
|---|---|---|---|
| Output ignores the brief and sounds generic | A planner rewrote the caption before the model saw it | Read the log for the caption and bpm the model received | `use_cot_caption=False`, `use_cot_metas=False` (§4) |
| Tempo requests are ignored | BPM was written into the caption text, not the `bpm` field | Check whether `bpm` is set as a parameter | Pass it as a field (§4) |
| Output sounds like stock library music | The brief named the category and mood, not rhythm and timbre | Count adjectives versus named sounds in the brief | Rewrite as pattern and instrument nouns (§5) |
| Takes drift from the brief even after the above | Running `turbo`, which has no CFG | Confirm the variant | Switch to `acestep-v15-sft`, `guidance_scale≈7`, 50 steps (§4) |
| `CUDA out of memory`, intermittently | The broker is advisory; residency is first-come, first-served | `vrooli capacity policy get` | `offload_dit_to_cpu=True`, wait for headroom (§6) |
| `Permission denied (publickey)` cloning | Global git config rewrites HTTPS to SSH | `git config --get-regexp insteadOf` | Pass `GIT_SSH_COMMAND` with the key (§2) |
| `flash-attn` fails to build | It does not declare torch as a build dependency | — | Drop it from requirements; sdpa is the fallback (§2) |
| Planner checkpoint not found | The bundle ships 1.7B; 0.6B is a separate download | `ls checkpoints/` | Fetch `acestep-5Hz-lm-0.6B` separately (§2) |
| The 1.7B planner OOMs where 0.6B fits | Expected on a ~16 GB card with co-resident tenants | — | Use 0.6B; this matches the published VRAM tiers |

---

### 9. Output expectations

You may:
- Create the scratchpad environment, the weights cache, and the candidates folder.
- Generate, measure, and select among takes.

You must not:
- Publish or distribute the audio.
- Stop another tenant's service to free the GPU without the operator's word.
- Add the environment to any scenario manifest or lockfile — it is deliberately ungoverned and throwaway. A durable version belongs in `music-tools` (see the note at the top).
- Present the publisher's commercial-use assertion as verified fact.
