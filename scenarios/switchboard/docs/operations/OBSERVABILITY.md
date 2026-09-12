# Observability — Switchboard

## Purpose Of This Document

What this scenario emits, what should be watched, and — importantly — what it
must never emit. Use it to answer: how would I know this is broken, and how
would I know it is broken in the specific way that costs money or leaks
something.

## Signals

The three questions worth instrumenting, in priority order:

1. **Is anyone being ignored?** Inbound messages accepted versus turns completed
   or explicitly refused. A growing gap is the scenario's defining failure and it
   is invisible in ordinary error rates, because nothing errors — the agent just
   goes quiet.
2. **Is money leaking?** Metered turns per thread against caps, and any
   agent-authored message that triggered a turn (which should be exactly zero).
3. **Is something leaking?** Refusals that did not fire, and any scope granted
   above a sender's tier (also exactly zero).

Everything else is ordinary service health.

## Logs

| Event | Level | Fields |
|---|---|---|
| Ingress accepted | info | channel, thread, remote message id, author kind, byte size |
| Ingress de-duplicated | debug | channel, remote message id |
| Binding resolved / ambiguous | info / warn | channel, address, agent id |
| Scope resolved | info | turn, sender tier, room ceiling, granted scopes, withheld scopes |
| Turn refused | **info, never error** | turn, reason. A refusal is correct behavior; logging it as an error trains operators to ignore errors |
| Turn dispatched / completed | info | turn, run id, duration |
| Silence by arbitration | debug | thread, rule that applied — mention, budget, or authorship |
| Budget or cap exhausted | warn | thread, window, limit |
| Adapter connect / disconnect / degraded | info / warn | channel, reason |
| Descriptor validation failure | **fatal** | file, field. Boot stops |
| Cross-node dispatch | info | machine, verb, outcome |

**What must never be logged.** Message bodies. Media content or filenames that
reveal content. Credential values. Contact addresses at debug level in bulk.

This is stricter than the usual "do not log secrets" rule and is easy to violate
accidentally in a debug statement on the ingress path — which is precisely where
someone will want one while building an adapter. Log the byte size and the
identifiers; never the text.

## Metrics

| Metric | Type | Why |
|---|---|---|
| `inbound_accepted_total{channel}` | counter | Denominator for the ignored-person gap |
| `turns_completed_total{channel}` / `turns_refused_total{reason}` | counter | The numerator, split by cause |
| `turns_silent_total{rule}` | counter | Deliberate silence, separated from failure. Conflating the two hides the real gap |
| `ingress_dedupe_hits_total{channel}` | counter | A sudden rise means a provider changed redelivery behavior |
| `ingress_ack_duration_seconds{channel}` | histogram | Against the 150 ms budget; overruns cause provider redelivery |
| `scope_resolution_duration_seconds` | histogram | On every turn; against the 20 ms budget |
| `scope_granted_above_tier_total` | counter | **Must be zero.** Any non-zero value is a security incident |
| `agent_authored_turns_total` | counter | **Must be zero.** Non-zero means the loop breaker failed and money is being spent |
| `metered_units_total{thread}` | counter | Spend visibility per thread |
| `budget_exhausted_total{thread}` | counter | Distinguishes an abusive thread from a mis-set cap |
| `channel_state{channel,state}` | gauge | live, unavailable, unimplemented, degraded |
| `adapter_send_failures_total{channel,terminal}` | counter | Transient versus terminal, split |

## Alerts / Health

`/health` reports SQLite reachability and per-adapter connection state.
`switchboard channels status --json` reports per-channel disposition on the
machine it runs on — locally a direct call, remotely the identical verb through
durable dispatch.

| Condition | Severity | Meaning |
|---|---|---|
| `scope_granted_above_tier_total > 0` | **critical** | A permission failed open. Escalate; do not restart before capturing evidence |
| `agent_authored_turns_total > 0` | **critical** | Loop breaker failed. Live metered spend |
| Accepted-minus-completed gap growing | high | People are being ignored |
| `ingress_ack_duration_seconds` p95 over budget | high | Providers will begin redelivering |
| Adapter degraded over 5 minutes | medium | A channel is effectively down; senders should already be seeing stated failures |
| Descriptor validation failure | high | Boot stopped; the scenario is down, loudly and by design |
| Media growth above the storage plan | low | The only unbounded line item |

## Telemetry Gaps

Honest list.

- **Nothing is emitted yet.** No domain code exists, so every metric above is a
  specification.
- **"Should have replied but did not" is not directly observable.** The
  accepted-minus-completed gap is a proxy: it cannot distinguish a correct
  silence from a dropped turn without the `turns_silent_total{rule}` split, which
  is why that metric exists rather than being folded into refusals.
- **No per-sender view.** A single abusive contact across several threads is
  invisible until the thread caps trip individually. This is the observability
  half of `SWBD-PROB-006`.
- **Local spend is a mirror.** `metered_units_total` can drift from the LPBS
  wallet between reconciliations; alerting on the mirror alone will occasionally
  be wrong in the cheap direction.
- **Cross-node visibility is a round trip.** A Mac node's channel state is cached
  with a stated lifetime, so a Mac that just went offline reads healthy until the
  cache expires or a delivery fails.
- **No end-to-end latency signal**, deliberately. Model inference dominates it
  and belongs to `agent-manager`.

## Cross-References

- `docs/operations/RUNBOOK.md` — the procedures these signals drive
- `docs/internal/PERFORMANCE.md` — the budgets the histograms check
- `docs/internal/SECURITY.md` — why two metrics must be exactly zero
- `docs/internal/PROBLEMS.md` — `SWBD-PROB-006`, the per-sender gap
- `docs/concepts/FLOWS.md` — the flow steps each event corresponds to
