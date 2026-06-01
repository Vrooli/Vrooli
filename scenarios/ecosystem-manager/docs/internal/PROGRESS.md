# Progress — Ecosystem Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-31 | Matthew Halloran | done | Controller validation & hardening (P2–P7). **P2** proceed-cap-flag degraded-gate policy: DTV-down or all-red ⇒ proceed with the highest-trust (`TrustRanker`) skill, halve the remaining iteration budget once (latch derived from the trace's first `GateDegradedCause`), flag + durable persistence + prominent UI badge. **P3** cross-repo vocabulary guard (EM `pkg/skillmap` + prompt-manager side) failing on any out-of-vocabulary skill `targetDimensions`. **P4** `PredictedReduction` on the decision trace (efficacy × est-tokens × dimension-weight) + UI predicted-vs-realized Δ and calibration-MAE indicator. **P5** doc reconciliation (DTV/test-genie now present-tense wired across GLOSSARY/DOMAINS/SEAMS/INTEGRATIONS/CONTROL-MODEL; PROBLEMS self-contradiction removed) + anti-drift guard. Decision-trace table gained durable DTV + P2/P4 columns. |
| 2026-05-31 | Matthew Halloran | done | EM-P2-001/002: wired DTV trust/cost priors + Layer-1 eligibility gate. New `pkg/dtv` Connect client (`SkillFitnessProvider`, fails open), `pkg/autosteer/dtv_selection.go` (`DTVEligibilityFilter` denies RED, `DTVPriorProvider` seeds prior=weight·base·trust·convergence). Selector's scalar prior generalized to a `PriorProvider` seam. Per-task fitness snapshot with TTL refresh at SELECT; DTV unreachable ⇒ exact P1 (uniform prior, allow-all), logged once. Profile `dtv` objective-block (gate_enabled/prior_weight/trust_floor/refresh_iters). Decision trace + `steer trace` CLI + UI surface DTV verdict/prior/exclusions. Consumes DTV's new `GetSkillFitness` RPC. |
| 2026-05-30 | Matthew Halloran | done | Documentation overhaul to react-vite v2 contract + new CONTROL-MODEL.md (closed-loop controller mental model) authored. |
| 2026-02-18 | unassigned | done | Added standard docs layout and manifest; added quickstart, guides, reference pages; added internal memory docs for maintenance loops; moved recycler guide under `docs/plans` and registered it. |
| 2026-02-13 | unassigned | done | Interoperability audit pass 2: added proto schemas under `packages/proto/schemas/ecosystem-manager/v1/`, UI proto-contracts validation layer, per-request discovery URL resolution for all scenario clients, and `buildExecuteResult` status tests. |

## Validation status

Live end-to-end controller runs (the gate on autonomous use). The unit/guard
suites pin the seams; these rows record real loop executions against real
scenarios.

| Date | Run | Target | Profile | Budget | Termination | Notes |
|---|---|---|---|---|---|---|
| 2026-05-31 | P0 baseline | bookmark-intelligence-hub / fall-foliage-explorer | comprehensive audit | n/a | n/a | Baselines captured as run references. fall-foliage-explorer: 56 findings (41 blocker/error) — campaign-flagged; multi-dimension spread confirms selection has real choices. |
| _pending_ | P1 attended | bookmark-intelligence-hub | balanced (max_iterations 3) | tiny | _pending_ | Mechanical end-to-end smoke; surface substrate bugs before unattended use. |
| _pending_ | P8 final | bookmark (stress) + fall-foliage (convergence) + backstop | balanced | full | _pending_ | Convergence proof + degraded-gate backstop after hardening. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller mental model
