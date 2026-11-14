# Observability — Device Studio

## Purpose Of This Document

Specify truthful operational visibility. The scaffold health route is real infrastructure; product metrics below are planned.

## Signals

Track submitted/admitted/refused work, input/compiled revisions, queue/running/checkpoint/cancel state, worker capability and capacity, output integrity, review scope, external-media consent and funding/reconciliation. Separate technical render success from visual approval and shared publication. Use per-attempt/parent identifiers to join child work without leaking secrets.

## Logs

Structured logs include safe project/job/attempt/correlation IDs, stage, duration, error category and artifact references. Redact tokens, private URLs, raw media/prompt content and account-sensitive details. Log provider returned identity/cost only when reported, with known/estimated/pending classification. Child process logs have resource/size limits and retention policy.

## Metrics

Measure preview seek/interaction latency, queue wait, final frame/encode time, peak CPU/RAM/VRAM/disk, cache hits, import rejection reasons, cancelled-child cleanup, retry/reconciliation age and review rejection categories. Meter authoritative service usage separately from local resource utilization. Missing telemetry is unknown; no fabricated zero counters or green capacity.

## Alerts / Health

Core health reflects ability to serve local operations. Optional provider/account/publication degradation does not falsely mark all local work unavailable. Alert on stuck owned work, cleanup failure, unresolved funded dispatch, exhausted storage and asset corruption. Thresholds are configured from measured profiles; no universal rendering SLA is claimed.

## Telemetry Gaps

No product job/worker/cost/privacy telemetry exists yet. Owner receipt semantics and target platform measurements are implementation gates. Public analytics or telemetry export is not automatically enabled; project privacy and installation policy control outgoing data.

## Cross-References

[RUNBOOK.md](RUNBOOK.md), [performance](../internal/PERFORMANCE.md), [security](../internal/SECURITY.md), [flows](../concepts/FLOWS.md).
