# Runbook — Persona

Operational procedures: starting and stopping, responding to incidents,
backup and restore, and routine maintenance.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and check this scenario?
- What do I do when a specific thing breaks?
- How do I back up and restore?
- What routine maintenance exists?

## Start / Stop / Status

Always through the Vrooli lifecycle — never by invoking binaries
directly.

```bash
make start                          # or: vrooli scenario start persona
make status                         # lifecycle + health
make logs                           # or: vrooli scenario logs persona
make stop
vrooli scenario test persona        # server-owned run; survives your cancel
```

Health is degraded rather than failed when an **optional** dependency is
unavailable. Health is failed when SQLite is unreachable. Note that
`agent-manager` being unreachable does **not** fail health — the
scenario is working correctly by refusing act-as, and reporting failure
would send an operator hunting in the wrong scenario.

## Common Incidents

### "Everything is refusing act-as"

**Almost always correct behaviour, not a bug.** Check `agent-manager`
first:

```bash
vrooli scenario status agent-manager
persona journal list --verb act-as --outcome refused
```

The refusal reason is journaled. If `agent-manager` is down, start it;
act-as resumes automatically with no state repair. **Do not look for a
flag to bypass verification — there isn't one, by design.**

### "A handoff is stuck"

Distinguish three cases before acting:

- **Waiting** — nobody has acted yet. Not stuck. Check whether delivery
  reached the operator: `persona handoffs show <id>` includes delivery
  attempts.
- **Delivery failing** — the relay is down or misconfigured. The queue
  still works; work it directly at `/handoffs`.
- **Expired** — a terminal state. It cannot be completed; the
  originating flow must open a new handoff. Do not attempt to revive it.

### "A code never arrives"

```bash
persona channels test <persona-id>
```

Distinguish adapter failure (credential rejected, provider unreachable)
from no-code-sent (the counterparty never dispatched one). Only the
first is this scenario's problem. **Never work around it by using a
different persona's channel** — a code fetched as the wrong identity is
worse than no code.

### "A document release failed"

Check that the target handoff is open and that the binding belongs to
the persona in question. Releases are idempotent per (binding, handoff),
so retrying a partially completed release is safe.

### "Retirement is blocked"

Working as designed. Linked accounts lack a recorded recovery path.
`persona accounts list --persona <id>` names the blocking links; record
recovery paths, then retire.

### "The journal looks wrong"

The journal is append-only and has no repair procedure by design. If a
row is believed incorrect, write a compensating entry — **never edit**.
If rows appear to be missing, that is a serious integrity finding:
export what exists, record it in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md), and treat it as a
security event.

## Backup / Restore

| Item | Method | Frequency | Notes |
|---|---|---|---|
| SQLite database | File copy while stopped, or SQLite online backup | Daily | Contains personas, bindings, ACL, handoffs, journal. |
| Journal export | `persona journal export` | Weekly, off-host | The one artifact worth keeping independently of the database. |
| Credentials | **Not here** | — | `secrets-manager` owns its own backup story. |
| Documents | **Not here** | — | `document-manager` owns its own backup story. |

**Restore ordering matters.** Restore `secrets-manager` and
`document-manager` before this scenario, or bindings will resolve to
material that does not exist yet and staleness checks will raise
spurious findings.

## Maintenance Tasks

| Task | Cadence | Why |
|---|---|---|
| Review open handoffs | Weekly | Handoffs expire; an unnoticed expiry silently blocks a flow. |
| Review staleness findings | Monthly | Expiring documents and failing mailboxes are cheap to fix early and expensive mid-enrolment. |
| Export the journal off-host | Weekly | The journal has no external anchoring; export is the mitigation. |
| Re-verify the fail-closed path | Each release | The security property most likely to regress silently. |
| Audit ACL entries | Quarterly | Access granted for one task tends to outlive it. |
| Review account links before retiring anyone | On demand | The orphaning failure is silent and permanent. |

## Escalation

| Situation | Action |
|---|---|
| Suspected unauthorised act-as | Revoke the ACL entry, export the journal for the window, treat as a security event. Do not delete anything. |
| Journal integrity doubt | Stop the scenario, preserve the database file, escalate. Do not restart into a suspect database. |
| Document released to an unexpected handoff | Read the release record for authority and target, revoke the binding, escalate to whoever owns the document. |
| Dependency owned by another scenario | Escalate there; do not work around it here. A workaround in this scenario is a security regression. |

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — what to watch
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — state machines behind these incidents
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — threat model
