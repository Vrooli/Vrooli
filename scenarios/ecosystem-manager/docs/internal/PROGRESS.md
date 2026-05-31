# Progress — Ecosystem Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-31 | Matthew Halloran | done | EM-P2-001/002: wired DTV trust/cost priors + Layer-1 eligibility gate. New `pkg/dtv` Connect client (`SkillFitnessProvider`, fails open), `pkg/autosteer/dtv_selection.go` (`DTVEligibilityFilter` denies RED, `DTVPriorProvider` seeds prior=weight·base·trust·convergence). Selector's scalar prior generalized to a `PriorProvider` seam. Per-task fitness snapshot with TTL refresh at SELECT; DTV unreachable ⇒ exact P1 (uniform prior, allow-all), logged once. Profile `dtv` objective-block (gate_enabled/prior_weight/trust_floor/refresh_iters). Decision trace + `steer trace` CLI + UI surface DTV verdict/prior/exclusions. Consumes DTV's new `GetSkillFitness` RPC. |
| 2026-05-30 | Matthew Halloran | done | Documentation overhaul to react-vite v2 contract + new CONTROL-MODEL.md (closed-loop controller mental model) authored. |
| 2026-02-18 | unassigned | done | Added standard docs layout and manifest; added quickstart, guides, reference pages; added internal memory docs for maintenance loops; moved recycler guide under `docs/plans` and registered it. |
| 2026-02-13 | unassigned | done | Interoperability audit pass 2: added proto schemas under `packages/proto/schemas/ecosystem-manager/v1/`, UI proto-contracts validation layer, per-request discovery URL resolution for all scenario clients, and `buildExecuteResult` status tests. |

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
