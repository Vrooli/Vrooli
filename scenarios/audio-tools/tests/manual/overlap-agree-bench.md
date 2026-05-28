# OverlapAgree realtime bench

This document captures the realtime characteristics of the rewritten
OverlapAgree algorithm on the actual Whisper resource. The algorithm
is correctness-validated under unit tests
(`api/internal/stt/strategy/overlap_agree_behavior_test.go` and
`overlap_agree_internal_test.go`); this bench validates real-mic
behaviour.

## What the rewrite did

Three coupled changes (see `docs/internal/PROBLEMS.md` "OverlapAgree
commit gap (RESOLVED 2026-05-28)" for full history):

1. **Correctness** — agreement normalizes case + trailing punctuation;
   post-cursor-advance commits use a divergence-free merge; tail flush
   on channel close is unconditional.
2. **Bounded agreement** — agreement walks at most 30 tokens, so
   variance accumulation stays flat regardless of utterance length.
3. **VAD-anchored triggering** — settle attempts fire on silence
   boundaries (frame RMS analysis), not stopwatch ticks. Whisper sees
   clean audio edges. `MaxWindowMs` safety net fallback unchanged.

## Prerequisites

- `vrooli scenario start audio-tools` reports healthy on
  `http://localhost:23345` (UI) and `http://localhost:19630` (API).
- `auto` now resolves to `overlap` for Local Whisper — explicit
  `audio-tools stt stream-config-set --strategy-preference overlap`
  is optional.
- A small wav corpus (existing fixtures are fine — see
  `api/internal/stt/segmenter/testaudio/` for canonical PCM samples,
  or supply your own).

## Bench A — wav corpus

For each wav in your chosen corpus, run:

```bash
audio-tools stt transcribe-stream --file <wav>
# while it runs, in another shell:
vrooli resource logs whisper --tail 200 | grep -E '(concurrent|asr)'
```

Record one row per wav in the table below.

| wav | duration (s) | Segments before Done | reconstructed text == ground truth | per-call p95 latency (ms) | peak Whisper concurrency |
| --- | --- | --- | --- | --- | --- |
| `<fill in>` | | | | | |
| `<fill in>` | | | | | |
| `<fill in>` | | | | | |

### Pass targets (all must hold)

- [ ] **Incremental commits**: Segment events emitted before `Done` ≥ 1
      per ~3 s of speech, on every wav ≥ 6 s. Fewer is a regression
      vs the design goal — back to Phase 2.
- [ ] **WER parity**: Word-error-rate of the reconstructed final text
      (lossless join of `Segment.Text` values, whitespace-normalized)
      is within ±1pp of VADSegment's WER on the **same wav**. Run
      VADSegment baseline by setting
      `audio-tools stt stream-config-set --strategy-preference vad`
      between runs. If OverlapAgree is materially worse, back to
      Phase 2 (likely `maxWindowMs` or `CommitRuns` tuning).
- [ ] **Latency budget**: per-call p95 latency ≤ 2× VADSegment per-call
      p95 on the same hardware. If exceeded: try a shorter
      `MaxWindowMs` (e.g., 15 000) before declaring failure — but
      record both numbers.
- [ ] **Concurrency stays sane**: peak Whisper concurrency at single
      user stays ≤ 1. The growing-buffer scheme calls Whisper less
      often than the old sliding-window scheme (only when accumulated
      uncommitted audio crosses `AdvanceMs`); if concurrency rises,
      raise the `--advance-ms` default and re-test.

## Bench B — live mic walk (web console)

Run on `http://localhost:23345` STT page with
`strategy_preference=overlap`. Tick each scenario after observing the
described behaviour with your own eyes. Brief notes optional but
encouraged.

- [ ] **Slow speech**: speak slowly with deliberate gaps between
      words. **Expect**: incremental Segments appear within ~3 s of
      starting; the final transcript reads correctly.
- [ ] **Fast speech**: speak a 10–15 s monologue without pauses.
      **Expect**: multiple Segments commit mid-monologue (≥ 3 over
      the run); reconstructed text matches what you said.
- [ ] **Long mid-sentence pause**: speak half a sentence, pause 2–3 s
      with mouth-noise / breath / nothing, then continue. **Expect**:
      the first half commits during the pause; the second half
      commits after it; nothing duplicates across the gap.
- [ ] **Two back-to-back utterances**: say "Hello there" then 1 s
      silence then "What is the time". **Expect**: at least two
      committed Segments, no overlap in their text, final text reads
      as the full phrase.

## Notes / observations

```
<paste anything notable here: surprising hallucinations, latency
spikes, console errors, screenshots, anything that informed the
pass/fail decision>
```

## Conclusion

- [ ] All targets in Bench A met across the chosen wav corpus.
- [ ] All four scenarios in Bench B met on live mic.
- [ ] Approved for Phase 4 (default flip to OverlapAgree).

If any box is unticked at sign-off, **do not run Phase 4**. Iterate on
the strategy and re-run this bench.

Bench run by: `<your name>`
Date: `<YYYY-MM-DD>`
Audio-tools commit: `<git rev-parse HEAD>`
Whisper resource image / model: `<docker inspect ... | grep Image>` /
  `<echo $AUDIO_WHISPER_MODEL>`
