# Target-Speaker Extraction

Target-speaker extraction isolates **one enrolled speaker's voice** from a
mixture so downstream speech-to-text transcribes only that person — even when a
second voice or background speech overlaps. It is exposed at `POST /v1/extract`
and consumed by audio-tools as a pre-recognition (ingress) audio stage.

## Approach: separation + ECAPA target-selection

ECAPA-TDNN (the verification model) is an *identity* model — it produces an
embedding, it cannot split a mixture. Extraction therefore composes two steps:

1. **Source separation.** A pretrained SpeechBrain **SepFormer** splits the
   mixture into N candidate source waveforms.
2. **Target selection.** Each separated source is embedded with the *existing*
   ECAPA model and compared (cosine similarity) to the enrolled profile
   embedding. The best-matching source is returned as the cleaned audio, and its
   similarity is reported as `X-Speaker-Score`.

This reuses the embedding model already loaded for verification — no new
identity/conditioning model is required. Alternatives considered: VoiceFilter
and SpeakerBeam (d-vector/embedding-*conditioned* separation). They can be more
accurate for a known target but require a second model and tighter coupling;
separation + ECAPA-select was chosen for v1 because it is additive and reuses
ECAPA. The approach is swappable behind the `/v1/extract` contract.

## Model + sample rate

| Env | Default | Why |
|---|---|---|
| `SPEAKER_EXTRACTION_MODEL` | `speechbrain/sepformer-wsj02mix` | Published 2-speaker SepFormer checkpoint. |
| `SPEAKER_EXTRACTION_SAMPLE_RATE` | `8000` | The 2-speaker SepFormer checkpoints run at 8 kHz; the 16 kHz checkpoints are *enhancement* (single-source denoise), not separation. Audio is resampled 16k→8k for separation and each source 8k→16k for embedding/return. |
| `SPEAKER_EXTRACTION_MATCH_THRESHOLD` | `0.25` | `X-Speaker-Matched` is `score >= threshold`. |

## Status — implemented, pending empirical tuning

The endpoint is fully implemented. What still needs a GPU + real two-speaker
audio (the "spike" the plan calls for) is **empirical tuning**:

- Which checkpoint (`sepformer-wsj02mix` vs `sepformer-libri2mix` vs a whamr
  variant) best isolates the target on real, reverberant, far-field audio.
- The right `SPEAKER_EXTRACTION_MATCH_THRESHOLD` (and whether to fall back to
  the original mixture below it).
- CPU vs GPU latency. SepFormer is heavier than ECAPA; for interactive use the
  GPU compose overlay (`docker-compose.gpu.yml`, `SPEAKER_VERIFICATION_DEVICE=cuda`)
  is recommended. CPU works but adds seconds per window.

## Known limitations (v1)

- **Fixed 2-source separation.** The default checkpoint assumes two speakers;
  3+ overlapping speakers are out of scope for v1.
- **Window-level, not frame-streaming.** audio-tools buffers ~3 s windows and
  extracts each; true frame-by-frame streaming extraction is a non-goal.
- **Speaker-permutation across windows.** Independent per-window separation can
  swap which output index is the target; ECAPA-select per window mitigates this
  (it picks by identity, not index), but boundary artifacts are possible.
