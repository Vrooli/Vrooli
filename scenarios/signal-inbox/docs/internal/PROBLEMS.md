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

## Work ladder

- Rung: W1
- Evidence: W0 passed on 2026-07-27 after the retained active goal `bookmark-intelligence-hub-rework-and-ideation` was retitled and its description explicitly named `signal-inbox`. Its mandatory claims — manual/tier-0 capture, immutable storage, operator-defined categories and confirmation, corpus-wide retrieval independent of category or disposition, and auto-disable/no-retry adapter safety — agree with `OT-P0-001` through `OT-P0-014`; `D-006`, `D-013`, `D-015`, and `D-016` do not supersede any P0 target.
- Blocker: W1 has not yet been measured.
- Measured: 2026-07-27

No implementation exists yet, so there are no observed defects. The table below
records **open gaps carried out of the design workshop** — things the design
knowingly does not resolve. They are listed here rather than in
[`DECISIONS.md`](DECISIONS.md) because a decision records what was chosen, while
these record what is still unknown.

| Gap | Why it is open | Blocks | Resolution path |
|---|---|---|---|
| `video-downloader` has no transcript-only request | The capability is assumed by `SIG-P1-005` and does not exist today. The scenario must not work around it by downloading media and transcribing locally — that duplicates a whole problem domain and violates D-018. | `SIG-P1-005` | Confirm the contract with `video-downloader` before planning P1. Until then the requirement stays blocked rather than resolved another way. |
| X and Reddit export formats are unverified | Platform formats are owned by the platforms and drift without notice. Chrome bookmark HTML was measured on 2026-07-27; the X archive and Reddit export were not available, so their fields were not inferred. | `SIG-P0-008` | Provide one real X archive and Reddit export. Record only each file's layout and URL/timestamp/identity field names, then validate the parser before treating import as done. Listed as a `manual` validation on the requirement. |
| X bookmark API access is unverified | Scopes, pricing tier, and rate limits for the bookmarks endpoint are believed to be gated but were not confirmed, and platform terms change independently of this repository. | Any future X adapter | Verify against current documentation before any adapter is written. The tier-0 archive path deliberately avoids needing this answer at all. |
| Classification accuracy has no baseline | The classifier's real accuracy is unknown and cannot be estimated before a corpus exists. The predecessor asserted ">85%, >95% with learning" with no way to measure either. | Trusting `SIG-P0-005` proposals | `SIG-P1-008` measures against the override corpus. Until it reports, every proposal is reviewed rather than trusted. |
| The ambient-view budget default is unvalidated | Chosen by reasoning about attention, not measured against real consumption. Too small hides work; too large reintroduces the context tax the budget exists to prevent. | Nothing; it is tunable | Calibrate against real walk-prep consumption once signals exist. |
| Disposition keeps no history | Only the current value is stored, so "when was this marked done, and by whom" is unanswerable. | Triage auditing, if ever needed | Accepted deliberately to keep the model small. Add a transition log only if a real question demands it. |
| Multi-consumer disposition is undefined | Single-primary-category (D-010) avoids the problem rather than solving it. If two consumers ever share a signal, "handled" has no single answer. | `SIG-P2-003` | Define the ownership rule before relaxing the single-category constraint, not after. |
| No hard-delete path exists | Deletion is deliberately absent (D-006), but material captured in genuine error — a credential in a screenshot, say — currently has no removal path. | Nothing today | If added, it must be an explicit, logged, single-signal operation and never reachable from a filter-driven bulk path. |
| Blob-store growth is unbounded | Pasted images accumulate with no size budget and no measurement. | Nothing today | Measure after the first real corpus; add a budget only if it turns out to matter. |
| Phase 1 baseline comparison was not comparable | Git Control Tower validation operation `4c7ea8aa-41e2-435c-9c30-336b913a1da9` returned `signal-inbox:ready:regression` with the only detail `phase did not apply to one or both captured targets`; its two proto-path checks have no typed producer synchronization. The native inference tests, `ai-go/search` suite, and linter passed. | Automated phase-close evidence only; no requirement status was advanced. | Re-run the phase baseline comparison after the first signal-inbox proto/API slice produces a typed target; do not treat the current baseline verdict as behavioral evidence for a requirement. |

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
