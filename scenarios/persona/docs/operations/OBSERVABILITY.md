# Observability — Persona

What this scenario emits, what should be watched, and what it
deliberately does not record.

## Purpose Of This Document

Use this document to answer:

- What signals exist?
- What is logged, and what must never be?
- Which metrics matter?
- What alerts are worth having?
- Where are the telemetry gaps?

## The Constraint That Shapes Everything Here

This scenario handles personal data, so **observability is subject to
the same minimisation rule as storage**. A log line or metric label that
leaks a legal name, an address, a code value, or document content is a
privacy incident regardless of how useful it would have been. Several
otherwise-obvious signals are therefore deliberately absent.

## Signals

| Signal | Source | Purpose |
|---|---|---|
| Health endpoint | `health` domain | API and SQLite reachability |
| Dependency reachability | `access`, `channels`, `documents` | Whether required scenarios are reachable; surfaced in Settings |
| Act-as outcomes | `access` | Granted versus refused, with reason class |
| Handoff queue depth and age | `handoffs` | The single most operationally meaningful number in the scenario |
| Code-retrieval outcomes | `channels` | Success, timeout, adapter-unavailable |
| Release outcomes | `documents` | Released versus refused |
| Staleness findings | `accounts` (P1) | Expiring documents, failing mailboxes, dead routes |

## Logs

**Logged**: verb, outcome, persona id, run id, authorising subject id,
reason class, duration, adapter id, dependency name.

**Never logged** — this list is binding:

- One-time code values, in any form, including truncated or hashed.
- Document content or filenames that reveal content.
- Legal names, addresses, or any persona attribute value. **Identifiers
  only.**
- Credentials or credential references that would be usable.
- Handoff checkpoint payloads, which may embed mid-enrolment form data.

Reason classes are enumerated rather than free-text so a refusal can be
counted and alerted on without embedding detail. The detail lives in the
journal, which is access-controlled; logs are not.

## Metrics

| Metric | Type | Why it matters |
|---|---|---|
| `handoffs_open` by age bucket | gauge | The operator-facing number. Growth means work is blocked on a human. |
| `handoffs_expired_total` | counter | **Every expiry is a failure of the delivery story**, not of the model. Should be near zero. |
| `act_as_refused_total` by reason class | counter | A spike on `authority_unreachable` is a dependency incident; a spike on `acl_denied` is a configuration or intrusion signal. |
| `act_as_latency` | histogram | Dominated by the live verification call; the scenario's main latency contributor. |
| `code_retrieval_outcome_total` by adapter and outcome | counter | Timeout rate per adapter is the health of the OTP story. |
| `release_refused_total` by reason | counter | Should be rare; a rise suggests a flow requesting releases it does not need. |
| `journal_entries_total` | counter | Monotonic by construction. **A decrease is an integrity alarm.** |
| `dependency_reachable` by name | gauge | Feeds the Settings surface and the degraded-state UX. |

## Alerts / Health

| Alert | Condition | Severity | Response |
|---|---|---|---|
| Journal counter decreased | `journal_entries_total` non-monotonic | **Critical** | Treat as a security event; see the runbook escalation table. |
| Handoff expiring soon | Open handoff within 20% of its deadline | Warning | Deliver again; work the queue. |
| Handoff expired | Any expiry | Warning | Investigate the delivery path, not the handoff. |
| Verification authority unreachable | `dependency_reachable{agent-manager}=0` for 5 min | Warning | Expected during `agent-manager` restarts. **Not critical here** — refusing is correct behaviour. |
| ACL denial spike | `act_as_refused_total{reason=acl_denied}` above baseline | Warning | Configuration drift or an agent attempting access it does not have. |
| Code retrieval timeout rate high | Per-adapter timeout rate over threshold | Warning | Check the adapter and the mailbox credential. |
| SQLite unreachable | Health failing | **Critical** | Scenario is down. |

## Telemetry Gaps

| Gap | Impact | Plan |
|---|---|---|
| No time-to-completion on handoffs | The most important product metric — how long a human takes — is unmeasured | Instrument at P0 implementation; it is the signal the monetization validation plan depends on |
| No per-persona usage view | Cannot tell which personas are load-bearing versus abandoned | Derive from the journal at P1 rather than adding new emission |
| Journal growth is unbounded and unmonitored | Slow-burn disk issue | Add a size gauge alongside export-and-archive |
| No metric distinguishes "no code sent" from "adapter failed" | The two have entirely different responses and currently look alike | Requires the counterparty-side signal; may be genuinely unknowable |
| Dependency reachability is polled, not evented | Up to one interval of staleness in the Settings view | Acceptable; the UI labels the age |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — what to do when an alert fires
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — release verification
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — why the never-log list is binding
- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy rules
