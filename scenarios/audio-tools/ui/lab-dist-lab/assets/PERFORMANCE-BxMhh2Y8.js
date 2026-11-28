const e=`# Performance — Audio Tools

This document owns performance interpretation and regression procedure. The
[pilot decision sheet](TESTING.md#pilot-decision-sheet-and-safe-first-slice) owns
the adopted product SLOs; experiment and test owners retain measured receipts.


Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| API health | responsive under lifecycle health timeout | \`/health\` check | active |
| UI health | responsive under lifecycle health timeout | \`/health\` check | active |
| Warm click to capture/session ready | p95 ≤ 500 ms; maximum ≤ 1,500 ms | Button event to usable capture and admitted stream, permission already granted and model resident | adopted, qualification unknown |
| Warm speech to first visible partial | p95 ≤ 1,000 ms; maximum ≤ 2,000 ms | First voiced sample position to rendered interim text | adopted, qualification unknown |
| Continuing visible update gap | p95 ≤ 750 ms; maximum ≤ 2,000 ms during voiced speech | Consecutive rendered revisions; record silence/VAD separately | adopted, qualification unknown |
| Committed transcript backlog | p95 ≤ 2,000 ms; maximum ≤ 5,000 ms | Captured position minus committed coverage position | adopted, qualification unknown |
| Stop to final tail | p95 ≤ 1,000 ms; maximum ≤ 2,000 ms | Stop event through known ending phrase and completed drain | adopted, qualification unknown |
| Expired discovery cache | Warm speech and stop bounds; readiness p95 ≤ 1,000 ms, maximum ≤ 2,000 ms | Inventory refresh must not block admission; capability checks still fail closed | adopted, qualification unknown |
| Installed cold engine | Readiness p95 ≤ 10 s; maximum ≤ 20 s | Engine stopped with artifact present; show preparation immediately | adopted, qualification unknown |

\`startToCaptureReadyMs\` starts when the shared consumer accepts
\`startRecording()\` and ends only after both the capture graph exists and the
stream has reached its ready handshake. It therefore includes microphone
permission/device acquisition, journal restore, capture-module setup, and
stream admission when those occur after the user action. The shared hook passes
its monotonic start timestamp into the provider; direct provider users fall
back to the provider \`start()\` boundary and must not claim button-to-ready
timing. This is readiness, not first-speech or first-text latency. Realtime
qualification requires p95 ≤ 500 ms and maximum ≤ 1,500 ms; a missing start,
capture, or connection milestone yields no measurement and cannot pass the
claim.

PRD OT-P0-005 makes responsive streaming a required product outcome. The
TESTING.md decision sheet has adopted these bands; adoption sets targets, not
passing results. The missing-model provisioning limit, concurrency, and session
retention limits remain separate open policy decisions. Do not duplicate or
silently relax the adopted bands here.

### Clock and denominator contract

Use a monotonic clock within each process. Correlate cross-process events by
session and interval identity; do not subtract unsynchronized browser/server
wall clocks. The browser owns click-to-ready and speech-to-visible-text timing.
Record both microphone-ready and provider/session-ready; the user-ready boundary
is when capture can safely accept the intended input with an honest UI state.

Measure speech-to-first-visible-partial from the first voiced input sample, and
stop-to-final-tail from stop action to the final committed text covering the
last accepted input. Track partial revision cadence and commit-position lag
throughout sustained speech, not only the first event. A final returned at EOF
does not count as a streaming partial.

Use the browser's monotonic clock for within-page durations. The current
metadata-only diagnostic detects signal by a PCM amplitude threshold; it
backdates that callback observation by the threshold-crossing sample's offset
within the delivered capture block. This is a sample-aligned signal proxy, not
an acoustic ground-truth voice annotation, and it does not replace the paced
corpus/onset reference used for acceptance.

The first-partial diagnostic also records the canonical input sample where that
signal threshold first crossed, the captured sample cursor when the interim
became visible, the post-onset audio duration available by that point, and the
difference between that audio span and wall-clock speech-to-visible latency.
That difference estimates work/transport/render lag after accounting for
captured speech duration; it is not a server-only decode timer. Use it with the
processed cursor and provider/model identity to distinguish slow arrival from
model context accumulation before changing model or chunk settings.

Report p50/p95, attempts, successes, failures, timeouts, maximum and sample count
for each route/profile/cold-warm cohort. Latency percentiles over successful
attempts must be labeled as such and paired with the full failure denominator;
timeouts remain violations or incomplete attempts, never omitted successes.
Separate permission prompts, model provisioning, queueing and steady-state
processing. No fleet availability percentage is established by a small soak.

Continuing visible-update gaps are recorded after the Dictation Studio React
layout commit for each changed, non-empty transcript value. They measure
consecutive consumer-visible revisions, not provider callback cadence. Each
turn exports the rendered revision count and bounded gap array; the realtime
soak requires at least two rendered revisions, an exact gap-count-to-revision
count match, and finite non-negative values on every turn. Missing, malformed,
or incomplete arrays fail closed. Silence/VAD intervals are not counted as
revisions and remain separately interpreted.

Lifecycle responsiveness is not voice readiness. Build duration is a tooling
measurement, not permission to spend minutes starting a dictation session.

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Cold / subsequent engine discovery | 10,421 ms / 3 ms in the inspected pair | Dated investigation in PROBLEMS.md; two samples, not a percentile | 2026-09-08 |
| Streaming file first partial | 936 ms, then 13,929 ms on a repeat | Dated investigation in PROBLEMS.md; unpaced file input, not browser microphone latency | 2026-09-08 |
| Product first partial and post-onset audio span | 1,096.7 ms wall; 1,094.9 ms audio available; 1.7 ms excess; stop-to-final-tail 87.3 ms | One headless Chrome fake-microphone trace on Sherpa Zipformer; [receipt](../../coverage/browser-first-partial-context-20261123.json), not a percentile | 2026-11-23 |
| Paced product partial diagnostic, visible updates, and final tail | n=30; signal-to-partial-diagnostic p50/p95/max 1,098.3/1,161.2/1,195.9 ms; post-onset audio p95 1,117.0 ms; excess p95 55.5 ms; update-gap p95 352.1 ms; stop-tail p95 165.8 ms; all 30 terminal sample cursors reconciled | Headless Chrome fake microphone through Dictation Studio and local Sherpa Zipformer; [receipt](../../coverage/browser-paced-sherpa-30-20261123.json). The receipt does not record first-visible timing separately. Synthetic cohort; diagnostic p95 exceeds the 1,000 ms target, and this is not device qualification | 2026-11-23 |
| Paced product partial diagnostic on the repo fixture | n=30; signal-to-partial-diagnostic p50/p95/max 771.1/801.1/810.4 ms; click-to-first-visible p50/p95/max 830.6/868.3/989.9 ms; stop-tail p95 143.3 ms; visible-update-gap p95 353.0 ms; all 30 turns reconciled captured/committed samples and sent/processed cursors | [Dependency-free Chrome DevTools runner](../../scripts/browser-paced-dictation.mjs) and [receipt](../../coverage/browser-paced-2026-11-24T052319602Z.json). Uses the 1.954-second \`dictation-reference.wav\` fixture (SHA-256 \`0c789111bdca878622c2db7f8097f7597dd42ee3eb6a1aefee416ef2504bef7d\`), so it is not directly comparable with the 11-second JFK cohort above. Synthetic media only; no physical-device or independent speech-quality claim | 2026-11-24 |
| Paced product 11-second JFK rerun | n=30; signal-to-first-visible p50/p95/max 1,108.7/1,122.8/1,157.1 ms; signal-to-partial-diagnostic p95 1,120.7 ms; click-to-first-visible p95 1,199.7 ms; stop-tail p95 108.4 ms; all 30 turns reconciled captured/committed samples and sent/processed cursors with visible partial, final, and update-gap evidence | [Runner](../../scripts/browser-paced-dictation.mjs) and [receipt](../../coverage/browser-paced-sherpa-jfk-rerun-2026-11-24.json); same 11-second fixture SHA-256 as the 2026-11-23 cohort. Compared with the prior diagnostic p95 of 1,161.2 ms, this run is 40.5 ms lower, but one cohort cannot establish causality or a passed 1,000 ms target. Synthetic media only; not speech-quality or device qualification | 2026-11-24 |
| Paced product 11-second JFK current-build repeat | n=30; signal-to-first-visible p50/p95/max 1,105.3/1,122.1/1,130.5 ms; signal-to-partial-diagnostic p95 1,107.9 ms; stop-tail p95 84.5 ms; update-gap p95 343.7 ms; all 30 turns reconciled captured/committed samples and sent/processed cursors with visible partial, final, and update-gap evidence | [Runner](../../scripts/browser-paced-dictation.mjs) and [receipt](../../coverage/browser-paced-sherpa-jfk-current-build-2026-11-24.json); same 11-second fixture SHA-256 and local Sherpa route. The p95 is 0.7 ms lower than the previous rerun, an immaterial difference; first-partial latency remains above the adopted 1,000 ms target. Synthetic media only; not speech-quality or device qualification | 2026-11-24 |
| Paced native first text on the 11-second JFK fixture | 310 MB Zipformer: 1,200 ms captured audio / 1,217 ms wall in the initial comparison, then 1,200 ms captured audio / 1,217–1,221 ms wall across five repeated runs; 20M Zipformer: 2,100 / 2,107 ms; 68 MB Kroko: 1,500 / 1,534 ms. The five repeated finals all failed the checked JFK opening and returned the same substitution. | Sherpa v1.13.2 native smoke with 100 ms PCM packet cadence; [initial model comparison](../../coverage/native-sherpa-model-first-text-20261124.json). The repeat is a diagnostic run, not a retained qualification receipt or percentile. It confirms model-context latency and a fixture-specific quality failure, but does not qualify product latency, general model quality, memory, or commercial license. See [the dated finding](PROBLEMS.md#2026-11-24--fixed-input-sherpa-latency-and-jfk-recognition-remain-model-bound). | 2026-11-24 |
| Native Sherpa thread and int8-encoder/joiner probe on the 11-second JFK fixture | At 1/2/4 threads, first text stayed at 1,200 ms captured audio; wall times were 1,223.1/1,218.8/1,216.5 ms. Transcripts were identical and failed the checked JFK opening. The int8 encoder/joiner reduced feed-call time (450 ms vs 584 ms at 2 threads) but still emitted first text at 1,200 ms and failed the opening. | Same pinned Sherpa v1.13.2 native smoke, fixture, 100 ms cadence, suffix assertion; one run per setting, not a product cohort or quality qualification. Parallelism and weight quantization reduce computation but do not change the observed first-text audio threshold; no default change is supported. See the dated finding in [PROBLEMS.md](PROBLEMS.md#2026-11-24--thread-count-and-int8-do-not-close-sherpa-first-text-latency). | 2026-11-24 |
| Native Sherpa smoke after per-stream PCM scratch reuse | On the 310 MB Zipformer and same 11-second JFK fixture, first text remained at 1,200 ms captured audio / 1,217 ms wall; total model-feed call time was 571 ms. The expected final suffix passed, but the output still substituted the JFK opening. | Rebuilt cgo test against pinned Sherpa v1.13.2 runtime with current resource model/punctuation artifacts and 100 ms packets; \`TestNativeStreamingModelSmoke\` passes with \`SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX='your country.'\`. One native adapter run, not a product/browser p95, quality qualification, or proof of a latency improvement. | 2026-11-24 |
| Native short-fixture thread-count probe | One 1.955-second fixture attempt each at 1/2/4 threads; first text stayed at 800 ms captured audio and 827.2/816.3/816.1 ms wall; model-feed call wall time was 132.4/98.0/91.6 ms | [Diagnostic receipt](/home/matthalloran8/.vrooli/plan-artifacts/audio-tools-portable-voice-development/sherpa-thread-count-diagnostic-20261124.json). Same temporary Sherpa model and runtime files, but identity is not verified against the resource pin. Single samples; synthetic/reference fixture; not a product cohort, percentile, or quality result. No thread default change is supported | 2026-11-24 |
| Concurrent native CPU streams at 2 and 4 sessions | Two streams: 3 runs including race; four streams: 3 runs including race. Every stream emitted first text after 1,200 ms captured audio; four-stream wall range was 1,218.1–1,223.9 ms across normal/race runs. All stream transcripts passed final-tail and same-fixture lexical parity checks. The four-stream active \`AcceptPCM\`/engine-close test also passed under \`-race\`, with no Go race report. A separate four-stream smoke peaked at 487 MiB Go test process-tree RSS normally and 529 MiB with \`-race\`; these are test-harness peaks, not product service memory. | Opt-in native Sherpa tests; raw provenance receipt: \`~/.vrooli/plan-artifacts/audio-tools-portable-voice-development/sherpa-cpu-two-stream-concurrency-20261124.json\`. These short runs showed no material first-text regression at the resource's declared four-session limit relative to the prior single-stream 1,217–1,221 ms diagnostic. This does not qualify sustained host pressure, production admission, non-CPU providers, or browser/product latency; first text still exceeds the 1,000 ms p95 target. | 2026-11-24 |
| Isolated D1 CPU and D2 constrained host inference / memory | D1 Whisper Small matched the 1.954 s fixture in 2,826 ms with 1,372,471,296 B peak in an 8-CPU/8-GiB networkless container. D2 Sherpa Zipformer int8 smoke passed in a 2-vCPU/4-GiB networkless container with 445,837,312 B peak; its native smoke measured 127.1 ms inference after 1.224 s load. Single runs | Owner artifact \`audio-tools:host-portability:20261124T063347.686551158Z:amd64\`; model-memory evidence IDs \`912ee50e-de49-4138-b38a-5d9a2a5d21b0\` (Whisper Small) and \`911b5052-9089-470c-a38b-33ea0db45353\` (Sherpa Zipformer) are persisted through the lab evidence API. Both memory checks passed their 2 GiB budgets. The aggregate portability receipt remains \`passed=false\` because D1 Kyutai accelerator inference is unavailable; D3–D6 are unclaimed. This is isolated inference and memory evidence, not browser capture, quality-corpus, or long-session qualification | 2026-11-24 |

These observations establish variability worth investigating, not a baseline or
a passed SLO. Read current persisted experiments through
\`audio-tools.setpoint-read\` with an explicit experiment ID. Replay WER/RTF and
replay finalization timing do not establish click-to-ready, speech-to-visible
partial, or native-device behavior. Missing proto scalar values remain unknown;
do not impute them as measured zero.

## Known Constraints

The \`capture_callback_processing_budget\` soak assertion is a real browser capture callback
processing measurement. The capture provider records a bounded callback count
and the maximum ratio of synchronous callback processing time to the source
audio duration in that callback; the Go soak decoder evaluates that maximum
against a 0.50 budget. A callback that takes more than half of its audio
duration leaves less than a 2x processing margin and fails the assertion.
Missing or invalid callback measurements fail closed. This captures local
resampling, batching, and diagnostic bookkeeping at the callback boundary; it
does not measure browser scheduling delay before callback entry, downstream
socket/server work, or end-to-end latency. The maximum is intentionally a
bounded summary, not a per-frame trace.

- Cold provider/acceleration probes can dominate discovery. Measure independent
  boundaries before changing caching, readiness or selection policy. The live
  resolver now runs status and acceleration checks concurrently; after its
  inventory TTL it refreshes asynchronously and coalesces callers. It may use
  that snapshot up to two TTLs only when each local provider passes a bounded
  500 ms read-only liveness check. A missing or older snapshot still requires a
  fresh control-plane observation and must remain an explicit preparation delay.
- One model can serialize competing requests. Include contention and slow
  consumers; average fast requests do not reveal queueing or lost final tails.
- The replay journal accepts only processed cursors below its next captured
  sequence. A future server cursor is a protocol failure; it must never compact
  unacknowledged PCM. Keep cursor validation at the journal boundary even when
  transports validate the wire shape.
- Cached health and advertised native streaming describe capability, not a
  completed interactive session. Buffered providers need honest final-only UX.
- CPU throttling, virtual-time replay, accelerated soaks and injected browser
  audio each prove narrower claims than native realtime capture.

## Regression Procedure

1. Select the affected route, corpus, engine, build, device and cold/warm cohort
   under [TESTING.md](TESTING.md). Record the target and measurement-method revision.
2. Capture button action, permission completion, capture/session readiness,
   first speech, first displayed partial, final commit and stop completion.
   Compare committed audio position with captured position, not the latest callback time.
3. Retain attempts, failures, timeouts, sample counts and per-cohort distributions.
   Do not pool cold and warm samples or omit failed attempts from the denominator.
4. Prove the relevant negative controls and run focused regressions through the
   [repository test protocol](../../../../docs/TESTING.md). Use paced product-path
   or real-duration validation only for the claims that require those boundaries.
5. Compare like-for-like receipts. Preserve unresolved findings in
   [PROBLEMS.md](PROBLEMS.md); do not copy every experiment into a prose ledger.

The browser-paced harness supports a deterministic client WebSocket close with
code 4001 after the first visible partial (\`--interrupt-after-partial\`). The
probe passes only when reconnect/replay is observed and the turn finishes with
complete audio and sequence accounting. This exercises the product reconnect
path but is not a physical network interruption or device qualification. The
earlier Chrome offline-emulation probe did not interrupt its established
WebSocket and remains a negative control:
[receipt](../../coverage/browser-offline-emulation-no-reconnect-2026-11-24.json).

The first injected run did reconnect and replay, then failed because the client
had not received an exact provider/model identity before the first finalized
segment, so retained-audio fallback correctly refused an unverifiable route.
The [pre-fix receipt](../../coverage/browser-ws-interruption-preidentity-failure-2026-11-24.json)
records that fail-closed route error. Session status now carries the selected
provider/model/tier early enough for exact-route recovery. A one-turn live
rerun passed after a lifecycle-managed API rebuild: the client observed
reconnect/replay, completed with local \`sherpa-stt\`, and reconciled all 79
sent/processed chunks and 62,805 captured/committed samples with no page errors
([receipt](../../coverage/browser-ws-interruption-recovered-2026-11-24.json)).
This single synthetic run exercises recovery but does not establish a
reliability rate or a physical-network/device qualification.

The first server-side replay probe exposed a separate failure: the server
terminalized the original receive error and did not feed duplicate retained
chunks into the new decoder. The handler now preserves a nonterminal ledger on
transport interruption, replays verified duplicate PCM into a resumed
pipeline, and requires explicit \`StreamEnd\` before final completion. A later
three-turn cohort completed its two uninterrupted turns but failed the
interrupted turn with \`stream_internal\` (sequence 78 received, final sample
coverage missing). The server log classified this as a committed-segment
identity conflict: replay changed the VAD segment range/text for an existing
stable segment ID. On resumed streams, the handler now preserves the first
durable segment text/range for that stable ID. A focused regression covers the
revised replay event, while the WebSocket regression covers interrupted
ledger retention and successful final completion.

The lifecycle-rebuilt current source first passed one live Dictation Studio
outage probe: 62,123 of 62,123 captured samples committed, sequence 79
processed, first visible partial 813 ms, stop tail 585 ms, and no page errors
([receipt](../../coverage/browser-ws-replay-conflict-fixed-2026-11-24.json)).
After adding \`--offline-attempt all\`, a five-turn cohort injected the same
1.5-second outage on every turn and passed all five: all sample/sequence/final
accounting reconciled, first-visible p50/p95/max was 782.2/837.3/837.3 ms, and
stop-tail p95 was 576.8 ms. The 3.1-second visible-update-gap p95 includes the
deliberately injected outage and its reconnect; it is not an uninterrupted
speech update-gap measurement. The
[five-turn receipt](../../coverage/browser-ws-recovery-five-turn-2026-11-24.json)
is still diagnostic-only, not a reliability estimate or physical-network/device
qualification. The preceding intermittent cohort remains as
[negative evidence](../../coverage/browser-ws-replay-intermittent-2026-11-24.json).
The separate 30-turn 11-second JFK cohort still exceeds the adopted
first-partial p95 target.

## Cross-References

- [\`../operations/OBSERVABILITY.md\`](../operations/OBSERVABILITY.md) — signals and telemetry
- [\`../operations/DEPLOYMENT.md\`](../operations/DEPLOYMENT.md) — release checklist
- [\`TESTING.md\`](TESTING.md) — coverage and test expectations
- [\`PROBLEMS.md\`](PROBLEMS.md) — unresolved performance debt
`;export{e as default};
