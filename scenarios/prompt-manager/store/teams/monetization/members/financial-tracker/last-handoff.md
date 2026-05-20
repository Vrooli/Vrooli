### Inputs needed from operator
- `cash`
- `monthlyBurn.aiApi`
- `monthlyBurn.infrastructure`
- `monthlyBurn.saas`
- `monthlyBurn.tooling`
- `timeAllocation.product`
- `timeAllocation.ops`

### Stale operator inputs
- None. `operator-inputs.json::lastUpdatedAt` remains null, so no populated values can age stale.

### Snapshot summary
- Cash: `pending-operator`
- Monthly burn: `pending-operator` across all four categories
- Monthly revenue: `0` `aspirational`; subscription MRR and tier/bundle revenue `pending-telemetry`
- Services revenue/time: fixed zeroes, `not-applicable-pre-launch`
- Channel attribution: `pending-telemetry`
- Runway/default-alive gap: `pending-operator`; cannot compute without cash and burn
- Time allocation: product `pending-operator` / services `0 fixed` / ops `pending-operator`
- Retention, churn, NRR, ARPU, LTV: `pending-telemetry`

### Material deltas since last snapshot
- No material financial delta since `ledger-snapshot/2026-05-19`.
- Source/shared state was accessible at `/home/matthalloran8/Vrooli`; sandbox cwd was empty.
- File-backed `ledger.jsonl` still had entries only through 2026-04-27 before today’s append.

### Flags raised
- `operator-inputs-unpopulated`
- `runway-warning-pending-operator`
- `services-trap-warning-not-applicable-pre-launch`
- `channel-attribution-pending-telemetry`

### Decisions raised this heartbeat
- None.
- Current pending decision count is 1: `dec-1778875348622351458` (`benchmark-update`), outside financial-tracker owned contexts.
- Owned-context pending count is 0.

### Assumptions checked
- Tier 2 majority within 12 months: no data.
- Tier 3 attach among non-technical users: no data.
- NRR >= 110% within 18 months: pre-launch, pending telemetry.
- Services-to-subscription conversion >= 40%: no active services lines.
- Services engagements >3 months without productization: no active services lines.

### Pending-telemetry fields
- Subscription MRR by tier and bundle: TELEMETRY_ROADMAP Gap 2.
- Retention, churn, NRR, ARPU, LTV: post-launch lifecycle telemetry.
- Channel attribution: TELEMETRY_ROADMAP Gap 5.
- Infrastructure cost aggregation: TELEMETRY_ROADMAP Gap 3.
- Gateway usage and token cost attribution: TELEMETRY_ROADMAP Gap 4.

### Knowledge entry written
- `ledger-snapshot/2026-05-20`: `knw-1779300162163053505`
- Ledger entry appended: `ledger-1779300084666586410`