# Problems — Vrooli Memory

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This register carries only scenario-specific, measured issues and deferred
work; it is intentionally non-empty while those issues remain unresolved.

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
| Classifier calibration is below the unattended-use bar: 17/50 reviewed assignments were correct (34%). | Automatic capture/import can mislabel durable project knowledge and corrupt compaction eligibility until operator corrections are applied. | Add corpus-derived few-shot examples, remeasure on a fresh operator-reviewed sample, and keep corrections append-only; see D-024 and `CALIBRATION-REVIEW.md`. |
| Facet embedding-space count is a guess (3: topic, rule/implication, entities). | Too few loses clustering recall; too many wastes inference. | `VMEM-P1-005` — needs real clustering output to settle. |
| `run_id` is nullable for heartbeat-spawned agents. | Memory→run backlink is absent for those writes. Documented upstream in `docs/agent-system/RUNTIME_ATTRIBUTION.md` with the token-claim overlay listed as future strengthening. | `VMEM-P1-002` — write path must tolerate absent correlation. |
| Adoption depends on the harness prompt block being installed and kept current. | A runtime with a stale or missing block silently keeps its private store, so the unification claim stops being true for it without any error. | `VMEM-P1-007`. |
| Deliberate-write path assumes agents notice what is worth remembering. | Untested. The 1-in-200 records measurement is evidence about *flags*, not about *noticing*. | `VMEM-P2-001` is the fallback if this proves false. |
| Summarization drift (fact mutation vs. intended fact dropping) is uninstrumented. | Quality-only: `forest` is rebuildable, so drift never costs data. | `VMEM-P2-003`, deferred by decision D-012. |
| Live local summarization currently returns `unavailable: unexpected EOF` through ai-gateway after 28–43 seconds, despite both scenarios reporting healthy. | A production compaction pass safely aborts before its atomic tree write, but ten live summaries cannot yet be reviewed. The focused atomicity and provider-error tests still prove no partial forest is written. | Retry after local provider recovery; see D-029 and the Phase 11 ledger evidence. Do not bypass ai-gateway or write summaries directly. |
| **Per-harness at-cap behaviour is unverified.** Whether a runtime over its size cap asks the agent to trim or silently truncates is undetermined for every harness except Codex's documented 32 KiB. | **Blocks a P0 requirement.** Silent truncation can drop a pinned standing rule with no signal, which defeats the guarantee pinning exists to provide. Pin-first ordering mitigates but does not verify. | `VMEM-P0-010` — determining actual at-cap behaviour is implementation work, not a documentation task (D-017). |
| **Hook availability is unverified for the two single-blob harnesses** (Codex, opencode). `pretooluse-bash-deny.sh` proves the mechanism exists in `claude-code` and `grok`; nothing establishes an equivalent pre-write surface elsewhere. | Store diff is the universal floor, but D-015 records that single-blob stores cannot be diffed reliably into discrete memories. If no hook exists for those two runtimes, capture has no precise channel for them and the fallback named in D-015 is instructing the CLI directly — accepting the compliance risk D-015 was written to avoid. | `VMEM-P1-008` — verify per-runtime before committing to capture as the sole write path for those harnesses. |

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| No confirmed architecture drift | The shipped domain boundaries match the implemented journal, facets, forest, recall, federation, and harness surfaces. | None. | Re-audit whenever a cross-domain persistence dependency is introduced. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
