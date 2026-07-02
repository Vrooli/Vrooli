# Flows

## Purpose Of This Document

The workflow and state-transition map for **meta-optimization-manager**. Most flows are simple read-aggregate-respond; the one flow with real lifecycle and ordering risk is `trials run`, which dispatches work to other scenarios and must isolate and attribute it.

## Flow Inventory

| Flow | Trigger | Domain |
|---|---|---|
| **Readiness status** | `status` | coverage |
| **Base-doc validation** | `coverage validate-docs` | coverage |
| **Convergence scan** | `convergence fitness` / `reference-health` | convergence |
| **Focus / gaps** | `focus`, `gaps` | focus |
| **Trials run** | `trials run` | trials |

## Flow Details

**Readiness status** — for each projection: read the owner's `space --projection <p> --json` (denominator) → read the live registry (numerator) → join → compute coverage + attach the denominator-confidence from the space doc → attach the latest trials trend → return one scoreboard. Any unreachable source yields that projection as "unavailable"; the rest still compute.

**Base-doc validation** — parse each space doc's referenced skills/providers/phases → check each against the live owner registry → emit a finding per stale/broken reference; additionally flag Guide rows whose skill count ≠ 1.

**Convergence scan** — for each registered template: gather raw structure (code-facts/cartographer) → compute the fitness counts (per-replica cost, drift surfaces, comment-only contracts, add/delete coordinated edits) → persist to `convergence_fitness`. For each gold-star generated golden: compare Generated date vs template last-commit (stale-from-template), read clean-on-all-tools from test-genie/scenario-auditor, compute stability + breadth → persist verdict.

**Focus / gaps** — aggregate gaps from coverage + convergence + the registry → rank by impact × importance → return with qualitative context. `gaps` lists/filters the registry directly.

**Trials run** — see the state machine below.

## State Machines

**Trial run lifecycle:**
```
resolve  → fixture for the task family (defines success + supplies the idempotency rev)
reuse?   → a recent identical (task, model, fixture-rev) run is returned as-is (no double spend)
dispatch → agent-manager: profile ensure → task create (scope = fixture target/) → run create --run-mode sandboxed
running  → agent-manager owns the sandbox; MoM polls run get until terminal
collect  → run diff + run get metrics = EVIDENCE (diff, tokens, wall-time, changed-files); NO verdict yet
evaluated→ MoM Evaluator: deterministic fixture oracle (copy target/ + apply diff + run check), else agent-judge
recorded → verdict + tokens + wall-time + fixture-rev appended to trials_runs
[failed] → any step degrades that one run to VerdictError (recorded), never blocks the suite, never fabricates a pass
```
agent-manager is ONLY the sandboxed-agent spawner (it returns evidence); deciding
whether the task was solved is MoM's job (the Evaluator).

**Gap lifecycle:**
```
open → in-focus (surfaced by focus) → resolved | deferred
```

## Maturity Ladder

All flows are **planned** (documentation-first; no code yet). The first implemented flow is **Readiness status** (the `coverage` vertical slice). `trials run` is P1; the UI flows are P2.

## Production Shape

- Every upstream read **degrades gracefully** — a down source never fails the whole response.
- `trials run` is **gated strictly behind explicit invocation** (never on a hot path) because it is computationally expensive; it runs sandboxed and attributes every changed file to its run.
- Coverage is computed live; only short-TTL snapshots are cached.

## Deferred / Unmodeled Flows

- UI rendering flows (operator console) — P2.
- Attested-readiness registration with search-hub — P2.
- Convergence auto-refresh cadence — deferred; scans are on-demand until a cadence is justified.

## Cross-References

- [DOMAINS.md](DOMAINS.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [DATA.md](DATA.md)
- [INTEGRATIONS.md](INTEGRATIONS.md) — the upstream scenarios each flow reads.
- [../internal/SEAMS.md](../internal/SEAMS.md) — the read-client + storage seams.
