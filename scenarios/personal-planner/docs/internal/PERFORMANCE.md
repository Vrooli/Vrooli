# Performance — Personal Planner

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario, and what does "fast enough"
  mean concretely?
- What budgets or thresholds apply, and where do they come from?
- What has actually been measured (versus targeted)?
- What performance risks and design constraints must implementation
  respect?

**Maturity note (read first):** Personal Planner is at design stage. The
budgets below are the *targets* set by the implementation plan (§23.6).
No product code exists to measure yet — see "Current Measurements".

## Budgets

Product targets from the implementation plan §23.6. These are release
goals, not measured results.

| Surface / operation | Target | Notes |
|---|---|---|
| Today / day API read | p95 < 500 ms | The Today view answers "what can I usefully do now?"; the scene must not gate this. |
| Native command acknowledgment | p95 < 500 ms | A mutation through the one authoritative command surface acknowledges quickly (canonical success may follow for async work). |
| First useful Today (perceived) | < 2 s | First real content is visible within 2 s; **decorative scene artwork must not block it**. |
| Interactive proposal generation | < 2 s for ≤ 200 items over ≤ 28 days | Larger scopes run asynchronously with visible progress rather than blocking interaction. |
| Forecast refresh | visible result < 5 s | While recomputing, the prior forecast is labeled "updating," never shown as current. |
| Reference-scale datasets | 1,000 work items / 10,000 events / 10,000 actuals | The bounded dataset sizes the above budgets are expected to hold at. |

Inherited platform/lifecycle budgets (from the generated template, not
product targets):

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5–10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

**None yet.** Personal Planner is at design stage: only the `health`
domain and the removable `notes` worked example are real code, so there
is no product workflow to measure against the budgets above. The plan's
targets (§23.6) become measurable as the corresponding surfaces are
built — starting with Today/day reads and command acknowledgment
(P04- era), then proposals and forecasts (P05).

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet (design stage). | n/a | n/a | 2026-09-18 |

## Known Constraints

These are architectural constraints implementation must respect so the
budgets above are achievable and stay achievable.

- **Do not run the long-horizon scheduling/forecast solver on every
  keystroke, timeline tick, hover, or theme change.** Expensive
  recomputation must be **debounced and coalesced**, and superseded runs
  cancelled by revision — a newer input invalidates an in-flight
  computation rather than racing it. The solver is a pure kernel over an
  immutable snapshot; it is invoked deliberately, not reactively on every
  UI event.
- **Scene artwork must never block first content.** The Observatory
  landscape is a replaceable decorative asset; the first useful Today
  view renders without waiting on it, and artwork is bundled/cached
  locally (never a third-party image host on the critical path) and never
  regenerated per visit. An art-free / reduced-scenery setting exists.
- **Bound the expensive shapes:** recurrence expansion is bounded (never
  materialize an unbounded series), history is paginated, and dense
  lists/timelines are virtualized so large workspaces stay interactive at
  reference scale.
- **Async over blocking for large work:** proposals beyond the interactive
  threshold (> 200 items or > 28 days) run as coalescing background jobs
  with visible progress; forecast refresh coalesces; the UI shows honest
  pending/updating/stale states rather than freezing.
- Vite production builds may process thousands of modules and take several
  minutes (inherited template constraint).

## Regression Procedure

Once product surfaces exist and baselines are captured, guard them as
follows. (No product baseline exists yet — this is the procedure to
follow when there is one.)

1. Run `make test`.
2. Capture relevant API/UI command timing for the affected surface
   (Today/day read, command ack, proposal, forecast) against a
   reference-scale dataset.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Compare against the budgets in this document; a regression past a
   budget is a defect, not an accepted constraint.
5. Record persistent findings in this document (accepted constraints and
   captured measurements) or [`PROBLEMS.md`](PROBLEMS.md) (unresolved
   debt), depending on which they are.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — coalescing jobs, forecast refresh, proposal apply
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
