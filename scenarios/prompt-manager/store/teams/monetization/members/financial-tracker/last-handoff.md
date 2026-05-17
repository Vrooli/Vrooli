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
- No material financial delta since `ledger-snapshot/2026-05-14`.
- `operator-inputs.json::lastUpdatedAt` remains null.
- Pending benchmark decision `dec-1778875348622351458` does not change current ledger math.

### Flags raised
- `operator-inputs-unpopulated`
- `runway-warning-pending-operator`
- `services-trap-warning-not-applicable-pre-launch`

### Decisions raised this heartbeat
- None.
- Current pending decision count is 1: `dec-1778875348622351458` (`benchmark-update`), outside financial-tracker owned contexts.
- Owned-context pending count is 0.

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
- `ledger-snapshot/2026-05-16`: `knw-1778954570153129770`
- Ledger entry appended: `ledger-1778954492350670741`
- Process friction observed but not filed: generated storage/report-friction instructions still show `knowledge-add --by`, while live CLI rejects `--by`; attempted friction write to `meta-optimization` was blocked by `team_mismatch` despite universal-source friction docs.