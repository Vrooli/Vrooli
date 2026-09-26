# Runbook — Switchboard

## Purpose Of This Document

The operational procedures for running this scenario: starting and stopping it,
diagnosing the incidents it is actually likely to have, backing it up, and
knowing when to stop and escalate.

This scenario has an unusual operational property worth stating at the top:
**when it is broken, a real person is being ignored.** A silent agent is
indistinguishable from a rude one, and unlike a failed batch job the damage is
social rather than technical. Prefer a stated failure over silence in every
procedure below.

## Start / Stop / Status

Always through the lifecycle. Never run a binary directly — that bypasses
process naming, port allocation, and health checks.

```bash
make start      # or: vrooli scenario start switchboard
make status
make logs
make stop
make test
make orient     # initialization gate progress while this scenario is still in setup
```

Health checks to trust, in order:

1. `make status` — lifecycle metadata resolves for API and UI.
2. `switchboard channels status --json` — per-channel disposition on this
   machine: live, unavailable with a reason, unimplemented, degraded.
3. `switchboard agents list --json` — bindings resolve, which proves
   `prompt-manager` reachability.

**`scenario status` reads stored snapshots and does not re-probe.** A green
status on a process running stale code is a known class of confusion; after a
rebuild, restart rather than trusting the snapshot.

## Common Incidents

### "The agent stopped replying"

The most common report, and it has six plausible causes. Work down in order —
this ordering is cheapest-first, not most-likely-first.

1. **Was it addressed?** In a group, the default is `speak_when: mentioned`.
   Silence without a mention is correct behavior, not an incident.
2. **Turn budget exhausted?** Check the thread's `budget_window`. The owner is
   notified once, not per suppressed turn, so the absence of repeated warnings
   proves nothing.
3. **Spend cap reached?** This refuses out loud, so there should be a message on
   the thread. If there is not, that is a real defect.
4. **Scope empty?** Check `scope_resolution_log` for the turn. A refusal states
   what was withheld; a missing refusal with an empty scope is a defect.
5. **Adapter degraded?** `switchboard channels status`. A disconnected adapter
   should surface on the thread, not just in logs.
6. **`agent-manager` unreachable?** Turns cannot execute; the sender should have
   received a stated unavailable reason.

If none of these explains it, the message may never have arrived — check the
ingress de-duplication table for the remote message id before assuming a
processing fault.

### "Messages arrived twice"

De-duplication is keyed on `(channel_id, remote_message_id)`. Two causes: the
adapter is not populating `remote_message_id`, or the provider is issuing a new
id per delivery attempt. The first is a defect in that adapter and is the far
more likely of the two. Confirm against `ingress_dedupe` before changing
anything, and note that a duplicate that reached dispatch also duplicated a
metered charge.

### "iMessage stopped working"

Ordered by likelihood, and this one is genuinely fragile by construction:

1. The Mac node is offline or on a different project revision — check
   `vrooli-bridge` node status first, because a version-skewed node fails
   dispatch in a way that reads exactly like a channel outage.
2. Full Disk Access was revoked, commonly by a macOS update.
3. A macOS update changed the local message store shape. Apple provides no
   supported inbound interface, so this will recur. It must present as
   unavailable with a stated reason, never as silence.
4. Messages is signed out of the Apple ID.

### "A channel shows unavailable"

That is the design working. The `reason` field names the single unmet
requirement — no Mac node, no public origin, no credential. Satisfy the
requirement rather than editing the descriptor: a descriptor edited to drop a
`requires` entry produces a channel that fails later and less clearly.

### "Boot fails naming a descriptor file"

Intended. An invalid descriptor fails boot loudly rather than being skipped,
because a silently skipped channel is one the operator believes is running. Fix
the named field, or move the file out of `data/channels/` to disable it.

### "Two agents are talking to each other"

Should be impossible: no turn starts in response to an `author_kind: agent`
message. If it is happening, either an adapter is mislabelling authorship or an
agent-authored message is re-entering through a different channel. **Stop the
affected thread's budget first** — this is a live metered spend — then diagnose.

### "The console shows something odd in a message"

Message bodies are untrusted input from unauthenticated senders. If content is
rendering as markup rather than as text, that is a security defect, not a display
bug. File it as such.

## Backup / Restore

| What | Backup | Restore |
|---|---|---|
| Scenario database | Standard scenario storage backup | Restores threads, contacts, tiers, bindings, and the resolution log |
| Media blobs | Same storage root, referenced by content hash from rows | Must be restored **with** the database — a row without its blob is a broken thread |
| Descriptors | Plain files in `data/channels/` | Copy them in; they are portable between installs by design |
| Credentials | **Not in scope.** Only references are stored | Restore through the credential authority |

Restoring a database without its media, or vice versa, produces dangling
references. Treat them as one unit.

## Maintenance Tasks

| Task | Cadence | Notes |
|---|---|---|
| Prune `ingress_dedupe` beyond 30 days | Automatic | Pruning too early risks re-answering a stale redelivery |
| Prune `budget_window` beyond 7 days | Automatic | — |
| Reconcile `spend_ledger` against LPBS | Per the metering interval | The local ledger is a mirror; LPBS is the authority |
| Re-probe channel availability | Scheduled, and after any failed delivery | Cached per machine with a stated lifetime; freshness is this scenario's responsibility, not the bridge's |
| Review contacts at `trusted` or above | Occasional, operator judgement | Tiers grant real capability and nothing expires them automatically |
| Watch media growth | Monthly | The only line item here that grows without bound |
| **Never** prune `scope_resolution_log` before a year | — | It is the audit record of what was permitted and must outlive the conversations it describes |

## Escalation

Stop and escalate to the operator rather than continuing, when:

- A refusal was expected and did not happen. A permission that failed open is a
  security incident, not a bug — capture `scope_resolution_log` before touching
  anything.
- Message content appears anywhere it should not: a log line, an error payload,
  a different thread, a different channel.
- An agent replied to a person whose tier should not have reached that
  capability. There is no un-send; preserve evidence first.
- Metered spend is climbing with no corresponding human activity.
- A non-owner tier is exposed in production while `SWBD-PROB-001` is open —
  that is an accepted-risk decision only the operator may make.

File defects outside this scenario's scope through the `report-bug` skill to
`scenario-qa`. Record completed non-trivial work with
`vrooli-memory journal note --kind work-record`.

## Cross-References

- `docs/operations/OBSERVABILITY.md` — the signals these procedures read
- `docs/operations/DEPLOYMENT.md` — release and rollback
- `docs/concepts/FLOWS.md` — the flows these incidents interrupt
- `docs/internal/SECURITY.md` — why the escalation triggers are what they are
- `docs/guides/troubleshooting.md` — template-level lifecycle and build issues
