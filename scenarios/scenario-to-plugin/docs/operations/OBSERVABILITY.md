# Observability — Scenario to Plugin

What this scenario emits, what it is safe to emit, and what is not yet
measured.

## Purpose Of This Document

Use this document to answer:

- What signals exist, and which one do I look at first?
- What is deliberately not logged, and why?
- Which alerts matter, and which would be noise?
- What is currently unmeasured?

## Signals

The primary operational question is always the same: **which gate closed,
and on what evidence?** Every signal below serves that question.

| Signal | Kind | Answers |
|---|---|---|
| Gate ladder position per package | State | Where a package stopped, and why. The first thing to check for any "why can't I publish" question. |
| Gate disposition | State | `passed`, `failed`, `degraded`, `unavailable`, `unverified`. A missing check is `unverified` and is never reported as a pass. |
| Conformance findings | Event | Which rule failed, in which file, at which offset. |
| Drift outcome + pinned manifest revision | Event | Whether a skill regressed or the CLI moved. Useless without the revision. |
| Scanner verdict | Event | Scanner name, version, and severity summary. |
| Rehearsal journey | State | Per-step gate results and per-command exit status. |
| Sandbox lifecycle | Event | Created, torn down. Non-matching counts mean a leak. |
| Publication outcome per channel | Event | Pushed, confirmed, failed — one record per channel per attempt. |
| Revocation outcome per channel | Event | Withdrawn, or still carrying the artifact. |
| Install attribution | Metric | Installs and referrer origins per plugin per channel. |

## Logs

Structured, one event per gate transition and per external call.

Every log line for a gate transition carries: package id, scenario slug,
source commit, stage, disposition, and — when the disposition is not
`passed` — the rule or gate that decided it.

**Deliberately never logged:**

- Credential values, tokens, or any resolved secret. Only reference names.
- Raw rehearsal command output. Output is redacted **at capture time**,
  not at read time; redacting on read would mean the unredacted bytes
  existed on disk.
- Full skill bodies. A finding carries file and offset, not content —
  logging the body would relocate a prompt-injection payload into the log
  pipeline and any downstream reader of it.
- Artifact bytes. Digests only.

The redaction posture here is stricter than most scenarios because this
scenario handles untrusted content by definition: a skill body under
review may be adversarial, and a log aggregator read by an agent is a
downstream injection surface.

## Metrics

| Metric | Type | Purpose |
|---|---|---|
| `packages_composed_total{scenario,result}` | counter | Build volume and failure rate. |
| `gate_transitions_total{stage,disposition}` | counter | Where the pipeline stops. The single most useful dashboard. |
| `conformance_findings_total{rule}` | counter | Which rule fires most; informs where authoring guidance is weak. |
| `drift_failures_total{scenario}` | counter | Skill ↔ CLI drift rate over time. A rising trend is a process signal, not a bug signal. |
| `rehearsal_duration_seconds{stage}` | histogram | Rehearsal is the slow stage; this is the capacity signal. |
| `sandboxes_leaked` | gauge | Created minus torn down. Should be zero. |
| `publications_total{channel,result}` | counter | Channel reliability. |
| `publication_confirm_latency_seconds{channel}` | histogram | Gap between push and confirmed retrieval. |
| `revocations_partial_total{channel}` | counter | Which registries cannot complete a withdrawal. |
| `plugin_installs_total{plugin,channel}` | counter | The channel-activation metric (`OT-P1-003`). |

## Alerts / Health

| Alert | Condition | Severity | Rationale |
|---|---|---|---|
| Sandbox leak | `sandboxes_leaked > 0` for 15 minutes | High | Leaked sandboxes consume real host resources and indicate teardown is not running on every exit path. |
| Signing authority unavailable | Attestation signing failures over 5 minutes | High | Blocks all publication. |
| Publication unconfirmed | A publication stays unconfirmed for 15 minutes | High | The registry accepted and may have dropped the artifact. |
| Partial revocation open | Any `revoked_partial` older than 1 hour | **Critical** | An artifact that should be withdrawn is still reachable by users. This is the highest-severity state this scenario can be in. |
| Drift failure spike | `drift_failures_total` rises across 3+ scenarios in a day | Medium | Suggests a `cli-manifest` contract change rather than many independent skill regressions. |
| Scanner unreachable | Scan invocation failures over 10 minutes | Medium | Attestation fails closed, so this blocks releases but does not endanger anything. |

Deliberately **not** alerted: individual conformance failures and
individual drift failures. Those are the system working. Alerting on them
would train operators to treat a closed gate as an incident, which is the
precise habit that leads to bypasses.

Health endpoint reports API readiness, database reachability, and the
reachability of each required scenario dependency.

## Telemetry Gaps

Honest list of what is not measured today. None of these should be
described as working.

| Gap | Impact | Resolution |
|---|---|---|
| Per-plugin install attribution | The `skill-registries` channel cannot be activated from evidence. This is the channel's own stated prerequisite. | `OT-P1-003` / `PLG-DIST-ATTRIBUTION`. |
| Install → subscription correlation | The commercial hypothesis is unfalsifiable without it. | Requires attribution plus an Offer Desk join; owned jointly. |
| Re-verification of published versions | Drift in an already-published package is discovered by users, not by us. | Not yet scheduled. Recorded in `PROBLEMS.md`. |
| Retention enforcement | Retention rules are documented but no prune job runs. | Recorded in `PROBLEMS.md`. |
| Time-to-deprecation per published version | The channel doctrine names this as a process-health signal; nothing emits it. | Depends on publication history plus re-verification. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — what to do when an alert fires
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — release checklist
- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy posture
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — why redaction is stricter here
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — the gaps above, tracked
