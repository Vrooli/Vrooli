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
   through each strategy and reports the metrics. Streaming rows
   (`vad_segment`, `overlap_agree`) route through the production
   `internal/stt/segmenter` path with speaker stages unbound by default,
   so eval exercises the same ingress/selector/egress construction point as
   live STT without reading or mutating the live stream-config row.

## Metrics

| Metric | Meaning | Reproducible? |
|---|---|---|
| **WER%** | `(S+I+D)/N_ref` on normalized text (lowercase + strip punctuation + collapse whitespace). S/I/D reported separately; deletions surface "dropped sections". | yes |
| **Whisper-calls** | backend calls the strategy made (the metered-provider seam). | yes |
| **Whisper-audio-seconds** | total audio fed to the backend. | yes |
| **RTF** | provider compute-time ÷ audio duration (`<1` = faster than real time). | yes |
| **Finalization latency p50/p95** | wall-clock from the last audio chunk to the terminal transcript, over repeated real-time-paced runs. | **no** (machine-load dependent) |
| **Partial-revisions** | how many times the live partial text changed before committing (stability/jitter). | yes |
| **Safety gates** | zero-tolerance committed-text retraction gate plus configurable contiguous dropped-span gate (`default=4` reference words). Reported per row/condition, not averaged into WER. | yes |
| **Commit metrics** | commit count and time-to-first-commit from the strategy/Segmenter event stream. | **no** for wall-clock timing; commit count is reproducible |
| **Length curves** | WER, p95 finalization latency, mean time-to-first-commit, and max dropped span by input-length bucket (10s/30s/1m/3m/5m/>5m). | mixed; quality/drop fields yes, wall-clock latency no |

The num[sot]:two run modes are a first-class contract: the **deterministic** pass feeds
chunks back-to-back and yields reproducible WER + compute (used for
pass/fail); the **real-time-paced** pass releases chunks at 1× audio rate
and yields wall-clock latency, reported only as a p50/p95 distribution over
repeated runs — never gated on a single sample.

Persisted experiments can opt into tail-paced latency with
`--latency-tail-seconds N`. In that mode the worker fast-feeds the prefix of
each clip and paces only the final N seconds at 1×. This is an affordable
final-tail approximation for long-form clips: it measures last-chunk →
terminal transcript behavior without sleeping through minutes of prefix
audio, but it intentionally does not model prefix backlog effects.

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

Persisted experiment reports also include Phase-7 safety fields. The
committed-text timeline is derived from `SegmentEvent`/terminal `Done`
states; a later state must preserve every previously committed normalized
token as a prefix or the retraction gate fails. Dropped-span detection uses
the final word alignment and fails when a contiguous run of deleted reference
words reaches the stored threshold (`audio-tools experiment start
--dropped-span-threshold`, default 4). These gates are deliberately separate
from WER so a single catastrophic omission stays visible even when the
aggregate rate looks acceptable.

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

# 4. Tune overlap-agree experiment knobs hermetically and re-run to see
#    latency/WER deltas without changing live STT settings:
audio-tools eval run --strategies overlap_agree --overlap-max-window-ms 12000 --realtime-repeats 5

# 5. Enqueue a persisted long-form experiment from the same corpus:
audio-tools experiment start --long-form true --target-duration-seconds 180 --gap-ms 5000 --seed 42 --json
audio-tools experiment watch <experiment-id>

# 5b. Measure long-form final-tail latency without pacing the full prefix:
audio-tools experiment start --long-form true --target-duration-seconds 180 \
  --realtime-repeats 2 --latency-tail-seconds 8 --json

# 5c. Populate the metric-vs-length CURVE in one run: a length sweep builds one
#     seeded input per duration, so the report's length buckets get multiple
#     points instead of a single long-form blob collapsing to one bucket:
audio-tools experiment start --sweep-durations 30,60,120,300 \
  --realtime-repeats 2 --latency-tail-seconds 8 --seed 42 --json

# 6. Add deterministic augmentation conditions over the whole realized input:
audio-tools experiment start --long-form true --target-duration-seconds 180 --seed 42 \
  --noise-types white,fan --snr-db 18,12,6 --json

# 7. Add target-speaker extraction / verification ablations over a mixed input:
audio-tools experiment start --long-form true --target-duration-seconds 180 --seed 42 \
  --competing-voices af_bella --snr-db 12 --target-profile-id my-voice \
  --speaker-extraction true --speaker-verification true --speaker-mode filter \
  --dropped-span-threshold 4 \
  --speaker-ablation true --json
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
batch". Experiments can also override `overlap_max_window_ms`, the hard
ceiling on uncommitted tail growth before force-commit. The harness answers,
with saved numbers: *does the stall-fallback or max-window ceiling lower
finalization latency without raising WER beyond an agreed delta?*

Today the harness recommends a winning strategy and explains measured
trade-offs. Mutating stream config remains intentionally outside
`EvalService`; the experiment/eval knobs are a per-run `StreamConfig`
snapshot layered over defaults, not a write to `stt_stream_config`.
A future bounded sweep RPC should call that same writer for apply semantics
rather than creating a second config mutation path.

Persisted experiments can also ask the worker to build a long-form input
from corpus clips before running the same harness. Long-form recipes store
the seed, gap length, target duration, realized clip-id order, assembled
reference transcript, and realized duration. The audio is concatenated in
memory with zero-byte canonical PCM silence gaps, then evaluated as one
synthetic clip; the generated audio bytes are not stored in SQLite or git.

A single long-form input collapses the length curve to one bucket, so to
answer "which strategy wins as messages get **long**" in one run, set
`--sweep-durations <csv>` (proto `long_form.sweep_durations_seconds`). The
worker then builds one seeded input per listed duration — each with a seed
derived from the base seed plus its duration so orderings differ — and the
report's per-strategy length curve (10s/30s/1m/3m/5m/>5m buckets) is populated
with one point per swept duration. A sweep supersedes
`--target-duration-seconds`. With `--realtime-repeats`, this yields the
finalization-latency-vs-length curve the lab exists to produce.

Target-speaker experiments (`--speaker-extraction`/`--speaker-verification`/
`--speaker-ablation`) require `--target-profile-id`; the recipe is rejected at
`StartExperiment` with an actionable message otherwise. Enroll a profile with
`audio-tools stt speaker-enroll --file <clip> --activate true` and list ids via
`audio-tools stt speaker-status`. Extraction only does observable work when the
input actually contains a competing voice, so pair it with `--competing-voices`.
Word loss attributable to the extraction (ingress) stage is filled by comparing
each extraction-on row against its extraction-off ablation sibling, so run with
`--speaker-ablation true` to see ingress attribution in the report.

Experiments can also realize augmentation conditions before replay. The
worker keeps a clean input, then adds generated noise beds (`white`, `fan`,
`percussive`, `music`) and/or Kokoro-synthesized competing voices at the
stored SNR grid. Mixing happens after concatenation, so noise continues
through long-form silence gaps. The recipe stores requested augmentation
fields plus realized condition notes, including resource-down skips for
competing voices. Reports expose one row per strategy and augmentation
condition, such as `batch / clean`, `batch / noise:fan/6db`, and
`batch / competing:voice/12db`; each row owns its WER, compute, latency, and
safety envelope. Speaker ablations compose the same grammar as
`strategy / augmentation-condition / speaker-condition`.

Experiments can also bind target-speaker extraction and egress speaker
verification from a stored recipe. The worker builds a per-run speaker config
snapshot from `target_profile_id`, extraction/verification toggles, mode,
threshold, and fallback settings, then binds the same production adapters used
by live STT into the eval `Segmenter`. With `--speaker-ablation true`, the
same realized inputs are evaluated under extraction/verification off/on
conditions; current reports expose those as condition-suffixed strategy rows,
while Phase 7 promotes attribution and safety envelopes to first-class fields.
The live speaker config cell is not read or written by experiment runs.

Persisted experiments run through `experiment.Manager`, not the request
context, so closing the CLI/UI does not cancel server-side work. The manager
keeps the default worker count at one to avoid stampeding Whisper and other
local resources; queue visibility comes from lifecycle events instead of
parallelism by default. `StreamExperimentEvents` emits `queued; N experiments
ahead` while a run is waiting, updates that message as earlier jobs start or
are canceled, then emits normal running progress (`loading corpus`, eval
steps, `storing report`) and a terminal event. The Dictation Studio lab
subscribes to that stream and falls back to `GetExperiment` polling only when
the browser transport cannot stream.

## Why the corpus can't ride CI

The audio is personal, machine-local, and git-ignored, so it can't be a
project-level CI regression gate in v1. A *local* baseline-metrics
comparison inside audio-tools is the intended evolution; an exportable
manifest (reference text + tags + metric baselines, audio stays local) is a
stretch follow-up.
