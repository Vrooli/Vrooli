# Ecosystem Fit — Performance Health

This records how Performance Health fits Vrooli's self-improving ecosystem,
per `prompt-manager skill read ecosystem-fit` and `docs/concepts/ECOSYSTEM.md`.
All five clusters were applied because this is a new scenario.

## Role (one line)

**Interface-enabler + meta self-improvement.** Performance Health is not a
standalone end-user product; it is the engineering-quality capability that gives
the rest of Vrooli (Test Genie, the Ecosystem Manager, agents) deterministic
performance judgment. Its dominant role is interface-enabler (a clean
programmatic surface other scenarios call); its secondary, defining role is
meta self-improvement (it advances Vrooli's testing/optimization meta-capability).

## Cluster 1 — Interfaces served / enabled

| Interface | Served? | "Done" obligation |
|---|---|---|
| **Programmatic (CLI / Connect)** | Yes — primary | A clean, reusable CLI + Connect surface other scenarios call. Two consumers are explicit: **Test Genie** delegates its `Performance` phase over the shared `scenario-validation/v1 ScenarioValidationService` (axes ① build-time + ③ Lighthouse); the **fleet** verbs are exact/structured queries discoverable through cli-health. Human-default CLI output with `--json` opt-in. |
| **Direct UI** | Yes — operator console | A polished, production-ready dashboard handling loading/error/empty states: audit workbench, trend charts, fleet dashboard, readiness + one-click autofix, budgets editor, trace viewer link-out. Not a marketing page. |
| Conversational / agentic | No | No widgets/tools obligation in v1. (The CLI/Connect surface is already agent-callable.) |
| Voice | No | n/a |
| Embodied / embedded | No | n/a |

## Cluster 2 — Functional role & multiplier raise

- **Role:** interface-enabler (programmatic perf surface) with a meta
  self-improvement payload.
- **Cheap multiplier raise spotted:** the legacy `scenario-performance-audit`
  skill was a hand-rolled LLM/agent flow (throwaway `capture.js`, `node -e`
  one-liners). This scenario turned that judgment into **deterministic Go code**
  (per-component aggregation, located findings via symbol lookup) — an LLM-to-code
  conversion that removes an entire class of per-run agent cost. Findings are
  deterministic by mandate (no AI hotspot hypotheses). The practice skill has
  since been removed; the `performance` steer skill now drives this engine.

## Cluster 3 — Compound-value seams

Concrete seams that let *future* scenarios/loops reuse this instead of re-implementing:

1. **Trend store as a data surface.** Per-run measurements + `common.v1.ExecutionMetrics`
   envelopes persisted in SQLite, additive and queryable — a durable optimization
   input for the Ecosystem Manager / meta-optimization, not a throwaway.
2. **`ScenarioValidationService` implementation.** The same sibling contract
   (unit-health, quality-health, structure-health, …) so Test Genie — and any
   future orchestrator — delegates perf judgment through one stable RPC.
3. **BAS perf-capture consumer.** Performance Health consumes BAS's `perf`
   capture artifact as a dumb mechanism. The tier model, autofix, and analysis
   live here; BAS stays agnostic. This keeps the capture mechanism reusable by
   anything (not just perf-health) and the judgment centralized.

## Cluster 4 — Self-improvement

**Yes — advances the testing/optimization meta-capability.** Performance Health
is the single home for performance regression detection and attribution across
the fleet, feeding deterministic offenders and trends into the self-improvement
loop. It also consolidates three previously-scattered efforts (test-genie native
perf phase, structure-health perf domain, the audit skill) into one owner,
reducing the system's maintenance surface — itself a meta upkeep win.

## Cluster 5 — Monetization & bundle fit (routed, not priced)

- **Bundle:** business bundle candidate (engineering-quality tooling), depth not
  headliner. Primary value is internal capability for Vrooli's own loop.
- **Free / metered / gated:** core readiness/audit/benchmark/startup and fleet
  queries should be **free** for self-hosters (BYOK stays valid — they run their
  own Chrome/Lighthouse/BAS). Any future hosted/metered surface routes through
  LPBS, never a reinvented credit system.
- **Decision deferred to canon.** Strategy and pricing are not decided here.
  Route to `docs/monetization/strategy/STRATEGY.md`, `docs/monetization/catalogs/CATALOG.md`,
  and `docs/concepts/PAID_FEATURES.md`; wire via `bundle-integration-steer`.
  Scenario-local stance recorded in `docs/business/MONETIZATION.md`.

## Source

- Plan: `~/.vrooli/plans/performance-health-scenario-bas-perf-capture-test-genie-perf-phase-migration.md`
- Canonical model: `docs/concepts/ECOSYSTEM.md`
