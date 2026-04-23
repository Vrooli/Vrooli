### Inputs needed from operator (pending-operator status in operator-inputs.json)
- `cash`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#cash-on-hand
- `monthlyBurn.aiApi`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#monthly-ai--api-cost
- `monthlyBurn.infrastructure`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#monthly-infrastructure-cost
- `monthlyBurn.saas`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#monthly-third-party-saas-cost
- `monthlyBurn.tooling`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#monthly-tooling-cost
- `timeAllocation.product`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#time-allocation
- `timeAllocation.ops`: last updated: never. Gathering guide: docs/monetization/HOW_TO_GATHER_INPUTS.md#time-allocation

### Stale operator inputs (updatedAt exceeds stalenessPolicy)
- No stale inputs (file has never been populated).

### Snapshot summary
- Cash: `pending-operator`
- Monthly burn: `pending-operator` (all four categories unpopulated)
- Monthly revenue: 0 `aspirational` (no subscriptions shipped; services not active)
- Runway: `pending-operator` (missing cash and burn)
- Default-alive gap: `pending-operator`
- Time allocation: product `pending-operator` / services 0% `not-applicable-pre-launch` / ops `pending-operator`

### Material deltas since last snapshot
- First heartbeat — no prior snapshot. No deltas computable.

### Flags raised
- `operator-inputs-unpopulated` (structural flag, not one of the three framework-level flags; pointed at the operator's to-do list, not at a framework breach)
- services-trap-warning: not-applicable-pre-launch
- runway-warning: pending-operator (cannot compute)
- assumption-drift: none

### Decisions raised this heartbeat
- none. No data yet to justify any of the six owned contexts. Team queue: 0 pending (well under 12-decision ceiling).

### Assumptions checked
- Assumption 1 (Tier 2 majority within 12 mo of Tier 2 offering): still holds — no data
- Assumption 2 (Tier 3 attach high among non-technical): still holds — no data
- Assumption 3 (NRR ≥ 110% within 18 mo of launch): still holds — pre-launch
- Assumption 4 (services→sub conversion ≥40% with productization + trust): still holds — no active services lines
- Assumption 5 (services engagements >3 mo without productization → trap): still holds — no active services lines

### Pending-telemetry fields
- Subscription MRR by tier/bundle → TELEMETRY_ROADMAP.md Gap 2 (LPBS/Stripe lifecycle events)
- Retention / monthly churn / LTV → post-launch, depends on Gap 2
- Infrastructure cost auto-aggregation → scenario-to-cloud cost API (currently satisfied by operator-inputs.json::monthlyBurn.infrastructure once populated)
- Gateway token cost auto-aggregation → API gateway (Tier 2 prereq); today bundled into `monthlyBurn.aiApi`

### Knowledge entry written
- topic: `ledger-snapshot-2026-04-23` (id: `knw-1776967259105788561`, no prior entry to supersede)