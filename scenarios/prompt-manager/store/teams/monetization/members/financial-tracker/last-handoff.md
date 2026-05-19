### Inputs needed from operator
- `cash`
- `monthlyBurn.aiApi`
- `monthlyBurn.infrastructure`
- `monthlyBurn.saas`
- `monthlyBurn.tooling`
- `timeAllocation.product`
- `timeAllocation.ops`

### Stale operator inputs
- None. No populated values exist, so nothing can age past the staleness policy.

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
- No material financial delta since `ledger-snapshot/2026-05-17`.
- `operator-inputs.json::lastUpdatedAt` remains null.
- Pending benchmark decision `dec-1778875348622351458` does not change current ledger math.
- Note: local `ledger.jsonl` only had entries through 2026-04-27 before this append, despite prior handoff saying a 2026-05-17 ledger entry was appended.

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
- `ledger-snapshot/2026-05-18`: `knw-1779127304853839656`
- Ledger entry appended: `ledger-1779127252070642576`
- CLI note: `prompt-manager team knowledge-add` now auto-attributes identity; `--by` is rejected.