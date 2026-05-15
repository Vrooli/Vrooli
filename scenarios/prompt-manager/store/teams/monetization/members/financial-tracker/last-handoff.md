### Inputs needed from operator
- `cash`: last updated never; status `pending-operator`
- `monthlyBurn.aiApi`: last updated never; status `pending-operator`
- `monthlyBurn.infrastructure`: last updated never; status `pending-operator`
- `monthlyBurn.saas`: last updated never; status `pending-operator`
- `monthlyBurn.tooling`: last updated never; status `pending-operator`
- `timeAllocation.product`: last updated never; status `pending-operator`
- `timeAllocation.ops`: last updated never; status `pending-operator`

### Stale operator inputs
- None. No populated values exist, so nothing can age past the staleness policy.

### Snapshot summary
- Cash: `pending-operator`
- Monthly burn: `pending-operator` across all four categories
- Monthly revenue: 0 `aspirational`; subscriptions `pending-telemetry`; services `not-applicable-pre-launch`
- Channel attribution: `pending-telemetry`
- Runway/default-alive gap: `pending-operator`; cannot compute without cash and burn
- Time allocation: product `pending-operator` / services 0 `not-applicable-pre-launch` / ops `pending-operator`
- Retention, churn, NRR, ARPU, LTV: `pending-telemetry`

### Material deltas since last snapshot
- No material delta since `ledger-snapshot/2026-05-13`.
- `operator-inputs.json::lastUpdatedAt` remains null.
- No supported LPBS/Stripe, services activation, channel-attribution, or infra cost telemetry found.

### Flags raised
- `operator-inputs-unpopulated`
- `runway-warning-pending-operator`
- `services-trap-warning-not-applicable-pre-launch`

### Decisions raised this heartbeat
- None.
- Owned-context pending count remains 3: `dec-1777061056395576280`, `dec-1777406561539481259`, `dec-1777406584829985287`; new decisions skipped per cap.
- Existing challenge responses for the COGS decision stack remain author-responded; no new response needed.

### Assumptions checked
- Tier 2 majority within 12 months: still no data.
- Tier 3 attach among non-technical users: still no data.
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
- `ledger-snapshot/2026-05-14`: `knw-1778781836691445206`
- Ledger entry appended: `ledger-1778781795594971362`