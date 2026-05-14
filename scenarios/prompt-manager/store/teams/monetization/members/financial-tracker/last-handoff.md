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
- Runway: `pending-operator`; cannot compute without cash and burn
- Default-alive gap: `pending-operator`; cannot compute without cash and burn
- Time allocation: product `pending-operator` / services 0 `not-applicable-pre-launch` / ops `pending-operator`
- Retention, churn, NRR, ARPU, LTV: `pending-telemetry`

### Material deltas since last snapshot
- No material delta since `ledger-snapshot/2026-04-27`.
- `operator-inputs.json::lastUpdatedAt` remains null.
- No LPBS/Stripe events, services activations, channel-attribution telemetry, or infra cost data found.

### Flags raised
- `operator-inputs-unpopulated`
- `runway-warning-pending-operator`
- `services-trap-warning-not-applicable-pre-launch`

### Decisions raised this heartbeat
- None.
- Own-context pending count is 3: `dec-1777061056395576280`, `dec-1777406561539481259`, `dec-1777406584829985287`; new decisions skipped per cap.
- Challenge responses recorded:
  - `knw-1778695371812806162` for `challenge-resolution-record/dec-1777406561539481259`
  - `knw-1778695371813093402` for `challenge-resolution-record/dec-1777406584829985287`

### Assumptions checked
- Tier 2 majority within 12 months: still no data.
- Tier 3 attach among non-technical users: still no data.
- NRR >= 110% within 18 months: pre-launch, pending telemetry.
- Services-to-subscription conversion >= 40%: no active services lines.
- Services engagements >3 months without productization: no active services lines.

### Pending-telemetry fields
- Subscription MRR by tier and bundle: TELEMETRY_ROADMAP Gap 2.
- Retention, churn, NRR, ARPU, LTV: post-launch subscription lifecycle telemetry.
- Channel attribution: TELEMETRY_ROADMAP Gap 5.
- Infrastructure cost aggregation: TELEMETRY_ROADMAP Gap 3; operator input remains interim source.
- Gateway usage and token cost attribution: TELEMETRY_ROADMAP Gap 4; currently bundled under `monthlyBurn.aiApi` once populated.

### Knowledge entry written
- `ledger-snapshot/2026-05-13`: `knw-1778695371813018132`
- Ledger entry appended: `ledger-1778695309108209538`