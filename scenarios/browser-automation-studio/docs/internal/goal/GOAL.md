# Goal — Browser Automation Studio rehabilitation

**Destination.** BAS is fast, reliable, polished, maintainable and production
ready, with far less technical debt: each concept has one owner, runtime code
drops from about 281k (measured 2026-09-29, `refactor_inventory.py --no-git`)
toward 205k lines, and the 24 preservation journeys pass on Linux x64.

**Stakes.** BAS failures block Vrooli development, agent browsing, end-to-end
testing, UI quality work and BAS monetization. The operator reports that the
budget and the relationship are at risk; one process-heavy week moved runtime
code −0.4%.

**Constraints.**
- Orchestrated epochs per `large-effort-orchestration`. No Swarm Manager, no
  Plan Manager plan.
- Worker Luna-medium, orchestrator Luna-high, Sol only for supervision.
- Preserve behavior and saved user data. Never weaken a `[REQ]` test.
- Do not restart the live BAS mid-run without the fallback check (TARGETS.md).
- No git; the operator commits. Commercial launch, billing and publication are
  out of scope.

**Gates.** Acceptance follows `large-effort-orchestration` §4.4 and the worker
card's growth budget. Daily qualification and quality bands are in TARGETS.md.

**Links.** [QUEUE.md](QUEUE.md) · [TARGETS.md](TARGETS.md) ·
[WORKAROUNDS.md](WORKAROUNDS.md) · [INSTALLS.md](INSTALLS.md) ·
[FEEDBACK.md](FEEDBACK.md) · [epochs/](epochs/) · Verification:
[TESTING.md](../TESTING.md) · Defects: [PROBLEMS.md](../../PROBLEMS.md) · History:
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/bas-rehabilitation/` · Design: plan artifact
`epoch-based-delivery-orchestration-and-bas-rehabilitation/source/TARGET_MODEL.md`.
