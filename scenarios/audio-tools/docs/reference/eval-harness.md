# STT evaluation harness + corpus

The eval harness measures the **accuracy, compute cost, and finalization
latency** of audio-tools' three streaming STT strategies (batch,
vad-segment, overlap-agree) on *real* audio clips with operator-corrected
ground truth. It exists because the streaming strategies have empirical
quality/latency trade-offs (boundary errors, dropped sections,
slower-than-batch finalization on hard audio) that can only be tuned
against measured numbers, not reasoned about in the abstract.

It mirrors the AI-search eval harness (`packages/ai-go/search/grading.go`
`GradeSuite → SuiteReport`): a corpus of cases, a transcriber seam, and a
per-case + aggregate report.

## The two pieces

1. **Corpus** (`CorpusService`, `api/internal/corpus`) — operator-recorded
   audio clips. Metadata (reference transcript, tags, duration, sample
   rate, format, source) lives in the `corpus` SQLite domain; the audio
   **bytes** live in the blob store under the git-ignored runtime data dir
   (`~/.vrooli/data/.../corpus-blobs`) — never git, never the DB. The blob
   namespace is variant-aware, so a shadow instance's corpus never collides
   with live's.
2. **Harness** (`EvalService`, `api/internal/eval`) — replays each clip
   through each strategy and reports the metrics.

## Metrics

| Metric | Meaning | Reproducible? |
|---|---|---|
| **WER%** | `(S+I+D)/N_ref` on normalized text (lowercase + strip punctuation + collapse whitespace). S/I/D reported separately; deletions surface "dropped sections". | yes |
| **Whisper-calls** | backend calls the strategy made (the metered-provider seam). | yes |
| **Whisper-audio-seconds** | total audio fed to the backend. | yes |
| **RTF** | provider compute-time ÷ audio duration (`<1` = faster than real time). | yes |
| **Finalization latency p50/p95** | wall-clock from the last audio chunk to the terminal transcript, over repeated real-time-paced runs. | **no** (machine-load dependent) |
| **Partial-revisions** | how many times the live partial text changed before committing (stability/jitter). | yes |

Two run modes are a first-class contract: the **deterministic** pass feeds
chunks back-to-back and yields reproducible WER + compute (used for
pass/fail); the **real-time-paced** pass releases chunks at 1× audio rate
and yields wall-clock latency, reported only as a p50/p95 distribution over
repeated runs — never gated on a single sample.

> **WER normalization is a documented non-goal of parity** with OpenAI
> Whisper's Python normalizer (number-word expansion, currency/unit rules).
> v1 uses a small local normalizer, so absolute WER is comparable *within*
> the harness (strategy-vs-strategy on the same footing), not against
> published Whisper benchmarks.

## Workflow

```bash
# 1. Record clips (preferred: the Dictation Studio UI — record → batch-
#    transcribe → correct the transcript to exact words → tag → save), or
#    import an s16le-mono-PCM file + reference from disk:
audio-tools corpus import clip.pcm --reference "the quick brown fox" --tags news,clean
audio-tools corpus list

# 2. Run the comparison (quality only — fast):
audio-tools eval run

# 3. Add finalization-latency measurement (slower; repeated real-time runs):
audio-tools eval run --realtime-repeats 5 --output report.json

# 4. Tune the stall-fallback and re-run to see the latency/WER deltas:
audio-tools stt stream-config-set --overlap-max-stall-rejects 2
audio-tools eval run --realtime-repeats 5
```

A live Whisper backend is required (the harness is an integration tool, not
fast CI). The deterministic fake-provider unit tests in
`api/internal/eval/harness_test.go` carry the default-suite coverage; the
real-Whisper integration test (`harness_integration_test.go`) is
build-tagged (`-tags whisper_integration`) and skipped without a backend.

## The stall-fallback lever this harness exists to tune

`overlap_max_stall_rejects` (see
[configuration.md](./configuration.md#streaming-stt-control-surface))
force-commits the freshest hypothesis tail after N consecutive
LocalAgreement divergence-rejects, bounding tail growth before the 25s
`max_window_ms` net — the fix for "overlap-agree finalizes slower than
batch". The harness answers, with saved numbers: *does the stall-fallback
lower finalization latency without raising WER beyond an agreed delta?*

## Why the corpus can't ride CI

The audio is personal, machine-local, and git-ignored, so it can't be a
project-level CI regression gate in v1. A *local* baseline-metrics
comparison inside audio-tools is the intended evolution; an exportable
manifest (reference text + tags + metric baselines, audio stays local) is a
stretch follow-up.
