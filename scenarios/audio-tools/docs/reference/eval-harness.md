# STT evaluation harness + corpus

The eval harness measures the **accuracy, compute cost, and finalization
latency** of audio-tools' num[sot]:three streaming STT strategies (batch,
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

The num[sot]:two run modes are a first-class contract: the **deterministic** pass feeds
chunks back-to-back and yields reproducible WER + compute (used for
pass/fail); the **real-time-paced** pass releases chunks at 1× audio rate
and yields wall-clock latency, reported only as a p50/p95 distribution over
repeated runs — never gated on a single sample.

> **WER normalization is a documented non-goal of parity** with OpenAI
> Whisper's Python normalizer (number-word expansion, currency/unit rules).
> v1 uses a small local normalizer, so absolute WER is comparable *within*
> the harness (strategy-vs-strategy on the same footing), not against
> published Whisper benchmarks.

## Interpreting the report

`EvalService.RunEval` returns both raw measurements and explanation fields.
Consumers should prefer the backend-owned `summary`, row `verdict`, row
`reasons`, and `warnings` fields over re-deriving strategy ranking in UI or
CLI code.

The default ranking policy is deterministic:

1. lowest WER wins;
2. if WER is effectively tied, lower p95 finalization latency wins when
   latency was measured;
3. if still tied, fewer Whisper calls wins;
4. final tie-break is the stable strategy id.

The report also includes corpus adequacy warnings. Fewer than num[threshold]:10 clips or
less than num[threshold]:120 seconds of evaluated audio is useful for local debugging but
too small to promote a new config without a broader corpus. If
`realtime_repeats=0`, latency is explicitly marked as not measured and the
recommendation is quality/cost-only.

Each `ClipReport` includes the raw reference/hypothesis, the normalized
reference/hypothesis, edit counts, and a word-level alignment path with
`match`, `substitution`, `insertion`, and `deletion` operations. This is the
debug surface for answering why a strategy lost on specific clips.

## Normalization policy

WER normalization lowercases text, removes Unicode punctuation and symbols,
collapses whitespace, and compares whitespace-delimited tokens. That means
capitalization, periods, quotes, most punctuation, and symbol-only differences
do not inflate WER.

Overlap-agree uses a separate agreement policy: each whitespace token is
lowercased and stripped of Unicode punctuation/symbols for comparison only.
The committed text still comes from the first agreeing hypothesis verbatim.
This lets `D.C.` agree with `DC` and leading smart quotes agree with plain
text, while token-boundary changes such as `well-known` versus `well known`
remain strict.

## Workflow

```bash
# 1. Record clips (preferred: the Dictation Studio UI — select a built-in
#    scripted prompt or write a custom/free-form prompt → record → stop into
#    transcribing → correct the transcript to exact words → tag → save), or
#    import an s16le-mono-PCM file + reference from disk:
audio-tools corpus import clip.pcm --reference "the quick brown fox" --tags news,clean
audio-tools corpus list

# 2. Run the comparison (quality only — fast):
audio-tools eval run

# 3. Add finalization-latency measurement (slower; repeated real-time runs):
audio-tools eval run --realtime-repeats 5 --output report.json

# 4. Tune the stall-fallback through the existing stream-config path and
#    re-run to see the latency/WER deltas:
audio-tools stt stream-config-set --overlap-max-stall-rejects 2
audio-tools eval run --realtime-repeats 5
```

A live Whisper backend is required (the harness is an integration tool, not
fast CI). The deterministic fake-provider unit tests in
`api/internal/eval/harness_test.go` carry the default-suite coverage; the
real-Whisper integration test (`harness_integration_test.go`) is
build-tagged (`-tags whisper_integration`) and skipped without a backend.

Dictation Studio's scripted prompt pack is the intended fast-start corpus
builder for local evaluation. The recorder exposes the operator-visible state
sequence `preparing → recording → transcribing → captured` and includes a live
audio-level meter; a stopped turn remains cancellable while waiting for the
terminal transcript. If the meter never moves, treat the resulting no-audio
warning as a capture problem and retry before saving the clip.

## The stall-fallback lever this harness exists to tune

`overlap_max_stall_rejects` (see
[configuration.md](./configuration.md#streaming-stt-control-surface))
force-commits the freshest hypothesis tail after N consecutive
LocalAgreement divergence-rejects, bounding tail growth before the 25s
`max_window_ms` net — the fix for "overlap-agree finalizes slower than
batch". The harness answers, with saved numbers: *does the stall-fallback
lower finalization latency without raising WER beyond an agreed delta?*

Today the harness recommends a winning strategy and explains measured
trade-offs. Mutating stream config remains intentionally outside
`EvalService`; use `audio-tools stt stream-config-set` or the STT admin UI,
which both reuse the validated `STTAdminService.UpdateStreamConfig` writer.
A future bounded sweep RPC should call that same writer for apply semantics
rather than creating a second config mutation path.

## Why the corpus can't ride CI

The audio is personal, machine-local, and git-ignored, so it can't be a
project-level CI regression gate in v1. A *local* baseline-metrics
comparison inside audio-tools is the intended evolution; an exportable
manifest (reference text + tags + metric baselines, audio stays local) is a
stretch follow-up.
