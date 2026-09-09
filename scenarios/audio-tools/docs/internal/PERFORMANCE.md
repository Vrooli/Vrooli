# Performance — Audio Tools

This document owns performance interpretation and regression procedure. The
[pilot decision sheet](TESTING.md#pilot-decision-sheet-and-safe-first-slice) owns
candidate product SLOs; experiment and test owners retain measured receipts.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Interactive voice | Pending adoption of the TESTING.md candidate cohort SLOs | Paced browser traces at the shared consumer boundary | proposed, not qualified |

PRD OT-P0-005 makes responsive streaming a required product outcome; pending
numbers do not make it optional. Adopt the candidate bands and define cold
readiness, admission timeout, supported concurrency and session limits in the
single TESTING.md decision sheet before a numeric completion claim. Do not
duplicate or silently relax those bands here.

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

Report p50/p95, attempts, successes, failures, timeouts, maximum and sample count
for each route/profile/cold-warm cohort. Latency percentiles over successful
attempts must be labeled as such and paired with the full failure denominator;
timeouts remain violations or incomplete attempts, never omitted successes.
Separate permission prompts, model provisioning, queueing and steady-state
processing. No fleet availability percentage is established by a small soak.

Lifecycle responsiveness is not voice readiness. Build duration is a tooling
measurement, not permission to spend minutes starting a dictation session.

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Cold / subsequent engine discovery | 10,421 ms / 3 ms in the inspected pair | Dated investigation in PROBLEMS.md; two samples, not a percentile | 2026-09-08 |
| Streaming file first partial | 936 ms, then 13,929 ms on a repeat | Dated investigation in PROBLEMS.md; unpaced file input, not browser microphone latency | 2026-09-08 |

These observations establish variability worth investigating, not a baseline or
a passed SLO. Read current persisted experiments through
`audio-tools.setpoint-read` with an explicit experiment ID. Replay WER/RTF and
replay finalization timing do not establish click-to-ready, speech-to-visible
partial, or native-device behavior. Missing proto scalar values remain unknown;
do not impute them as measured zero.

## Known Constraints

- Cold provider/acceleration probes can dominate discovery. Measure independent
  boundaries before changing caching, readiness or selection policy.
- One model can serialize competing requests. Include contention and slow
  consumers; average fast requests do not reveal queueing or lost final tails.
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

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
