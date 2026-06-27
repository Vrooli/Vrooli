# Problems — Meta-Optimization Manager

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

> Note: fleet-level gaps that this scenario *measures* (e.g. MISSING Answer
> cells, the Validate experience-ladder, the SKETCH Guide denominator) are
> **domain data**, not this scenario's debt — they live in the space docs and
> the runtime gaps registry, not here.

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

### 2026-06-24 — `space --projection` verb not built on owners; coverage uses the doc-parse fallback

**Symptom:** Coverage does not read denominators through the decoupled shared contract; it parses the owner space docs directly.

**Root cause:** The shared read contract (a `space --projection <p> --json` verb on search-hub/test-genie/prompt-manager) is a design dependency that has not been built yet.

**Workaround (in place, working):** `SpaceReader` (`api/internal/coverage/spacereader.go`) prefers the owner `space` verb but **falls back to parsing `scenarios/<owner>/docs/spaces/<p>-space.md` with the same `api-core/spacedoc` parser the verb will use** — guaranteed-consistent output. The scoreboard numbers are real today; only the decoupling is pending. `status` is fully implemented.

**Real fix:** Add the `space` verb (schema-validated JSON) to each owner; `SpaceReader` then takes the preferred path automatically with no MoM change.

**Owner:** unassigned.

**Refs:** `api/internal/coverage/spacereader.go`; `../concepts/ARCHITECTURE.md` (Contracts And Data Flow); the three `*/docs/spaces/*-space.md` docs.

### 2026-06-26 — trials unproven end-to-end on a live local model

**Symptom:** `trials run` is verb/flag-correct with real fixtures + deterministic oracles, but has never completed a full live pass (opencode + local model) producing a scored verdict.

**Root cause:** The diff-apply path is an untested assumption: the runner scopes the agent to `fixture.TargetDir` (`--scope-path`), captures the diff via `agent-manager run diff`, and the evaluator applies it with `git apply -p1` inside a copy of `target/`. This only lands cleanly if agent-manager emits diff paths rooted at the scope-path with `a/…  b/…` prefixes; if rooted elsewhere, every trial silently becomes `VerdictError`. Separately, `agentJudge` is `nil` in prod — correct today (all 5 fixtures carry an oracle → oracle-less degrades to honest `VerdictError`), but a future oracle-less family would silently yield `VerdictError` with only a log line.

**Workaround:** Keep OT-P1-001 unaccepted until the live pass; trust the oracle-backed verdicts only.

**Real fix:** (1) One operator live-e2e (the `bugfix` family is smallest); capture the raw `run diff` output, confirm/repair the `-p1` rooting, and lock the captured diff as a golden fixture. (2) Add a corpus-load-time guard asserting every fixture has an oracle or a configured judge, so future oracle-less families fail loudly instead of degrading silently.

**Owner:** unassigned.

**Refs:** `api/internal/trials/runner.go`, `api/internal/trials/evaluator.go` (`applyDiff`, the `nil` judge).

### 2026-06-24 — convergence coordinated-edit walkthrough is unproven mechanization

**Symptom:** The add/delete coordinated-edit count (a reference-pattern-fitness sub-lens) has no existing programmatic engine.

**Root cause:** That sub-lens is a skill (judgment) today; mechanizing the *counting* is genuinely new work and may be fragile.

**Workaround:** Mark its findings lower-confidence until proven against a frozen template fixture.

**Real fix:** Build + validate the walkthrough counter; promote its confidence once it matches hand-audited counts.

**Owner:** unassigned.

**Refs:** `docs/agent-system/REFERENCE_PATTERN_FITNESS.md`; requirement MOM-CONVERGENCE-001.

### 2026-06-24 — prd-control-tower LLM generation returned HTTP 500

**Symptom:** `prd-control-tower prd generate … --publish` failed with `api error (500)`.

**Root cause:** The generation LLM backend was unavailable (env); the deterministic validator (`prd validate`) was healthy.

**Workaround:** PRD hand-authored to the canonical template and validated deterministically (0 violations).

**Real fix:** None required for this scenario; if a regeneration is wanted later, retry once the LLM backend is healthy.

**Owner:** unassigned.

**Refs:** `DECISIONS.md` (2026-06-24 documentation-first row).

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
