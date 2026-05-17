# Streaming STT — Boundary-Accuracy Tuning

The local STT pipeline (`VADSegmenter` over the Whisper sidecar) trades
end-to-end latency for transcript quality on a small set of operator-
tunable knobs. Defaults bias toward accuracy without sacrificing
streaming snappiness; operators can dial either direction.

## Levers

| Field (pipeline.Config) | Default | Range | Effect |
|---|---|---|---|
| `SegmentSilenceMs` | 2500 | 800–3000 | Minimum sustained silence that closes a segment. Lower = snappier, more frequent cuts. Higher = longer phrases per segment, fewer Whisper calls. |
| `PreRollMs` | 300 | 0–800 | Audio carried from the trailing edge of segment N into the leading edge of segment N+1. Restores Whisper's pre-word context so the first phoneme is not clipped. |
| `TrailingPadMs` | 200 | 0–600 | Real silence kept past the last voiced frame. Whisper handles natural word-tail decay better than a hard cut. |
| `InitialPromptWords` | 20 | 0–50 | Count of last-words from segment N forwarded as Whisper's `initial_prompt` for segment N+1. Provides text left-context across boundaries. Capped at 50 to stay well below the 224-token prompt limit. |
| `OverlapBytes` | 8192 | 0–16384 | Legacy byte-level overlap (kept for backwards-compat). Prefer `PreRollMs` for new operators. |

## Why pre-roll + trailing pad

Before these levers, each segment was cut at the start of the silence run
and the next segment began on the very next byte. Whisper had zero
pre-word audio and zero text context, so word *onsets* and word *codas*
at segment boundaries were routinely clipped or hallucinated. Pre-roll
restores the audio context; trailing pad restores the decay; the rolling
`initial_prompt` restores the text context.

## Why dedup is automatic

`PreRollMs > 0` means segment N+1 *will* re-transcribe the last
~300 ms of segment N — Whisper will emit some of segment N's words again.
The strategy deduplicates against a per-session committed string via
`pipeline.DeduplicateOverlap` before emitting `SegmentEvent.Text`, so
consumers never see the repeats. No client-side dedup is needed.

## Model-size selection

The local Whisper sidecar's model size is picked by the whisper
resource's `whisper recommend-model` CLI based on detected
GPU VRAM / CPU cores / system RAM, capped by
`WHISPER_RESOURCE_BUDGET_PCT` (default 50). Operators can pin a model
via `WHISPER_DEFAULT_MODEL=<size>`. See
`resources/whisper/docs/OPERATIONS.md` for the table.

`LocalProvider.Model()` and `Result.ModelID` now report the actually-
loaded model via the `whisperinfo.Client` seam — no more fabricated
`whisper-large-v3` regardless of what is actually running.
