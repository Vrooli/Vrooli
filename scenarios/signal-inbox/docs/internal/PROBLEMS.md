# Problems — Signal Inbox

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

No implementation exists yet, so there are no observed defects. The table below
records **open gaps carried out of the design workshop** — things the design
knowingly does not resolve. They are listed here rather than in
[`DECISIONS.md`](DECISIONS.md) because a decision records what was chosen, while
these record what is still unknown.

| Gap | Why it is open | Blocks | Resolution path |
|---|---|---|---|
| `video-downloader` has no transcript-only request | The capability is assumed by `SIG-P1-005` and does not exist today. The scenario must not work around it by downloading media and transcribing locally — that duplicates a whole problem domain and violates D-018. | `SIG-P1-005` | Confirm the contract with `video-downloader` before planning P1. Until then the requirement stays blocked rather than resolved another way. |
| Platform export formats are unverified | The X archive, Reddit export, and browser bookmark HTML shapes are owned by the platforms and drift without notice. No format here has been validated against a real file. | `SIG-P0-008` | Export one real file per platform and validate the parser against it before treating import as done. Listed as a `manual` validation on the requirement. |
| X bookmark API access is unverified | Scopes, pricing tier, and rate limits for the bookmarks endpoint are believed to be gated but were not confirmed, and platform terms change independently of this repository. | Any future X adapter | Verify against current documentation before any adapter is written. The tier-0 archive path deliberately avoids needing this answer at all. |
| Classification accuracy has no baseline | The classifier's real accuracy is unknown and cannot be estimated before a corpus exists. The predecessor asserted ">85%, >95% with learning" with no way to measure either. | Trusting `SIG-P0-005` proposals | `SIG-P1-008` measures against the override corpus. Until it reports, every proposal is reviewed rather than trusted. |
| The ambient-view budget default is unvalidated | Chosen by reasoning about attention, not measured against real consumption. Too small hides work; too large reintroduces the context tax the budget exists to prevent. | Nothing; it is tunable | Calibrate against real walk-prep consumption once signals exist. |
| Disposition keeps no history | Only the current value is stored, so "when was this marked done, and by whom" is unanswerable. | Triage auditing, if ever needed | Accepted deliberately to keep the model small. Add a transition log only if a real question demands it. |
| Multi-consumer disposition is undefined | Single-primary-category (D-010) avoids the problem rather than solving it. If two consumers ever share a signal, "handled" has no single answer. | `SIG-P2-003` | Define the ownership rule before relaxing the single-category constraint, not after. |
| No hard-delete path exists | Deletion is deliberately absent (D-006), but material captured in genuine error — a credential in a screenshot, say — currently has no removal path. | Nothing today | If added, it must be an explicit, logged, single-signal operation and never reachable from a filter-driven bulk path. |
| Blob-store growth is unbounded | Pasted images accumulate with no size budget and no measurement. | Nothing today | Measure after the first real corpus; add a budget only if it turns out to matter. |

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
