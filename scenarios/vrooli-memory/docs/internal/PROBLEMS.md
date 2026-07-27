# Problems — Vrooli Memory

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

### Open gaps carried from design (2026-07-27)

| Gap | Impact | Tracked As |
|---|---|---|
| Compaction scoring shape `cohesion × slots freed` is plausible but not derived. | Bad summaries if the shape is wrong; first thing to re-examine on contact with real clustering output. | `VMEM-P0-007` — constants tuned during implementation, not decided on paper. |
| Facet embedding-space count is a guess (3: topic, rule/implication, entities). | Too few loses clustering recall; too many wastes inference. | `VMEM-P1-005` — needs real clustering output to settle. |
| `run_id` is nullable for heartbeat-spawned agents. | Memory→run backlink is absent for those writes. Documented upstream in `docs/agent-system/RUNTIME_ATTRIBUTION.md` with the token-claim overlay listed as future strengthening. | `VMEM-P1-002` — write path must tolerate absent correlation. |
| Adoption depends on the harness prompt block being installed and kept current. | A runtime with a stale or missing block silently keeps its private store, so the unification claim stops being true for it without any error. | `VMEM-P1-007`. |
| Deliberate-write path assumes agents notice what is worth remembering. | Untested. The 1-in-200 records measurement is evidence about *flags*, not about *noticing*. | `VMEM-P2-001` is the fallback if this proves false. |
| Summarization drift (fact mutation vs. intended fact dropping) is uninstrumented. | Quality-only: `forest` is rebuildable, so drift never costs data. | `VMEM-P2-003`, deferred by decision D-012. |
| **Per-harness at-cap behaviour is unverified.** Whether a runtime over its size cap asks the agent to trim or silently truncates is undetermined for every harness except Codex's documented 32 KiB. | **Blocks a P0 requirement.** Silent truncation can drop a pinned standing rule with no signal, which defeats the guarantee pinning exists to provide. Pin-first ordering mitigates but does not verify. | `VMEM-P0-010` — determining actual at-cap behaviour is implementation work, not a documentation task (D-017). |
| **Hook availability is unverified for the two single-blob harnesses** (Codex, opencode). `pretooluse-bash-deny.sh` proves the mechanism exists in `claude-code` and `grok`; nothing establishes an equivalent pre-write surface elsewhere. | Store diff is the universal floor, but D-015 records that single-blob stores cannot be diffed reliably into discrete memories. If no hook exists for those two runtimes, capture has no precise channel for them and the fallback named in D-015 is instructing the CLI directly — accepting the compliance risk D-015 was written to avoid. | `VMEM-P1-008` — verify per-runtime before committing to capture as the sole write path for those harnesses. |
| **Pin budget value and review interval are unvalidated.** D-018 establishes that attention rather than tokens is the binding constraint, but not where the threshold sits. | A cap set below real need produces routine trade-off prompts on pins that all genuinely belong, training operators to dismiss them. | `VMEM-P1-010` — calibrate against a real imported pin set. |

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
