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
| Classifier calibration was below the unattended-use bar in the legacy 17/50 review (34%). | That historical sample is superseded by the repeatable live fixture: 432/432 held-out work records correct (100.00%) and 558/558 fall-through triples unanimous (100.00%) through the normal gateway role, with no provider changes. | Keep the repeatable fixture current; do not treat the legacy human sample as the current calibration gate. See `CALIBRATION-REVIEW.md`. |
| Facet embedding-space count is a guess (3: topic, rule/implication, entities). | Too few loses clustering recall; too many wastes inference. | `VMEM-P1-005` — needs real clustering output to settle. |
| `run_id` is nullable for heartbeat-spawned agents. | Memory→run backlink is absent for those writes. Documented upstream in `docs/agent-system/RUNTIME_ATTRIBUTION.md` with the token-claim overlay listed as future strengthening. | `VMEM-P1-002` — write path must tolerate absent correlation. |
| Adoption depends on the harness prompt block being installed and kept current. | A runtime with a stale or missing block silently keeps its private store, so the unification claim stops being true for it without any error. | `VMEM-P1-007`. |
| Deliberate-write path assumes agents notice what is worth remembering. | Untested. The 1-in-200 records measurement is evidence about *flags*, not about *noticing*. | `VMEM-P2-001` is the fallback if this proves false. |
| Summarization drift (fact mutation vs. intended fact dropping) is uninstrumented. | Quality-only: `forest` is rebuildable, so drift never costs data. | `VMEM-P2-003`, deferred by decision D-012. |
| Live local summarization previously returned `unavailable: unexpected EOF` through ai-gateway after 28–43 seconds. | The provider timeout was separated from classification and the live compaction pass now completed 31 summaries atomically; the old failure is retained only as historical context in the progress log. | Keep the five-minute summary timeout and atomic forest write; do not bypass ai-gateway or write summaries directly. |
| **Per-harness at-cap behaviour is unverified.** Whether a runtime over its size cap asks the agent to trim or silently truncates is undetermined for every harness except Codex's documented 32 KiB. | **Blocks a P0 requirement.** Silent truncation can drop a pinned standing rule with no signal, which defeats the guarantee pinning exists to provide. Pin-first ordering mitigates but does not verify. | `VMEM-P0-010` — determining actual at-cap behaviour is implementation work, not a documentation task (D-017). |
| **The two single-blob harnesses (Codex, opencode) expose no usable pre-write hook in the installed runtime.** `pretooluse-bash-deny.sh` proves the mechanism exists in `claude-code` and `grok`; the per-runtime matrix records the negative result for Codex/opencode and the other no-hook harnesses. | Store diff is the universal floor, but D-015 records that single-blob stores cannot be diffed reliably into discrete memories. Capture therefore remains precise for hook-capable runtimes and best-effort via governed store-diff recovery elsewhere; the limitation is explicit rather than guessed. | `VMEM-P1-008` — retain the verified capability matrix and add a native hook when a resource exposes one. |

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

## Work ladder

- Rung: W0 passed
- Evidence: The deterministic named-mention search `swarm-manager goals list --json | jq -r --arg s 'vrooli-memory' '.goals[].goal | select((.name + " " + .title + " " + .description) | test($s)) | .name'` returns the active governing goal `complete-and-validate-vrooli-memory-to-a-p0-p1-bar-with-a`. Its description requires the plan, generic policy/engine boundary, every P0/P1 requirement with named passing evidence, federation/harness validation, and a healthy ledger-ready scenario; the plan's P0/P1 operational targets describe the same outcome.
- Blocker: None at W0. Re-measurement continues at W1/W2; no maturity claim is made until the requirement evidence and plan acceptance are complete.
- Measured: 2026-08-04
