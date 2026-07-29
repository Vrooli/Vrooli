# Problems — Device Sync Hub

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

_None yet._

## Work ladder

### 2026-07-28 — large remote upload investigation

**W0 (contract):** blocked/unverifiable. `swarm-manager goals list --json` found
no goal that names `device-sync-hub`; the PRD independently names server-relayed
file transfer as P0 and chunked/resumable large-file upload with per-file
progress as P1 (`OT-P1-001`). The operator's 200+ MB X archive failure directly
exercises that P1 capability, so this repair is in scope without changing the
contract.

**Investigation:** local API accepts a 2 GiB multipart maximum and rejects an
unauthenticated request before reading its body; the external Cloudflare route
redirects unauthenticated requests at about 1.7 MB, so the authenticated
200 MB edge failure cannot be reproduced from this host without an operator
session. The hub emitted no matching API log. The current UI sends one whole
file request and discards its existing XHR byte-progress callback. The most
likely root cause is an edge request-size limit before the request reaches the
origin, with a generic non-JSON edge response collapsed into the UI's generic
error. A server-side maximum is ruled out by
`api/handlers/transfer/upload_handler.go` (`maxUploadBytes = 2 GiB`).

**Next rung:** W3 implementation: replace one-shot large uploads with bounded
chunked/resumable sessions and surface per-file upload state, progress, and
actionable errors. Re-run W0 after the repair; W1/W2/W3 gates are intentionally
not claimed while W0 is unverifiable.

## UX Issues

- Resolved 2026-07-28: received-item cards showed filename, retention, expiry,
  and actions without explaining that each item is a server-relayed copy or
  which trusted devices may retrieve it. `ReceivePanel` now names Hub storage,
  broadcast versus directed availability, and expiry-driven retention on every
  card.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
