# STT evaluation harness + corpus

The eval harness measures the **accuracy, compute cost, and finalization
latency** of audio-tools' num[sot]:three streaming STT strategies (batch,
vad-segment, overlap-agree) on *real* audio clips with operator-corrected
ground truth. It exists because the streaming strategies have empirical
quality/latency trade-offs (boundary errors, dropped sections,
slower-than-batch finalization on hard audio) that can only be tuned
against measured numbers, not reasoned about in the abstract.

## Three quality surfaces — pick by cost and authority

STT quality is observable at three tiers of increasing cost and authority.
Do not conflate them:

| Surface | Question it answers | Cost | Authority |
|---|---|---|---|
| **STT readiness** (diagnostics, layer 1) | Does the provider chain accept and process audio at all? | ~100 ms, one bundled tone | Reachability only — **no** transcript claim |
| **STT quality smoke** (diagnostics, layer 2) | Does no-speech stay silent, and does a known clean clip transcribe roughly right? | sub-second, two tiny bundled fixtures | Regression tripwire — catches gross breakage, not accuracy grading |
| **Dictation Studio / eval harness** (this doc) | What is the real WER/latency/safety of each strategy on a real corpus? | seconds–minutes, operator corpus | The promotion-grade quality authority |

The diagnostics **readiness** step proves ASR reachability by sending a
bundled smoke tone through the provider chain; it makes no transcript
claim (the CLI/UI mark it `quality not assessed` when the layer-2 smoke is
not configured).

The diagnostics **quality-smoke** layer (`api/internal/diagnostics/quality_smoke.go`)
drives two tiny bundled fixtures — a digital-silence clip and a short
clean-speech clip ("The quick brown fox jumps.") — through the *same* STT
chain and the *same* shared egress policy (`internal/stt/quality`) that
user-facing transcription uses. It grades two things:

- **No-speech safety (hard gate):** the silence fixture's user-facing
  transcript must be empty after the egress policy. A surviving transcript
  is a hallucination leak (the classic "Thanks for watching!") and **fails**
  the STT step — readiness stays reported as reachable so operators see the
  fault is transcript safety, not provider reachability.
- **Clean-speech WER (soft gate):** the speech fixture is graded by
  normalized WER (reusing `internal/eval`'s normalizer/WER) against a lenient
  threshold (`cleanSpeechWERThreshold`, default `0.34`). Over-threshold drift
  **warns** but never fails — a health check should not go red on model
  quality jitter. When it warns, run a Dictation Studio experiment for
  corpus-grade grading.

Quality smoke deliberately does **not** run corpus sweeps, strategy
promotion, microphone capture, or multi-minute sleeps — those remain this
harness's job. Use quality smoke to catch obvious regressions fast; use this
harness for any real quality claim.

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
2. **Harness** (`api/internal/eval`) — internal replay/report machinery used
   by persisted experiments. Streaming rows (`vad_segment`, `overlap_agree`)
   route through the production `internal/stt/segmenter` path with per-run
   speaker/config snapshots, so experiments exercise the same
   ingress/selector/egress construction point as live STT without reading or
   mutating the live stream-config row.

### Provider cells

`ExperimentRecipe.cells` is the provider-neutral execution surface. A
`kyutai:passthrough` cell creates a fresh Kyutai adapter and routes canonical
PCM through the production Segmenter rather than assigning a Kyutai label to a
Whisper result. Dictation Studio exposes this cell as the `passthrough`
choice. Unknown engines fail before a run is persisted.

Current scope is deliberately explicit: deterministic and full-real-time
native-streaming cells run through their declared provider, including
materialized long-form duration sweeps. Product-path, fault-profile,
policy-profile, augmentation, and speaker-ablation cells refuse server-only
execution rather than being relabelled as completed evidence. Product-path
evidence must come from the browser qualification harness; fault and policy
profiles require their corresponding harnesses before they can contribute to
promotion.

Provider-cell reports also carry a provider-scoped `promotion_verdict`. It
uses the shared trust-floor rubric and is intentionally conservative: only a
full-real-time duration sweep can credit the duration profiles it actually
reaches. When an experiment report is retrieved, compatible succeeded reports
from the same recorded host/OS/architecture contribute their earned real-time
duration rungs, so the 30-second, 1-minute, 5-minute, 15-minute, and 60-minute
ladder can be built across persisted runs. Deterministic replay remains quality
evidence, never a substitute for a real-time long-form turn. A cell declares
one lane: a `realtime` cell derives its WER and safety result from its first
paced repeat, while an explicitly `deterministic` cell derives those fields
from its unpaced replay. The runner never attaches an unpaced failure to a
report labeled `realtime`.
Recovery, fault, browser-product-path, and device categories remain explicit
blocking reasons until their dedicated qualification artifacts are persisted.
An `interval_accounting` artifact is also stricter than its name implies: it
must include `all_intervals_accounted: true`,
`duplicate_committed_segments: 0`, and `silent_terminal_outcomes: 0` in its
machine provenance before a passing record can credit promotion. This prevents
a duration or WER result from implicitly asserting no-loss delivery semantics.
A strategy recommendation is therefore never a stable-engine promotion by
itself.

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
| **Scaling analysis** | One raw point per evaluated duration plus backend-owned `flat` / `linear` / `superlinear` / `inconclusive` classifications, fitted model stats, units, and log-log growth exponents for every registered metric. | mixed; compute fields yes, wall-clock latency no |

The two run modes are a first-class contract. An explicit **deterministic**
cell feeds chunks back-to-back and yields reproducible WER + compute. A
**real-time-paced** cell releases chunks at 1× audio rate; its first paced
repeat supplies WER/safety and all repeats supply the latency distribution.
This keeps native streaming providers from being judged by an incompatible
unpaced transport while retaining a separately requestable deterministic lane.

Persisted experiments can opt into tail-paced latency with
`--latency-tail-seconds N`. In that mode the worker fast-feeds the prefix of
each clip and paces only the final N seconds at 1×. This is an affordable
final-tail approximation for long-form clips: it measures last-chunk →
terminal transcript behavior without sleeping through minutes of prefix
audio, but it intentionally does not model prefix backlog effects.
Reports produced this way include a `tail_latency_approximation` warning; use
full real-time repeats before treating latency scaling as a promotion gate.

Persisted experiments expose the per-run stream tuning levers as typed CLI
flags, so sweeps do not need hand-written recipe JSON. `--overlap-max-window-ms`,
`--overlap-max-stall-rejects`, `--overlap-window-ms`, and
`--overlap-commit-runs` apply only to `overlap_agree` rows.
`--vad-silence-ms` applies only to `vad_segment` rows. Passing
`--overlap-max-stall-rejects 0` explicitly disables the stall fallback; omitting
it keeps the stream-config default.

> **WER normalization is a documented non-goal of parity** with OpenAI
> Whisper's Python normalizer (number-word expansion, currency/unit rules).
> v1 uses a small local normalizer, so absolute WER is comparable *within*
> the harness (strategy-vs-strategy on the same footing), not against
> published Whisper benchmarks.

## Interpreting the report

Persisted experiment reports return both raw measurements and explanation
fields. Consumers should prefer the backend-owned `summary`, row `verdict`,
row `reasons`, and `warnings` fields over re-deriving strategy ranking in UI
or CLI code.

The default ranking policy is deterministic and safety-gated:

1. a row that passes the safety gates always ranks ahead of a row that fails;
2. among rows with the same safety outcome, lowest WER wins;
3. if WER is effectively tied, acceptable long-form scaling wins over a
   superlinear row when scaling was measured;
4. if still tied, lower p95 finalization latency wins when
   latency was measured;
5. if still tied, fewer Whisper calls wins;
6. final tie-break is the stable strategy id.

If every evaluated row fails the safety gates, the report still selects a
deterministic least-bad diagnostic row, but marks it
`recommended-despite-safety-failure` and emits a safety warning. Treat that
state as a corpus/config debugging result, not as a promotion recommendation.

The report also includes corpus adequacy warnings. Fewer than num[threshold]:10 clips or
less than num[threshold]:120 seconds of evaluated audio is useful for local debugging but
too small to promote a new config without a broader corpus. If
`realtime_repeats=0`, latency is explicitly marked as not measured and the
recommendation is quality/cost-only.

Duration sweeps add a `scaling` object to each strategy row. `scaling.points`
keeps the raw realized-duration measurements: WER, finalization p50/p95 and
sample count, time-to-first-commit, commits, partial revisions, dropped-span
max, Whisper calls, backend audio seconds, provider latency, and RTF. The
backend fits constant, linear, n-log-n, and quadratic models over positive
durations, then adds a weighted log-log fit that reports `exponent` and
`exponent_r_squared`. `metric_fits` contains one fit per registered metric:
finalization p95, time-to-first-commit, WER, max dropped span, provider
latency, Whisper calls, backend audio seconds, and RTF. At least
num[threshold]:3 distinct positive durations are required to fit; otherwise
the row gets `insufficient_scaling_points` and remains `inconclusive`.
Superlinear classification is stricter: it requires at least
num[threshold]:4 distinct durations, an exponent above the linear margin, a
useful fit, and adequate latency samples. Single-sample latency evidence caps
confidence low. These classes are empirical for the measured machine/corpus,
not formal Big-O claims. The compute classification uses the highest-risk fit
across provider latency, Whisper call count, processed audio seconds, and RTF;
`compute_fit.metric` names the metric that drove the aggregate classification.

When WER is tied or effectively tied, the recommendation prefers a strategy
with acceptable long-form scaling over a lower short-clip p95 row that shows
superlinear growth. If no scaling data is present, the older aggregate ranking
policy is preserved. UI and CLI surfaces render the backend-owned
classification and warnings; they should not fit curves or re-rank rows
client-side.

Each `ClipReport` includes the raw reference/hypothesis, the normalized
reference/hypothesis, edit counts, and a word-level alignment path with
`match`, `substitution`, `insertion`, and `deletion` operations. This is the
debug surface for answering why a strategy lost on specific clips.

Persisted experiment reports include safety fields. The committed-text
timeline is derived from `SegmentEvent`/terminal `Done` states; a later state
must preserve every previously committed normalized token as a prefix or the
retraction gate fails. Dropped-span detection uses the final word alignment and
fails when a contiguous run of deleted reference words reaches the stored
threshold (`audio-tools experiment start --dropped-span-threshold`, default 4).
These gates are deliberately separate from WER so a single catastrophic
omission stays visible even when the aggregate rate looks acceptable.

## Experiment JSON contract

`audio-tools experiment --json` uses proto JSON with snake_case field names.
Unlike the default CLI proto renderer, the experiment surface emits
zero/default scalar values so best-case metrics remain explicit: a perfect
condition includes fields such as `"wer": 0`,
`"finalization_latency_p95_ms": 0`, `"partial_revisions": 0`,
`"estimated_seconds": 0`, and zero deltas. Unset message fields are still
omitted rather than emitted as `null`.

Envelope shapes are command-specific:

| Command | JSON envelope |
|---|---|
| `experiment start` | `{experiment, estimated_seconds}` |
| `experiment get` | `{experiment, runs}` |
| `experiment wait` | terminal with report available: `{experiment, report, runs}`; otherwise `{experiment, runs}` |
| `experiment report` | `{experiment, report, runs}` |
| `experiment list` | `{experiments}` |
| `experiment compare` | `{experiments:[{experiment, report}]}` |
| `experiment delete` | `{id, deleted_report}` |

`report` is the canonical structured metrics payload and uses snake_case
proto fields throughout. Experiment run rows persist their per-cell
`condition_json` identity only; they do not carry a second metrics blob, so new
automation should read metrics from `report`. `experiment.machine_json` stores
server-side provenance captured at submit time, such as host/runtime and recipe
digest information; it is not a metrics source.

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

# 2. Enqueue a persisted quality-only comparison over the corpus:
audio-tools experiment start --strategies batch,vad_segment,overlap_agree --json
audio-tools experiment wait "<experiment-id>" --json

# Optional: stream server-owned progress while the worker runs. Canceling this
# client does not abort the experiment.
audio-tools experiment watch "<experiment-id>"

# 3. Add finalization-latency measurement (slower; repeated real-time runs):
audio-tools experiment start --strategies batch,vad_segment,overlap_agree --realtime-repeats 5 --json
audio-tools experiment report "<experiment-id>"

# 4. Tune overlap-agree experiment knobs hermetically and compare reports to see
#    latency/WER deltas without changing live STT settings:
audio-tools experiment start --strategies overlap_agree --overlap-max-window-ms 12000 --realtime-repeats 5 --json
audio-tools experiment compare "<baseline-id>,<tuned-id>"

# 5. Enqueue a persisted long-form experiment from the same corpus:
audio-tools experiment start --long-form true --target-duration-seconds 180 --gap-ms 5000 --seed 42 --json
audio-tools experiment watch "<experiment-id>"

# 5b. Measure long-form final-tail latency without pacing the full prefix:
audio-tools experiment start --long-form true --target-duration-seconds 180 --realtime-repeats 2 --latency-tail-seconds 8 --json

# 5b-2. Sweep overlap-agree tail-latency knobs hermetically from flags:
audio-tools experiment start --strategies overlap_agree --overlap-max-stall-rejects 3 --overlap-window-ms 2000 --overlap-commit-runs 2 --realtime-repeats 2 --latency-tail-seconds 8 --json

# 5c. Populate the metric-vs-length CURVE in one run: a length sweep builds one
#     seeded input per duration, so the report's length buckets get multiple
#     points instead of a single long-form blob collapsing to one bucket:
audio-tools experiment start --sweep-durations 30,60,120,300 --realtime-repeats 2 --latency-tail-seconds 8 --seed 42 --json

# 6. Add deterministic augmentation conditions over the whole realized input:
audio-tools experiment start --long-form true --target-duration-seconds 180 --seed 42 --noise-types white,fan --snr-db 18,12,6 --json

# 7. Check/enroll speaker profiles, then add target-speaker extraction /
#    verification ablations over a mixed input:
audio-tools stt speaker-status --json
audio-tools experiment start --long-form true --target-duration-seconds 180 --seed 42 --competing-voices af_bella --snr-db 12 --target-profile-id my-voice --speaker-extraction true --speaker-verification true --speaker-mode filter --dropped-span-threshold 4 --speaker-ablation true --json

# 8. Delete terminal experiment artifacts when they are no longer useful.
#    Running/queued experiments must be canceled or allowed to finish first.
audio-tools experiment delete "<experiment-id>" --yes
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
LocalAgreement divergence-rejects, bounding tail growth before the 10s
`max_window_ms` net — the fix for "overlap-agree finalizes slower than
batch". Experiments can also override `overlap_max_window_ms`, the hard
ceiling on uncommitted tail growth before force-commit. The harness answers,
with saved numbers: *does the stall-fallback or max-window ceiling lower
finalization latency without raising WER beyond an agreed delta?*

Today persisted experiment reports recommend a winning strategy and explain
measured trade-offs. Mutating stream config remains intentionally outside the
experiment lab; the experiment/eval knobs are a per-run `StreamConfig`
snapshot layered over defaults, not a write to `stt_stream_config`. A future
bounded sweep/apply operation should call the existing STT config writer for
apply semantics rather than creating a second config mutation path.

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
For sweep runs, the worker serializes real-time repeats internally
(`realtime_concurrency=1` in the realized metadata) to reduce backend
contention between duration points. Ordinary non-sweep latency experiments
keep the default bounded concurrency so UI-triggered runs remain practical.

Target-speaker experiments (`--speaker-extraction`/`--speaker-verification`/
`--speaker-ablation`) require `--target-profile-id`; the recipe is rejected at
`StartExperiment` with an actionable message otherwise. Enroll a profile with
`audio-tools stt speaker-enroll --file <clip> --activate true` and list ids via
`audio-tools stt speaker-status` or `audio-tools stt speaker-status --json`.
Extraction only does observable work when the input actually contains a
competing voice, so pair it with `--competing-voices`. Word loss attributable
to the extraction (ingress) stage is filled by comparing each extraction-on row
against its extraction-off ablation sibling, so run with
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
conditions and reports expose those as condition-suffixed strategy rows. If a
condition requests extraction or verification while the speaker resource is not
configured or not ready, that condition is marked skipped with a report warning
instead of producing a clean green zero-effect row. Ingress attribution is only
computed when both sibling cells actually ran. The live speaker config cell is
not read or written by experiment runs.

Persisted experiments run through `experiment.Manager`, not the request
context, so closing the CLI/UI does not cancel server-side work. The manager
keeps the default worker count at one to avoid stampeding Whisper and other
local resources; queue visibility comes from lifecycle events instead of
parallelism by default. `StreamExperimentEvents` emits `queued; N experiments
ahead` while a run is waiting, updates that message as earlier jobs start or
are canceled, then emits normal running progress (`loading corpus`, eval
per-condition/per-strategy/per-clip quality and latency steps,
`storing report`) and a terminal event. The `StartExperiment` response carries
a server-computed `estimated_seconds` value from realized clip duration,
strategy count, condition count, and repeat settings unless the client
explicitly supplies an override. The manager also enforces a runtime budget:
the estimate plus 25% headroom (at least two minutes), or 30 minutes when a
duration cannot be estimated. Exceeding that budget is a terminal **failed**
experiment with a persisted `runtime budget` error, not an endlessly-running
job; explicit operator cancellation remains `canceled`. The Dictation Studio lab
subscribes to that stream and falls back to `GetExperiment` polling only when
the browser transport cannot stream.

## Why the corpus can't ride CI

The audio is personal, machine-local, and git-ignored, so it can't be a
project-level CI regression gate in v1. A *local* baseline-metrics
comparison inside audio-tools is the intended evolution; an exportable
manifest (reference text + tags + metric baselines, audio stays local) is a
stretch follow-up.
